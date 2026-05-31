# Changelog

## [0.13.5](https://github.com/jmsnll/cardimportd/compare/v0.13.4...v0.13.5) (2026-05-31)


### Bug Fixes

* **ci:** restore .gitkeep before goreleaser to avoid dirty state ([54a3578](https://github.com/jmsnll/cardimportd/commit/54a3578f25433eaae10795bf45625aac35c205a3))

## [0.13.4](https://github.com/jmsnll/cardimportd/compare/v0.13.3...v0.13.4) (2026-05-31)


### Bug Fixes

* **ci:** chain release workflow via workflow_dispatch from release-please ([3f43dd4](https://github.com/jmsnll/cardimportd/commit/3f43dd4a979b44f069c7ed7c2f53667326a15fec))
* **ci:** trigger release workflow on published release, not tag push ([990fddf](https://github.com/jmsnll/cardimportd/commit/990fddff3a5776eac7fcba4fc4d878e1808a99de))
* **lint:** add errcheck exclusions for intentional defer/SSE patterns ([55d8165](https://github.com/jmsnll/cardimportd/commit/55d8165a4af942345a653134d7516bba7b19eec4))
* **lint:** resolve all 102 golangci-lint issues ([a8e367c](https://github.com/jmsnll/cardimportd/commit/a8e367c7706b692e7ec0a297bd46819174ecbde5))

## [0.13.3](https://github.com/jmsnll/cardimportd/compare/v0.13.2...v0.13.3) (2026-05-31)


### Bug Fixes

* add vite-env.d.ts to resolve TS2882 CSS side-effect import error ([540ac1f](https://github.com/jmsnll/cardimportd/commit/540ac1fb19bbc96c3b352a578aa5ac7fe62bc61d))
* **ci:** commit static/.gitkeep so go:embed resolves without a UI build ([29028ff](https://github.com/jmsnll/cardimportd/commit/29028ff09e65ef42efcd0156500f6276689d6df2))
* use all:static embed directive to include dot-files in CI ([7d8d2db](https://github.com/jmsnll/cardimportd/commit/7d8d2dba2768571c82af4db451a3c3c3b0593e9c))

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
