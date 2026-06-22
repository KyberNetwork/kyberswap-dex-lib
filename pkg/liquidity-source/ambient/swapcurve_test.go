package ambient

import (
	"testing"

	"github.com/holiman/uint256"
)

func TestSwapToLimit_SmallBuy(t *testing.T) {
	var seeds, concLiq uint256.Int
	seeds.SetUint64(1_000_000_000)
	concLiq.SetUint64(500_000_000)

	curve := &CurveState{
		PriceRoot:    GetSqrtRatioAtTick(0),
		AmbientSeeds: seeds,
		ConcLiq:      concLiq,
		SeedDeflator: 0,
		ConcGrowth:   0,
	}
	origPrice := curve.PriceRoot // value copy

	pool := &PoolParams{
		FeeRate:      2500,
		ProtocolTake: 0,
		TickSize:     16,
	}

	swap := &SwapDirective{
		InBaseQty:  true,
		IsBuy:      true,
		LimitPrice: GetSqrtRatioAtTick(MaxTick - 1),
	}
	swap.Qty.SetUint64(10_000)

	accum := NewSwapAccum()
	SwapToLimit(curve, accum, swap, pool, MaxTick)

	if curve.PriceRoot.Cmp(&origPrice) <= 0 {
		t.Errorf("buy should increase price: before=%s after=%s", origPrice.ToBig(), curve.PriceRoot.ToBig())
	}
	if !swap.Qty.IsZero() {
		t.Errorf("swap should be fully consumed, got qty=%s", swap.Qty.ToBig())
	}
	if accum.BaseFlow.Sign() <= 0 {
		t.Errorf("buy: baseFlow should be positive, got %s", FlowToBig(accum.BaseFlow))
	}
	if accum.QuoteFlow.Sign() >= 0 {
		t.Errorf("buy: quoteFlow should be negative, got %s", FlowToBig(accum.QuoteFlow))
	}

	t.Logf("Swap result: price %s -> %s, baseFlow=%s, quoteFlow=%s",
		origPrice.ToBig(), curve.PriceRoot.ToBig(), FlowToBig(accum.BaseFlow), FlowToBig(accum.QuoteFlow))
}

func TestSwapToLimit_SmallSell(t *testing.T) {
	var seeds, concLiq uint256.Int
	seeds.SetUint64(1_000_000_000)
	concLiq.SetUint64(500_000_000)

	curve := &CurveState{
		PriceRoot:    GetSqrtRatioAtTick(0),
		AmbientSeeds: seeds,
		ConcLiq:      concLiq,
		SeedDeflator: 0,
		ConcGrowth:   0,
	}
	origPrice := curve.PriceRoot

	pool := &PoolParams{
		FeeRate:      2500,
		ProtocolTake: 0,
		TickSize:     16,
	}

	swap := &SwapDirective{
		InBaseQty:  true,
		IsBuy:      false,
		LimitPrice: GetSqrtRatioAtTick(MinTick + 1),
	}
	swap.Qty.SetUint64(10_000)

	accum := NewSwapAccum()
	SwapToLimit(curve, accum, swap, pool, MinTick)

	if curve.PriceRoot.Cmp(&origPrice) >= 0 {
		t.Errorf("sell should decrease price: before=%s after=%s", origPrice.ToBig(), curve.PriceRoot.ToBig())
	}
	if !swap.Qty.IsZero() {
		t.Errorf("swap should be fully consumed, got qty=%s", swap.Qty.ToBig())
	}

	t.Logf("Swap result: price %s -> %s, baseFlow=%s, quoteFlow=%s",
		origPrice.ToBig(), curve.PriceRoot.ToBig(), FlowToBig(accum.BaseFlow), FlowToBig(accum.QuoteFlow))
}

func TestSwapToLimit_HitsBump(t *testing.T) {
	var seeds, concLiq uint256.Int
	seeds.SetUint64(100_000)
	concLiq.SetUint64(50_000)

	curve := &CurveState{
		PriceRoot:    GetSqrtRatioAtTick(100),
		AmbientSeeds: seeds,
		ConcLiq:      concLiq,
		SeedDeflator: 0,
		ConcGrowth:   0,
	}

	pool := &PoolParams{
		FeeRate:      2500,
		ProtocolTake: 0,
		TickSize:     16,
	}

	swap := &SwapDirective{
		InBaseQty:  true,
		IsBuy:      true,
		LimitPrice: GetSqrtRatioAtTick(MaxTick - 1),
	}
	swap.Qty.SetUint64(1_000_000)

	bumpTick := int32(200)
	bumpPrice := GetSqrtRatioAtTick(bumpTick)
	bumpPrice.Sub(&bumpPrice, new(uint256.Int).SetUint64(1))

	accum := NewSwapAccum()
	SwapToLimit(curve, accum, swap, pool, bumpTick)

	if curve.PriceRoot.Cmp(&bumpPrice) != 0 {
		t.Errorf("should stop at bump price: got %s, want %s", curve.PriceRoot.ToBig(), bumpPrice.ToBig())
	}
	if swap.Qty.IsZero() {
		t.Error("should have remaining swap qty after hitting bump")
	}

	t.Logf("Hit bump: price=%s, remaining qty=%s", curve.PriceRoot.ToBig(), swap.Qty.ToBig())
}

func TestDeriveImpact_Symmetry(t *testing.T) {
	var seeds uint256.Int
	seeds.SetUint64(1_000_000)

	curve := &CurveState{
		PriceRoot:    GetSqrtRatioAtTick(0),
		AmbientSeeds: seeds,
		SeedDeflator: 0,
		ConcGrowth:   0,
	}

	var flow uint256.Int
	flow.SetUint64(1000)

	_, buyPrice := DeriveImpact(curve, flow, true, true)
	if buyPrice.Cmp(&curve.PriceRoot) <= 0 {
		t.Errorf("buy should push price up: from %s to %s", curve.PriceRoot.ToBig(), buyPrice.ToBig())
	}

	_, sellPrice := DeriveImpact(curve, flow, true, false)
	if sellPrice.Cmp(&curve.PriceRoot) >= 0 {
		t.Errorf("sell should push price down: from %s to %s", curve.PriceRoot.ToBig(), sellPrice.ToBig())
	}
}
