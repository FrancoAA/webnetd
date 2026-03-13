package main

import (
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/websocket"
)

type indexData struct {
	AuthEnabled bool
}

type server struct {
	shell     string
	auth      *auth
	uploadDir string
	indexTmpl *template.Template
	upgrader  websocket.Upgrader
	mux       *http.ServeMux
}

// Client-to-server message types.
type wsMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

type resizeMsg struct {
	Rows uint16 `json:"rows"`
	Cols uint16 `json:"cols"`
}

func newServer(shell string, authEnabled bool, uploadDir string) *server {
	tmpl := template.Must(template.New("index").Parse(indexHTML))

	s := &server{
		shell:     shell,
		uploadDir: uploadDir,
		indexTmpl: tmpl,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}

	if authEnabled {
		s.auth = newAuth()
		log.Printf("auth: PIN is %s", s.auth.pin)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)

	if s.auth != nil {
		mux.HandleFunc("/login", s.auth.handleLogin)
		mux.HandleFunc("/ws", s.auth.requireAuth(s.handleWS))
		mux.HandleFunc("/upload", s.auth.requireAuth(s.handleUpload))
	} else {
		mux.HandleFunc("/ws", s.handleWS)
		mux.HandleFunc("/upload", s.handleUpload)
	}

	s.mux = mux
	return s
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	s.indexTmpl.Execute(w, indexData{AuthEnabled: s.auth != nil})
}

func (s *server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 100 MB max
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		http.Error(w, "file too large", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Sanitize filename: only use base name, reject path traversal
	name := filepath.Base(header.Filename)
	if name == "." || name == ".." || name == "/" {
		http.Error(w, "invalid filename", http.StatusBadRequest)
		return
	}

	destPath := filepath.Join(s.uploadDir, name)

	dst, err := os.Create(destPath)
	if err != nil {
		log.Printf("upload: create %s: %v", destPath, err)
		http.Error(w, "failed to create file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	written, err := io.Copy(dst, file)
	if err != nil {
		log.Printf("upload: write %s: %v", destPath, err)
		http.Error(w, "failed to write file", http.StatusInternalServerError)
		return
	}

	log.Printf("upload: %s (%d bytes) from %s", name, written, r.RemoteAddr)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":   true,
		"name": name,
		"size": written,
		"path": destPath,
	})
}

func (s *server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade: %v", err)
		return
	}
	defer conn.Close()

	term, err := newTerminal(s.shell, 24, 80)
	if err != nil {
		log.Printf("spawn terminal: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte("failed to start shell: "+err.Error()))
		return
	}
	defer term.close()

	log.Printf("session started: remote=%s pid=%d", r.RemoteAddr, term.cmd.Process.Pid)

	// PTY -> WebSocket
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 4096)
		for {
			n, err := term.ptmx.Read(buf)
			if err != nil {
				return
			}
			if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
				return
			}
		}
	}()

	// WebSocket -> PTY
wsloop:
	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}

		switch msgType {
		case websocket.BinaryMessage:
			// Raw terminal input
			if _, err := term.ptmx.Write(msg); err != nil {
				log.Printf("pty write: %v", err)
				break wsloop
			}
		case websocket.TextMessage:
			// JSON control messages
			var m wsMessage
			if err := json.Unmarshal(msg, &m); err != nil {
				continue
			}
			switch m.Type {
			case "resize":
				var r resizeMsg
				if err := json.Unmarshal(m.Data, &r); err == nil && r.Rows > 0 && r.Cols > 0 {
					term.resize(r.Rows, r.Cols)
				}
			case "input":
				var input string
				if err := json.Unmarshal(m.Data, &input); err == nil {
					term.ptmx.Write([]byte(input))
				}
			}
		}
	}

	// Close the PTY to unblock the read goroutine, preventing deadlock.
	term.close()
	<-done
	log.Printf("session ended: remote=%s", r.RemoteAddr)
}
