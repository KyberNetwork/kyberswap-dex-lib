package erc4626

import (
	"math/big"

	"github.com/holiman/uint256"
)

type SwapType uint8

const (
	None SwapType = iota
	Deposit
	Redeem
	Both
)

type (
	Gas struct {
		Deposit uint64 `json:"d,omitempty"`
		Redeem  uint64 `json:"r,omitempty"`
	}

	Extra struct {
		Gas          Gas            `json:"g"`
		SwapTypes    SwapType       `json:"sT,omitempty"`
		MaxDeposit   *uint256.Int   `json:"mD,omitempty"`
		MaxRedeem    *uint256.Int   `json:"mR,omitempty"`
		DepositRates []*uint256.Int `json:"dR,omitempty"`
		RedeemRates  []*uint256.Int `json:"rR,omitempty"`
		TotalAssets  *uint256.Int   `json:"tA,omitempty"`
	}

	StaticExtra struct {
		IsNativeAsset bool `json:"isNativeAsset"`
	}

	Meta struct {
		BlockNumber   uint64 `json:"bN"`
		IsNativeAsset bool   `json:"isNativeAsset,omitempty"`
		IsDeposit     bool   `json:"isDeposit,omitempty"`
	}

	PoolState struct {
		MaxDeposit   *big.Int
		MaxRedeem    *big.Int
		TotalAssets  *big.Int
		TotalSupply  *big.Int
		DepositRates []*big.Int
		RedeemRates  []*big.Int

		blockNumber uint64
	}
)
