package stablesfast

import (
	"context"
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// The rate every Stables market ships with: 5_000 pips = 0.50%.
const shippedFeePips = 5_000

func tracked(feePips int64) *Hook {
	return &Hook{Extra: Extra{FeePips: feePips, Registered: true, Tracked: true}}
}

func exactIn(amountIn, amountOut int64) *uniswapv4.AfterSwapParams {
	return &uniswapv4.AfterSwapParams{
		BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true, AmountSpecified: big.NewInt(amountIn)},
		AmountIn:         big.NewInt(amountIn),
		AmountOut:        big.NewInt(amountOut),
	}
}

func exactOut(amountIn, amountOut int64) *uniswapv4.AfterSwapParams {
	return &uniswapv4.AfterSwapParams{
		BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: false, AmountSpecified: big.NewInt(amountOut)},
		AmountIn:         big.NewInt(amountIn),
		AmountOut:        big.NewInt(amountOut),
	}
}

// beforeSwap charges NOTHING, in either direction. The contract returns
// BeforeSwapDeltaLibrary.ZERO_DELTA unconditionally; it is in the callback list only to write
// the oracle observation. A non-zero delta here would double-charge every swap on top of the
// afterSwap fee.
func TestBeforeSwap_ChargesNothingInEitherDirection(t *testing.T) {
	for _, calcOut := range []bool{true, false} {
		result, err := tracked(shippedFeePips).BeforeSwap(&uniswapv4.BeforeSwapParams{
			CalcOut:         calcOut,
			AmountSpecified: big.NewInt(1_000_000_000),
		})
		require.NoError(t, err)

		assert.Equal(t, bignumber.ZeroBI, result.DeltaSpecified)
		assert.Equal(t, bignumber.ZeroBI, result.DeltaUnspecified)
		// The pool's own fee tier is untouched: the hook never overrides the LP fee.
		assert.Zero(t, result.SwapFee)
		// The oracle write is the whole of the cost, and it runs in both directions.
		assert.EqualValues(t, gasObservation, result.Gas)
	}
}

// Exact-in: the unspecified leg is the OUTPUT, so the trader receives amountOut minus the
// skim. Charging the INPUT here is the pre-2026-09-19 behaviour and is now wrong.
func TestAfterSwap_ExactIn_ChargesTheRealisedOutput(t *testing.T) {
	result, err := tracked(shippedFeePips).AfterSwap(exactIn(2_000_000_000, 1_000_000_000))
	require.NoError(t, err)

	// 0.50% of the OUTPUT, not of the 2x-larger input.
	assert.Equal(t, big.NewInt(5_000_000), result.HookFee)
	assert.EqualValues(t, gasAccrue, result.Gas)
}

// Exact-out: the unspecified leg is the realised INPUT, so the trader pays amountIn plus the
// skim.
func TestAfterSwap_ExactOut_ChargesTheRealisedInput(t *testing.T) {
	result, err := tracked(shippedFeePips).AfterSwap(exactOut(1_000_000_000, 2_000_000_000))
	require.NoError(t, err)

	assert.Equal(t, big.NewInt(5_000_000), result.HookFee)
	assert.EqualValues(t, gasAccrue, result.Gas)
}

// A partial fill is charged on the FILL. The contract reads pool.swap's own BalanceDelta, so
// an unfilled remainder is simply absent from it. This is the property that moving the skim
// out of beforeSwap bought, and the one an aggregator setting its own sqrtPriceLimitX96 cares
// about: AmountSpecified is what was asked for, AmountOut is what arrived, and only the
// latter may be charged.
func TestAfterSwap_PartialFill_IsChargedOnTheFill(t *testing.T) {
	params := exactIn(1_000_000_000, 400_000_000) // asked 1,000 in; the limit bound the fill
	params.AmountSpecified = big.NewInt(1_000_000_000)

	result, err := tracked(shippedFeePips).AfterSwap(params)
	require.NoError(t, err)

	// 0.50% of the 400 that filled, not of the 1,000 requested.
	assert.Equal(t, big.NewInt(2_000_000), result.HookFee)
}

// Exact-output is flat, NOT a gross-up. The contract charges the pips on the pool's realised
// net input, so the total is net + floor(net*fee/1e6) and never net*fee/(1e6-fee). Several
// neighbouring hooks in this directory DO gross up; pinning the difference here stops a
// refactor quietly aligning this one with them.
func TestAfterSwap_ExactOut_IsFlatNotGrossedUp(t *testing.T) {
	const net = 1_000_000_000

	result, err := tracked(shippedFeePips).AfterSwap(exactOut(net, 2_000_000_000))
	require.NoError(t, err)

	assert.Equal(t, big.NewInt(5_000_000), result.HookFee, "flat: floor(net*fee/1e6)")
	assert.EqualValues(t, net+5_000_000, net+result.HookFee.Int64())
	// The gross-up an inverted path would have produced: 1_000_000_000*5_000/995_000 =
	// 5_025_125, giving a 1_005_025_125 total. 25,125 units apart on a 1,000-unit leg.
	assert.NotEqual(t, big.NewInt(5_025_125), result.HookFee)
}

// FullMath.mulDiv truncates. Rounding the other way would quote a skim larger than the
// contract takes, which reads as the pool being cheaper than it is.
func TestSkim_TruncatesLikeFullMath(t *testing.T) {
	// floor(199 * 5_000 / 1_000_000) = floor(0.995) = 0
	result, err := tracked(shippedFeePips).AfterSwap(exactIn(10_000, 199))
	require.NoError(t, err)
	// Compared by value: a computed zero and big.NewInt(0) differ in their internal limb
	// slice, and assert.Equal is a deep comparison.
	assert.EqualValues(t, 0, result.HookFee.Int64())

	// floor(201 * 5_000 / 1_000_000) = floor(1.005) = 1
	result, err = tracked(shippedFeePips).AfterSwap(exactIn(10_000, 201))
	require.NoError(t, err)
	assert.EqualValues(t, 1, result.HookFee.Int64())
}

// `feePipsFor` returns zero for a pool this hook never registered and for every pool while
// the protocol guard is halted. Both keep trading; only the protocol's cut stops. Quoting a
// skim in either case would under-quote the output.
func TestZeroRate_QuotesTheBareCurve(t *testing.T) {
	h := tracked(0)

	before, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{
		CalcOut:         true,
		AmountSpecified: big.NewInt(1_000_000_000),
	})
	require.NoError(t, err)
	assert.Equal(t, bignumber.ZeroBI, before.DeltaSpecified)
	// Registered, so the oracle still runs and still costs gas at a zero rate.
	assert.EqualValues(t, gasObservation, before.Gas)

	after, err := h.AfterSwap(exactIn(1_000_000_000, 1_000_000_000))
	require.NoError(t, err)
	assert.Equal(t, bignumber.ZeroBI, after.HookFee)
	assert.Zero(t, after.Gas)
}

// An untracked pool is not a free pool: Track has simply not read the rate yet. Quoting
// zero would promise the whole skim back to the trader, so refuse instead.
func TestUntracked_Refuses(t *testing.T) {
	h := &Hook{}

	_, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, AmountSpecified: big.NewInt(1)})
	assert.ErrorIs(t, err, ErrPoolIsNotTracked)

	_, err = h.AfterSwap(exactIn(1, 1))
	assert.ErrorIs(t, err, ErrPoolIsNotTracked)
}

// Track with no RPC client is the refresh-failure path, and the two cases must diverge. A
// pool already read keeps its last rate, because dropping it out of routing over a transient
// client outage is worse than pricing it a block stale. A pool NEVER read stays untracked and
// refuses, because the alternative is quoting a zero rate and promising the whole skim back.
func TestTrack_WithoutRpcClient_KeepsTheLastReadingOrRefuses(t *testing.T) {
	persisted, err := tracked(shippedFeePips).Track(context.Background(), &uniswapv4.HookParam{})
	require.NoError(t, err)

	var revived Hook
	require.NoError(t, json.Unmarshal(persisted, &revived))
	assert.EqualValues(t, shippedFeePips, revived.FeePips)
	assert.True(t, revived.Registered)
	assert.True(t, revived.Tracked)

	_, err = (&Hook{}).Track(context.Background(), &uniswapv4.HookParam{})
	assert.ErrorIs(t, err, ErrPoolIsNotTracked)
}

// The factory is what dex-lib actually calls, and it must rehydrate persisted Extra rather
// than starting every refresh cycle untracked.
func TestFactory_RehydratesPersistedExtra(t *testing.T) {
	encoded, err := json.Marshal(Extra{FeePips: shippedFeePips, Registered: true, Tracked: true})
	require.NoError(t, err)

	hook, ok := uniswapv4.GetHook(HookAddresses[0], &uniswapv4.HookParam{
		HookExtra: uniswapv4.HookExtra(encoded),
	})
	require.True(t, ok, "the deployed hook address must resolve to this package, not the fallback")

	h, ok := hook.(*Hook)
	require.True(t, ok)
	assert.EqualValues(t, shippedFeePips, h.FeePips)
	assert.True(t, h.Tracked)
	assert.Equal(t, string(valueobject.ExchangeUniswapV4StablesFast), h.GetExchange())
}

// The permission bits are the low 14 of the hook's own address and cannot move, so the
// simulator's swap-permission gate must see both callbacks for the deployed address.
func TestHookAddress_DeclaresBothSwapCallbacks(t *testing.T) {
	h := &uniswapv4.BaseHook{}
	for _, addr := range HookAddresses {
		assert.True(t, uniswapv4.HasSwapPermissions(addr), addr.Hex())
		assert.True(t, h.CanBeforeSwap(addr), addr.Hex())
		assert.True(t, h.CanAfterSwap(addr), addr.Hex())
	}
}

func TestCloneState_IsIndependent(t *testing.T) {
	h := tracked(shippedFeePips)
	cloned, ok := h.CloneState().(*Hook)
	require.True(t, ok)

	cloned.FeePips = 1
	assert.EqualValues(t, shippedFeePips, h.FeePips)
}

// A pool keyed at this hook that the factory never registered gets neither a skim nor an
// oracle write: `feePipsFor` short-circuits on a zero recipient and `_writeObservation`
// returns early on a zero cardinality. Pricing it as a registered pool would charge the
// router for gas the swap never spends.
func TestUnregistered_IsAPureNoOp(t *testing.T) {
	h := &Hook{Extra: Extra{Tracked: true}}

	before, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{
		CalcOut:         true,
		AmountSpecified: big.NewInt(1_000_000_000),
	})
	require.NoError(t, err)
	assert.Equal(t, bignumber.ZeroBI, before.DeltaSpecified)
	assert.Zero(t, before.Gas)

	after, err := h.AfterSwap(exactIn(1_000_000_000, 1_000_000_000))
	require.NoError(t, err)
	assert.Equal(t, bignumber.ZeroBI, after.HookFee)
	assert.Zero(t, after.Gas)
}

// A market created with Uniswap's DYNAMIC_FEE_FLAG earns whatever slot0.lpFee the keeper last
// wrote, and this plugin has no say in it: BeforeSwap leaves SwapFee at zero, so the simulator
// keeps the rate the tracker read from slot0. Proved from the outside — the whole v4 simulator,
// a dynamic pool and a static pool at the same tracked rate must quote to the wei — because
// that is the property a later "helpful" SwapFee override would break.
func TestDynamicFeePool_QuotesAtTheTrackedStoredRate(t *testing.T) {
	t.Parallel()

	const (
		dynamicFeeFlag uint32 = 0x800000 // LPFeeLibrary.DYNAMIC_FEE_FLAG
		storedLpFee    uint32 = 12_000   // what the keeper last wrote, as slot0 reports it
		nvda                  = "0x0000000000000000000000000000000000000a01"
		aiusd                 = "0x0000000000000000000000000000000000000a02"
	)
	extra := `{"liquidity":1000000000000000000000000,"sqrtPriceX96":79228162514264337593543950336,"tickSpacing":50,"tick":0,"ticks":[{"index":-887250,"liquidityGross":1000000000000000000000000,"liquidityNet":1000000000000000000000000},{"index":887250,"liquidityGross":1000000000000000000000000,"liquidityNet":-1000000000000000000000000}],"hX":{"f":5000,"r":true,"t":true}}`
	staticExtra := func(fee uint32) string {
		return `{"0x0":[false,false],"fee":` + big.NewInt(int64(fee)).String() + `,"tS":50,"hooks":"0xc9932584c5154e4f58313a2e5423522e74e540cc","uR":"0x0000000000000000000000000000000000000001","pm2":"0x0000000000000000000000000000000000000002","mc3":"0x0000000000000000000000000000000000000003"}`
	}
	quote := func(keyFee uint32) *big.Int {
		sim, err := uniswapv4.NewPoolSimulator(entity.Pool{
			Address:  "0x00000000000000000000000000000000000000000000000000000000000000d1",
			Exchange: string(valueobject.ExchangeUniswapV4StablesFast),
			Type:     "uniswap-v4",
			SwapFee:  float64(storedLpFee), // the tracker: protocolFee 0, so slot0.lpFee verbatim
			Reserves: entity.PoolReserves{"1000000000000000000000000", "1000000000000000000000000"},
			Tokens: []*entity.PoolToken{
				{Address: nvda, Symbol: "NVDA", Decimals: 18, Swappable: true},
				{Address: aiusd, Symbol: "AIUSD", Decimals: 18, Swappable: true},
			},
			Extra:       extra,
			StaticExtra: staticExtra(keyFee),
		}, valueobject.ChainIDRobinhood)
		require.NoError(t, err)
		require.Equal(t, string(valueobject.ExchangeUniswapV4StablesFast), sim.GetExchange())

		out, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: nvda, Amount: big.NewInt(1_000_000_000_000_000_000)},
			TokenOut:      aiusd,
		})
		require.NoError(t, err)
		return out.TokenAmountOut.Amount
	}

	dynamic, static := quote(dynamicFeeFlag), quote(storedLpFee)
	assert.Equal(t, static.String(), dynamic.String(),
		"a dynamic pool prices at slot0.lpFee, exactly as a static pool at that tier would")

	// And that rate is really 1.2% + the 0.50% skim, not the 0x800000 flag read as a fee:
	// 1e18 in on a deep price-1 pool comes back just under 1e18 * 0.988 * 0.995.
	expected, _ := new(big.Int).SetString("983060000000000000", 10)
	tolerance, _ := new(big.Int).SetString("1000000000000000", 10) // 0.1%, the curve's own slip
	assert.Less(t, new(big.Int).Abs(new(big.Int).Sub(dynamic, expected)).Cmp(tolerance), 0,
		"got %s, expected within 0.1%% of %s", dynamic, expected)
}
