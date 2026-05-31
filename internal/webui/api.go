package webui

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jmsnll/cardimportd/internal/config"
	"github.com/jmsnll/cardimportd/internal/notify"
)

type apiHandler struct {
	acc ConfigAccessor
	bus *EventBus
}

func writeJSON(w http.ResponseWriter, v any, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("webui: encode response", "err", err)
	}
}

func apiError(w http.ResponseWriter, msg string, status int) {
	writeJSON(w, map[string]string{"error": msg}, status)
}

func requireJSON(w http.ResponseWriter, r *http.Request) bool {
	ct := r.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		apiError(w, "content-type must be application/json", http.StatusUnsupportedMediaType)
		return false
	}
	return true
}

// -- /api/config --------------------------------------------------------------

func (h *apiHandler) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, h.acc.Get(), http.StatusOK)
	case http.MethodPost:
		h.postConfig(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *apiHandler) postConfig(w http.ResponseWriter, r *http.Request) {
	if !requireJSON(w, r) {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
	var newCfg config.Config
	if err := json.NewDecoder(r.Body).Decode(&newCfg); err != nil {
		apiError(w, fmt.Sprintf("invalid body: %s", err), http.StatusBadRequest)
		return
	}
	if newCfg.ImportRoot == "" {
		apiError(w, "import_root must not be empty", http.StatusBadRequest)
		return
	}
	if len(newCfg.WatchPaths) == 0 {
		apiError(w, "watch_paths must contain at least one entry", http.StatusBadRequest)
		return
	}
	if newCfg.Cards == nil {
		newCfg.Cards = make(map[string]config.CardEntry)
	}

	if err := newCfg.Save(h.acc.Path); err != nil {
		slog.Error("webui: save config", "err", err)
		apiError(w, fmt.Sprintf("failed to save config: %s", err), http.StatusInternalServerError)
		return
	}
	h.acc.Set(&newCfg)
	writeJSON(w, map[string]bool{"ok": true}, http.StatusOK)
}

// -- /api/cards ---------------------------------------------------------------

func (h *apiHandler) handleCards(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, h.acc.Get().Cards, http.StatusOK)
}

// -- /api/cards/{uuid} --------------------------------------------------------

func (h *apiHandler) handleCard(w http.ResponseWriter, r *http.Request) {
	uuid := strings.TrimPrefix(r.URL.Path, "/api/cards/")
	if uuid == "" {
		apiError(w, "uuid is required", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodPost:
		h.updateCard(w, r, uuid)
	case http.MethodDelete:
		h.deleteCard(w, r, uuid)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

type cardUpdateRequest struct {
	Owner  string            `json:"owner"`
	Status config.CardStatus `json:"status"`
}

func (h *apiHandler) updateCard(w http.ResponseWriter, r *http.Request, uuid string) {
	if !requireJSON(w, r) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
	var req cardUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiError(w, fmt.Sprintf("invalid body: %s", err), http.StatusBadRequest)
		return
	}
	if req.Status != config.StatusActive && req.Status != config.StatusPending {
		apiError(w, fmt.Sprintf("status must be %q or %q", config.StatusActive, config.StatusPending), http.StatusBadRequest)
		return
	}

	updated := copyConfig(h.acc.Get())
	entry, exists := updated.Cards[uuid]
	if !exists {
		now := time.Now().UTC()
		entry = config.CardEntry{FirstSeen: &now}
	}
	entry.Owner = req.Owner
	entry.Status = req.Status
	updated.Cards[uuid] = entry

	if err := updated.Save(h.acc.Path); err != nil {
		slog.Error("webui: save config", "err", err)
		apiError(w, fmt.Sprintf("failed to save config: %s", err), http.StatusInternalServerError)
		return
	}
	h.acc.Set(updated)
	writeJSON(w, map[string]bool{"ok": true}, http.StatusOK)
}

func (h *apiHandler) deleteCard(w http.ResponseWriter, r *http.Request, uuid string) {
	current := h.acc.Get()
	if _, exists := current.Cards[uuid]; !exists {
		apiError(w, fmt.Sprintf("card %q not found", uuid), http.StatusNotFound)
		return
	}
	updated := copyConfig(current)
	delete(updated.Cards, uuid)

	if err := updated.Save(h.acc.Path); err != nil {
		slog.Error("webui: save config", "err", err)
		apiError(w, fmt.Sprintf("failed to save config: %s", err), http.StatusInternalServerError)
		return
	}
	h.acc.Set(updated)
	writeJSON(w, map[string]bool{"ok": true}, http.StatusOK)
}

// -- /api/notify/test ---------------------------------------------------------

func (h *apiHandler) handleNotifyTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	adapter := r.URL.Query().Get("adapter")
	cfg := h.acc.Get()

	var n notify.Notifier
	switch adapter {
	case "", "pushover":
		po := cfg.Notifications.Pushover
		if po == nil || po.AppToken == "" || po.UserKey == "" {
			apiError(w, "pushover is not configured", http.StatusBadRequest)
			return
		}
		n = notify.NewPushoverNotifier(po.AppToken, po.UserKey)
	case "ntfy":
		nt := cfg.Notifications.Ntfy
		if nt == nil || nt.URL == "" {
			apiError(w, "ntfy is not configured", http.StatusBadRequest)
			return
		}
		n = notify.NewNtfyNotifier(nt.URL, nt.Token)
	case "webhook":
		wh := cfg.Notifications.Webhook
		if wh == nil || wh.URL == "" {
			apiError(w, "webhook is not configured", http.StatusBadRequest)
			return
		}
		n = notify.NewWebhookNotifier(wh.URL, wh.Secret)
	default:
		apiError(w, fmt.Sprintf("unknown adapter %q", adapter), http.StatusBadRequest)
		return
	}

	ev := notify.Event{
		Kind:   notify.KindImportCompleted,
		Detail: "test notification from cardimportd webui",
		Time:   time.Now().UTC(),
	}
	if err := n.Notify(context.Background(), ev); err != nil {
		slog.Error("webui: notify test", "adapter", adapter, "err", err)
		apiError(w, fmt.Sprintf("notification failed: %s", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]bool{"ok": true}, http.StatusOK)
}

// -- /api/events --------------------------------------------------------------

func (h *apiHandler) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	if h.bus == nil {
		fmt.Fprintf(w, ": keep-alive\n\n")
		flusher.Flush()
		return
	}
	ch := h.bus.Subscribe()
	defer h.bus.Unsubscribe(ch)
	for {
		select {
		case <-r.Context().Done():
			return
		case evt, ok := <-ch:
			if !ok {
				return
			}
			if data, err := evt.Marshal(); err == nil {
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			}
		}
	}
}

// -- /api/fs ------------------------------------------------------------------

type fsResponse struct {
	Path    string   `json:"path"`
	Entries []string `json:"entries"`
}

func (h *apiHandler) handleFS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		path = "/"
	}
	path = filepath.Clean(path)

	info, err := os.Stat(path)
	if err != nil {
		apiError(w, fmt.Sprintf("path not found: %s", err), http.StatusBadRequest)
		return
	}
	if !info.IsDir() {
		apiError(w, fmt.Sprintf("%q is not a directory", path), http.StatusBadRequest)
		return
	}

	des, err := os.ReadDir(path)
	if err != nil {
		apiError(w, fmt.Sprintf("cannot read directory: %s", err), http.StatusBadRequest)
		return
	}

	names := make([]string, 0, len(des))
	for _, de := range des {
		if de.IsDir() {
			names = append(names, de.Name())
		}
	}
	sort.Strings(names)

	writeJSON(w, fsResponse{Path: path, Entries: names}, http.StatusOK)
}

// -- helpers ------------------------------------------------------------------

func copyConfig(src *config.Config) *config.Config {
	dst := *src
	dst.Cards = make(map[string]config.CardEntry, len(src.Cards))
	for k, v := range src.Cards {
		dst.Cards[k] = v
	}
	dst.WatchPaths = append([]string(nil), src.WatchPaths...)
	dst.FileExtensions = append([]string(nil), src.FileExtensions...)
	return &dst
}
