package ponsv2

import (
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/suite"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
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
	referenceQuoteToken       = "0x322f0929c4625ed5bad873c95208d54e1c003b2d"
	referenceLaunchedToken    = "0x0b6ebf6e9d2d9a7e1f4e9d1453138781fd81ba95"
)

// PoolFactoryTrackerTestSuite is a live-RPC integration test against Robinhood
// chain mainnet, gated out of CI via test.SkipCI (matching the convention
// used by pkg/liquidity-source/virtual-fun/v2).
type PoolFactoryTrackerTestSuite struct {
	suite.Suite

	backfiller *poolfactory.FilterLogsBackfiller
	tracker    *PoolTracker
}

func (ts *PoolFactoryTrackerTestSuite) SetupTest() {
	// Multicall3's canonical deterministic-deployment address; confirmed
	// present on Robinhood chain during dex-explorer (non-zero bytecode at
	// block 25970537).
	rpcClient := ethrpc.New("https://rpc.mainnet.chain.robinhood.com").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862be2a173976CA11"))

	config := &Config{
		DexID:   DexType,
		ChainID: valueobject.ChainIDRobinhood,
		Factory: "0x7E1EAbd52Ae29598e6483F72dCf1a70b14284dB8",
	}

	backfiller, err := poolfactory.NewFilterLogsBackfiller(
		NewPoolFactoryDecoder(config), rpcClient, poolfactory.BackfillConfig{
			FactoryAddresses:     []common.Address{common.HexToAddress(config.Factory)},
			StartBlock:           referenceCurveLaunchBlock,
			MaxBlockRangePerScan: 5_000,
		},
	)
	ts.Require().NoError(err)
	ts.backfiller = backfiller

	tracker, err := NewPoolTracker(config, rpcClient)
	ts.Require().NoError(err)
	ts.tracker = tracker
}

// TestBackfillDiscoversReferenceCurve scans a single window around the
// reference curve's launch block (via the same poolfactory.FilterLogsBackfiller
// pool-service drives for historical discovery) and asserts it is discovered
// with the tokens implied by the log alone. Fee/tax/reserved-token/graduated
// state is no longer known at discovery time -- see TestGetNewPoolState,
// where that ground truth now lives.
func (ts *PoolFactoryTrackerTestSuite) TestBackfillDiscoversReferenceCurve() {
	pools, _, _, err := ts.backfiller.Backfill(ts.T().Context(), nil)
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
	ts.Require().Len(found.Tokens, 2)
	ts.Require().Equal(common.HexToAddress(referenceQuoteToken), common.HexToAddress(found.Tokens[0].Address))
	ts.Require().Equal(common.HexToAddress(referenceLaunchedToken), common.HexToAddress(found.Tokens[1].Address))
}

// TestGetNewPoolState refreshes the reference curve's live state and asserts
// both the mutable reserve pair and the immutable curve config
// (fee/tax/reserved-tokens/isNativeQuote) match what dex-explorer read
// on-chain. This is now the only place that config is fetched -- discovery
// (pool_factory.go's DecodePoolCreated) is RPC-free.
func (ts *PoolFactoryTrackerTestSuite) TestGetNewPoolState() {
	seedPool := entity.Pool{
		Address:  referenceCurveAddress,
		Exchange: valueobject.ExchangePonsV2,
		Type:     DexType,
		Reserves: []string{"0", "0"},
		Tokens:   []*entity.PoolToken{{Address: referenceQuoteToken}, {Address: referenceLaunchedToken}},
	}

	updated, err := ts.tracker.GetNewPoolState(ts.T().Context(), seedPool, pool.GetNewPoolStateParams{})
	ts.Require().NoError(err)

	var extra Extra
	ts.Require().NoError(json.Unmarshal([]byte(updated.Extra), &extra))
	ts.Require().True(extra.QuoteReserve.Sign() > 0)
	ts.Require().True(extra.TokenReserve.Sign() > 0)

	var staticExtra StaticExtra
	ts.Require().NoError(json.Unmarshal([]byte(updated.StaticExtra), &staticExtra))
	ts.Require().EqualValues(100, staticExtra.FeeBps)
	ts.Require().EqualValues(1000, staticExtra.CreatorTaxBps)
	ts.Require().Equal("285714285714285714285714285", staticExtra.ReservedTokens.String())
	ts.Require().False(staticExtra.IsNativeQuote)

	sim, err := NewPoolSimulator(updated)
	ts.Require().NoError(err)
	ts.Require().NotNil(sim)
}

func TestPoolFactoryTrackerTestSuite(t *testing.T) {
	t.Parallel()
	test.SkipCI(t)

	suite.Run(t, new(PoolFactoryTrackerTestSuite))
}
