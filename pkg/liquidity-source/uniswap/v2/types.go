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

	// TokenTaxID is the index of the tax token within the pool's token list (compact for storage),
	// or -1 when the pool has no tax token. Resolve the address via pool.Tokens[TokenTaxID].
	// BuyTax / SellTax are the rates in basis points; nil means no tax in that direction.
	// TaxChecked marks the pool was probed; TaxChecked with TokenTaxID < 0 means it is not a tax pool.
	TokenTaxID int          `json:"tokenTaxId,omitempty"`
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
