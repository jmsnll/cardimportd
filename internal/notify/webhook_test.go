package notify

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWebhookNotifier_Success(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ev := testEvent()
	n := NewWebhookNotifier(srv.URL, "")
	if err := n.Notify(context.Background(), ev); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var decoded Event
	if err := json.Unmarshal(gotBody, &decoded); err != nil {
		t.Fatalf("body is not valid JSON: %v", err)
	}
	if decoded.Kind != ev.Kind {
		t.Errorf("got kind %q, want %q", decoded.Kind, ev.Kind)
	}
}

func TestWebhookNotifier_HMACSignature(t *testing.T) {
	secret := "super-secret"
	var gotSig string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSig = r.Header.Get("X-Cardimportd-Signature")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := NewWebhookNotifier(srv.URL, secret)
	if err := n.Notify(context.Background(), testEvent()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(gotBody)
	want := fmt.Sprintf("sha256=%x", mac.Sum(nil))
	if gotSig != want {
		t.Errorf("HMAC mismatch\ngot:  %s\nwant: %s", gotSig, want)
	}
}

func TestWebhookNotifier_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	n := NewWebhookNotifier(srv.URL, "")
	if err := n.Notify(context.Background(), testEvent()); err == nil {
		t.Fatal("expected error for non-2xx status")
	}
}
