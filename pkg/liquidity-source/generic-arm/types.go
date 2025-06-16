package genericarm

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type SwapType uint8

const (
	None SwapType = iota
	ZeroToOne
	OneToZero
	Both
)

type Extra struct {
	TradeRate0       *uint256.Int   `json:"r0"`
	TradeRate1       *uint256.Int   `json:"r1"`
	PriceScale       *uint256.Int   `json:"ps"`
	WithdrawsQueued  *uint256.Int   `json:"wq"`
	WithdrawsClaimed *uint256.Int   `json:"wc"`
	LiquidityAsset   common.Address `json:"la"`
	SwapTypes        SwapType       `json:"swapTypes"`
}
