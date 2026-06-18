package uniswapv2

import (
	"math/big"

	tokentax "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2/token-tax"
)

type SwapInfo struct {
	// EffectiveAmountIn is the amount the pair actually receives after any sell tax on tokenIn.
	// GrossAmountOut is the amount the pair actually sends out before any buy tax on tokenOut.
	// Reserves move by these pair-side amounts, not by the user-side in/out amounts.
	EffectiveAmountIn *big.Int
	GrossAmountOut    *big.Int
}

type ReserveData struct {
	Reserve0           *big.Int
	Reserve1           *big.Int
	BlockTimestampLast uint32
}

func (d ReserveData) IsZero() bool {
	return d.Reserve0 == nil && d.Reserve1 == nil
}

type Extra struct {
	Fee          uint64 `json:"fee"`
	FeePrecision uint64 `json:"feePrecision"`

	TaxInfo *tokentax.TaxInfo `json:"taxInfo,omitempty"`
}

type PoolMeta struct {
	Extra
	PoolMetaGeneric
}

type PoolMetaGeneric struct {
	ApprovalAddress string `json:"approvalAddress,omitempty"`
	NoFOT           bool   `json:"noFOT,omitempty"`
}
