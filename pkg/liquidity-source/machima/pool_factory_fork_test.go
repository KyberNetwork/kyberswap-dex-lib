package machima

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const machimaFactory = "0xADd30837a707cCE4567eEa2C27d0617270d54C75"

// TestEventCreatedPoolIsUsable covers the path bootstrap never exercises: a pool discovered from a
// PoolCreated log. pool-service saves whatever DecodePoolCreated returns verbatim — no ticks, no
// tax, zero reserves — and only later calls GetNewPoolState with the pool's own logs. So the check
// that matters is whether that second step turns the bare record into something quotable.
func TestEventCreatedPoolIsUsable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping fork test in short mode")
	}
	rpcURL := os.Getenv("BASE_RPC_URL")
	if rpcURL == "" {
		t.Skip("BASE_RPC_URL not set")
	}

	ctx := context.Background()
	client, err := ethclient.Dial(rpcURL)
	require.NoError(t, err)
	defer client.Close()

	// Real Machima pool creation on Base: pool 0xd4829d18…a4a8, created at block 48500863.
	const createdAtBlock = 48500863
	poolAddress := common.HexToAddress("0xd4829d181e93059ae602ce5a5b59ff4d6736a4a8")

	factoryLogs, err := client.FilterLogs(ctx, ethereum.FilterQuery{
		Addresses: []common.Address{common.HexToAddress(machimaFactory)},
		FromBlock: big.NewInt(createdAtBlock),
		ToBlock:   big.NewInt(createdAtBlock),
	})
	require.NoError(t, err)

	cfg := &Config{
		DexID:           DexType,
		ClankNow:        clankNowAddr,
		SwapAdapter:     swapAdapterAddr,
		TickLensAddress: tickLensAddr,
		RouterAddress:   aggregatorRouter,
		WETH:            wethAddr,
		USDC:            usdcAddr,
		XMA:             xmaAddr,
	}

	// Step 1: decode the PoolCreated log, exactly as EventDecoder.GetNewPoolsFromLogs does.
	factory := NewPoolFactory(cfg)
	var created *entity.Pool
	for _, l := range factoryLogs {
		if len(l.Topics) == 0 || !factory.IsEventSupported(l.Topics[0]) {
			continue
		}
		p, decErr := factory.DecodePoolCreated(l)
		require.NoError(t, decErr)
		if common.HexToAddress(p.Address) == poolAddress {
			created = p
			break
		}
	}
	require.NotNil(t, created, "PoolCreated log for the target pool must decode")

	// This is precisely what pool-service persists at creation time.
	var freshExtra Extra
	require.NoError(t, json.Unmarshal([]byte(created.Extra), &freshExtra))
	assert.Empty(t, freshExtra.Ticks, "a freshly created pool has no ticks yet")
	assert.False(t, freshExtra.HasTax, "and no tax yet")
	assert.Equal(t, defaultTickSpacing, freshExtra.TickSpacing, "but tickSpacing must be seeded")
	t.Logf("decoded new pool %s tokens=%s/%s", created.Address,
		created.Tokens[0].Address[:10], created.Tokens[1].Address[:10])

	// Step 2: the pool's own logs from that block (the Mint that seeded liquidity).
	poolLogs, err := client.FilterLogs(ctx, ethereum.FilterQuery{
		Addresses: []common.Address{poolAddress},
		FromBlock: big.NewInt(createdAtBlock),
		ToBlock:   big.NewInt(createdAtBlock),
	})
	require.NoError(t, err)
	require.NotEmpty(t, poolLogs, "expected the seeding Mint in the creation block")
	t.Logf("replaying %d pool logs from block %d", len(poolLogs), createdAtBlock)

	tracker, err := NewPoolTracker(cfg,
		ethrpc.New(rpcURL).SetMulticallContract(common.HexToAddress(multicall3)), nil)
	require.NoError(t, err)

	updated, err := tracker.GetNewPoolState(ctx, *created, poolpkg.GetNewPoolStateParams{
		Logs: toEthLogs(poolLogs),
	})
	require.NoError(t, err, "event-driven refresh of a brand new pool must succeed")

	// Step 3: the refreshed pool must be complete enough to route.
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(updated.Extra), &extra))
	assert.NotEmpty(t, extra.Ticks, "ticks must be derived from the Mint log")
	assert.True(t, extra.HasTax, "tax must be layered on")
	assert.NotZero(t, extra.PoolDeploymentTime)
	require.NotNil(t, extra.SqrtPriceX96)
	assert.Positive(t, extra.SqrtPriceX96.Sign())
	t.Logf("after event refresh: ticks=%d tick=%v hasTax=%v buy=%d sell=%d reserves=%v",
		len(extra.Ticks), extra.Tick, extra.HasTax, extra.BuyTaxBps, extra.SellTaxBps, updated.Reserves)

	// Liquidity net across all ticks must cancel, otherwise the tick set is corrupt.
	var netSum big.Int
	for _, tick := range extra.Ticks {
		netSum.Add(&netSum, tick.LiquidityNet)
	}
	assert.Zero(t, netSum.Sign(), "sum of liquidityNet across ticks must be zero")

	sim, err := NewPoolSimulator(updated, valueobject.ChainIDBase)
	require.NoError(t, err, "an event-created pool must build a simulator")

	var staticExtra StaticExtra
	require.NoError(t, json.Unmarshal([]byte(updated.StaticExtra), &staticExtra))
	counter, token := updated.Tokens[0].Address, updated.Tokens[1].Address
	if staticExtra.Token == counter {
		counter, token = token, counter
	}

	res, err := sim.CalcAmountOut(poolpkg.CalcAmountOutParams{
		TokenAmountIn: poolpkg.TokenAmount{Token: counter, Amount: big.NewInt(1e14)},
		TokenOut:      token,
	})
	require.NoError(t, err, "event-created pool must quote")
	assert.Positive(t, res.TokenAmountOut.Amount.Sign())

	// And that quote must agree with the on-chain quoter.
	want, err := quote(rpcURL, counter, token, big.NewInt(1e14))
	require.NoError(t, err)
	diff := new(big.Int).Sub(res.TokenAmountOut.Amount, want)
	diff.Abs(diff)
	tol := new(big.Int).Div(want, big.NewInt(10000))
	assert.LessOrEqual(t, diff.Cmp(tol), 0,
		"event-created pool quote sim=%s quoter=%s diff=%s", res.TokenAmountOut.Amount, want, diff)
	t.Logf("quote parity: sim=%s quoter=%s diff=%s", res.TokenAmountOut.Amount, want, diff)
}

func toEthLogs(logs []ethtypes.Log) []ethtypes.Log { return logs }

// TestEventCreatedPoolBeforeItsMint covers the ordering the interval trigger can win: a pool is
// decoded from PoolCreated and saved with no ticks, and the 10s interval fires before the seeding
// Mint is processed. That refresh must not error (it would mark the pool failed and spam logs) and
// must leave the pool merely un-routable, not corrupt.
func TestEventCreatedPoolBeforeItsMint(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping fork test in short mode")
	}
	rpcURL := os.Getenv("BASE_RPC_URL")
	if rpcURL == "" {
		t.Skip("BASE_RPC_URL not set")
	}

	ctx := context.Background()
	cfg := &Config{
		DexID: DexType, ClankNow: clankNowAddr, SwapAdapter: swapAdapterAddr,
		TickLensAddress: tickLensAddr, RouterAddress: aggregatorRouter,
		WETH: wethAddr, USDC: usdcAddr, XMA: xmaAddr,
	}

	created, err := NewPoolFactory(cfg).DecodePoolCreated(poolCreatedLog())
	require.NoError(t, err)

	tracker, err := NewPoolTracker(cfg,
		ethrpc.New(rpcURL).SetMulticallContract(common.HexToAddress(multicall3)), nil)
	require.NoError(t, err)

	// Interval trigger: no logs at all.
	updated, err := tracker.GetNewPoolState(ctx, *created, poolpkg.GetNewPoolStateParams{})
	require.NoError(t, err, "interval refresh of a tick-less pool must not error")

	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(updated.Extra), &extra))
	assert.True(t, extra.HasTax, "tax is still fetched even with no ticks")
	assert.NotZero(t, extra.PoolDeploymentTime)
	require.NotNil(t, extra.SqrtPriceX96, "slot0 is still refreshed")
	assert.NotEmpty(t, updated.Reserves)

	// Un-routable rather than corrupt: the simulator refuses a pool with no ticks, so routing
	// skips it until the Mint arrives.
	_, err = NewPoolSimulator(updated, valueobject.ChainIDBase)
	assert.Error(t, err, "a tick-less pool must be rejected, not silently quoted")
	t.Logf("tick-less pool correctly rejected by simulator: %v", err)
}
