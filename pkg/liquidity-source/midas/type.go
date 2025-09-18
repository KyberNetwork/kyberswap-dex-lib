package midas

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
)

type TokenConfig struct {
	DataFeed  common.Address `json:"dataFeed"`
	Fee       *uint256.Int   `json:"fee"`
	Allowance *uint256.Int   `json:"allowance"`
	Stable    bool           `json:"stable"`
}

type VaultStateResponse struct {
	PaymentTokens     []common.Address
	Paused            bool
	FnPaused          bool
	InstantDailyLimit *big.Int
	DailyLimits       *big.Int
	InstantFee        *big.Int
	MinAmount         *big.Int
	TokenConfig       struct {
		DataFeed  common.Address
		Fee       *big.Int
		Allowance *big.Int
		Stable    bool
	}

	TokenBalance *big.Int // for redemption vault with swapper
}

type VaultState struct {
	TokenRemoved      bool         `json:"tokenRemoved"`
	Paused            bool         `json:"paused"`
	FnPaused          bool         `json:"fnPaused"`
	InstantDailyLimit *uint256.Int `json:"instantDailyLimit"`
	DailyLimits       *uint256.Int `json:"dailyLimits"`
	InstantFee        *uint256.Int `json:"instantFee"`
	MinAmount         *uint256.Int `json:"minAmount"`
	TokenConfig       *TokenConfig `json:"tokenConfig"`
	MTokenRate        *uint256.Int `json:"mTokenRate"`
	TokenRate         *uint256.Int `json:"tokenRate"`
}

type RedemptionVaultState struct {
	VaultState
	TokenBalance *uint256.Int `json:"tokenBalance"`
	MTbillRate   *uint256.Int `json:"mTbillRate,omitempty"`
}

type Extra struct {
	DepositVault    *VaultState           `json:"depositVault,omitempty"`
	RedemptionVault *RedemptionVaultState `json:"redemptionVault,omitempty"`

	MTokenRate *uint256.Int `json:"mTokenRate"`
	TokenRate  *uint256.Int `json:"tokenRate"`
}

type redemptionVaultType uint8
type depositVaultType uint8

type StaticExtra struct {
	DataFeed       common.Address `json:"dataFeed"`
	MTokenDataFeed common.Address `json:"mTokenDataFeed"`
	CanDeposit     bool           `json:"canDeposit"`
	CanRedeem      bool           `json:"canRedeem"`

	DepositVault        common.Address       `json:"depositVault,omitempty"`
	RedemptionVault     common.Address       `json:"redemptionVault,omitempty"`
	DepositVaultType    *depositVaultType    `json:"depositVaultType,omitempty"`
	RedemptionVaultType *redemptionVaultType `json:"redemptionVaultType,omitempty"`

	MTbillRedemptionVault common.Address `json:"mTbillRedemptionVault,omitempty"`
	LiquidityProvider     common.Address `json:"liquidityProvider,omitempty"`
}

type Meta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type SwapInfo struct {
	IsDeposit          bool            `json:"isDeposit"`
	DepositVault       *common.Address `json:"depositVault,omitempty"`
	RedemptionVault    *common.Address `json:"redemptionVault,omitempty"`
	SwapAmountInBase18 *uint256.Int    `json:"swapAmountInBase18"`

	Gas       int64        `json:"-"`
	Fee       *uint256.Int `json:"-"`
	AmountOut *uint256.Int `json:"-"`
}

type IDepositVault interface {
	DepositInstant(amountTokenIn *uint256.Int) (*SwapInfo, error)
	UpdateState(swapInfo *SwapInfo) error
}

type IRedemptionVault interface {
	RedeemInstant(amountMTokenIn *uint256.Int) (*SwapInfo, error)
	UpdateState(swapInfo *SwapInfo) error
}

func (v *VaultStateResponse) ToVaultState(token string, mTokenRate, tokenRate *big.Int) *VaultState {
	if v == nil || v == (*VaultStateResponse)(nil) {
		return nil
	}

	return &VaultState{
		TokenRemoved:      !lo.Contains(v.PaymentTokens, common.HexToAddress(token)),
		Paused:            v.Paused,
		FnPaused:          v.FnPaused,
		InstantDailyLimit: uint256.MustFromBig(v.InstantDailyLimit),
		DailyLimits:       uint256.MustFromBig(v.DailyLimits),
		InstantFee:        uint256.MustFromBig(v.InstantFee),
		MinAmount:         uint256.MustFromBig(v.MinAmount),
		TokenConfig: &TokenConfig{
			DataFeed:  v.TokenConfig.DataFeed,
			Fee:       uint256.MustFromBig(v.TokenConfig.Fee),
			Allowance: uint256.MustFromBig(v.TokenConfig.Allowance),
			Stable:    v.TokenConfig.Stable,
		},
		MTokenRate: uint256.MustFromBig(mTokenRate),
		TokenRate:  uint256.MustFromBig(tokenRate),
	}
}

func (v *VaultStateResponse) ToRedemptionVaultState(token string, mTokenRate, tokenRate *big.Int) *RedemptionVaultState {
	vaultState := v.ToVaultState(token, mTokenRate, tokenRate)
	if vaultState == nil {
		return nil
	}

	return &RedemptionVaultState{
		VaultState:   *vaultState,
		TokenBalance: uint256.MustFromBig(v.TokenBalance),
	}
}
