# cardimportd — Design Document

## Overview

`cardimportd` is a Go daemon that runs on a Synology DSM NAS. It watches for USB card readers (SD / CFExpress Type A) being inserted, identifies the card by its filesystem UUID, and imports photos and videos to per-owner destination directories. It is intentionally small: no database, no web UI, no dependencies beyond the Go standard library and two external packages.

---

## System Context

| Attribute | Value |
|---|---|
| Platform | Synology DSM (Linux, no native udev in userspace) |
| Hardware | Dual-slot USB reader — UHS-II SD + CFExpress Type A |
| Cameras | Fujifilm (RAF + JPEG) and Sony (ARW + JPEG) |
| Scale | ~20 cards, thousands of files per shoot, videos up to 100 GB |
| Operators | 2 people |

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      cardimportd                        │
│                                                         │
│  ┌──────────┐   mount    ┌──────────────────────────┐   │
│  │ watcher  │──events───▶│        daemon loop        │   │
│  └──────────┘            │  (cmd/cardimportd/main)   │   │
│                          └────────────┬─────────────┘   │
│                                       │                  │
│              ┌──────────┬─────────────┼──────────┐      │
│              ▼          ▼             ▼          ▼      │
│          ┌────────┐ ┌────────┐ ┌──────────┐ ┌────────┐  │
│          │ config │ │  meta  │ │ importer │ │ notify │  │
│          └────────┘ └────────┘ └──────────┘ └────────┘  │
└─────────────────────────────────────────────────────────┘
```

### Package responsibilities

| Package | Responsibility |
|---|---|
| `internal/watcher` | Poll `/proc/mounts` every 2 s; emit `MountEvent` on new USB mounts matching `/volumeUSBX/usbshareY` |
| `internal/config` | Load/save `config.yaml`; hold card registry (UUID → owner/status); atomic YAML writes |
| `internal/meta` | Extract `DateTimeOriginal` + `CameraModel` from image EXIF (Fuji RAF, Sony ARW, JPEG) and from MP4/MOV container (`mvhd` box) |
| `internal/importer` | Walk source, dedup, stream-copy with in-flight SHA-256, rename on verify |
| `internal/notify` | `Notifier` interface + structured-log implementation; future push adapters drop in here |

---

## Device Detection

Synology DSM automatically mounts USB block devices. Mount points follow the pattern:

```
/volumeUSBN/usbshareM   (e.g. /volumeUSB1/usbshare1)
```

The daemon reads `/proc/mounts` every 2 seconds and diffs against the previous snapshot. Any newly seen entry whose mount point starts with `/volumeUSB` triggers a card-identification attempt.

**Card identity** is the filesystem UUID obtained via:

```sh
blkid -s UUID -o value /dev/sdX
```

The UUID persists across power cycles and reader changes. A reformat changes the UUID, requiring re-registration — this is acceptable and expected behaviour.

---

## Card Registration Flow

```
Card inserted
     │
     ▼
UUID in config?
     ├─ yes, status=active  ──▶  start import
     └─ no / status=pending ──▶  write pending entry to config
                                 log warning with UUID
                                 wait for operator to set owner + status=active
                                 (re-insert or restart daemon to retry)
```

Config is hot-reloaded on each mount event so you can update it while the daemon is running and re-insert the card without restarting.

---

## Configuration

Stored at `/usr/local/etc/cardimportd/config.yaml` (overridable via `--config` flag).

```yaml
watch_paths:
  - /volumeUSB1/usbshare
  - /volumeUSB2/usbshare

import_root: /volume1/photos

cards:
  "1A2B-3C4D":
    owner: "James"
    status: active
  "5E6F-7A8B":
    owner: "Sophie"
    status: active
  "XXXX-YYYY":
    owner: ""
    status: pending
    first_seen: "2026-05-30T10:00:00Z"

file_extensions:
  - .jpg
  - .jpeg
  - .raf
  - .arw
  - .mp4
  - .mov
  - .xmp

log_path: /var/log/cardimportd.log
```

---

## Destination Layout

```
/volume1/photos/{owner}'s Library/YYYY/MM/DD/{original-filename}
```

Date is derived from **EXIF `DateTimeOriginal`**. Fallback chain:

1. EXIF `DateTimeOriginal` (images)
2. MP4/MOV `mvhd` box `creation_time` (videos)
3. File `mtime` (last resort; logged as a warning)

---

## Duplicate Detection

**Primary dedup key**: `DateTimeOriginal` + `CameraModel` + lowercased filename stem

This handles the common case of camera file-index rollover (e.g. `DSCF0001.RAF` appearing on multiple cards).

**Size tiebreaker**: if a file with the same key already exists at the destination, compare sizes. Keep the **larger** file (higher-bitrate version wins).

**Result**: skip identical, replace with larger, always log the decision.

---

## Copy & Verification

```
source file
    │
    ▼
sha256.New() ──── io.TeeReader ────▶ dst/{filename}.tmp
    │                                        │
    └── source digest                        ▼
                                      sha256.New() on tmp
                                             │
                                      digests match?
                                        ├─ yes ──▶ os.Rename tmp → dst/{filename}
                                        └─ no  ──▶ delete tmp, mark failed
```

- Streams via `io.TeeReader` — no file ever fully loaded into memory.
- Temp file is a sibling of the destination (same volume, rename is atomic).
- On failure: temp file is deleted, error is logged, import continues with next file.

---

## Notification Interface

```go
type Notifier interface {
    Notify(ctx context.Context, event Event) error
}

type EventKind string
const (
    KindNewCardPending    EventKind = "new_card_pending"
    KindImportStarted     EventKind = "import_started"
    KindImportCompleted   EventKind = "import_completed"
    KindImportFailed      EventKind = "import_failed"
)

type Event struct {
    Kind      EventKind
    CardUUID  string
    Owner     string
    MountPath string
    Time      time.Time
    Detail    string
    Stats     *ImportStats  // nil for non-completion events
}

type ImportStats struct {
    Total       int
    Imported    int
    Skipped     int
    Failed      int
    BytesCopied int64
    Duration    time.Duration
}
```

**Current implementation**: `LogNotifier` writes structured JSON to the configured log file.

**Planned implementations** (future PRs — add adapters without changing importer):

| Adapter | Notes |
|---|---|
| Pushover | Mobile push; ideal for personal use |
| ntfy.sh | Self-hosted push; works on Synology via Docker |
| Slack webhook | If already using Slack |
| Email (SMTP) | Lowest common denominator fallback |

To add a new notifier: implement `Notifier`, add it to the `MultiNotifier` list in config, done.

---

## Operational Concerns

### Running as a service on DSM

Synology supports user-defined startup scripts via **Task Scheduler** (Control Panel → Task Scheduler → Triggered Task → Boot-up). Drop the binary at `/usr/local/bin/cardimportd` and add a triggered task that runs it with `--config /usr/local/etc/cardimportd/config.yaml`.

A future improvement is a proper `/etc/init/` Upstart job or a systemd unit (DSM 7 ships systemd).

### Signal handling

| Signal | Behaviour |
|---|---|
| `SIGTERM` / `SIGINT` | Graceful shutdown; wait for in-progress import to finish |
| `SIGHUP` | Reload config from disk |

### Logging

Structured JSON via `log/slog` to stdout + a file sink. Fields: `time`, `level`, `msg`, plus context fields (`card_uuid`, `owner`, `src`, `dst`, `bytes`) where relevant.

---

## Future Work

- [ ] Post-import card eject (`udisksctl power-off`)
- [ ] Post-import card wipe with verification prompt
- [ ] XMP sidecar co-location (keep `.xmp` next to its RAW)
- [ ] Push notification adapters (Pushover, ntfy, Slack, SMTP)
- [ ] DSM systemd unit file
- [ ] Progress reporting via temp file for external monitoring
- [ ] `cardimportctl` CLI for listing pending cards and triggering manual imports

---

## Decision Log

| # | Decision | Rationale |
|---|---|---|
| 1 | Poll `/proc/mounts` rather than inotify on `/dev` | Synology's udev is not reliably accessible from userspace; polling is 2 s max lag, zero dependency |
| 2 | Filesystem UUID as card identity | Stable across power cycles and reader changes; readable without mounting the card |
| 3 | `dsoprea/go-exif` for EXIF | Best support for Fuji RAF and Sony ARW among pure-Go libraries; handles TIFF-based RAW natively |
| 4 | Custom RAF JPEG-section reader | RAF embeds a JPEG at a fixed offset; we seek to it rather than parsing the full RAF container |
| 5 | In-flight SHA-256 via `io.TeeReader` | Single read pass; no second I/O pass needed; memory-constant for 100 GB+ files |
| 6 | Atomic rename from `.tmp` | Prevents partial files appearing in destination during copy or on crash |
| 7 | Dedup by EXIF datetime+model+stem | Handles index rollover (RAW0001.RAF on multiple cards) while allowing same-filename re-imports if the file changed |
| 8 | `log/slog` over zap/zerolog | Stdlib in Go 1.21+; no transitive deps; adequate for this scale |
| 9 | YAML config, no database | 20 cards, 2 operators — a text file is auditable, diffable, and editable over SSH |
| 10 | Single-card import at a time | Avoids I/O contention on the NAS spindles; simplifies error reporting |
