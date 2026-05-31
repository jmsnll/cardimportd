# Changelog

## [0.13.2](https://github.com/jmsnll/cardimportd/compare/v0.13.1...v0.13.2) (2026-05-31)


### Features

* add blkid startup check and exponential backoff on poll errors ([ddb43b3](https://github.com/jmsnll/cardimportd/commit/ddb43b384e2b720bbd0339883beee8ff61c05041))
* add blkid startup check and exponential backoff on poll errors ([#20](https://github.com/jmsnll/cardimportd/issues/20)) ([4e2c750](https://github.com/jmsnll/cardimportd/commit/4e2c750230ef92c92b564bd241d240d4dc9538c3))
* add watchdog restart loop and upgrade path to DSM package ([2e9a648](https://github.com/jmsnll/cardimportd/commit/2e9a64853a5013cabfb0b34f44771d683dc86f80))
* add watchdog restart loop and upgrade path to DSM package ([#18](https://github.com/jmsnll/cardimportd/issues/18)) ([0eb27ba](https://github.com/jmsnll/cardimportd/commit/0eb27baf8f6b693920c74366163f9023db6f8b9c))
* add webui bind address, optional basic auth, and health endpoint ([8258376](https://github.com/jmsnll/cardimportd/commit/8258376cd14157bd6caa3ef54358f2d6b930a816))
* add webui bind address, optional basic auth, and health endpoint ([#21](https://github.com/jmsnll/cardimportd/issues/21)) ([e5e8be6](https://github.com/jmsnll/cardimportd/commit/e5e8be64d9554a2772b3edf465c1d6c43d5eedb5))
* cap history log at 10,000 entries to prevent unbounded growth ([26a4259](https://github.com/jmsnll/cardimportd/commit/26a425926ab41118b47360147895601b46937607))
* cap history log at 10,000 entries to prevent unbounded growth ([#19](https://github.com/jmsnll/cardimportd/issues/19)) ([401a785](https://github.com/jmsnll/cardimportd/commit/401a785f0f77b4f1338135b6b4b8c2bfd62bacfd))
* complete UI — settings, notifications, card fields, live progress ([cd53fda](https://github.com/jmsnll/cardimportd/commit/cd53fda04e78f18f92abaf796bfa0798b526faa0))


### Bug Fixes

* **ci:** install golangci-lint v2 via correct module path ([078251a](https://github.com/jmsnll/cardimportd/commit/078251a37dca80a297d47557104757d63f01699b))
* **ci:** install golangci-lint via go install to avoid Go version mismatch ([8cd278e](https://github.com/jmsnll/cardimportd/commit/8cd278e3cd3320f33c814b7b7a886a9071ed81d1))
* **ci:** install govulncheck directly to avoid action version issues ([6ab0d45](https://github.com/jmsnll/cardimportd/commit/6ab0d45c7f8347ef99d17589a741f4b38226fcd2))
* upgrade to Go 1.26.3 and x/net v0.53.0 to address CVEs ([1c16b05](https://github.com/jmsnll/cardimportd/commit/1c16b05bf42e715731ba91e9290dd3a73b40f7bb))

## [0.13.1](https://github.com/jmsnll/cardimportd/compare/v0.13.0...v0.13.1) (2026-05-31)


### Features

* add blkid startup check and exponential backoff on poll errors ([ddb43b3](https://github.com/jmsnll/cardimportd/commit/ddb43b384e2b720bbd0339883beee8ff61c05041))
* add watchdog restart loop and upgrade path to DSM package ([2e9a648](https://github.com/jmsnll/cardimportd/commit/2e9a64853a5013cabfb0b34f44771d683dc86f80))
* add webui bind address, optional basic auth, and health endpoint ([8258376](https://github.com/jmsnll/cardimportd/commit/8258376cd14157bd6caa3ef54358f2d6b930a816))
* cap history log at 10,000 entries to prevent unbounded growth ([26a4259](https://github.com/jmsnll/cardimportd/commit/26a425926ab41118b47360147895601b46937607))
* complete UI — settings, notifications, card fields, live progress ([cd53fda](https://github.com/jmsnll/cardimportd/commit/cd53fda04e78f18f92abaf796bfa0798b526faa0))


### Bug Fixes

* **ci:** install golangci-lint via go install to avoid Go version mismatch ([8cd278e](https://github.com/jmsnll/cardimportd/commit/8cd278e3cd3320f33c814b7b7a886a9071ed81d1))
* **ci:** install govulncheck directly to avoid action version issues ([6ab0d45](https://github.com/jmsnll/cardimportd/commit/6ab0d45c7f8347ef99d17589a741f4b38226fcd2))
* **ui:** repair tab switching, loading state, and credential fields ([6e42dc9](https://github.com/jmsnll/cardimportd/commit/6e42dc9a8365fd426c39ebb37700fc8aaccbf931))
