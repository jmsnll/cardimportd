# Changelog

## [0.13.15](https://github.com/jmsnll/cardimportd/compare/v0.13.14...v0.13.15) (2026-06-03)


### Features

* **photomigrate:** include .aac files in default extension set ([8c5d38a](https://github.com/jmsnll/cardimportd/commit/8c5d38a2a3749d01ef4dbf3c8d0dcde289120d52))
* **photomigrate:** skip [@ea](https://github.com/ea)Dir metadata dirs, fix collision race, add progress logging ([0089c3e](https://github.com/jmsnll/cardimportd/commit/0089c3e8b463c55bd7bb9b0bb3a6b922267273f9))


### Bug Fixes

* **ci:** bump Go 1.26.4, fix lint config and errcheck failures ([19eb51d](https://github.com/jmsnll/cardimportd/commit/19eb51d11246aba2709ace92848f5ad384c78c9f))
* **photomigrate:** move XMP sidecars alongside their raw files ([3739e65](https://github.com/jmsnll/cardimportd/commit/3739e654fa9b68993984ed82b32c35828fec5112))
* **photomigrate:** treat LRF files as sidecars alongside ARW ([e32ebe1](https://github.com/jmsnll/cardimportd/commit/e32ebe135428708a137b14278c6d33f52e83f672))


### Performance

* **photomigrate:** more workers by default, narrower resolve mutex ([bc42869](https://github.com/jmsnll/cardimportd/commit/bc428694cc7028cd3ec106ebdc270c37a2a7810d))
* **photomigrate:** use os.Rename for same-filesystem moves ([03291d2](https://github.com/jmsnll/cardimportd/commit/03291d2bece9085ef1cb1d920b7460148c244604))

## [0.13.14](https://github.com/jmsnll/cardimportd/compare/v0.13.13...v0.13.14) (2026-06-02)


### Bug Fixes

* **importer:** make MirrorFail test root-safe ([f2f79bb](https://github.com/jmsnll/cardimportd/commit/f2f79bbf467b62821c24bd7372ed9ead1acaa2af))


### Performance

* **photomigrate:** parallel worker pool for concurrent file moves ([c6dd448](https://github.com/jmsnll/cardimportd/commit/c6dd448d647a5ec918404a2dd4354695d0f0cf8d))

## [0.13.13](https://github.com/jmsnll/cardimportd/compare/v0.13.12...v0.13.13) (2026-06-02)


### Features

* import stamp, rated_only filter, EXIF rating, force-reimport, photomigrate tool ([1fa501b](https://github.com/jmsnll/cardimportd/commit/1fa501bab2dfaab1d7758d2985424702af44d40a))


### Bug Fixes

* acquire importMu before AlreadyImported check to prevent logical race ([f80eedc](https://github.com/jmsnll/cardimportd/commit/f80eedc48d43b0f9617847d5088d7d9410b08a1e))
* **dashboard:** show owner name on mounted cards and handle zero total during active import ([3a97077](https://github.com/jmsnll/cardimportd/commit/3a97077039ec52219345507333fb79a046126412))
* include rated_only in import stamp so config change invalidates it ([d4201ca](https://github.com/jmsnll/cardimportd/commit/d4201ca41a12ba94241eb10bbd42dd3cee1aaae6))
* log stamp walk errors; fix appendAssign lint in test helpers ([5e0a7e4](https://github.com/jmsnll/cardimportd/commit/5e0a7e403253f368d003d18c3737562c491691e3))
* move history log default to /usr/local/etc/cardimportd/ and expose -history-log flag ([09510f1](https://github.com/jmsnll/cardimportd/commit/09510f128d6ccb1fd27f7a37f83273c703910df6))
* **photomigrate:** SHA-256 content check for skip, errors.Is, src/dst overlap guard ([9bb4bcb](https://github.com/jmsnll/cardimportd/commit/9bb4bcbc115ed123c676323dff2174885379f4c2))

## [0.13.12](https://github.com/jmsnll/cardimportd/compare/v0.13.11...v0.13.12) (2026-06-02)


### Bug Fixes

* **ci:** output spk job binary to build/ so make _spk finds it ([7506cbe](https://github.com/jmsnll/cardimportd/commit/7506cbe167d5dd250b3c3eaf55a18cd8af498355))

## [0.13.11](https://github.com/jmsnll/cardimportd/compare/v0.13.10...v0.13.11) (2026-06-02)


### Bug Fixes

* **ci:** commit .gitkeep and recreate it after UI builds ([488b3d7](https://github.com/jmsnll/cardimportd/commit/488b3d751f91695eecbb1111feda40447e34523d))
* include gitkeep file needed for the release process ([c921de2](https://github.com/jmsnll/cardimportd/commit/c921de255d0cff2fd82df60fb983f0a3e846a527))

## [0.13.10](https://github.com/jmsnll/cardimportd/compare/v0.13.9...v0.13.10) (2026-06-02)


### Features

* **config:** add User type and migrate card owners into cfg.Users on load ([38a1d99](https://github.com/jmsnll/cardimportd/commit/38a1d99c64ebcb31b3006cd8af115f2f3550a733))
* **ui/cards:** owner dropdown + ActivateCardModal for pending cards ([b5fdc76](https://github.com/jmsnll/cardimportd/commit/b5fdc76c36f6809ceb9b76614fc29d9567e0a11a))
* **ui/settings:** add People section for user management ([c7fc070](https://github.com/jmsnll/cardimportd/commit/c7fc070bad22705bb61655387e8793df28ac1cb7))
* **webui:** add User CRUD endpoints and DestinationTemplate field ([46af77a](https://github.com/jmsnll/cardimportd/commit/46af77a66558abdf643692c37f24a1910510f793))

## [0.13.9](https://github.com/jmsnll/cardimportd/compare/v0.13.8...v0.13.9) (2026-06-02)


### Features

* trigger import immediately when pending card is activated ([0031cea](https://github.com/jmsnll/cardimportd/commit/0031ceabd00bee186f07700bd6204ce24a7cc231))
* **ui/cards:** refresh button, mounted badge, last import, preflight scan ([e9305db](https://github.com/jmsnll/cardimportd/commit/e9305dbadad4eb57d0bc18014eadb76ec61b609b))
* **ui:** add History tab, ImportProgress component, SSE reconnect resync ([b9a88ef](https://github.com/jmsnll/cardimportd/commit/b9a88ef6126e12d65ca08dc39e3a9adc51c74c94))
* **webui:** add history, status, preflight endpoints + card lifecycle SSE events ([29eede2](https://github.com/jmsnll/cardimportd/commit/29eede2e2c8426a6dffe7ab07a696dcd357f60c5))

## [0.13.8](https://github.com/jmsnll/cardimportd/compare/v0.13.7...v0.13.8) (2026-06-02)


### Bug Fixes

* assign UUID via UUID.txt for cards with no blkid identifier ([58852f8](https://github.com/jmsnll/cardimportd/commit/58852f880c07b8d872dc80ced351d4b044f4e7f3))
* fall back to geometry fingerprint for SD cards with no blkid identifier ([6d87628](https://github.com/jmsnll/cardimportd/commit/6d8762833fa8072933f4d86333a394ed80a84111))
* use JSON log format in package scripts to match app output ([d0b2040](https://github.com/jmsnll/cardimportd/commit/d0b204035acdd6d053da3d553b19550157bc0508))

## [0.13.7](https://github.com/jmsnll/cardimportd/compare/v0.13.6...v0.13.7) (2026-06-02)


### Bug Fixes

* detect cards mounted at startup and handle exFAT SD cards with no UUID ([205bffa](https://github.com/jmsnll/cardimportd/commit/205bffadf7bd92bda2fe58fd4442710226cb26b1))

## [0.13.6](https://github.com/jmsnll/cardimportd/compare/v0.13.5...v0.13.6) (2026-06-01)


### Bug Fixes

* improve package information ([928ea16](https://github.com/jmsnll/cardimportd/commit/928ea16b87fd535df80b8a9ea8f49c964e3f14ad))

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
