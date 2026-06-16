package main

import (
	"context"
	"embed"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/coder/websocket"
)

//go:embed web/index.html
var webFiles embed.FS

const addr = ":8080"

var (
	inputMu      sync.Mutex
	currentMask  byte
	shutdownOnce sync.Once
	shuttingDown atomic.Bool
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/ws", wsHandler)

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("Local URL: http://localhost%s", addr)
	for _, ip := range lanIPs() {
		log.Printf("LAN URL:   http://%s%s", ip, addr)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		finalReleaseAll("shutdown")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("HTTP shutdown error: %v", err)
		}
	}()

	log.Printf("Listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("HTTP server error: %v", err)
	}

	finalReleaseAll("exit")
}

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
		if err := applyMask(currentMask, newMask); err != nil {
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
	if err := applyMask(currentMask, 0); err != nil {
		log.Printf("SendInput release error: %v", err)
	}
	currentMask = 0
}

func finalReleaseAll(reason string) {
	shutdownOnce.Do(func() {
		shuttingDown.Store(true)
		log.Printf("Releasing all keys (%s)...", reason)
		inputMu.Lock()
		defer inputMu.Unlock()
		if err := releaseAll(); err != nil {
			log.Printf("SendInput releaseAll error: %v", err)
		}
		currentMask = 0
	})
}

func lanIPs() []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Printf("Could not list network interfaces: %v", err)
		return nil
	}

	var ips []string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip4 := ip.To4()
			if ip4 == nil || !isPrivateIPv4(ip4) {
				continue
			}
			ips = append(ips, ip4.String())
		}
	}
	return ips
}

func isPrivateIPv4(ip net.IP) bool {
	text := ip.String()
	return strings.HasPrefix(text, "10.") ||
		strings.HasPrefix(text, "192.168.") ||
		strings.HasPrefix(text, "172.16.") || strings.HasPrefix(text, "172.17.") ||
		strings.HasPrefix(text, "172.18.") || strings.HasPrefix(text, "172.19.") ||
		strings.HasPrefix(text, "172.20.") || strings.HasPrefix(text, "172.21.") ||
		strings.HasPrefix(text, "172.22.") || strings.HasPrefix(text, "172.23.") ||
		strings.HasPrefix(text, "172.24.") || strings.HasPrefix(text, "172.25.") ||
		strings.HasPrefix(text, "172.26.") || strings.HasPrefix(text, "172.27.") ||
		strings.HasPrefix(text, "172.28.") || strings.HasPrefix(text, "172.29.") ||
		strings.HasPrefix(text, "172.30.") || strings.HasPrefix(text, "172.31.")
}
