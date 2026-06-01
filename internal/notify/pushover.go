package notify

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const pushoverAPIURL = "https://api.pushover.net/1/messages.json"

type pushoverNotifier struct {
	appToken string
	userKey  string
	client   *http.Client
}

// NewPushoverNotifier returns a Notifier that delivers events via Pushover.
// appToken is the application API token; userKey is the recipient user/group key.
func NewPushoverNotifier(appToken, userKey string) Notifier {
	return &pushoverNotifier{
		appToken: appToken,
		userKey:  userKey,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *pushoverNotifier) Notify(ctx context.Context, event Event) error {
	title, message := formatEvent(event)
	priority := priorityFor(event.Kind)

	body := url.Values{
		"token":    {p.appToken},
		"user":     {p.userKey},
		"title":    {title},
		"message":  {message},
		"priority": {fmt.Sprintf("%d", priority)},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, pushoverAPIURL,
		strings.NewReader(body.Encode()))
	if err != nil {
		return fmt.Errorf("pushover: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("pushover: send: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("pushover: unexpected status %d", resp.StatusCode)
	}
	return nil
}

func ownerDisplay(owner, label string) string {
	if label != "" {
		return owner + " (" + label + ")"
	}
	return owner
}

func formatEvent(e Event) (title, message string) {
	display := ownerDisplay(e.Owner, e.CardLabel)
	switch e.Kind {
	case KindNewCardPending:
		return "cardimportd: new card",
			fmt.Sprintf("Unknown card inserted (UUID: %s)\nEdit config to activate.", e.CardUUID)
	case KindImportStarted:
		return "cardimportd: import started",
			fmt.Sprintf("Importing %s's card…", display)
	case KindImportCompleted:
		if s := e.Stats; s != nil {
			return fmt.Sprintf("cardimportd: %s imported", display),
				fmt.Sprintf("%d files imported (%s), %d skipped, %d failed\nDuration: %s",
					s.Imported, humanizeBytes(s.BytesCopied), s.Skipped, s.Failed, s.Duration.Round(time.Second))
		}
		return "cardimportd: import complete", "Import finished."
	case KindImportFailed:
		return "cardimportd: import FAILED",
			fmt.Sprintf("Import failed for %s\n%s", display, e.Detail)
	default:
		return "cardimportd", string(e.Kind)
	}
}

func humanizeBytes(b int64) string {
	switch {
	case b >= 1_000_000_000:
		return fmt.Sprintf("%.1f GB", float64(b)/1_000_000_000)
	case b >= 1_000_000:
		return fmt.Sprintf("%.1f MB", float64(b)/1_000_000)
	case b >= 1_000:
		return fmt.Sprintf("%.1f KB", float64(b)/1_000)
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// priorityFor maps event kinds to Pushover priority levels.
//   -1 = low (quiet), 0 = normal, 1 = high
func priorityFor(k EventKind) int {
	switch k {
	case KindImportFailed:
		return 1
	case KindImportStarted:
		return -1
	default:
		return 0
	}
}
