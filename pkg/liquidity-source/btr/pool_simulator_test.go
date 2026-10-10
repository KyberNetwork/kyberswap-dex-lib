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
		name  string
		flags uint16
		want  error
	}{
		{"live core is SWAP_GATED", liveCoreFlags, ErrBTRGatedLane},
		{"clean", SwapEnabledBit, nil},
		{"halted", SwapEnabledBit | HaltBit, ErrBTRGated},
		{"swap disabled", 0, ErrBTRGated},
		{"gated only", SwapEnabledBit | SwapGatedBit, ErrBTRGatedLane},
		{"halt mask wrong bits", SwapEnabledBit | 4, ErrBTRGated},
	}
	for _, c := range cases {
		if got := gate(c.flags); got != c.want {
			t.Fatalf("%s: gate(%d)=%v want %v", c.name, c.flags, got, c.want)
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
