package algebrav1

import (
	_ "embed"
)

//go:embed abis/AlgebraV1Pool.json
var algebraV1PoolJson []byte

//go:embed abis/ERC20.json
var erc20Json []byte
