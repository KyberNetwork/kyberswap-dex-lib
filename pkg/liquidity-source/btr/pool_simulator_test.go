package btr

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Live Monad core flag word (2026-10-08): SWAP_ENABLED|LIABILITY_SWAP|
// DEPOSIT_GATED|SWAP_GATED|COOP_ENABLED = 3632.
const liveCoreFlags uint16 = 3632

func TestGate(t *testing.T) {
	cases := []struct {
		name        string
		flags       uint16
		laneGranted bool
		want        error
	}{
		{"live core is SWAP_GATED, no lane", liveCoreFlags, false, ErrBTRGatedLane},
		{"live core is SWAP_GATED, lane granted", liveCoreFlags, true, nil},
		{"clean", SwapEnabledBit, false, nil},
		{"clean, lane granted", SwapEnabledBit, true, nil},
		{"halted, no lane", SwapEnabledBit | HaltBit, false, ErrBTRGated},
		// A lane never bypasses a halt or a missing SWAP_ENABLED.
		{"halted, lane granted", SwapEnabledBit | HaltBit, true, ErrBTRGated},
		{"halt anchor, lane granted", SwapEnabledBit | HaltAnchorBit, true, ErrBTRGated},
		{"swap disabled", 0, false, ErrBTRGated},
		{"swap disabled, lane granted", 0, true, ErrBTRGated},
		{"gated only, no lane", SwapEnabledBit | SwapGatedBit, false, ErrBTRGatedLane},
		{"gated only, lane granted", SwapEnabledBit | SwapGatedBit, true, nil},
		{"halt anchor, no lane", SwapEnabledBit | HaltAnchorBit, false, ErrBTRGated},
	}
	for _, c := range cases {
		if got := gate(c.flags, c.laneGranted); got != c.want {
			t.Fatalf("%s: gate(%d,%v)=%v want %v", c.name, c.flags, c.laneGranted, got, c.want)
		}
	}
}

func TestOpenPathAllowed(t *testing.T) {
	// The full predicate, not the weaker `(flags&5)==0 && (flags&16)!=0`
	// (which is true on the live gated word).
	if openPathAllowed(liveCoreFlags) {
		t.Fatal("live SWAP_GATED word must not be open-path allowed")
	}
	if !openPathAllowed(SwapEnabledBit) {
		t.Fatal("clean SWAP_ENABLED word must be allowed")
	}
}

func TestSelector(t *testing.T) {
	// swap_qe selector is suffix-mined; a change is a contract-level event.
	got := "0x" + hex.EncodeToString(poolABI.Methods["swap_qe"].ID)
	if got != SwapSelector {
		t.Fatalf("swap_qe selector changed: %s", got)
	}
}

func TestInterpolate(t *testing.T) {
	ladder := []Point{
		{In: "1000", Out: "999"},
		{In: "2000", Out: "1990"},
	}
	// Exact at a sampled point.
	if got, err := interpolate(ladder, big.NewInt(2000)); err != nil || got.String() != "1990" {
		t.Fatalf("exact point: got %v err %v", got, err)
	}
	// Midpoint linear: 1500 -> 999 + (1990-999)*500/1000 = 999+495 = 1494.
	if got, err := interpolate(ladder, big.NewInt(1500)); err != nil || got.String() != "1494" {
		t.Fatalf("midpoint: got %v err %v", got, err)
	}
	// Below the first point scales from the origin.
	if got, err := interpolate(ladder, big.NewInt(500)); err != nil || got.String() != "499" {
		t.Fatalf("below-first: got %v err %v", got, err)
	}
	// Above the last point fails closed.
	if _, err := interpolate(ladder, big.NewInt(2001)); err != ErrBTRAmountTooLarge {
		t.Fatalf("above-last: err %v", err)
	}
	if _, err := interpolate(nil, big.NewInt(1)); err != ErrBTRNoRoute {
		t.Fatalf("empty: err %v", err)
	}
}

func TestCalcAmountOutGated(t *testing.T) {
	extra, _ := json.Marshal(Extra{
		Flags: map[string]uint16{lowerAddr(WMON): liveCoreFlags, lowerAddr(USDC): liveCoreFlags},
	})
	ep := entity.Pool{
		Address: CreditPoolCore,
		Tokens: []*entity.PoolToken{
			{Address: USDC, Decimals: 6, Swappable: true},
			{Address: WMON, Decimals: 18, Swappable: true},
		},
		Reserves: entity.PoolReserves{"0", "0"},
		Extra:    string(extra),
	}
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: ep})
	if err != nil {
		t.Fatalf("simulator: %v", err)
	}
	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: WMON, Amount: big.NewInt(1e18)},
		TokenOut:      USDC,
	})
	if err != ErrBTRGatedLane {
		t.Fatalf("live core must return ErrBTRGatedLane, got %v", err)
	}
}

// newTestSim builds a simulator over the single BTR core with the given Extra.
func newTestSim(t *testing.T, extra Extra) *PoolSimulator {
	t.Helper()
	b, err := json.Marshal(extra)
	if err != nil {
		t.Fatalf("marshal extra: %v", err)
	}
	ep := entity.Pool{
		Address: CreditPoolCore,
		Tokens: []*entity.PoolToken{
			{Address: USDC, Decimals: 6, Swappable: true},
			{Address: WMON, Decimals: 18, Swappable: true},
		},
		Reserves: entity.PoolReserves{"0", "0"},
		Extra:    string(b),
	}
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: ep})
	if err != nil {
		t.Fatalf("simulator: %v", err)
	}
	return sim
}

func ladderExtra(flags uint16, laneGranted bool) Extra {
	return Extra{
		Flags:       map[string]uint16{lowerAddr(WMON): flags, lowerAddr(USDC): flags},
		LaneGranted: laneGranted,
		Ladders: map[string][]Point{
			pairKey(WMON, USDC): {
				{In: "1000000000000000000", Out: "42000000"},
				{In: "2000000000000000000", Out: "83000000"},
			},
		},
	}
}

// With the lane granted, a SWAP_GATED leg quotes: the on-chain pool admits the
// executor under lane 0x400, so the simulator must not refuse it.
func TestCalcAmountOutLaneGranted(t *testing.T) {
	sim := newTestSim(t, ladderExtra(liveCoreFlags, true))
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: WMON, Amount: big.NewInt(1_000_000_000_000_000_000)},
		TokenOut:      USDC,
	})
	if err != nil {
		t.Fatalf("lane-granted gated leg must quote, got %v", err)
	}
	if res.TokenAmountOut.Amount.String() != "42000000" {
		t.Fatalf("amount out = %s", res.TokenAmountOut.Amount)
	}
}

// Without the lane the same gated leg still refuses.
func TestCalcAmountOutLaneNotGranted(t *testing.T) {
	sim := newTestSim(t, ladderExtra(liveCoreFlags, false))
	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: WMON, Amount: big.NewInt(1_000_000_000_000_000_000)},
		TokenOut:      USDC,
	})
	if err != ErrBTRGatedLane {
		t.Fatalf("gated leg without a lane must return ErrBTRGatedLane, got %v", err)
	}
}

// A lane grants only the SWAP_GATED bit: HALT and a missing SWAP_ENABLED still
// refuse, whatever LaneGranted says.
func TestCalcAmountOutLaneGrantedStillRefusesClosedFlags(t *testing.T) {
	amount := big.NewInt(1_000_000_000_000_000_000)
	for _, c := range []struct {
		name  string
		flags uint16
	}{
		{"halted", SwapEnabledBit | SwapGatedBit | HaltBit},
		{"halt anchor", SwapEnabledBit | SwapGatedBit | HaltAnchorBit},
		{"swap disabled", SwapGatedBit},
	} {
		sim := newTestSim(t, ladderExtra(c.flags, true))
		_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: WMON, Amount: amount},
			TokenOut:      USDC,
		})
		if err != ErrBTRGated {
			t.Fatalf("%s: lane must not bypass, got %v want ErrBTRGated", c.name, err)
		}
	}
}

// GetMetaInfo reports Gated only when a leg is closed to this executor.
func TestGetMetaInfoLaneGranted(t *testing.T) {
	if meta := newTestSim(t, ladderExtra(liveCoreFlags, true)).GetMetaInfo(WMON, USDC).(PoolMeta); meta.Gated {
		t.Fatal("lane-granted gated core must not report Gated")
	}
	if meta := newTestSim(t, ladderExtra(liveCoreFlags, false)).GetMetaInfo(WMON, USDC).(PoolMeta); !meta.Gated {
		t.Fatal("un-granted gated core must report Gated")
	}
}

func TestCalcAmountOutUsesLadder(t *testing.T) {
	extra, _ := json.Marshal(Extra{
		Flags: map[string]uint16{lowerAddr(WMON): SwapEnabledBit, lowerAddr(USDC): SwapEnabledBit},
		Ladders: map[string][]Point{
			pairKey(WMON, USDC): {
				{In: "1000000000000000000", Out: "42000000"},
				{In: "2000000000000000000", Out: "83000000"},
			},
		},
	})
	ep := entity.Pool{
		Address: CreditPoolCore,
		Tokens: []*entity.PoolToken{
			{Address: USDC, Decimals: 6, Swappable: true},
			{Address: WMON, Decimals: 18, Swappable: true},
		},
		Reserves: entity.PoolReserves{"0", "0"},
		Extra:    string(extra),
	}
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: ep})
	if err != nil {
		t.Fatalf("simulator: %v", err)
	}
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: WMON, Amount: big.NewInt(1_000_000_000_000_000_000)},
		TokenOut:      USDC,
	})
	if err != nil {
		t.Fatalf("calc: %v", err)
	}
	if res.TokenAmountOut.Amount.String() != "42000000" {
		t.Fatalf("amount out = %s", res.TokenAmountOut.Amount)
	}
}

func TestRegistration(t *testing.T) {
	if pool.Factory(DexType) == nil {
		t.Fatal("btr pool factory not registered")
	}
}
