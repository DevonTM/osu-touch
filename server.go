package main

import (
	"log"
	"net/http"

	"github.com/coder/websocket"

	"osu-touch/input"
)

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	data, err := webFiles.ReadFile("web/index.html")
	if err != nil {
		http.Error(w, "index.html not embedded", http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(data)
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // LAN tool: allow phone browser origins without setup.
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
		if err := input.ApplyMask(currentMask, newMask); err != nil {
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
	if err := input.ApplyMask(currentMask, 0); err != nil {
		log.Printf("SendInput release error: %v", err)
	}
	currentMask = 0
}
