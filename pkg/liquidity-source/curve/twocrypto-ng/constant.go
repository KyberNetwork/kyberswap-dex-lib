package twocryptong

import (
	"errors"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/int256"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/holiman/uint256"
)

const (
	DexType = "curve-twocrypto-ng"

	DefaultGas = 240000

	poolMethodD                  = "D"
	poolMethodFeeGamma           = "fee_gamma"
	poolMethodMidFee             = "mid_fee"
	poolMethodOutFee             = "out_fee"
	poolMethodInitialAGamma      = "initial_A_gamma"
	poolMethodInitialAGammaTime  = "initial_A_gamma_time"
	poolMethodFutureAGamma       = "future_A_gamma"
	poolMethodFutureAGammaTime   = "future_A_gamma_time"
	poolMethodXcpProfit          = "xcp_profit"
	poolMethodVirtualPrice       = "virtual_price"
	poolMethodAllowedExtraProfit = "allowed_extra_profit"
	poolMethodAdjustmentStep     = "adjustment_step"
	poolMethodBalances           = "balances"
	poolMethodPriceScale         = "price_scale"
	poolMethodPriceOracle        = "price_oracle"
	poolMethodLastPrices         = "last_prices"
	poolMethodMath               = "MATH"
	poolMethodPolicy             = "POLICY"

	poolMethodVersion                    = "version"
	poolMethodLastTimestamp              = "last_timestamp"
	poolMethodDonationShares             = "donation_shares"
	poolMethodDonationDuration           = "donation_duration"
	poolMethodLastDonationReleaseTs      = "last_donation_release_ts"
	poolMethodDonationProtectionExpiryTs = "donation_protection_expiry_ts"
	poolMethodDonationProtectionPeriod   = "donation_protection_period"
	poolMethodLpXcpProfit                = "lp_xcp_profit"
	poolMethodReservedProfitFraction     = "reserved_profit_fraction"
	poolMethodAdminFee                   = "admin_fee"

	policyMethodState        = "state"
	policyMethodFastHalfLife = "FAST_HALF_LIFE"
	policyMethodSlowHalfLife = "SLOW_HALF_LIFE"
	policyMethodKappa        = "KAPPA"
	policyMethodDeadband     = "DEADBAND"
	policyMethodMinCap       = "MIN_CAP"
	policyMethodMaxCap       = "MAX_CAP"

	PoolMethodD                  = poolMethodD
	PoolMethodFeeGamma           = poolMethodFeeGamma
	PoolMethodMidFee             = poolMethodMidFee
	PoolMethodOutFee             = poolMethodOutFee
	PoolMethodInitialAGamma      = poolMethodInitialAGamma
	PoolMethodInitialAGammaTime  = poolMethodInitialAGammaTime
	PoolMethodFutureAGamma       = poolMethodFutureAGamma
	PoolMethodFutureAGammaTime   = poolMethodFutureAGammaTime
	PoolMethodXcpProfit          = poolMethodXcpProfit
	PoolMethodVirtualPrice       = poolMethodVirtualPrice
	PoolMethodAllowedExtraProfit = poolMethodAllowedExtraProfit
	PoolMethodAdjustmentStep     = poolMethodAdjustmentStep
	PoolMethodBalances           = poolMethodBalances
	PoolMethodPriceScale         = poolMethodPriceScale
	PoolMethodPriceOracle        = poolMethodPriceOracle
	PoolMethodLastPrices         = poolMethodLastPrices
	PoolMethodMath               = poolMethodMath
	PoolMethodPolicy             = poolMethodPolicy

	MaxLoopLimit = 256
	NumTokens    = 2
)

var (
	PriceMask = uint256.MustFromHex("0xffffffffffffffffffffffffffffffff")

	U_1e3  = uint256.MustFromDecimal("1000")
	U_1260 = uint256.MustFromDecimal("1260")
	U_1e6  = uint256.MustFromDecimal("1000000")
	U_1e10 = uint256.MustFromDecimal("10000000000")
	U_1e12 = uint256.MustFromDecimal("1000000000000")
	U_1e14 = uint256.MustFromDecimal("100000000000000")
	U_1e16 = uint256.MustFromDecimal("10000000000000000")
	U_1e18 = uint256.MustFromDecimal("1000000000000000000")
	U_2e18 = uint256.MustFromDecimal("2000000000000000000")
	U_3e18 = uint256.MustFromDecimal("3000000000000000000")
	U_1e20 = uint256.MustFromDecimal("100000000000000000000")
	U_1e36 = uint256.MustFromDecimal("1000000000000000000000000000000000000")

	I_1e48, _ = int256.FromDec("1000000000000000000000000000000000000000000000000")
	I_1e46, _ = int256.FromDec("10000000000000000000000000000000000000000000000")
	I_1e44, _ = int256.FromDec("100000000000000000000000000000000000000000000")
	I_1e42, _ = int256.FromDec("1000000000000000000000000000000000000000000")
	I_1e40, _ = int256.FromDec("10000000000000000000000000000000000000000")
	I_1e38, _ = int256.FromDec("100000000000000000000000000000000000000")
	I_1e36, _ = int256.FromDec("1000000000000000000000000000000000000")
	I_1e34, _ = int256.FromDec("10000000000000000000000000000000000")
	I_3e32, _ = int256.FromDec("300000000000000000000000000000000")
	I_1e32, _ = int256.FromDec("100000000000000000000000000000000")
	I_1e30, _ = int256.FromDec("1000000000000000000000000000000")
	I_1e28, _ = int256.FromDec("10000000000000000000000000000")
	I_1e26, _ = int256.FromDec("100000000000000000000000000")
	I_1e24, _ = int256.FromDec("1000000000000000000000000")
	I_1e22, _ = int256.FromDec("10000000000000000000000")
	I_1e20, _ = int256.FromDec("100000000000000000000")
	I_4e18, _ = int256.FromDec("4000000000000000000")
	I_1e18, _ = int256.FromDec("1000000000000000000")
	I_1e16, _ = int256.FromDec("10000000000000000")
	I_4e14, _ = int256.FromDec("400000000000000")
	I_2e14, _ = int256.FromDec("200000000000000")
	I_1e14, _ = int256.FromDec("100000000000000")
	I_1e12, _ = int256.FromDec("1000000000000")
	I_1e10, _ = int256.FromDec("10000000000")
	I_4e8, _  = int256.FromDec("400000000")
	I_1e8, _  = int256.FromDec("100000000")
	I_1e6, _  = int256.FromDec("1000000")
	I_1e4, _  = int256.FromDec("10000")
	I_1e2, _  = int256.FromDec("100")
	I_27, _   = int256.FromDec("27")
	I_9, _    = int256.FromDec("9")

	AMultiplier   = uint256.MustFromDecimal("10000")
	MinGamma      = uint256.MustFromDecimal("10000000000")
	MaxGammaSmall = uint256.MustFromDecimal("20000000000000000")
	MaxGamma      = uint256.MustFromDecimal("199000000000000000")
	MinA          = number.Div(number.Mul(number.Number_4, AMultiplier), number.Number_10) // 4 = NCoins ** NCoins
	MaxA          = number.Mul(number.Mul(number.Number_4, AMultiplier), U_1e3)
	MinD          = uint256.MustFromDecimal("100000000000000000")
	MaxD          = uint256.MustFromDecimal("1000000000000000000000000000000000")
	MinFrac       = uint256.MustFromDecimal("10000000000000000")
	MaxFrac       = uint256.MustFromDecimal("100000000000000000000")

	MinX0 = uint256.MustFromDecimal("1000000000")
	MaxX1 = uint256.MustFromDecimal("1000000000000000000000000000000000")

	CbrtConst1 = uint256.MustFromDecimal("115792089237316195423570985008687907853269000000000000000000")
	CbrtConst2 = uint256.MustFromDecimal("115792089237316195423570985008687907853269")

	NumTokensU256 = uint256.NewInt(NumTokens)

	SupportedImplementation = mapset.NewSet("twocrypto-optimized")

	Precision = U_1e18
)

var (
	ErrInvalidReserve      = errors.New("invalid reserve")
	ErrInvalidNumToken     = errors.New("invalid number of token")
	ErrZero                = errors.New("zero")
	ErrLoss                = errors.New("loss")
	ErrDDoesNotConverge    = errors.New("d does not converge")
	ErrYDoesNotConverge    = errors.New("y does not converge")
	ErrWadExpOverflow      = errors.New("wad_exp overflow")
	ErrUnsafeY             = errors.New("unsafe value for y")
	ErrUnsafeA             = errors.New("unsafe values A")
	ErrUnsafeGamma         = errors.New("unsafe values gamma")
	ErrUnsafeD             = errors.New("unsafe values D")
	ErrUnsafeX0            = errors.New("unsafe values x[0]")
	ErrUnsafeXi            = errors.New("unsafe values x[i]")
	ErrCoinIndexOutOfRange = errors.New("coin index out of range")
	ErrExchange0Coins      = errors.New("do not exchange 0 coins")

	// ErrPoolVersionNotSupported refuses state this package cannot run: pools without allowed_extra_profit
	// (version v3.0.0) rebalance by another rule, and only some of them quote here (see PoolSimulator.isV3).
	ErrPoolVersionNotSupported = errors.New("pool version not supported")
	// ErrStateAfterTrade refuses quotes on a v3.0.0 pool's state after UpdateBalance: the pool may have
	// rebalanced there by a rule this package does not run.
	ErrStateAfterTrade = errors.New("state after a trade not simulated for this pool version")
	// ErrNegativeDonation and ErrTweakedSupply are v2.1.0d's tweak_price asserts.
	ErrNegativeDonation = errors.New("negative donation")
	ErrTweakedSupply    = errors.New("tweaked supply must shrink")
	// ErrUnbalanced is v3.0.0's _assert_balance: a trade may not leave the balances 1000:1 or worse.
	ErrUnbalanced = errors.New("!balance")
)

const (
	// packed_rebalancing_params holds [allowed_extra_profit, adjustment_step, ma_time], 64 bits each.
	// The EMA uses this ma_time; the ma_time() view reports it multiplied by ln 2.
	poolMethodPackedRebalancingParams = "packed_rebalancing_params"
	PoolMethodPackedRebalancingParams = poolMethodPackedRebalancingParams
)

// MaTimeFromRebalancingParams returns the EMA time constant packed in packed_rebalancing_params.
func MaTimeFromRebalancingParams(packed *big.Int) *uint256.Int {
	var maTime uint256.Int
	maTime.SetFromBig(packed)
	return maTime.And(&maTime, maTimeMask)
}

var maTimeMask = new(uint256.Int).SetUint64(1<<64 - 1)
