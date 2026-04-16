package ambient

import (
	"math/big"
	"testing"
)

func TestGetSqrtRatioAtTick_KnownValues(t *testing.T) {
	tests := []struct {
		tick     int32
		expected string
	}{
		{0, "18446744073709551616"},                         // 2^64 = 1.0 in Q64.64
		{MinTick, "65538"},                                  // MIN_SQRT_RATIO
		{MaxTick, "21267430153580247136652501917186561138"}, // MAX_SQRT_RATIO
	}

	for _, tt := range tests {
		got := GetSqrtRatioAtTick(tt.tick)
		expected, _ := new(big.Int).SetString(tt.expected, 10)
		if got.Cmp(expected) != 0 {
			t.Errorf("GetSqrtRatioAtTick(%d) = %s, want %s", tt.tick, got, expected)
		}
	}
}

func TestGetTickAtSqrtRatio_Roundtrip(t *testing.T) {
	ticks := []int32{0, 1, -1, 100, -100, 1000, -1000, 10000, -10000, 100000, -100000}
	for _, tick := range ticks {
		price := GetSqrtRatioAtTick(tick)
		gotTick := GetTickAtSqrtRatio(price)
		if gotTick != tick {
			t.Errorf("roundtrip tick %d: getSqrt=%s, getTick=%d", tick, price, gotTick)
		}
	}
}

func TestGetSqrtRatioAtTick_Boundaries(t *testing.T) {
	minPrice := GetSqrtRatioAtTick(MinTick)
	if minPrice.Cmp(MinSqrtRatio) != 0 {
		t.Errorf("MinTick price: got %s, want %s", minPrice, MinSqrtRatio)
	}

	maxPrice := GetSqrtRatioAtTick(MaxTick)
	if maxPrice.Cmp(MaxSqrtRatio) != 0 {
		t.Errorf("MaxTick price: got %s, want %s", maxPrice, MaxSqrtRatio)
	}
}

func TestActiveLiquidity(t *testing.T) {
	curve := &CurveState{
		PriceRoot:    big.NewInt(18446744073709551),
		AmbientSeeds: big.NewInt(1000000),
		ConcLiq:      big.NewInt(500000),
		SeedDeflator: 0,
		ConcGrowth:   0,
	}
	liq := ActiveLiquidity(curve)
	// With zero deflator, ambient = seeds * (1 + 0) = seeds
	expected := new(big.Int).Add(big.NewInt(1000000), big.NewInt(500000))
	if liq.Cmp(expected) != 0 {
		t.Errorf("ActiveLiquidity = %s, want %s", liq, expected)
	}
}

func TestCalcFeeOverFlow(t *testing.T) {
	flow := big.NewInt(1_000_000)
	feeRate := uint16(2500) // 0.25%
	protoProp := uint8(0)

	liqFee, protoFee := CalcFeeOverFlow(flow, feeRate, protoProp)

	expectedTotal := big.NewInt(2500) // 1M * 2500 / 1M
	if liqFee.Cmp(expectedTotal) != 0 {
		t.Errorf("liqFee = %s, want %s", liqFee, expectedTotal)
	}
	if protoFee.Sign() != 0 {
		t.Errorf("protoFee = %s, want 0", protoFee)
	}

	// With protocol take
	protoProp = 128 // 50%
	liqFee, protoFee = CalcFeeOverFlow(flow, feeRate, protoProp)
	if protoFee.Cmp(big.NewInt(1250)) != 0 {
		t.Errorf("protoFee with 50%% take = %s, want 1250", protoFee)
	}
	if liqFee.Cmp(big.NewInt(1250)) != 0 {
		t.Errorf("liqFee with 50%% take = %s, want 1250", liqFee)
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
	lots := big.NewInt(100)
	liq := LotsToLiquidity(lots)
	expected := big.NewInt(100 * 1024) // 100 << 10
	if liq.Cmp(expected) != 0 {
		t.Errorf("LotsToLiquidity(100) = %s, want %s", liq, expected)
	}

	// Odd lots (knockout flag) should mask out the flag
	oddLots := big.NewInt(101) // bit 0 set
	liq = LotsToLiquidity(oddLots)
	expected = big.NewInt(100 * 1024) // 100 << 10 (flag cleared)
	if liq.Cmp(expected) != 0 {
		t.Errorf("LotsToLiquidity(101) = %s, want %s", liq, expected)
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
	mezz := MezzKey(tick)
	mezzB := MezzBit(tick)
	termB := TermBit(tick)

	reconstructed := WeldLobbyMezzTerm(lobby, mezzB, termB)
	if reconstructed != tick {
		t.Errorf("weld roundtrip: got %d, want %d (lobby=%d mezz=%d mezzBit=%d termBit=%d)",
			reconstructed, tick, lobby, mezz, mezzB, termB)
	}
}
