package dexv2

import (
	"fmt"
	"math/big"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const goldenFile = "testdata/golden_quotes.txt"

func d4Snapshot(t *testing.T) (*PoolSimulator, entity.Pool) {
	t.Helper()
	b, err := os.ReadFile("testdata/ethereum_d4_dynamic_snapshot.json")
	require.NoError(t, err)
	var f struct {
		Extra       Extra
		StaticExtra StaticExtra
		Address     string
		Tokens      []string
	}
	require.NoError(t, json.Unmarshal(b, &f))
	extra, _ := json.Marshal(f.Extra)
	static, _ := json.Marshal(f.StaticExtra)
	p := entity.Pool{Address: f.Address, Extra: string(extra), StaticExtra: string(static),
		Tokens:   []*entity.PoolToken{{Address: f.Tokens[0], Decimals: 6}, {Address: f.Tokens[1], Decimals: 6}},
		Reserves: []string{"10000000000", "10000000000"}}
	sim, err := NewPoolSimulator(p, valueobject.ChainIDEthereum)
	require.NoError(t, err)
	return sim, p
}

func dumpLimits(sim *PoolSimulator) string {
	limits := sim.CalculateLimit()
	keys := make([]string, 0, len(limits))
	for k := range limits {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&sb, "%s=%s;", k[strings.LastIndexByte(k, ':')+1:], limits[k])
	}
	return sb.String()
}

// goldenTrace quotes both directions at several sizes and then runs sequential
// quote -> UpdateBalance chains, so refactors that touch the simulator's math,
// state or clone behavior are caught by a byte-for-byte comparison.
func goldenTrace(t *testing.T) string {
	var sb strings.Builder
	sims := []struct {
		name string
		mk   func() (*PoolSimulator, entity.Pool)
	}{
		{"d3-static", func() (*PoolSimulator, entity.Pool) { return snapshotSimulator(t) }},
		{"d3-dynamic7", func() (*PoolSimulator, entity.Pool) { return dynamicSnapshot(t, 7) }},
		{"d4-dynamic", func() (*PoolSimulator, entity.Pool) { return d4Snapshot(t) }},
	}
	amounts := []int64{10_000, 123_456, 10_000_000, 100_000_000, 1_000_000_000, 50_000_000_000}
	for _, s := range sims {
		sim, p := s.mk()
		for dir := range 2 {
			for _, a := range amounts {
				q, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: p.Tokens[dir].Address, Amount: big.NewInt(a)},
					TokenOut:      p.Tokens[1-dir].Address})
				if err != nil {
					fmt.Fprintf(&sb, "%s dir=%d in=%d err=%v\n", s.name, dir, a, err)
					continue
				}
				fmt.Fprintf(&sb, "%s dir=%d in=%d out=%s gas=%d\n", s.name, dir, a, q.TokenAmountOut.Amount, q.Gas)
			}
		}
		// Sequential chains share one inventory and mutate the simulator.
		chain := sim.CloneState().(*PoolSimulator)
		limit := swaplimit.NewInventory(DexType, chain.CalculateLimit())
		for i, step := range []struct {
			dir    int
			amount int64
		}{{0, 100_000_000}, {0, 100_000_000}, {1, 30_000_000}, {0, 7_777_777}, {1, 250_000_000}} {
			in := pool.TokenAmount{Token: p.Tokens[step.dir].Address, Amount: big.NewInt(step.amount)}
			q, err := chain.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: in, TokenOut: p.Tokens[1-step.dir].Address, Limit: limit})
			if err != nil {
				fmt.Fprintf(&sb, "%s chain[%d] dir=%d in=%d err=%v\n", s.name, i, step.dir, step.amount, err)
				continue
			}
			chain.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: in, TokenAmountOut: *q.TokenAmountOut, SwapInfo: q.SwapInfo, SwapLimit: limit})
			fmt.Fprintf(&sb, "%s chain[%d] dir=%d in=%d out=%s gas=%d reserves=%s,%s tick=%d sqrt=%s liq=%s dyn=%s limits=%s\n",
				s.name, i, step.dir, step.amount, q.TokenAmountOut.Amount, q.Gas,
				chain.tokenReserves[0].Dec(), chain.tokenReserves[1].Dec(), chain.V3Pool.TickCurrent, chain.V3Pool.SqrtRatioX96, chain.V3Pool.Liquidity,
				chain.V3Pool.dynamicPacked.Dec(), dumpLimits(chain))
		}
		// The original is untouched by the chain on its clone.
		fmt.Fprintf(&sb, "%s original-limits=%s\n", s.name, dumpLimits(sim))
	}
	return sb.String()
}

func TestGoldenQuotes(t *testing.T) {
	got := goldenTrace(t)
	if os.Getenv("UPDATE_GOLDEN") != "" {
		require.NoError(t, os.WriteFile(goldenFile, []byte(got), 0o644))
	}
	want, err := os.ReadFile(goldenFile)
	require.NoError(t, err)
	require.Equal(t, string(want), got)
}
