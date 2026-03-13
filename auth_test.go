package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestGeneratePIN(t *testing.T) {
	pin := generatePIN()
	if len(pin) != 6 {
		t.Fatalf("expected 6-digit PIN, got %q", pin)
	}
	for _, c := range pin {
		if c < '0' || c > '9' {
			t.Fatalf("PIN contains non-digit character: %q", pin)
		}
	}
}

func TestGeneratePINUniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		pin := generatePIN()
		seen[pin] = true
	}
	if len(seen) < 50 {
		t.Fatalf("expected reasonable uniqueness across 100 PINs, got %d unique", len(seen))
	}
}

func TestVerifyPIN(t *testing.T) {
	a := newAuth()

	if !a.verifyPIN(a.pin) {
		t.Fatal("correct PIN should verify")
	}
	if a.verifyPIN("000000") && a.pin != "000000" {
		t.Fatal("wrong PIN should not verify")
	}
	if a.verifyPIN("") {
		t.Fatal("empty PIN should not verify")
	}
	if a.verifyPIN("12345") {
		t.Fatal("short PIN should not verify")
	}
}

func TestTokenCreateAndValidate(t *testing.T) {
	a := newAuth()

	token := a.createToken()
	if token == "" {
		t.Fatal("token should not be empty")
	}
	if !a.validToken(token) {
		t.Fatal("freshly created token should be valid")
	}
	if a.validToken("bogus-token") {
		t.Fatal("bogus token should be invalid")
	}
	if a.validToken("") {
		t.Fatal("empty token should be invalid")
	}
}

func TestTokenExpiry(t *testing.T) {
	a := newAuth()
	token := a.createToken()

	// Manually expire the token
	a.mu.Lock()
	a.tokens[token] = time.Now().Add(-1 * time.Second)
	a.mu.Unlock()

	if a.validToken(token) {
		t.Fatal("expired token should be invalid")
	}

	// Verify it was cleaned up
	a.mu.Lock()
	_, exists := a.tokens[token]
	a.mu.Unlock()
	if exists {
		t.Fatal("expired token should be removed from map")
	}
}

func TestTokenFromRequest(t *testing.T) {
	a := newAuth()

	// From query parameter
	req := httptest.NewRequest("GET", "/ws?token=abc123", nil)
	if got := a.tokenFromRequest(req); got != "abc123" {
		t.Fatalf("expected token from query, got %q", got)
	}

	// From cookie (takes precedence)
	req = httptest.NewRequest("GET", "/ws?token=fromquery", nil)
	req.AddCookie(&http.Cookie{Name: "webnetd_token", Value: "fromcookie"})
	if got := a.tokenFromRequest(req); got != "fromcookie" {
		t.Fatalf("expected cookie token to take precedence, got %q", got)
	}
}

func TestHandleLoginSuccess(t *testing.T) {
	a := newAuth()

	form := url.Values{"pin": {a.pin}}
	req := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	a.handleLogin(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		OK    bool   `json:"ok"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if !resp.OK || resp.Token == "" {
		t.Fatalf("expected ok=true and non-empty token, got %+v", resp)
	}

	// Verify the returned token is actually valid
	if !a.validToken(resp.Token) {
		t.Fatal("returned token should be valid")
	}

	// Verify cookie was set
	cookies := w.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "webnetd_token" && c.Value == resp.Token {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected webnetd_token cookie to be set")
	}
}

func TestHandleLoginWrongPIN(t *testing.T) {
	a := newAuth()

	form := url.Values{"pin": {"999999"}}
	req := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	a.handleLogin(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestHandleLoginWrongMethod(t *testing.T) {
	a := newAuth()

	req := httptest.NewRequest("GET", "/login", nil)
	w := httptest.NewRecorder()
	a.handleLogin(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestRequireAuthMiddleware(t *testing.T) {
	a := newAuth()
	token := a.createToken()

	handler := a.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Without token
	req := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	handler(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", w.Code)
	}

	// With valid token in query
	req = httptest.NewRequest("GET", "/protected?token="+token, nil)
	w = httptest.NewRecorder()
	handler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid token, got %d", w.Code)
	}

	// With valid token in cookie
	req = httptest.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "webnetd_token", Value: token})
	w = httptest.NewRecorder()
	handler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with cookie token, got %d", w.Code)
	}

	// With invalid token
	req = httptest.NewRequest("GET", "/protected?token=invalid", nil)
	w = httptest.NewRecorder()
	handler(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with invalid token, got %d", w.Code)
	}
}
