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
