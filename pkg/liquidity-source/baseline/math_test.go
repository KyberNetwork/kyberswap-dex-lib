package baseline

import (
	"errors"
	"math/big"
	"testing"

	"github.com/holiman/uint256"
)

func TestExpWadBIMatchesSoladyVectors(t *testing.T) {
	tests := []struct {
		x    string
		want string
	}{
		{"-41446531673892822312", "1"},
		{"-41446531673892822313", "0"},
		{"-3000000000000000000", "49787068367863942"},
		{"-2000000000000000000", "135335283236612691"},
		{"-1000000000000000000", "367879441171442321"},
		{"0", "1000000000000000000"},
		{"1000000000000000000", "2718281828459045235"},
		{"2000000000000000000", "7389056098930650227"},
		{"3000000000000000000", "20085536923187667741"},
		{"10000000000000000000", "22026465794806716516980"},
	}

	for _, tt := range tests {
		got, err := expWadBI(mustTestBI(t, tt.x))
		if err != nil {
			t.Fatalf("expWadBI(%s) failed: %v", tt.x, err)
		}
		assertTestBIEqual(t, "expWadBI("+tt.x+")", tt.want, got)
	}
}

func TestLnWadBIMatchesSoladyVectors(t *testing.T) {
	tests := []struct {
		x    string
		want string
	}{
		{"1", "-41446531673892822313"},
		{"42", "-37708862055609454007"},
		{"10000", "-32236191301916639577"},
		{"1000000000", "-20723265836946411157"},
		{"2718281828459045235", "999999999999999999"},
		{"11723640096265400935", "2461607324344817918"},
		{"340282366920938463463374607431768211456", "47276307437780177293"},
	}

	for _, tt := range tests {
		got, err := lnWadBI(mustTestBI(t, tt.x))
		if err != nil {
			t.Fatalf("lnWadBI(%s) failed: %v", tt.x, err)
		}
		assertTestBIEqual(t, "lnWadBI("+tt.x+")", tt.want, got)
	}
}

func TestLnWadBIRejectsValuesAboveInt256Domain(t *testing.T) {
	x := new(big.Int).Lsh(big.NewInt(1), 300)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("lnWadBI panicked: %v", r)
		}
	}()

	if _, err := lnWadBI(x); !errors.Is(err, errInvalidCurveState) {
		t.Fatalf("lnWadBI(1 << 300) error mismatch: want=%v got=%v", errInvalidCurveState, err)
	}
}

func TestApplyQuoteStateSettlesPendingSurplusBelowSafety(t *testing.T) {
	state := &QuoteState{
		TotalSupply:             uint256.MustFromDecimal("100000000000000000000"),
		TotalBTokens:            uint256.MustFromDecimal("94000000000000000000"),
		TotalReserves:           uint256.MustFromDecimal("1000"),
		PendingSurplus:          uint256.MustFromDecimal("25"),
		SettlePendingSurplus:    true,
		LiquidityFeePct:         uint256.MustFromDecimal("1000000000000000000"),
		QuoteBlockBuyDeltaCirc:  uint256.NewInt(0),
		QuoteBlockSellDeltaCirc: uint256.NewInt(0),
	}

	applyQuoteState(state, big.NewInt(10), big.NewInt(-100), big.NewInt(3))

	assertTestBIEqual(t, "total reserves", "1122", uToBI(state.TotalReserves))
	assertTestBIEqual(t, "pending surplus", "3", uToBI(state.PendingSurplus))
	if state.SettlePendingSurplus {
		t.Fatal("expected pending surplus settlement flag to clear")
	}
}

func TestApplyQuoteStateClearsPendingSurplusAboveSafety(t *testing.T) {
	state := &QuoteState{
		TotalSupply:             uint256.MustFromDecimal("100000000000000000000"),
		TotalBTokens:            uint256.MustFromDecimal("96000000000000000000"),
		TotalReserves:           uint256.MustFromDecimal("1000"),
		PendingSurplus:          uint256.MustFromDecimal("25"),
		SettlePendingSurplus:    true,
		LiquidityFeePct:         uint256.MustFromDecimal("1000000000000000000"),
		QuoteBlockBuyDeltaCirc:  uint256.NewInt(0),
		QuoteBlockSellDeltaCirc: uint256.NewInt(0),
	}

	applyQuoteState(state, big.NewInt(10), big.NewInt(-100), big.NewInt(3))

	assertTestBIEqual(t, "total reserves", "1097", uToBI(state.TotalReserves))
	assertTestBIEqual(t, "pending surplus", "0", uToBI(state.PendingSurplus))
	if state.SettlePendingSurplus {
		t.Fatal("expected pending surplus settlement flag to clear")
	}
}

func TestApplyQuoteStateDoesNotSettleLocallyRecordedPendingSurplusSameBlock(t *testing.T) {
	state := &QuoteState{
		TotalSupply:             uint256.MustFromDecimal("100000000000000000000"),
		TotalBTokens:            uint256.MustFromDecimal("94000000000000000000"),
		TotalReserves:           uint256.MustFromDecimal("1000"),
		PendingSurplus:          uint256.NewInt(0),
		LiquidityFeePct:         uint256.MustFromDecimal("1000000000000000000"),
		QuoteBlockBuyDeltaCirc:  uint256.NewInt(0),
		QuoteBlockSellDeltaCirc: uint256.NewInt(0),
	}

	applyQuoteState(state, big.NewInt(10), big.NewInt(-100), big.NewInt(3))
	applyQuoteState(state, big.NewInt(10), big.NewInt(-100), big.NewInt(3))

	assertTestBIEqual(t, "total reserves", "1194", uToBI(state.TotalReserves))
	assertTestBIEqual(t, "pending surplus", "6", uToBI(state.PendingSurplus))
}

func TestRPCQuoteStateDoesNotInferPendingSurplusSettlement(t *testing.T) {
	state := rpcQuoteState{
		TotalSupply:             mustTestBI(t, "100000000000000000000"),
		TotalBTokens:            mustTestBI(t, "94000000000000000000"),
		TotalReserves:           mustTestBI(t, "1000"),
		PendingSurplus:          mustTestBI(t, "25"),
		LiquidityFeePct:         wadBI,
		QuoteBlockBuyDeltaCirc:  big.NewInt(1),
		QuoteBlockSellDeltaCirc: big.NewInt(0),
	}.toQuoteState()

	if state.SettlePendingSurplus {
		t.Fatal("current-block pending surplus must not be inferred as stale settlement")
	}
}

func mustTestBI(t *testing.T, s string) *big.Int {
	t.Helper()
	x, ok := new(big.Int).SetString(s, 10)
	if !ok {
		t.Fatalf("invalid big.Int test value %q", s)
	}
	return x
}

func assertTestBIEqual(t *testing.T, label, want string, got *big.Int) {
	t.Helper()
	expected := mustTestBI(t, want)
	if got.Cmp(expected) != 0 {
		t.Fatalf("%s mismatch: want=%s got=%s", label, expected, got)
	}
}
