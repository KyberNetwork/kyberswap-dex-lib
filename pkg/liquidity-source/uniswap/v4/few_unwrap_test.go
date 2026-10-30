package uniswapv4

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	fewTestUSDC   = "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
	fewTestUSDT   = "0xdac17f958d2ee523a2206206994597c13d831ec7"
	fewTestFwUSDC = "0x0492560fa7cfd6a85e50d8be3f77318994f8f429"
	fewTestFwUSDT = "0xef87f4608e601e8564800265aee1c1ffadf73283"

	// USDC.balanceOf(fwUSDC) at Ethereum block 26303854. A 300k USDT->USDC route through
	// fwUSDT/fwUSDC pool 0x6199c1a8...dbac47 would need to unwrap ~299k fwUSDC.
	fewTestFwUSDCReserve = 132281398554
)

// newFewTestPool models fwUSDT/fwUSDC pool 0x6199c1a8...dbac47 (no hook, fee 500): full-range
// liquidity at price 1 worth ~2.5M of each side, so pool depth alone never limits these swaps
// and any rejection comes from the fwUSDC unwrap balance. fwR is the tracked Extra field
// (nil = pool not re-tracked since fwR was added).
func newFewTestPool(t *testing.T, fwR string) *PoolSimulator {
	extra := `{"liquidity":2538749704384,"sqrtPriceX96":79228162514264337593543950336,"tickSpacing":10,"tick":0,` +
		`"ticks":[{"index":-887270,"liquidityGross":2538749704384,"liquidityNet":2538749704384},` +
		`{"index":887270,"liquidityGross":2538749704384,"liquidityNet":-2538749704384}]`
	if fwR != "" {
		extra += `,"fwR":` + fwR
	}
	extra += `}`

	p, err := NewPoolSimulator(entity.Pool{
		Address:  "0x6199c1a871328a693bbc9cd80a7e4874a4a7e2ebc862b51fa04bb6b587dbac47",
		Exchange: "uniswap-v4",
		Type:     DexType,
		Reserves: []string{"2538749704384", "2538749704384"},
		Tokens: []*entity.PoolToken{
			{Address: fewTestFwUSDC, Decimals: 6, Swappable: true},
			{Address: fewTestFwUSDT, Decimals: 6, Swappable: true},
		},
		StaticExtra: `{"0x0":[false,false],"fee":500,"tS":10,"hooks":"0x0000000000000000000000000000000000000000"}`,
		Extra:       extra,
	}, valueobject.ChainIDEthereum)
	require.NoError(t, err)
	return p
}

func usdtIn(amount int64) pool.TokenAmount {
	return pool.TokenAmount{Token: fewTestUSDT, Amount: big.NewInt(amount)}
}

// The route unwraps the pool's fwUSDC output into USDC, and FewWrappedToken.unwrapTo pays that
// USDC out of the fwUSDC contract's own balance: a quote above it reverts on-chain however deep
// the pool is, so it must be rejected here rather than routed.
func TestFewUnwrap_RejectsAboveUnderlyingBalance(t *testing.T) {
	p := newFewTestPool(t, `[132281398554,500000000000]`)

	_, err := p.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: usdtIn(300_000e6), TokenOut: fewTestUSDC})
	assert.ErrorIs(t, err, ErrFewUnwrapExceeded)

	// Keeping fwUSDC skips the unwrap, so only pool depth applies.
	res, err := p.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: usdtIn(300_000e6), TokenOut: fewTestFwUSDC})
	require.NoError(t, err)
	assert.Greater(t, res.TokenAmountOut.Amount.Int64(), int64(fewTestFwUSDCReserve))

	// Exact-out unwraps exactly the requested amount.
	_, err = p.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: fewTestUSDC, Amount: big.NewInt(fewTestFwUSDCReserve + 1)},
		TokenIn:        fewTestUSDT,
	})
	assert.ErrorIs(t, err, ErrFewUnwrapExceeded)
	_, err = p.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: fewTestUSDC, Amount: big.NewInt(fewTestFwUSDCReserve)},
		TokenIn:        fewTestUSDT,
	})
	assert.NoError(t, err, "unwrapping the whole balance does not revert")
}

// A route can unwrap through the same pool more than once; each unwrap spends the same balance,
// so UpdateBalance must carry what is left into the next quote, and a clone taken before must not see it.
func TestFewUnwrap_UpdateBalanceSpendsBalance(t *testing.T) {
	p := newFewTestPool(t, `[132281398554,500000000000]`)
	clone := p.CloneState()

	// 100k USDT unwraps ~99.7k fwUSDC, within the balance.
	res, err := p.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: usdtIn(100_000e6), TokenOut: fewTestUSDC})
	require.NoError(t, err)
	p.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  usdtIn(100_000e6),
		TokenAmountOut: *res.TokenAmountOut,
		SwapInfo:       res.SwapInfo,
	})

	_, err = p.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: usdtIn(100_000e6), TokenOut: fewTestUSDC})
	assert.ErrorIs(t, err, ErrFewUnwrapExceeded, "only ~32.5k USDC is left to unwrap")

	_, err = clone.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: usdtIn(100_000e6), TokenOut: fewTestUSDC})
	assert.NoError(t, err, "clone keeps the pre-swap balance")
}

// A pool tracked before fwR existed has no balance to check against; it must keep quoting as it
// did, instead of losing every unwrap route until its next refresh.
func TestFewUnwrap_UntrackedPoolUnchanged(t *testing.T) {
	p := newFewTestPool(t, "")

	_, err := p.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: usdtIn(300_000e6), TokenOut: fewTestUSDC})
	assert.NoError(t, err)
}
