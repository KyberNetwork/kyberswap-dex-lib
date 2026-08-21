package rangepool

import (
	"math/big"
	"strings"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/base"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Offline (no-RPC) unit tests for the Range Pool simulator, built from a static
// entity.Pool snapshot of the live ROME/USDT pool captured from a real Stage-2 tracker
// run at mainnet block 25459279. The expected CalcAmountOut / CalcAmountIn amounts and
// fees below are querySwap-VERIFIED: each was cross-checked to the wei against
// Router.querySwapSingleTokenExactIn / …ExactOut at the same block when the snapshot was
// captured (the RPC parity gate lives in range-pool-parity_integration_test.go; this file
// freezes a slice of it as a fast offline regression).
//
// ROME/USDT is a 2-token 50/50 pool; both tokens are 6-decimal (scaling factor 1e12).
//
//	token 0 = ROME 0x2bd1f344a2398340c2b1119da98816ea723f5f0f
//	token 1 = USDT 0xdac17f958d2ee523a2206206994597c13d831ec7
const (
	romeToken = "0x2bd1f344a2398340c2b1119da98816ea723f5f0f"
	usdtToken = "0xdac17f958d2ee523a2206206994597c13d831ec7"

	romeUSDTSnapshot = `{"address":"0xaf037e69f0fa8d1633443cc0c67d0b73e3694b36","exchange":"range-pool","type":"range-pool","timestamp":1783169539,"reserves":["11722718035","165904197"],"tokens":[{"address":"0x2bd1f344a2398340c2b1119da98816ea723f5f0f","swappable":true},{"address":"0xdac17f958d2ee523a2206206994597c13d831ec7","swappable":true}],"extra":"{\"hook\":{},\"fee\":\"3500000000000000\",\"aggrFee\":\"400000000000000000\",\"balsE18\":[\"11722718035000000000000\",\"165904197000000000000\"],\"decs\":[\"1000000000000\",\"1000000000000\"],\"rates\":[\"1000000000000000000\",\"1000000000000000000\"],\"virtualBalances\":[\"586875103390492706063249\",\"569563619470154789309965\"],\"minimumTradeAmount\":\"1000000000000\",\"isPoolRegistered\":true,\"isPoolInitialized\":true,\"isPoolPaused\":false,\"isPoolInRecoveryMode\":false,\"isVaultPaused\":false,\"isHookStopped\":false}","staticExtra":"{\"hook\":\"0xf31e1f37e1f9c2c531e6bc3ad89ffc9206ce85d9\",\"buffs\":[\"\",\"\"],\"normalizedWeights\":[\"500000000000000000\",\"500000000000000000\"],\"decimalScalingFactors\":[\"1000000000000\",\"1000000000000\"],\"factoryAddress\":\"0x5D6D1dC0D045a8DE284C7Ab5FE83aCd7bdc5d4E0\"}","totalSupply":"5085800492409183741797","blockNumber":25459279}`
)

func newROMEUSDTSim(t *testing.T) *base.PoolSimulator {
	t.Helper()
	var entityPool entity.Pool
	require.NoError(t, json.Unmarshal([]byte(romeUSDTSnapshot), &entityPool))
	sim, err := NewPoolSimulator(pool.FactoryParams{
		EntityPool: entityPool,
		ChainID:    valueobject.ChainIDEthereum,
	})
	require.NoError(t, err)
	return sim
}

// TestCentralFactoryRegistration proves the Range connector is wired into the central
// pool factory: constructing by DexType (the path the pool service uses) yields a working
// Range simulator, and the type is advertised as exact-out-capable.
func TestCentralFactoryRegistration(t *testing.T) {
	t.Parallel()

	factory := pool.Factory(DexType)
	require.NotNil(t, factory, "range-pool must be registered in the central pool factory")

	var entityPool entity.Pool
	require.NoError(t, json.Unmarshal([]byte(romeUSDTSnapshot), &entityPool))
	sim, err := factory(pool.FactoryParams{EntityPool: entityPool, ChainID: valueobject.ChainIDEthereum})
	require.NoError(t, err)
	require.NotNil(t, sim)

	// EXACT_OUT support is registered too (base.PoolSimulator implements IPoolExactOutSimulator).
	_, canCalcAmountIn := pool.CanCalcAmountIn[DexType]
	assert.True(t, canCalcAmountIn, "range-pool must advertise exact-out support")
}

func TestCalcAmountOut(t *testing.T) {
	t.Parallel()

	// querySwap-verified golden values at block 25459279.
	tests := []struct {
		name              string
		tokenIn, tokenOut string
		amountIn          int64
		expectedOut       string
		expectedFee       string
	}{
		{"ROME->USDT", romeToken, usdtToken, 100_000_000, "96694132", "350000"},
		{"USDT->ROME", usdtToken, romeToken, 50_000_000, "51334904", "175000"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			sim := newROMEUSDTSim(t)
			res, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return sim.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: tc.tokenIn, Amount: big.NewInt(tc.amountIn)},
					TokenOut:      tc.tokenOut,
				})
			})
			require.NoError(t, err)
			assert.Equal(t, tc.expectedOut, res.TokenAmountOut.Amount.String())
			assert.Equal(t, tc.expectedFee, res.Fee.Amount.String())
		})
	}
}

func TestCalcAmountIn(t *testing.T) {
	t.Parallel()

	// querySwap-verified golden values at block 25459279.
	tests := []struct {
		name              string
		tokenIn, tokenOut string
		amountOut         int64
		expectedIn        string
		expectedFee       string
	}{
		{"ROME->USDT (want USDT out)", romeToken, usdtToken, 50_000_000, "51705207", "180968"},
		{"USDT->ROME (want ROME out)", usdtToken, romeToken, 100_000_000, "97407694", "340926"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			sim := newROMEUSDTSim(t)
			res, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountInResult, error) {
				return sim.CalcAmountIn(pool.CalcAmountInParams{
					TokenAmountOut: pool.TokenAmount{Token: tc.tokenOut, Amount: big.NewInt(tc.amountOut)},
					TokenIn:        tc.tokenIn,
				})
			})
			require.NoError(t, err)
			assert.Equal(t, tc.expectedIn, res.TokenAmountIn.Amount.String())
			assert.Equal(t, tc.expectedFee, res.Fee.Amount.String())
		})
	}
}

// TestUpdateBalance asserts UpdateBalance consumes the SwapInfo from CalcAmountOut and
// moves reserves the right way: the in-token reserve grows by (amountIn - aggregateFee)
// and the out-token reserve shrinks by amountOut.
func TestUpdateBalance(t *testing.T) {
	t.Parallel()
	sim := newROMEUSDTSim(t)

	before := sim.GetReserves()
	reserveInBefore := new(big.Int).Set(before[0])
	reserveOutBefore := new(big.Int).Set(before[1])

	amountIn := big.NewInt(100_000_000)
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: romeToken, Amount: amountIn},
		TokenOut:      usdtToken,
	})
	require.NoError(t, err)

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: romeToken, Amount: amountIn},
		TokenAmountOut: *res.TokenAmountOut,
		SwapInfo:       res.SwapInfo,
	})

	after := sim.GetReserves()

	// in-token reserve up by amountIn minus the aggregate (protocol+creator) fee.
	aggFee := res.SwapInfo.(shared.SwapInfo).AggregateFee
	wantIn := new(big.Int).Add(reserveInBefore, amountIn)
	wantIn.Sub(wantIn, aggFee)
	assert.Equal(t, wantIn.String(), after[0].String(), "in-token reserve after swap")

	// out-token reserve down by amountOut.
	wantOut := new(big.Int).Sub(reserveOutBefore, res.TokenAmountOut.Amount)
	assert.Equal(t, wantOut.String(), after[1].String(), "out-token reserve after swap")
}

// TestCloneState checks the clone is isolated: mutating the clone via UpdateBalance must
// not touch the original's reserves, and both must quote identically beforehand.
func TestCloneState(t *testing.T) {
	t.Parallel()
	orig := newROMEUSDTSim(t)
	clone := orig.CloneState()

	origBefore := orig.GetReserves()

	amountIn := big.NewInt(100_000_000)
	res, err := clone.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: romeToken, Amount: amountIn},
		TokenOut:      usdtToken,
	})
	require.NoError(t, err)

	clone.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: romeToken, Amount: amountIn},
		TokenAmountOut: *res.TokenAmountOut,
		SwapInfo:       res.SwapInfo,
	})

	// Original reserves are untouched by the clone's swap.
	origAfter := orig.GetReserves()
	assert.Equal(t, origBefore[0].String(), origAfter[0].String(), "original in-token reserve unchanged")
	assert.Equal(t, origBefore[1].String(), origAfter[1].String(), "original out-token reserve unchanged")

	// The clone did move.
	cloneAfter := clone.GetReserves()
	assert.NotEqual(t, origBefore[1].String(), cloneAfter[1].String(), "clone out-token reserve moved")
}

// TestPausedPoolRejectsSwap: a liveness-gated pool is disabled by the tracker zeroing its
// reserves; base.PoolSimulator then marks it paused and CalcAmountOut must error.
func TestPausedPoolRejectsSwap(t *testing.T) {
	t.Parallel()

	// Simulate the tracker's liveness gate (zeroed reserves) on the same snapshot.
	pausedSnapshot := strings.Replace(romeUSDTSnapshot,
		`"reserves":["11722718035","165904197"]`,
		`"reserves":["0","0"]`, 1)

	var entityPool entity.Pool
	require.NoError(t, json.Unmarshal([]byte(pausedSnapshot), &entityPool))
	sim, err := NewPoolSimulator(pool.FactoryParams{
		EntityPool: entityPool,
		ChainID:    valueobject.ChainIDEthereum,
	})
	require.NoError(t, err)

	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: romeToken, Amount: big.NewInt(100_000_000)},
		TokenOut:      usdtToken,
	})
	assert.ErrorIs(t, err, shared.ErrPoolIsPaused)
}

// TestZeroAmountRejected: a zero-amount swap must never produce a (positive) quote — the
// simulator returns an error on both sides rather than a zero/near-zero price. (The full
// sub-MINIMUM_TRADE_AMOUNT dust-revert parity against the chain is covered by the RPC gate
// in range-pool-parity_integration_test.go, which needs an 18-decimal token to reach the
// regime; both tokens here are 6-decimal.)
func TestZeroAmountRejected(t *testing.T) {
	t.Parallel()
	sim := newROMEUSDTSim(t)

	_, errOut := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: romeToken, Amount: big.NewInt(0)},
		TokenOut:      usdtToken,
	})
	assert.Error(t, errOut, "EXACT_IN with zero input must error")

	_, errIn := sim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: usdtToken, Amount: big.NewInt(0)},
		TokenIn:        romeToken,
	})
	assert.Error(t, errIn, "EXACT_OUT with zero output must error")
}
