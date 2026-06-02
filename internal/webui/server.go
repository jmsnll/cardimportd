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
	"github.com/jmsnll/cardimportd/internal/history"
)

//go:embed all:static
var staticFiles embed.FS

// ConfigAccessor is a thread-safe accessor pair passed from main.
type ConfigAccessor struct {
	Get  func() *config.Config
	Set  func(*config.Config)
	Path string // absolute path to config.yaml on disk
}

// Server is a small HTTP management server embedded in the daemon.
type Server struct {
	acc       ConfigAccessor
	bus       *EventBus
	hist      *history.Log
	getMounts func() []MountedVolume
	port      int
}

// New constructs a Server with the given accessor, event bus, history log, mount accessor, and listen port.
func New(acc ConfigAccessor, bus *EventBus, hist *history.Log, getMounts func() []MountedVolume, port int) *Server {
	return &Server{acc: acc, bus: bus, hist: hist, getMounts: getMounts, port: port}
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
	api := &apiHandler{acc: s.acc, bus: s.bus, hist: s.hist, getMounts: s.getMounts}
	api.startEventLoop(ctx)
	mux.HandleFunc("/api/config", api.handleConfig)
	mux.HandleFunc("/api/cards", api.handleCards)
	mux.HandleFunc("/api/cards/", api.handleCard) // /api/cards/{uuid}
	mux.HandleFunc("/api/notify/test", api.handleNotifyTest)
	mux.HandleFunc("/api/events", api.handleEvents)
	mux.HandleFunc("/api/fs", api.handleFS)
	mux.HandleFunc("/api/history", api.handleHistory)
	mux.HandleFunc("/api/status", api.handleStatus)
	mux.HandleFunc("/api/preflight/", api.handlePreflight)

	// Health endpoint — exempt from auth.
	mux.HandleFunc("/healthz", api.handleHealth)

	// Resolve bind address from config, defaulting to 0.0.0.0.
	webuiCfg := s.acc.Get().WebUI
	bindAddr := webuiCfg.BindAddress
	if bindAddr == "" {
		bindAddr = "0.0.0.0"
	}

	// Wrap mux with basic auth middleware when credentials are configured.
	var handler http.Handler = mux
	if webuiCfg.Username != "" && webuiCfg.Password != "" {
		username := webuiCfg.Username
		password := webuiCfg.Password
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/healthz" {
				mux.ServeHTTP(w, r)
				return
			}
			u, p, ok := r.BasicAuth()
			if !ok || u != username || p != password {
				w.Header().Set("WWW-Authenticate", `Basic realm="cardimportd"`)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			mux.ServeHTTP(w, r)
		})
	}

	addr := net.JoinHostPort(bindAddr, strconv.Itoa(s.port))
	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	slog.Info("webui: listening", "addr", addr)

	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second) //nolint:gosec // ctx is already cancelled; Background is correct here
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
