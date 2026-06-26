package ambient

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
)

func TestGetSqrtRatioAtTick_KnownValues(t *testing.T) {
	tests := []struct {
		tick     int32
		expected string
	}{
		{0, "18446744073709551616"}, // 2^64
		{MinTick, "65538"},
		{MaxTick, "21267430153580247136652501917186561138"},
	}

	for _, tt := range tests {
		got := GetSqrtRatioAtTick(tt.tick)
		expected, _ := new(big.Int).SetString(tt.expected, 10)
		if got.ToBig().Cmp(expected) != 0 {
			t.Errorf("GetSqrtRatioAtTick(%d) = %s, want %s", tt.tick, got.ToBig(), expected)
		}
	}
}

func TestGetTickAtSqrtRatio_Roundtrip(t *testing.T) {
	ticks := []int32{0, 1, -1, 100, -100, 1000, -1000, 10000, -10000, 100000, -100000}
	for _, tick := range ticks {
		price := GetSqrtRatioAtTick(tick)
		gotTick := GetTickAtSqrtRatio(price)
		if gotTick != tick {
			t.Errorf("roundtrip tick %d: getSqrt=%s, getTick=%d", tick, price.ToBig(), gotTick)
		}
	}
}

func TestGetSqrtRatioAtTick_Boundaries(t *testing.T) {
	minPrice := GetSqrtRatioAtTick(MinTick)
	if minPrice.Cmp(&uMinSqrtRatio) != 0 {
		t.Errorf("MinTick price: got %s, want %s", minPrice.ToBig(), uMinSqrtRatio.ToBig())
	}

	maxPrice := GetSqrtRatioAtTick(MaxTick)
	if maxPrice.Cmp(&uMaxSqrtRatio) != 0 {
		t.Errorf("MaxTick price: got %s, want %s", maxPrice.ToBig(), uMaxSqrtRatio.ToBig())
	}
}

func TestActiveLiquidity(t *testing.T) {
	var seeds, concLiq uint256.Int
	seeds.SetUint64(1_000_000)
	concLiq.SetUint64(500_000)

	curve := &CurveState{
		PriceRoot:    *uint256.NewInt(18446744073709551),
		AmbientSeeds: seeds,
		ConcLiq:      concLiq,
		SeedDeflator: 0,
		ConcGrowth:   0,
	}
	var liq uint256.Int
	ActiveLiquidity(&liq, curve)

	expected := new(big.Int).Add(big.NewInt(1_000_000), big.NewInt(500_000))
	if liq.ToBig().Cmp(expected) != 0 {
		t.Errorf("ActiveLiquidity = %s, want %s", liq.ToBig(), expected)
	}
}

func TestCalcFeeOverFlow(t *testing.T) {
	var flow uint256.Int
	flow.SetUint64(1_000_000)
	feeRate := uint16(2500) // 0.25%
	protoProp := uint8(0)

	liqFee, protoFee := CalcFeeOverFlow(flow, feeRate, protoProp)

	if liqFee.Uint64() != 2500 {
		t.Errorf("liqFee = %s, want 2500", liqFee.ToBig())
	}
	if !protoFee.IsZero() {
		t.Errorf("protoFee = %s, want 0", protoFee.ToBig())
	}

	// With protocol take
	protoProp = 128 // 50%
	liqFee, protoFee = CalcFeeOverFlow(flow, feeRate, protoProp)
	if protoFee.Uint64() != 1250 {
		t.Errorf("protoFee with 50%% take = %s, want 1250", protoFee.ToBig())
	}
	if liqFee.Uint64() != 1250 {
		t.Errorf("liqFee with 50%% take = %s, want 1250", liqFee.ToBig())
	}
}

func TestCompoundStack(t *testing.T) {
	// (1+0) * (1+0) - 1 = 0
	if CompoundStack(0, 0) != 0 {
		t.Error("CompoundStack(0,0) should be 0")
	}

	// Commutativity
	a, b := uint64(100000), uint64(200000)
	if CompoundStack(a, b) != CompoundStack(b, a) {
		t.Error("CompoundStack should be commutative")
	}
}

func TestLotsToLiquidity(t *testing.T) {
	var lots, liq uint256.Int
	lots.SetUint64(100)
	LotsToLiquidity(&liq, &lots)
	if liq.Uint64() != 100*1024 {
		t.Errorf("LotsToLiquidity(100) = %s, want %d", liq.ToBig(), 100*1024)
	}

	// Odd lots (knockout flag) should mask out the flag
	lots.SetUint64(101) // bit 0 set
	LotsToLiquidity(&liq, &lots)
	if liq.Uint64() != 100*1024 {
		t.Errorf("LotsToLiquidity(101) = %s, want %d", liq.ToBig(), 100*1024)
	}
}

func TestBitmapHelpers(t *testing.T) {
	if CastBitmapIndex(0) != 128 {
		t.Error("CastBitmapIndex(0) should be 128")
	}
	if CastBitmapIndex(-128) != 0 {
		t.Error("CastBitmapIndex(-128) should be 0")
	}
	if CastBitmapIndex(127) != 255 {
		t.Error("CastBitmapIndex(127) should be 255")
	}
	if UncastBitmapIndex(128) != 0 {
		t.Error("UncastBitmapIndex(128) should be 0")
	}
	if UncastBitmapIndex(0) != -128 {
		t.Error("UncastBitmapIndex(0) should be -128")
	}
}

func TestWeldTick(t *testing.T) {
	tick := int32(-197080)
	lobby := LobbyKey(tick)
	mezzB := MezzBit(tick)
	termB := TermBit(tick)

	reconstructed := WeldLobbyMezzTerm(lobby, mezzB, termB)
	if reconstructed != tick {
		t.Errorf("weld roundtrip: got %d, want %d (lobby=%d mezzBit=%d termBit=%d)",
			reconstructed, tick, lobby, mezzB, termB)
	}
}
