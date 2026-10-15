package saturn

import "errors"

const (
	DexType = "saturn"

	defaultReserves = "1000000000000000000000000"

	SwapFacilityAddress = "0x0bc305e7e13113caed3f5486849e9518a1cc4173"
	USDatAddress        = "0x23238f20b894f29041f48d88ee91131c395aaa71"
	USDCAddress         = "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
)

const (
	swapFacilityMethodPaused              = "paused"
	swapFacilityMethodIsApprovedExtension = "isApprovedExtension"

	usdatMethodPaused             = "paused"
	usdatMethodIsAllowedAsset     = "isAllowedAsset"
	usdatMethodAssetCap           = "assetCap"
	usdatMethodAssetBalanceOf     = "assetBalanceOf"
	usdatMethodAssetDecimals      = "assetDecimals"
	usdatMethodIsWhitelistEnabled = "isWhitelistEnabled"
	usdatMethodIsWhitelisted      = "isWhitelisted"
)

const pyusdxDecimals uint8 = 6

var (
	defaultGas = Gas{Mint: 180000}
)

var (
	ErrPoolPaused             = errors.New("pool is paused")
	ErrSwapFacilityPaused     = errors.New("swap facility is paused")
	ErrExtensionNotApproved   = errors.New("extension is not approved")
	ErrAssetNotAllowed        = errors.New("asset is not allowed")
	ErrExecutorNotWhitelisted = errors.New("executor is not whitelisted")
	ErrAssetCapReached        = errors.New("asset cap reached")
	ErrInvalidTokenIn         = errors.New("invalid tokenIn")
	ErrInvalidTokenOut        = errors.New("invalid tokenOut")
	ErrInvalidAmountIn        = errors.New("invalid amountIn")
	ErrInvalidAmountOut       = errors.New("invalid amountOut")
	ErrOverflow               = errors.New("math overflow")
)
