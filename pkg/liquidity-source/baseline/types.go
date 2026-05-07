package baseline

import (
	"math/big"

	"github.com/holiman/uint256"
)

type Metadata struct {
	Offset int `json:"offset"`
}

// Extra is serialized into entity.Pool.Extra
type Extra struct {
	RelayAddress string `json:"r"`

	QuoteState *QuoteState `json:"q,omitempty"`
}

type SwapInfo struct {
	RelayAddress string      `json:"relayAddress"`
	BToken       string      `json:"bToken"`
	IsBuy        bool        `json:"isBuy"`
	AmountOut    string      `json:"amountOut,omitempty"`
	State        *QuoteState `json:"-"`
	ReserveDelta string      `json:"reserveDelta,omitempty"`
	Fee          string      `json:"fee,omitempty"`
}

type CurveParams struct {
	BLV           *uint256.Int `json:"blv,omitempty"`
	Circ          *uint256.Int `json:"c,omitempty"`
	Supply        *uint256.Int `json:"x,omitempty"`
	SwapFee       *uint256.Int `json:"sf,omitempty"`
	Reserves      *uint256.Int `json:"y,omitempty"`
	TotalSupply   *uint256.Int `json:"ts,omitempty"`
	ConvexityExp  *uint256.Int `json:"n,omitempty"`
	LastInvariant *uint256.Int `json:"k,omitempty"`
}

type QuoteState struct {
	SnapshotCurveParams     CurveParams  `json:"s"`
	QuoteBlockBuyDeltaCirc  *uint256.Int `json:"bb,omitempty"`
	QuoteBlockSellDeltaCirc *uint256.Int `json:"bs,omitempty"`
	TotalSupply             *uint256.Int `json:"ts,omitempty"`
	TotalBTokens            *uint256.Int `json:"tb,omitempty"`
	TotalReserves           *uint256.Int `json:"tr,omitempty"`
	ReserveDecimals         uint8        `json:"rd,omitempty"`
	LiquidityFeePct         *uint256.Int `json:"lf,omitempty"`
	PendingSurplus          *uint256.Int `json:"ps,omitempty"`
	SettlePendingSurplus    bool         `json:"sps,omitempty"`
	MaxSellDelta            *uint256.Int `json:"ms,omitempty"`
	SnapshotActivePrice     *uint256.Int `json:"ap,omitempty"`
}

type quoteResult struct {
	AmountOut     *uint256.Int
	Fee           *uint256.Int
	AccountingFee *uint256.Int
	ReserveDelta  *big.Int
	State         *QuoteState
}

type PoolMeta struct {
	Pool        string `json:"p"`
	BlockNumber uint64 `json:"bN"`
	IsBuyBase   bool   `json:"isBuyBase,omitempty"`
}
