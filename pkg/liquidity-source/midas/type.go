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
	MTokenDataFeed common.Address

	// For redemption vaults
	TokenBalance *big.Int

	// For redemption vault with swapper
	MTbillRedemptionVault common.Address

	// For redemption vault with ustb
	UstbRedemption common.Address
}

type VaultState struct {
	TokenRemoved      bool         `json:"tokenRemoved,omitempty"`
	Paused            bool         `json:"paused,omitempty"`
	FnPaused          bool         `json:"fnPaused,omitempty"`
	TokenConfig       *TokenConfig `json:"tokenConfig,omitempty"`
	InstantDailyLimit *uint256.Int `json:"instantDailyLimit,omitempty"`
	DailyLimits       *uint256.Int `json:"dailyLimits,omitempty"`
	InstantFee        *uint256.Int `json:"instantFee,omitempty"`
	MinAmount         *uint256.Int `json:"minAmount,omitempty"`
	MTokenRate        *uint256.Int `json:"mTokenRate,omitempty"`
	TokenRate         *uint256.Int `json:"tokenRate,omitempty"`
	MTokenDataFeed    *uint256.Int `json:"mTokenDataFeed"`

	TokenBalance *uint256.Int `json:"tokenBalance,omitempty"`
}

type RedemptionVaultWithUstbState struct {
	VaultState
	UstbRedemptionState *RedemptionState `json:"ustbRedemption,omitempty"`
}

type RedemptionState struct {
	SuperstateToken          common.Address  `json:"superstateToken"`
	USDC                     common.Address  `json:"usdc"`
	RedemptionFee            *uint256.Int    `json:"redemptionFee"`
	UstbBalance              *uint256.Int    `json:"ustbBalance"`
	ChainlinkPrice           *ChainlinkPrice `json:"chainlinkPrice"`
	ChainLinkFeedPrecision   *uint256.Int    `json:"chainLinkFeedPrecision"`
	SuperstateTokenPrecision *uint256.Int    `json:"superstateTokenPrecision"`
}

type RedemptionVaultWithSwapperState struct {
	VaultState
	MTbillRedemptionVault *RedemptionVaultWithUstbState `json:"mTbillRedemptionVault"`
}

type ChainlinkPrice struct {
	IsBadData bool         `json:"isBadData"`
	UpdatedAt *uint256.Int `json:"updatedAt"`
	Price     *uint256.Int `json:"price"`
}

type StaticExtra struct {
	IsDepositVault bool   `json:"isDepositVault"`
	VaultType      string `json:"type"`
}

type Meta struct {
	BlockNumber     uint64 `json:"blockNumber"`
	DepositVault    string `json:"depositVault,omitempty"`
	RedemptionVault string `json:"redemptionVault,omitempty"`
}

type SwapInfo struct {
	IsDeposit bool `json:"isDeposit"`

	Gas       int64        `json:"-"`
	Fee       *uint256.Int `json:"-"`
	AmountOut *uint256.Int `json:"-"`

	AmountTokenInBase18  *uint256.Int `json:"amountTokenInBase18"`
	AmountMTokenInBase18 *uint256.Int `json:"amountMTokenInBase18"`

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
