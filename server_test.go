package main

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestHandleIndex(t *testing.T) {
	srv := newServer("/bin/sh", false, ".")

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Fatalf("expected text/html content type, got %q", ct)
	}
	body := w.Body.String()
	if !strings.Contains(body, "webnetd") {
		t.Fatal("index page should contain 'webnetd'")
	}
	if !strings.Contains(body, "false") || !strings.Contains(body, "AUTH_ENABLED") {
		t.Fatal("index page should have AUTH_ENABLED set to false when auth is disabled")
	}
	if strings.Contains(body, "AUTH_ENABLED = true") || strings.Contains(body, "AUTH_ENABLED =  true") {
		t.Fatal("AUTH_ENABLED should be false, not true")
	}
}

func TestHandleIndexWithAuth(t *testing.T) {
	srv := newServer("/bin/sh", true, ".")

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	hasTrue := strings.Contains(body, "AUTH_ENABLED =  true") || strings.Contains(body, "AUTH_ENABLED = true")
	if !hasTrue {
		t.Fatal("index page should have AUTH_ENABLED = true when auth is enabled")
	}
}

func TestHandleIndexNotFound(t *testing.T) {
	srv := newServer("/bin/sh", false, ".")

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHandleUpload(t *testing.T) {
	uploadDir := t.TempDir()
	srv := newServer("/bin/sh", false, uploadDir)

	// Create multipart request
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "testfile.txt")
	if err != nil {
		t.Fatal(err)
	}
	part.Write([]byte("hello world"))
	writer.Close()

	req := httptest.NewRequest("POST", "/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		OK   bool   `json:"ok"`
		Name string `json:"name"`
		Size int64  `json:"size"`
		Path string `json:"path"`
	}
	if unmarshalErr := json.Unmarshal(w.Body.Bytes(), &resp); unmarshalErr != nil {
		t.Fatalf("failed to parse response: %v", unmarshalErr)
	}
	if !resp.OK || resp.Name != "testfile.txt" || resp.Size != 11 {
		t.Fatalf("unexpected response: %+v", resp)
	}

	// Verify file was written
	content, err := os.ReadFile(filepath.Join(uploadDir, "testfile.txt"))
	if err != nil {
		t.Fatalf("failed to read uploaded file: %v", err)
	}
	if string(content) != "hello world" {
		t.Fatalf("expected 'hello world', got %q", content)
	}
}

func TestHandleUploadWrongMethod(t *testing.T) {
	srv := newServer("/bin/sh", false, ".")

	req := httptest.NewRequest("GET", "/upload", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleUploadNoFile(t *testing.T) {
	srv := newServer("/bin/sh", false, t.TempDir())

	req := httptest.NewRequest("POST", "/upload", strings.NewReader(""))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=xxx")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleUploadPathTraversal(t *testing.T) {
	uploadDir := t.TempDir()
	srv := newServer("/bin/sh", false, uploadDir)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("file", "../../../etc/passwd")
	part.Write([]byte("malicious"))
	writer.Close()

	req := httptest.NewRequest("POST", "/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 (filepath.Base strips traversal), got %d", w.Code)
	}

	// File should be saved as just "passwd" in the upload dir, not in /etc/
	if _, err := os.Stat(filepath.Join(uploadDir, "passwd")); os.IsNotExist(err) {
		t.Fatal("file should be saved with base name only")
	}
}

func TestHandleUploadRequiresAuthWhenEnabled(t *testing.T) {
	srv := newServer("/bin/sh", true, t.TempDir())

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("file", "test.txt")
	part.Write([]byte("data"))
	writer.Close()

	req := httptest.NewRequest("POST", "/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth, got %d", w.Code)
	}
}

func TestWebSocketSession(t *testing.T) {
	srv := newServer("/bin/sh", false, ".")
	ts := httptest.NewServer(srv)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial: %v", err)
	}
	defer conn.Close()

	// Send a command
	cmd := []byte("echo hello-webnetd\n")
	if err := conn.WriteMessage(websocket.BinaryMessage, cmd); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Read output until we find our marker
	deadline := time.Now().Add(5 * time.Second)
	var output []byte
	for time.Now().Before(deadline) {
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		output = append(output, msg...)
		if strings.Contains(string(output), "hello-webnetd") {
			break
		}
	}

	if !strings.Contains(string(output), "hello-webnetd") {
		t.Fatalf("expected to find 'hello-webnetd' in output, got:\n%s", output)
	}
}

func TestWebSocketResize(t *testing.T) {
	srv := newServer("/bin/sh", false, ".")
	ts := httptest.NewServer(srv)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial: %v", err)
	}
	defer conn.Close()

	// Send resize message
	resizeMsg := `{"type":"resize","data":{"rows":50,"cols":120}}`
	if err := conn.WriteMessage(websocket.TextMessage, []byte(resizeMsg)); err != nil {
		t.Fatalf("write resize: %v", err)
	}

	// Verify terminal is responsive after resize by sending a command
	time.Sleep(100 * time.Millisecond)
	cmd := []byte("echo resize-ok\n")
	if err := conn.WriteMessage(websocket.BinaryMessage, cmd); err != nil {
		t.Fatalf("write after resize: %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	var output []byte
	for time.Now().Before(deadline) {
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		output = append(output, msg...)
		if strings.Contains(string(output), "resize-ok") {
			break
		}
	}

	if !strings.Contains(string(output), "resize-ok") {
		t.Fatalf("terminal should remain functional after resize")
	}
}

func TestWebSocketRequiresAuthWhenEnabled(t *testing.T) {
	srv := newServer("/bin/sh", true, ".")
	ts := httptest.NewServer(srv)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		conn.Close()
		t.Fatal("expected websocket dial to fail without auth")
	}
	if resp != nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", resp.StatusCode, body)
		}
	}
}
