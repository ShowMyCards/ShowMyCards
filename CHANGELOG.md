# Changelog

## [0.3.0](https://github.com/ShowMyCards/ShowMyCards/compare/v0.2.0...v0.3.0) (2026-05-26)


### Features

* add filter for cards in collection ([#87](https://github.com/ShowMyCards/ShowMyCards/issues/87)) ([9348ad8](https://github.com/ShowMyCards/ShowMyCards/commit/9348ad89609a5621eb2fe48f6b458462153e8a59))
* add search builder ([#101](https://github.com/ShowMyCards/ShowMyCards/issues/101)) ([00ca3b4](https://github.com/ShowMyCards/ShowMyCards/commit/00ca3b4b5a3a6e8191b7cd9ae3999c8e7dbf6e55))
* enable SQLite WAL mode for safer hot backups ([#100](https://github.com/ShowMyCards/ShowMyCards/issues/100)) ([68bb8a1](https://github.com/ShowMyCards/ShowMyCards/commit/68bb8a1799d5c619aa12305e2505d1cce7e285ac))
* Scryfall shows sync state and failures in a UI Banner ([#99](https://github.com/ShowMyCards/ShowMyCards/issues/99)) ([8f014fd](https://github.com/ShowMyCards/ShowMyCards/commit/8f014fd3213be1230e7d3f3df65bb4d3592f0566))


### Bug Fixes

* address govulns, remove vendor directory ([#108](https://github.com/ShowMyCards/ShowMyCards/issues/108)) ([982e752](https://github.com/ShowMyCards/ShowMyCards/commit/982e752e5f30c62cfb2a4fae4865fa0d3ce9f938))
* drop non-valid settings keys ([#91](https://github.com/ShowMyCards/ShowMyCards/issues/91)) ([af8399b](https://github.com/ShowMyCards/ShowMyCards/commit/af8399b7cff534dc876d6f55866e9fd5276ec16d))
* frontend tests locally and in ci now work correctly ([#102](https://github.com/ShowMyCards/ShowMyCards/issues/102)) ([ec33819](https://github.com/ShowMyCards/ShowMyCards/commit/ec338196130afc44c75615ee41d7f2df10049a85))
* inventory reduction handling ([#96](https://github.com/ShowMyCards/ShowMyCards/issues/96)) ([e5903e1](https://github.com/ShowMyCards/ShowMyCards/commit/e5903e1de630bb2257fa350d68d4e66bc768a0cf))
* use proper import icon ([#97](https://github.com/ShowMyCards/ShowMyCards/issues/97)) ([05376b6](https://github.com/ShowMyCards/ShowMyCards/commit/05376b68acca32e9254fc6fc4c7a7edcf9d22e10))

## [0.2.0](https://github.com/ShowMyCards/ShowMyCards/compare/v0.1.0...v0.2.0) (2026-05-14)


### Features

* add card language for searching ([#81](https://github.com/ShowMyCards/ShowMyCards/issues/81)) ([8c5356d](https://github.com/ShowMyCards/ShowMyCards/commit/8c5356d92211573c7582c9cf2dded2205fed3349))
* display card language ([#67](https://github.com/ShowMyCards/ShowMyCards/issues/67)) ([6f1bb20](https://github.com/ShowMyCards/ShowMyCards/commit/6f1bb2069d1ca67087395d3acbb180950d9ab587))


### Bug Fixes

* **backend:** stabilise bulk data test on Linux CI ([#77](https://github.com/ShowMyCards/ShowMyCards/issues/77)) ([563b385](https://github.com/ShowMyCards/ShowMyCards/commit/563b3853fe3699b8c006d956d3b5484978239b5c))
* **ci:** allow GitHub Actions to create PRs in repo-settings script ([5f007db](https://github.com/ShowMyCards/ShowMyCards/commit/5f007db0124fcd71c081fbf1bd20bead74a29ebe))
* **ci:** allow GitHub Actions to create PRs in repo-settings script ([2d5f83b](https://github.com/ShowMyCards/ShowMyCards/commit/2d5f83bcd065c8527e972250fc832e8a7fd38800))
* **ci:** unblock release-please PRs from required checks ([#85](https://github.com/ShowMyCards/ShowMyCards/issues/85)) ([80e6be8](https://github.com/ShowMyCards/ShowMyCards/commit/80e6be8b22eb40162169b4bf82495a92b851d4ef))
* database location ([69f2752](https://github.com/ShowMyCards/ShowMyCards/commit/69f27525d05a8d61ef49a0a78e47f60f06e90f50))
* database location ([5590e62](https://github.com/ShowMyCards/ShowMyCards/commit/5590e62987420b19826bae76e9d5454246808438))
* inconsistent backend api path usage ([#64](https://github.com/ShowMyCards/ShowMyCards/issues/64)) ([0ffc268](https://github.com/ShowMyCards/ShowMyCards/commit/0ffc2686447dc103040dabf3f65f0fe882800d1e))
* scryfall rate limit ([#70](https://github.com/ShowMyCards/ShowMyCards/issues/70)) ([b624472](https://github.com/ShowMyCards/ShowMyCards/commit/b624472c0598673b6cc5f5e03430354b06f7785f))
* **security:** bump go toolchain to 1.26.3 to resolve govulncheck findings ([#82](https://github.com/ShowMyCards/ShowMyCards/issues/82)) ([c5bb30b](https://github.com/ShowMyCards/ShowMyCards/commit/c5bb30bd6c01c50c953a7ebbdb74932070764a9f)), closes [#74](https://github.com/ShowMyCards/ShowMyCards/issues/74)
* **security:** resolve bun audit findings in frontend and website ([#78](https://github.com/ShowMyCards/ShowMyCards/issues/78)) ([ea26ed8](https://github.com/ShowMyCards/ShowMyCards/commit/ea26ed861c1c2735b237865c21e2d24b400749fd))
