package curve

import _ "embed"

//go:embed abi/AddressProvider.json
var addressProviderABIBytes []byte

//go:embed abi/MainRegistry.json
var mainRegistryABIBytes []byte

//go:embed abi/MetaPoolFactory.json
var metaPoolFactoryABIBybtes []byte

//go:embed abi/CryptoFactory.json
var cryptoFactoryABIBytes []byte

//go:embed abi/CryptoRegistry.json
var cryptoRegistryABIBytes []byte

//go:embed abi/MetaPool.json
var metaABIBytes []byte

//go:embed abi/Aave.json
var aaveABIBytes []byte

//go:embed abi/PlainOraclePool.json
var plainOraclePoolABIBytes []byte

//go:embed abi/BasePool.json
var basePoolABIBytes []byte

//go:embed abi/Two.json
var twoABIBytes []byte

//go:embed abi/Tricrypto.json
var tricryptoABIBytes []byte

//go:embed abi/Oracle.json
var oracleABIBytes []byte

//go:embed abi/Compound.json
var compoundABIBytes []byte

//go:embed abi/ERC20.json
var erc20ABIBytes []byte

//go:embed pools/arbitrum.json
var arbitrumPoolsBytes []byte

//go:embed pools/avalanche.json
var avalanchePoolsBytes []byte

//go:embed pools/ethereum.json
var ethereumPoolsBytes []byte

//go:embed pools/fantom.json
var fantomPoolsBytes []byte

//go:embed pools/optimism.json
var optimismPoolsBytes []byte

//go:embed pools/polygon.json
var polygonPoolsBytes []byte

//go:embed pools/base.json
var basePoolsBytes []byte

// Ellipsis pool bytes

//go:embed pools/ellipsis/bsc.json
var ellipsisBscPoolsBytes []byte

// Pancake-stable pool bytes

//go:embed pools/pancake-stable/bsc.json
var pancakeStablePoolsBytes []byte

var bytesByPath = map[string][]byte{
	"pools/arbitrum.json":  arbitrumPoolsBytes,
	"pools/avalanche.json": avalanchePoolsBytes,
	"pools/ethereum.json":  ethereumPoolsBytes,
	"pools/fantom.json":    fantomPoolsBytes,
	"pools/optimism.json":  optimismPoolsBytes,
	"pools/polygon.json":   polygonPoolsBytes,
	"pools/base.json":      basePoolsBytes,

	"pools/ellipsis/bsc.json":       ellipsisBscPoolsBytes,
	"pools/pancake-stable/bsc.json": pancakeStablePoolsBytes,
}
