# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.11.6] - 2023-09-11

### Added
- Fix: handle error insufficient liquidity in `PMM` price levels (#140)


## [v0.11.5] - 2023-09-11

### Added
- Support `pancake-stable` on Bsc and `curve` on Base (#137) (#139)


## [v0.11.4] - 2023-09-08

### Added
- Fetch `limit-order` Operator signature (#136) (#138)


## [v0.11.3] - 2023-09-07

### Fixed
- Fix: store the recipient in `RFQ` extra to use in encoding


## [v0.11.1] - 2023-09-06

### Added
- Handle RFQ for Kyber `PMM` (#134)


## [v0.11.0] - 2023-09-05

### Added
- Integrate Kyber `PMM` (#132)

### Changed
- Refactor: moved bootstrap code to pool-service


## [v0.10.4] - 2023-08-29

### Added
-  Integrate velodromev2 (#128)


## [v0.10.3] - 2023-08-29

### Added
- Mapping token address for balancer and move mapping outside for new pool (#119) (#126)


## [v0.10.2] - 2023-08-28

### Fixed
- Lowercase `synapse` config file (#124)


## [v0.10.1] - 2023-08-25

### Added
- Support `synapse` on base (#121)


## [v0.10.0] - 2023-08-24

### Added 
- Update LO pool list updater/tracker multi SCs (#116)
  - Add config flag
  - Return contractAddress in sim meta
