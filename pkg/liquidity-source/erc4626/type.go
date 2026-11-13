package erc4626

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
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
		// MintHeadroom is maxSupply - totalSupply, the shares still mintable; nil = no cap tracked.
		MintHeadroom *uint256.Int `json:"mH,omitempty"`
		// MaxAge is VaultCfg.MaxAge, 0 = no staleness check.
		MaxAge int64 `json:"mA,omitempty"`
	}

	StaticExtra struct {
		IsNativeAsset bool `json:"isNativeAsset,omitempty"`
		// BalanceKey is the hex storage key of balanceOf[AddrDummy] found by the maxRedeem cap probe, see probe.go.
		BalanceKey string `json:"bK,omitempty"`
		// NoBalanceKey marks a vault probed without finding the key; it is not probed again.
		NoBalanceKey bool `json:"nbK,omitempty"`
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
		// Resolved by ResolveLimits.
		Paused       bool
		MintHeadroom *big.Int
		DepositRates []*big.Int
		RedeemRates  []*big.Int

		BlockNumber uint64

		// Cap probe, see probe.go. BalanceOf is the balanceOf(AddrDummy) read-back, nil without a probe.
		BalanceOf *big.Int
		// CapKnown: the marker read back, so MaxRedeem is the real cap. Slot is the probe outcome to persist
		// when SlotChanged.
		CapKnown, SlotChanged bool
		Slot                  BalanceSlot
	}
)

// BalanceSlot is the cached cap-probe outcome.
func (s StaticExtra) BalanceSlot() BalanceSlot {
	return BalanceSlot{Key: common.HexToHash(s.BalanceKey), None: s.NoBalanceKey}
}
