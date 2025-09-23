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

	depositVaultMinMTokenAmountForFirstDepositMethod = "minMTokenAmountForFirstDeposit"
	depositVaultTotalMintedMethod                    = "totalMinted"
	depositVaultMaxSupplyCapMethod                   = "maxSupplyCap"

	redemptionVaultSwapperMTbillRedemptionVaultMethod = "mTbillRedemptionVault"
	redemptionVaultUstbUstbRedemptionMethod           = "ustbRedemption"

	redemptionSuperstateTokenMethod          = "SUPERSTATE_TOKEN"
	redemptionChainlinkFeedPrecisionMethod   = "CHAINLINK_FEED_PRECISION"
	redemptionSuperstateTokenPrecisionMethod = "SUPERSTATE_TOKEN_PRECISION"
	redemptionUsdcMethod                     = "USDC"
	redemptionRedemptionFeeMethod            = "redemptionFee"
	redemptionGetChainlinkPriceMethod        = "getChainlinkPrice"

	dataFeedGetDataInBase18Method = "getDataInBase18"
)

const (
	depositVault     VaultType = "depositVault"
	depositVaultUstb VaultType = "depositVaultUstb"

	redemptionVault        VaultType = "redemptionVault"
	redemptionVaultSwapper VaultType = "redemptionVaultSwapper"
	redemptionVaultUstb    VaultType = "redemptionVaultUstb"
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

	dummyAddress = "0xFFfFfFffFFfffFFfFFfFFFFFffFFFffffFfFFFfF"
)

var (
	ErrInvalidToken               = errors.New("invalid token")
	ErrInvalidAmount              = errors.New("invalid amount")
	ErrInvalidSwap                = errors.New("invalid swap")
	ErrTokenRemoved               = errors.New("MV: token not exists")
	ErrMVExceedAllowance          = errors.New("MV: exceed allowance")
	ErrMVExceedLimit              = errors.New("MV: exceed limit")
	ErrDVInsufficientBalance      = errors.New("DV: insufficient balance")
	ErrRateZero                   = errors.New("DV: rate zero")
	ErrDepositVaultPaused         = errors.New("DV: deposit vault paused")
	ErrDepositInstantFnPaused     = errors.New("DV: depositInstant fn paused")
	ErrDVInvalidMintAmount        = errors.New("DV: invalid mint amount")
	ErrDvMTokenAmountLtMin        = errors.New("DV: mToken amount < min")
	ErrDvMintAmountLtMin          = errors.New("DV: mint amount < min")
	ErrDvMaxSupplyCapExceeded     = errors.New("DV: max supply cap exceeded")
	ErrRedemptionVaultPaused      = errors.New("RV: redemption vault paused")
	ErrRedeemInstantFnPaused      = errors.New("RV: redeemInstant fn paused")
	ErrRVUUstbFeeNotZero          = errors.New("RVU: USTB fee not zero")
	ErrRVUInsufficientUstbBalance = errors.New("RVU: insufficient USTB balance")
	ErrBadArgsUsdcOutAmountZero   = errors.New("BadArgs: USDC out amount zero")
	ErrNotSupported               = errors.New("not supported")
)
