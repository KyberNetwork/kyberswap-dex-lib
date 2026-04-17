package fermi

import (
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
)

// PoolListTrackerTestSuite exercises the full PoolsListUpdater → PoolTracker pipeline
// against live Ethereum mainnet. These tests are skipped in CI (require live RPC).
type PoolListTrackerTestSuite struct {
	suite.Suite

	updater *PoolsListUpdater
	tracker *PoolTracker
}

func (ts *PoolListTrackerTestSuite) SetupTest() {
	rpcURL := "https://eth.drpc.org"
	rpcClient := ethrpc.New(rpcURL).
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	config := &Config{
		DexId:        DexType,
		ChainId:      1,
		FermiSwapper: fermiSwapperAddr,
		FermiEngine:  fermiEngineAddr,
		TraderVault:  fermiTraderVaultAddr,
		Titan: TitanConfig{
			URLs: []string{testTitanURLEU, testTitanURLAP, testTitanURLUS},
		},
	}

	ts.updater = NewPoolsListUpdater(config, rpcClient)
	tracker, err := NewPoolTracker(config, rpcClient)
	require.NoError(ts.T(), err)
	ts.tracker = tracker
}

func (ts *PoolListTrackerTestSuite) TestGetNewPools() {
	ctx := ts.T().Context()

	pools, _, err := ts.updater.GetNewPools(ctx, nil)
	require.NoError(ts.T(), err)
	require.NotEmpty(ts.T(), pools, "expected at least one active FermiSwap pair")

	ts.T().Logf("discovered %d FermiSwap pools", len(pools))
	for _, p := range pools {
		ts.T().Logf("  pool=%s tokens=%v", p.Address, func() []string {
			addrs := make([]string, len(p.Tokens))
			for i, t := range p.Tokens {
				addrs[i] = t.Address
			}
			return addrs
		}())
		require.Equal(ts.T(), DexType, p.Type)
		require.Len(ts.T(), p.Tokens, 2)
	}
}

func (ts *PoolListTrackerTestSuite) TestGetNewPoolState() {
	ctx := ts.T().Context()

	pools, _, err := ts.updater.GetNewPools(ctx, nil)
	require.NoError(ts.T(), err)
	require.NotEmpty(ts.T(), pools)

	for _, p := range pools {
		ts.T().Logf("tracking pool %s", p.Address)

		newState, err := ts.tracker.GetNewPoolState(ctx, p, pool.GetNewPoolStateParams{})
		require.NoError(ts.T(), err, "GetNewPoolState failed for pool %s", p.Address)

		// Extra must be set.
		require.NotEmpty(ts.T(), newState.Extra, "Extra must not be empty after tracking")

		// Deserialize and validate the Extra.
		sim, err := NewPoolSimulator(newState)
		require.NoError(ts.T(), err)
		require.NotNil(ts.T(), sim)

		ts.T().Logf("  curve=%v  block=%d  overrides=%v", sim.curve != nil, sim.blockNumber, sim.stateOverrides != nil)

		// Verify state overrides flow through to PoolMeta.
		meta, ok := sim.GetMetaInfo("", "").(PoolMeta)
		require.True(ts.T(), ok)
		require.NotNil(ts.T(), meta.StateOverrides, "PoolMeta must carry state overrides from Titan")
	}
}

func TestPoolListTrackerTestSuite(t *testing.T) {
	t.Parallel()
	test.SkipCI(t)
	if testing.Short() {
		t.Skip("live network test; pass without -short to run")
	}

	suite.Run(t, new(PoolListTrackerTestSuite))
}
