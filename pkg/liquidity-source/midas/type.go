package midas

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
)

type VaultType string

type TokenConfig struct {
	DataFeed  common.Address `json:"dataFeed"`
	Fee       *uint256.Int   `json:"fee"`
	Allowance *uint256.Int   `json:"allowance"`
	Stable    bool           `json:"stable"`
}

type ChainlinkPriceRpcResult struct {
	IsBadData bool
	UpdatedAt *big.Int
	Price     *big.Int
}

type RedemptionRpcResult struct {
	SuperstateToken          common.Address
	Usdc                     common.Address
	RedemptionFee            *big.Int
	UstbBalance              *big.Int
	ChainlinkPrice           ChainlinkPriceRpcResult
	ChainLinkFeedPrecision   *big.Int
	SuperstateTokenPrecision *big.Int
}

type VaultStateRpcResult struct {
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
	MTokenDataFeed       common.Address
	WaivedFeeRestriction bool

	MTokenRate *big.Int
	TokenRate  *big.Int

	// For deposit vault
	MinMTokenAmountForFirstDeposit *big.Int
	TotalMinted                    *big.Int
	MaxSupplyCap                   *big.Int
	MTokenTotalSupply              *big.Int

	// For redemption vault
	TokenBalance *big.Int

	// For redemption vault with ustb
	Redemption RedemptionRpcResult

	// For redemption vault with swapper
	MToken1Balance        *big.Int
	MToken2Balance        *big.Int
	SwapperVaultType      VaultType
	MTbillRedemptionVault *VaultStateRpcResult
}

type VaultState struct {
	TokenRemoved         bool         `json:"tokenRemoved"`
	Paused               bool         `json:"paused"`
	FnPaused             bool         `json:"fnPaused"`
	TokenConfig          *TokenConfig `json:"tokenConfig"`
	InstantDailyLimit    *uint256.Int `json:"instantDailyLimit"`
	DailyLimits          *uint256.Int `json:"dailyLimits"`
	InstantFee           *uint256.Int `json:"instantFee"`
	MinAmount            *uint256.Int `json:"minAmount"`
	MTokenRate           *uint256.Int `json:"mTokenRate"`
	TokenRate            *uint256.Int `json:"tokenRate"`
	WaivedFeeRestriction bool         `json:"waivedFeeRestriction"`

	// For deposit vault
	MinMTokenAmountForFirstDeposit *uint256.Int `json:"minMTokenAmountForFirstDeposit,omitempty"`
	TotalMinted                    *uint256.Int `json:"totalMinted,omitempty"`
	MaxSupplyCap                   *uint256.Int `json:"maxSupplyCap,omitempty"`
	MTokenTotalSupply              *uint256.Int `json:"mTokenTotalSupply,omitempty"`

	// For redemption vault
	TokenBalance *uint256.Int `json:"tokenBalance,omitempty"`

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
	SuperstateToken          common.Address  `json:"superstateToken"`
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
	Vault     string    `json:"vault"`
	VaultType VaultType `json:"type"`
}

type Meta struct {
	BlockNumber uint64 `json:"blockNumber"`
	Vault       string `json:"vault"`
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
	DepositInstant(amountTokenIn *uint256.Int) (*SwapInfo, error)
	UpdateState(swapInfo *SwapInfo)
	CloneState() any
}

type IRedemptionVault interface {
	GetMTokenRate() *uint256.Int
	RedeemInstant(amountMTokenIn *uint256.Int, tokenOut string) (*SwapInfo, error)
	UpdateState(swapInfo *SwapInfo)
	CloneState() any
}

func (r *RedemptionRpcResult) ToRedemptionState() *RedemptionState {
	if r == nil {
		return nil
	}

	return &RedemptionState{
		SuperstateToken: r.SuperstateToken,
		Usdc:            r.Usdc,
		RedemptionFee:   uint256.MustFromBig(r.RedemptionFee),
		UstbBalance:     uint256.MustFromBig(r.UstbBalance),
		ChainlinkPrice: &ChainlinkPrice{
			IsBadData: r.ChainlinkPrice.IsBadData,
			UpdatedAt: uint256.MustFromBig(r.ChainlinkPrice.UpdatedAt),
			Price:     uint256.MustFromBig(r.ChainlinkPrice.Price),
		},
		ChainLinkFeedPrecision:   uint256.MustFromBig(r.ChainLinkFeedPrecision),
		SuperstateTokenPrecision: uint256.MustFromBig(r.SuperstateTokenPrecision),
	}
}

func (v *VaultStateRpcResult) ToVaultState(vaultType VaultType, token string) *VaultState {
	if v == nil {
		return nil
	}

	vault := &VaultState{
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
		MTokenRate: uint256.MustFromBig(v.MTokenRate),
		TokenRate:  uint256.MustFromBig(v.TokenRate),
	}

	switch vaultType {
	case depositVault:
		vault.MinMTokenAmountForFirstDeposit = uint256.MustFromBig(v.MinMTokenAmountForFirstDeposit)
		vault.TotalMinted = uint256.MustFromBig(v.TotalMinted)
		vault.MaxSupplyCap = uint256.MustFromBig(v.MaxSupplyCap)
		vault.MTokenTotalSupply = uint256.MustFromBig(v.MTokenTotalSupply)
	case redemptionVault:
		vault.TokenBalance = uint256.MustFromBig(v.TokenBalance)
	case redemptionVaultUstb:
		vault.TokenBalance = uint256.MustFromBig(v.TokenBalance)
		vault.Redemption = v.Redemption.ToRedemptionState()
	case redemptionVaultSwapper:
		vault.TokenBalance = uint256.MustFromBig(v.TokenBalance)
		vault.MToken1Balance = uint256.MustFromBig(v.MToken1Balance)
		vault.MToken2Balance = uint256.MustFromBig(v.MToken2Balance)
		vault.SwapperVaultType = v.SwapperVaultType
		vault.MTbillRedemptionVault = v.MTbillRedemptionVault.ToVaultState(v.SwapperVaultType, token)
	}

	return vault
}
