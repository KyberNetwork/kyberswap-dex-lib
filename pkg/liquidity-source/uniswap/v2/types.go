package uniswapv2

import (
	"math/big"

	"github.com/holiman/uint256"
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

	// TaxToken is the address of the token that charges transfer tax (lowercase).
	// BuyTax / SellTax are the rates in basis points; nil means no tax in that direction.
	// TaxChecked marks the pool was probed; TaxChecked with an empty TaxToken means it is not a tax pool.
	TaxToken   string       `json:"taxToken,omitempty"`
	BuyTax     *uint256.Int `json:"buyTax,omitempty"`
	SellTax    *uint256.Int `json:"sellTax,omitempty"`
	TaxChecked bool         `json:"taxChecked,omitempty"`
}

type PoolMeta struct {
	Extra
	PoolMetaGeneric
}

type PoolMetaGeneric struct {
	ApprovalAddress string `json:"approvalAddress,omitempty"`
	NoFOT           bool   `json:"noFOT,omitempty"`
}
