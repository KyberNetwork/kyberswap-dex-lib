package stonkbrokersfunv2

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
)

// wethV2Pad is used as a single-pad config for the list-updater test so the
// live call stays small -- production Config.Pads carries all 8 lanes (see
// the eight quote lanes).
const wethV2Pad = "0xFCd61B25BbF3AbD6cf0070D6328E351cc30EEC9f"

type PoolsListUpdaterTestSuite struct {
	suite.Suite
	updater *PoolsListUpdater
}

func (ts *PoolsListUpdaterTestSuite) SetupSuite() {
	cfg := &Config{
		DexID:   DexType,
		ChainID: 4663,
		Pads:    []string{wethV2Pad},
		Lens:    "0x25b5Df581f4b2Ed450203f375ad8A28b17F115B3",
	}
	client := ethrpc.New(robinhoodRPCURL()).
		SetMulticallContract(common.HexToAddress(multicallAddress))
	ts.updater = NewPoolsListUpdater(cfg, client)
}

// TestGetNewPools_DiscoversFixedPadOnChain proves the on-chain, cursor-based
// discovery (findings.explorer.pool_discovery: view_enum) actually decodes
// launchCount()/getLaunch()/modesOf()/bufferSecsOf() against the live WETH
// pad, and that reserves are deliberately left "0","0" at discovery time
// (AGENTS.md) for the tracker to fill.
func (ts *PoolsListUpdaterTestSuite) TestGetNewPools_DiscoversFixedPadOnChain() {
	t := ts.T()

	pools, metadata, err := ts.updater.GetNewPools(context.Background(), nil)
	ts.Require().NoError(err)
	ts.Require().NotEmpty(pools, "WETH pad has at least launch id 176 (and likely far more) discoverable")
	t.Logf("discovered %d pools, first address %s", len(pools), pools[0].Address)
	t.Logf("cursor metadata: %s", string(metadata))

	for _, p := range pools {
		ts.Require().Equal("0", p.Reserves[0])
		ts.Require().Equal("0", p.Reserves[1])
		ts.Require().Len(p.Tokens, 2)
		ts.Require().NotEmpty(p.StaticExtra)
	}

	// The two LaunchModes flags that decide whether a route can execute at
	// all must survive into StaticExtra -- CalcAmountOut refuses on either,
	// and it can only do that if discovery persisted them. eoaOnly in
	// particular is set on 74 of the 283 launches live at the time of
	// writing, so a full pad scan must observe at least one.
	var sawEoaOnly, sawCap bool
	for _, p := range pools {
		var se StaticExtra
		ts.Require().NoError(json.Unmarshal([]byte(p.StaticExtra), &se))
		sawEoaOnly = sawEoaOnly || se.EoaOnly
		sawCap = sawCap || se.MaxBuyPpm != 0
	}
	t.Logf("eoaOnly seen: %v, maxBuyPpm seen: %v", sawEoaOnly, sawCap)
	ts.Require().True(sawEoaOnly, "WETH pad has eoaOnly launches; StaticExtra must carry the flag")

	// Re-running with the returned cursor must not re-discover anything
	// already seen (idempotent forward-only paging).
	pools2, _, err := ts.updater.GetNewPools(context.Background(), metadata)
	ts.Require().NoError(err)
	ts.Require().Empty(pools2, "cursor must prevent re-discovering already-seen launches")
}

func TestPoolsListUpdaterTestSuite(t *testing.T) {
	t.Parallel()
	test.SkipCI(t)
	suite.Run(t, new(PoolsListUpdaterTestSuite))
}
