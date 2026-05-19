package fermi

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const testStaticExtra = `{"fS":"0xb1076fe3ab5e28005c7c323bac5ac06a680d452e","t0":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","t1":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","d0":18,"d1":6}`

const testExtra = `{
  "bn":25090812,
  "curve": {
    "mp":"225323000000",
    "fb":0,
    "sf":0,
    "sd":"10000000000000000000",
    "max":"100000000000000000000",
    "ds0":"1000000000000000000",
    "ds1":"1000000",
    "sp":[{"xl":"0","xh":"1000000000000000000","c0":"0","c1":"0","c2":"0","c3":"0"}],
    "ip":[{"xl":"0","xh":"1000000000000000000","c0":"0","c1":"0","c2":"0","c3":"0"}],
    "vr0":"5776727482444948532",
    "vr1":"21789832560"
  }
}`

const testExtraNoCurve = `{"bn":25090812}`

// Production pool state at block 25120997 (WETH/USDC); midPrice absent in
// realPoolExtra (Titan was unavailable at snapshot time).
const realPoolExtra = `{"bn":25120997,"curve":{"mp":"","fb":0,"sf":5000,"sd":"1000000000000","max":"30000000000","ds0":"1000000000000000000","ds1":"1000000","sp":[{"xl":"10000000000","xh":"1000000000000000","c0":"0","c1":"0","c2":"1","c3":"0"},{"xl":"1000000000000000","xh":"10000000000000000","c0":"0","c1":"0","c2":"2","c3":"1"},{"xl":"10000000000000000","xh":"30000000000000000","c0":"0","c1":"0","c2":"2","c3":"3"},{"xl":"30000000000000000","xh":"100000000000000000","c0":"0","c1":"0","c2":"5","c3":"5"},{"xl":"100000000000000000","xh":"300000000000000000","c0":"0","c1":"0","c2":"30","c3":"10"},{"xl":"300000000000000000","xh":"500000000000000000","c0":"0","c1":"0","c2":"30","c3":"40"},{"xl":"500000000000000000","xh":"1000000000000000000","c0":"0","c1":"0","c2":"30","c3":"70"}],"ip":[{"xl":"-1000000000000000000","xh":"-500000000000000000","c0":"0","c1":"0","c2":"-3","c3":"4"},{"xl":"-500000000000000000","xh":"0","c0":"0","c1":"0","c2":"-1","c3":"1"},{"xl":"0","xh":"500000000000000000","c0":"0","c1":"0","c2":"1","c3":"0"},{"xl":"500000000000000000","xh":"1000000000000000000","c0":"0","c1":"0","c2":"-5","c3":"1"}],"vr0":"943000000000000","vr1":"61298294372"}}`

// realPoolExtraWithMid injects a realistic midPrice (~$2120) into realPoolExtra.
const realPoolExtraWithMid = `{"bn":25120997,"curve":{"mp":"212000000000","fb":0,"sf":5000,"sd":"1000000000000","max":"30000000000","ds0":"1000000000000000000","ds1":"1000000","sp":[{"xl":"10000000000","xh":"1000000000000000","c0":"0","c1":"0","c2":"1","c3":"0"},{"xl":"1000000000000000","xh":"10000000000000000","c0":"0","c1":"0","c2":"2","c3":"1"},{"xl":"10000000000000000","xh":"30000000000000000","c0":"0","c1":"0","c2":"2","c3":"3"},{"xl":"30000000000000000","xh":"100000000000000000","c0":"0","c1":"0","c2":"5","c3":"5"},{"xl":"100000000000000000","xh":"300000000000000000","c0":"0","c1":"0","c2":"30","c3":"10"},{"xl":"300000000000000000","xh":"500000000000000000","c0":"0","c1":"0","c2":"30","c3":"40"},{"xl":"500000000000000000","xh":"1000000000000000000","c0":"0","c1":"0","c2":"30","c3":"70"}],"ip":[{"xl":"-1000000000000000000","xh":"-500000000000000000","c0":"0","c1":"0","c2":"-3","c3":"4"},{"xl":"-500000000000000000","xh":"0","c0":"0","c1":"0","c2":"-1","c3":"1"},{"xl":"0","xh":"500000000000000000","c0":"0","c1":"0","c2":"1","c3":"0"},{"xl":"500000000000000000","xh":"1000000000000000000","c0":"0","c1":"0","c2":"-5","c3":"1"}],"vr0":"943000000000000","vr1":"61298294372"}}`

const realPoolStaticExtra = `{"fS":"0xb1076fe3ab5e28005c7c323bac5ac06a680d452e","t0":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","t1":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","d0":18,"d1":6}`

const (
	wethAddr = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
	usdcAddr = "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
)

func makeTestPool(extra, staticExtra string) entity.Pool {
	return entity.Pool{
		Address:     "0xb1076fe3ab5e28005c7c323bac5ac06a680d452e_" + wethAddr + "_" + usdcAddr,
		Exchange:    DexType,
		Type:        DexType,
		Reserves:    entity.PoolReserves{"5000000000000000000", "10000000000"},
		Extra:       extra,
		StaticExtra: staticExtra,
		Tokens: []*entity.PoolToken{
			{Address: wethAddr, Decimals: 18, Swappable: true},
			{Address: usdcAddr, Decimals: 6, Swappable: true},
		},
	}
}

func makeRealPool(extra string) entity.Pool {
	return entity.Pool{
		Address:     "0xb1076fe3ab5e28005c7c323bac5ac06a680d452e_" + wethAddr + "_" + usdcAddr,
		Exchange:    DexType,
		Type:        DexType,
		Reserves:    entity.PoolReserves{"943000000000000", "61298294372"},
		Extra:       extra,
		StaticExtra: realPoolStaticExtra,
		Tokens: []*entity.PoolToken{
			{Address: wethAddr, Decimals: 18, Swappable: true},
			{Address: usdcAddr, Decimals: 6, Swappable: true},
		},
		BlockNumber: 25120997,
	}
}

// makeRealPoolWithVault clones realPoolExtraWithMid with custom vault balances.
func makeRealPoolWithVault(t *testing.T, vr0, vr1 string) entity.Pool {
	t.Helper()
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(realPoolExtraWithMid), &extra))
	extra.Curve.VaultReserve0 = vr0
	extra.Curve.VaultReserve1 = vr1
	b, _ := json.Marshal(extra)

	return entity.Pool{
		Address:     "fermi-swaplimit-test",
		Exchange:    DexType,
		Type:        DexType,
		StaticExtra: `{"fS":"0xb1076fe3ab5e28005c7c323bac5ac06a680d452e","t0":"` + wethAddr + `","t1":"` + usdcAddr + `"}`,
		Tokens: []*entity.PoolToken{
			{Address: wethAddr, Swappable: true},
			{Address: usdcAddr, Swappable: true},
		},
		Extra:    string(b),
		Reserves: entity.PoolReserves{vr0, vr1},
	}
}

type stubSwapLimit struct{ balances map[string]*big.Int }

func newStubSwapLimit(balances map[string]*big.Int) *stubSwapLimit {
	cp := make(map[string]*big.Int, len(balances))
	for k, v := range balances {
		cp[k] = new(big.Int).Set(v)
	}
	return &stubSwapLimit{balances: cp}
}

func (s *stubSwapLimit) Clone() pool.SwapLimit           { return newStubSwapLimit(s.balances) }
func (s *stubSwapLimit) GetExchange() string             { return DexType }
func (s *stubSwapLimit) GetLimit(k string) *big.Int      { return s.balances[k] }
func (s *stubSwapLimit) GetSwapped() map[string]*big.Int { return nil }
func (s *stubSwapLimit) GetAllowedSenders() string       { return "" }
func (s *stubSwapLimit) UpdateLimit(decKey, incKey string, decDelta, incDelta *big.Int) (*big.Int, *big.Int, error) {
	if s.balances[decKey] != nil {
		s.balances[decKey] = new(big.Int).Sub(s.balances[decKey], decDelta)
	}
	if s.balances[incKey] != nil {
		s.balances[incKey] = new(big.Int).Add(s.balances[incKey], incDelta)
	}
	return s.balances[incKey], s.balances[decKey], nil
}

func TestNewPoolSimulator(t *testing.T) {
	sim, err := NewPoolSimulator(makeTestPool(testExtra, testStaticExtra))
	require.NoError(t, err)
	assert.NotNil(t, sim.curve)
	assert.Equal(t, uint64(25090812), sim.blockNumber)
}

func TestCalcAmountOut_Bid_Curve(t *testing.T) {
	// 1 WETH → USDC at flat midPrice 2253.23, zero spread.
	// out = 1e18 * 1e6 * 225323000000 / (1e8 * 1e18) = 2253230000
	sim, err := NewPoolSimulator(makeTestPool(testExtra, testStaticExtra))
	require.NoError(t, err)

	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: wethAddr, Amount: big.NewInt(1_000_000_000_000_000_000)},
		TokenOut:      usdcAddr,
	})
	require.NoError(t, err)
	assert.Equal(t, "2253230000", res.TokenAmountOut.Amount.String())
	assert.Equal(t, int64(defaultGas), res.Gas)
}

func TestCalcAmountOut_Ask_Curve(t *testing.T) {
	// 1000 USDC → WETH, out ≈ 0.4438 WETH
	sim, err := NewPoolSimulator(makeTestPool(testExtra, testStaticExtra))
	require.NoError(t, err)

	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: usdcAddr, Amount: big.NewInt(1_000_000_000)},
		TokenOut:      wethAddr,
	})
	require.NoError(t, err)
	assert.Equal(t, "443807334360007633", res.TokenAmountOut.Amount.String())
}

func TestCalcAmountOut_NoCurve(t *testing.T) {
	sim, err := NewPoolSimulator(makeTestPool(testExtraNoCurve, testStaticExtra))
	require.NoError(t, err)
	require.Nil(t, sim.curve)

	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: wethAddr, Amount: big.NewInt(1_000_000_000_000_000_000)},
		TokenOut:      usdcAddr,
	})
	assert.ErrorIs(t, err, ErrCurveNotAvailable)
}

func TestCalcAmountOut_Bid_AboveMax(t *testing.T) {
	// 15 WETH → ~31 800 USDC sizeInput > 30 000 cap → ErrInsufficientLiquidity
	sim, err := NewPoolSimulator(makeRealPool(realPoolExtraWithMid))
	require.NoError(t, err)

	amtIn := new(big.Int).Mul(big.NewInt(15), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: wethAddr, Amount: amtIn},
		TokenOut:      usdcAddr,
	})
	assert.ErrorIs(t, err, ErrInsufficientLiquidity)
}

func TestCalcAmountOut_Ask_AboveMax(t *testing.T) {
	// 40 000 USDC > maxAmountIn 30 000 → ErrInsufficientLiquidity
	sim, err := NewPoolSimulator(makeRealPool(realPoolExtraWithMid))
	require.NoError(t, err)

	amtIn := new(big.Int).Mul(big.NewInt(40000), new(big.Int).Exp(big.NewInt(10), big.NewInt(6), nil))
	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: usdcAddr, Amount: amtIn},
		TokenOut:      wethAddr,
	})
	assert.ErrorIs(t, err, ErrInsufficientLiquidity)
}

func TestCalcAmountOut_InvalidToken(t *testing.T) {
	sim, err := NewPoolSimulator(makeTestPool(testExtra, testStaticExtra))
	require.NoError(t, err)

	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xdeadbeef", Amount: big.NewInt(1e9)},
		TokenOut:      usdcAddr,
	})
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestCalcAmountOut_ZeroAmountIn(t *testing.T) {
	sim, err := NewPoolSimulator(makeTestPool(testExtra, testStaticExtra))
	require.NoError(t, err)

	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: wethAddr, Amount: big.NewInt(0)},
		TokenOut:      usdcAddr,
	})
	assert.ErrorIs(t, err, ErrInvalidAmountIn)
}

func TestCalcAmountOut_NilAmountIn(t *testing.T) {
	sim, err := NewPoolSimulator(makeTestPool(testExtra, testStaticExtra))
	require.NoError(t, err)

	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: wethAddr, Amount: nil},
		TokenOut:      usdcAddr,
	})
	assert.ErrorIs(t, err, ErrInvalidAmountIn)
}

func TestCalcAmountOut_NegativeAmountIn(t *testing.T) {
	sim, err := NewPoolSimulator(makeTestPool(testExtra, testStaticExtra))
	require.NoError(t, err)

	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: wethAddr, Amount: big.NewInt(-1)},
		TokenOut:      usdcAddr,
	})
	assert.ErrorIs(t, err, ErrInvalidAmountIn)
}

func TestCalcAmountOut_FeeIsZero(t *testing.T) {
	// Fee is embedded in the quote — CalcAmountOut must report fee = 0.
	sim, err := NewPoolSimulator(makeTestPool(testExtra, testStaticExtra))
	require.NoError(t, err)

	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: wethAddr, Amount: big.NewInt(1e18)},
		TokenOut:      usdcAddr,
	})
	require.NoError(t, err)
	assert.Equal(t, big.NewInt(0), res.Fee.Amount)
	assert.Equal(t, wethAddr, res.Fee.Token)
}

func TestCalcAmountOut_Monotonicity(t *testing.T) {
	sim, err := NewPoolSimulator(makeTestPool(testExtra, testStaticExtra))
	require.NoError(t, err)

	sizes := []int64{
		100_000_000_000_000_000,
		500_000_000_000_000_000,
		1_000_000_000_000_000_000,
		2_000_000_000_000_000_000,
	}
	prev := big.NewInt(0)
	for _, s := range sizes {
		res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: wethAddr, Amount: big.NewInt(s)},
			TokenOut:      usdcAddr,
		})
		require.NoError(t, err, "amtIn=%d", s)
		assert.True(t, res.TokenAmountOut.Amount.Cmp(prev) > 0, "amtIn=%d", s)
		prev = res.TokenAmountOut.Amount
	}
}

// ---- UpdateBalance / CloneState / meta ----

func TestUpdateBalance_NoOp(t *testing.T) {
	sim, err := NewPoolSimulator(makeTestPool(testExtra, testStaticExtra))
	require.NoError(t, err)

	r0Before := new(big.Int).Set(sim.Info.Reserves[0])
	r1Before := new(big.Int).Set(sim.Info.Reserves[1])

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: wethAddr, Amount: big.NewInt(1e18)},
		TokenAmountOut: pool.TokenAmount{Token: usdcAddr, Amount: big.NewInt(2253230000)},
		Fee:            pool.TokenAmount{Token: wethAddr, Amount: big.NewInt(0)},
	})

	assert.Equal(t, 0, sim.Info.Reserves[0].Cmp(r0Before))
	assert.Equal(t, 0, sim.Info.Reserves[1].Cmp(r1Before))
}

func TestCloneState(t *testing.T) {
	sim, err := NewPoolSimulator(makeTestPool(testExtra, testStaticExtra))
	require.NoError(t, err)

	cloned := sim.CloneState()
	require.NotNil(t, cloned)

	origRes, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: wethAddr, Amount: big.NewInt(1e18)},
		TokenOut:      usdcAddr,
	})
	require.NoError(t, err)

	clonedRes, err := cloned.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: wethAddr, Amount: big.NewInt(1e18)},
		TokenOut:      usdcAddr,
	})
	require.NoError(t, err)

	assert.Equal(t, origRes.TokenAmountOut.Amount.String(), clonedRes.TokenAmountOut.Amount.String())
}

func TestGetMetaInfo(t *testing.T) {
	sim, err := NewPoolSimulator(makeTestPool(testExtra, testStaticExtra))
	require.NoError(t, err)

	meta, ok := sim.GetMetaInfo("any", "any").(PoolMeta)
	require.True(t, ok)
	assert.Equal(t, "0xb1076fe3ab5e28005c7c323bac5ac06a680d452e", meta.FermiSwapper)
	assert.Equal(t, uint64(25090812), meta.BlockNumber)
}

// ---- evalCurveAmountOut direct tests ----

func TestEvalCurveAmountOut_NoCurve(t *testing.T) {
	sim, err := NewPoolSimulator(makeTestPool(testExtraNoCurve, testStaticExtra))
	require.NoError(t, err)
	require.Nil(t, sim.curve)

	_, err = sim.evalCurveAmountOut(wethAddr, big.NewInt(1_000_000_000_000_000_000), nil)
	assert.ErrorIs(t, err, ErrCurveNotAvailable)
}

func TestEvalCurveAmountOut_FlatBid(t *testing.T) {
	// out = 1e18 * 1e6 * 225323000000 / (1e8 * 1e18) = 2253230000
	sim, err := NewPoolSimulator(makeTestPool(testExtra, testStaticExtra))
	require.NoError(t, err)

	res, err := sim.evalCurveAmountOut(wethAddr, big.NewInt(1_000_000_000_000_000_000), nil)
	require.NoError(t, err)
	assert.Equal(t, "2253230000", res.TokenAmountOut.Amount.String())
}

func TestEvalCurveAmountOut_FlatAsk(t *testing.T) {
	// 1000 USDC → ≈ 0.4438 WETH
	sim, err := NewPoolSimulator(makeTestPool(testExtra, testStaticExtra))
	require.NoError(t, err)

	res, err := sim.evalCurveAmountOut(usdcAddr, big.NewInt(1_000_000_000), nil)
	require.NoError(t, err)
	assert.Equal(t, "443807334360007633", res.TokenAmountOut.Amount.String())
}

func TestEvalCurveAmountOut_BidSweep(t *testing.T) {
	sim, err := NewPoolSimulator(makeTestPool(testExtra, testStaticExtra))
	require.NoError(t, err)

	sizes := []string{
		"10000000000000000",
		"50000000000000000",
		"100000000000000000",
		"500000000000000000",
		"1000000000000000000",
		"2000000000000000000",
		"5000000000000000000",
	}

	prev := big.NewInt(0)
	for _, sz := range sizes {
		res, err := sim.evalCurveAmountOut(wethAddr, bi(sz), nil)
		require.NoError(t, err, "size=%s", sz)
		assert.True(t, res.TokenAmountOut.Amount.Cmp(prev) > 0, "size=%s", sz)
		prev = res.TokenAmountOut.Amount
	}
}

// ---- production snapshot tests (block 25120997) ----

func TestReal_NoMidPrice_Bid(t *testing.T) {
	sim, err := NewPoolSimulator(makeRealPool(realPoolExtra))
	require.NoError(t, err)

	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: wethAddr, Amount: big.NewInt(1_000_000_000_000_000_000)},
		TokenOut:      usdcAddr,
	})
	assert.ErrorIs(t, err, ErrInvalidCurveData, "empty midPrice should fail cleanly")
}

func TestReal_NoMidPrice_Ask(t *testing.T) {
	sim, err := NewPoolSimulator(makeRealPool(realPoolExtra))
	require.NoError(t, err)

	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: usdcAddr, Amount: big.NewInt(1_000_000_000)},
		TokenOut:      wethAddr,
	})
	assert.ErrorIs(t, err, ErrInvalidCurveData, "empty midPrice should fail cleanly")
}

func TestReal_Bid_WETH_to_USDC(t *testing.T) {
	sim, err := NewPoolSimulator(makeRealPool(realPoolExtraWithMid))
	require.NoError(t, err)

	tests := []struct {
		label    string
		amountIn *big.Int
		expected string
	}{
		{"0.01_WETH", big.NewInt(10_000_000_000_000_000), "21199999"},
		{"0.1_WETH", big.NewInt(100_000_000_000_000_000), "211999047"},
		{"1_WETH", big.NewInt(1_000_000_000_000_000_000), "2119993025"},
	}
	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: wethAddr, Amount: new(big.Int).Set(tc.amountIn)},
				TokenOut:      usdcAddr,
			})
			require.NoError(t, err)
			assert.Equal(t, tc.expected, res.TokenAmountOut.Amount.String())
		})
	}
}

func TestReal_Ask_USDC_to_WETH(t *testing.T) {
	sim, err := NewPoolSimulator(makeRealPool(realPoolExtraWithMid))
	require.NoError(t, err)

	tests := []struct {
		label    string
		amountIn *big.Int
		expected string
	}{
		{"100_USDC", big.NewInt(100_000_000), "47169764159445515"},
		{"1000_USDC", big.NewInt(1_000_000_000), "471650943396698066"},
		{"5000_USDC", big.NewInt(5_000_000_000), "2358376685567550119"},
	}
	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: usdcAddr, Amount: new(big.Int).Set(tc.amountIn)},
				TokenOut:      wethAddr,
			})
			require.NoError(t, err)
			assert.Equal(t, tc.expected, res.TokenAmountOut.Amount.String())
		})
	}
}

// TestUpdateBalance_MutatesVaultLocally confirms a second CalcAmountOut after
// UpdateBalance sees a shifted inventory ratio.
func TestUpdateBalance_MutatesVaultLocally(t *testing.T) {
	ep := makeRealPoolWithVault(t, "10000000000000000000", "20000000000")
	sim, err := NewPoolSimulator(ep)
	require.NoError(t, err)

	r1, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: wethAddr, Amount: big.NewInt(1_000_000_000_000_000_000)},
		TokenOut:      usdcAddr,
	})
	require.NoError(t, err)
	out1 := new(big.Int).Set(r1.TokenAmountOut.Amount)

	clone := sim.CloneState().(*PoolSimulator)
	clone.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: wethAddr, Amount: big.NewInt(1_000_000_000_000_000_000)},
		TokenAmountOut: pool.TokenAmount{Token: usdcAddr, Amount: out1},
	})

	r2, err := clone.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: wethAddr, Amount: big.NewInt(1_000_000_000_000_000_000)},
		TokenOut:      usdcAddr,
	})
	require.NoError(t, err)
	require.NotEqual(t, out1.String(), r2.TokenAmountOut.Amount.String(),
		"UpdateBalance must shift vault so next quote differs")

	// Original must be unaffected.
	r3, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: wethAddr, Amount: big.NewInt(1_000_000_000_000_000_000)},
		TokenOut:      usdcAddr,
	})
	require.NoError(t, err)
	assert.Equal(t, out1.String(), r3.TokenAmountOut.Amount.String(),
		"CloneState must deep-copy Curve so original is unaffected")
}

// TestSwapLimit_GatesOnSharedInventory confirms the limit overrides snapshot
// vault values: near-zero limit must reject the quote.
func TestSwapLimit_GatesOnSharedInventory(t *testing.T) {
	ep := makeRealPoolWithVault(t, "10000000000000000000", "20000000000")
	sim, err := NewPoolSimulator(ep)
	require.NoError(t, err)

	limit := newStubSwapLimit(map[string]*big.Int{
		wethAddr: new(big.Int).Mul(big.NewInt(10), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)),
		usdcAddr: big.NewInt(1), // near-empty
	})
	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: wethAddr, Amount: big.NewInt(1_000_000_000_000_000_000)},
		TokenOut:      usdcAddr,
		Limit:         limit,
	})
	require.ErrorIs(t, err, ErrInsufficientLiquidity)

	// Without limit (falls back to snapshot vault) should succeed.
	r, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: wethAddr, Amount: big.NewInt(1_000_000_000_000_000_000)},
		TokenOut:      usdcAddr,
	})
	require.NoError(t, err)
	require.Greater(t, r.TokenAmountOut.Amount.Sign(), 0)
}

// TestUpdateBalance_WritesToSwapLimit confirms a hop's accounting propagates
// to the shared SwapLimit.
func TestUpdateBalance_WritesToSwapLimit(t *testing.T) {
	ep := makeRealPoolWithVault(t, "10000000000000000000", "20000000000")
	sim, err := NewPoolSimulator(ep)
	require.NoError(t, err)

	limit := newStubSwapLimit(map[string]*big.Int{
		wethAddr: new(big.Int).Mul(big.NewInt(10), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)),
		usdcAddr: big.NewInt(20_000_000_000),
	})
	amtIn := big.NewInt(1_000_000_000_000_000_000)
	amtOut := big.NewInt(2_100_000_000)

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: wethAddr, Amount: amtIn},
		TokenAmountOut: pool.TokenAmount{Token: usdcAddr, Amount: amtOut},
		SwapLimit:      limit,
	})

	wantWETH := new(big.Int).Add(new(big.Int).Mul(big.NewInt(10), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)), amtIn)
	wantUSDC := new(big.Int).Sub(big.NewInt(20_000_000_000), amtOut)
	assert.Equal(t, wantWETH.String(), limit.GetLimit(wethAddr).String())
	assert.Equal(t, wantUSDC.String(), limit.GetLimit(usdcAddr).String())
}

// TestCalculateLimit_ExposesVaultPerToken confirms each pool surfaces vault
// balances per token for router-service's max-merge.
func TestCalculateLimit_ExposesVaultPerToken(t *testing.T) {
	ep := makeRealPoolWithVault(t, "10000000000000000000", "20000000000")
	sim, err := NewPoolSimulator(ep)
	require.NoError(t, err)

	limit := sim.CalculateLimit()
	require.Len(t, limit, 2)
	assert.Equal(t, "10000000000000000000", limit[wethAddr].String())
	assert.Equal(t, "20000000000", limit[usdcAddr].String())
}
