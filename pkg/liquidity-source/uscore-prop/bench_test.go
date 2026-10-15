package uscoreprop

import (
	"math"
	"math/big"
	"os"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// realPool is the USDC/WHYPE pool as tracked on HyperEVM block ~47845700.
func realPool(tb testing.TB) entity.Pool {
	tb.Helper()
	data, err := os.ReadFile("testdata/pool_usdc_whype.json")
	require.NoError(tb, err)
	var p entity.Pool
	require.NoError(tb, json.Unmarshal(data, &p))
	return p
}

func realSim(tb testing.TB) *PoolSimulator {
	tb.Helper()
	sim, err := NewPoolSimulatorWith(realPool(tb), math.MaxInt64)
	require.NoError(tb, err)
	return sim
}

// 50 USDC in, a mid-curve amount between two samples.
func realQuote(sim *PoolSimulator) pool.CalcAmountOutParams {
	return pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: sim.Info.Tokens[0], Amount: big.NewInt(50_123_456)},
		TokenOut:      sim.Info.Tokens[1],
	}
}

func BenchmarkNewPoolSimulator(b *testing.B) {
	p := realPool(b)
	for b.Loop() {
		_, _ = NewPoolSimulatorWith(p, math.MaxInt64)
	}
}

func BenchmarkCalcAmountOut(b *testing.B) {
	sim := realSim(b)
	q := realQuote(sim)
	for b.Loop() {
		_, _ = sim.CalcAmountOut(q)
	}
}

func BenchmarkCloneUpdateBalance(b *testing.B) {
	sim := realSim(b)
	q := realQuote(sim)
	res, err := sim.CalcAmountOut(q)
	require.NoError(b, err)
	params := pool.UpdateBalanceParams{TokenAmountIn: q.TokenAmountIn, TokenAmountOut: *res.TokenAmountOut, Fee: *res.Fee}
	for b.Loop() {
		sim.CloneState().(*PoolSimulator).UpdateBalance(params)
	}
}
