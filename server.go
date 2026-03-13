package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type server struct {
	shell    string
	upgrader websocket.Upgrader
	mux      *http.ServeMux
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

func newServer(shell string) *server {
	s := &server{
		shell: shell,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/ws", s.handleWS)
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
	w.Write([]byte(indexHTML))
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
				if err != io.EOF {
					log.Printf("pty read: %v", err)
				}
				return
			}
			if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
				return
			}
		}
	}()

	// WebSocket -> PTY
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
				return
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

	<-done
	log.Printf("session ended: remote=%s", r.RemoteAddr)
}
