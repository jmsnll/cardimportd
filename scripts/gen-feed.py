#!/usr/bin/env python3
"""Generate packages.json for the Synology Package Center feed."""
import json
import sys


BASE = sys.argv[1]
SPK_VER = sys.argv[2]

COMMON = {
    "package": "cardimportd",
    "version": SPK_VER,
    "dname": "Card Importer",
    "desc": "Watches for SD/CFExpress card insertions and imports photos/videos.",
    "distributor": "jmsnll",
    "qinst": True,
    "qupgrade": True,
    "qstart": True,
}

# One entry per Linux arch. DSM matches on uname -m, not CPU family name.
packages = [
    {
        **COMMON,
        "arch": "x86_64",
        "link": f"{BASE}/cardimportd-{SPK_VER}-x86_64.spk",
    },
    {
        **COMMON,
        "arch": "armv8",
        "link": f"{BASE}/cardimportd-{SPK_VER}-armv8.spk",
    },
]

print(json.dumps({"packages": packages}, indent=2))
