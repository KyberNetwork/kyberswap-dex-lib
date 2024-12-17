package virtualfun

import (
	"math/big"

	"github.com/holiman/uint256"
)

type StaticExtra struct {
	BondingAddress string `json:"bondingAddress"`
}

type Extra struct {
	GradThreshold *big.Int `json:"gradThreshold"`
	KLast         *big.Int `json:"kLast"`
	BuyTax        *big.Int `json:"buyTax"`
	SellTax       *big.Int `json:"sellTax"`
	ReserveA      *big.Int `json:"reserveA"`
	ReserveB      *big.Int `json:"reserveB"`
}

type SwapInfo struct {
	IsBuy          bool         `json:"isBuy"`
	BondingAddress string       `json:"bondingAddress"`
	TokenAddress   string       `json:"tokenAddress"`
	NewReserveA    *uint256.Int `json:"-"`
	NewReserveB    *uint256.Int `json:"-"`
	NewBalanceA    *uint256.Int `json:"-"`
	NewBalanceB    *uint256.Int `json:"-"`
}

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}
