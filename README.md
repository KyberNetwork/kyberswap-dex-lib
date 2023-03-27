# Aggregator Backend

## Prerequisites
* Go 1.16+
* Redis
* Docker + docker-compose (optional)

## Quick start
```shell
# Clone repo
git clone git@github.com:KyberNetwork/dmm-aggregator-backend.git

# Install dependencies
go mod download

# Start Redis (master, slave, sentinel)
docker compose up -d

# Build app
go build -o app ./cmd/app

# Start scanner
./app -c internal/pkg/config/ethereum.yaml scan

# Start API
./app -c internal/pkg/config/ethereum.yaml api
```

## Benchmark
```shell

## install dependencies
brew install redis
pip install gdown 
go install github.com/google/pprof@latest


## Prepare redis polygon data
gdown "https://drive.google.com/uc?id=1pNA9Ygf_jBnsT7ZQYu_iJQsTz5Yafkes"

## run the redis server with downloaded file named dump.rdb, no need to specify the rdb filename because dump.rdb is default filename
redis-server

## Open new tab, then clone benchmark repo
git clone -b feat/benchmark_bf git@github.com:KyberNetwork/dmm-aggregator-backend.git

# Install dependencies
go mod download

# Run benchmark rate and pprof
go test github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/usecase/benchmark -run "^TestProfileSingleAlgorithmConcurrently$"
go test github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/usecase/benchmark -run "^TestBenchmarkAlgorithm$"

# Wait for 15 minute, the rate result is stored in test_results.csv, pprof result is in pprof/ folder, example to view pprof cpu of bruteforce algo
go tool pprof -http localhost:8080 internal/pkg/usecase/benchmark/pprof/bruteforce_cpu.pprof


```
_Note:_
- You should change `internal/pkg/config/ethereum.yaml` to the config of the network that you want.
- The Redis Sentinel config in docker-compose.yml does not work on localhost at the moment, but you can ignore it, the API can still work normally without Redis Sentinel.

## Supported dexes
### Polygon
  - kyberswap
  - kyberswap-static
  - kyberswapv2
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
  - kyberswapv2
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
  - kyberswapv2
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
  - kyberswapv2
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
  - kyberswapv2
  - beethovenx
  - synapse
  - morpheus
### Cronos
  - kyberswap
  - kyberswap-static
  - kyberswapv2
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
  - kyberswapv2
  - sushiswap
  - curve
  - balancer
  - swapr

### BitTorrent Chain
  - kyberswap
  - kyberswap-static
  - kyberswapv2

### Oasis
  - kyberswap
  - kyberswap-static
  - kyberswapv2
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
  - kyberswapv2

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

### scan
Start scan server

Example: 
```bash
app -c internal/pkg/config/polygon.yaml scan
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
