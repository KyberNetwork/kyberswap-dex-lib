package integral

import (
	"bytes"
	"cmp"
	"context"
	"io"
	"math/big"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
)

// liveDex is one Algebra Integral deployment: the RPC URL comes from env, the pools from its
// factory's poolByPair over tokens common on its chain, plus pools known to be live.
type liveDex struct {
	dexID   string
	rpcEnv  string
	factory string
	tokens  []string
	known   []string
	// multicall is the chain's own when Multicall3 will not do: on Arbitrum its block.number is the
	// L1 block, at which the plugin reads that follow a state read would find nothing.
	multicall string
}

var liveDexes = []liveDex{
	{"supernova-v4", "ETHEREUM_RPC_URL", "0x44B7fBd4D87149eFa5347c451E74B9FD18E89c55", []string{
		"0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2", "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
		"0xdAC17F958D2ee523a2206206994597C13D831ec7", "0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599",
		"0x6B175474E89094C44Da98b954EedeAC495271d0F", "0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0",
	}, []string{"0x5996451816d3789ade9456dd27512895e1312003", "0x5f9946fd12cb85ca24f906b200ccf4c62d780305",
		"0xfc6b05f3e5fcdd27995d6c17a71cf8e7a8f966d2", "0xed1050e2fe96f7327dc659613c79e41458e9cf05",
		"0x948f5d52713fa7fef212703678546b903895d807", "0x16f63f60bf4889a1792d9d639193b4c2fb683a11"}, ""},
	{"quickswap-v4", "BASE_RPC_URL", "0xC5396866754799B9720125B104AE01d935Ab9C7b", []string{
		"0x4200000000000000000000000000000000000006", "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
		"0xd9aAEc86B65D86f6A7B5B1b0c42FFA531710b6CA", "0x50c5725949A6F0c72E6C4a641F24049A917DB0Cb",
	}, []string{"0x5a9ad2bb92b0b3e5c571fdd5125114e04e02be1a"}, ""},
	{"thena-fusion-v3", "BSC_RPC_URL", "0x30055F87716d3DFD0E5198C27024481099fB4A98", []string{
		"0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c", "0x55d398326f99059fF775485246999027B3197955",
		"0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d", "0x2170Ed0880ac9A755fd29B2688956BD959F933F8",
		"0x7130d2A12B9BCbFAe4f2634d864A1Ee1Ce3Ead9c",
	}, []string{"0x9ea0f51fd2133d995cf00229bc523737415ad318"}, ""},
	{"camelot-v4", "ARBITRUM_RPC_URL", "0xBefC4b405041c5833f53412fF997ed2f697a2f37", []string{
		"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1", "0xaf88d065e77c8cC2239327C5EDb3A432268e5831",
		"0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8", "0xFd086bC7CD5C481DCC9C85ebE478A1C0b69FCbb9",
		"0x912CE59144191C1204E64559FE8253a0e49E6548",
	}, []string{"0x20cdea535dcf2b93bce5e4db8786ac5163e28d81"}, "0x7eCfBaa8742fDf5756DAC92fbc8b90a19b8815bF"},
	{"swap-x-cl", "SONIC_RPC_URL", "0x8121a3F8c4176E9765deEa0B95FA2BDfD3016794", []string{
		"0x039e2fB66102314Ce7b64Ce5Ce3E5183bc94aD38", "0x29219dd400f2Bf60E5a23d13Be72B486D4038894",
		"0x50c42dEAcD8Fc9773493ED674b675bE577f2634b", "0x6047828dc181963ba44974801FF68e538dA5eaF9",
	}, []string{"0x5c4b7d607aaf7b5cde9f09b5f03cf3b5c923aeea"}, ""},
}

// TestLensLive: for live pools of each deployment, the lens reads a tick set that checks against the
// pool's liquidity, agrees with FetchRPCData and with every tick the tracker reads by index at the
// same block; a bootstrap through it keeps the plugin data; and a pool without state gets its whole
// state from its first update. Each deployment runs when its RPC URL is set, such as
// ETHEREUM_RPC_URL.
func TestLensLive(t *testing.T) {
	for _, dex := range liveDexes {
		t.Run(dex.dexID, func(t *testing.T) {
			url := os.Getenv(dex.rpcEnv)
			if url == "" {
				t.Skipf("%s is not set", dex.rpcEnv)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Minute)
			defer cancel()

			raw, err := gethrpc.DialOptions(ctx, url, gethrpc.WithHTTPClient(&http.Client{Transport: retryOn429{}}))
			require.NoError(t, err)
			client := ethclient.NewClient(raw)
			rpc := ethrpc.NewWithClient(client).
				SetMulticallContract(common.HexToAddress(cmp.Or(dex.multicall, "0xcA11bde05977b3631167028862bE2a173976CA11")))
			// A free RPC tier answers slowly once rate-limited, so the guard's default timeout of an
			// attempt would cut the stateless update short.
			tracker := NewPoolTracker(&Config{DexID: dex.dexID, UseLens: true, UseBasePluginV2: true,
				LensGuard: ephemeral.GuardConfig{TimeoutSecs: 600}}, rpc, nil)

			pools := livePools(ctx, t, client, dex)
			require.GreaterOrEqual(t, len(pools), 2, "too few live pools to check")

			head, err := client.BlockNumber(ctx)
			require.NoError(t, err)
			block := new(big.Int).SetUint64(head - 3)

			for _, address := range pools {
				checkPool(ctx, t, client, tracker, address, block)
			}
		})
	}
}

func checkPool(ctx context.Context, t *testing.T, client *ethclient.Client, tracker *PoolTracker, address string,
	block *big.Int) {
	snap, err := tracker.snapshot(ctx, address, block)
	require.NoError(t, err, address)
	require.Equal(t, block.Uint64(), snap.block, "%s: the lens read another block", address)

	p := entity.Pool{Address: address, Exchange: tracker.config.DexID, Type: DexType,
		Tokens: liveTokens(ctx, t, client, address)}
	rpcData, err := tracker.FetchRPCData(ctx, &p, block.Uint64(), nil)
	require.NoError(t, err, address)
	require.Zero(t, rpcData.Liquidity.Cmp(snap.liquidity), "%s: liquidity", address)
	require.Equal(t, snap.tick, int(rpcData.State.Tick), "%s: tick", address)
	require.Zero(t, rpcData.State.Price.ToBig().Cmp(snap.sqrtPriceX96), "%s: price", address)

	indexes := make([]int, len(snap.ticks))
	for i, tick := range snap.ticks {
		indexes[i] = tick.TickIdx
		require.Positive(t, tick.LiquidityGross.Sign(), "%s: tick %d holds nothing", address, tick.TickIdx)
		if i > 0 {
			require.Less(t, snap.ticks[i-1].TickIdx, tick.TickIdx, "%s: ticks not ascending", address)
		}
	}
	byIndex, err := tracker.queryRPCTicksByIndexes(ctx, address, indexes, block.Uint64(), nil)
	require.NoError(t, err, address)
	for i, tick := range byIndex {
		require.Equal(t, snap.ticks[i].TickIdx, tick.TickIdx, address)
		require.Zero(t, snap.ticks[i].LiquidityGross.Cmp(tick.LiquidityGross), "%s: gross of %d", address,
			tick.TickIdx)
		require.Zero(t, snap.ticks[i].LiquidityNet.Cmp(tick.LiquidityNet), "%s: net of %d", address, tick.TickIdx)
	}

	booted, err := tracker.BootstrapPoolState(ctx, p, pool.GetNewPoolStateParams{})
	require.NoError(t, err, address)
	require.True(t, hasState(booted), address)
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(booted.Extra), &extra), address)
	require.NotNil(t, extra.GlobalState.Price, address)
	plugin := extra.VolatilityOracle != nil || extra.DynamicFee != nil || extra.SlidingFee != nil

	// A pool as a creation-event backfill leaves it: no Extra, then its first log.
	logs := recentLogs(ctx, t, client, address)
	header, err := client.HeaderByNumber(ctx, nil)
	require.NoError(t, err)
	headers := map[uint64]entity.BlockHeader{header.Number.Uint64(): {Timestamp: header.Time}}
	stateless := entity.Pool{Address: address, Exchange: p.Exchange, Type: DexType, Tokens: p.Tokens}
	updated, err := tracker.GetNewPoolState(ctx, stateless, pool.GetNewPoolStateParams{Logs: logs,
		BlockHeaders: headers})
	require.NoError(t, err, address)
	require.True(t, hasState(updated), "%s: a pool without state was not read whole", address)
	var updatedExtra Extra
	require.NoError(t, json.Unmarshal([]byte(updated.Extra), &updatedExtra), address)
	require.GreaterOrEqual(t, updated.BlockNumber, header.Number.Uint64(), address)

	reset, ok := tracker.ResetPoolState(updated, header.Number.Uint64())
	require.True(t, ok)
	require.False(t, hasState(reset))

	t.Logf("%s: block %d, tick %d, liquidity %s, %d ticks (%d at the bounds), plugin %v (timepoints %d, "+
		"after a stateless update %d ticks at block %d from %d real logs)",
		address, snap.block, snap.tick, snap.liquidity, len(snap.ticks), boundTicks(snap.ticks), plugin,
		len(extra.Timepoints), len(updatedExtra.Ticks), updated.BlockNumber, realLogs(logs))
}

// livePools returns the known pools and those poolByPair finds, keeping the ones with liquidity.
func livePools(ctx context.Context, t *testing.T, client *ethclient.Client, dex liveDex) []string {
	seen := map[string]bool{}
	var pools []string
	add := func(address string) {
		if seen[address] || len(pools) >= 3 {
			return
		}
		seen[address] = true
		liquidity, err := client.CallContract(ctx, ethereum.CallMsg{To: new(common.HexToAddress(address)),
			Data: poolV12ABI.Methods[poolLiquidityMethod].ID}, nil)
		if err == nil && new(big.Int).SetBytes(liquidity).Sign() > 0 {
			pools = append(pools, address)
		}
	}
	for _, address := range dex.known {
		add(address)
	}
	for i, a := range dex.tokens {
		for _, b := range dex.tokens[i+1:] {
			data, err := factoryABI.Pack("poolByPair", common.HexToAddress(a), common.HexToAddress(b))
			require.NoError(t, err)
			out, err := client.CallContract(ctx, ethereum.CallMsg{To: new(common.HexToAddress(dex.factory)),
				Data: data}, nil)
			if err != nil || len(out) < 32 {
				continue
			}
			if address := common.BytesToAddress(out[12:32]); address != (common.Address{}) {
				add(strings.ToLower(address.Hex()))
			}
		}
	}
	return pools
}

func liveTokens(ctx context.Context, t *testing.T, client *ethclient.Client, address string) []*entity.PoolToken {
	var tokens []*entity.PoolToken
	for _, method := range []string{"token0", "token1"} {
		out, err := client.CallContract(ctx, ethereum.CallMsg{To: new(common.HexToAddress(address)),
			Data: poolV12ABI.Methods[method].ID}, nil)
		require.NoError(t, err)
		tokens = append(tokens, &entity.PoolToken{Address: strings.ToLower(common.BytesToAddress(out).Hex()),
			Swappable: true})
	}
	return tokens
}

// recentLogs returns the pool's logs of the last few hundred blocks, scanned 10 blocks at a time as
// free RPC tiers allow, or, when it had none, the empty log at the head that triggers an update.
func recentLogs(ctx context.Context, t *testing.T, client *ethclient.Client, address string) []ethtypes.Log {
	head, err := client.BlockNumber(ctx)
	require.NoError(t, err)
	for to := head - 1; to > head-300; to -= 10 {
		logs, err := client.FilterLogs(ctx, ethereum.FilterQuery{Addresses: []common.Address{
			common.HexToAddress(address)}, FromBlock: new(big.Int).SetUint64(to - 9),
			ToBlock: new(big.Int).SetUint64(to)})
		if err == nil && len(logs) > 0 {
			return logs
		}
	}
	return []ethtypes.Log{{Address: common.HexToAddress(address), BlockNumber: head - 1}}
}

func realLogs(logs []ethtypes.Log) int {
	n := 0
	for _, l := range logs {
		if len(l.Topics) > 0 {
			n++
		}
	}
	return n
}

func boundTicks(ticks []tickspkg.Tick) int {
	n := 0
	for _, tick := range ticks {
		if tick.TickIdx == lensStartTick || tick.TickIdx == -lensStartTick {
			n++
		}
	}
	return n
}

// retryOn429 retries a request a free RPC tier turned away for its rate, after a growing pause.
type retryOn429 struct{}

func (retryOn429) RoundTrip(req *http.Request) (*http.Response, error) {
	var body []byte
	if req.Body != nil {
		var err error
		if body, err = io.ReadAll(req.Body); err != nil {
			return nil, err
		}
	}
	for attempt := 0; ; attempt++ {
		req.Body = io.NopCloser(bytes.NewReader(body))
		resp, err := http.DefaultTransport.RoundTrip(req)
		if err != nil || resp.StatusCode != http.StatusTooManyRequests || attempt == 8 {
			return resp, err
		}
		_ = resp.Body.Close()
		time.Sleep(time.Duration(250<<attempt) * time.Millisecond)
	}
}
