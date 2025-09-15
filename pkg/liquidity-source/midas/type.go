package midas

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type TokenConfig struct {
	DataFeed  common.Address `json:"dataFeed"`
	Fee       *uint256.Int   `json:"fee"`
	Allowance *uint256.Int   `json:"allowance"`
	Stable    bool           `json:"stable"`
}

type Extra struct {
	TokenRemoved           bool         `json:"tokenRemoved"`
	TokenConfig            *TokenConfig `json:"tokenConfig"`
	DepositInstantFnPaused bool         `json:"depositInstantFnPaused"`
	RedeemInstantFnPaused  bool         `json:"redeemInstantFnPaused"`
	InstantDailyLimit      *uint256.Int `json:"instantDailyLimit"`
	DailyLimits            *uint256.Int `json:"dailyLimits"`
	TokenRate              *uint256.Int `json:"tokenRate"`
	MTokenRate             *uint256.Int `json:"mTokenRate"`
	InstantFee             *uint256.Int `json:"instantFee"`
	MinAmount              *uint256.Int `json:"minAmount"`

	DepositVaultPaused    bool `json:"depositVaultPaused"`
	RedemptionVaultPaused bool `json:"redemptionVaultPaused"`
}

type redemptionVaultType uint8
type depositVaultType uint8

type StaticExtra struct {
	DataFeed            common.Address      `json:"dataFeed"`
	MTokenDataFeed      common.Address      `json:"mTokenDataFeed"`
	DepositVault        common.Address      `json:"depositVault"`
	RedemptionVault     common.Address      `json:"redemptionVault"`
	DepositVaultType    depositVaultType    `json:"depositVaultType"`
	RedemptionVaultType redemptionVaultType `json:"redemptionVaultType"`
}

type Meta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type SwapInfo struct {
	IsDeposit       bool            `json:"isDeposit"`
	DepositVault    *common.Address `json:"depositVault,omitempty"`
	RedemptionVault *common.Address `json:"redemptionVault,omitempty"`
	AssetsInBase18  *big.Int        `json:"assetsInBase18,omitempty"`

	fee       *big.Int
	amountOut *big.Int
}

type IRedemptionVault interface {
	CalcAndValidateRedeem(amountMTokenIn *uint256.Int)
}
