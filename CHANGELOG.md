# Changelog

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
