package efraprop

import (
	"math/big"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
)

const (
	testNVDA  = "0x1111111111111111111111111111111111111111"
	testUSDC  = "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"
	testVenue = "0x2222222222222222222222222222222222222222"
)

// testLadders quote a stock token worth 180 USDC: 8 decimals against USDC's 6, so 1e8 raw
// tokens is 180e6 raw USDC at the probed mid, and the second point is 1% worse.
func testLadders() [2][]ladder.Point {
	return [2][]ladder.Point{
		{{1e8, 180e6}, {2e8, 356.4e6}}, // token in -> USDC out
		{{180e6, 1e8}, {356.4e6, 2e8}}, // USDC in -> token out
	}
}

func testPool(t *testing.T, ladders [2][]ladder.Point, ts int64) entity.Pool {
	t.Helper()
	return entity.Pool{
		Address:  syntheticPoolAddress(DexType, testNVDA, testUSDC),
		Exchange: DexType,
		Type:     DexType,
		Tokens: []*entity.PoolToken{
			{Address: testNVDA, Decimals: 8, Swappable: true},
			{Address: testUSDC, Decimals: 6, Swappable: true},
		},
		Reserves:    entity.PoolReserves{"90000000000", "2000000000"},
		Extra:       mustJSON(t, ladder.Extra{Ladders: ladders}),
		StaticExtra: mustJSON(t, StaticExtra{Venue: testVenue, Taker: defaultDest.Hex()}),
		Timestamp:   ts,
	}
}

func mustSim(t *testing.T, ladders [2][]ladder.Point, ts int64) *PoolSimulator {
	t.Helper()
	sim, err := NewPoolSimulator(pool.FactoryParams{
		EntityPool: testPool(t, ladders, ts),
		Opts:       pool.FactoryOpts{StaleCheck: true},
	})
	require.NoError(t, err)
	return sim
}

func quote(t *testing.T, sim pool.IPoolSimulator, tokenIn, tokenOut string, amountIn int64) (*pool.CalcAmountOutResult, error) {
	t.Helper()
	return sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: big.NewInt(amountIn)},
		TokenOut:      tokenOut,
		Limit:         swaplimit.NewInventory(DexType, sim.CalculateLimit()),
	})
}

func TestCalcAmountOut_BothDirections(t *testing.T) {
	sim := mustSim(t, testLadders(), time.Now().Unix())

	res, err := quote(t, sim, testNVDA, testUSDC, 1e8)
	require.NoError(t, err)
	assert.Equal(t, "180000000", res.TokenAmountOut.Amount.String(), "sell 1 token at the probed mid")

	res, err = quote(t, sim, testUSDC, testNVDA, 180e6)
	require.NoError(t, err)
	assert.Equal(t, "100000000", res.TokenAmountOut.Amount.String(), "buy 1 token at the probed mid")
}

// Past the last probed point the venue's bins have run out (quoteExactIn answers 0 there),
// so the pool must refuse the size instead of extrapolating the spline.
func TestCalcAmountOut_PastLastPoint(t *testing.T) {
	sim := mustSim(t, testLadders(), time.Now().Unix())

	_, err := quote(t, sim, testNVDA, testUSDC, 3e8)
	assert.ErrorIs(t, err, ladder.ErrAmountInTooLarge)
}

// The venue's quotes expire 15 blocks after the maker's last update, so route finding must
// reject a ladder older than maxAge. The liquidity indexer runs with StaleCheck off and has
// to keep pricing, or pool state stops being recorded.
func TestNewPoolSimulator_Stale(t *testing.T) {
	old := time.Now().Add(-maxAge - time.Minute).Unix()

	_, err := NewPoolSimulator(pool.FactoryParams{
		EntityPool: testPool(t, testLadders(), old),
		Opts:       pool.FactoryOpts{StaleCheck: true},
	})
	assert.ErrorIs(t, err, ladder.ErrStale)

	indexer, err := NewPoolSimulator(pool.FactoryParams{EntityPool: testPool(t, testLadders(), old)})
	require.NoError(t, err)
	_, err = quote(t, indexer, testNVDA, testUSDC, 1e8)
	require.NoError(t, err)
}

// One venue holds every market, so the encoder must approve the venue it was probed on, and
// it reads the refresh block from the same struct.
func TestGetMetaInfo(t *testing.T) {
	sim := mustSim(t, testLadders(), time.Now().Unix())
	sim.Info.BlockNumber = 123

	meta, ok := sim.GetMetaInfo("", "").(PoolMeta)
	require.True(t, ok)
	assert.Equal(t, testVenue, meta.ApprovalAddress)
	assert.Equal(t, testVenue, meta.Venue)
	assert.Equal(t, uint64(123), meta.BlockNumber)
}

// A fill moves the side's cursor, so a clone keeps its own consumption and a second pass
// through the same pool prices the worse half of the curve, not the first-fill price.
func TestCloneState_IsolatesConsumption(t *testing.T) {
	sim := mustSim(t, testLadders(), time.Now().Unix())

	res, err := quote(t, sim, testNVDA, testUSDC, 1e8)
	require.NoError(t, err)
	cloned := sim.CloneState()
	cloned.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: testNVDA, Amount: big.NewInt(1e8)},
		TokenAmountOut: *res.TokenAmountOut,
	})

	_, err = quote(t, sim, testNVDA, testUSDC, 1e8)
	require.NoError(t, err, "the original pool still quotes a full first fill")

	next, err := quote(t, cloned, testNVDA, testUSDC, 1e8)
	require.NoError(t, err)
	assert.Equal(t, "176400000", next.TokenAmountOut.Amount.String(),
		"a second fill in the same block lands past the first")
}

func TestCalcAmountOut_UnknownToken(t *testing.T) {
	sim := mustSim(t, testLadders(), time.Now().Unix())
	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testUSDC, Amount: big.NewInt(1e6)},
		TokenOut:      "0x3333333333333333333333333333333333333333",
	})
	assert.ErrorIs(t, err, ladder.ErrInvalidToken)
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}
