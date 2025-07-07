package genericarm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type SwapType uint8
type ArmType uint8

const (
	None SwapType = iota
	ZeroToOne
	OneToZero
	Both
)

const (
	Pegged ArmType = iota
	Pricable
)

type Extra struct {
	TradeRate0         *uint256.Int   `json:"r0"`
	TradeRate1         *uint256.Int   `json:"r1"`
	PriceScale         *uint256.Int   `json:"ps"`
	WithdrawsQueued    *uint256.Int   `json:"wq"`
	WithdrawsClaimed   *uint256.Int   `json:"wc"`
	LiquidityAsset     common.Address `json:"la"`
	SwapTypes          SwapType       `json:"swapType"`
	ArmType            ArmType        `json:"armType"`
	HasWithdrawalQueue bool           `json:"hasWithdrawalQueue"`
}

type PoolState struct {
	Token0           common.Address
	Token1           common.Address
	TradeRate0       *big.Int
	TradeRate1       *big.Int
	PriceScale       *big.Int
	WithdrawsQueued  *big.Int
	WithdrawsClaimed *big.Int
	Reserve0         *big.Int
	Reserve1         *big.Int
	LiquidityAsset   common.Address
}
