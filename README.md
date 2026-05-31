# cardimportd

A daemon for Synology DSM that watches for SD and CFExpress card insertions and automatically imports photos and videos into per-owner directories.

[![CI](https://github.com/jmsnll/cardimportd/actions/workflows/ci.yml/badge.svg)](https://github.com/jmsnll/cardimportd/actions/workflows/ci.yml)
[![Release](https://github.com/jmsnll/cardimportd/actions/workflows/release.yml/badge.svg)](https://github.com/jmsnll/cardimportd/actions/workflows/release.yml)
[![Latest Release](https://img.shields.io/github/v/release/jmsnll/cardimportd)](https://github.com/jmsnll/cardimportd/releases/latest)
[![Go](https://img.shields.io/badge/go-1.26-blue)](go.mod)
[![License](https://img.shields.io/github/license/jmsnll/cardimportd)](LICENSE)
[![Sponsor](https://img.shields.io/github/sponsors/jmsnll?label=sponsor&logo=githubsponsors)](https://github.com/sponsors/jmsnll)

---

## What it does

Insert a card into your USB reader. `cardimportd` detects the mount, identifies the card by its filesystem UUID, and copies every new photo and video into a dated folder tree under the card owner's library — verified with SHA-256, deduplicated by EXIF timestamp and camera model, and announced via Pushover, ntfy, or webhook.

Designed for a two-person household running Fujifilm and Sony cameras with a dual-slot UHS-II / CFExpress reader on a Synology NAS.

---

## Features

- **Automatic detection** — polls `/proc/mounts`; no udev required
- **Card identity by UUID** — stable across power cycles and reader changes; survives camera index rollover (`DSCF0001.RAF` on multiple cards)
- **EXIF-aware dating** — reads `DateTimeOriginal` from JPEG, RAF, ARW, TIFF; falls back to MP4/MOV container timestamp, then file mtime
- **SHA-256 copy verification** — streamed in a single pass via `io.TeeReader`; temp file is renamed atomically on success
- **Smart deduplication** — skips identical files; keeps the larger copy if sizes differ
- **Mirror support** — optionally copies every imported file to a second destination for 3-2-1 backups
- **Free-space preflight** — aborts before copying if the destination volume has insufficient space
- **Custom destination templates** — configurable `text/template` paths per-card or globally
- **Post-import hook** — run any shell command after a successful import
- **Checksum manifest** — optionally writes a SHA-256 manifest alongside imported files
- **Push notifications** — Pushover, ntfy, and generic webhook; per-adapter event filtering
- **Import history** — append-only JSONL log; queryable via the web UI
- **Web management UI** — browser-based config editor, card manager, and notification tester
- **Hot config reload** — `SIGHUP` reloads `config.yaml`; no restart needed
- **No database** — plain YAML config; auditable and editable over SSH

---

## Installation

### Synology Package Center (recommended)

1. Open **Package Center** → **Settings** → **Package Sources** → **Add**
2. Enter the source URL:
   ```
   https://jmsnll.github.io/cardimportd/
   ```
3. Search for **Card Importer** and install.
4. Edit `/var/packages/cardimportd/etc/config.yaml` to register your cards (see [Configuration](#configuration)).
5. Start the package.

### Manual binary

Download the binary for your NAS architecture from the [latest release](https://github.com/jmsnll/cardimportd/releases/latest):

| Architecture | Most Synology models |
|---|---|
| `linux-amd64` | DS923+, DS1522+, and most Intel/AMD NAS |
| `linux-arm64` | DS220j, DS418, and other ARM-based NAS |

```sh
# Example for amd64
curl -L https://github.com/jmsnll/cardimportd/releases/latest/download/cardimportd-linux-amd64.tar.gz \
  | tar xz -C /usr/local/bin

mkdir -p /etc/cardimportd
curl -L https://raw.githubusercontent.com/jmsnll/cardimportd/main/config.example.yaml \
  -o /etc/cardimportd/config.yaml

cardimportd --config /etc/cardimportd/config.yaml
```

To run as a service, add a triggered task in **Control Panel → Task Scheduler → Triggered Task → Boot-up**.

---

## CLI flags

<!-- AUTO-GENERATED from cmd/cardimportd/main.go -->
| Flag | Default | Description |
|---|---|---|
| `--config` | `/usr/local/etc/cardimportd/config.yaml` | Path to `config.yaml` |
| `--webui-port` | `8085` | Web UI port (set to `0` to disable) |
| `--proc-mounts` | `/proc/mounts` | Mounts file to poll |
| `--poll-interval` | `2s` | Watcher poll interval |
<!-- END AUTO-GENERATED -->

---

## Configuration

<!-- AUTO-GENERATED from internal/config/config.go -->
`config.yaml` is loaded on startup and re-read on `SIGHUP`. All fields are optional unless noted.

```yaml
# Mount paths that cardimportd watches for new USB volumes. (required)
watch_paths:
  - /volumeUSB1/usbshare
  - /volumeUSB2/usbshare

# Root directory for all imports. (required)
import_root: /volume1/photos

# Optional: mirror every imported file to a second root for 3-2-1 backup.
# mirror_root: /volumeUSB3/usbshare/mirror

# Abort import if the destination volume has less free space than this.
# 0 (default) disables the check.
# min_free_gb: 10

# Cards are keyed by filesystem UUID.
# Find a card's UUID with: blkid -s UUID -o value /dev/sdX
#
# status: active   — import runs automatically on insertion
# status: pending  — card seen but not yet configured; set owner and flip to active
cards:
  "1A2B-3C4D":
    owner: "James"
    label: "X-T5 Main"        # optional human-readable name shown in notifications
    status: active
    destination_template: ""   # override global template for this card (see below)
  "5E6F-7A8B":
    owner: "Sophie"
    label: "A7C II"
    status: active

# File extensions to import. Defaults to the list below if omitted.
file_extensions:
  - .jpg
  - .jpeg
  - .raf             # Fujifilm
  - .arw             # Sony
  - .lrf             # Sony sidecar
  - .cr3             # Canon
  - .cr2             # Canon (legacy)
  - .nef             # Nikon
  - .nrw             # Nikon (compact)
  - .dng             # Adobe DNG
  - .orf             # Olympus
  - .rw2             # Panasonic
  - .heic
  - .heif
  - .mp4
  - .mov
  - .mxf
  - .wav
  - .aif
  - .xmp

# Custom destination path template (text/template syntax).
# Available variables: .Owner  .Year  .Month  .Day  .CameraModel  .CardUUID
# Default: "{{ .Owner }}'s Library/{{ .Year }}/{{ .Month }}/{{ .Day }}"
# destination_template: "{{ .Owner }}/{{ .Year }}/{{ .Month }}/{{ .Day }}/{{ .CameraModel }}"

# Run a shell command after every successful import.
# Environment variables: CARDIMPORTD_OWNER, CARDIMPORTD_MOUNT,
#   CARDIMPORTD_FILES_IMPORTED, CARDIMPORTD_FILES_SKIPPED,
#   CARDIMPORTD_FILES_FAILED, CARDIMPORTD_BYTES_COPIED
# post_import_hook: "rsync -a /volume1/photos/ backup@nas2:/volume1/photos/"

# Write a SHA-256 manifest file alongside imported files.
# write_manifest: true

# Notifications (all adapters are optional; combine freely).
# Per-adapter event filtering: list any subset of the four event kinds.
# Omit `events` to receive all events.
notifications:
  pushover:
    app_token: "your-app-token"
    user_key:  "your-user-key"
    # events: [import_completed, import_failed]

  ntfy:
    url: "https://ntfy.sh/your-topic"   # or self-hosted: http://nas:8080/topic
    token: ""                            # optional Bearer token
    # events: [import_completed, import_failed, new_card_pending]

  webhook:
    url: "https://example.com/hook"
    secret: ""   # if set, signs each request with X-Cardimportd-Signature (HMAC-SHA256)
    # events: [import_completed]
```
<!-- END AUTO-GENERATED -->

---

## Card registration

The first time an unknown card is inserted, `cardimportd` writes a `pending` entry to `config.yaml`:

```yaml
cards:
  "XXXX-YYYY":
    owner: ""
    status: pending
    first_seen: "2026-05-30T10:00:00Z"
```

Edit the file to set `owner` and change `status` to `active`, then re-insert the card. No restart required. The web UI can do this for you via the **Cards** page.

---

## Destination layout

Files are copied to:

```
{import_root}/{owner}'s Library/{YYYY}/{MM}/{DD}/{original-filename}
```

For example:

```
/volume1/photos/James's Library/2026/05/28/DSCF1234.RAF
/volume1/photos/James's Library/2026/05/28/DSCF1234.JPG
/volume1/photos/Sophie's Library/2026/05/28/DSC00123.ARW
```

The date comes from **EXIF `DateTimeOriginal`**. Fallback chain:

1. EXIF `DateTimeOriginal` (JPEG, RAF, ARW, TIFF)
2. MP4/MOV `mvhd` box creation time
3. File mtime (last resort; logged as a warning)

### Custom templates

Override the destination layout globally or per card using `destination_template`:

```yaml
# Group by camera model then date
destination_template: "{{ .Owner }}/{{ .CameraModel }}/{{ .Year }}-{{ .Month }}-{{ .Day }}"
```

Available variables: `.Owner`, `.Year`, `.Month`, `.Day`, `.CameraModel`, `.CardUUID`.

---

## Notifications

Events are sent to every configured adapter. Use the `events` filter list to limit which events each adapter receives.

| Event | When |
|---|---|
| `new_card_pending` | An unregistered card is inserted |
| `import_started` | Import begins for a known card |
| `import_completed` | Import finishes, with file counts and bytes copied |
| `import_failed` | A fatal error stops an import |

### Pushover

Sign up at [pushover.net](https://pushover.net), create an application, and copy the tokens into `config.yaml`.

### ntfy

Works with [ntfy.sh](https://ntfy.sh) or a self-hosted instance. Set `token` for topics that require authentication.

### Webhook

`cardimportd` POSTs a JSON body to the configured URL. If `secret` is set, each request includes an `X-Cardimportd-Signature: sha256=<hex>` header (HMAC-SHA256 of the body).

---

## Web UI

`cardimportd` serves a management UI on `--webui-port` (default **8085**). Open `http://your-nas:8085` in a browser.

| Page / Endpoint | Description |
|---|---|
| **Cards** | View registered cards, activate pending cards, delete cards |
| **Settings** | Edit the full config and save to disk |
| **Notifications** | Send a test notification to any configured adapter |
| `GET /api/history` | Recent import runs as JSON (`?limit=N`, default 50) |
| `GET /api/events` | Live import progress stream (Server-Sent Events) |

Set `--webui-port 0` to disable the UI entirely.

---

## Building from source

Requires Go 1.26+ and Node.js 20+ (for the web UI).

```sh
git clone https://github.com/jmsnll/cardimportd.git
cd cardimportd
```

<!-- AUTO-GENERATED from Makefile -->
| Command | Description |
|---|---|
| `make build` | Build the binary (builds UI first) |
| `make test` | Run the full test suite with race detection |
| `make bench` | Run benchmarks and save results to `benchmarks/<version>.txt` |
| `make vet` | Run `go vet` |
| `make build-dsm-amd64` | Cross-compile for Synology (x86\_64) |
| `make build-dsm-arm64` | Cross-compile for Synology (ARM64) |
| `make spk-amd64` | Build a `.spk` Synology package (amd64) |
| `make spk-arm64` | Build a `.spk` Synology package (arm64) |
| `make clean` | Remove binaries and `ui/node_modules` |
| `make ui` | Build the web UI only |
<!-- END AUTO-GENERATED -->

---

## Sponsor

If `cardimportd` saves you time, consider sponsoring:

**[github.com/sponsors/jmsnll](https://github.com/sponsors/jmsnll)**
