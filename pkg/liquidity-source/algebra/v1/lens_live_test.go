package algebrav1

import (
	"context"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	v3Entities "github.com/daoleno/uniswapv3-sdk/entities"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	sourcePool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// liveExchange is an Algebra V1 deployment, how its tick table is indexed, and tokens common enough
// that its factory has pools of them.
type liveExchange struct {
	dexID   string
	chain   string
	factory string
	dirFee  bool
	table   tickTable
	tokens  []string
}

var liveExchanges = []liveExchange{
	{"camelot-v3", "ARBITRUM", "0x1a3c9B1d2F0529D97f2afC5136Cc23e58f1FD35B", true, tickTableRaw, []string{
		"0x82aF49447D8a07e3bd95BD0d56f35241523fBab1", "0xaf88d065e77c8cC2239327C5EDb3A432268e5831",
		"0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8", "0xFd086bC7CD5C481DCC9C85ebE478A1C0b69FCbb9",
		"0x912CE59144191C1204E64559FE8253a0e49E6548"}},
	{"quickswap-v3", "POLYGON", "0x411b0fAcC3489691f28ad58c47006AF5E3Ab3A28", false, tickTableCompressed, []string{
		"0x0d500B1d8E8eF31E21C99d1Db9A6444d3ADf1270", "0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174",
		"0x7ceB23fD6bC0adD59E62ac25578270cFf1b9f619", "0xc2132D05D31c914a87C6611C10748AEb04B58e8F",
		"0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359"}},
	{"thena-fusion", "BSC", "0x306F06C147f064A010530292A1EB6737c3e378e4", false, tickTableCompressed, []string{
		"0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c", "0x55d398326f99059fF775485246999027B3197955",
		"0xe9e7CEA3DedcA5984780Bafc599bD69ADd087D56", "0x2170Ed0880ac9A755fd29B2688956BD959F933F8",
		"0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d"}},
	{"zyberswap-v3", "OPTIMISM", "0x0C8f7b0cb986b31c67D994fb5c224592A03A4AfD", false, tickTableRaw, []string{
		"0x4200000000000000000000000000000000000006", "0x0b2C639c533813f4Aa9D7837CAf62653d097Ff85",
		"0x7F5c764cBc14f9669B88837ca1490cCa17c31607", "0x94b008aA00579c1307B0EF2c499aD98a8ce58e58"}},
	{"swapbased-v3", "BASE", "0xc207628e5e2b59e9c690071e68c7c1c4193b0252", false, tickTableRaw, []string{
		"0x4200000000000000000000000000000000000006", "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
		"0xd9aAEc86B65D86f6A7B5B1b0c42FFA531710b6CA", "0x50c5725949A6F0c72E6C4a641F24049A917DB0Cb"}},
}

// TestLiveLens: on each deployment, a bootstrap through the lens learns how the exchange indexes
// its tick table, holds the price and liquidity the pool reports and every tick the pool reports
// for those indices at the same block, reads the block it was pinned at, and a pool without state
// gets it whole from its first update. Run with one RPC URL per chain:
//
//	ALGEBRA_V1_RPC_ARBITRUM=https://... ALGEBRA_V1_RPC_POLYGON=https://... ALGEBRA_V1_RPC_BSC=... \
//	ALGEBRA_V1_RPC_OPTIMISM=... ALGEBRA_V1_RPC_BASE=... \
//	go test ./pkg/liquidity-source/algebra/v1/ -run LiveLens -v
func TestLiveLens(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("live")
	}
	ran := false
	for _, ex := range liveExchanges {
		rpcURL := os.Getenv("ALGEBRA_V1_RPC_" + ex.chain)
		if rpcURL == "" {
			continue
		}
		ran = true
		t.Run(ex.dexID+"/"+strings.ToLower(ex.chain), func(t *testing.T) { liveExchangeLens(t, ex, rpcURL) })
	}
	if !ran {
		t.Skip("set ALGEBRA_V1_RPC_<CHAIN> to run this")
	}
}

func liveExchangeLens(t *testing.T, ex liveExchange, rpcURL string) {
	ctx := t.Context()
	live := testutil.DialLive(t, rpcURL)
	ec := live.Eth
	client := live.Ethrpc
	tracker := NewPoolTracker(&Config{DexID: ex.dexID, UseDirectionalFee: ex.dirFee, SkipFeeCalculating: true,
		UseLens: true}, client, nil)

	pools := livePools(ctx, ec, ex)
	require.NotEmpty(t, pools, "no pool with liquidity among the tokens")

	for _, p := range pools {
		// A load-balanced endpoint may answer the pinned reads from a node a block behind the lens'
		// latest one; on a chain of quarter-second blocks a retry finds it caught up.
		var boot entity.Pool
		var err error
		for range 3 {
			if boot, err = tracker.BootstrapPoolState(ctx, p, sourcePool.GetNewPoolStateParams{}); err == nil {
				break
			}
			time.Sleep(time.Second)
		}
		require.NoError(t, err, p.Address)
		var extra Extra
		require.NoError(t, json.Unmarshal([]byte(boot.Extra), &extra))

		// The price and liquidity against the pool's own answers, at the lens's block.
		rpcData, err := tracker.FetchRPCData(ctx, &p, boot.BlockNumber)
		require.NoError(t, err)
		assert.Zero(t, rpcData.Liquidity.Cmp(extra.Liquidity), "liquidity of %s", p.Address)
		assert.Zero(t, rpcData.State.Price.Cmp(extra.GlobalState.Price), "price of %s", p.Address)

		// Each tick against the pool's own answer for it, at the lens's block.
		viaRPC, err := tracker.queryRPCTicksByIndexes(ctx, p.Address,
			lo.Map(extra.Ticks, func(tk v3Entities.Tick, _ int) int { return tk.Index }), boot.BlockNumber)
		require.NoError(t, err)
		byIndex := lo.SliceToMap(viaRPC, func(tk tickspkg.Tick) (int, tickspkg.Tick) { return tk.TickIdx, tk })
		for _, tk := range extra.Ticks {
			got := byIndex[tk.Index]
			assert.Zero(t, tk.LiquidityGross.Cmp(got.LiquidityGross), "gross of tick %d of %s", tk.Index, p.Address)
			assert.Zero(t, tk.LiquidityNet.Cmp(got.LiquidityNet), "net of tick %d of %s", tk.Index, p.Address)
		}

		// A pinned read reads that block, the chain's own.
		pinned := boot.BlockNumber - 2
		snap, err := tracker.snapshot(ctx, p.Address, new(big.Int).SetUint64(pinned))
		require.NoError(t, err)
		assert.Equal(t, pinned, snap.block, "the lens read another block than pinned")

		t.Logf("%s spacing %d, %d ticks at block %d, table %d", p.Address, extra.TickSpacing, len(extra.Ticks),
			boot.BlockNumber, tracker.tickTable.Load())
	}
	assert.Equal(t, ex.table, tracker.tickTable.Load(), "learnt the wrong tick table")

	// A pool without state gets it whole from its first update, at the batch's newest block.
	p, log := liveLog(t, ctx, ec, pools)
	if log == nil {
		t.Log("no recent log of the pools; skipping the stateless update")
		return
	}
	stateless := p
	stateless.Extra = ""
	fresh, err := tracker.GetNewPoolState(ctx, stateless, sourcePool.GetNewPoolStateParams{
		Logs:         []ethtypes.Log{*log},
		BlockHeaders: map[uint64]entity.BlockHeader{log.BlockNumber: {}},
	})
	require.NoError(t, err)
	assert.True(t, tickspkg.HasState(fresh), "the pool did not get its state")
	assert.Equal(t, log.BlockNumber, fresh.BlockNumber)
	var freshExtra Extra
	require.NoError(t, json.Unmarshal([]byte(fresh.Extra), &freshExtra))
	t.Logf("stateless %s: %d ticks at block %d from a log", p.Address, len(freshExtra.Ticks), fresh.BlockNumber)
}

// livePools returns up to three pools with liquidity among the exchange's tokens.
func livePools(ctx context.Context, ec *ethclient.Client, ex liveExchange) []entity.Pool {
	var pools []entity.Pool
	for i := 0; i < len(ex.tokens) && len(pools) < 3; i++ {
		for j := i + 1; j < len(ex.tokens) && len(pools) < 3; j++ {
			addr := callAddress(ctx, ec, ex.factory, "d9a641e1", ex.tokens[i], ex.tokens[j])
			if addr == (common.Address{}) || callWord(ctx, ec, addr.Hex(), "1a686502").Sign() == 0 {
				continue
			}
			pools = append(pools, entity.Pool{
				Address:  strings.ToLower(addr.Hex()),
				Exchange: ex.dexID,
				Type:     DexTypeAlgebraV1,
				Tokens: []*entity.PoolToken{
					{Address: strings.ToLower(callAddress(ctx, ec, addr.Hex(), "0dfe1681").Hex()), Swappable: true},
					{Address: strings.ToLower(callAddress(ctx, ec, addr.Hex(), "d21220a7").Hex()), Swappable: true},
				},
			})
		}
	}
	return pools
}

// liveLog finds a recent log of one of pools, ten blocks at a time as free endpoints allow.
func liveLog(t *testing.T, ctx context.Context, ec *ethclient.Client, pools []entity.Pool) (entity.Pool,
	*ethtypes.Log) {
	head, err := ec.BlockNumber(ctx)
	require.NoError(t, err)
	addresses := lo.Map(pools, func(p entity.Pool, _ int) common.Address { return common.HexToAddress(p.Address) })
	for to := head; to > head-1000; to -= 10 {
		logs, err := ec.FilterLogs(ctx, ethereum.FilterQuery{Addresses: addresses,
			FromBlock: new(big.Int).SetUint64(to - 9), ToBlock: new(big.Int).SetUint64(to)})
		if err != nil || len(logs) == 0 {
			continue
		}
		l := logs[len(logs)-1]
		p, _ := lo.Find(pools, func(p entity.Pool) bool { return common.HexToAddress(p.Address) == l.Address })
		return p, &l
	}
	return entity.Pool{}, nil
}

func callWord(ctx context.Context, ec *ethclient.Client, to, selector string, args ...string) *big.Int {
	data := common.FromHex(selector)
	for _, a := range args {
		data = append(data, common.LeftPadBytes(common.HexToAddress(a).Bytes(), 32)...)
	}
	target := common.HexToAddress(to)
	out, err := ec.CallContract(ctx, ethereum.CallMsg{To: &target, Data: data}, nil)
	if err != nil || len(out) < 32 {
		return new(big.Int)
	}
	return new(big.Int).SetBytes(out[:32])
}

func callAddress(ctx context.Context, ec *ethclient.Client, to, selector string, args ...string) common.Address {
	return common.BigToAddress(callWord(ctx, ec, to, selector, args...))
}
