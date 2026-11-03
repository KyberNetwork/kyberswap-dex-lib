package dexv2

import (
	"math/big"
	"os"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// The 1000 USDC output was independently verified against the deployed contract
// at block 26332115 and that block's timestamp using a forked EVM simulation.
func snapshotSimulator(t *testing.T) (*PoolSimulator, entity.Pool) {
	t.Helper()
	b, err := os.ReadFile("testdata/ethereum_d3_snapshot.json")
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Extra       Extra
		StaticExtra StaticExtra
		Address     string
	}
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatal(err)
	}
	f.StaticExtra.TrackLiquidityLimits = true
	// Historical price fixture predates liquidity tracking; use ample synthetic
	// limits so this test isolates the independently verified swap math.
	for i := range 2 {
		f.Extra.FloatBalance[i] = big.NewInt(0)
		f.Extra.SwapOutWithdraw[i] = big.NewInt(1_000_000_000_000)
		f.Extra.SwapOutBorrow[i] = big.NewInt(1_000_000_000_000)
		f.Extra.LiquidityDebt[i] = big.NewInt(1_000_000_000_000)
	}
	extra, _ := json.Marshal(f.Extra)
	static, _ := json.Marshal(f.StaticExtra)
	p := entity.Pool{Address: f.Address, Extra: string(extra), StaticExtra: string(static), Tokens: []*entity.PoolToken{{Address: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", Decimals: 6}, {Address: "0xdac17f958d2ee523a2206206994597c13d831ec7", Decimals: 6}}, Reserves: []string{"10000000000", "10000000000"}}
	sim, err := NewPoolSimulator(p, valueobject.ChainIDEthereum)
	if err != nil {
		t.Fatal(err)
	}
	return sim, p
}

func TestEthereumD3Snapshot(t *testing.T) {
	sim, p := snapshotSimulator(t)
	for _, tc := range []struct {
		in           int
		amount, want int64
	}{{0, 1000000000, 1000353801}, {1, 10000000, 9990187}} {
		q, err := sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: p.Tokens[tc.in].Address, Amount: big.NewInt(tc.amount)}, TokenOut: p.Tokens[1-tc.in].Address})
		if err != nil {
			t.Fatal(err)
		}
		if q.TokenAmountOut.Amount.Cmp(big.NewInt(tc.want)) != 0 {
			t.Fatalf("direction %d: got %s want %d", tc.in, q.TokenAmountOut.Amount, tc.want)
		}
	}
}

func TestD3SettlementConstraints(t *testing.T) {
	sim, p := snapshotSimulator(t)
	params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: p.Tokens[0].Address, Amount: big.NewInt(1_000_000_000)}, TokenOut: p.Tokens[1].Address}
	first, err := sim.CalcAmountOut(params)
	if err != nil {
		t.Fatal(err)
	}
	again, err := sim.CalcAmountOut(params)
	if err != nil || first.TokenAmountOut.Amount.Cmp(again.TokenAmountOut.Amount) != 0 {
		t.Fatalf("repeated quote differs: %v", err)
	}
	sim.extra.WithdrawPaused[1] = true
	if _, err := sim.CalcAmountOut(params); err != ErrUnsupportedLiquidityState {
		t.Fatalf("withdraw pause: %v", err)
	}
	sim.extra.WithdrawPaused[1] = false
	sim.extra.SupplyPaused[0] = true
	if _, err := sim.CalcAmountOut(params); err != ErrUnsupportedLiquidityState {
		t.Fatalf("supply pause: %v", err)
	}
	sim.extra.SupplyPaused[0] = false
	sim.liquidityOut[1].SetUint64(1)
	if _, err := sim.CalcAmountOut(params); err != ErrUnsupportedLiquidityState {
		t.Fatalf("output limit: %v", err)
	}
}

func TestD3SharedLimitAndClone(t *testing.T) {
	sim, p := snapshotSimulator(t)
	clone := sim.CloneState().(*PoolSimulator)
	limit := swaplimit.NewInventory(DexType, sim.CalculateLimit())
	params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: p.Tokens[0].Address, Amount: big.NewInt(1_000_000_000)}, TokenOut: p.Tokens[1].Address, Limit: limit}
	q, err := sim.CalcAmountOut(params)
	if err != nil {
		t.Fatal(err)
	}
	original := new(big.Int).Set(limit.GetLimit(sim.limitKey(1, false)))
	sim.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: params.TokenAmountIn, TokenAmountOut: *q.TokenAmountOut, SwapInfo: q.SwapInfo, SwapLimit: limit})
	// SwapInfo belongs to the quote; subsequent caller mutation must not change state.
	si := q.SwapInfo.(SwapInfo)
	stateLiquidity := new(uint256.Int).Set(sim.V3Pool.Liquidity)
	si.nextStateLiquidity.SetUint64(0)
	if sim.V3Pool.Liquidity.Cmp(stateLiquidity) != 0 {
		t.Fatal("updated state aliases quote liquidity")
	}
	want := new(big.Int).Sub(original, q.TokenAmountOut.Amount)
	if got := limit.GetLimit(sim.limitKey(1, false)); got.Cmp(want) != 0 {
		t.Fatalf("limit %s, want %s", got, want)
	}
	cloneQuote, err := clone.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: params.TokenAmountIn, TokenOut: params.TokenOut})
	if err != nil || cloneQuote.TokenAmountOut.Amount.Cmp(q.TokenAmountOut.Amount) != 0 {
		t.Fatalf("clone changed: %v", err)
	}
}

func TestLiquidityTrackingVersionGate(t *testing.T) {
	_, p := snapshotSimulator(t)
	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		t.Fatal(err)
	}
	extra.FloatBalance[0] = nil
	b, err := json.Marshal(extra)
	if err != nil {
		t.Fatal(err)
	}
	p.Extra = string(b)
	if _, err = NewPoolSimulator(p, valueobject.ChainIDEthereum); err != ErrUnsupportedLiquidityState {
		t.Fatalf("tracked state must fail closed: %v", err)
	}
	var static StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &static); err != nil {
		t.Fatal(err)
	}
	static.TrackLiquidityLimits = false
	b, err = json.Marshal(static)
	if err != nil {
		t.Fatal(err)
	}
	p.StaticExtra = string(b)
	sim, err := NewPoolSimulator(p, valueobject.ChainIDEthereum)
	if err != nil {
		t.Fatal(err)
	}
	if sim.CalculateLimit() != nil {
		t.Fatal("legacy state unexpectedly registered a shared limit")
	}
	if _, err := sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: p.Tokens[0].Address, Amount: big.NewInt(1_000_000_000)}, TokenOut: p.Tokens[1].Address}); err != nil {
		t.Fatal(err)
	}
}

func TestFetchDynamicFeeFlagRejected(t *testing.T) {
	_, p := snapshotSimulator(t)
	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		t.Fatal(err)
	}
	extra.DexVariables2.SetBit(extra.DexVariables2, 141, 1)
	b, err := json.Marshal(extra)
	if err != nil {
		t.Fatal(err)
	}
	p.Extra = string(b)
	sim, err := NewPoolSimulator(p, valueobject.ChainIDEthereum)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sim.CalcAmountOut(pool.CalcAmountOutParams{}); err != ErrUnsupportedDynamicFee {
		t.Fatalf("got %v", err)
	}
}
