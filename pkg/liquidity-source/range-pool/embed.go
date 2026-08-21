package rangepool

import _ "embed"

//go:embed abis/RangePool.json
var rangePoolJson []byte

//go:embed abis/RangePoolFactory.json
var rangePoolFactoryJson []byte

//go:embed abis/RangeVault.json
var rangeVaultJson []byte

// rangeRouterJson is the standard Router's querySwapSingleTokenExactIn/Out ABI. It is
// used only by the RPC-gated parity test as the on-chain pricing oracle (the connector
// prices offline and never calls querySwap on the hot path).
//
//go:embed abis/RangeRouter.json
var rangeRouterJson []byte
