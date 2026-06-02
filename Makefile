.PHONY: help build build-photomigrate test bench lint vet build-dsm-amd64 build-dsm-arm64 deploy deploy-restart clean ui

NAS_HOST  ?= root@vault.jamesneill.co.uk
NAS_BIN    = /var/packages/cardimportd/target/cardimportd

BINARY    := cardimportd
CMD       := ./cmd/$(BINARY)
BUILD_DIR := build
VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "  build             Build the binary (includes UI)"
	@echo "  build-photomigrate  Build the photomigrate migration tool"
	@echo "  ui                Install JS deps and build the web UI"
	@echo "  test              Run tests with race detector"
	@echo "  bench             Run benchmarks and save results to benchmarks/"
	@echo "  vet               Run go vet"
	@echo "  lint              Run vet (alias)"
	@echo "  build-dsm-amd64   Cross-compile for Synology x86_64 (DS923+, DS1522+, etc.)"
	@echo "  build-dsm-arm64   Cross-compile for Synology arm64 (DS220j, DS418, etc.)"
	@echo "  spk-amd64         Build a .spk package for x86_64"
	@echo "  spk-arm64         Build a .spk package for armv8"
	@echo "  deploy            Build for amd64 and SCP binary to NAS (NAS_HOST=$(NAS_HOST))"
	@echo "  deploy-restart    deploy + restart the service on the NAS"
	@echo "  clean             Remove build/ and ui/node_modules"

ui:
	cd ui && npm install && npm run build
	touch internal/webui/static/.gitkeep

build: ui
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY) $(CMD)

build-photomigrate:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/photomigrate ./cmd/photomigrate

test:
	go test -race ./...

# Run benchmarks and save results to benchmarks/<version>.txt.
# VERSION can be overridden: make bench VERSION=v1.2.3
bench:
	@mkdir -p benchmarks
	go test -bench=. -benchmem -count=5 -benchtime=1s \
	    ./internal/meta/ ./internal/importer/ \
	    | tee benchmarks/$(VERSION).txt
	@echo "Results saved to benchmarks/$(VERSION).txt"

vet:
	go vet ./...

lint: vet

# Cross-compile for Synology DSM.
# Most modern Synology NAS (DS923+, DS1522+, etc.) use x86_64.
# ARM-based models (DS220j, DS418, etc.) need arm64.
build-dsm-amd64: ui
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY)-linux-amd64 $(CMD)

build-dsm-arm64: ui
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY)-linux-arm64 $(CMD)

# Deploy to NAS without restarting the service.
# Uploads to a temp file then renames atomically so the running binary isn't disturbed.
# Override the host: make deploy NAS_HOST=user@192.168.1.x
deploy: build-dsm-amd64
	scp -O $(BUILD_DIR)/$(BINARY)-linux-amd64 $(NAS_HOST):$(NAS_BIN).new
	ssh $(NAS_HOST) "mv $(NAS_BIN).new $(NAS_BIN)"
	@echo "Deployed to $(NAS_HOST):$(NAS_BIN)"

# Stop the service, deploy, then start it again.
deploy-restart: build-dsm-amd64
	ssh $(NAS_HOST) "/var/packages/cardimportd/scripts/start-stop-status stop"
	scp -O $(BUILD_DIR)/$(BINARY)-linux-amd64 $(NAS_HOST):$(NAS_BIN)
	ssh $(NAS_HOST) "/var/packages/cardimportd/scripts/start-stop-status start"

clean:
	rm -rf $(BUILD_DIR)
	rm -rf ui/node_modules

# Build a Synology .spk package (requires the binary to be built first).
# Usage: make spk-amd64   or   make spk-arm64
SPK_VERSION := $(patsubst v%,%,$(VERSION))

spk-amd64: build-dsm-amd64
	$(MAKE) _spk ARCH=x86_64 BINARY_SUFFIX=linux-amd64

spk-arm64: build-dsm-arm64
	$(MAKE) _spk ARCH=armv8 BINARY_SUFFIX=linux-arm64

_spk:
	@echo "Building SPK for $(ARCH)..."
	@tmpdir=$$(mktemp -d) && \
	 target=$$tmpdir/target && \
	 mkdir -p $$target && \
	 cp $(BUILD_DIR)/$(BINARY)-$(BINARY_SUFFIX) $$target/$(BINARY) && \
	 cp config.example.yaml $$target/config.example.yaml && \
	 chmod +x $$target/$(BINARY) && \
	 tar czf $$tmpdir/package.tgz -C $$target . && \
	 rm -rf $$target && \
	 cp -r package/scripts $$tmpdir/ && \
	 cp -r package/conf    $$tmpdir/ && \
	 grep -v '^#' package/INFO | \
	 sed -e "s/arch=\"x86_64\"/arch=\"$(ARCH)\"/" \
	     -e "s/version=\"[^\"]*\"/version=\"$(SPK_VERSION)\"/" > $$tmpdir/INFO && \
	 chmod +x $$tmpdir/scripts/* && \
	 tar cf $(BUILD_DIR)/$(BINARY)-$(SPK_VERSION)-$(ARCH).spk -C $$tmpdir INFO package.tgz scripts conf && \
	 rm -rf $$tmpdir && \
	 echo "Created $(BUILD_DIR)/$(BINARY)-$(SPK_VERSION)-$(ARCH).spk"
