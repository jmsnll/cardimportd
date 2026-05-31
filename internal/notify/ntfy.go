package notify

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type ntfyNotifier struct {
	url    string
	token  string
	client *http.Client
}

// NewNtfyNotifier returns a Notifier that delivers events to an ntfy topic.
// url must be the full topic URL (e.g. https://ntfy.sh/mycards).
// token is optional; when non-empty it is sent as a Bearer token.
func NewNtfyNotifier(url, token string) Notifier {
	return &ntfyNotifier{
		url:    url,
		token:  token,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (n *ntfyNotifier) Notify(ctx context.Context, event Event) error {
	title, message := formatEvent(event)
	body := title + "\n" + message

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.url,
		strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("ntfy: build request: %w", err)
	}
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Title", title)
	if n.token != "" {
		req.Header.Set("Authorization", "Bearer "+n.token)
	}

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("ntfy: send: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ntfy: unexpected status %d", resp.StatusCode)
	}
	return nil
}
