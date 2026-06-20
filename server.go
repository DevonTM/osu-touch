package main

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/coder/websocket"
)

type indexTemplateData struct {
	AppName    string
	AppVersion string
	Key1       string
	Key2       string
}

func withServerHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", appName+"/"+appVersion)
		next.ServeHTTP(w, r)
	})
}

func requireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method == method {
		return true
	}
	w.Header().Set("Allow", method)
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	return false
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	data, err := webFiles.ReadFile("web/index.html")
	if err != nil {
		http.Error(w, "index.html not embedded", http.StatusInternalServerError)
		return
	}
	if err := renderIndex(w, data); err != nil {
		log.Printf("Template render error: %v", err)
		http.Error(w, "index.html render error", http.StatusInternalServerError)
	}
}

func renderIndex(w http.ResponseWriter, data []byte) error {
	keys := keyInput.Keys()
	tmpl, err := template.New("index.html").Parse(string(data))
	if err != nil {
		return err
	}

	var page bytes.Buffer
	if err := tmpl.Execute(&page, indexTemplateData{
		AppName:    appName,
		AppVersion: appVersion,
		Key1:       keys.First.Label,
		Key2:       keys.Second.Label,
	}); err != nil {
		return err
	}
	_, err = w.Write(page.Bytes())
	return err
}

func authHandler(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if !validPairingPIN(r.URL.Query().Get("pin"), pairingPIN) {
		// Match WebSocket auth failure timing so /auth is not the faster guessing path.
		time.Sleep(250 * time.Millisecond)
		http.Error(w, "invalid pairing pin", http.StatusUnauthorized)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	if !validPairingPIN(r.URL.Query().Get("pin"), pairingPIN) {
		// Slow down casual LAN guessing without leaking whether a PIN was close.
		time.Sleep(250 * time.Millisecond)
		http.Error(w, "invalid pairing pin", http.StatusUnauthorized)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // LAN tool: allow mobile browser origins without setup.
	})
	if err != nil {
		log.Printf("WebSocket accept error: %v", err)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "bye")

	remote := r.RemoteAddr
	log.Printf("WebSocket connected: %s", remote)
	var prev byte
	defer func() {
		releaseConnectionMask(prev)
		log.Printf("WebSocket disconnected: %s", remote)
	}()

	ctx := r.Context()
	for {
		msgType, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		if msgType != websocket.MessageBinary || len(data) != 1 {
			continue
		}

		newMask := data[0] & 0x03
		inputMu.Lock()
		if shuttingDown.Load() {
			inputMu.Unlock()
			return
		}
		if err := keyInput.ApplyMask(currentMask, newMask); err != nil {
			log.Printf("SendInput error: %v", err)
		}
		currentMask = newMask
		prev = newMask
		inputMu.Unlock()
	}
}

func releaseConnectionMask(mask byte) {
	inputMu.Lock()
	defer inputMu.Unlock()
	if shuttingDown.Load() || mask&0x03 == 0 {
		return
	}
	if err := keyInput.ApplyMask(currentMask, 0); err != nil {
		log.Printf("SendInput release error: %v", err)
	}
	currentMask = 0
}
