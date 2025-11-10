package algebrav1

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

//go:embed abis/AlgebraV1Factory.json
var algebraV1FactoryJson []byte

//go:embed abis/TickLens.json
var ticklensJson []byte
