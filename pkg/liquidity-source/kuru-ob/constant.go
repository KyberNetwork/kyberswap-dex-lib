package kuruob

import (
	orderbook "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/order-book"
)

const (
	DexType = "kuru-ob"

	maxPriceLevels = 32
)

var (
	defaultGas = orderbook.Gas{Base: 221703, Level: 84155}
)
