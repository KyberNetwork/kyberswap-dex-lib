module github.com/KyberNetwork/kyberswap-dex-lib

go 1.20

require (
	github.com/KyberNetwork/elastic-go-sdk/v2 v2.0.2
	github.com/KyberNetwork/ethrpc v0.2.0
	github.com/KyberNetwork/logger v0.1.0
	github.com/daoleno/uniswap-sdk-core v0.1.5
	github.com/daoleno/uniswapv3-sdk v0.4.0
	github.com/dgraph-io/ristretto v0.1.1
	github.com/ethereum/go-ethereum v1.11.6
	github.com/go-resty/resty/v2 v2.7.0
	github.com/golang/mock v1.6.0
	github.com/machinebox/graphql v0.2.2
	github.com/orcaman/concurrent-map v1.0.0
	github.com/pkg/errors v0.9.1
	github.com/redis/go-redis/v9 v9.0.4
	github.com/samber/lo v1.38.1
	github.com/sourcegraph/conc v0.3.0
	github.com/stretchr/testify v1.8.3
	golang.org/x/sync v0.2.0
)

require (
	github.com/btcsuite/btcd/btcec/v2 v2.2.0 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/daoleno/uniswap-sdk-core v0.1.5 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/deckarep/golang-set/v2 v2.1.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/holiman/uint256 v1.2.2-0.20230321075855-87b91420868c // indirect
	github.com/matryer/is v1.4.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/shirou/gopsutil v3.21.11+incompatible // indirect
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/sirupsen/logrus v1.9.0 // indirect
	github.com/tklauser/go-sysconf v0.3.10 // indirect
	github.com/tklauser/numcpus v0.5.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.9.0 // indirect
	go.uber.org/zap v1.24.0 // indirect
	golang.org/x/crypto v0.1.0 // indirect
	golang.org/x/exp v0.0.0-20230206171751-46f607a40771 // indirect
	golang.org/x/net v0.8.0 // indirect
	golang.org/x/sys v0.6.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/natefinch/npipe.v2 v2.0.0-20160621034901-c1b8fa8bdcce // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/daoleno/uniswap-sdk-core v0.1.5 => github.com/KyberNetwork/uniswap-sdk-core v0.1.5
	github.com/daoleno/uniswapv3-sdk v0.4.0 => github.com/KyberNetwork/uniswapv3-sdk v0.4.0
)
