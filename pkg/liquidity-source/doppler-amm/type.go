package doppleramm

import (
	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
)

// StaticExtra carries the DopplerAMM market identity. A market is a
// configuration hash, not a pool contract, so every read is keyed by MarketID;
// TokenX/TokenY fix which reserve is index 0.
type StaticExtra struct {
	MarketID string `json:"marketId"`
	TokenX   string `json:"tokenX"`
	TokenY   string `json:"tokenY"`
	AMM      string `json:"amm"`
	Quoter   string `json:"quoter"`
	// Launch Markets only. Module is the launch module; Levels are the ascending
	// level boundaries of its multicurve (V3 ticks at spacing 1), immutable.
	Module string  `json:"module,omitempty"`
	Levels []int32 `json:"levels,omitempty"`
}

// Extra holds the fee rates resolved at refresh time. The module LP fee is not
// readable from state (it lives in the component's config pointer), so the
// tracker recovers it from one quoter probe; the protocol base fee comes from
// protocolFeeForMarket.
type Extra struct {
	// TokenX is repeated here so the encoder can derive inputIsX from the
	// route summary, which carries Extra but not StaticExtra.
	TokenX     string `json:"tokenX"`
	BaseFeeBps uint16 `json:"baseFeeBps"`
	LpFeeBps   uint16 `json:"lpFeeBps"`

	// Launch Markets only: the module's live state. Nets[i] is the liquidity
	// change when crossing StaticExtra.Levels[i] upward. SwapFeePips replaces
	// LpFeeBps as the module fee.
	SwapFeePips  uint32        `json:"swapFeePips,omitempty"`
	SqrtPriceX96 *uint256.Int  `json:"sqrtPriceX96,omitempty"`
	Liquidity    *uint256.Int  `json:"liquidity,omitempty"`
	Nets         []*int256.Int `json:"nets,omitempty"`
}
