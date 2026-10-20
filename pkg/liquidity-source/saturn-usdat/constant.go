package saturnusdat

import "errors"

const (
	DexType = "saturn-usdat"

	// Facility is the SwapFacility proxy: swap router and USDC spender.
	Facility = "0x0bc305e7e13113caed3f5486849e9518a1cc4173"
	// USDat is the MultiMint extension token; it is also the pool address.
	USDat = "0x23238f20b894f29041f48d88ee91131c395aaa71"
	USDC  = "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"

	// DefaultDecimals is the decimals of USDC and of USDat (PYUSDX_DECIMALS).
	DefaultDecimals = 6

	// defaultGas is the observed gasUsed of SwapFacility.swap(USDC, USDat, ...).
	defaultGas int64 = 185000
)

const (
	facilityMethodPaused              = "paused"
	facilityMethodIsApprovedExtension = "isApprovedExtension"

	usdatMethodPaused             = "paused"
	usdatMethodIsWhitelistEnabled = "isWhitelistEnabled"
	usdatMethodIsWhitelisted      = "isWhitelisted"
	usdatMethodIsAllowedAsset     = "isAllowedAsset"
	usdatMethodAssetCap           = "assetCap"
	usdatMethodAssetBalanceOf     = "assetBalanceOf"
	usdatMethodAssetDecimals      = "assetDecimals"
	usdatMethodExtensionDecimals  = "PYUSDX_DECIMALS"
)

// maxTenPow is the largest n for which 10**n fits in 256 bits.
const maxTenPow = 77

var (
	ErrInvalidSwapPath      = errors.New("saturn-usdat: only USDC -> USDat is supported")
	ErrZeroAmount           = errors.New("saturn-usdat: zero amount")
	ErrOverflow             = errors.New("saturn-usdat: uint256 arithmetic overflow")
	ErrFacilityPaused       = errors.New("saturn-usdat: swap facility is paused")
	ErrUSDatPaused          = errors.New("saturn-usdat: USDat is paused")
	ErrExtensionNotApproved = errors.New("saturn-usdat: USDat is not an approved extension")
	ErrAssetNotAllowed      = errors.New("saturn-usdat: asset is not allowed")
	ErrAssetCapReached      = errors.New("saturn-usdat: asset cap reached")
	ErrNotWhitelisted       = errors.New("saturn-usdat: executor is not whitelisted")
	ErrInvalidExtra         = errors.New("saturn-usdat: invalid pool extra")
)
