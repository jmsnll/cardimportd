// Package config loads and saves the cardimportd YAML configuration file.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"
)

// CardStatus describes the operational state of a registered card.
type CardStatus string

const (
	StatusActive  CardStatus = "active"
	StatusPending CardStatus = "pending"
)

var defaultFileExtensions = []string{
	".jpg", ".jpeg",
	".raf",
	".arw", ".lrf",
	".cr3", ".cr2",
	".nef", ".nrw",
	".dng",
	".orf",
	".rw2",
	".heic", ".heif",
	".mp4", ".mov", ".mxf",
	".wav", ".aif",
	".xmp",
}

// CardEntry holds the per-card configuration keyed by filesystem UUID.
type CardEntry struct {
	Owner               string     `yaml:"owner"                json:"owner"`
	Label               string     `yaml:"label,omitempty"      json:"label,omitempty"`
	Status              CardStatus `yaml:"status"               json:"status"`
	FirstSeen           *time.Time `yaml:"first_seen,omitempty" json:"first_seen,omitempty"`
	DestinationTemplate string     `yaml:"destination_template,omitempty" json:"destination_template,omitempty"`
}


// PushoverConfig holds credentials for the Pushover push notification service.
type PushoverConfig struct {
	AppToken string   `yaml:"app_token"        json:"app_token"`
	UserKey  string   `yaml:"user_key"         json:"user_key"`
	Events   []string `yaml:"events,omitempty" json:"events,omitempty"`
}

// NtfyConfig holds settings for an ntfy topic (ntfy.sh or self-hosted).
// URL must be the full topic URL, e.g. https://ntfy.sh/mycards or
// http://nas:8080/mycards. Token is optional (Bearer auth).
type NtfyConfig struct {
	URL    string   `yaml:"url"              json:"url"`
	Token  string   `yaml:"token,omitempty"  json:"token,omitempty"`
	Events []string `yaml:"events,omitempty" json:"events,omitempty"`
}

// WebhookConfig holds settings for a generic HTTP webhook.
// If Secret is non-empty the request carries an X-Cardimportd-Signature header
// (HMAC-SHA256 of the JSON body, hex-encoded, prefixed with "sha256=").
type WebhookConfig struct {
	URL    string   `yaml:"url"              json:"url"`
	Secret string   `yaml:"secret,omitempty" json:"secret,omitempty"`
	Events []string `yaml:"events,omitempty" json:"events,omitempty"`
}

// NotificationConfig groups optional push notification adapters.
type NotificationConfig struct {
	Pushover *PushoverConfig `yaml:"pushover,omitempty" json:"pushover,omitempty"`
	Ntfy     *NtfyConfig     `yaml:"ntfy,omitempty"     json:"ntfy,omitempty"`
	Webhook  *WebhookConfig  `yaml:"webhook,omitempty"  json:"webhook,omitempty"`
}

// WebUIConfig holds optional settings for the embedded HTTP management server.
// When Username and Password are both non-empty, HTTP basic auth is required on
// all routes (except /healthz). BindAddress defaults to "0.0.0.0" if empty.
type WebUIConfig struct {
	BindAddress string `yaml:"bind_address,omitempty" json:"bind_address,omitempty"`
	Username    string `yaml:"username,omitempty"     json:"username,omitempty"`
	Password    string `yaml:"password,omitempty"     json:"password,omitempty"`
}

// Config is the top-level configuration structure for cardimportd.
type Config struct {
	WatchPaths          []string             `yaml:"watch_paths"      json:"watch_paths"`
	ImportRoot          string               `yaml:"import_root"      json:"import_root"`
	MinFreeGB           float64              `yaml:"min_free_gb,omitempty" json:"min_free_gb,omitempty"`
	MirrorRoot          string               `yaml:"mirror_root,omitempty" json:"mirror_root,omitempty"`
	Cards               map[string]CardEntry `yaml:"cards"            json:"cards"`
	FileExtensions      []string             `yaml:"file_extensions"  json:"file_extensions"`
	LogPath             string               `yaml:"log_path"         json:"log_path"`
	WebUI               WebUIConfig          `yaml:"webui,omitempty"  json:"webui,omitempty"`
	Notifications       NotificationConfig   `yaml:"notifications,omitempty" json:"notifications,omitempty"`
	PostImportHook      string               `yaml:"post_import_hook,omitempty" json:"post_import_hook,omitempty"`
	WriteManifest       bool                 `yaml:"write_manifest,omitempty" json:"write_manifest,omitempty"`
	DestinationTemplate string               `yaml:"destination_template,omitempty" json:"destination_template,omitempty"`
}

// Default returns a minimal working configuration seeded with Synology-typical paths.
func Default() *Config {
	return &Config{
		WatchPaths:     []string{"/volumeUSB1/usbshare", "/volumeUSB2/usbshare"},
		ImportRoot:     "/volume1/photos",
		Cards:          make(map[string]CardEntry),
		FileExtensions: append([]string(nil), defaultFileExtensions...),
	}
}

// Load reads and parses the YAML config at path.
// Unknown YAML keys are rejected. FileExtensions defaults if empty.
func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("config: open %q: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	var cfg Config
	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("config: decode %q: %w", path, err)
	}

	if len(cfg.FileExtensions) == 0 {
		cfg.FileExtensions = append([]string(nil), defaultFileExtensions...)
		slog.Info("config: no file_extensions set, using defaults", "extensions", cfg.FileExtensions)
	}
	if cfg.Cards == nil {
		cfg.Cards = make(map[string]CardEntry)
	}
	if err := validateDestTemplates(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func validateDestTemplates(cfg *Config) error {
	if cfg.DestinationTemplate != "" {
		if _, err := template.New("g").Parse(cfg.DestinationTemplate); err != nil {
			return fmt.Errorf("config: invalid destination_template: %w", err)
		}
	}
	for uuid, card := range cfg.Cards {
		if card.DestinationTemplate != "" {
			if _, err := template.New(uuid).Parse(card.DestinationTemplate); err != nil {
				return fmt.Errorf("config: invalid destination_template for %q: %w", uuid, err)
			}
		}
	}
	return nil
}

// Save atomically writes cfg to path via a .tmp sibling + os.Rename.
func (c *Config) Save(path string) error {
	tmp := path + ".tmp"
	dir := filepath.Dir(path)

	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		tf, tfErr := os.CreateTemp(dir, ".config-*.tmp")
		if tfErr != nil {
			return fmt.Errorf("config: create temp file in %q: %w", dir, tfErr)
		}
		tmp = tf.Name()
		f = tf
	}

	enc := yaml.NewEncoder(f)
	enc.SetIndent(2)
	encErr := enc.Encode(c)
	closeEncErr := enc.Close()
	closeFileErr := f.Close()

	for _, e := range []error{encErr, closeEncErr, closeFileErr} {
		if e != nil {
			_ = os.Remove(tmp)
			return fmt.Errorf("config: write temp file %q: %w", tmp, e)
		}
	}

	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("config: rename %q to %q: %w", tmp, path, err)
	}

	slog.Debug("config: saved", "path", path)
	return nil
}

// RegisterPending adds a StatusPending entry for uuid if not already present.
// Returns true if a new entry was inserted (caller should then Save).
func (c *Config) RegisterPending(uuid string) bool {
	if _, exists := c.Cards[uuid]; exists {
		return false
	}
	if c.Cards == nil {
		c.Cards = make(map[string]CardEntry)
	}
	now := time.Now().UTC()
	c.Cards[uuid] = CardEntry{Status: StatusPending, FirstSeen: &now}
	slog.Info("config: registered new pending card", "uuid", uuid)
	return true
}

// LookupCard returns the CardEntry for uuid and whether it was found.
func (c *Config) LookupCard(uuid string) (CardEntry, bool) {
	entry, ok := c.Cards[uuid]
	return entry, ok
}
