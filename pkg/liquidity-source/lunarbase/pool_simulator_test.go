package lunarbase

import (
	"math/big"
	"strings"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// TestCloneStateUpdateBalance verifies UpdateBalance mutates only the cloned
// reserves and never the original. SqrtPriceX96 is operator-set on the
// fix/incident contract and is never written by a swap, so we check it
// stays put on both copies.
func TestCloneStateUpdateBalance(t *testing.T) {
	wrappedNative := strings.ToLower(valueobject.WrappedNativeMap[valueobject.ChainIDBase])

	extraBytes, err := json.Marshal(Extra{
		SqrtPriceX96:      uint256.NewInt(1),
		FeeAskX24:         0,
		FeeBidX24:         1,
		LatestUpdateBlock: 1,
		ConcentrationK:    5000,
	})
	if err != nil {
		t.Fatalf("marshal extra: %v", err)
	}

	staticExtraBytes, err := json.Marshal(StaticExtra{
		HasNative: true,
	})
	if err != nil {
		t.Fatalf("marshal static extra: %v", err)
	}

	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: entity.Pool{
		Address:  "0x00003bf45ce34bf1bea78669f9a40ee630e11b99",
		Exchange: DexType,
		Type:     DexType,
		Reserves: entity.PoolReserves{"100", "200"},
		Tokens: []*entity.PoolToken{
			{Address: wrappedNative, Decimals: 18},
			{Address: "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913", Decimals: 6},
		},
		Extra:       string(extraBytes),
		StaticExtra: string(staticExtraBytes),
	}, ChainID: valueobject.ChainIDBase})
	if err != nil {
		t.Fatalf("new simulator: %v", err)
	}

	cloned := sim.CloneState()
	cloned.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: sim.GetTokens()[0], Amount: big.NewInt(10)},
		TokenAmountOut: pool.TokenAmount{Token: sim.GetTokens()[1], Amount: big.NewInt(20)},
		Fee:            pool.TokenAmount{Token: sim.GetTokens()[1], Amount: big.NewInt(0)},
		SwapInfo: SwapInfo{
			nextSqrtPriceX96: uint256.NewInt(2),
		},
	})

	if sim.GetReserves()[0].Cmp(big.NewInt(100)) != 0 || sim.GetReserves()[1].Cmp(big.NewInt(200)) != 0 {
		t.Fatalf("original reserves mutated: got %s/%s", sim.GetReserves()[0], sim.GetReserves()[1])
	}
	if sim.SqrtPriceX96.Uint64() != 1 {
		t.Fatalf("original price mutated: got %d", sim.SqrtPriceX96.Uint64())
	}
	if cloned.(*PoolSimulator).SqrtPriceX96.Uint64() != 1 {
		t.Fatalf("cloned price unexpectedly mutated (swaps must not move SqrtPriceX96): got %d",
			cloned.(*PoolSimulator).SqrtPriceX96.Uint64())
	}

	meta := sim.GetMetaInfo(sim.GetTokens()[1], sim.GetTokens()[0]).(PoolMeta)
	if meta.ApprovalAddress != strings.ToLower("0x00003bf45ce34bf1bea78669f9a40ee630e11b99") {
		t.Fatalf("unexpected approval address: got %s", meta.ApprovalAddress)
	}
}

// TestNewPoolSimulatorRejectsStalePool exercises the route-finding path
// (Opts.StaleCheck=true). The pool is `BlockNumber - LatestUpdateBlock > BlockDelay`
// so the constructor must refuse to build a simulator.
func TestNewPoolSimulatorRejectsStalePool(t *testing.T) {
	wrappedNative := strings.ToLower(valueobject.WrappedNativeMap[valueobject.ChainIDBase])

	pX96 := new(uint256.Int).Lsh(uint256.NewInt(1), 96)
	extraBytes, err := json.Marshal(Extra{
		SqrtPriceX96:      pX96,
		FeeAskX24:         0,
		FeeBidX24:         1,
		LatestUpdateBlock: 10,
		BlockDelay:        2,
		ConcentrationK:    5000,
	})
	if err != nil {
		t.Fatalf("marshal extra: %v", err)
	}

	staticExtraBytes, err := json.Marshal(StaticExtra{
		HasNative: true,
	})
	if err != nil {
		t.Fatalf("marshal static extra: %v", err)
	}

	_, err = NewPoolSimulator(pool.FactoryParams{
		EntityPool: entity.Pool{
			Address:     "0x00003bf45ce34bf1bea78669f9a40ee630e11b99",
			Exchange:    DexType,
			Type:        DexType,
			BlockNumber: 13, // 13 - 10 > 2 → stale
			Reserves:    entity.PoolReserves{"1000000000000000000000", "1000000000000000000000"},
			Tokens: []*entity.PoolToken{
				{Address: wrappedNative, Decimals: 18},
				{Address: "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913", Decimals: 6},
			},
			Extra:       string(extraBytes),
			StaticExtra: string(staticExtraBytes),
		},
		ChainID: valueobject.ChainIDBase,
		Opts: pool.FactoryOpts{
			StaleCheck: true,
		},
	})
	assert.ErrorIs(t, err, ErrStalePool)
}

// TestPunishmentModelPersistsFeeThroughUpdateBalance exercises the full
// CalcAmountOut → UpdateBalance round trip for a punishment-model pool
// (ConcentrationK == 0, MaxPunishmentX24 > 0). It reuses the on-chain fixture
// from TestPunishmentQuoteMatchesOnChain to verify the pool simulator, not
// just the underlying math, persists the ratcheted fee: a route splitting a
// swap into two hops through the same pool must price the second hop against
// the post-first-hop fee, and cloned simulators used by other route
// candidates must not see it.
func TestPunishmentModelPersistsFeeThroughUpdateBalance(t *testing.T) {
	wrappedNative := strings.ToLower(valueobject.WrappedNativeMap[valueobject.ChainIDBSC])

	extraBytes, err := json.Marshal(Extra{
		SqrtPriceX96:     uint256.MustFromDecimal("2124565750896485315338783686656"),
		FeeAskX24:        7975,
		FeeBidX24:        2188,
		MaxPunishmentX24: 83886,
	})
	if err != nil {
		t.Fatalf("marshal extra: %v", err)
	}

	staticExtraBytes, err := json.Marshal(StaticExtra{})
	if err != nil {
		t.Fatalf("marshal static extra: %v", err)
	}

	newSim := func() *PoolSimulator {
		sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: entity.Pool{
			Address:  "0x00007904d186680c709519e71f4dc3e2df8f1b99",
			Exchange: DexType,
			Type:     DexType,
			Reserves: entity.PoolReserves{"39134580821500176234", "42770804199297762732014"},
			Tokens: []*entity.PoolToken{
				{Address: wrappedNative, Decimals: 18},
				{Address: "0x55d398326f99059ff775485246999027b3197955", Decimals: 18},
			},
			Extra:       string(extraBytes),
			StaticExtra: string(staticExtraBytes),
		}, ChainID: valueobject.ChainIDBSC})
		if err != nil {
			t.Fatalf("new simulator: %v", err)
		}
		return sim
	}

	sim := newSim()
	cloned := sim.CloneState().(*PoolSimulator)

	dx := big.NewInt(0).SetInt64(1_000_000_000_000_000_000) // 1e18, matches TestPunishmentQuoteMatchesOnChain
	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: sim.GetTokens()[0], Amount: dx},
		TokenOut:      sim.GetTokens()[1],
	})
	if err != nil {
		t.Fatalf("first hop CalcAmountOut: %v", err)
	}
	if got, want := result.TokenAmountOut.Amount.String(), "718956327424094777629"; got != want {
		t.Fatalf("first hop amountOut = %s, want %s", got, want)
	}

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: sim.GetTokens()[0], Amount: dx},
		TokenAmountOut: pool.TokenAmount{Token: sim.GetTokens()[1], Amount: result.TokenAmountOut.Amount},
		Fee:            *result.Fee,
		SwapInfo:       result.SwapInfo,
	})

	if got, want := sim.FeeBidX24, uint32(3039); got != want {
		t.Errorf("post-swap FeeBidX24 = %d, want %d (punishment must persist through UpdateBalance)", got, want)
	}
	if got, want := sim.FeeAskX24, uint32(7975); got != want {
		t.Errorf("post-swap FeeAskX24 = %d, want unchanged %d", got, want)
	}

	if got, want := cloned.FeeBidX24, uint32(2188); got != want {
		t.Errorf("cloned simulator FeeBidX24 = %d, want unchanged %d (must not see the other route's mutation)", got, want)
	}

	// A second hop through the same (now-mutated) pool must price against the
	// ratcheted fee, not the original — this is what a router split relies on.
	secondHop, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: sim.GetTokens()[0], Amount: dx},
		TokenOut:      sim.GetTokens()[1],
	})
	if err != nil {
		t.Fatalf("second hop CalcAmountOut: %v", err)
	}
	if secondHop.Fee.Amount.Cmp(result.Fee.Amount) <= 0 {
		t.Errorf("second hop fee %s must exceed first hop fee %s (punishment ratchets up)",
			secondHop.Fee.Amount, result.Fee.Amount)
	}
}
