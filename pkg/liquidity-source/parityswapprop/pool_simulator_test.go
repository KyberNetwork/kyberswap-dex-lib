package parityswapprop

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	weth = "0x0bd7d308f8e1639fab988df18a8011f41eacad73"
	usdg = "0x5fc5360d0400a0fd4f2af552add042d716f1d168"
)

// livePool builds the entity.Pool JSON exactly as the tracker produces it,
// seeded with on-chain values captured from a real swap.
func livePool(t *testing.T) entity.Pool {
	t.Helper()

	staticExtra := `{"base":"` + weth + `","quote":"` + usdg + `","baseScale":"1000000000000000000","quoteScale":"1000000"}`
	extra := `{
		"paused": false,
		"feeBps": 0,
		"maxSpreadBps": 50,
		"maxStaleness": 8,
		"maxSwapNotional": "500000000",
		"maxBlockNotional": "1500000000",
		"minBaseReserve": "25000000000000000",
		"minQuoteReserve": "50000000",
		"oracle": "0xc484f39b1c25fc7fcb140fbc0824a6ff9143e405",
		"oracleBid": 190510625160,
		"oracleMid": 190584000000,
		"oracleAsk": 190657374840,
		"oracleUpdatedAt": 1786599215,
		"lastSwapBlock": 0,
		"blockNotional": "0"
	}`

	return entity.Pool{
		Address:     "0xd778f470c69bce130d1cef08852f34bf296b4e67",
		Exchange:    string(DexType),
		Type:        DexType,
		Timestamp:   1786599215,
		Reserves:    entity.PoolReserves{"250000000000000000", "500000000"},
		Tokens:      []*entity.PoolToken{{Address: weth, Decimals: 18, Swappable: true}, {Address: usdg, Decimals: 6, Swappable: true}},
		StaticExtra: staticExtra,
		Extra:       extra,
	}
}

func TestNewPoolSimulator_CalcAmountOut_MatchesOnChainSwap(t *testing.T) {
	ep := livePool(t)
	ep.Extra = `{"paused":false,"feeBps":0,"maxSpreadBps":50,"maxStaleness":3600,"maxSwapNotional":"500000000","maxBlockNotional":"1500000000","minBaseReserve":"25000000000000000","minQuoteReserve":"50000000","oracle":"0xc484f39b1c25fc7fcb140fbc0824a6ff9143e405","oracleBid":190510625160,"oracleMid":190584000000,"oracleAsk":190657374840,"oracleUpdatedAt":9999999999,"lastSwapBlock":0,"blockNotional":"0"}`

	sim, err := NewPoolSimulator(ep)
	if err != nil {
		t.Fatalf("NewPoolSimulator: %v", err)
	}

	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: weth, Amount: big.NewInt(100000000000000000)},
		TokenOut:      usdg,
	})
	if err != nil {
		t.Fatalf("CalcAmountOut: %v", err)
	}
	if res.TokenAmountOut.Amount.Cmp(big.NewInt(190510625)) != 0 {
		t.Fatalf("amountOut = %s, want 190510625", res.TokenAmountOut.Amount)
	}
	if res.TokenAmountOut.Token != usdg {
		t.Fatalf("output token = %s, want %s", res.TokenAmountOut.Token, usdg)
	}
	if res.Fee.Amount.Sign() != 0 {
		t.Fatalf("fee = %s, want 0 (feeBps=0 live)", res.Fee.Amount)
	}
	if res.Gas != defaultGas {
		t.Fatalf("gas = %d, want %d", res.Gas, defaultGas)
	}
}

func TestPoolSimulator_CalcAmountOut_Paused(t *testing.T) {
	ep := livePool(t)
	ep.Extra = `{"paused":true}`

	sim, err := NewPoolSimulator(ep)
	if err != nil {
		t.Fatalf("NewPoolSimulator: %v", err)
	}

	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: weth, Amount: big.NewInt(1e9)},
		TokenOut:      usdg,
	})
	if err != ErrPoolPaused {
		t.Fatalf("want ErrPoolPaused, got %v", err)
	}
}

func TestPoolSimulator_CalcAmountOut_InvalidToken(t *testing.T) {
	sim, err := NewPoolSimulator(livePool(t))
	if err != nil {
		t.Fatalf("NewPoolSimulator: %v", err)
	}

	cases := []struct {
		name              string
		tokenIn, tokenOut string
	}{
		{"tokenIn not in pool", "0x000000000000000000000000000000000000ff", usdg},
		{"tokenOut not in pool", weth, "0x000000000000000000000000000000000000ff"},
		{"tokenIn == tokenOut", weth, weth},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: tc.tokenIn, Amount: big.NewInt(1e9)},
				TokenOut:      tc.tokenOut,
			})
			if err != ErrInvalidToken {
				t.Fatalf("want ErrInvalidToken, got %v", err)
			}
		})
	}
}

func TestPoolSimulator_CalcAmountOut_ZeroAmountIn(t *testing.T) {
	sim, err := NewPoolSimulator(livePool(t))
	if err != nil {
		t.Fatalf("NewPoolSimulator: %v", err)
	}

	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: weth, Amount: big.NewInt(0)},
		TokenOut:      usdg,
	})
	if err != ErrZeroAmount {
		t.Fatalf("want ErrZeroAmount, got %v", err)
	}
}

func TestPoolSimulator_CalcAmountOut_StaleOracle_PropagatesFromMath(t *testing.T) {
	// Live oracleUpdatedAt (1786599215) plus maxStaleness (8s) is far in the
	// past relative to time.Now() -- this is the pool's actual on-chain state,
	// so StalePrice fires for real, not just on a synthetic fixture.
	sim, err := NewPoolSimulator(livePool(t))
	if err != nil {
		t.Fatalf("NewPoolSimulator: %v", err)
	}

	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: weth, Amount: big.NewInt(100000000000000000)},
		TokenOut:      usdg,
	})
	if err != ErrStalePrice {
		t.Fatalf("want ErrStalePrice, got %v", err)
	}
}

func TestPoolSimulator_CloneState_IsIndependent(t *testing.T) {
	sim, err := NewPoolSimulator(livePool(t))
	if err != nil {
		t.Fatalf("NewPoolSimulator: %v", err)
	}

	cloned := sim.CloneState().(*PoolSimulator)
	cloned.Info.Reserves[0].SetInt64(999)

	if sim.Info.Reserves[0].Cmp(big.NewInt(999)) == 0 {
		t.Fatalf("mutating the clone's reserves mutated the original")
	}
	if sim.Info.Reserves[0].Cmp(big.NewInt(250000000000000000)) != 0 {
		t.Fatalf("original reserve[0] changed unexpectedly: %s", sim.Info.Reserves[0])
	}
}

// freshExtraJSON is livePool's Extra with maxStaleness/oracleUpdatedAt
// widened (matching TestNewPoolSimulator_CalcAmountOut_MatchesOnChainSwap's
// approach) so CalcAmountOut exercises the real formula instead of universally
// hitting ErrStalePrice, plus an overridable maxBlockNotional for the
// block-notional-cap accumulation tests below.
func freshExtraJSON(maxBlockNotional string) string {
	return `{"paused":false,"feeBps":0,"maxSpreadBps":50,"maxStaleness":3600,"maxSwapNotional":"500000000","maxBlockNotional":"` +
		maxBlockNotional +
		`","minBaseReserve":"25000000000000000","minQuoteReserve":"50000000","oracle":"0xc484f39b1c25fc7fcb140fbc0824a6ff9143e405","oracleBid":190510625160,"oracleMid":190584000000,"oracleAsk":190657374840,"oracleUpdatedAt":9999999999,"lastSwapBlock":0,"blockNotional":"0"}`
}

func TestPoolSimulator_CalcAmountOut_RepeatedCallsAreDeterministic(t *testing.T) {
	ep := livePool(t)
	ep.Extra = freshExtraJSON("1500000000")
	sim, err := NewPoolSimulator(ep)
	if err != nil {
		t.Fatalf("NewPoolSimulator: %v", err)
	}

	params := pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: weth, Amount: big.NewInt(100000000000000000)},
		TokenOut:      usdg,
	}
	res1, err := sim.CalcAmountOut(params)
	if err != nil {
		t.Fatalf("call 1: %v", err)
	}
	res2, err := sim.CalcAmountOut(params)
	if err != nil {
		t.Fatalf("call 2: %v", err)
	}
	if res1.TokenAmountOut.Amount.Cmp(res2.TokenAmountOut.Amount) != 0 {
		t.Fatalf("non-deterministic: %s vs %s", res1.TokenAmountOut.Amount, res2.TokenAmountOut.Amount)
	}
	// CalcAmountOut must be pure -- confirms neither call mutated reserves.
	if sim.Info.Reserves[baseIdx].Cmp(big.NewInt(250000000000000000)) != 0 {
		t.Fatalf("CalcAmountOut mutated reserves: %s", sim.Info.Reserves[baseIdx])
	}
}

// TestPoolSimulator_UpdateBalance_UpdatesReservesAndBlockAccounting exercises
// the CalcAmountOut -> UpdateBalance loop for a single swap: reserves move by
// the exact swapped amounts, and Extra.LastSwapBlock/BlockNotional pick up
// CalcAmountOut's SwapInfo verbatim.
func TestPoolSimulator_UpdateBalance_UpdatesReservesAndBlockAccounting(t *testing.T) {
	ep := livePool(t)
	ep.BlockNumber = 35120382
	ep.Extra = freshExtraJSON("1500000000")
	sim, err := NewPoolSimulator(ep)
	if err != nil {
		t.Fatalf("NewPoolSimulator: %v", err)
	}

	amountIn := big.NewInt(100000000000000000)
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: weth, Amount: amountIn},
		TokenOut:      usdg,
	})
	if err != nil {
		t.Fatalf("CalcAmountOut: %v", err)
	}

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: weth, Amount: amountIn},
		TokenAmountOut: *res.TokenAmountOut,
		SwapInfo:       res.SwapInfo,
	})

	wantBase := new(big.Int).Add(big.NewInt(250000000000000000), amountIn)
	if sim.Info.Reserves[baseIdx].Cmp(wantBase) != 0 {
		t.Fatalf("base reserve = %s, want %s", sim.Info.Reserves[baseIdx], wantBase)
	}
	wantQuote := new(big.Int).Sub(big.NewInt(500000000), res.TokenAmountOut.Amount)
	if sim.Info.Reserves[quoteIdx].Cmp(wantQuote) != 0 {
		t.Fatalf("quote reserve = %s, want %s", sim.Info.Reserves[quoteIdx], wantQuote)
	}

	swapInfo, ok := res.SwapInfo.(SwapInfo)
	if !ok {
		t.Fatalf("SwapInfo is not of type SwapInfo: %T", res.SwapInfo)
	}
	if sim.extra.LastSwapBlock != swapInfo.SwapBlock {
		t.Fatalf("LastSwapBlock = %d, want %d", sim.extra.LastSwapBlock, swapInfo.SwapBlock)
	}
	if sim.extra.BlockNotional.Cmp(swapInfo.BlockNotional) != 0 {
		t.Fatalf("BlockNotional = %s, want %s", sim.extra.BlockNotional, swapInfo.BlockNotional)
	}
}

// TestPoolSimulator_UpdateBalance_AccumulatesBlockNotionalAcrossChainedSwaps
// exercises a route that simulates two swaps against the same pool within
// one block: the second swap's block-notional-cap check must see the first
// swap's notional, mirroring PmmPool.swap()'s on-chain
// lastSwapBlock==block.number accumulate-or-reset logic. Without a real
// UpdateBalance, the second call would incorrectly succeed forever since
// accumulated would never advance past 0.
func TestPoolSimulator_UpdateBalance_AccumulatesBlockNotionalAcrossChainedSwaps(t *testing.T) {
	ep := livePool(t)
	ep.BlockNumber = 35120382
	// Each 0.1 WETH swap has notional=190510625 (see
	// TestNewPoolSimulator_CalcAmountOut_MatchesOnChainSwap). 300000000 fits
	// one such swap but not two back-to-back (190510625*2 = 381021250).
	ep.Extra = freshExtraJSON("300000000")
	sim, err := NewPoolSimulator(ep)
	if err != nil {
		t.Fatalf("NewPoolSimulator: %v", err)
	}

	amountIn := big.NewInt(100000000000000000)
	swap := func() (*pool.CalcAmountOutResult, error) {
		return sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: weth, Amount: amountIn},
			TokenOut:      usdg,
		})
	}

	res1, err := swap()
	if err != nil {
		t.Fatalf("swap 1: unexpected error: %v", err)
	}
	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: weth, Amount: amountIn},
		TokenAmountOut: *res1.TokenAmountOut,
		SwapInfo:       res1.SwapInfo,
	})

	if _, err := swap(); err != ErrBlockCapExceeded {
		t.Fatalf("swap 2: want ErrBlockCapExceeded (swap 1's notional must carry forward), got %v", err)
	}
}

// TestPoolSimulator_CloneState_UpdateBalance_DoesNotMutateOriginal guards the
// specific invariant CloneState's doc comment relies on: UpdateBalance always
// reassigns extra.BlockNotional wholesale rather than mutating the shared
// pointer in place, so CloneState's shallow copy of `extra` plus its deep
// copy of Info.Reserves is sufficient. If UpdateBalance ever starts mutating
// extra.BlockNotional in place instead of reassigning it, this test catches
// the resulting clone/original aliasing bug.
func TestPoolSimulator_CloneState_UpdateBalance_DoesNotMutateOriginal(t *testing.T) {
	ep := livePool(t)
	ep.BlockNumber = 35120382
	ep.Extra = freshExtraJSON("1500000000")
	sim, err := NewPoolSimulator(ep)
	if err != nil {
		t.Fatalf("NewPoolSimulator: %v", err)
	}

	amountIn := big.NewInt(100000000000000000)
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: weth, Amount: amountIn},
		TokenOut:      usdg,
	})
	if err != nil {
		t.Fatalf("CalcAmountOut: %v", err)
	}

	cloned := sim.CloneState().(*PoolSimulator)
	cloned.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: weth, Amount: amountIn},
		TokenAmountOut: *res.TokenAmountOut,
		SwapInfo:       res.SwapInfo,
	})

	if sim.Info.Reserves[baseIdx].Cmp(big.NewInt(250000000000000000)) != 0 {
		t.Fatalf("original base reserve mutated by clone's UpdateBalance: %s", sim.Info.Reserves[baseIdx])
	}
	if sim.Info.Reserves[quoteIdx].Cmp(big.NewInt(500000000)) != 0 {
		t.Fatalf("original quote reserve mutated by clone's UpdateBalance: %s", sim.Info.Reserves[quoteIdx])
	}
	if sim.extra.LastSwapBlock != 0 {
		t.Fatalf("original LastSwapBlock mutated by clone's UpdateBalance: %d", sim.extra.LastSwapBlock)
	}
	if sim.extra.BlockNotional.Sign() != 0 {
		t.Fatalf("original BlockNotional mutated by clone's UpdateBalance: %s", sim.extra.BlockNotional)
	}

	wantClonedBase := new(big.Int).Add(big.NewInt(250000000000000000), amountIn)
	if cloned.Info.Reserves[baseIdx].Cmp(wantClonedBase) != 0 {
		t.Fatalf("cloned base reserve = %s, want %s", cloned.Info.Reserves[baseIdx], wantClonedBase)
	}
}

func TestPoolSimulator_GetMetaInfo(t *testing.T) {
	ep := livePool(t)
	ep.BlockNumber = 12345
	sim, err := NewPoolSimulator(ep)
	if err != nil {
		t.Fatalf("NewPoolSimulator: %v", err)
	}

	meta, ok := sim.GetMetaInfo(weth, usdg).(MetaInfo)
	if !ok {
		t.Fatalf("GetMetaInfo did not return MetaInfo")
	}
	if meta.BlockNumber != 12345 {
		t.Fatalf("BlockNumber = %d, want 12345", meta.BlockNumber)
	}
}
