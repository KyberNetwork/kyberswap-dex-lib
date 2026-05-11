package ambient

import (
	"math/big"
	"testing"
)

func TestSwapToLimit_SmallBuy(t *testing.T) {
	// Simulate a small buy on a curve with known state
	curve := &CurveState{
		PriceRoot:    GetSqrtRatioAtTick(0), // price = 1.0
		AmbientSeeds: big.NewInt(1_000_000_000),
		ConcLiq:      big.NewInt(500_000_000),
		SeedDeflator: 0,
		ConcGrowth:   0,
	}
	origPrice := new(big.Int).Set(curve.PriceRoot)

	pool := &PoolParams{
		FeeRate:      2500, // 0.25%
		ProtocolTake: 0,
		TickSize:     16,
	}

	swap := &SwapDirective{
		Qty:        big.NewInt(10_000),
		InBaseQty:  true,
		IsBuy:      true,
		LimitPrice: GetSqrtRatioAtTick(MaxTick - 1),
	}

	accum := NewSwapAccum()
	SwapToLimit(curve, accum, swap, pool, MaxTick)

	// Price should increase for a buy
	if curve.PriceRoot.Cmp(origPrice) <= 0 {
		t.Errorf("buy should increase price: before=%s after=%s", origPrice, curve.PriceRoot)
	}

	// Swap should be fully consumed
	if swap.Qty.Sign() != 0 {
		t.Errorf("swap should be fully consumed, got qty=%s", swap.Qty)
	}

	// Base flow should be positive (user pays), quote flow negative (user receives)
	if accum.BaseFlow.Sign() <= 0 {
		t.Errorf("buy: baseFlow should be positive, got %s", accum.BaseFlow)
	}
	if accum.QuoteFlow.Sign() >= 0 {
		t.Errorf("buy: quoteFlow should be negative, got %s", accum.QuoteFlow)
	}

	t.Logf("Swap result: price %s -> %s, baseFlow=%s, quoteFlow=%s",
		origPrice, curve.PriceRoot, accum.BaseFlow, accum.QuoteFlow)
}

func TestSwapToLimit_SmallSell(t *testing.T) {
	curve := &CurveState{
		PriceRoot:    GetSqrtRatioAtTick(0),
		AmbientSeeds: big.NewInt(1_000_000_000),
		ConcLiq:      big.NewInt(500_000_000),
		SeedDeflator: 0,
		ConcGrowth:   0,
	}
	origPrice := new(big.Int).Set(curve.PriceRoot)

	pool := &PoolParams{
		FeeRate:      2500,
		ProtocolTake: 0,
		TickSize:     16,
	}

	swap := &SwapDirective{
		Qty:        big.NewInt(10_000),
		InBaseQty:  true,
		IsBuy:      false,
		LimitPrice: GetSqrtRatioAtTick(MinTick + 1),
	}

	accum := NewSwapAccum()
	SwapToLimit(curve, accum, swap, pool, MinTick)

	if curve.PriceRoot.Cmp(origPrice) >= 0 {
		t.Errorf("sell should decrease price: before=%s after=%s", origPrice, curve.PriceRoot)
	}
	if swap.Qty.Sign() != 0 {
		t.Errorf("swap should be fully consumed, got qty=%s", swap.Qty)
	}

	t.Logf("Swap result: price %s -> %s, baseFlow=%s, quoteFlow=%s",
		origPrice, curve.PriceRoot, accum.BaseFlow, accum.QuoteFlow)
}

func TestSwapToLimit_HitsBump(t *testing.T) {
	curve := &CurveState{
		PriceRoot:    GetSqrtRatioAtTick(100),
		AmbientSeeds: big.NewInt(100_000),
		ConcLiq:      big.NewInt(50_000),
		SeedDeflator: 0,
		ConcGrowth:   0,
	}

	pool := &PoolParams{
		FeeRate:      2500,
		ProtocolTake: 0,
		TickSize:     16,
	}

	// Large swap that should hit the bump tick boundary
	swap := &SwapDirective{
		Qty:        big.NewInt(1_000_000),
		InBaseQty:  true,
		IsBuy:      true,
		LimitPrice: GetSqrtRatioAtTick(MaxTick - 1),
	}

	bumpTick := int32(200)
	bumpPrice := new(big.Int).Sub(GetSqrtRatioAtTick(bumpTick), big.NewInt(1))

	accum := NewSwapAccum()
	SwapToLimit(curve, accum, swap, pool, bumpTick)

	// Should stop at the bump price
	if curve.PriceRoot.Cmp(bumpPrice) != 0 {
		t.Errorf("should stop at bump price: got %s, want %s", curve.PriceRoot, bumpPrice)
	}

	// Should have remaining qty
	if swap.Qty.Sign() == 0 {
		t.Error("should have remaining swap qty after hitting bump")
	}

	t.Logf("Hit bump: price=%s, remaining qty=%s", curve.PriceRoot, swap.Qty)
}

func TestDeriveImpact_Symmetry(t *testing.T) {
	curve := &CurveState{
		PriceRoot:    GetSqrtRatioAtTick(0),
		AmbientSeeds: big.NewInt(1_000_000),
		ConcLiq:      big.NewInt(0),
		SeedDeflator: 0,
		ConcGrowth:   0,
	}

	flow := big.NewInt(1000)

	// Buy with base tokens
	_, buyPrice := DeriveImpact(curve, flow, true, true)

	// The resulting price should be higher (buy pushes price up)
	if buyPrice.Cmp(curve.PriceRoot) <= 0 {
		t.Errorf("buy should push price up: from %s to %s", curve.PriceRoot, buyPrice)
	}

	// Sell with base tokens
	_, sellPrice := DeriveImpact(curve, flow, true, false)
	if sellPrice.Cmp(curve.PriceRoot) >= 0 {
		t.Errorf("sell should push price down: from %s to %s", curve.PriceRoot, sellPrice)
	}
}
