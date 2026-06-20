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
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/DevonTM/osu-touch/input"
)

//go:embed web/index.html
var webFiles embed.FS

var (
	keyInput     *input.Controller
	inputMu      sync.Mutex
	currentMask  byte
	pairingPIN   string
	shutdownOnce sync.Once
	shuttingDown atomic.Bool
)

func main() {
	setConsoleTitle(appName)
	log.Printf("%s v%s - wireless touch keypad for osu!", appName, appVersion)

	keyInput = input.NewController(inputKeys())
	log.Printf("Input keys: %s / %s", keyInput.Keys().First.Label, keyInput.Keys().Second.Label)

	var err error
	pairingPIN, err = newPairingPIN()
	if err != nil {
		log.Fatalf("Pairing PIN generation error: %v", err)
	}
	log.Printf("Pairing PIN: %s", pairingPIN)

	mux := http.NewServeMux()
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/ws", wsHandler)

	server := &http.Server{
		Handler:           withServerHeader(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}
	listener, err := net.Listen("tcp", serverAddr())
	if err != nil {
		log.Fatalf("HTTP listen error: %v", err)
	}

	logServerURLs(listener.Addr())
	log.Println("Server is ready")
	log.Println("Open the LAN URL on your phone")
	log.Println("Waiting for WebSocket client...")

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

	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("HTTP server error: %v", err)
	}

	finalReleaseAll("exit")
}

func finalReleaseAll(reason string) {
	shutdownOnce.Do(func() {
		shuttingDown.Store(true)
		log.Printf("Releasing all keys (%s)...", reason)
		inputMu.Lock()
		defer inputMu.Unlock()
		if err := keyInput.ReleaseAll(); err != nil {
			log.Printf("SendInput releaseAll error: %v", err)
		}
		currentMask = 0
	})
}
