.PHONY: build test lint vet build-dsm-amd64 build-dsm-arm64 clean

BINARY  := cardimportd
CMD     := ./cmd/$(BINARY)

build:
	go build -o $(BINARY) $(CMD)

test:
	go test -race ./...

vet:
	go vet ./...

lint: vet

# Cross-compile for Synology DSM.
# Most modern Synology NAS (DS923+, DS1522+, etc.) use x86_64.
# ARM-based models (DS220j, DS418, etc.) need arm64.
build-dsm-amd64:
	GOOS=linux GOARCH=amd64 go build -o $(BINARY)-linux-amd64 $(CMD)

build-dsm-arm64:
	GOOS=linux GOARCH=arm64 go build -o $(BINARY)-linux-arm64 $(CMD)

clean:
	rm -f $(BINARY) $(BINARY)-linux-amd64 $(BINARY)-linux-arm64

# Build a Synology .spk package (requires the binary to be built first).
# Usage: make spk-amd64   or   make spk-arm64
SPK_VERSION := 1.0.0-0001

spk-amd64: build-dsm-amd64
	$(MAKE) _spk ARCH=x86_64 BINARY_SUFFIX=linux-amd64

spk-arm64: build-dsm-arm64
	$(MAKE) _spk ARCH=armv8 BINARY_SUFFIX=linux-arm64

_spk:
	@echo "Building SPK for $(ARCH)..."
	@tmpdir=$$(mktemp -d) && \
	 target=$$tmpdir/target && \
	 mkdir -p $$target && \
	 cp $(BINARY)-$(BINARY_SUFFIX) $$target/$(BINARY) && \
	 cp config.example.yaml $$target/config.example.yaml && \
	 chmod +x $$target/$(BINARY) && \
	 tar czf $$tmpdir/package.tgz -C $$target . && \
	 cp -r package/scripts $$tmpdir/ && \
	 cp -r package/conf    $$tmpdir/ && \
	 grep -v '^#' package/INFO | \
	 sed -e "s/arch=\"x86_64\"/arch=\"$(ARCH)\"/" \
	     -e "s/version=\"[^\"]*\"/version=\"$(SPK_VERSION)\"/" > $$tmpdir/INFO && \
	 chmod +x $$tmpdir/scripts/* && \
	 tar cf $(BINARY)-$(SPK_VERSION)-$(ARCH).spk -C $$tmpdir . && \
	 rm -rf $$tmpdir && \
	 echo "Created $(BINARY)-$(SPK_VERSION)-$(ARCH).spk"
