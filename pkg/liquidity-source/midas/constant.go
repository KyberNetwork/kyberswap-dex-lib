package midas

import (
	"errors"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const (
	DexType = "midas"

	vPausedMethod               = "paused"
	vMTokenDataFeedMethod       = "mTokenDataFeed"
	vGetPaymentTokensMethod     = "getPaymentTokens"
	vMTokenMethod               = "mToken"
	vTokensConfigMethod         = "tokensConfig"
	vInstantDailyLimitMethod    = "instantDailyLimit"
	vInstantFeeMethod           = "instantFee"
	vDailyLimitsMethod          = "dailyLimits"
	vMinAmountMethod            = "minAmount"
	vFnPausedMethod             = "fnPaused"
	vWaivedFeeRestrictionMethod = "waivedFeeRestriction"

	dvMinMTokenAmountForFirstDepositMethod = "minMTokenAmountForFirstDeposit"
	dvTotalMintedMethod                    = "totalMinted"
	dvMaxSupplyCapMethod                   = "maxSupplyCap"

	rvSwapperLiquidityProviderMethod     = "liquidityProvider"
	rvSwapperMTbillRedemptionVaultMethod = "mTbillRedemptionVault"

	rvUstbUstbRedemptionMethod = "ustbRedemption"

	redemptionSuperstateTokenMethod          = "SUPERSTATE_TOKEN"
	redemptionChainlinkFeedPrecisionMethod   = "CHAINLINK_FEED_PRECISION"
	redemptionSuperstateTokenPrecisionMethod = "SUPERSTATE_TOKEN_PRECISION"
	redemptionUsdcMethod                     = "USDC"
	redemptionRedemptionFeeMethod            = "redemptionFee"
	redemptionGetChainlinkPriceMethod        = "getChainlinkPrice"

	dataFeedGetDataInBase18Method = "getDataInBase18"
)

const (
	depositVault VaultType = "dv"

	redemptionVault        VaultType = "rv"
	redemptionVaultUstb    VaultType = "rvUstb"
	redemptionVaultSwapper VaultType = "rvSwapper"
)

const (
	depositInstantDefaultGas = 252674

	redeemInstantDefaultGas = 250236
	redeemInstantSwapperGas = 675714
	redeemInstantUstbGas    = 445950
)

var (
	stableCoinRate       = u256.TenPow(18)
	oneDayInSecond int64 = 86400

	feeDenominator = u256.UBasisPoint
	usdcPrecision  = u256.TenPow(6)
)

var (
	ErrInvalidToken                 = errors.New("invalid token")
	ErrInvalidAmount                = errors.New("invalid amount")
	ErrInvalidSwap                  = errors.New("invalid swap")
	ErrZeroSwap                     = errors.New("zero swap")
	ErrTokenRemoved                 = errors.New("MV: token not exists")
	ErrMVExceedAllowance            = errors.New("MV: exceed allowance")
	ErrMVExceedLimit                = errors.New("MV: exceed limit")
	ErrRVInsufficientBalance        = errors.New("RV: insufficient balance")
	ErrRateZero                     = errors.New("DV: rate zero")
	ErrDVPaused                     = errors.New("DV: deposit vault paused")
	ErrDepositInstantFnPaused       = errors.New("DV: depositInstant fn paused")
	ErrDVInvalidMintAmount          = errors.New("DV: invalid mint amount")
	ErrDvMTokenAmountLtMin          = errors.New("DV: mToken amount < min")
	ErrDvMintAmountLtMin            = errors.New("DV: mint amount < min")
	ErrDvMaxSupplyCapExceeded       = errors.New("DV: max supply cap exceeded")
	ErrRVPaused                     = errors.New("RV: redemption vault paused")
	ErrRedeemInstantFnPaused        = errors.New("RV: redeemInstant fn paused")
	ErrRVInsufficientMToken2Balance = errors.New("RV: insufficient mToken2 balance")
	ErrRVUUstbFeeNotZero            = errors.New("RVU: USTB fee not zero")
	ErrRVUInsufficientUstbBalance   = errors.New("RVU: insufficient USTB balance")
	ErrRVUInvalidToken              = errors.New("RVU: invalid token")
	ErrBadArgsUsdcOutAmountZero     = errors.New("BadArgs: USDC out amount zero")
	ErrNotSupported                 = errors.New("not supported")
	ErrInvalidTokenRate             = errors.New("invalid token rate")
)
