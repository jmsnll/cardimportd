# cardimportd

A daemon for Synology DSM that watches for SD and CFExpress card insertions and automatically imports photos and videos into per-owner directories.

[![CI](https://github.com/jmsnll/cardimportd/actions/workflows/ci.yml/badge.svg)](https://github.com/jmsnll/cardimportd/actions/workflows/ci.yml)
[![Release](https://github.com/jmsnll/cardimportd/actions/workflows/release.yml/badge.svg)](https://github.com/jmsnll/cardimportd/actions/workflows/release.yml)
[![Latest Release](https://img.shields.io/github/v/release/jmsnll/cardimportd)](https://github.com/jmsnll/cardimportd/releases/latest)
[![Go](https://img.shields.io/badge/go-1.22-blue)](go.mod)
[![License](https://img.shields.io/github/license/jmsnll/cardimportd)](LICENSE)
[![Sponsor](https://img.shields.io/github/sponsors/jmsnll?label=sponsor&logo=githubsponsors)](https://github.com/sponsors/jmsnll)

---

## What it does

Insert a card into your USB reader. `cardimportd` detects the mount, identifies the card by its filesystem UUID, and copies every new photo and video into a dated folder tree under the card owner's library — verified with SHA-256, deduplicated by EXIF timestamp and camera model, and announced via Pushover.

Designed for a two-person household running Fujifilm and Sony cameras with a dual-slot UHS-II / CFExpress reader on a Synology NAS.

---

## Features

- **Automatic detection** — polls `/proc/mounts`; no udev required
- **Card identity by UUID** — stable across power cycles and reader changes; survives camera index rollover (`DSCF0001.RAF` on multiple cards)
- **EXIF-aware dating** — reads `DateTimeOriginal` from JPEG, RAF, ARW, TIFF; falls back to MP4/MOV container timestamp, then file mtime
- **SHA-256 copy verification** — streamed in a single pass via `io.TeeReader`; temp file is renamed atomically on success
- **Smart deduplication** — skips identical files; keeps the larger copy if sizes differ
- **Pushover notifications** — import started, completed, and new-card-pending events
- **Hot config reload** — update `config.yaml` and re-insert the card; no restart needed
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

## Configuration

`config.yaml` is loaded on startup and re-read on each card insertion event.

```yaml
# Mount paths that cardimportd watches for new USB volumes.
watch_paths:
  - /volumeUSB1/usbshare
  - /volumeUSB2/usbshare

# Root directory for all imports.
import_root: /volume1/photos

# Cards are keyed by filesystem UUID.
# Find a card's UUID with: blkid -s UUID -o value /dev/sdX
#
# status: active   — import runs automatically on insertion
# status: pending  — card seen but not yet configured; set owner and flip to active
cards:
  "1A2B-3C4D":
    owner: "James"
    status: active
  "5E6F-7A8B":
    owner: "Sophie"
    status: active

# File extensions to import.
file_extensions:
  - .jpg
  - .jpeg
  - .raf   # Fujifilm RAW
  - .arw   # Sony RAW
  - .mp4
  - .mov
  - .xmp

log_path: /var/log/cardimportd.log

# Optional: Pushover notifications (https://pushover.net)
# notifications:
#   pushover:
#     app_token: "your-app-token"
#     user_key:  "your-user-key"
```

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

Edit the file to set `owner` and change `status` to `active`, then re-insert the card. No restart required.

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

---

## Notifications

`cardimportd` sends Pushover notifications for:

| Event | When |
|---|---|
| `new_card_pending` | An unregistered card is inserted |
| `import_started` | Import begins for a known card |
| `import_completed` | Import finishes, with file counts and bytes copied |
| `import_failed` | A fatal error stops an import |

Set up a free [Pushover](https://pushover.net) account, create an application, and add the tokens to `config.yaml`.

---

## Building from source

Requires Go 1.22+.

```sh
git clone https://github.com/jmsnll/cardimportd.git
cd cardimportd

# Local binary
make build

# Cross-compile for Synology (amd64 / arm64)
make build-dsm-amd64
make build-dsm-arm64

# Synology .spk package
make spk-amd64
make spk-arm64
```

---

## Sponsor

If `cardimportd` saves you time, consider sponsoring:

**[github.com/sponsors/jmsnll](https://github.com/sponsors/jmsnll)**
