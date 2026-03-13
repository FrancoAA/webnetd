package main

import (
	"crypto/rand"
	"crypto/subtle"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"sync"
	"time"
)

type auth struct {
	pin     string
	tokens  map[string]time.Time
	mu      sync.Mutex
}

func newAuth() *auth {
	pin := generatePIN()
	return &auth{
		pin:    pin,
		tokens: make(map[string]time.Time),
	}
}

func generatePIN() string {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		panic("failed to generate PIN: " + err.Error())
	}
	return fmt.Sprintf("%06d", n.Int64())
}

func (a *auth) verifyPIN(pin string) bool {
	return subtle.ConstantTimeCompare([]byte(a.pin), []byte(pin)) == 1
}

func (a *auth) createToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("failed to generate token: " + err.Error())
	}
	token := fmt.Sprintf("%x", b)

	a.mu.Lock()
	a.tokens[token] = time.Now().Add(24 * time.Hour)
	a.mu.Unlock()

	return token
}

func (a *auth) validToken(token string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	exp, ok := a.tokens[token]
	if !ok {
		return false
	}
	if time.Now().After(exp) {
		delete(a.tokens, token)
		return false
	}
	return true
}

func (a *auth) tokenFromRequest(r *http.Request) string {
	if c, err := r.Cookie("webnetd_token"); err == nil {
		return c.Value
	}
	return r.URL.Query().Get("token")
}

func (a *auth) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := a.tokenFromRequest(r)
		if !a.validToken(token) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func (a *auth) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	pin := r.FormValue("pin")
	if !a.verifyPIN(pin) {
		log.Printf("auth: failed login attempt from %s", r.RemoteAddr)
		http.Error(w, "invalid pin", http.StatusForbidden)
		return
	}

	token := a.createToken()
	http.SetCookie(w, &http.Cookie{
		Name:     "webnetd_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400,
	})

	log.Printf("auth: successful login from %s", r.RemoteAddr)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"ok":true,"token":%q}`, token)
}
