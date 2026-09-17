package lunya

import (
	"errors"

	"github.com/holiman/uint256"
)

const DexType = "lunya"

// ILunyaPool.poolType(). CL and CP pools are the same contract (LunyaPoolCL; a CP pool is a CL pool held
// at full range) and price on the Uniswap V3 curve. STABLE pools are LunyaPoolSTABLE, a tickless pool on
// a two-coin StableSwap curve. One factory lists all three.
const (
	poolTypeCL     = 0
	poolTypeCP     = 1
	poolTypeStable = 2
)

// ILunyaPool.feeToken(): which token the swap fee is charged in. Paid is Uniswap V3's behaviour (the
// input token); Token0/Token1 pin the fee to one side, so half of all swaps pay it out of the output.
const (
	feeTokenPaid   = 0
	feeTokenToken0 = 1
	feeTokenToken1 = 2
)

// Plugins flags, as read by both pools' swap.
const (
	pluginFlagBeforeSwap = 1 << 2
	pluginFlagDynamicFee = 1 << 10
)

const (
	// pluginStatusActive is SecurityModule.Status.Active; any other status reverts beforeSwap.
	pluginStatusActive = 0

	// feeDenominator is SwapFee.DENOMINATOR (StableSwapMath.FEE_DENOMINATOR); SwapFee.MAX is one below.
	feeDenominator = 1_000_000

	// TickTree indexes initialized ticks by tick, not by tick spacing: leaf word w = tick >> 8, whose
	// non-emptiness is bit (w + leafOffset) of the second layer, whose word s is bit s of tickTreeRoot.
	// Leaf words span [MIN_TICK >> 8, MAX_TICK >> 8].
	leafOffset  = 3466
	minLeafWord = -3466
	maxLeafWord = 3465

	wordChunkSize = 256
	tickChunkSize = 100

	// StableSwapMath.MAX_ITER and A_PRECISION, and the round cap of bisectToTargetPrice.
	maxIter      = 255
	aPrecision   = 100
	bisectRounds = 40
)

const (
	poolMethodSlot0        = "slot0"
	poolMethodLiquidity    = "liquidity"
	poolMethodFeeToken     = "feeToken"
	poolMethodPlugin       = "plugin"
	poolMethodPluginConfig = "pluginConfig"
	poolMethodTickTreeRoot = "tickTreeRoot"
	poolMethodTickBitmap   = "tickBitmap"
	poolMethodTicks        = "ticks"

	poolMethodCurveReserve0        = "curveReserve0"
	poolMethodCurveReserve1        = "curveReserve1"
	poolMethodRate0                = "rate0"
	poolMethodRate1                = "rate1"
	poolMethodPriceScaleSqrtQ96    = "priceScaleSqrtQ96"
	poolMethodAmplificationX100    = "amplificationX100"
	poolMethodAmplificationRamping = "amplificationRamping"
	poolMethodAmplificationRamp    = "amplificationRamp"

	pluginMethodCurrentFee = "currentFee"
	pluginMethodStatus     = "status"
)

var (
	// TickMath.MIN_SQRT_RATIO / MAX_SQRT_RATIO
	minSqrtRatio = uint256.NewInt(4295128739)
	maxSqrtRatio = uint256.MustFromDecimal("1461446703485210103287273052203988822378723970342")

	// Measured through ks-dex-adapter-lib's LunyaAdapter on an Arc testnet fork, cold storage.
	// LunyaDefaultPlugin's beforeSwap writes an oracle observation and its afterSwap settles limit
	// orders, so a Lunya swap costs more than a bare Uniswap V3 one.
	clGas = Gas{BaseGas: 200_000, CrossInitTickGas: 25_000} // 198k-278k per swap, median 225k
	// A STABLE swap solves the curve by Newton's method, and bisects it when a price limit binds.
	stableGas = Gas{BaseGas: 210_000} // 200k-302k per swap, median 202k
)

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrInvalidAmount         = errors.New("invalid amount")
	ErrZeroAmount            = errors.New("zero amount")
	ErrInsufficientReserve   = errors.New("insufficient reserve")
	ErrPartialFill           = errors.New("not enough liquidity to fill the output")
	ErrNotInitialized        = errors.New("pool not initialized")
	ErrNoLiquidity           = errors.New("no liquidity")
	ErrTradingHalted         = errors.New("trading halted")
	ErrInvalidFee            = errors.New("invalid fee")
	ErrInvalidSqrtPriceLimit = errors.New("invalid sqrt price limit")
	ErrLiquidityOverflow     = errors.New("liquidity overflow")
	ErrInt256Overflow        = errors.New("int256 overflow")
	ErrArithmetic            = errors.New("stableswap arithmetic error")
	ErrPriceOutOfRange       = errors.New("price out of range")
	ErrUnsupportedPoolType   = errors.New("unsupported pool type")
	ErrMalformedLog          = errors.New("malformed event log")
	ErrFailedCall            = errors.New("pool state call failed")
)
