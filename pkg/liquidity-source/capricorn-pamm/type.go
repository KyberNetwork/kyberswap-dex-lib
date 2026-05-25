package capricornpamm

import (
	"github.com/holiman/uint256"
)

type StaticExtra struct {
	Factory  string `json:"factory"`
	OracleId string `json:"oracleId"` // bytes32 key used by OracleRegistry
}

// Extra is rewritten end-to-end on each refresh.
type Extra struct {
	FeeBps uint64 `json:"feeBps"` // observability only — quoteExactIn is post-fee
	Paused bool   `json:"paused"`

	// Unquoteable is set when the on-chain swap path is not executable:
	// oracleRegistry.paused, getPrice reverted, push-age within safety
	// buffer of the StalePrice gate, or both quoteExactIn directions
	// reverted. CalcAmountOut returns ErrPoolUnavailable.
	Unquoteable bool `json:"unquoteable,omitempty"`

	// PublishTime / MaxPushPriceAge are persisted for quote-time staleness
	// re-check. MaxPushPriceAge is the live registry value (admin-rotatable).
	PublishTime     uint64 `json:"publishTime,omitempty"`
	MaxPushPriceAge uint64 `json:"maxPushPriceAge,omitempty"`

	// Ladder0/Ladder1 are post-fee (amountIn, amountOut) probes from
	// quoteExactIn, sorted ascending. The simulator linearly interpolates
	// between points. Largest grid point is bounded by maxAmountIn.
	Ladder0 []LadderPoint `json:"ladder0"`
	Ladder1 []LadderPoint `json:"ladder1"`
}

type LadderPoint struct {
	AmountIn  *uint256.Int `json:"in"`
	AmountOut *uint256.Int `json:"out"`
}

type SwapInfo struct {
	Reserve0 *uint256.Int `json:"r0"`
	Reserve1 *uint256.Int `json:"r1"`
}

type MetaInfo struct {
	BlockNumber uint64 `json:"bN"`
}
