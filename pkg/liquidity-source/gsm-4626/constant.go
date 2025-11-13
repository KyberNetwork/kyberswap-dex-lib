package gsm4626

import (
	"errors"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const (
	DexType = "gsm-4626"

	gsmMethodPriceStrategy         = "PRICE_STRATEGY"
	gsmMethodGhoToken              = "GHO_TOKEN"
	gsmMethodUnderlyingAsset       = "UNDERLYING_ASSET"
	gsmMethodCanSwap               = "canSwap"
	gsmMethodGetAvailableLiquidity = "getAvailableLiquidity"
	gsmMethodGetExposureCap        = "getExposureCap"
	gsmMethodGetFeeStrategy        = "getFeeStrategy"

	priceStrategyMethodPriceRatio = "PRICE_RATIO"

	feeStrategyMethodGetSellFee = "getSellFee"
	feeStrategyMethodGetBuyFee  = "getBuyFee"

	sellAssetGas                 int64 = 105277
	getAssetAmountForBuyAssetGas int64 = 33930
	buyAssetGas                  int64 = 135073
)

var (
	ray              = u256.TenPow(27)
	percentageFactor = u256.TenPow(4)
)

var (
	ErrInsufficientAvailableExogenousAssetLiquidity = errors.New("INSUFFICIENT_AVAILABLE_EXOGENOUS_ASSET_LIQUIDITY")
	ErrExogenousAssetExposureTooHigh                = errors.New("EXOGENOUS_ASSET_EXPOSURE_TOO_HIGH")
	ErrCannotSwap                                   = errors.New("cannot swap")
	ErrInvalidAmount                                = errors.New("invalid amount")
)
