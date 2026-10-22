package machima

import (
	"context"
	"math/big"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const tickLensWordABI = `[{"inputs":[{"name":"pool","type":"address"},{"name":"tickBitmapIndex","type":"int16"}],
"name":"getPopulatedTicksInWord","outputs":[{"components":[{"name":"tick","type":"int24"},
{"name":"liquidityNet","type":"int128"},{"name":"liquidityGross","type":"uint128"}],
"name":"populatedTicks","type":"tuple[]"}],"stateMutability":"view","type":"function"}]`

// TestLensLive: on live Base pools, the lens bootstrap gives the ticks TickLens reads at the same
// block and keeps the Machima half; a pool without state, as a PoolCreated log leaves it, gets
// both from an update, with its own logs or from the interval refresh.
//
// Requires BASE_RPC_URL (an archive node: one pool's logs are read at its creation block).
func TestLensLive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping fork test in short mode")
	}
	rpcURL := os.Getenv("BASE_RPC_URL")
	if rpcURL == "" {
		t.Skip("BASE_RPC_URL not set")
	}

	ctx := t.Context()
	client, err := ethclient.Dial(rpcURL)
	require.NoError(t, err)
	defer client.Close()

	newTracker := func(useLens bool) *PoolTracker {
		tracker, err := NewPoolTracker(&Config{
			DexID:           DexType,
			ClankNow:        clankNowAddr,
			SwapAdapter:     swapAdapterAddr,
			TickLensAddress: tickLensAddr,
			RouterAddress:   aggregatorRouter,
			WETH:            wethAddr,
			USDC:            usdcAddr,
			XMA:             xmaAddr,
			UseLens:         useLens,
		}, ethrpc.New(rpcURL).SetMulticallContract(common.HexToAddress(multicall3)), nil)
		require.NoError(t, err)
		return tracker
	}
	lensTracker, tickLensTracker := newTracker(true), newTracker(false)

	// Pool 0xd4829d18…a4a8 as its PoolCreated log leaves it, with the logs of its creation block.
	const createdAtBlock = 48500863
	created, creationLogs := createdPool(t, ctx, client, createdAtBlock,
		common.HexToAddress("0xd4829d181e93059ae602ce5a5b59ff4d6736a4a8"))
	require.False(t, ticks.HasState(*created), "a pool from its PoolCreated log has no state")

	xmaStatic := `{"token":"` + xmaAddr + `","routerAddress":"` + aggregatorRouter + `","weth":"` + wethAddr +
		`","usdc":"` + usdcAddr + `","xma":"` + xmaAddr + `"}`
	pools := []entity.Pool{
		{
			Address: "0x531aae7d71343c663821604c57520b1602567006", SwapFee: defaultFee, Exchange: DexType, Type: DexType,
			Tokens:   []*entity.PoolToken{{Address: wethAddr, Swappable: true}, {Address: xmaAddr, Swappable: true}},
			Reserves: entity.PoolReserves{"0", "0"}, Extra: `{"tickSpacing":200}`, StaticExtra: xmaStatic,
		},
		*created,
	}

	for _, seed := range pools {
		t.Run(seed.Address, func(t *testing.T) {
			lensPool := retry(t, func() (entity.Pool, error) {
				return lensTracker.BootstrapPoolState(ctx, seed, pool.GetNewPoolStateParams{})
			})
			lensExtra := mustExtra(t, lensPool)
			require.NotZero(t, lensExtra.LensBlock)
			assert.Equal(t, lensExtra.LensBlock, lensPool.BlockNumber, "the pool's block is the one the lens read at")
			assert.NotZero(t, lensExtra.PoolDeploymentTime, "the Machima half is read after the lens")

			want := tickLensAt(t, ctx, client, seed.Address, int(lensExtra.TickSpacing), lensPool.BlockNumber)
			assert.Equal(t, want, tickSet(lensExtra.Ticks), "lens and TickLens at block %d", lensPool.BlockNumber)

			oldPool := retry(t, func() (entity.Pool, error) {
				return tickLensTracker.BootstrapPoolState(ctx, seed, pool.GetNewPoolStateParams{})
			})
			oldExtra := mustExtra(t, oldPool)
			assert.Equal(t, oldExtra.ProtocolState, lensExtra.ProtocolState)
			assert.Equal(t, tickSet(oldExtra.Ticks), tickSet(lensExtra.Ticks), "TickLens path at latest")
			t.Logf("%s: lens block %d, %d ticks, tick %v, tax buy %d sell %d hasTax %v; TickLens path %d ticks",
				seed.Address, lensPool.BlockNumber, len(lensExtra.Ticks), lensExtra.Tick, lensExtra.BuyTaxBps,
				lensExtra.SellTaxBps, lensExtra.HasTax, len(oldExtra.Ticks))
		})
	}

	t.Run("without state, with its own logs", func(t *testing.T) {
		got := retry(t, func() (entity.Pool, error) {
			return lensTracker.GetNewPoolState(ctx, *created, pool.GetNewPoolStateParams{Logs: creationLogs})
		})
		extra := mustExtra(t, got)
		require.True(t, ticks.HasState(got))
		assert.NotZero(t, extra.PoolDeploymentTime)
		assert.GreaterOrEqual(t, got.BlockNumber, extra.LensBlock, "the late logs must not lower the pool's block")
		assert.Equal(t, tickLensAt(t, ctx, client, created.Address, int(extra.TickSpacing), got.BlockNumber),
			tickSet(extra.Ticks))
		t.Logf("%d creation logs at %d: lens block %d, pool block %d, %d ticks", len(creationLogs),
			createdAtBlock, extra.LensBlock, got.BlockNumber, len(extra.Ticks))
	})

	t.Run("without state, on the interval", func(t *testing.T) {
		// A tracker of its own, and a new one per attempt: the guard has just read this pool for the
		// subtest above, and a read retried after a 429 is not a second read.
		var fresh *PoolTracker
		got := retry(t, func() (entity.Pool, error) {
			fresh = newTracker(true)
			return fresh.GetNewPoolState(ctx, *created, pool.GetNewPoolStateParams{})
		})
		extra := mustExtra(t, got)
		require.True(t, ticks.HasState(got))
		assert.NotEmpty(t, extra.Ticks, "the interval reads a pool without state whole")
		assert.NotZero(t, extra.PoolDeploymentTime)

		// Read again at once, the guard declines: the pool stays without state, and the block its
		// read must reach survives the Machima rewrite of Extra.
		declined := retry(t, func() (entity.Pool, error) {
			return fresh.GetNewPoolState(ctx, *created, pool.GetNewPoolStateParams{Logs: creationLogs})
		})
		assert.False(t, ticks.HasState(declined))
		assert.Equal(t, uint64(createdAtBlock), ticks.LensDue(declined))
		assert.NotZero(t, mustExtra(t, declined).PoolDeploymentTime)
		assert.Positive(t, fresh.NextUpdateAt(declined), "it is scheduled for when the guard takes it")

		// Without the lens the interval gives it a liquidity and no ticks, after which it reads as
		// a pool with state.
		old := retry(t, func() (entity.Pool, error) {
			return tickLensTracker.GetNewPoolState(ctx, *created, pool.GetNewPoolStateParams{})
		})
		t.Logf("lens: %d ticks; without the lens: hasState %v with %d ticks", len(extra.Ticks),
			ticks.HasState(old), len(mustExtra(t, old).Ticks))
	})
}

func createdPool(t *testing.T, ctx context.Context, client *ethclient.Client, block int64,
	address common.Address) (*entity.Pool, []ethtypes.Log) {
	t.Helper()
	factoryLogs := retry(t, func() ([]ethtypes.Log, error) {
		return client.FilterLogs(ctx, ethereum.FilterQuery{
			Addresses: []common.Address{common.HexToAddress(machimaFactory)},
			FromBlock: big.NewInt(block), ToBlock: big.NewInt(block),
		})
	})
	factory := NewPoolFactory(&Config{DexID: DexType, RouterAddress: aggregatorRouter, WETH: wethAddr,
		USDC: usdcAddr, XMA: xmaAddr})
	var created *entity.Pool
	for _, l := range factoryLogs {
		if len(l.Topics) == 0 || !factory.IsEventSupported(l.Topics[0]) {
			continue
		}
		p, err := factory.DecodePoolCreated(l)
		require.NoError(t, err)
		if common.HexToAddress(p.Address) == address {
			created = p
		}
	}
	require.NotNil(t, created)

	poolLogs := retry(t, func() ([]ethtypes.Log, error) {
		return client.FilterLogs(ctx, ethereum.FilterQuery{
			Addresses: []common.Address{address}, FromBlock: big.NewInt(block), ToBlock: big.NewInt(block),
		})
	})
	require.NotEmpty(t, poolLogs)
	return created, poolLogs
}

func tickLensAt(t *testing.T, ctx context.Context, client *ethclient.Client, poolAddress string, spacing int,
	block uint64) []string {
	t.Helper()
	parsed, err := abi.JSON(strings.NewReader(tickLensWordABI))
	require.NoError(t, err)
	lens := common.HexToAddress(tickLensAddr)

	var set []string
	lower, upper := floorDiv(-887272, spacing)>>8, (887272/spacing)>>8
	for word := lower; word <= upper; word++ {
		data, err := parsed.Pack("getPopulatedTicksInWord", common.HexToAddress(poolAddress), int16(word))
		require.NoError(t, err)
		out := retry(t, func() ([]byte, error) {
			return client.CallContract(ctx, ethereum.CallMsg{To: &lens, Data: data}, new(big.Int).SetUint64(block))
		})
		var res []struct {
			Tick           *big.Int
			LiquidityNet   *big.Int
			LiquidityGross *big.Int
		}
		require.NoError(t, parsed.UnpackIntoInterface(&res, "getPopulatedTicksInWord", out))
		for _, tk := range res {
			set = append(set, tk.Tick.String()+"/"+tk.LiquidityNet.String()+"/"+tk.LiquidityGross.String())
		}
	}
	slices.SortFunc(set, compareTick)
	return set
}

func tickSet(ts []uniswapv3.Tick) []string {
	set := make([]string, 0, len(ts))
	for _, tk := range ts {
		set = append(set, big.NewInt(int64(tk.Index)).String()+"/"+tk.LiquidityNet.String()+"/"+tk.LiquidityGross.String())
	}
	slices.SortFunc(set, compareTick)
	return set
}

func compareTick(a, b string) int {
	ai, _ := new(big.Int).SetString(strings.Split(a, "/")[0], 10)
	bi, _ := new(big.Int).SetString(strings.Split(b, "/")[0], 10)
	return ai.Cmp(bi)
}

func floorDiv(a, b int) int {
	q := a / b
	if a%b != 0 && a < 0 {
		q--
	}
	return q
}

func mustExtra(t *testing.T, p entity.Pool) Extra {
	t.Helper()
	extra, err := unmarshalExtra(p.Extra)
	require.NoError(t, err)
	return extra
}

// retry runs a step again while the node answers 429: a shared test key is rate limited.
func retry[T any](t *testing.T, step func() (T, error)) T {
	t.Helper()
	var (
		v   T
		err error
	)
	for attempt := range 8 {
		if v, err = step(); err == nil || !strings.Contains(err.Error(), "429") {
			break
		}
		time.Sleep(time.Duration(attempt+1) * 2 * time.Second)
	}
	require.NoError(t, err)
	return v
}
