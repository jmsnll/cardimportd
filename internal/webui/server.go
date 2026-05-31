package webui

import (
	"context"
	"embed"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/jmsnll/cardimportd/internal/config"
)

//go:embed static
var staticFiles embed.FS

// ConfigAccessor is a thread-safe accessor pair passed from main.
type ConfigAccessor struct {
	Get  func() *config.Config
	Set  func(*config.Config)
	Path string // absolute path to config.yaml on disk
}

// Server is a small HTTP management server embedded in the daemon.
type Server struct {
	acc    ConfigAccessor
	runner *ImportRunner
	port   int
}

// New constructs a Server with the given accessor, runner, and listen port.
func New(acc ConfigAccessor, runner *ImportRunner, port int) *Server {
	return &Server{acc: acc, runner: runner, port: port}
}

// Start registers routes and listens until ctx is cancelled.
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// Static files — strip the "static" prefix from the embedded FS.
	stripped, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return err
	}
	mux.Handle("/", http.FileServer(http.FS(stripped)))

	// API routes.
	api := &apiHandler{acc: s.acc, runner: s.runner}
	mux.HandleFunc("/api/config", api.handleConfig)
	mux.HandleFunc("/api/cards", api.handleCards)
	mux.HandleFunc("/api/cards/", api.handleCard)   // /api/cards/{uuid}
	mux.HandleFunc("/api/notify/test", api.handleNotifyTest)

	addr := net.JoinHostPort("0.0.0.0", strconv.Itoa(s.port))
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	slog.Info("webui: listening", "addr", addr)

	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
