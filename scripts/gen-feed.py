#!/usr/bin/env python3
"""Generate packages.json for the Synology Package Center feed."""
import json
import sys


def main():
    base = sys.argv[1]
    spk_ver = sys.argv[2]
    archs = sys.argv[3:]

    packages = [
        {
            "package": "cardimportd",
            "version": spk_ver,
            "dname": "Card Importer",
            "desc": "Watches for SD/CFExpress card insertions and imports photos/videos.",
            "arch": [arch],
            "distributor": "jmsnll",
            "link": f"{base}/cardimportd-{spk_ver}-{arch}.spk",
            "qinst": True,
            "qupgrade": True,
            "qstart": True,
        }
        for arch in archs
    ]

    print(json.dumps({"packages": packages}, indent=2))


if __name__ == "__main__":
    main()
