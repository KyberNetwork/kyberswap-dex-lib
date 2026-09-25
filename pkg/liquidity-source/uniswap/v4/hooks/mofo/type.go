package mofo

import "math/big"

// poolInfoRaw is the ethrpc.Call decode target for MofoHook's `pools` public mapping getter, field order matching
// hookABI's outputs (uint24 and uint40 decode as *big.Int).
type poolInfoRaw struct {
	FeePips           *big.Int
	LaunchedAt        *big.Int
	CoinIsCurrency1   bool
	BirthSqrtPriceX96 *big.Int
}
