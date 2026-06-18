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
