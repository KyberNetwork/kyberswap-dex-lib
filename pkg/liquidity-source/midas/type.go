package midas

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
)

type VaultType string

type TokenConfig struct {
	Fee       *uint256.Int `json:"fee"`
	Allowance *uint256.Int `json:"allowance"`
	Stable    bool         `json:"stable"`
}

type ChainlinkPriceRpcResult struct {
	IsBadData bool
	UpdatedAt *big.Int
	Price     *big.Int
}

type RedemptionRpcResult struct {
	Usdc                     common.Address
	RedemptionFee            *big.Int
	UstbBalance              *big.Int
	ChainlinkPrice           ChainlinkPriceRpcResult
	ChainLinkFeedPrecision   *big.Int
	SuperstateTokenPrecision *big.Int
}

type TokenConfigRpcResult struct {
	DataFeed  common.Address
	Fee       *big.Int
	Allowance *big.Int
	Stable    bool
}

type VaultStateRpcResult struct {
	MToken string

	PaymentTokens        []common.Address
	Paused               bool
	FnPaused             bool
	InstantDailyLimit    *big.Int
	DailyLimits          *big.Int
	InstantFee           *big.Int
	MinAmount            *big.Int
	TokensConfig         []TokenConfigRpcResult
	MTokenDataFeed       common.Address
	WaivedFeeRestriction bool
	MTokenDecimals       uint8

	MTokenRate *big.Int
	TokenRates []*big.Int

	// For deposit vault
	MinMTokenAmountForFirstDeposit *big.Int
	TotalMinted                    *big.Int
	MaxSupplyCap                   *big.Int
	MTokenTotalSupply              *big.Int

	// For redemption vault
	TokenBalances []*big.Int

	// For redemption vault with ustb
	Redemption RedemptionRpcResult

	// For redemption vault with swapper
	MToken2Balance        *big.Int
	SwapperVaultType      VaultType
	MTbillRedemptionVault *VaultStateRpcResult
}

type VaultState struct {
	MToken string `json:"mToken"`

	PaymentTokens        []string       `json:"paymentTokens"`
	Paused               bool           `json:"paused"`
	FnPaused             bool           `json:"fnPaused"`
	TokenConfigs         []TokenConfig  `json:"tokensConfig"`
	InstantDailyLimit    *uint256.Int   `json:"instantDailyLimit"`
	DailyLimits          *uint256.Int   `json:"dailyLimits"`
	InstantFee           *uint256.Int   `json:"instantFee"`
	MinAmount            *uint256.Int   `json:"minAmount"`
	MTokenRate           *uint256.Int   `json:"mTokenRate"`
	TokenRates           []*uint256.Int `json:"tokenRates"`
	WaivedFeeRestriction bool           `json:"waivedFeeRestriction"`
	MTokenDecimals       uint8          `json:"mTokenDecimals"`

	// For deposit vault
	MinMTokenAmountForFirstDeposit *uint256.Int `json:"minMTokenAmountForFirstDeposit,omitempty"`
	TotalMinted                    *uint256.Int `json:"totalMinted,omitempty"`
	MaxSupplyCap                   *uint256.Int `json:"maxSupplyCap,omitempty"`
	MTokenTotalSupply              *uint256.Int `json:"mTokenTotalSupply,omitempty"`

	// For redemption vault
	TokenBalances []*uint256.Int `json:"tokenBalances,omitempty"`

	// For redemption vault with ustb
	Redemption *RedemptionState `json:"redemption,omitempty"`

	// For redemption vault with swapper
	MToken1Balance        *uint256.Int `json:"mToken1Balance,omitempty"`
	MToken2Balance        *uint256.Int `json:"mToken2Balance,omitempty"`
	SwapperVaultType      VaultType    `json:"swapperVaultType,omitempty"`
	MTbillRedemptionVault *VaultState  `json:"mTbillRedemptionVault,omitempty"`
}

type RedemptionVaultWithUstbState struct {
	VaultState
	UstbRedemptionState *RedemptionState `json:"ustbRedemption,omitempty"`
}

type RedemptionState struct {
	Usdc                     common.Address  `json:"usdc"`
	RedemptionFee            *uint256.Int    `json:"redemptionFee"`
	UstbBalance              *uint256.Int    `json:"ustbBalance"`
	ChainlinkPrice           *ChainlinkPrice `json:"chainlinkPrice"`
	ChainLinkFeedPrecision   *uint256.Int    `json:"chainLinkFeedPrecision"`
	SuperstateTokenPrecision *uint256.Int    `json:"superstateTokenPrecision"`
}

type ChainlinkPrice struct {
	IsBadData bool         `json:"isBadData"`
	UpdatedAt *uint256.Int `json:"updatedAt"`
	Price     *uint256.Int `json:"price"`
}

type StaticExtra struct {
	IsDv      bool      `json:"isDv"`
	VaultType VaultType `json:"type"`
}

type Meta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type SwapInfo struct {
	IsDeposit            bool         `json:"isDeposit"`
	AmountTokenInBase18  *uint256.Int `json:"amountTokenInBase18"`
	AmountMTokenInBase18 *uint256.Int `json:"amountMTokenInBase18"`

	gas       int64
	fee       *uint256.Int
	amountOut *uint256.Int

	mToken1AmountInBase18 *uint256.Int
	mToken2AmountInBase18 *uint256.Int
}

type IDepositVault interface {
	DepositInstant(amountTokenIn *uint256.Int, tokenIn string) (*SwapInfo, error)
	UpdateState(swapInfo *SwapInfo, tokenIn string)
	CloneState() any
	GetMToken() string
}

type IRedemptionVault interface {
	RedeemInstant(amountMTokenIn *uint256.Int, tokenOut string) (*SwapInfo, error)
	UpdateState(swapInfo *SwapInfo, tokenIn string)
	CloneState() any
	GetMTokenRate() *uint256.Int
	GetMToken() string
}

func (r *RedemptionRpcResult) ToRedemptionState() *RedemptionState {
	if r == nil {
		return nil
	}

	return &RedemptionState{
		Usdc:          r.Usdc,
		RedemptionFee: uint256.MustFromBig(r.RedemptionFee),
		UstbBalance:   uint256.MustFromBig(r.UstbBalance),
		ChainlinkPrice: &ChainlinkPrice{
			IsBadData: r.ChainlinkPrice.IsBadData,
			UpdatedAt: uint256.MustFromBig(r.ChainlinkPrice.UpdatedAt),
			Price:     uint256.MustFromBig(r.ChainlinkPrice.Price),
		},
		ChainLinkFeedPrecision:   uint256.MustFromBig(r.ChainLinkFeedPrecision),
		SuperstateTokenPrecision: uint256.MustFromBig(r.SuperstateTokenPrecision),
	}
}

func (v *VaultStateRpcResult) ToVaultState(mToken string, vaultType VaultType) *VaultState {
	if v == nil {
		return nil
	}

	vault := &VaultState{
		MToken: mToken,
		PaymentTokens: lo.Map(v.PaymentTokens, func(token common.Address, _ int) string {
			return strings.ToLower(token.String())
		}),
		Paused:            v.Paused,
		FnPaused:          v.FnPaused,
		InstantDailyLimit: uint256.MustFromBig(v.InstantDailyLimit),
		DailyLimits:       uint256.MustFromBig(v.DailyLimits),
		InstantFee:        uint256.MustFromBig(v.InstantFee),
		MinAmount:         uint256.MustFromBig(v.MinAmount),
		TokenConfigs: lo.Map(v.TokensConfig, func(cfg TokenConfigRpcResult, _ int) TokenConfig {
			return TokenConfig{
				Fee:       uint256.MustFromBig(cfg.Fee),
				Allowance: uint256.MustFromBig(cfg.Allowance),
				Stable:    cfg.Stable,
			}
		}),
		MTokenRate: uint256.MustFromBig(v.MTokenRate),
		TokenRates: lo.Map(v.TokenRates, func(rate *big.Int, _ int) *uint256.Int {
			return uint256.MustFromBig(rate)
		}),
		WaivedFeeRestriction: v.WaivedFeeRestriction,
		MTokenDecimals:       v.MTokenDecimals,
	}

	switch vaultType {
	case depositVault:
		vault.MinMTokenAmountForFirstDeposit = uint256.MustFromBig(v.MinMTokenAmountForFirstDeposit)
		vault.TotalMinted = uint256.MustFromBig(v.TotalMinted)
		vault.MaxSupplyCap = uint256.MustFromBig(v.MaxSupplyCap)
		vault.MTokenTotalSupply = uint256.MustFromBig(v.MTokenTotalSupply)
	case redemptionVault:
		vault.TokenBalances = toU256Slice(v.TokenBalances)
	case redemptionVaultUstb:
		vault.TokenBalances = toU256Slice(v.TokenBalances)
		vault.Redemption = v.Redemption.ToRedemptionState()
	case redemptionVaultSwapper:
		vault.TokenBalances = toU256Slice(v.TokenBalances)
		vault.MToken2Balance = uint256.MustFromBig(v.MToken2Balance)
		if vault.MTbillRedemptionVault != nil {
			vault.SwapperVaultType = v.SwapperVaultType
			vault.MTbillRedemptionVault = v.MTbillRedemptionVault.ToVaultState(v.MTbillRedemptionVault.MToken, v.SwapperVaultType)
		}
	}

	return vault
}
