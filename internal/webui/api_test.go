package webui

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/jmsnll/cardimportd/internal/config"
	"github.com/jmsnll/cardimportd/internal/history"
)

// newTestAccessor creates a ConfigAccessor backed by a temp config file.
func newTestAccessor(t *testing.T) (ConfigAccessor, func() *config.Config) {
	t.Helper()
	cfg := &config.Config{
		WatchPaths:     []string{"/volumeUSB1"},
		ImportRoot:     "/volume1/photos",
		Cards:          make(map[string]config.CardEntry),
		FileExtensions: []string{".jpg", ".raf"},
	}
	f, err := os.CreateTemp(t.TempDir(), "config*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	cfgPath := f.Name()
	f.Close()
	if err := cfg.Save(cfgPath); err != nil {
		t.Fatal(err)
	}
	current := cfg
	acc := ConfigAccessor{
		Get:  func() *config.Config { return current },
		Set:  func(c *config.Config) { current = c },
		Path: cfgPath,
	}
	return acc, func() *config.Config { return current }
}

func newHandler(t *testing.T) (*apiHandler, func() *config.Config) {
	t.Helper()
	acc, getCurrent := newTestAccessor(t)
	hist := history.New(t.TempDir() + "/history.jsonl")
	return &apiHandler{acc: acc, bus: nil, hist: hist, getMounts: func() []MountedVolume { return nil }}, getCurrent
}

func jsonBody(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.NewBuffer(b)
}

// -- /api/config --

func TestHandleConfig_GET(t *testing.T) {
	h, _ := newHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	rr := httptest.NewRecorder()
	h.handleConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "import_root") {
		t.Error("response body missing import_root")
	}
}

func TestHandleConfig_POST_Valid(t *testing.T) {
	h, getCurrent := newHandler(t)
	body := jsonBody(t, map[string]any{
		"import_root": "/volume1/new",
		"watch_paths": []string{"/volumeUSB1"},
		"cards":       map[string]any{},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/config", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.handleConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rr.Code, rr.Body)
	}
	if getCurrent().ImportRoot != "/volume1/new" {
		t.Errorf("ImportRoot not updated: got %q", getCurrent().ImportRoot)
	}
}

func TestHandleConfig_POST_WrongMethod(t *testing.T) {
	h, _ := newHandler(t)
	req := httptest.NewRequest(http.MethodPut, "/api/config", nil)
	rr := httptest.NewRecorder()
	h.handleConfig(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rr.Code)
	}
}

func TestHandleConfig_POST_WrongContentType(t *testing.T) {
	h, _ := newHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/api/config", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()
	h.handleConfig(rr, req)
	if rr.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want 415", rr.Code)
	}
}

func TestHandleConfig_POST_MissingImportRoot(t *testing.T) {
	h, _ := newHandler(t)
	body := jsonBody(t, map[string]any{"watch_paths": []string{"/v"}})
	req := httptest.NewRequest(http.MethodPost, "/api/config", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.handleConfig(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestHandleConfig_POST_MissingWatchPaths(t *testing.T) {
	h, _ := newHandler(t)
	body := jsonBody(t, map[string]any{"import_root": "/volume1/photos"})
	req := httptest.NewRequest(http.MethodPost, "/api/config", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.handleConfig(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestHandleConfig_POST_InvalidJSON(t *testing.T) {
	h, _ := newHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/api/config", strings.NewReader("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.handleConfig(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

// -- /api/cards --

func TestHandleCards_GET(t *testing.T) {
	h, _ := newHandler(t)
	h.acc.Get().Cards["AABB-1234"] = config.CardEntry{Owner: "James", Status: config.StatusActive}

	req := httptest.NewRequest(http.MethodGet, "/api/cards", nil)
	rr := httptest.NewRecorder()
	h.handleCards(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "AABB-1234") {
		t.Error("response missing card UUID")
	}
}

func TestHandleCards_WrongMethod(t *testing.T) {
	h, _ := newHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/api/cards", nil)
	rr := httptest.NewRecorder()
	h.handleCards(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rr.Code)
	}
}

// -- /api/cards/{uuid} --

func TestHandleCard_Update(t *testing.T) {
	h, getCurrent := newHandler(t)
	body := jsonBody(t, map[string]any{"owner": "Sophie", "status": "active"})
	req := httptest.NewRequest(http.MethodPost, "/api/cards/AABB-1234", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.handleCard(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rr.Code, rr.Body)
	}
	entry, ok := getCurrent().Cards["AABB-1234"]
	if !ok {
		t.Fatal("card not found after update")
	}
	if entry.Owner != "Sophie" || entry.Status != config.StatusActive {
		t.Errorf("card = %+v, want {Owner:Sophie, Status:active}", entry)
	}
}

func TestHandleCard_Update_InvalidStatus(t *testing.T) {
	h, _ := newHandler(t)
	body := jsonBody(t, map[string]any{"owner": "James", "status": "unknown"})
	req := httptest.NewRequest(http.MethodPost, "/api/cards/AABB-1234", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.handleCard(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestHandleCard_Delete(t *testing.T) {
	h, getCurrent := newHandler(t)
	h.acc.Get().Cards["AABB-1234"] = config.CardEntry{Owner: "James", Status: config.StatusActive}

	req := httptest.NewRequest(http.MethodDelete, "/api/cards/AABB-1234", nil)
	rr := httptest.NewRecorder()
	h.handleCard(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rr.Code, rr.Body)
	}
	if _, ok := getCurrent().Cards["AABB-1234"]; ok {
		t.Error("card still present after delete")
	}
}

func TestHandleCard_Delete_NotFound(t *testing.T) {
	h, _ := newHandler(t)
	req := httptest.NewRequest(http.MethodDelete, "/api/cards/NONEXISTENT", nil)
	rr := httptest.NewRecorder()
	h.handleCard(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
}

func TestHandleCard_WrongMethod(t *testing.T) {
	h, _ := newHandler(t)
	req := httptest.NewRequest(http.MethodPut, "/api/cards/AABB-1234", nil)
	rr := httptest.NewRecorder()
	h.handleCard(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rr.Code)
	}
}

// -- /api/notify/test --

func TestHandleNotifyTest_WrongMethod(t *testing.T) {
	h, _ := newHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/notify/test", nil)
	rr := httptest.NewRecorder()
	h.handleNotifyTest(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rr.Code)
	}
}

func TestHandleNotifyTest_PushoverNotConfigured(t *testing.T) {
	h, _ := newHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/api/notify/test?adapter=pushover", nil)
	rr := httptest.NewRecorder()
	h.handleNotifyTest(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestHandleNotifyTest_NtfyNotConfigured(t *testing.T) {
	h, _ := newHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/api/notify/test?adapter=ntfy", nil)
	rr := httptest.NewRecorder()
	h.handleNotifyTest(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestHandleNotifyTest_WebhookSuccess(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	h, _ := newHandler(t)
	h.acc.Get().Notifications.Webhook = &config.WebhookConfig{URL: upstream.URL}

	req := httptest.NewRequest(http.MethodPost, "/api/notify/test?adapter=webhook", nil)
	rr := httptest.NewRecorder()
	h.handleNotifyTest(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rr.Code, rr.Body)
	}
}

func TestHandleNotifyTest_NtfySuccess(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	h, _ := newHandler(t)
	h.acc.Get().Notifications.Ntfy = &config.NtfyConfig{URL: upstream.URL}

	req := httptest.NewRequest(http.MethodPost, "/api/notify/test?adapter=ntfy", nil)
	rr := httptest.NewRecorder()
	h.handleNotifyTest(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rr.Code, rr.Body)
	}
}

func TestHandleNotifyTest_UnknownAdapter(t *testing.T) {
	h, _ := newHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/api/notify/test?adapter=telegram", nil)
	rr := httptest.NewRecorder()
	h.handleNotifyTest(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

// -- /api/history --

func TestHandleHistory_EmptyReturnsArray(t *testing.T) {
	h, _ := newHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/history", nil)
	rr := httptest.NewRecorder()
	h.handleHistory(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var entries []history.Entry
	if err := json.Unmarshal(rr.Body.Bytes(), &entries); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if entries == nil {
		t.Error("expected non-nil empty array, got nil")
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestHandleHistory_LimitQueryParam(t *testing.T) {
	h, _ := newHandler(t)
	// Append 10 entries.
	for i := 0; i < 10; i++ {
		if err := h.hist.Append(history.Entry{UUID: "test-uuid", Owner: "james"}); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/history?limit=5", nil)
	rr := httptest.NewRecorder()
	h.handleHistory(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var entries []history.Entry
	if err := json.Unmarshal(rr.Body.Bytes(), &entries); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(entries) != 5 {
		t.Errorf("expected 5 entries with limit=5, got %d", len(entries))
	}
}

func TestHandleHistory_WrongMethod(t *testing.T) {
	h, _ := newHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/api/history", nil)
	rr := httptest.NewRecorder()
	h.handleHistory(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rr.Code)
	}
}

// -- /api/status --

func TestHandleStatus_NoMounts(t *testing.T) {
	h, _ := newHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rr := httptest.NewRecorder()
	h.handleStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var resp StatusResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.MountedCards == nil {
		t.Error("mounted_cards should not be nil")
	}
	if len(resp.MountedCards) != 0 {
		t.Errorf("expected 0 mounted cards, got %d", len(resp.MountedCards))
	}
	if resp.ActiveImport != nil {
		t.Error("active_import should be nil when no import is running")
	}
}

func TestHandleStatus_WithMounts(t *testing.T) {
	h, _ := newHandler(t)
	h.getMounts = func() []MountedVolume {
		return []MountedVolume{
			{UUID: "AABB-1234", MountPoint: "/volumeUSB1/usbshare1", Device: "/dev/sda1", FSType: "exfat"},
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rr := httptest.NewRecorder()
	h.handleStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var resp StatusResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.MountedCards) != 1 {
		t.Errorf("expected 1 mounted card, got %d", len(resp.MountedCards))
	}
	if resp.MountedCards[0].UUID != "AABB-1234" {
		t.Errorf("unexpected UUID: %q", resp.MountedCards[0].UUID)
	}
}

func TestHandleStatus_WrongMethod(t *testing.T) {
	h, _ := newHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/api/status", nil)
	rr := httptest.NewRecorder()
	h.handleStatus(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rr.Code)
	}
}

// -- /api/preflight/{uuid} --

func TestHandlePreflight_NotMounted(t *testing.T) {
	h, _ := newHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/preflight/AABB-1234", nil)
	rr := httptest.NewRecorder()
	h.handlePreflight(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
}

func TestHandlePreflight_WrongMethod(t *testing.T) {
	h, _ := newHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/api/preflight/AABB-1234", nil)
	rr := httptest.NewRecorder()
	h.handlePreflight(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rr.Code)
	}
}

func TestHandlePreflight_MissingUUID(t *testing.T) {
	h, _ := newHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/preflight/", nil)
	rr := httptest.NewRecorder()
	h.handlePreflight(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestHandlePreflight_CountsFiles(t *testing.T) {
	dir := t.TempDir()
	// Create some fake files.
	for _, name := range []string{"IMG_001.jpg", "IMG_002.RAF", "README.txt"} {
		f, err := os.Create(dir + "/" + name)
		if err != nil {
			t.Fatal(err)
		}
		f.Close()
	}

	h, _ := newHandler(t)
	h.getMounts = func() []MountedVolume {
		return []MountedVolume{
			{UUID: "AABB-1234", MountPoint: dir},
		}
	}
	// Config already has .jpg and .raf extensions from newTestAccessor.
	req := httptest.NewRequest(http.MethodGet, "/api/preflight/AABB-1234", nil)
	rr := httptest.NewRecorder()
	h.handlePreflight(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rr.Code, rr.Body)
	}
	var result struct {
		UUID     string `json:"uuid"`
		OnCard   int    `json:"total_on_card"`
		ToImport int    `json:"to_import"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.UUID != "AABB-1234" {
		t.Errorf("uuid = %q, want AABB-1234", result.UUID)
	}
	// Expects 2 matching files (.jpg and .raf), not the .txt
	if result.OnCard != 2 {
		t.Errorf("total_on_card = %d, want 2", result.OnCard)
	}
	if result.ToImport != 2 {
		t.Errorf("to_import = %d, want 2", result.ToImport)
	}
}
