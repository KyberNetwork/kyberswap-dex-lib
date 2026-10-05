package erc4626

import "github.com/holiman/uint256"

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
		MaxDeposit   *uint256.Int   `json:"mD,omitempty"`
		MaxRedeem    *uint256.Int   `json:"mR,omitempty"`
		DepositRates []*uint256.Int `json:"dR,omitempty"`
		RedeemRates  []*uint256.Int `json:"rR,omitempty"`
		TotalAssets  *uint256.Int   `json:"tA,omitempty"`
	}

	StaticExtra struct {
		IsNativeAsset bool `json:"isNativeAsset,omitempty"`
	}

	Meta struct {
		BlockNumber   uint64 `json:"blockNumber"`
		IsNativeAsset bool   `json:"isNativeAsset,omitempty"`
		IsDeposit     bool   `json:"isDeposit,omitempty"`
	}

	PoolState struct {
		MaxDeposit   *uint256.Int
		MaxRedeem    *uint256.Int
		TotalAssets  *uint256.Int
		TotalSupply  *uint256.Int
		DepositRates []*uint256.Int
		RedeemRates  []*uint256.Int

		BlockNumber uint64
	}
)
