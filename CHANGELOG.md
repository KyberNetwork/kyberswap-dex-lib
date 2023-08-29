# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v1.21.5] - 2023-08-29

### Added
- Support `dackie-v3` and `rocketswap-v2` on Base (#393)

### Fixed
- lowercase whitelisttoken MIM (#393)


## [v1.21.4] - 2023-08-28

### Added 
- Dynamic config cache for shrink decimal func on all chains (#389)


## [v1.21.3] - 2023-08-28

### Added 
- Support `synapse` on base (#388)

### Fixed
- Fix bug `alien-base` wrong rate (#388)


## [v1.21.2] - 2023-08-25

### Added
- Support `baso`, `synthswap-v3`, `synthswap` on base (#385)
- Use LO contract address from pool if available (#378)
  - Use LO contract address from pool if available 
  - Check valid eth address 
  - Bump dex-lib v0.10.0


## [v1.21.1] - 2023-08-24

### Added
- Support `kyberswap-elastic`, `horizon`, `balancer`, `balancer-composable-stable` on base (#380)
- Support `balancer`, `balancer-composable` on avalanche (#380)


## [v1.21.0] - 2023-08-23

### Added 
- Support base (#374)


## [v1.20.0] - 2023-08-23   

### Added
- New cache route mechanism and dynamic config cache (#360)


## [v1.19.2] - 2023-08-21   

### Added
- AG-665: integrate echo-dex-v3 (#358)
- Add linea config files (#362)

### Fixed
- Cleanup doveswap-v3 (#359)
- Revert back to self-hosted (#361)
- AG-645: fix bug invalid swap (#364)
