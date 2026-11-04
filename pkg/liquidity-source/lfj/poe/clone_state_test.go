package poe

import (
	"math/big"
	"sync"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const (
	cloneTokenA = "0x00000000000000000000000000000000000000a1"
	cloneTokenB = "0x00000000000000000000000000000000000000b2"
)

func newClonePool(t *testing.T) pool.IPoolSimulator {
	t.Helper()
	extra, err := json.Marshal(Extra{Price: uint256.NewInt(1_000_000_000_000_000_000), FeeHbps: uint256.NewInt(3000),
		Alpha: uint256.NewInt(10500), Expiry: 1 << 40})
	require.NoError(t, err)
	p, err := NewPoolSimulator(entity.Pool{Address: "0xpoe", Exchange: "lfj-poe", Type: "lfj-poe",
		Reserves: entity.PoolReserves{"1000000000000000000000", "1000000000000000000000"}, Tokens: cloneTestTokens(),
		Extra: string(extra)})
	require.NoError(t, err)
	return p
}

func cloneTestTokens() []*entity.PoolToken {
	return []*entity.PoolToken{{Address: cloneTokenA, Decimals: 18, Swappable: true},
		{Address: cloneTokenB, Decimals: 18, Swappable: true}}
}

func cloneReserveStrings(rs []*big.Int) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.String()
	}
	return out
}

// cloneSwap quotes 1e18 of token A for token B on p and applies the swap to p.
func cloneSwap(t *testing.T, p pool.IPoolSimulator) {
	tokenIn := pool.TokenAmount{Token: cloneTokenA, Amount: bignumber.TenPowInt(18)}
	res, err := p.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: tokenIn, TokenOut: cloneTokenB})
	if err != nil {
		t.Error(err)
		return
	}
	p.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: tokenIn, TokenAmountOut: *res.TokenAmountOut,
		Fee: *res.Fee, SwapInfo: res.SwapInfo})
}

// A swap applied to a clone moves the clone's reserves (GetReserves) and leaves the original's as they were.
func TestCloneStateSwapLeavesTheOriginalReserves(t *testing.T) {
	orig := newClonePool(t)
	before := cloneReserveStrings(orig.GetReserves())

	clone := orig.CloneState()
	cloneSwap(t, clone)

	require.NotEqual(t, before, cloneReserveStrings(clone.GetReserves()), "the clone's swap did not move its reserves")
	require.Equal(t, before, cloneReserveStrings(orig.GetReserves()), "a swap on a clone moved the original's reserves")
}

// Swaps applied to clones of one pool on two goroutines, as two requests of a router do, share no writes:
// under -race this test reports none.
func TestCloneStateSwapsOnClonesConcurrently(t *testing.T) {
	orig := newClonePool(t)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 50 {
				cloneSwap(t, orig.CloneState())
			}
		}()
	}
	wg.Wait()
}
