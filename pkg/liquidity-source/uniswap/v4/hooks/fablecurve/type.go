package fablecurve

import "math/big"

// poolInfoRaw is the ethrpc.Call decode target for FableHook's `pools` public mapping getter, field order matching
// hookABI's outputs (uint24 and uint40 decode as *big.Int). BirthSqrtPriceX96 and Mesh are decoded to keep the
// getter's full layout; pricing does not read them (see the package doc).
type poolInfoRaw struct {
	FeePips           *big.Int
	LaunchedAt        *big.Int
	CoinIsCurrency1   bool
	BirthSqrtPriceX96 *big.Int
	Mesh              bool
}
