#!/bin/bash
# Gather every available identifier for an SD/CF card device.
# Usage: identify-card.sh [device]   e.g. identify-card.sh /dev/usb1p1

set -euo pipefail

DEVICE="${1:-/dev/usb1p1}"
DEVNAME="$(basename "$DEVICE")"

# Derive parent disk name from sysfs symlink (most reliable method)
REAL_PART="$(readlink -f /sys/class/block/"$DEVNAME" 2>/dev/null || true)"
if [ -n "$REAL_PART" ]; then
    DISKNAME="$(basename "$(dirname "$REAL_PART")")"
else
    # Fallback: strip trailing pN suffix (usb1p1 -> usb1, sda1 -> sda)
    DISKNAME="$(echo "$DEVNAME" | sed 's/p\?[0-9]\+$//')"
fi
DISK="/dev/$DISKNAME"

hr() { printf '\n%s\n' "$(printf '=%.0s' {1..60})"; echo "  $*"; printf '%s\n' "$(printf '=%.0s' {1..60})"; }
field() { printf '  %-30s %s\n' "$1:" "$2"; }
run() { "$@" 2>&1 || echo "  (command failed or not available)"; }

echo ""
echo "  SD card identification data — $(date)"
field "device (partition)" "$DEVICE"
field "device (disk)"      "$DISK"

hr "blkid — all fields"
run blkid "$DEVICE"
echo ""
echo "  --- individual fields ---"
for f in UUID PARTUUID LABEL TYPE PTTYPE PTUUID; do
    val="$(blkid -s "$f" -o value "$DEVICE" 2>/dev/null || true)"
    field "$f" "${val:-(empty)}"
done

hr "fdisk — partition geometry"
run fdisk -l "$DISK"

hr "sysfs — partition attributes (/sys/class/block/$DEVNAME)"
for f in size start ro removable; do
    val="$(cat /sys/class/block/"$DEVNAME"/"$f" 2>/dev/null || true)"
    field "$f" "${val:-(not found)}"
done

hr "sysfs — disk attributes (/sys/class/block/$DISKNAME)"
for f in size ro removable; do
    val="$(cat /sys/class/block/"$DISKNAME"/"$f" 2>/dev/null || true)"
    field "$f" "${val:-(not found)}"
done

hr "sysfs — SCSI device attributes"
SCSI_DEV="$(readlink -f /sys/class/block/"$DISKNAME"/device 2>/dev/null || true)"
if [ -n "$SCSI_DEV" ]; then
    field "scsi path" "$SCSI_DEV"
    for f in vendor model rev type; do
        val="$(cat "$SCSI_DEV/$f" 2>/dev/null | tr -d '[:space:]' || true)"
        field "$f" "${val:-(not found)}"
    done
else
    echo "  (no SCSI device link found)"
fi

hr "sysfs — USB device tree (walking up from partition)"
if [ -n "$REAL_PART" ]; then
    echo "  resolved sysfs path: $REAL_PART"
    dir="$REAL_PART"
    for i in $(seq 1 10); do
        dir="$(dirname "$dir")"
        [ "$dir" = "/" ] && break
        found=0
        for f in idVendor idProduct manufacturer product serial bcdDevice speed; do
            val="$(cat "$dir/$f" 2>/dev/null | tr -d '\n' || true)"
            if [ -n "$val" ]; then
                [ "$found" -eq 0 ] && echo "" && echo "  -- $dir --"
                field "$f" "$val"
                found=1
            fi
        done
    done
else
    echo "  (could not resolve sysfs symlink)"
fi

hr "sysfs — MMC CID (native slot only, not USB reader)"
found_cid=0
for cid_path in /sys/class/mmc_host/mmc*/mmc*:*/cid; do
    [ -f "$cid_path" ] || continue
    cid="$(cat "$cid_path")"
    echo "  $cid_path: $cid"
    # Decode CID fields (128-bit register)
    # Bytes: [0]=MID [1-2]=OID [3-7]=PNM [8]=PRV [9-12]=PSN [13-14]=MDT [15]=CRC
    mid="${cid:0:2}"
    oid_hex="${cid:2:4}"
    oid_ascii="$(printf '%b' "\\x${oid_hex:0:2}\\x${oid_hex:2:2}" 2>/dev/null || true)"
    pnm_hex="${cid:6:10}"
    pnm_ascii="$(printf '%b' "\\x${pnm_hex:0:2}\\x${pnm_hex:2:2}\\x${pnm_hex:4:2}\\x${pnm_hex:6:2}\\x${pnm_hex:8:2}" 2>/dev/null || true)"
    psn="${cid:18:8}"
    field "  manufacturer ID (MID)" "0x$mid"
    field "  OEM/App ID (OID)"      "$oid_ascii (0x$oid_hex)"
    field "  product name (PNM)"    "$pnm_ascii"
    field "  product serial (PSN)"  "0x$psn"
    found_cid=1
done
[ "$found_cid" -eq 0 ] && echo "  (no MMC host found — card is in a USB reader; CID not exposed)"

hr "raw exFAT boot sector (partition: $DEVICE)"
echo "  Reading first 512 bytes..."
BOOT="$(dd if="$DEVICE" bs=512 count=1 2>/dev/null | xxd 2>/dev/null || true)"
if [ -n "$BOOT" ]; then
    echo "$BOOT"
    echo ""
    # exFAT-specific fields from the Volume Boot Record
    # OEM name: bytes 3-10
    OEM="$(dd if="$DEVICE" bs=1 skip=3 count=8 2>/dev/null | cat 2>/dev/null || true)"
    field "OEM name (bytes 3-10)" "'$OEM'"
    # Volume Serial Number: bytes 100-103 (little-endian uint32)
    VSN_RAW="$(dd if="$DEVICE" bs=1 skip=100 count=4 2>/dev/null | od -A n -t x1 | tr -d ' \n' || true)"
    field "Volume Serial Number (bytes 100-103 LE hex)" "${VSN_RAW:-(read failed)}"
    if [ "${#VSN_RAW}" -eq 8 ]; then
        VSN_BE="${VSN_RAW:6:2}${VSN_RAW:4:2}${VSN_RAW:2:2}${VSN_RAW:0:2}"
        field "Volume Serial Number (as big-endian)" "0x$VSN_BE"
    fi
    # File System Revision: bytes 104-105
    REV="$(dd if="$DEVICE" bs=1 skip=104 count=2 2>/dev/null | od -A n -t x1 | tr -d ' \n' || true)"
    field "FS revision (bytes 104-105)" "${REV:-(read failed)}"
else
    echo "  (could not read boot sector — check permissions)"
fi

hr "lsusb"
run lsusb

hr "lsusb -v (filtered to USB storage devices)"
lsusb -v 2>/dev/null | grep -A 40 'Mass Storage\|iManufacturer\|iProduct\|iSerial\|bcdUSB\|idVendor\|idProduct' | head -80 || echo "  (lsusb -v not available or no data)"

hr "udevadm info"
run udevadm info --query=all --name="$DEVICE"

hr "sg_inq — SCSI inquiry (unit serial number, if available)"
run sg_inq "$DISK"

echo ""
echo "  Done."
