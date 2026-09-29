package synthereum

import (
	"errors"
)

const (
	DexType = "synthereum"

	// pool types. Exported: the executor-side encoder reads PoolMeta.PoolType to
	// choose which on-chain entry point a swap encodes to.
	PoolTypeMultiLP = "multi-lp"
	PoolTypeWrapper = "wrapper"

	reserveZero = "0"
	// placeholder reserve for the synthetic side of the fixed-rate wrapper (wrap capacity is unbounded)
	defaultSynthReserve = "100000000000000000000000000"

	// SynthereumMultiLpLiquidityPool methods
	poolMethodMaxTokensCapacity    = "maxTokensCapacity"
	poolMethodTotalSyntheticTokens = "totalSyntheticTokens"
	poolMethodFeePercentage        = "feePercentage"
	poolMethodSynthereumFinder     = "synthereumFinder"
	poolMethodPriceFeedIdentifier  = "priceFeedIdentifier"
	poolMethodCollateralToken      = "collateralToken"
	poolMethodSyntheticToken       = "syntheticToken"

	// SynthereumFinder / SynthereumPriceFeed methods (resolve + read the Chainlink-backed
	// oracle rate the pool itself uses for getMintTradeInfo/getRedeemTradeInfo)
	finderMethodGetImplementationAddress = "getImplementationAddress"
	priceFeedMethodGetLatestPrice        = "getLatestPrice"

	// ERC4626 vault methods (Morpho vault holding the wrapper's collateral)
	// maxWithdraw, not previewRedeem(balanceOf): previewRedeem values the wrapper's
	// shares, but a Morpho vault lends most of its assets out, so only maxWithdraw is
	// actually redeemable right now. Unwrapping above it reverts NotEnoughLiquidity().
	vaultMethodMaxWithdraw = "maxWithdraw"
	vaultMethodMaxDeposit  = "maxDeposit"

	// SynthereumFixedRateLendingWrapper methods
	wrapperMethodTotalSyntheticTokens = "totalSyntheticTokens"
	wrapperMethodConversionRate       = "conversionRate"
	// lendingModule() returns (moduleId, bearingToken); the bearing token is the
	// ERC4626 vault the wrapper deposits its collateral into.
	wrapperMethodLendingModule = "lendingModule"

	// SynthereumRegistry methods (PoolRegistry / FixedRateRegistry) used to
	// enumerate every deployed pool from the Finder.
	registryMethodGetSyntheticTokens = "getSyntheticTokens"
	registryMethodGetCollaterals     = "getCollaterals"
	registryMethodGetVersions        = "getVersions"
	registryMethodGetElements        = "getElements"
)

// priceFeedInterfaceName is SynthereumInterfaces.PriceFeed, the bytes32 key the pool's
// Finder uses to resolve the price-feed module address. Solidity encodes a short string
// literal assigned to bytes32 as its raw ASCII bytes, left-aligned and right-padded with
// zeros — this is not keccak256("PriceFeed").
var priceFeedInterfaceName = interfaceName("PriceFeed")

// Registries resolved from the Finder: PoolRegistry lists the multi-lp liquidity
// pools, FixedRateRegistry the fixed-rate lending wrappers. Both expose the same
// enumeration surface, so discovery walks them identically and only the resulting
// pool type differs.
var registriesByPoolType = map[string][32]byte{
	PoolTypeMultiLP: interfaceName("PoolRegistry"),
	PoolTypeWrapper: interfaceName("FixedRateRegistry"),
}

// interfaceName encodes a Finder key. Solidity encodes a short string literal
// assigned to bytes32 as its raw ASCII bytes, left-aligned and right-padded with
// zeros -- this is not keccak256 of the name.
func interfaceName(s string) (b [32]byte) {
	copy(b[:], s)
	return b
}

// Measured per-hop cost inside the aggregator's executor, read from the pool
// sub-call in Tenderly traces of real router swaps (not standalone pool calls, and
// excluding intrinsic gas and router overhead) -- the same basis every other source
// in this repo uses, e.g. uniswap-v2 76562, uniswap-v3 109334, curve 128000.
//
// Gas does not scale with amountIn: redeem/wrap/unwrap measured byte-identical
// across a 10,000x range, and mint varies 831602..869606 with the set of LPs the
// pool has to walk, not with trade size. The values below are the maximum observed
// per operation (mint also covers jBRL's 837687, which is cheaper than jEUR's).
var defaultGas = Gas{
	Mint:   869606,
	Redeem: 1043470,
	Wrap:   428000,
	Unwrap: 405027,
}

var (
	ErrMissingFinder           = errors.New("missing finder address in config")
	ErrRegistryUnavailable     = errors.New("failed to resolve pool registries from finder")
	ErrMissingVault            = errors.New("wrapper reported no lending vault")
	ErrInvalidToken            = errors.New("invalid token")
	ErrInvalidAmountIn         = errors.New("invalid amount in")
	ErrUnsupportedPoolType     = errors.New("unsupported pool type")
	ErrTradeUnavailable        = errors.New("trade info unavailable")
	ErrExceedsMaxCapacity      = errors.New("exceeds max synthetic tokens capacity")
	ErrExceedsRedeemCapacity   = errors.New("exceeds total synthetic tokens available for redeem")
	ErrInsufficientWrapReserve = errors.New("insufficient wrapper collateral reserve")
	ErrExceedsWrapCapacity     = errors.New("exceeds the vault's remaining deposit capacity")
	ErrWrongSynthTokenRounding = errors.New("unwrap amount is not an exact multiple of the wrapper's scaling factor")
	ErrZeroAmountOut           = errors.New("zero amount out")
	ErrOverflow                = errors.New("overflow")
)
