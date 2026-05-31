#!/bin/sh
systemctl stop cardimportd || true
systemctl disable cardimportd || true
