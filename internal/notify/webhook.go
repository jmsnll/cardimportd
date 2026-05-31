package notify

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type webhookNotifier struct {
	url    string
	secret string
	client *http.Client
}

// NewWebhookNotifier returns a Notifier that POSTs JSON-encoded events to url.
// If secret is non-empty, each request includes an X-Cardimportd-Signature
// header containing the HMAC-SHA256 of the body ("sha256=<hex>").
func NewWebhookNotifier(url, secret string) Notifier {
	return &webhookNotifier{
		url:    url,
		secret: secret,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (w *webhookNotifier) Notify(ctx context.Context, event Event) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("webhook: marshal event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.url,
		bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("webhook: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if w.secret != "" {
		mac := hmac.New(sha256.New, []byte(w.secret))
		mac.Write(body)
		req.Header.Set("X-Cardimportd-Signature",
			fmt.Sprintf("sha256=%x", mac.Sum(nil)))
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook: send: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook: unexpected status %d", resp.StatusCode)
	}
	return nil
}
