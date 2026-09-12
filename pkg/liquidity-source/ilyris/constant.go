// Package ilyris is the KyberSwap aggregator adapter for Ilyris, a discrete-bin AMM
// (Liquidity Book / DLMM family) on Robinhood Chain.
package ilyris

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

const (
	// DexType is the exchange identifier, registered in valueobject.Exchange.
	DexType = valueobject.ExchangeIlyris

	// Fee arithmetic, mirroring src/BinPool.sol exactly.
	//
	// FEE_PRECISION is 1e9 and the swap fee is charged on the INPUT before any bin is
	// touched: netIn = amountIn * (FEE_PRECISION - rate) / FEE_PRECISION. Charging it
	// per-bin instead would compound it across a multi-bin traversal and quote low.
	feePrecision = 1_000_000_000
	bps          = 10_000
)
