package mento

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/int256"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// Monad (143) USDC/USDm FPMM 0x463c0d1F04bcd99A1efCF94AC2a75bc19Ea4A7E5,
// state captured at block 107045873 (timestamp 1790083320).
const (
	usdcUsdmPool = "0x463c0d1f04bcd99a1efcf94ac2a75bc19ea4a7e5"
	usdc         = "0x754704bc059f8c67012fed69bc8a327a5aafb603"
	usdm         = "0xbc69212b8e4d445b2307c9d32dd68e2a4df00115"

	snapshotBlock     = 107045873
	snapshotTimestamp = 1790083320
	rateTimestamp     = 1790081281
	rateExpiry        = 3720
)

const snapshotExtra = `{
	"lpFee": 3, "protocolFee": 2,
	"rateNum": "999935510000000000", "rateDen": "1000000000000000000",
	"rateTs": 1790081281, "rateExpiry": 3720,
	"tradingMode": 0, "marketHours": false,
	"limits": [
		{"l0":"2500000000000000000000","l1":"5000000000000000000000","dec":6,"lu0":1790074988,"lu1":1790052298,"nf0":"-62512118000000000","nf1":"-62512118000000000"},
		{"l0":"2500000000000000000000","l1":"5000000000000000000000","dec":18,"lu0":1790074988,"lu1":1790052298,"nf0":"62506930900000000","nf1":"62506930900000000"}
	]
}`

func snapshotPool(t *testing.T) entity.Pool {
	t.Helper()
	return entity.Pool{
		Address:     usdcUsdmPool,
		Exchange:    string(DexType),
		Type:        DexType,
		Timestamp:   snapshotTimestamp,
		BlockNumber: snapshotBlock,
		Reserves:    entity.PoolReserves{"154962484450", "199720064300261140443613"},
		Tokens: []*entity.PoolToken{
			{Address: usdc, Decimals: 6, Swappable: true},
			{Address: usdm, Decimals: 18, Swappable: true},
		},
		StaticExtra: `{"dec0":"1000000","dec1":"1000000000000000000"}`,
		Extra:       snapshotExtra,
	}
}

func newSnapshotSim(t *testing.T) *PoolSimulator {
	t.Helper()
	sim, err := NewPoolSimulator(snapshotPool(t))
	require.NoError(t, err)
	return sim
}

// withNow pins the simulator clock for the duration of a test. Tests that
// use it must not run in parallel.
func withNow(t *testing.T, now uint64) {
	t.Helper()
	prev := nowFunc
	nowFunc = func() uint64 { return now }
	t.Cleanup(func() { nowFunc = prev })
}

func calc(sim *PoolSimulator, tokenIn string, amountIn string, tokenOut string) (*pool.CalcAmountOutResult, error) {
	return sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: bignumber.NewBig10(amountIn)},
		TokenOut:      tokenOut,
	})
}

func TestCalcAmountOut_MatchesOnChainGetAmountOut(t *testing.T) {
	withNow(t, snapshotTimestamp)
	sim := newSnapshotSim(t)

	// Values are FPMM.getAmountOut() read at the snapshot block.
	testutil.TestCalcAmountOut(t, sim, map[int]map[int]map[string]string{
		0: {1: {
			"1000000":      "999435542245000000",
			"10000000":     "9994355422450000000",
			"100000000":    "99943554224500000000",
			"1000000000":   "999435542245000000000",
			"10000000000":  "9994355422450000000000",
			"100000000000": "99943554224500000000000",
		}},
		1: {0: {
			"1000000000000000000":      "999564",
			"10000000000000000000":     "9995644",
			"100000000000000000000":    "99956446",
			"1000000000000000000000":   "999564461",
			"10000000000000000000000":  "9995644619",
			"100000000000000000000000": "99956446191",
		}},
	})
}

func TestCalcAmountOut_FeeAndSwapInfo(t *testing.T) {
	withNow(t, snapshotTimestamp)
	sim := newSnapshotSim(t)

	res, err := calc(sim, usdc, "1000000", usdm)
	require.NoError(t, err)
	assert.Equal(t, usdm, res.Fee.Token)
	// gross 999935510000000000 - net 999435542245000000
	assert.Equal(t, "499967755000000", res.Fee.Amount.String())
	assert.Equal(t, defaultGas, res.Gas)

	si, ok := res.SwapInfo.(SwapInfo)
	require.True(t, ok)
	// token0 (USDC, 6 dec): scaledIn = 1e15, minus 5 bps = 999500000000000.
	// The 5-minute window had expired (now > lu0 + 300), so netflow0 resets
	// and lastUpdated0 moves to now; the 1-day window is still open.
	assert.Equal(t, "999500000000000", si.Limits[0].Netflow0.Dec())
	assert.Equal(t, uint32(snapshotTimestamp), si.Limits[0].LastUpdated0)
	assert.Equal(t, "-61512618000000000", si.Limits[0].Netflow1.Dec())
	assert.Equal(t, uint32(1790052298), si.Limits[0].LastUpdated1)
	// token1 (USDm, 18 dec): scaledOut = 999435542245000 flows out.
	assert.Equal(t, "-999435542245000", si.Limits[1].Netflow0.Dec())
	assert.Equal(t, "61507495357755000", si.Limits[1].Netflow1.Dec())

	// CalcAmountOut must not have touched the simulator's own state.
	assert.Equal(t, "-62512118000000000", sim.limits[0].Netflow0.Dec())
	assert.Equal(t, uint32(1790074988), sim.limits[0].LastUpdated0)
}

func TestCalcAmountOut_Guards(t *testing.T) {
	withNow(t, snapshotTimestamp)

	t.Run("invalid tokens", func(t *testing.T) {
		sim := newSnapshotSim(t)
		_, err := calc(sim, usdc, "1000000", usdc)
		assert.ErrorIs(t, err, ErrInvalidToken)
		_, err = calc(sim, "0x00000000000000000000000000000000000000ff", "1000000", usdm)
		assert.ErrorIs(t, err, ErrInvalidToken)
	})
	t.Run("zero amount", func(t *testing.T) {
		_, err := calc(newSnapshotSim(t), usdc, "0", usdm)
		assert.ErrorIs(t, err, ErrZeroAmountIn)
	})
	t.Run("dust rounds to zero output", func(t *testing.T) {
		// 1 wei of USDm is worth < 1 unit of USDC.
		_, err := calc(newSnapshotSim(t), usdm, "1", usdc)
		assert.ErrorIs(t, err, ErrZeroAmountOut)
	})
	t.Run("amount out must be below reserve", func(t *testing.T) {
		sim := newSnapshotSim(t)
		// 200k USDm -> ~199.9k USDC > reserve0 (154,962 USDC)
		_, err := calc(sim, usdm, "200000000000000000000000", usdc)
		assert.ErrorIs(t, err, ErrInsufficientLiquidity)
		// exactly reserve1 out: 199720064300261140443613 USDm needs
		// amountIn = out * 1e6 * 1e4 / (N * 9995) rounded up; use a value
		// that maps to >= reserve1 and one that maps below.
		_, err = calc(sim, usdc, "199833000000", usdm)
		assert.ErrorIs(t, err, ErrInsufficientLiquidity)
		res, err := calc(sim, usdc, "199000000000", usdm)
		require.NoError(t, err)
		assert.Equal(t, 1, sim.Info.Reserves[1].Cmp(res.TokenAmountOut.Amount))
	})
	t.Run("unquoteable", func(t *testing.T) {
		ep := snapshotPool(t)
		ep.Extra = `{"rateNum":"1","rateDen":"1","unquoteable":true}`
		sim, err := NewPoolSimulator(ep)
		require.NoError(t, err)
		_, err = calc(sim, usdc, "1000000", usdm)
		assert.ErrorIs(t, err, ErrUnquoteable)
	})
	t.Run("trading suspended", func(t *testing.T) {
		sim := newSnapshotSim(t)
		sim.tradingMode = 1
		_, err := calc(sim, usdc, "1000000", usdm)
		assert.ErrorIs(t, err, ErrTradingSuspended)
	})
	t.Run("zero rate", func(t *testing.T) {
		sim := newSnapshotSim(t)
		sim.rateNumerator = new(uint256.Int)
		_, err := calc(sim, usdc, "1000000", usdm)
		assert.ErrorIs(t, err, ErrInvalidRate)
	})
}

func TestCalcAmountOut_RateStaleness(t *testing.T) {
	expiry := uint64(rateTimestamp + rateExpiry)

	withNow(t, expiry-rateStalenessBufferSeconds) // now + buffer == expiry: still valid
	_, err := calc(newSnapshotSim(t), usdc, "1000000", usdm)
	require.NoError(t, err)

	withNow(t, expiry-rateStalenessBufferSeconds+1) // inside the safety buffer
	_, err = calc(newSnapshotSim(t), usdc, "1000000", usdm)
	assert.ErrorIs(t, err, ErrNoRecentRate)

	withNow(t, expiry+1) // expired on-chain too
	_, err = calc(newSnapshotSim(t), usdc, "1000000", usdm)
	assert.ErrorIs(t, err, ErrNoRecentRate)
}

func TestCalcAmountOut_MarketHours(t *testing.T) {
	saturday := ts("2026-09-26T12:00:00Z")
	// keep the rate fresh at the pinned clock
	fresh := func(t *testing.T) *PoolSimulator {
		sim := newSnapshotSim(t)
		sim.rateTimestamp = saturday
		return sim
	}

	withNow(t, saturday)
	sim := fresh(t)
	_, err := calc(sim, usdc, "1000000", usdm) // stablecoin pool: not enforced
	require.NoError(t, err)

	sim = fresh(t)
	sim.enforceMarketHours = true
	_, err = calc(sim, usdc, "1000000", usdm)
	assert.ErrorIs(t, err, ErrFXMarketClosed)

	withNow(t, ts("2026-09-28T12:00:00Z")) // Monday
	sim = fresh(t)
	sim.enforceMarketHours = true
	sim.rateTimestamp = ts("2026-09-28T12:00:00Z")
	_, err = calc(sim, usdc, "1000000", usdm)
	require.NoError(t, err)
}

// limitedPool is the snapshot with tight trading limits: 100 tokens per
// 5 minutes and 1000 per day on both tokens, with 90 tokens of net inflow
// already recorded on USDC and 90 of net outflow on USDm in the current
// 5-minute window.
func limitedPool(t *testing.T, now uint64, lastUpdated0 uint32) *PoolSimulator {
	t.Helper()
	ep := snapshotPool(t)
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(snapshotExtra), &extra))
	extra.RateTimestamp = now
	for i := range extra.Limits {
		extra.Limits[i].Limit0 = int256.MustFromDec("100000000000000000")  // 100e15
		extra.Limits[i].Limit1 = int256.MustFromDec("1000000000000000000") // 1000e15
		extra.Limits[i].LastUpdated0 = lastUpdated0
		extra.Limits[i].LastUpdated1 = lastUpdated0
	}
	extra.Limits[0].Netflow0 = int256.MustFromDec("90000000000000000")
	extra.Limits[0].Netflow1 = int256.MustFromDec("90000000000000000")
	extra.Limits[1].Netflow0 = int256.MustFromDec("-90000000000000000")
	extra.Limits[1].Netflow1 = int256.MustFromDec("-90000000000000000")
	b, err := json.Marshal(extra)
	require.NoError(t, err)
	ep.Extra = string(b)
	sim, err := NewPoolSimulator(ep)
	require.NoError(t, err)
	return sim
}

func TestCalcAmountOut_TradingLimits(t *testing.T) {
	now := uint64(snapshotTimestamp)
	withNow(t, now)

	t.Run("L0 exceeded on input token within window", func(t *testing.T) {
		sim := limitedPool(t, now, uint32(now-100))
		// 20 USDC in: net 19.99 -> 109.99 > 100
		_, err := calc(sim, usdc, "20000000", usdm)
		assert.ErrorIs(t, err, ErrL0LimitExceeded)
		// 9 USDC in: net 8.9955 -> 98.9955 <= 100
		_, err = calc(sim, usdc, "9000000", usdm)
		require.NoError(t, err)
	})
	t.Run("L0 exceeded on output token within window", func(t *testing.T) {
		sim := limitedPool(t, now, uint32(now-100))
		// 20 USDm in -> ~19.99 USDC out; USDC netflow +90 + 0 - 19.99 fine,
		// USDm netflow -90 + 19.99 fine -> passes. Push USDm out instead:
		// 20 USDC in -> 19.99 USDm out: USDm netflow -90 - 19.99 < -100.
		sim.limits[0].Netflow0 = new(int256.Int) // isolate the output side
		_, err := calc(sim, usdc, "20000000", usdm)
		assert.ErrorIs(t, err, ErrL0LimitExceeded)
	})
	t.Run("window expired resets netflow", func(t *testing.T) {
		sim := limitedPool(t, now, uint32(now-limitWindow0-1))
		res, err := calc(sim, usdc, "20000000", usdm)
		require.NoError(t, err)
		si := res.SwapInfo.(SwapInfo)
		assert.Equal(t, "19990000000000000", si.Limits[0].Netflow0.Dec())
		assert.Equal(t, uint32(now), si.Limits[0].LastUpdated0)
		// L1 window (1 day) is still open: 90 + 19.99
		assert.Equal(t, "109990000000000000", si.Limits[0].Netflow1.Dec())
	})
	t.Run("L1 exceeded", func(t *testing.T) {
		sim := limitedPool(t, now, uint32(now-limitWindow0-1))
		sim.limits[0].Netflow1 = int256.MustFromDec("990000000000000000") // 990 of 1000
		_, err := calc(sim, usdc, "20000000", usdm)
		assert.ErrorIs(t, err, ErrL1LimitExceeded)
	})
	t.Run("sequential swaps accumulate through UpdateBalance", func(t *testing.T) {
		sim := limitedPool(t, now, uint32(now-100))
		sim.limits[0].Netflow0 = new(int256.Int)
		sim.limits[1].Netflow0 = new(int256.Int)
		// 60 USDC each: first ok (59.97), second ok (119.94 > 100 -> fails)
		testutil.TestCalcAmountOutWithUpdateBalance(t, sim, map[int]map[int][][][2]string{
			0: {1: {{
				{"60000000", "59966132534700000000"},
				{"60000000", ""},
			}}},
		})
	})
}

func TestUpdateBalance(t *testing.T) {
	withNow(t, snapshotTimestamp)
	sim := newSnapshotSim(t)

	res, err := calc(sim, usdc, "1000000", usdm)
	require.NoError(t, err)
	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: usdc, Amount: big.NewInt(1_000_000)},
		TokenAmountOut: *res.TokenAmountOut,
		Fee:            *res.Fee,
		SwapInfo:       res.SwapInfo,
	})

	// reserve0 grows by amountIn minus the 2 bps protocol fee (200 units).
	assert.Equal(t, "154963484250", sim.Info.Reserves[0].String())
	assert.Equal(t, "199719064864718895443613", sim.Info.Reserves[1].String())
	assert.Equal(t, "999500000000000", sim.limits[0].Netflow0.Dec())
	assert.Equal(t, "-999435542245000", sim.limits[1].Netflow0.Dec())

	// price is fixed: the next quote is identical
	res2, err := calc(sim, usdc, "1000000", usdm)
	require.NoError(t, err)
	assert.Equal(t, res.TokenAmountOut.Amount, res2.TokenAmountOut.Amount)
}

func TestCloneState(t *testing.T) {
	withNow(t, snapshotTimestamp)
	sim := newSnapshotSim(t)
	testutil.TestCloneState(t, sim, pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: usdc, Amount: big.NewInt(1_000_000)},
		TokenOut:      usdm,
	}, nil)

	clone := sim.CloneState().(*PoolSimulator)
	res, err := calc(clone, usdc, "1000000", usdm)
	require.NoError(t, err)
	clone.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: usdc, Amount: big.NewInt(1_000_000)},
		TokenAmountOut: *res.TokenAmountOut,
		Fee:            *res.Fee,
		SwapInfo:       res.SwapInfo,
	})
	assert.Equal(t, "154962484450", sim.Info.Reserves[0].String())
	assert.Equal(t, "-62512118000000000", sim.limits[0].Netflow0.Dec())
	assert.Equal(t, "999500000000000", clone.limits[0].Netflow0.Dec())
}

func TestNewPoolSimulator_RejectsUntrackedPool(t *testing.T) {
	t.Parallel()
	ep := snapshotPool(t)
	ep.Extra = ""
	_, err := NewPoolSimulator(ep)
	require.Error(t, err)
	ep.Extra = `{"lpFee":3}`
	_, err = NewPoolSimulator(ep)
	require.Error(t, err)
}

func TestExtraJSONRoundTrip(t *testing.T) {
	t.Parallel()
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(snapshotExtra), &extra))
	b, err := json.Marshal(extra)
	require.NoError(t, err)
	var again Extra
	require.NoError(t, json.Unmarshal(b, &again))
	assert.Equal(t, extra.RateNumerator.Dec(), again.RateNumerator.Dec())
	assert.Equal(t, extra.Limits[0].Netflow0.Dec(), again.Limits[0].Netflow0.Dec())
	assert.Equal(t, "-62512118000000000", again.Limits[0].Netflow0.Dec())
	assert.Equal(t, extra.Limits[1].Limit1.Dec(), again.Limits[1].Limit1.Dec())
}

func TestGetMetaInfo(t *testing.T) {
	t.Parallel()
	sim := newSnapshotSim(t)
	assert.Equal(t, MetaInfo{BlockNumber: snapshotBlock}, sim.GetMetaInfo(usdc, usdm))
}
