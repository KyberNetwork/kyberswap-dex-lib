package midas

import (
	"errors"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const (
	DexType = "midas"

	vaultPausedMethod                   = "paused"
	depositVaultMTokenDataFeedMethod    = "mTokenDataFeed"
	depositVaultGetPaymentTokensMethod  = "getPaymentTokens"
	depositVaultTokensConfigMethod      = "tokensConfig"
	depositVaultInstantDailyLimitMethod = "instantDailyLimit"
	depositVaultInstantFeeMethod        = "instantFee"
	depositVaultDailyLimitsMethod       = "dailyLimits"
	depositVaultMinAmountMethod         = "minAmount"

	dataFeedGetDataInBase18Method = "getDataInBase18"

	vaultFnPausedMethod = "fnPaused"
)

const (
	redemptionVaultBuidl redemptionVaultType = iota
	redemptionVaultSwapper
	redemptionVaultUstb
)

var (
	StableCoinRate       = u256.TenPow(18)
	oneDayInSecond int64 = 86400
)

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrInvalidAmount         = errors.New("invalid amount")
	ErrTokenRemoved          = errors.New("MV: token not exists")
	ErrRateZero              = errors.New("DV: rate zero")
	ErrMVExceedAllowance     = errors.New("MV: exceed allowance")
	ErrDepositVaultPaused    = errors.New("deposit vault paused")
	ErrFnPaused              = errors.New("fn paused")
	ErrRedemptionVaultPaused = errors.New("redemption vault paused")
)
