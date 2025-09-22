package midas

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
)

type redemptionVaultType uint8
type depositVaultType uint8

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
	TokenBalance *big.Int
}

type VaultState struct {
	TokenRemoved      bool         `json:"tokenRemoved"`
	Paused            bool         `json:"paused"`
	FnPaused          bool         `json:"fnPaused"`
	TokenConfig       *TokenConfig `json:"tokenConfig"`
	InstantDailyLimit *uint256.Int `json:"instantDailyLimit"`
	DailyLimits       *uint256.Int `json:"dailyLimits"`
	InstantFee        *uint256.Int `json:"instantFee"`
	MinAmount         *uint256.Int `json:"minAmount"`
	MTokenRate        *uint256.Int `json:"mTokenRate"`
	TokenRate         *uint256.Int `json:"tokenRate"`

	TokenBalance *uint256.Int `json:"tokenBalance"`
}

type RedemptionVaultWithUstbState struct {
	VaultState
	SuperstateToken          common.Address  `json:"superstateToken"`
	USDC                     common.Address  `json:"usdc"`
	RedemptionFee            *uint256.Int    `json:"redemptionFee"`
	USTBBalance              *uint256.Int    `json:"ustbBalance"`
	ChainlinkPrice           *ChainlinkPrice `json:"chainlinkPrice"`
	ChainLinkFeedPrecision   *uint256.Int    `json:"chainLinkFeedPrecision"`
	SuperstateTokenPrecision *uint256.Int    `json:"superstateTokenPrecision"`
}

type RedemptionVaultWithSwapperState struct {
	VaultState
	MTbillRedemptionVault *RedemptionVaultWithUstbState `json:"mTbillRedemptionVault"`
}

type ChainlinkPrice struct {
	IsBadData bool
	UpdatedAt *uint256.Int
	Price     *uint256.Int
}

type Extra[T VaultState | RedemptionVaultWithSwapperState | RedemptionVaultWithUstbState] struct {
	DepositVault    *VaultState `json:"depositVault,omitempty"`
	RedemptionVault *T          `json:"redemptionVault,omitempty"`
}

type StaticExtra struct {
	DataFeed       string `json:"dataFeed"`
	MTokenDataFeed string `json:"mTokenDataFeed"`

	CanDeposit bool `json:"canDeposit"`
	CanRedeem  bool `json:"canRedeem"`

	DepositVaultType depositVaultType `json:"depositVaultType"`
	DepositVault     string           `json:"depositVault"`

	RedemptionVaultType redemptionVaultType `json:"redemptionVaultType"`
	RedemptionVault     string              `json:"redemptionVault"`
}

type Meta struct {
	BlockNumber     uint64 `json:"blockNumber"`
	DepositVault    string `json:"depositVault,omitempty"`
	RedemptionVault string `json:"redemptionVault,omitempty"`
}

type SwapInfo struct {
	IsDeposit          bool         `json:"isDeposit"`
	SwapAmountInBase18 *uint256.Int `json:"swapAmountInBase18"`

	Gas       int64        `json:"-"`
	Fee       *uint256.Int `json:"-"`
	AmountOut *uint256.Int `json:"-"`

	mTbillAmountInBase18 *uint256.Int
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
	if v == nil {
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
		MTokenRate:   uint256.MustFromBig(mTokenRate),
		TokenRate:    uint256.MustFromBig(tokenRate),
		TokenBalance: uint256.MustFromBig(v.TokenBalance),
	}
}
