package integral

import (
	_ "embed"
)

//go:embed abis/AlgebraV1Pool.json
var algebraV1PoolJson []byte

//go:embed abis/AlgebraV1PoolDirectionalFee.json
var algebraV1DirFeePoolJson []byte

//go:embed abis/AlgebraV1DataStorageOperator.json
var algebraV1DataStorageOperatorJson []byte

//go:embed abis/AlgebraV1DirFeeDataStorageOperator.json
var algebraV1DirFeeDataStorageOperatorJson []byte

//go:embed abis/ERC20.json
var erc20Json []byte

//go:embed abis/TickLens.json
var ticklensJson []byte

//go:embed abi_temp/AlgebraPool.json
var algebraIntegralPoolJson []byte

//go:embed abi_temp/AlgebraPlugin.json
var algebraPluginJson []byte

//go:embed abi_temp/AlgebraBasePluginV1.json
var algebraBasePluginV1Json []byte
