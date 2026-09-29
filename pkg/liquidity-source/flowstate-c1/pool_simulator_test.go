package flowstatec1

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// newTestPool builds a PoolSimulator from numbers taken from a live quoteBuyFromPool
// call against Market 0x8eFb662F738D0f5d9f146803FD02A36c6B67e60d, pool CASHCAT
// 0x1c8Fe931c9be6583d9a2E5C05712a0F6d1e4faeD, asset USDG, on Robinhood Chain (4663):
// quoteBuyFromPool(pool, USDG, 1e15) -> (available=true, fillableAmount=1e15,
// quoteAmount=219, feeAmount=2, feeBps=100, quoteAsset=USDG).
func newTestPool(t *testing.T, fillableAmount string) *PoolSimulator {
	t.Helper()

	staticExtra, err := json.Marshal(StaticExtra{
		Market:     "0x8efb662f738d0f5d9f146803fd02a36c6b67e60d",
		Pool:       "0x1c8fe931c9be6583d9a2e5c05712a0f6d1e4faed",
		QuoteAsset: "0x5fc5360d0400a0fd4f2af552add042d716f1d168",
	})
	require.NoError(t, err)

	extra, err := json.Marshal(Extra{
		Available:      true,
		ProbeAmount:    mustUint256(t, "1000000000000000"),
		ProbeQuoteCost: mustUint256(t, "219"),
		FillableAmount: mustUint256(t, fillableAmount),
		FeeBps:         100,
	})
	require.NoError(t, err)

	entityPool := entity.Pool{
		Address:  "0x1c8fe931c9be6583d9a2e5c05712a0f6d1e4faed-0x5fc5360d0400a0fd4f2af552add042d716f1d168",
		Exchange: "flowstate-c1",
		Type:     DexType,
		Reserves: []string{"0", fillableAmount},
		Tokens: []*entity.PoolToken{
			{Address: "0x5fc5360d0400a0fd4f2af552add042d716f1d168", Swappable: true}, // USDG
			{Address: "0x020bfc650a365f8bb26819deaabf3e21291018b4", Swappable: true}, // CASHCAT
		},
		StaticExtra: string(staticExtra),
		Extra:       string(extra),
	}

	sim, err := NewPoolSimulator(entityPool)
	require.NoError(t, err)

	return sim
}

func mustUint256(t *testing.T, s string) *uint256.Int {
	t.Helper()
	v, err := uint256.FromDecimal(s)
	require.NoError(t, err)
	return v
}

func TestCalcAmountOut_FlatRateWithinDepth(t *testing.T) {
	sim := newTestPool(t, "16304318509466102233884") // real CASHCAT balance at block 41,844,217-ish

	// Same amount as the probe: amountOut must equal the probe's token amount exactly.
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0x5fc5360d0400a0fd4f2af552add042d716f1d168",
			Amount: big.NewInt(219),
		},
		TokenOut: "0x020bfc650a365f8bb26819deaabf3e21291018b4",
	})
	require.NoError(t, err)
	require.Equal(t, big.NewInt(1_000_000_000_000_000), res.TokenAmountOut.Amount)

	// No curve impact: 10x the input should give ~10x the output (linear rate).
	res10x, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0x5fc5360d0400a0fd4f2af552add042d716f1d168",
			Amount: big.NewInt(2190),
		},
		TokenOut: "0x020bfc650a365f8bb26819deaabf3e21291018b4",
	})
	require.NoError(t, err)
	require.Equal(t, new(big.Int).Mul(res.TokenAmountOut.Amount, big.NewInt(10)), res10x.TokenAmountOut.Amount)
}

func TestCalcAmountOut_ExceedsDepth(t *testing.T) {
	// Depth capped at the probe amount itself: any request for more must fail rather
	// than silently overpromise inventory that the pool doesn't actually hold.
	sim := newTestPool(t, "1000000000000000")

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0x5fc5360d0400a0fd4f2af552add042d716f1d168",
			Amount: big.NewInt(2190), // would need 1e16 tokens out, only 1e15 available
		},
		TokenOut: "0x020bfc650a365f8bb26819deaabf3e21291018b4",
	})
	require.ErrorIs(t, err, ErrInsufficientLiquidity)
}

func TestCalcAmountOut_Unavailable(t *testing.T) {
	sim := newTestPool(t, "1000000000000000")
	sim.Available = false

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0x5fc5360d0400a0fd4f2af552add042d716f1d168",
			Amount: big.NewInt(1),
		},
		TokenOut: "0x020bfc650a365f8bb26819deaabf3e21291018b4",
	})
	require.ErrorIs(t, err, ErrPoolNotAvailable)
}

func TestCalcAmountOut_ZeroOutput(t *testing.T) {
	// A tiny input that rounds down to zero tokens out must error, not return a
	// zero-output "successful" swap (AGENTS.md: zero-output swaps must error). Probe
	// rate here is 1000 quote wei per 1 token wei, so 1 quote wei in floors to 0 out.
	staticExtra, err := json.Marshal(StaticExtra{
		Market:     "0x8efb662f738d0f5d9f146803fd02a36c6b67e60d",
		Pool:       "0x1c8fe931c9be6583d9a2e5c05712a0f6d1e4faed",
		QuoteAsset: "0x5fc5360d0400a0fd4f2af552add042d716f1d168",
	})
	require.NoError(t, err)

	extra, err := json.Marshal(Extra{
		Available:      true,
		ProbeAmount:    mustUint256(t, "1"),
		ProbeQuoteCost: mustUint256(t, "1000"),
		FillableAmount: mustUint256(t, "1000000000000000000"),
		FeeBps:         100,
	})
	require.NoError(t, err)

	sim, err := NewPoolSimulator(entity.Pool{
		Address:  "0x1c8fe931c9be6583d9a2e5c05712a0f6d1e4faed-0x5fc5360d0400a0fd4f2af552add042d716f1d168",
		Exchange: "flowstate-c1",
		Type:     DexType,
		Reserves: []string{"0", "1000000000000000000"},
		Tokens: []*entity.PoolToken{
			{Address: "0x5fc5360d0400a0fd4f2af552add042d716f1d168", Swappable: true},
			{Address: "0x020bfc650a365f8bb26819deaabf3e21291018b4", Swappable: true},
		},
		StaticExtra: string(staticExtra),
		Extra:       string(extra),
	})
	require.NoError(t, err)

	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0x5fc5360d0400a0fd4f2af552add042d716f1d168",
			Amount: big.NewInt(1),
		},
		TokenOut: "0x020bfc650a365f8bb26819deaabf3e21291018b4",
	})
	require.ErrorIs(t, err, ErrZeroAmountOut)
}
