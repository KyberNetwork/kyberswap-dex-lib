package everlongclamm

import (
	"github.com/holiman/uint256"

	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
)

type StaticExtra struct {
	PoolManager string `json:"pm"`
	ALM         string `json:"alm"`
	Router      string `json:"router"`
	Hook        string `json:"hook"`
	Fee         uint32 `json:"fee"`
	Parameters  string `json:"params"`
	TickSpacing int    `json:"ts"`
}

// Extra is the per-refresh pool state. The embedded ExtraTickU256 carries the CL state
// (slot0 price/tick, active liquidity, and the ALM rung ladder flattened to V3 ticks);
// the pool fees are the ClammHook's directional OUTPUT haircuts in WAD, latched
// pre-trade from ALM.poolFee(zeroForOne) — the in-pool LP fee is overridden to 0.
type Extra struct {
	uniswapv3.ExtraTickU256
	PoolFee0For1Wad *uint256.Int `json:"pf01"`
	PoolFee1For0Wad *uint256.Int `json:"pf10"`
}

// PoolMeta carries everything the executor needs to build the
// ClammSwapRouter.swap(PoolKey,zeroForOne,amountSpecified,sqrtPriceLimitX96,amountOutMin,recipient,deadline)
// call. amountSpecified must be NEGATIVE (exact input); exact output reverts in the hook.
type PoolMeta struct {
	Router      string       `json:"router"`
	PoolManager string       `json:"pm"`
	Hook        string       `json:"hook"`
	Fee         uint32       `json:"fee"`
	Parameters  string       `json:"params"`
	PriceLimit  *uint256.Int `json:"priceLimit"`
}
