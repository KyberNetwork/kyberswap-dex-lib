package doppleramm

import (
	"context"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// TestLiveMarketRefresh reads a real Base market end to end: multicall decode of
// reserves and the per-market protocol fee, the quoter probe that recovers the
// module LP fee, and a quote through the simulator. Skipped unless LIVE_RPC is
// set, because it needs a Base RPC and a market that exists right now.
func TestLiveMarketRefresh(t *testing.T) {
	rpc := os.Getenv("LIVE_RPC")
	if rpc == "" {
		t.Skip("LIVE_RPC not set")
	}

	const marketID = "0xcab05a53e320d3f8d8f72e749bda740e02b504c32e8f91a63bfd3e7bd4c6ad74"
	staticExtra, err := json.Marshal(StaticExtra{
		MarketID: marketID,
		TokenX:   "0x4200000000000000000000000000000000000006",
		TokenY:   "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
		AMM:      "0x533b7ee11f4f9be09191ebf6667abe73c01fe2c2",
		Quoter:   "0x384e4c09071f7e511be34e38b4fcba23be51ea55",
	})
	require.NoError(t, err)

	ep := entity.Pool{
		Address:  marketID,
		Exchange: DexType,
		Type:     DexType,
		Reserves: entity.PoolReserves{"0", "0"},
		Tokens: []*entity.PoolToken{
			{Address: "0x4200000000000000000000000000000000000006", Swappable: true},
			{Address: "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913", Swappable: true},
		},
		StaticExtra: string(staticExtra),
	}

	client := ethrpc.New(rpc).
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	tracker, err := NewPoolTracker(testConfig(), client)
	require.NoError(t, err)

	refreshed, err := tracker.GetNewPoolState(context.Background(), ep, pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	t.Logf("reserves=%v extra=%s block=%d", refreshed.Reserves, refreshed.Extra, refreshed.BlockNumber)

	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(refreshed.Extra), &extra))
	require.Equal(t, uint16(10), extra.BaseFeeBps)
	require.Equal(t, uint16(5), extra.LpFeeBps, "market 0xcab05a53 charges 5 bps LP fee")

	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: refreshed})
	require.NoError(t, err)

	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: ep.Tokens[0].Address, Amount: bi("1000000000000")},
		TokenOut:      ep.Tokens[1].Address,
	})
	require.NoError(t, err)
	require.Equal(t, "2096", res.TokenAmountOut.Amount.String(), "matches quoteSwapExactInput")
}

// TestLiveMarketDiscovery backfills every MarketCreated log on Base and checks that the 6 known spot
// markets (including 0xcab05a53, WETH/USDC) and 4 launch markets are listed under their exchange.
func TestLiveMarketDiscovery(t *testing.T) {
	rpc := os.Getenv("LIVE_RPC")
	if rpc == "" {
		t.Skip("LIVE_RPC not set")
	}

	client := ethrpc.New(rpc).
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	pools := listLivePools(t, client)

	byExchange := map[string]map[string]*entity.Pool{}
	for i, p := range pools {
		if byExchange[p.Exchange] == nil {
			byExchange[p.Exchange] = map[string]*entity.Pool{}
		}
		byExchange[p.Exchange][p.Address] = &pools[i]
	}
	require.GreaterOrEqual(t, len(byExchange["doppler-amm"]), 6)
	require.GreaterOrEqual(t, len(byExchange["doppler-fun"]), 4)
	spot := byExchange["doppler-amm"]["0xcab05a53e320d3f8d8f72e749bda740e02b504c32e8f91a63bfd3e7bd4c6ad74"]
	require.NotNil(t, spot)
	require.Equal(t, DexType, spot.Type)
	require.Equal(t, "0x4200000000000000000000000000000000000006", spot.Tokens[0].Address)
	require.Equal(t, "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913", spot.Tokens[1].Address)
	t.Logf("spot=%d launch=%d", len(byExchange["doppler-amm"]), len(byExchange["doppler-fun"]))
}
