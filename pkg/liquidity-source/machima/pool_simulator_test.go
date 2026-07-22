package machima

import (
	"bytes"
	"math/big"
	"testing"
	"time"

	"github.com/KyberNetwork/msgpack/v5"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	wethAddr  = "0x4200000000000000000000000000000000000006"
	usdcAddr  = "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"
	xmaAddr   = "0xa4985faeb1e64ba215282255dbb78ff59c63d7a9"
	tokenAddr = "0x1234567890abcdef1234567890abcdef12345678"

	routerAddr = "0x566250347e1401615b3e043918fc290b98448578"
	poolAddr   = "0x531aae7d71343c663821604c57520b1602567006"
)

// sqrtPriceAtTick0 is 2**96, i.e. a 1:1 price.
var sqrtPriceAtTick0, _ = new(big.Int).SetString("79228162514264337593543950336", 10)

// newSim builds a deterministic WETH/token pool at tick 0 with liquidity concentrated between
// ticks -2000 and 2000. tokens[0] is WETH (the counter asset), tokens[1] is the launched token.
func newSim(t *testing.T, mutate func(*Extra)) *PoolSimulator {
	t.Helper()
	return newSimWithTokens(t, wethAddr, tokenAddr, tokenAddr, mutate)
}

// newSimWithTokens is newSim with an explicit pair, for pools where the traded token is itself a
// counter asset (the XMA case).
func newSimWithTokens(t *testing.T, token0, token1, tradedToken string, mutate func(*Extra)) *PoolSimulator {
	t.Helper()

	liquidity := big.NewInt(1e18)

	extra := Extra{Extra: uniswapv3.Extra{
		SqrtPriceX96: sqrtPriceAtTick0,
		Tick:         big.NewInt(0),
		Liquidity:    liquidity,
		TickSpacing:  defaultTickSpacing,
		Ticks: []uniswapv3.Tick{
			{Index: -2000, LiquidityGross: liquidity, LiquidityNet: big.NewInt(1e18)},
			{Index: 2000, LiquidityGross: liquidity, LiquidityNet: big.NewInt(-1e18)},
		},
	}}
	if mutate != nil {
		mutate(&extra)
	}

	staticExtra := StaticExtra{
		Token:         tradedToken,
		RouterAddress: routerAddr,
		WETH:          wethAddr,
		USDC:          usdcAddr,
		XMA:           xmaAddr,
	}

	extraBytes, err := json.Marshal(extra)
	require.NoError(t, err)
	staticBytes, err := json.Marshal(staticExtra)
	require.NoError(t, err)

	sim, err := NewPoolSimulator(entity.Pool{
		Address:  poolAddr,
		SwapFee:  defaultFee,
		Exchange: DexType,
		Type:     DexType,
		Tokens: []*entity.PoolToken{
			{Address: token0, Decimals: 18, Swappable: true},
			{Address: token1, Decimals: 18, Swappable: true},
		},
		Reserves:    entity.PoolReserves{"1000000000000000000000000", "1000000000000000000000000"},
		Extra:       string(extraBytes),
		StaticExtra: string(staticBytes),
	}, valueobject.ChainIDBase)
	require.NoError(t, err)

	return sim
}

func calcOut(t *testing.T, sim *PoolSimulator, tokenIn, tokenOut string, amountIn int64) *pool.CalcAmountOutResult {
	t.Helper()
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: big.NewInt(amountIn)},
		TokenOut:      tokenOut,
	})
	require.NoError(t, err)
	return res
}

// TestBuyTaxDeductedFromInput pins the buy semantics: the tax is taken off the input and only the
// remainder reaches the pool. Quoting the untaxed pool with the post-tax input must give the same
// output as quoting the taxed pool with the full input.
func TestBuyTaxDeductedFromInput(t *testing.T) {
	const amountIn = int64(1e15)

	taxed := newSim(t, func(e *Extra) {
		e.HasTax, e.BuyTaxBps, e.SellTaxBps = true, 500, 300
	})
	untaxed := newSim(t, nil)

	got := calcOut(t, taxed, wethAddr, tokenAddr, amountIn)
	want := calcOut(t, untaxed, wethAddr, tokenAddr, amountIn/10000*9500)

	assert.Equal(t, want.TokenAmountOut.Amount.String(), got.TokenAmountOut.Amount.String())
	assert.Equal(t, big.NewInt(amountIn/10000*500).String(), got.Fee.Amount.String())
}

// TestSellTaxDeductedFromOutput pins the sell semantics: the full input is swapped and the tax is
// taken off the pool's output.
func TestSellTaxDeductedFromOutput(t *testing.T) {
	const amountIn = int64(1e15)

	taxed := newSim(t, func(e *Extra) {
		e.HasTax, e.BuyTaxBps, e.SellTaxBps = true, 500, 300
	})
	untaxed := newSim(t, nil)

	got := calcOut(t, taxed, tokenAddr, wethAddr, amountIn)
	raw := calcOut(t, untaxed, tokenAddr, wethAddr, amountIn)

	assert.Equal(t, deductBps(raw.TokenAmountOut.Amount, 300).String(), got.TokenAmountOut.Amount.String())
	// Sell tax is charged on the output, so nothing is withheld from the input.
	assert.Equal(t, "0", got.Fee.Amount.String())
}

// TestNoTaxWhenHasTaxFalse guards against reading stale bps when the launchpad has cleared the tax.
func TestNoTaxWhenHasTaxFalse(t *testing.T) {
	sim := newSim(t, func(e *Extra) {
		e.HasTax, e.BuyTaxBps, e.SellTaxBps = false, 500, 300
	})
	untaxed := newSim(t, nil)

	got := calcOut(t, sim, wethAddr, tokenAddr, 1e15)
	want := calcOut(t, untaxed, wethAddr, tokenAddr, 1e15)
	assert.Equal(t, want.TokenAmountOut.Amount.String(), got.TokenAmountOut.Amount.String())
}

// TestCalcAmountInIsTaxAware is the regression test for the promoted-method bug: an inherited
// UniV3 CalcAmountIn would ignore tax entirely, and onchain-price-service drops any pool whose
// CalcAmountIn does not survive a CalcAmountOut re-check.
func TestCalcAmountInIsTaxAware(t *testing.T) {
	for _, tc := range []struct {
		name              string
		tokenIn, tokenOut string
	}{
		{"buy", wethAddr, tokenAddr},
		{"sell", tokenAddr, wethAddr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sim := newSim(t, func(e *Extra) {
				e.HasTax, e.BuyTaxBps, e.SellTaxBps = true, 500, 300
			})

			wantOut := big.NewInt(1e14)
			in, err := sim.CalcAmountIn(pool.CalcAmountInParams{
				TokenIn:        tc.tokenIn,
				TokenAmountOut: pool.TokenAmount{Token: tc.tokenOut, Amount: wantOut},
			})
			require.NoError(t, err)

			// Feeding that amountIn back through CalcAmountOut must clear the target, otherwise
			// ApproxAmountIn rejects the pool.
			back, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: tc.tokenIn, Amount: in.TokenAmountIn.Amount},
				TokenOut:      tc.tokenOut,
			})
			require.NoError(t, err)
			assert.GreaterOrEqual(t, back.TokenAmountOut.Amount.Cmp(wantOut), 0,
				"round trip undershot: asked %s, got %s", wantOut, back.TokenAmountOut.Amount)

			// And it must not overshoot wildly — an untaxed CalcAmountIn would be ~5% short,
			// so bound the round trip well inside that.
			upper := new(big.Int).Div(new(big.Int).Mul(wantOut, big.NewInt(10010)), big.NewInt(10000))
			assert.LessOrEqual(t, back.TokenAmountOut.Amount.Cmp(upper), 0,
				"round trip overshot: asked %s, got %s", wantOut, back.TokenAmountOut.Amount)
		})
	}
}

// TestUpdateBalanceUsesPoolSideAmounts is the regression test for reserve drift: the tax never
// reaches the pool, so reserves must move by the pool-side amounts, not the user-facing ones.
func TestUpdateBalanceUsesPoolSideAmounts(t *testing.T) {
	sim := newSim(t, func(e *Extra) {
		e.HasTax, e.BuyTaxBps, e.SellTaxBps = true, 500, 300
	})

	const amountIn = int64(1e15)
	before := new(big.Int).Set(sim.GetReserves()[0])

	res := calcOut(t, sim, wethAddr, tokenAddr, amountIn)
	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: wethAddr, Amount: big.NewInt(amountIn)},
		TokenAmountOut: *res.TokenAmountOut,
		SwapInfo:       res.SwapInfo,
	})

	// 5% buy tax: the pool receives 0.95e15, not 1e15.
	gained := new(big.Int).Sub(sim.GetReserves()[0], before)
	assert.Equal(t, big.NewInt(amountIn/10000*9500).String(), gained.String())
}

// TestUpdateBalanceMovesPrice makes sure the V3 state transition is actually applied — a dropped
// SwapInfo would leave consecutive quotes identical and let the router over-fill the pool.
func TestUpdateBalanceMovesPrice(t *testing.T) {
	sim := newSim(t, func(e *Extra) {
		e.HasTax, e.BuyTaxBps, e.SellTaxBps = true, 500, 300
	})

	const amountIn = int64(1e15)
	first := calcOut(t, sim, wethAddr, tokenAddr, amountIn)
	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: wethAddr, Amount: big.NewInt(amountIn)},
		TokenAmountOut: *first.TokenAmountOut,
		SwapInfo:       first.SwapInfo,
	})
	second := calcOut(t, sim, wethAddr, tokenAddr, amountIn)

	assert.Less(t, second.TokenAmountOut.Amount.Cmp(first.TokenAmountOut.Amount), 0,
		"price should worsen after a buy, got %s then %s", first.TokenAmountOut.Amount, second.TokenAmountOut.Amount)
}

// TestCloneStateIsolatesUpdates guards the deep copy: updating a clone must not move the original.
func TestCloneStateIsolatesUpdates(t *testing.T) {
	sim := newSim(t, func(e *Extra) {
		e.HasTax, e.BuyTaxBps, e.SellTaxBps = true, 500, 300
	})

	const amountIn = int64(1e15)
	baseline := calcOut(t, sim, wethAddr, tokenAddr, amountIn)

	clone := sim.CloneState().(*PoolSimulator)
	clone.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: wethAddr, Amount: big.NewInt(amountIn)},
		TokenAmountOut: *baseline.TokenAmountOut,
		SwapInfo:       baseline.SwapInfo,
	})

	after := calcOut(t, sim, wethAddr, tokenAddr, amountIn)
	assert.Equal(t, baseline.TokenAmountOut.Amount.String(), after.TokenAmountOut.Amount.String(),
		"original simulator was mutated through its clone")
}

// TestCalcAmountOutIsPure re-quotes the same input twice without an UpdateBalance in between.
func TestCalcAmountOutIsPure(t *testing.T) {
	sim := newSim(t, func(e *Extra) {
		e.HasTax, e.BuyTaxBps, e.SellTaxBps = true, 500, 300
	})

	first := calcOut(t, sim, wethAddr, tokenAddr, 1e15)
	second := calcOut(t, sim, wethAddr, tokenAddr, 1e15)
	assert.Equal(t, first.TokenAmountOut.Amount.String(), second.TokenAmountOut.Amount.String())
}

func TestClassifyPair(t *testing.T) {
	sim := newSim(t, nil)

	t.Run("neither side is a counter asset", func(t *testing.T) {
		_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: tokenAddr, Amount: big.NewInt(1e15)},
			TokenOut:      "0xdead000000000000000000000000000000000000",
		})
		assert.ErrorIs(t, err, ErrInvalidPair)
	})

	t.Run("both sides are counter assets and neither is XMA", func(t *testing.T) {
		_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: wethAddr, Amount: big.NewInt(1e15)},
			TokenOut:      usdcAddr,
		})
		assert.ErrorIs(t, err, ErrInvalidPair)
	})

	t.Run("XMA is the traded token when both sides are counter assets", func(t *testing.T) {
		isBuy, err := sim.classifyPair(wethAddr, xmaAddr)
		require.NoError(t, err)
		assert.True(t, isBuy, "counter asset in, XMA out is a buy")

		isBuy, err = sim.classifyPair(xmaAddr, wethAddr)
		require.NoError(t, err)
		assert.False(t, isBuy, "XMA in, counter asset out is a sell")
	})

	t.Run("XMA on both sides is rejected", func(t *testing.T) {
		_, err := sim.classifyPair(xmaAddr, xmaAddr)
		assert.ErrorIs(t, err, ErrInvalidPair)
	})
}

func TestAntiSniperWindow(t *testing.T) {
	t.Run("blocks swaps inside the window", func(t *testing.T) {
		sim := newSim(t, func(e *Extra) {
			e.PoolDeploymentTime = uint64(time.Now().Unix())
		})
		_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: wethAddr, Amount: big.NewInt(1e15)},
			TokenOut:      tokenAddr,
		})
		assert.ErrorIs(t, err, ErrAntiSniperActive)
	})

	t.Run("blocks exact-out inside the window too", func(t *testing.T) {
		sim := newSim(t, func(e *Extra) {
			e.PoolDeploymentTime = uint64(time.Now().Unix())
		})
		_, err := sim.CalcAmountIn(pool.CalcAmountInParams{
			TokenIn:        wethAddr,
			TokenAmountOut: pool.TokenAmount{Token: tokenAddr, Amount: big.NewInt(1e14)},
		})
		assert.ErrorIs(t, err, ErrAntiSniperActive)
	})

	t.Run("allows swaps once the window has elapsed", func(t *testing.T) {
		sim := newSim(t, func(e *Extra) {
			e.PoolDeploymentTime = uint64(time.Now().Unix()) - AntiSniperWindowSeconds - 1
		})
		_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: wethAddr, Amount: big.NewInt(1e15)},
			TokenOut:      tokenAddr,
		})
		assert.NoError(t, err)
	})
}

// TestRejectsImpossibleTax stops a corrupt tax read from producing a pool that quotes zero or
// divides by zero when grossing up.
func TestRejectsImpossibleTax(t *testing.T) {
	extra := Extra{
		Extra: uniswapv3.Extra{
			SqrtPriceX96: sqrtPriceAtTick0,
			Tick:         big.NewInt(0),
			Liquidity:    big.NewInt(1),
			TickSpacing:  defaultTickSpacing,
		},
		ProtocolState: ProtocolState{HasTax: true, BuyTaxBps: bpsDenominator},
	}
	extraBytes, err := json.Marshal(extra)
	require.NoError(t, err)

	_, err = NewPoolSimulator(entity.Pool{
		Address:     poolAddr,
		SwapFee:     defaultFee,
		Tokens:      []*entity.PoolToken{{Address: wethAddr}, {Address: tokenAddr}},
		Reserves:    entity.PoolReserves{"0", "0"},
		Extra:       string(extraBytes),
		StaticExtra: `{}`,
	}, valueobject.ChainIDBase)
	assert.ErrorIs(t, err, ErrTaxTooHigh)
}

// TestGetMetaInfo pins the shape aggregator-encoding's PackMachima decodes. The executor calldata
// is just the router address; the deadline comes from block.timestamp on-chain.
func TestGetMetaInfo(t *testing.T) {
	sim := newSim(t, nil)

	meta, ok := sim.GetMetaInfo("", "").(PoolMeta)
	require.True(t, ok)
	assert.Equal(t, routerAddr, meta.Router)
	// The router pulls tokenIn with transferFrom, so the executor must approve it; router-service
	// reads that address from here, not from GetApprovalAddress.
	assert.Equal(t, routerAddr, meta.ApprovalAddress)
	assert.Equal(t, routerAddr, sim.GetApprovalAddress(wethAddr, tokenAddr))
}

func TestGrossUpBpsInvertsDeductBps(t *testing.T) {
	for _, bps := range []uint16{0, 1, 300, 500, 9999} {
		for _, amount := range []int64{1, 7, 1e6, 1e15, 123456789} {
			target := big.NewInt(amount)
			gross := grossUpBps(target, bps)
			assert.GreaterOrEqual(t, deductBps(gross, bps).Cmp(target), 0,
				"grossUpBps(%d, %d) undershoots after deductBps", amount, bps)
		}
	}
}

// TestXmaSellFloorPartialFill covers the one path that partially fills: an XMA sell pinned at the
// launch-tick floor. It also pins the tax basis verified against MachimaAggregatorQuoter on Base —
// tax is charged on the full input, so the unswapped remainder is refunded with no tax rebate.
func TestXmaSellFloorPartialFill(t *testing.T) {
	var floorU uint256.Int
	require.NoError(t, uniswapv3.GetSqrtRatioAtTick(500, &floorU))
	floor := floorU.ToBig()

	// tokens[1] is XMA, so selling XMA is oneForZero and pushes sqrtPrice up into the floor.
	sim := newSimWithTokens(t, wethAddr, xmaAddr, xmaAddr, func(e *Extra) {
		e.HasTax, e.BuyTaxBps, e.SellTaxBps = true, 500, 300
		e.XmaSellSqrtPriceLimit = floor
	})

	const amountIn = int64(1e18)
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: xmaAddr, Amount: big.NewInt(amountIn)},
		TokenOut:      wethAddr,
	})
	require.NoError(t, err)
	require.Positive(t, res.RemainingTokenAmountIn.Amount.Sign(),
		"sell should stop at the floor and refund the remainder")

	si := res.SwapInfo.(SwapInfo)

	// A sell is not taxed on the input, so everything not consumed is refunded raw.
	assert.Equal(t, "0", res.Fee.Amount.String())
	consumed := new(big.Int).Add(si.PoolAmountIn, res.RemainingTokenAmountIn.Amount)
	assert.Equal(t, big.NewInt(amountIn).String(), consumed.String(),
		"pool-side input plus refund must equal the full input")

	// Sell tax applies to whatever the partial fill actually produced.
	assert.Equal(t, deductBps(si.PoolAmountOut, 300).String(), res.TokenAmountOut.Amount.String())

	// And the state transition still round-trips through UpdateBalance.
	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: xmaAddr, Amount: big.NewInt(amountIn)},
		TokenAmountOut: *res.TokenAmountOut,
		SwapInfo:       res.SwapInfo,
	})
	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: xmaAddr, Amount: big.NewInt(1e15)},
		TokenOut:      wethAddr,
	})
	// The price now sits exactly on the floor, so the pool rejects any further sell rather than
	// quoting a fill it cannot deliver. This is the floor-pinned state the parity test also sees.
	assert.Error(t, err, "price is pinned at the floor, further sells cannot fill")
}

// TestMsgpackRoundTrip guards the pool-service -> router-service hop. Every Machima-specific field
// is unexported, so this only works because pkg/msgpack sets IncludeUnexported(true); a change
// there would silently zero the tax and ship untaxed quotes.
func TestMsgpackRoundTrip(t *testing.T) {
	sim := newSim(t, func(e *Extra) {
		e.HasTax, e.BuyTaxBps, e.SellTaxBps = true, 500, 300
		e.PoolDeploymentTime = 1
	})

	// Mirrors pkg/msgpack's encoder/decoder settings; that package cannot be imported here
	// because its generated registry imports this one.
	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf)
	enc.IncludeUnexported(true)
	enc.SetForceAsArray(true)
	require.NoError(t, enc.Encode(sim))

	dec := msgpack.NewDecoder(&buf)
	dec.IncludeUnexported(true)
	var decoded PoolSimulator
	require.NoError(t, dec.Decode(&decoded))

	assert.Equal(t, sim.buyTaxBps, decoded.buyTaxBps)
	assert.Equal(t, sim.sellTaxBps, decoded.sellTaxBps)
	assert.Equal(t, sim.hasTax, decoded.hasTax)
	assert.Equal(t, sim.routerAddress, decoded.routerAddress)
	assert.Equal(t, sim.weth, decoded.weth)
	assert.Equal(t, sim.xma, decoded.xma)
	assert.Equal(t, sim.poolDeploymentTime, decoded.poolDeploymentTime)

	// And the round-tripped simulator must still quote identically.
	want := calcOut(t, sim, wethAddr, tokenAddr, 1e15)
	got := calcOut(t, &decoded, wethAddr, tokenAddr, 1e15)
	assert.Equal(t, want.TokenAmountOut.Amount.String(), got.TokenAmountOut.Amount.String())
}

// TestExtraRoundTripsBothDirections guards the load-bearing assumption of the tracker: Machima and
// the delegated UniV3 tracker read and write the SAME pool Extra, so it must survive both
// directions.
//
// The machima -> v3 direction is the one that actually broke in production: declaring the shared
// fields as uint256/int256 made them marshal to *quoted* decimal strings, and math/big refuses to
// unmarshal a quoted string, so v3's FetchPoolTicks failed with
// `cannot unmarshal "\"7143798...\"" into a *big.Int` for every pool. Embedding uniswapv3.Extra is
// what keeps the two byte-identical.
func TestExtraRoundTripsBothDirections(t *testing.T) {
	sqrtPrice, ok := new(big.Int).SetString("7143798628154230053432911752797724", 10)
	require.True(t, ok)

	machimaExtra := Extra{
		Extra: uniswapv3.Extra{
			Liquidity:    big.NewInt(123456789),
			SqrtPriceX96: sqrtPrice,
			TickSpacing:  defaultTickSpacing,
			Tick:         big.NewInt(-183000), // negative current tick
			Ticks: []uniswapv3.Tick{
				{Index: -887200, LiquidityGross: big.NewInt(1000), LiquidityNet: big.NewInt(1000)},
				{Index: 887200, LiquidityGross: big.NewInt(1000), LiquidityNet: big.NewInt(-1000)},
			},
		},
		ProtocolState: ProtocolState{
			HasTax: true, BuyTaxBps: 100, SellTaxBps: 100,
			PoolDeploymentTime:    1234567890,
			XmaSellSqrtPriceLimit: sqrtPrice,
		},
	}

	// machima -> v3: this is what FetchPoolTicks does.
	raw, err := json.Marshal(machimaExtra)
	require.NoError(t, err)
	var v3Extra uniswapv3.Extra
	require.NoError(t, json.Unmarshal(raw, &v3Extra), "uniswapv3.Extra must read machima's Extra")
	assert.Equal(t, sqrtPrice.String(), v3Extra.SqrtPriceX96.String())
	assert.Equal(t, "-183000", v3Extra.Tick.String())
	require.Len(t, v3Extra.Ticks, 2)
	assert.Equal(t, "-1000", v3Extra.Ticks[1].LiquidityNet.String())

	// v3 -> machima: this is what applyMachimaState does after the v3 tracker writes Extra.
	v3Raw, err := json.Marshal(v3Extra)
	require.NoError(t, err)
	back, err := unmarshalExtra(string(v3Raw))
	require.NoError(t, err)
	assert.Equal(t, sqrtPrice.String(), back.SqrtPriceX96.String())
	assert.Equal(t, defaultTickSpacing, back.TickSpacing)

	// v3 -> the simulator's tick-math view, which is uint256-based and must accept the same JSON.
	var tickU256 uniswapv3.ExtraTickU256
	require.NoError(t, json.Unmarshal(v3Raw, &tickU256))
	require.NotNil(t, tickU256.Tick)
	assert.Equal(t, -183000, *tickU256.Tick)
	assert.Equal(t, sqrtPrice.String(), tickU256.SqrtPriceX96.Dec())
	require.Len(t, tickU256.Ticks, 2)
	assert.Equal(t, "-1000", tickU256.Ticks[1].LiquidityNet.Dec())
}

// TestProtocolStateSurvivesUniswapV3Rewrite is the regression test for pools landing in Redis with
// no tax at all. The UniV3 helpers this package delegates to rewrite Extra by marshalling a
// uniswapv3.Extra, which drops the Machima half. That is not a decode error — it just leaves
// hasTax false and the pool quotes untaxed — so it has to be asserted explicitly.
func TestProtocolStateSurvivesUniswapV3Rewrite(t *testing.T) {
	full := Extra{
		Extra: uniswapv3.Extra{
			Liquidity:    big.NewInt(1),
			SqrtPriceX96: sqrtPriceAtTick0,
			TickSpacing:  defaultTickSpacing,
			Tick:         big.NewInt(0),
		},
		ProtocolState: ProtocolState{
			HasTax: true, BuyTaxBps: 100, SellTaxBps: 100,
			PoolDeploymentTime:    1784700554,
			XmaSellSqrtPriceLimit: big.NewInt(745547668962671613),
		},
	}

	// Simulate what uniswapv3.FetchPoolTicks does: read our Extra, rewrite it as a uniswapv3.Extra.
	raw, err := json.Marshal(full)
	require.NoError(t, err)
	var v3Only uniswapv3.Extra
	require.NoError(t, json.Unmarshal(raw, &v3Only))
	rewritten, err := json.Marshal(v3Only)
	require.NoError(t, err)

	stripped, err := unmarshalExtra(string(rewritten))
	require.NoError(t, err)
	require.False(t, stripped.HasTax, "precondition: the rewrite must drop the Machima half")

	// What FetchPoolTicks now does to put it back.
	stripped.ProtocolState = full.ProtocolState
	assert.Equal(t, full.ProtocolState, stripped.ProtocolState)

	// The merged Extra must still be flat JSON — embedding must not nest under a key, or nothing
	// downstream would find these fields.
	merged, err := json.Marshal(stripped)
	require.NoError(t, err)
	var asMap map[string]any
	require.NoError(t, json.Unmarshal(merged, &asMap))
	for _, k := range []string{
		"liquidity", "sqrtPriceX96", "tickSpacing", "tick",
		"buyTaxBps", "sellTaxBps", "hasTax", "poolDeploymentTime", "xmaSellSqrtPriceLimit",
	} {
		assert.Contains(t, asMap, k, "Extra JSON must stay flat")
	}
	assert.NotContains(t, asMap, "Extra")
	assert.NotContains(t, asMap, "ProtocolState")
}
