package stable

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v2/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

const (
	cloneTokenA = "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0"
	cloneTokenB = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
)

func newCloneTestPool(t *testing.T, address string, reserves ...string) *PoolSimulator {
	t.Helper()
	poolStr := `{
		"address": "` + address + `",
		"exchange": "balancer-v2-stable",
		"type": "balancer-v2-stable",
		"reserves": ["` + reserves[0] + `", "` + reserves[1] + `"],
		"tokens": [
			{"address": "` + cloneTokenA + `", "decimals": 18, "swappable": true},
			{"address": "` + cloneTokenB + `", "decimals": 18, "swappable": true}
		],
		"extra": "{\"amp\":\"0xf4240\",\"swapFeePercentage\":\"0x16bcc41e90000\",\"scalingFactors\":[\"0xFFB10F9BCF7D41A\",\"0xde0b6b3a7640000\"],\"paused\":false}",
		"staticExtra": "{\"poolId\":\"0x851523a36690bf267bbfec389c823072d82921a90002000000000000000001ed\",\"poolType\":\"MetaStable\",\"poolTypeVersion\":1,\"poolSpecialization\":2,\"vault\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\"}"
	}`
	var entityPool entity.Pool
	require.NoError(t, json.Unmarshal([]byte(poolStr), &entityPool))
	sim, err := NewPoolSimulator(entityPool, nil)
	require.NoError(t, err)
	return sim
}

func cloneQuote(t *testing.T, sim pool.IPoolSim, amountIn uint64) string {
	t.Helper()
	res, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmtOutResult, error) {
		return sim.CalcAmtOut(pool.CalcAmtOutParams{
			TokenAmountIn: pool.TokenAmt{Token: cloneTokenA, Amount: *uint256.NewInt(amountIn)},
			TokenOut:      cloneTokenB,
		})
	})
	require.NoError(t, err)
	return res.TokenAmountOut.Amount.String()
}

// TestCloneStateQuotesIdentically checks that a clone quotes the same as its original before any update.
func TestCloneStateQuotesIdentically(t *testing.T) {
	t.Parallel()
	orig := newCloneTestPool(t, "0xpool", "1152882153159026494", "873225053252443292")
	clone, ok := orig.CloneState().(*PoolSimulator)
	require.True(t, ok)

	for _, amountIn := range []uint64{1e15, 1e16, 1e17} {
		require.Equal(t, cloneQuote(t, orig, amountIn), cloneQuote(t, clone, amountIn), "amountIn=%d", amountIn)
	}
}

// TestCloneStateIsIndependent checks that UpdateState on a clone changes neither the original's reserves
// nor its quotes.
func TestCloneStateIsIndependent(t *testing.T) {
	t.Parallel()
	orig := newCloneTestPool(t, "0xpool", "1152882153159026494", "873225053252443292")
	before := cloneQuote(t, orig, 1e16)

	clone, ok := orig.CloneState().(*PoolSimulator)
	require.True(t, ok)
	clone.UpdateState(pool.UpdateStateParams{
		TokenAmountIn:  pool.TokenAmt{Token: cloneTokenA, Amount: *uint256.NewInt(1e18)},
		TokenAmountOut: pool.TokenAmt{Token: cloneTokenB, Amount: *uint256.NewInt(1e17)},
		SwapInfo:       shared.SwapInfo{},
	})

	require.Equal(t, big.NewInt(1152882153159026494).Cmp(orig.Info.Reserves[0]), 0, "original reserve 0 changed")
	require.Equal(t, big.NewInt(873225053252443292).Cmp(orig.Info.Reserves[1]), 0, "original reserve 1 changed")
	require.Equal(t, before, cloneQuote(t, orig, 1e16), "original quote changed after clone update")
	require.NotEqual(t, before, cloneQuote(t, clone, 1e16), "clone quote did not change after its update")
}

// TestCloneStateBasePoolIsIndependent checks that a clone's update of a nested base pool does not write into
// the original's base pool.
func TestCloneStateBasePoolIsIndependent(t *testing.T) {
	t.Parallel()
	base := newCloneTestPool(t, "0xbase", "1000000000000000000", "2000000000000000000")
	orig := newCloneTestPool(t, "0xparent", "3000000000000000000", "4000000000000000000")
	orig.basePools = map[string]shared.IBasePool{"0xbase": base}

	clone, ok := orig.CloneState().(*PoolSimulator)
	require.True(t, ok)
	clone.basePools["0xbase"].UpdateState(pool.UpdateStateParams{
		SwapInfo: shared.SwapInfo{Hops: []shared.Hop{{
			Pool: "0xleaf", TokenIn: cloneTokenA, TokenOut: cloneTokenB,
			AmountIn: uint256.NewInt(5), AmountOut: uint256.NewInt(7),
		}}},
	})

	require.Equal(t, big.NewInt(1000000000000000000).Cmp(base.Info.Reserves[0]), 0, "original base reserve 0 changed")
	require.Equal(t, big.NewInt(2000000000000000000).Cmp(base.Info.Reserves[1]), 0, "original base reserve 1 changed")
	require.NotSame(t, base, clone.basePools["0xbase"])
}
