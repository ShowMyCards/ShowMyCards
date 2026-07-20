# Changelog

## [0.4.0](https://github.com/ShowMyCards/ShowMyCards/compare/v0.3.0...v0.4.0) (2026-07-20)


### Features

* add support for ManaBox syntax in import screen ([#145](https://github.com/ShowMyCards/ShowMyCards/issues/145)) ([dec0507](https://github.com/ShowMyCards/ShowMyCards/commit/dec05074b33f33cf2a9076555ed7d9d9e6ddfd07))
* allow partial card relocation ([#122](https://github.com/ShowMyCards/ShowMyCards/issues/122)) ([1a54d8b](https://github.com/ShowMyCards/ShowMyCards/commit/1a54d8be6865d3a174236f5bc5f2668f33a6641a))
* **backend:** add deck lists foundation (FR98 milestone 1a) ([#113](https://github.com/ShowMyCards/ShowMyCards/issues/113)) ([7020498](https://github.com/ShowMyCards/ShowMyCards/commit/702049810b629f2f5ca6558b44d6a9f00e56dbf0))
* **backend:** add scheduled gzipped JSON backups (FR90) ([#131](https://github.com/ShowMyCards/ShowMyCards/issues/131)) ([ff7d014](https://github.com/ShowMyCards/ShowMyCards/commit/ff7d014f4653e8887d8990dec375f9c6885682cb))
* **backend:** add symbology scheduled task and symbol SVG endpoint ([#104](https://github.com/ShowMyCards/ShowMyCards/issues/104)) ([#130](https://github.com/ShowMyCards/ShowMyCards/issues/130)) ([255998f](https://github.com/ShowMyCards/ShowMyCards/commit/255998f4021d800f7d19d2b31b714a6f82e748a9))
* **backend:** deck allocation service + deck item endpoints (FR98 1b) ([#146](https://github.com/ShowMyCards/ShowMyCards/issues/146)) ([28a976a](https://github.com/ShowMyCards/ShowMyCards/commit/28a976a61fadca282fea6a15f86172058a933cdc))
* **backend:** ingest Scryfall bulk data as gzipped JSONL ([#155](https://github.com/ShowMyCards/ShowMyCards/issues/155)) ([38c0019](https://github.com/ShowMyCards/ShowMyCards/commit/38c0019b2e63bcbf5ed5c8eee57ff5d9580ac7f0))
* display card title in card language ([#110](https://github.com/ShowMyCards/ShowMyCards/issues/110)) ([137e34f](https://github.com/ShowMyCards/ShowMyCards/commit/137e34ff08613f732699503af8b66586db146f3a))


### Bug Fixes

* add edit and delete controls for lists ([#124](https://github.com/ShowMyCards/ShowMyCards/issues/124)) ([#127](https://github.com/ShowMyCards/ShowMyCards/issues/127)) ([8e6776e](https://github.com/ShowMyCards/ShowMyCards/commit/8e6776ed3247f788053716cbf5047641462c459c))
* allow sending cards to arbitrary locations ([#147](https://github.com/ShowMyCards/ShowMyCards/issues/147)) ([cab66c1](https://github.com/ShowMyCards/ShowMyCards/commit/cab66c129b029036211a87183f00a8880b85ed24))
* back-fill English prices for non-English list items ([#128](https://github.com/ShowMyCards/ShowMyCards/issues/128)) ([e2a7735](https://github.com/ShowMyCards/ShowMyCards/commit/e2a7735a6d0027be672b1f1097f128d9fa98aa85))
* **backend:** bump Go toolchain to 1.26.4 for stdlib vulns ([#120](https://github.com/ShowMyCards/ShowMyCards/issues/120)) ([30f7ac7](https://github.com/ShowMyCards/ShowMyCards/commit/30f7ac7d8785bc31bd62ed3003c317d378db7d7d))
* **deps:** clear govulncheck and bun audit findings blocking all merges ([#150](https://github.com/ShowMyCards/ShowMyCards/issues/150)) ([2dcbd34](https://github.com/ShowMyCards/ShowMyCards/commit/2dcbd3448750bb2d94d5656b46ce9f4906066879))
* **deps:** pin esbuild &gt;=0.28.1 to clear RCE advisory GHSA-gv7w-rqvm-qjhr ([#142](https://github.com/ShowMyCards/ShowMyCards/issues/142)) ([2f02e0f](https://github.com/ShowMyCards/ShowMyCards/commit/2f02e0fd8c0ba8472d6851db6bcdb005f78ecb87))
* display prices for non-english cards ([#118](https://github.com/ShowMyCards/ShowMyCards/issues/118)) ([409fe62](https://github.com/ShowMyCards/ShowMyCards/commit/409fe62a37d11f70ae7d0290b2367d288c43bc76))
* **docker:** repin golang base image to digest shipping Go 1.26.4 ([#121](https://github.com/ShowMyCards/ShowMyCards/issues/121)) ([d109bde](https://github.com/ShowMyCards/ShowMyCards/commit/d109bde3e0d671c666ad0f48e6998db2377de724))
* **frontend:** card back button returns to originating page ([#116](https://github.com/ShowMyCards/ShowMyCards/issues/116)) ([#129](https://github.com/ShowMyCards/ShowMyCards/issues/129)) ([77c0fb4](https://github.com/ShowMyCards/ShowMyCards/commit/77c0fb4f2a2e990824361dda0e7e69280d5a0a87))
* **frontend:** close create-list dialog and surface feedback ([#123](https://github.com/ShowMyCards/ShowMyCards/issues/123)) ([#126](https://github.com/ShowMyCards/ShowMyCards/issues/126)) ([ef5c838](https://github.com/ShowMyCards/ShowMyCards/commit/ef5c8386997344f9784d256df692e7919d4dd140))
* group card instances by storage location ([#114](https://github.com/ShowMyCards/ShowMyCards/issues/114)) ([bd97ff9](https://github.com/ShowMyCards/ShowMyCards/commit/bd97ff9c0c3019eb458b49bfe294ef196dfceba7))
* load all storage locations in non-paginating views ([#148](https://github.com/ShowMyCards/ShowMyCards/issues/148)) ([f06e485](https://github.com/ShowMyCards/ShowMyCards/commit/f06e485945437329e7c570ef64b4d29edbecba2d))

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
