package hiddenocean

import (
	"context"
	"math/big"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
)

const (
	rpcURL           = "https://rpc.hyperliquid.xyz/evm"
	multicallAddress = "0xcA11bde05977b3631167028862bE2a173976CA11"
)

type PoolTrackerTestSuite struct {
	suite.Suite
	tracker *PoolTracker
	updater *PoolsListUpdater
	config  *Config
}

func (ts *PoolTrackerTestSuite) SetupSuite() {
	ts.config = &Config{
		DexID:           DexType,
		RegistryAddress: "0x17c29d91852051073EFB6f1A8E1074Fe43512961",
		NewPoolLimit:    5,
	}

	client := ethrpc.New(rpcURL).
		SetMulticallContract(common.HexToAddress(multicallAddress))

	ts.tracker = NewPoolTracker(ts.config, client)
	ts.updater = NewPoolsListUpdater(ts.config, client)
}

func (ts *PoolTrackerTestSuite) TestGetNewPools() {
	t := ts.T()

	pools, newMetadata, err := ts.updater.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.NotEmpty(t, pools, "should discover at least one pool from registry")

	t.Logf("discovered %d pools", len(pools))
	for i, p := range pools {
		t.Logf("  pool[%d]: address=%s tokens=[%s, %s]",
			i, p.Address, p.Tokens[0].Address, p.Tokens[1].Address)
	}

	// Metadata should contain updated offset
	var meta Metadata
	require.NoError(t, json.Unmarshal(newMetadata, &meta))
	assert.Greater(t, meta.Offset, 0)
}

func (ts *PoolTrackerTestSuite) TestGetNewPoolState() {
	t := ts.T()

	// First discover pools from registry
	pools, _, err := ts.updater.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.NotEmpty(t, pools, "need at least one pool to test tracker")

	// Use the first discovered pool for tracking
	testPool := pools[0]
	t.Logf("tracking pool: %s", testPool.Address)

	// Fetch live state
	updatedPool, err := ts.tracker.GetNewPoolState(
		context.Background(),
		testPool,
		pool.GetNewPoolStateParams{},
	)
	require.NoError(t, err)

	// Verify reserves are populated
	require.Len(t, updatedPool.Reserves, 2)
	t.Logf("reserves: [%s, %s]", updatedPool.Reserves[0], updatedPool.Reserves[1])

	// Verify block number is set
	assert.Greater(t, updatedPool.BlockNumber, uint64(0), "block number should be set")
	t.Logf("blockNumber: %d", updatedPool.BlockNumber)

	// Verify timestamp is recent
	assert.Greater(t, updatedPool.Timestamp, int64(0), "timestamp should be set")

	// Verify Extra is populated and parseable
	require.NotEmpty(t, updatedPool.Extra)
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(updatedPool.Extra), &extra))

	assert.NotNil(t, extra.SqrtPriceX96, "sqrtPriceX96 should be set")
	assert.False(t, extra.SqrtPriceX96.IsZero(), "sqrtPriceX96 should be non-zero")

	assert.NotNil(t, extra.Liquidity, "liquidity should be set")

	assert.NotNil(t, extra.SqrtPaX96, "sqrtPaX96 should be set")
	assert.NotNil(t, extra.SqrtPbX96, "sqrtPbX96 should be set")

	t.Logf("sqrtPriceX96=%s liquidity=%s fee=%d sqrtPa=%s sqrtPb=%s",
		extra.SqrtPriceX96.Dec(), extra.Liquidity.Dec(), extra.Fee,
		extra.SqrtPaX96.Dec(), extra.SqrtPbX96.Dec())

	// Verify range: sqrtPa < sqrtPb (when liquidity is non-zero)
	if !extra.Liquidity.IsZero() {
		assert.True(t, extra.SqrtPaX96.Cmp(extra.SqrtPbX96) < 0,
			"sqrtPa should be less than sqrtPb")
	}
}

func (ts *PoolTrackerTestSuite) TestGetNewPoolStateAndSimulate() {
	t := ts.T()

	// Discover + track
	pools, _, err := ts.updater.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.NotEmpty(t, pools)

	updatedPool, err := ts.tracker.GetNewPoolState(
		context.Background(),
		pools[0],
		pool.GetNewPoolStateParams{},
	)
	require.NoError(t, err)

	// Build a simulator from the live state
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: updatedPool})
	if err != nil {
		t.Skipf("pool not viable for simulation: %v", err)
	}

	require.NotNil(t, sim)
	t.Logf("simulator built: address=%s tokens=%v", sim.Info.Address, sim.Info.Tokens)

	// If liquidity is non-zero, try a swap with a meaningful amount (1e15 wei = 0.001 token)
	if sim.liquidity != nil && !sim.liquidity.IsZero() {
		tokenIn := sim.Info.Tokens[0]
		tokenOut := sim.Info.Tokens[1]

		amountIn := new(big.Int).Exp(big.NewInt(10), big.NewInt(15), nil) // 1e15

		result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  tokenIn,
				Amount: amountIn,
			},
			TokenOut: tokenOut,
		})

		if err != nil {
			t.Logf("swap failed (may be at boundary): %v", err)
		} else {
			assert.True(t, result.TokenAmountOut.Amount.Sign() > 0,
				"amountOut should be positive for a live pool swap")
			t.Logf("swap: in=%s %s → out=%s %s, fee=%s, gas=%d",
				amountIn.String(), tokenIn, result.TokenAmountOut.Amount.String(), tokenOut,
				result.Fee.Amount.String(), result.Gas)
		}
	}
}

func TestPoolTrackerTestSuite(t *testing.T) {
	t.Parallel()
	test.SkipCI(t)
	suite.Run(t, new(PoolTrackerTestSuite))
}
