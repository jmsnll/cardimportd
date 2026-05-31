#!/bin/sh
set -e

if [ ! -f /etc/cardimportd/config.yaml ]; then
    mkdir -p /etc/cardimportd
    cp /etc/cardimportd/config.example.yaml /etc/cardimportd/config.yaml
fi

systemctl daemon-reload
systemctl enable cardimportd || true
