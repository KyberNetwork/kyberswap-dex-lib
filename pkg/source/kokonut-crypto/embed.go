package kokonutcrypto

import _ "embed"

//go:embed abi/PoolRegistry.json
var poolRegistryABIBytes []byte

//go:embed abi/CryptoSwap2Pool.json
var cryptoSwap2PoolABIBytes []byte

//go:embed abi/ERC20.json
var erc20ABIBytes []byte
