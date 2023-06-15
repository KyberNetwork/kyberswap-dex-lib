# Router Service

## Prerequisites
* Go 1.19+
* Redis
* Docker + docker-compose (optional)

## Quick start
```shell
# Clone repo
git clone git@github.com:KyberNetwork/router-service.git

# Install dependencies
go mod download

# Start Redis (master, slave, sentinel)
docker compose up -d

### Optional
### 
# Run redis-server with dump data 
# Firstly, download dump data for the chain at: 
https://console.cloud.google.com/storage/browser/shared-data-backup-0409cbcf/redis/pool-service-ethereum?pageState=(%22StorageObjectListTable%22:(%22f%22:%22%255B%255D%22))&authuser=0&hl=vi&prefix=&forceOnObjectsSortingFiltering=false
# Then, rename this downloaded file to `dump.rdb` and run this command in the same folder: 
redis-server
###
###

# Build app
go build -o app ./cmd/app

# Start indexer
./app -c internal/pkg/config/files/dev/ethereum.yaml indexer

# Start API
./app -c internal/pkg/config/files/dev/ethereum.yaml api
```

## Benchmark
_Note: pprof profile is only available in dev environment, so please use config yaml file in dev folder._
- After run the api server, go to the link http://localhost:8080/debug/pprof/profile?seconds=30 to download `profle` file in 30s time range
- Then run this command in the same folder
```shell
go tool pprof -http=":8081" ./profile
```

## Supported dexes
### Polygon
  - kyberswap
  - kyberswap-static
  - kyberswap-elastic
  - uniswapv3
  - sushiswap
  - quickswap
  - jetswap
  - wault
  - polycat
  - dfyn
  - polydex
  - firebird
  - oneswap
  - iron-stable
  - curve
  - synapse
  - balancer
  - gravity
  - cometh
  - apeswap
  - dinoswap
  - dodo
### Binance Smart Chain 
  - kyberswap
  - kyberswap-static
  - kyberswap-elastic
  - firebird
  - oneswap
  - ellipsis
  - nerve
  - apeswap
  - jetswap
  - mdex
  - pancake
  - wault
  - biswap
  - pancake-legacy
  - synapse
  - pantherswap
### Avalanche
  - kyberswap
  - kyberswap-static
  - kyberswap-elastic
  - iron-stable
  - pangolin
  - traderjoe
  - curve
  - synapse
  - axial
  - lydia
  - yetiswap
### Ethereum
  - kyberswap
  - kyberswap-static
  - kyberswap-elastic
  - uniswap (v2)
  - sushiswap
  - curve (3pool, aave, saave, hbtc, ren, sbtc, eurs, link)
  - balancer (v2)
  - synapse
  - shibaswap
  - defiswap
### Fantom
  - spookyswap
  - spiritswap
  - curve
  - jetswap
  - paintswap
  - sushiswap
  - kyberswap
  - kyberswap-static
  - kyberswap-elastic
  - beethovenx
  - synapse
  - morpheus
### Cronos
  - kyberswap
  - kyberswap-static
  - kyberswap-elastic
  - vvs finance
  - cronaswap
  - crodex
  - mmf
  - kryptodex
  - empiredex
  - photonswap
### Aurora
  - kyberswap
  - kyberswap-static
  - trisolaris
  - wannaswap
  - nearpad

### Arbitrum
  - kyberswap
  - kyberswap-static
  - kyberswap-elastic
  - sushiswap
  - curve
  - balancer
  - swapr

### BitTorrent Chain
  - kyberswap
  - kyberswap-static
  - kyberswap-elastic

### Oasis
  - kyberswap
  - kyberswap-static
  - kyberswap-elastic
  - valleyswap
  - yuzuswap
  - gemkeeper
  - lizard

### Velas
  - kyberswap
  - kyberswap-static
  - wagyuswap
  - astroswap

### Optimism
  - curve
  - synapse
  - uniswapv3
  - zipswap

### Rinkeby
  - kyberswap-elastic

## Development
### Install Dependencies

```bash
go mod download
```

### Build

```bash
go build -o app ./cmd/app
```

## Usage

```bash
$ app [global flags] command <params>
```
## Global Environments
| Environment variable | Note                                                                                                                                                                   |
|---------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------| 
| PUBLIC_RPC | Chain rpc is used to check block height and monitoring rpcs api                                                                                                        |
| RPCS       | Chain rpcs is used to call contract. It is separated by comma ex: `rpc1,rpc2,rpc3`                                                                                     
| LOG_LEVEL | Log level (`trace`,`debug`,`info`,`warn`,`error`,`fatal`,`panic`) default: `info`                                                                                      |
| REDIS_HOST | Redis host                                                                                                                                                             |
| REDIS_PORT | Redis host                                                                                                                                                             |
| REDIS_DBNUMBER | Redis dbNumber                                                                                                                                                         |
| REDIS_PASSWORD | Redis password                                                                                                                                                         |
| REDIS_PREFIX | Redis prefix                                                                                                                                                           |
| LOG_SENTRY_DSN | Sentry DSN                                                                                                                                                             |
| ALLOW_SUBGRAPH_ERROR | Boolean ("true", "false", "0", "1" are acceptable) - Sometimes, the subgraph of Uniswap V3 has indexing_error issue, but the error is non-fatal so we can still use it |
| BLACKLISTED_POOLS | string (list of blacklisted pools separated by comma                                                                                                                   |
---
## Predefined config file
| Path                               | Note |
|------------------------------------|-----------| 
| internal/pkg/config/ethereum.yaml  | ethereum configuration file |
| internal/pkg/config/polygon.yaml   | polygon configuration file |
| internal/pkg/config/bsc.yaml       | bsc configuration file |
| internal/pkg/config/avalanche.yaml | avalanche configuration file |
| internal/pkg/config/fantom.yaml    | fantom configuration file |
| internal/pkg/config/cronos.yaml    | cronos configuration file |
| internal/pkg/config/aurora.yaml    | aurora configuration file |
---
## Global Flags
| CLI Global parameter | Note |
|---------------|-----------| 
| c,config | config file |
---
## Commands

### api
Start the http server

Example: 
```bash
app -c internal/pkg/config/polygon.yaml api
```

### indexer
Start indexer server

Example: 
```bash
app -c internal/pkg/config/polygon.yaml indexer
```
#Multicall2 addresses
| Chain   | Address |
| ------- | ------- |
| Ethereum    | [0x5ba1e12693dc8f9c48aad8770482f4739beed696](https://etherscan.io/address/0x5ba1e12693dc8f9c48aad8770482f4739beed696) |
| BSC Mainnet | [0xed386Fe855C1EFf2f843B910923Dd8846E45C5A4](https://bscscan.com/address/0xed386Fe855C1EFf2f843B910923Dd8846E45C5A4) |
| Matic       | [0xed386Fe855C1EFf2f843B910923Dd8846E45C5A4](https://polygonscan.com/address/0xed386Fe855C1EFf2f843B910923Dd8846E45C5A4#code) |

# Curve config
## curve-base
```json
  {
    "id": "pool address",
    "name": "pool name (optional)",
    "type": "curve-base",
    "lpToken": "pool lp token",
    "aPrecision": "APrecision (in contract)",
    "version": 0,
    "tokens": [
      {
        "address": "token address",
        "precision": "PRECISION_MUL (in contract)",
        "rate": "RATES (in contract)"
      }
    ]
  }
```

| Version   | Function | Example |
| ------- | ------- | ------- |
| 0 | uint256 balances(uint256) | [0x5ba1e12693dc8f9c48aad8770482f4739beed696](https://etherscan.io/address/0x5ba1e12693dc8f9c48aad8770482f4739beed696) |
| 1 | uint256 balances(int128) | [0x93054188d876f558f4a66b2ef1d97d16edf0895b](https://bscscan.com/address/0x93054188d876f558f4a66b2ef1d97d16edf0895b) |

## curve-meta
```json
  {
      "id": "pool address",
      "type": "curve-meta",
      "name": "pool name (optional)",
      "lpToken": "pool lp token",
      "basePool": "BASE_POOL",
      "rateMultiplier": "rate_multiplier (in contract)",
      "aPrecision": "A_PRECISION (in contract)",
      "tokens": [
        {
          "address": "token address",
          "precision": "1"
        }
      ],
      "underlyingTokens": [
        "underlying tokens"
      ]
    }
```

## curve-aave
```json
  {
    "id": "pool address",
    "lpToken": "pool lp token",
    "type": "curve-aave",
    "name": "pool name (optional)",
    "tokens": [
      {
        "address": "token address",
        "precision": "PRECISION_MUL (in contract)"
      }
    ],
    "underlyingTokens": [
      "underlying tokens"
    ]
  }
```

## curve-tricrypto
```json
{
  "id": "pool address",
  "name": "pool name (optional)",
  "type": "curve-tricrypto",
  "tokens": [
    {
      "address": "token address",
      "precision": "PRECISIONS (in contract)"
    }
  ],
  "lpToken": "pool lp token"
}
```
