package stonkbrokersfunv2

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
)

const (
	defaultRobinhoodRPCURL = "https://rpc.mainnet.chain.robinhood.com"
	multicallAddress       = "0xcA11bde05977b3631167028862bE2a173976CA11"
	lensAddress            = "0x25b5Df581f4b2Ed450203f375ad8A28b17F115B3"

	// wethV3Pad is the WETH-lane Smart Launch V3 pad. V3 is deliberately out of
	// the configured pad list -- see TestV3Pad_ScopeDecision.
	wethV3Pad = "0x5BCEefBa6fDf437A7388aDC5c9056c827baca3B3"
)

// robinhoodRPCURL is the endpoint the live tests dial. The public RPC rate
// limits aggressively (and answers Cloudflare challenges once it does), so
// STONKBROKERS_RPC_URL can point these at a private/archive endpoint. Keep the
// key in the environment, never in the repo.
func robinhoodRPCURL() string {
	if u := os.Getenv("STONKBROKERS_RPC_URL"); u != "" {
		return u
	}
	return defaultRobinhoodRPCURL
}

// testPoolJSON is this package's reference launch: WETH V2 pad, on-chain
// launch id 176. Reserves start
// at "0","0" -- this is exactly what discovery leaves behind
// (AGENTS.md: don't set reserves at discovery time); GetNewPoolState fills
// the real values.
const testPoolJSON = `{
	"address": "0xfcd61b25bbf3abd6cf0070d6328e351cc30eec9f_176",
	"exchange": "stonkbrokers-fun-v2",
	"type": "stonkbrokers-fun-v2",
	"timestamp": 1787720000,
	"reserves": ["0", "0"],
	"tokens": [
		{"address": "0x391d8735013cc60f7cca0f2ee611a14dc2e66666", "swappable": true},
		{"address": "0x0bd7d308f8e1639fab988df18a8011f41eacad73", "swappable": true}
	],
	"extra": "{}",
	"staticExtra": "{\"pad\":\"0xfcd61b25bbf3abd6cf0070d6328e351cc30eec9f\",\"lens\":\"0x25b5df581f4b2ed450203f375ad8a28b17f115b3\",\"launchId\":\"176\",\"isWethLane\":true,\"quoteDecimals\":18,\"bufferTaxBps\":9999,\"startTaxBps\":0,\"decayPerMinuteBps\":0,\"bufferSecs\":0,\"windowSecs\":0,\"startTime\":1787691927,\"deadline\":1787691927,\"openEnded\":true,\"postTaxBps\":100,\"maxBuyPpm\":0,\"gradMcapUsd8\":5000000000000,\"loadedSupply\":\"1000000000000000000000000000\",\"quoteUsdFeed\":\"0x78f3556b67e17df817d51ef5a990cdaf09e8d3a9\",\"ethUsdFeed\":\"0x78f3556b67e17df817d51ef5a990cdaf09e8d3a9\"}"
}`

type PoolTrackerTestSuite struct {
	suite.Suite
	tracker *PoolTracker
}

func (ts *PoolTrackerTestSuite) SetupSuite() {
	cfg := &Config{
		DexID:   DexType,
		ChainID: 4663,
	}
	client := ethrpc.New(robinhoodRPCURL()).
		SetMulticallContract(common.HexToAddress(multicallAddress))

	tracker, err := NewPoolTracker(cfg, client)
	ts.Require().NoError(err)
	ts.tracker = tracker
}

// TestGetNewPoolState_LiveWethLaunch176 is the tracker-side half of the
// live-verified WETH launch 176 fixture (paired with
// TestCalcAmountOut_LiveVerifiedWethLaunch176 in pool_simulator_test.go).
// It hits the real chain -- SkipCI so it doesn't flake CI on a network hop,
// but genuinely proves GetNewPoolState decodes real on-chain bytes.
func (ts *PoolTrackerTestSuite) TestGetNewPoolState_LiveWethLaunch176() {
	t := ts.T()

	var p entity.Pool
	require.NoError(t, json.Unmarshal([]byte(testPoolJSON), &p))

	updated, err := ts.tracker.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
	require.NoError(t, err)

	t.Logf("blockNumber: %d", updated.BlockNumber)
	t.Logf("reserves:    %v", updated.Reserves)
	t.Logf("extra:       %s", updated.Extra)

	require.Greater(t, updated.BlockNumber, uint64(46_325_622), "block must be pinned forward, never fabricated")
	require.Len(t, updated.Reserves, 2)
	require.NotEqual(t, "0", updated.Reserves[0])
	require.NotEqual(t, "0", updated.Reserves[1])

	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(updated.Extra), &extra))
	require.True(t, extra.Armed)
	require.False(t, extra.Aborted)
	require.NotNil(t, extra.DirectFeed)
	require.True(t, extra.DirectFeed.Ok, "WETH lane's quoteUsdFeed must resolve (direct-feed mode)")
	require.Nil(t, extra.Twap, "WETH lane is direct-feed mode, Twap must stay unset")
}

func TestPoolTrackerTestSuite(t *testing.T) {
	t.Parallel()
	test.SkipCI(t)
	suite.Run(t, new(PoolTrackerTestSuite))
}

// TestV3Pad_ScopeDecision records WHY the 8 Smart Launch V3 (external-token /
// BYO) pads are deliberately NOT in the pad list, and keeps that decision
// honest by re-checking it against the chain.
//
// V3 needs no code of its own: the pads share the StonkSafeLaunchpadV2 ABI and
// the SafeLaunchLensV2 lens, and `externalToken` -- the only distinguishing
// field -- is written in createLaunch, emitted in LaunchCreated, and read
// nowhere else in the pad or its four linked libraries, so it never branches
// the buy path. Adding them would be a config-only change.
//
// It is not worth making. Exactly one launch exists across all 8 V3 pads (WETH
// lane, id 1) and it is eoaOnly, so buy() reverts NotEoa() for any contract
// caller including the executor. Discovery drops eoaOnly launches, so pointing
// the lister at the V3 pad yields nothing at all: V3 would add zero routable
// liquidity today. If a routable V3 launch ever appears, this test starts
// failing and the decision gets revisited.
func (ts *PoolTrackerTestSuite) TestV3Pad_ScopeDecision() {
	t := ts.T()

	client := ethrpc.New(robinhoodRPCURL()).
		SetMulticallContract(common.HexToAddress(multicallAddress))

	lister := NewPoolsListUpdater(&Config{
		DexID: DexType, ChainID: 4663,
		Pads: []string{wethV3Pad}, Lens: lensAddress,
	}, client)

	pools, _, err := lister.GetNewPools(context.Background(), nil)
	ts.Require().NoError(err, "the V2 lister must decode a V3 pad unchanged")
	ts.Require().Empty(pools,
		"V3's only launch is eoaOnly and must be dropped; %d pool(s) came back, so a routable V3 launch now exists",
		len(pools))
	t.Logf("V3 WETH pad: %d routable launches (expected 0)", len(pools))
}

func mustStaticExtra(t *testing.T, p entity.Pool) StaticExtra {
	t.Helper()
	var se StaticExtra
	require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &se))
	return se
}

// TestQuoteMatchesLens is the correctness property that matters most: the
// simulator's price must equal what the pad's own SafeLaunchLensV2 quotes, to
// the wei, across the whole amount range. The lens is an independent on-chain
// implementation of the same curve, so agreeing with it catches any drift in
// the ported tax/rounding maths.
func (ts *PoolTrackerTestSuite) TestQuoteMatchesLens() {
	t := ts.T()
	ctx := context.Background()

	client := ethrpc.New(robinhoodRPCURL()).
		SetMulticallContract(common.HexToAddress(multicallAddress))

	var ep entity.Pool
	ts.Require().NoError(json.Unmarshal([]byte(testPoolJSON), &ep))
	p, err := ts.tracker.GetNewPoolState(ctx, ep, pool.GetNewPoolStateParams{})
	ts.Require().NoError(err)

	sim, err := NewPoolSimulator(p)
	ts.Require().NoError(err)

	launchID, ok := new(big.Int).SetString(mustStaticExtra(t, p).LaunchID, 10)
	ts.Require().True(ok)

	for _, amountIn := range []*big.Int{
		big.NewInt(1),
		big.NewInt(1_000_000_000_000),
		big.NewInt(10_000_000_000_000_000),
		big.NewInt(1_000_000_000_000_000_000),
	} {
		res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: p.Tokens[1].Address, Amount: amountIn},
			TokenOut:      p.Tokens[0].Address,
		})
		ts.Require().NoError(err, "amountIn=%s", amountIn)

		// quoteBuy returns two values, so ethrpc takes the copyTuple path and
		// needs one struct destination.
		var lens struct {
			TokensOut *big.Int
			TaxBps    *big.Int
		}
		req := client.NewRequest().SetContext(ctx).SetBlockNumber(new(big.Int).SetUint64(p.BlockNumber))
		req.AddCall(&ethrpc.Call{
			ABI: lensABI, Target: lensAddress, Method: methodQuoteBuy,
			Params: []any{common.HexToAddress(wethPad), launchID, amountIn},
		}, []any{&lens})
		_, err = req.Aggregate()
		ts.Require().NoError(err)

		t.Logf("amountIn=%-20s sim=%-30s lens=%-30s taxBps=%s",
			amountIn, res.TokenAmountOut.Amount, lens.TokensOut, lens.TaxBps)
		ts.Require().Zero(res.TokenAmountOut.Amount.Cmp(lens.TokensOut),
			"simulator disagrees with the lens at amountIn=%s", amountIn)
	}
}
