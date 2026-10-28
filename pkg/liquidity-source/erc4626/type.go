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
		MaxDeposit   *uint256.Int   `json:"mD,omitempty"`
		MaxRedeem    *uint256.Int   `json:"mR,omitempty"`
		MinDeposit   *uint256.Int   `json:"minD,omitempty"`
		DepositRates []*uint256.Int `json:"dR,omitempty"`
		RedeemRates  []*uint256.Int `json:"rR,omitempty"`
		TotalAssets  *uint256.Int   `json:"tA,omitempty"`
		// DepositsPaused is set only for vaults with DepositsPausedMethod.
		DepositsPaused bool `json:"p,omitempty"`
		// RedeemsPaused is set only for vaults with RedeemsPausedMethod.
		RedeemsPaused bool `json:"rP,omitempty"`
		// MintHeadroom is maxSupply - totalSupply, the shares still mintable; nil = no cap tracked.
		MintHeadroom *uint256.Int `json:"mH,omitempty"`
		// MaxAge is VaultCfg.MaxAge, 0 = no staleness check (deposits and redeems).
		MaxAge int64 `json:"mA,omitempty"`
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
		MaxDeposit  *big.Int
		MaxRedeem   *big.Int
		MinDeposit  *big.Int
		TotalAssets *big.Int
		TotalSupply *big.Int
		// Raw limit reads, see AddLimitCalls. Paused is 0/1, nil if the read failed or is not configured.
		RawPaused, MaxSupply *big.Int
		// Morpho VaultV2 permissions of AddrDummy, 0/1, nil if the vault has no such view.
		CanReceiveShares, CanSendShares, CanSendAssets *big.Int
		// Raw redeem pause read, see AddRedeemLimitCalls. Same encoding as RawPaused.
		RawRedeemsPaused *big.Int
		// Resolved by ResolveLimits.
		Paused, RedeemsPaused bool
		MintHeadroom          *big.Int
		DepositRates          []*big.Int
		RedeemRates           []*big.Int

		BlockNumber uint64
	}
)
