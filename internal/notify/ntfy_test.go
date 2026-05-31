package notify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNtfyNotifier_Success(t *testing.T) {
	var gotAuth, gotTitle string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotTitle = r.Header.Get("Title")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := NewNtfyNotifier(srv.URL, "")
	if err := n.Notify(context.Background(), testEvent()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAuth != "" {
		t.Errorf("expected no auth header, got %q", gotAuth)
	}
	if gotTitle == "" {
		t.Error("expected Title header to be set")
	}
}

func TestNtfyNotifier_BearerToken(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := NewNtfyNotifier(srv.URL, "secret-token")
	if err := n.Notify(context.Background(), testEvent()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAuth != "Bearer secret-token" {
		t.Errorf("expected Bearer token, got %q", gotAuth)
	}
}

func TestNtfyNotifier_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	n := NewNtfyNotifier(srv.URL, "")
	if err := n.Notify(context.Background(), testEvent()); err == nil {
		t.Fatal("expected error for non-2xx status")
	}
}

func testEvent() Event {
	return Event{
		Kind:      KindImportCompleted,
		CardUUID:  "test-uuid",
		Owner:     "Alice",
		MountPath: "/volumeUSB1/usbshare",
		Time:      time.Now(),
		Stats: &ImportStats{
			Total: 10, Imported: 9, Skipped: 1,
			Duration: 5 * time.Second,
		},
	}
}
