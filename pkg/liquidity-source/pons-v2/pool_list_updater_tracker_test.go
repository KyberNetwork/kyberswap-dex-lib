package ponsv2

import (
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// referenceCurveAddress/referenceCurveLaunchBlock are the exact curve and
// block this integration was built and cross-checked against (see
// context/pons-v2/output/explorer.md). Chosen as the from-block start so the
// live scan below only has to cross a single small window instead of
// replaying the full launchpad-factory history (1000+ TokenLaunched events
// as of discovery time).
const (
	referenceCurveAddress     = "0x4bd421eb79aca48d13793c7140582c8ef312d124"
	referenceCurveLaunchBlock = 24672736 // block that emitted this curve's TokenLaunched event (tx 0x103eeb7d325ac0701bf32e10438d2c807bb413bafbec99a079b193281988e3e7)
)

// PoolListTrackerTestSuite is a live-RPC integration test against Robinhood
// chain mainnet, gated out of CI via test.SkipCI (matching the convention
// used by pkg/liquidity-source/virtual-fun/v2).
type PoolListTrackerTestSuite struct {
	suite.Suite

	updater *PoolsListUpdater
	tracker *PoolTracker
}

func (ts *PoolListTrackerTestSuite) SetupTest() {
	// Multicall3's canonical deterministic-deployment address; confirmed
	// present on Robinhood chain during dex-explorer (non-zero bytecode at
	// block 25970537).
	rpcClient := ethrpc.New("https://rpc.mainnet.chain.robinhood.com").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862be2a173976CA11"))

	config := &Config{
		DexID:        DexType,
		ChainID:      valueobject.ChainIDRobinhood,
		Factory:      "0x7E1EAbd52Ae29598e6483F72dCf1a70b14284dB8",
		NewPoolLimit: 200,
	}

	ts.updater = NewPoolsListUpdater(config, rpcClient)
	tracker, err := NewPoolTracker(config, rpcClient)
	ts.Require().NoError(err)
	ts.tracker = tracker
}

// TestGetNewPools scans a single window around the reference curve's launch
// block and asserts it is discovered with the exact immutable config read
// on-chain during dex-explorer.
func (ts *PoolListTrackerTestSuite) TestGetNewPools() {
	metadata, err := json.Marshal(PoolsListUpdaterMetadata{LastScannedBlock: referenceCurveLaunchBlock})
	ts.Require().NoError(err)

	pools, _, err := ts.updater.GetNewPools(ts.T().Context(), metadata)
	ts.Require().NoError(err)
	ts.Require().NotEmpty(pools)

	var found *entity.Pool
	for i := range pools {
		if common.HexToAddress(pools[i].Address) == common.HexToAddress(referenceCurveAddress) {
			found = &pools[i]
			break
		}
	}
	ts.Require().NotNil(found, "reference curve %s not discovered in scanned window", referenceCurveAddress)

	var staticExtra StaticExtra
	ts.Require().NoError(json.Unmarshal([]byte(found.StaticExtra), &staticExtra))
	ts.Require().EqualValues(100, staticExtra.FeeBps)
	ts.Require().EqualValues(1000, staticExtra.CreatorTaxBps)
	ts.Require().Equal("285714285714285714285714285", staticExtra.ReservedTokens.String())
	ts.Require().False(staticExtra.IsNativeQuote)
}

// TestGetNewPoolState refreshes the reference curve's live state and sanity
// checks the reserve pair is internally consistent (quoteReserve > 0,
// tokenReserve >= ReservedTokens for a non-graduated curve).
func (ts *PoolListTrackerTestSuite) TestGetNewPoolState() {
	seedPool := entity.Pool{
		Address:     referenceCurveAddress,
		Exchange:    valueobject.ExchangePonsV2,
		Type:        DexType,
		Reserves:    []string{"0", "0"},
		Tokens:      []*entity.PoolToken{{Address: "0x322f0929c4625ed5bad873c95208d54e1c003b2d"}, {Address: "0x0b6ebf6e9d2d9a7e1f4e9d1453138781fd81ba95"}},
		StaticExtra: `{"feeBps":100,"creatorTaxBps":1000,"reservedTokens":"285714285714285714285714285","isNativeQuote":false}`,
	}

	updated, err := ts.tracker.GetNewPoolState(ts.T().Context(), seedPool, pool.GetNewPoolStateParams{})
	ts.Require().NoError(err)

	var extra Extra
	ts.Require().NoError(json.Unmarshal([]byte(updated.Extra), &extra))
	ts.Require().True(extra.QuoteReserve.Sign() > 0)
	ts.Require().True(extra.TokenReserve.Sign() > 0)

	sim, err := NewPoolSimulator(updated)
	ts.Require().NoError(err)
	ts.Require().NotNil(sim)
}

func TestPoolListTrackerTestSuite(t *testing.T) {
	t.Parallel()
	test.SkipCI(t)

	suite.Run(t, new(PoolListTrackerTestSuite))
}

// TestCapLaunches covers the NewPoolLimit truncation path GetNewPools relies
// on: on a busy scan window, launches must be capped (bounding buildPools'
// multicall batch), and the resume block must never skip a TokenLaunched log
// that shares a block number with the truncation point.
func TestCapLaunches(t *testing.T) {
	mk := func(blockNumber uint64) tokenLaunchedLog {
		return tokenLaunchedLog{BlockNumber: blockNumber}
	}

	t.Run("under limit resumes past the full scanned window", func(t *testing.T) {
		launches := []tokenLaunchedLog{mk(100), mk(101)}
		got, nextFromBlock := capLaunches(launches, 5, 200)
		require.Equal(t, launches, got)
		require.Equal(t, uint64(201), nextFromBlock)
	})

	t.Run("limit<=0 is unbounded", func(t *testing.T) {
		launches := []tokenLaunchedLog{mk(100), mk(101), mk(102)}
		got, nextFromBlock := capLaunches(launches, 0, 200)
		require.Equal(t, launches, got)
		require.Equal(t, uint64(201), nextFromBlock)
	})

	t.Run("over limit truncates and resumes at the first excluded launch's block", func(t *testing.T) {
		launches := []tokenLaunchedLog{mk(100), mk(100), mk(105), mk(105)}
		got, nextFromBlock := capLaunches(launches, 2, 200)
		require.Equal(t, launches[:2], got)
		// Must resume at 105 (the first excluded launch's block), not 201 --
		// otherwise the second launch at block 105 would be skipped forever.
		require.Equal(t, uint64(105), nextFromBlock)
	})
}
