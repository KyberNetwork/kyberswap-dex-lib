package infinitypools

import _ "embed"

//go:embed abi/ERC20.json
var erc20ABIJson []byte

//go:embed abi/InfinityPool.json
var infinityPoolABIJson []byte
