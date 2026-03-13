package lunarbase

import (
	"math/big"
	"strings"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestCloneStateDeepCopy(t *testing.T) {
	wrappedNative := strings.ToLower(valueobject.WrappedNativeMap[valueobject.ChainIDBase])

	extraBytes, err := json.Marshal(Extra{
		PX96:              uint256.NewInt(1),
		Fee:               1,
		LatestUpdateBlock: 1,
		ConcentrationK:    5000,
	})
	if err != nil {
		t.Fatalf("marshal extra: %v", err)
	}

	staticExtraBytes, err := json.Marshal(StaticExtra{
		PeripheryAddress: defaultPeripheryAddress,
		Permit2Address:   defaultPermit2Address,
		RawTokenX:        valueobject.ZeroAddress,
		RawTokenY:        "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
		WrappedNative:    wrappedNative,
	})
	if err != nil {
		t.Fatalf("marshal static extra: %v", err)
	}

	sim, err := NewPoolSimulator(entity.Pool{
		Address:  defaultCoreAddress,
		Exchange: DexType,
		Type:     DexType,
		Reserves: entity.PoolReserves{"100", "200"},
		Tokens: []*entity.PoolToken{
			{Address: wrappedNative, Decimals: 18},
			{Address: "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913", Decimals: 6},
		},
		Extra:       string(extraBytes),
		StaticExtra: string(staticExtraBytes),
	}, valueobject.ChainIDBase)
	if err != nil {
		t.Fatalf("new simulator: %v", err)
	}

	cloned := sim.CloneState()
	cloned.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: sim.GetTokens()[0], Amount: big.NewInt(10)},
		TokenAmountOut: pool.TokenAmount{Token: sim.GetTokens()[1], Amount: big.NewInt(20)},
		Fee:            pool.TokenAmount{Token: sim.GetTokens()[1], Amount: big.NewInt(0)},
		SwapInfo: SwapInfo{
			NextPX96: uint256.NewInt(2),
		},
	})

	if sim.GetReserves()[0].Cmp(big.NewInt(100)) != 0 || sim.GetReserves()[1].Cmp(big.NewInt(200)) != 0 {
		t.Fatalf("original reserves mutated: got %s/%s", sim.GetReserves()[0], sim.GetReserves()[1])
	}
	if sim.priceX96.Uint64() != 1 {
		t.Fatalf("original price mutated: got %d", sim.priceX96.Uint64())
	}
	if cloned.(*PoolSimulator).priceX96.Uint64() != 2 {
		t.Fatalf("cloned price was not updated: got %d", cloned.(*PoolSimulator).priceX96.Uint64())
	}

	meta := sim.GetMetaInfo(sim.GetTokens()[1], sim.GetTokens()[0]).(PoolMeta)
	if meta.RouterAddress != defaultPeripheryAddress {
		t.Fatalf("unexpected router address: got %s", meta.RouterAddress)
	}
	if meta.Permit2Address != defaultPermit2Address {
		t.Fatalf("unexpected permit2 address: got %s", meta.Permit2Address)
	}
	if meta.ApprovalAddress != defaultPermit2Address {
		t.Fatalf("unexpected approval address: got %s", meta.ApprovalAddress)
	}
}

func TestCalcAmountOutReturnsInsufficientLiquidityWhenPriceIsStale(t *testing.T) {
	wrappedNative := strings.ToLower(valueobject.WrappedNativeMap[valueobject.ChainIDBase])

	extraBytes, err := json.Marshal(Extra{
		PX96:              new(uint256.Int).Lsh(uint256.NewInt(1), 96),
		Fee:               1,
		LatestUpdateBlock: 10,
		BlockDelay:        2,
		ConcentrationK:    5000,
	})
	if err != nil {
		t.Fatalf("marshal extra: %v", err)
	}

	staticExtraBytes, err := json.Marshal(StaticExtra{
		PeripheryAddress: defaultPeripheryAddress,
		Permit2Address:   defaultPermit2Address,
		RawTokenX:        valueobject.ZeroAddress,
		RawTokenY:        "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
		WrappedNative:    wrappedNative,
	})
	if err != nil {
		t.Fatalf("marshal static extra: %v", err)
	}

	sim, err := NewPoolSimulator(entity.Pool{
		Address:     defaultCoreAddress,
		Exchange:    DexType,
		Type:        DexType,
		BlockNumber: 13,
		Reserves:    entity.PoolReserves{"1000000000000000000000", "1000000000000000000000"},
		Tokens: []*entity.PoolToken{
			{Address: wrappedNative, Decimals: 18},
			{Address: "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913", Decimals: 6},
		},
		Extra:       string(extraBytes),
		StaticExtra: string(staticExtraBytes),
	}, valueobject.ChainIDBase)
	if err != nil {
		t.Fatalf("new simulator: %v", err)
	}

	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: sim.GetTokens()[0], Amount: big.NewInt(1)},
		TokenOut:      sim.GetTokens()[1],
	})
	if err != ErrInsufficientLiquidity {
		t.Fatalf("expected ErrInsufficientLiquidity, got %v", err)
	}
}
