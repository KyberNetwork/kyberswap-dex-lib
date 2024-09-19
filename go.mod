module github.com/KyberNetwork/kyberswap-dex-lib

go 1.22.7

require (
	github.com/KyberNetwork/blockchain-toolkit v0.8.1
	github.com/KyberNetwork/elastic-go-sdk/v2 v2.0.2
	github.com/KyberNetwork/ethrpc v0.7.3-0.20240919101855-8d4012c8c2ba
	github.com/KyberNetwork/iZiSwap-SDK-go v1.1.0
	github.com/KyberNetwork/int256 v0.1.4
	github.com/KyberNetwork/logger v0.2.0
	github.com/KyberNetwork/msgpack/v5 v5.4.2
	github.com/KyberNetwork/pancake-v3-sdk v0.2.0
	github.com/KyberNetwork/uniswapv3-sdk-uint256 v0.5.0
	github.com/daoleno/uniswap-sdk-core v0.1.7
	github.com/daoleno/uniswapv3-sdk v0.4.0
	github.com/davecgh/go-spew v1.1.1
	github.com/deckarep/golang-set/v2 v2.6.0
	github.com/dgraph-io/ristretto v0.1.1
	github.com/ethereum/go-ethereum v1.13.9
	github.com/go-resty/resty/v2 v2.10.0
	github.com/goccy/go-json v0.10.2
	github.com/golang/mock v1.6.0
	github.com/google/go-cmp v0.6.0
	github.com/holiman/uint256 v1.3.1
	github.com/machinebox/graphql v0.2.2
	github.com/mitchellh/mapstructure v1.4.1
	github.com/orcaman/concurrent-map v1.0.0
	github.com/pkg/errors v0.9.1
	github.com/samber/lo v1.38.1
	github.com/sirupsen/logrus v1.9.3
	github.com/sourcegraph/conc v0.3.0
	github.com/stretchr/testify v1.9.0
	golang.org/x/sync v0.8.0
)

require github.com/klauspost/compress v1.17.8

require (
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/bits-and-blooms/bitset v1.14.3 // indirect
	github.com/btcsuite/btcd/btcec/v2 v2.3.4 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/consensys/bavard v0.1.15 // indirect
	github.com/consensys/gnark-crypto v0.12.1 // indirect
	github.com/crate-crypto/go-kzg-4844 v0.7.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.3.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/ethereum/c-kzg-4844 v1.0.3 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/golang/glog v1.1.2 // indirect
	github.com/golang/snappy v0.0.5-0.20220116011046-fa5810519dcb // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/huin/goupnp v1.3.0 // indirect
	github.com/jackpal/go-nat-pmp v1.0.2 // indirect
	github.com/matryer/is v1.4.1 // indirect
	github.com/mmcloughlin/addchain v0.4.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/shirou/gopsutil v3.21.11+incompatible // indirect
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/supranational/blst v0.3.13 // indirect
	github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7 // indirect
	github.com/tklauser/go-sysconf v0.3.14 // indirect
	github.com/tklauser/numcpus v0.8.0 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/crypto v0.27.0 // indirect
	golang.org/x/exp v0.0.0-20240909161429-701f63a606c0 // indirect
	golang.org/x/net v0.29.0 // indirect
	golang.org/x/sys v0.25.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	rsc.io/tmplfunc v0.0.3 // indirect
)

replace (
	github.com/daoleno/uniswap-sdk-core v0.1.5 => github.com/KyberNetwork/uniswap-sdk-core v0.1.5
	github.com/daoleno/uniswapv3-sdk v0.4.0 => github.com/KyberNetwork/uniswapv3-sdk v0.5.0
)
