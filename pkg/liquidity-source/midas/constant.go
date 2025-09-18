package midas

import (
	"errors"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const (
	DexType = "midas"

	vaultPausedMethod            = "paused"
	vaultMTokenDataFeedMethod    = "mTokenDataFeed"
	vaultGetPaymentTokensMethod  = "getPaymentTokens"
	vaultTokensConfigMethod      = "tokensConfig"
	vaultInstantDailyLimitMethod = "instantDailyLimit"
	vaultInstantFeeMethod        = "instantFee"
	vaultDailyLimitsMethod       = "dailyLimits"
	vaultMinAmountMethod         = "minAmount"
	vaultFnPausedMethod          = "fnPaused"

	redemptionVaultSwapperMTbillRedemptionVaultMethod = "mTbillRedemptionVault"
	redemptionVaultSwapperLiquidityProviderMethod     = "liquidityProvider"

	dataFeedGetDataInBase18Method = "getDataInBase18"
)

const (
	RedemptionVaultWithBuidl redemptionVaultType = iota
	RedemptionVaultWithSwapper
	RedemptionVaultWithUstb
)

const (
	DepositVaultDefault  depositVaultType = iota
	DepositVaultWithUstb                  = iota
)

const (
	depositInstantVaultDefaultGas    = 252674
	redemptionInstantVaultSwapperGas = 0
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
	ErrNotSupported          = errors.New("not supported")
	ErrRVAmountMTokenLteFee  = errors.New("RV: amountMTokenIn < fee")
	ErrMVExceedLimit         = errors.New("MV: exceed limit")
)
