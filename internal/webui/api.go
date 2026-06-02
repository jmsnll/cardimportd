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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jmsnll/cardimportd/internal/config"
	"github.com/jmsnll/cardimportd/internal/history"
	"github.com/jmsnll/cardimportd/internal/notify"
)

// MountedVolume describes a card currently visible to the daemon.
type MountedVolume struct {
	UUID       string `json:"uuid"`
	MountPoint string `json:"mount_point"`
	Device     string `json:"device"`
	FSType     string `json:"fstype"`
}

// StatusResponse is returned by GET /api/status.
type StatusResponse struct {
	MountedCards []MountedVolume `json:"mounted_cards"`
	ActiveImport *ProgressEvent  `json:"active_import"`
}

type apiHandler struct {
	acc       ConfigAccessor
	bus       *EventBus
	hist      *history.Log
	getMounts func() []MountedVolume

	mu           sync.RWMutex
	activeImport *ProgressEvent
}

// startEventLoop subscribes to the bus and keeps activeImport up-to-date.
func (h *apiHandler) startEventLoop(ctx context.Context) {
	if h.bus == nil {
		return
	}
	ch := h.bus.Subscribe()
	go func() {
		defer h.bus.Unsubscribe(ch)
		for {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-ch:
				if !ok {
					return
				}
				switch evt.Kind {
				case ProgressKindStarted, ProgressKindProgress:
					cp := evt
					h.mu.Lock()
					h.activeImport = &cp
					h.mu.Unlock()
				case ProgressKindCompleted, ProgressKindFailed:
					h.mu.Lock()
					h.activeImport = nil
					h.mu.Unlock()
				}
			}
		}
	}()
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
		slog.Error("webui: notify test", "adapter", adapter, "err", err) //nolint:gosec // adapter is validated via switch before reaching this line
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
		_, _ = fmt.Fprintf(w, ": keep-alive\n\n")
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
				_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
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

// -- /healthz -----------------------------------------------------------------

func (h *apiHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
}

// -- /api/history -------------------------------------------------------------

func (h *apiHandler) handleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	limit := 50
	if s := r.URL.Query().Get("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			if n > 500 {
				n = 500
			}
			limit = n
		}
	}
	entries, err := h.hist.Recent(limit)
	if err != nil {
		slog.Error("webui: history read", "err", err)
		apiError(w, "failed to read history", http.StatusInternalServerError)
		return
	}
	if entries == nil {
		entries = []history.Entry{}
	}
	writeJSON(w, entries, http.StatusOK)
}

// -- /api/status --------------------------------------------------------------

func (h *apiHandler) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	h.mu.RLock()
	ai := h.activeImport
	h.mu.RUnlock()

	var mounts []MountedVolume
	if h.getMounts != nil {
		mounts = h.getMounts()
	}
	if mounts == nil {
		mounts = []MountedVolume{}
	}

	resp := StatusResponse{
		MountedCards: mounts,
		ActiveImport: ai,
	}
	writeJSON(w, resp, http.StatusOK)
}

// -- /api/preflight/{uuid} ----------------------------------------------------

func (h *apiHandler) handlePreflight(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	uuid := strings.TrimPrefix(r.URL.Path, "/api/preflight/")
	if uuid == "" {
		apiError(w, "uuid is required", http.StatusBadRequest)
		return
	}

	var mountPoint string
	if h.getMounts != nil {
		for _, m := range h.getMounts() {
			if m.UUID == uuid {
				mountPoint = m.MountPoint
				break
			}
		}
	}
	if mountPoint == "" {
		apiError(w, "card not currently mounted", http.StatusNotFound)
		return
	}

	cfg := h.acc.Get()
	extensions := make(map[string]bool, len(cfg.FileExtensions))
	for _, ext := range cfg.FileExtensions {
		extensions[strings.ToLower(ext)] = true
	}

	// TODO: future improvement — check destination to compute already-imported
	// count and return a more accurate to_import value.
	type preflightResult struct {
		UUID     string `json:"uuid"`
		OnCard   int    `json:"total_on_card"`
		ToImport int    `json:"to_import"`
	}
	var onCard int
	_ = filepath.Walk(mountPoint, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if extensions[strings.ToLower(filepath.Ext(path))] {
			onCard++
		}
		return nil
	})
	writeJSON(w, preflightResult{UUID: uuid, OnCard: onCard, ToImport: onCard}, http.StatusOK)
}

// -- /api/users ---------------------------------------------------------------

func (h *apiHandler) handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		users := h.acc.Get().Users
		if users == nil {
			users = []config.User{}
		}
		writeJSON(w, users, http.StatusOK)
	case http.MethodPost:
		h.createUser(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *apiHandler) createUser(w http.ResponseWriter, r *http.Request) {
	if !requireJSON(w, r) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req config.User
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiError(w, "invalid body: "+err.Error(), http.StatusBadRequest)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		apiError(w, "name is required", http.StatusBadRequest)
		return
	}
	updated := copyConfig(h.acc.Get())
	for _, u := range updated.Users {
		if u.Name == req.Name {
			apiError(w, "user already exists", http.StatusConflict)
			return
		}
	}
	updated.Users = append(updated.Users, req)
	if err := updated.Save(h.acc.Path); err != nil {
		apiError(w, "failed to save: "+err.Error(), http.StatusInternalServerError)
		return
	}
	h.acc.Set(updated)
	writeJSON(w, map[string]bool{"ok": true}, http.StatusOK)
}

// -- /api/users/{name} --------------------------------------------------------

func (h *apiHandler) handleUser(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/users/")
	if name == "" {
		apiError(w, "name is required", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodPut:
		h.updateUser(w, r, name)
	case http.MethodDelete:
		h.deleteUser(w, r, name)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *apiHandler) updateUser(w http.ResponseWriter, r *http.Request, name string) {
	if !requireJSON(w, r) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req struct {
		Name                string `json:"name"`
		DestinationTemplate string `json:"destination_template"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiError(w, "invalid body: "+err.Error(), http.StatusBadRequest)
		return
	}
	updated := copyConfig(h.acc.Get())
	found := false
	for i, u := range updated.Users {
		if u.Name != name {
			continue
		}
		newName := strings.TrimSpace(req.Name)
		if newName == "" {
			newName = name
		}
		updated.Users[i].Name = newName
		updated.Users[i].DestinationTemplate = req.DestinationTemplate
		if newName != name {
			for uuid, entry := range updated.Cards {
				if entry.Owner == name {
					entry.Owner = newName
					updated.Cards[uuid] = entry
				}
			}
		}
		found = true
		break
	}
	if !found {
		apiError(w, "user not found", http.StatusNotFound)
		return
	}
	if err := updated.Save(h.acc.Path); err != nil {
		apiError(w, "failed to save: "+err.Error(), http.StatusInternalServerError)
		return
	}
	h.acc.Set(updated)
	writeJSON(w, map[string]bool{"ok": true}, http.StatusOK)
}

func (h *apiHandler) deleteUser(w http.ResponseWriter, r *http.Request, name string) {
	current := h.acc.Get()
	var blocked []string
	for uuid, entry := range current.Cards {
		if entry.Owner == name {
			blocked = append(blocked, uuid)
		}
	}
	if len(blocked) > 0 {
		sort.Strings(blocked)
		writeJSON(w, map[string]any{
			"error": "user is referenced by cards; reassign or rename first",
			"cards": blocked,
		}, http.StatusConflict)
		return
	}
	updated := copyConfig(current)
	newUsers := updated.Users[:0]
	found := false
	for _, u := range updated.Users {
		if u.Name == name {
			found = true
			continue
		}
		newUsers = append(newUsers, u)
	}
	if !found {
		apiError(w, "user not found", http.StatusNotFound)
		return
	}
	updated.Users = newUsers
	if err := updated.Save(h.acc.Path); err != nil {
		apiError(w, "failed to save: "+err.Error(), http.StatusInternalServerError)
		return
	}
	h.acc.Set(updated)
	writeJSON(w, map[string]bool{"ok": true}, http.StatusOK)
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
	dst.Users = append([]config.User(nil), src.Users...)
	dst.WebUI = src.WebUI
	return &dst
}
