# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.3.3] - 2023-03-22
### Fixed
- wrong convert json number to big.Int

## [2.3.2] - 2023-03-22
### Fixed
- missing refactor code for old route api (`/route/encode/`)

## [2.3.1] - 2023-03-22
### Added
- support Kyberswap Limit Order (arbitrum)

## [2.3.0] - 2023-03-21
### Added
- support Kyberswap Limit Order (bsc, fantom)

## [2.2.2] - 2023-03-21
### Fixed
- wrong fee amount validation when charge fee by currency_out

## [2.2.1] - 2023-03-21
### Changed
- update cache config on ethereum, polygon, arbitrum and optimism

## [2.2.0] - 2023-03-20
### Added
- support permit swap tx
- new fee tiers for Elastic

## [2.1.4] - 2023-03-14
### Changed
- update elastic subgraph for scanner on arbitrum

## [2.1.3] - 2023-03-09
### Added
- add cache and timeout API handlers
### Changed
- return gasPrice in Wei instead of Gwei
### Fixed
- do not set default slippage when user set slippage tolerance to zero
- fix camelot encoding logic

## [2.1.2] - 2023-03-01
### Fixed
- add a timeout for requests to subgraph to avoid the goroutine hanging up

## [2.1.1] - 2023-02-28
### Added
- add request ui as tag for tracing GetRoutes API
### Fixed
- return 400 when pool set is empty
- return 400 when no path found


## [2.1.0] - 2023-02-23
### Added
- support camelot dex
- add validations for `feeAmount` and `amountIn` in GetRoutes and BuildRoute APIs
- add error handling for route not found
- add default slippageTolerance for BuildRoute API
- set epsilon for cached route

## [2.0.0] - 2023-02-20
### Added
- GetRoutes and BuildRoute APIs

### Changed
- Separate configs for dev/prod

## [1.29.5] - 2023-02-20
### Added
- add logs and metrics for UpdateReserves function
- add los for uniswapv3 scanner

## [1.29.4] - 2023-02-19
### Added
- add log pool size

## [1.29.3] - 2023-02-19
### Added
- add maxAmountIn validation
### Changed
- decrease poolSet size

## [1.29.2] - 2023-02-15
### Added
- add recover for uniswapv3 UpdateReserves function

## [1.29.1] - 2023-02-14
### Added
- add log when get/set poolIdsByExchange map on scanner

## [1.29.0] - 2023-02-08
### Added
- support fraxswap dex

### Fixed
- Platypus validation bug

## [1.28.8] - 2023-01-18
### Changed
- encode correct swapFee for biswap

## [1.28.7] - 2023-01-18
### Fixed
- curve base pool, prefer getting data from pool over the main registry

## [1.28.6] - 2023-01-14
### Fixed
- uniswapv3 and promm sdks

## [1.28.5] - 2023-01-11
### Fixed
- fallback when there is no market price

## [1.28.4] - 2023-01-10
### Fixed
-  issue Curve Tricrypto division by 0

## [1.28.3] - 2023-01-10
### Fixed
-  issue Curve Two division by 0

### Changed
- update default gas for dexes
- refactor dodo + dmm gas estimation

## [1.28.2] - 2023-01-09
### Fixed
- handle issue Platypus subgraph returns zero address pool

## [1.28.1] - 2023-01-04
### Fixed
- "transfer amount exceeds balance" when swapping through two uniswap pools

## [1.28.0] - 2023-01-03
### Changed
- Encode V5 for Meta Aggregation Router V2 and Executor V2

## [1.27.1] - 2022-11-10
### Fixed
- curve scanner: skip init pool when poolPath is not configured

## [1.27.0] - 2022-11-10
### Added
- support Lido
- blacklist pool

### Changed
- refactor: context

## [1.26.0] - 2022-11-09
### Added
- support Metavault

## [1.25.1] - 2022-11-03
### Fixed
- fix wrong subgraph url in config

## [1.25.0] - 2022-11-03

### Added
- Curve V2
- indicate market price available in client data

### Fixed
- Curve issue on Optimism (add new type plain oracle pool)
- bring back ctx span

### Changed
- refactor: replace string amountIn by *big.Int amountIn

## [1.24.2] - 2022-10-31

### Fixed
- add redis timeout for bsc scanner

## [1.24.1] - 2022-10-31

### Fixed
- remove default Redis prefix to fix bug on production

## [1.24.0] - 2022-10-28

### Added
- dynamic config reloading (enabled dexes, whitelisted tokens, feature flags)

### Fixed
- update gas estimation for all pools

### Changed
- improve logger (use Zap as default)

## [1.23.0] - 2022-10-27
### Added
- support Madmex

## [1.22.0] - 2022-10-19

### Added
- add integrity for client data
- integrate token catalog (push new token to token catalog service)

## [1.21.4] - 2022-10-18

### Changed
- update executor contract ethereum (maker-psm + hashflow)
- enable maker-psm
- update uniswapv3 scan params

### Fixed
- update maker psm encode swap data abi

## [1.21.3] - 2022-10-12

### Changed
- whitelist AAVE and LINK on Avalanche


## [1.21.2] - 2022-10-06

### Fixed
- some Uniswap V3 pools state are not updated due to RPC call error

## [1.21.1] - 2022-10-05

### Changed
- disable MakerPSM

## [1.21.0] - 2022-10-04

### Added
- support Synthetix

## [1.20.0] - 2022-09-29

### Added
- support MakerPSM

### Changed
- change updateReserveBulk for UniswapV3, Elastic

### Fixed
- fix encoding mode determination logic

## [1.19.6] - 2022-09-26

### Fixed
- dystopia bug
- gmx + platypus CanSwapTo

## [1.19.5] - 2022-09-21

### Changed
- whitelist KNC (Arbitrum and Optimism)

## [1.19.4] - 2022-09-21

### Changed
- whitelist wstETH (Arbitrum and Optimism)
- parse duration from config files

## [1.19.3] - 2022-09-15

### Added
- support Ethereum POW

## [1.19.2] - 2022-09-10

### Changed
- whitelist MA(Avalanche) and LINK(Polygon)

## [1.19.1] - 2022-09-06

### Fixed
- avoid using panic in sdks (uniswapv3 + promm), which crashed the API

## [1.19.0] - 2022-09-05

### Added
- support GMX exchange

## [1.18.1] - 2022-08-26

### Changed
- disable Biswap due to changeable swap fee

## [1.18.0] - 2022-08-25

### Added
- add MMF dex
- add MMF token to whitelist
- add VERSA token to whitelist
- add dexes API to return enabled dexes

### Fixed
- bug can't swap via elastic pool that has full range liquidity

## [1.17.6] - 2022-08-18

### Fixed
- token API: prefer market price from Coingecko for accuracy

## [1.17.5] - 2022-08-18

### Fixed
- price API: prefer market price from Coingecko for accuracy

## [1.17.4] - 2022-08-18

### Changed
- prefer market price from Coingecko for accuracy
- reduce requests per minute to avoid Coingecko rate limit

## [1.17.3] - 2022-08-15

### Added
- add miMatic to Polygon whitelist tokens
- add YUSD to Avalanche whitelist tokens
- add sAVAX to Avalanche whitelist tokens

## [1.17.2] - 2022-08-11

### Fixed
- balancer weighted pool formula
- balancer pools swap fees are not updated


## [1.17.1] - 2022-08-10

### Removed
- Dystopia dex support on Polygon

## [1.17.0] - 2022-08-09
### Added
- support balancer meta stable pool
- add stMATIC to Polygon whitelist tokens
- add KNC to whitelist on 5 chains: ethereum, polygon, bsc, avalanche, fantom
- add platypus pure to update price logic

### Changed
- refactor aggregator scanner
- new aggregation routers (MetaAggregationRouter)

### Fixed
- platypus missing reserves
- velodrome incorrect decimals

### Removed
- remove sAVAX, UST from Avalanche whitelist tokens
- remove firebird dex
- remove balancer stable/metastable from calculating price logic
