package integral

import (
	"context"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/algebra"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestPoolTracker_GetNewPoolState(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip()
	}

	d := &PoolTracker{
		PoolTracker: algebra.PoolTracker[Timepoint, TimepointRPC]{
			EthrpcClient: ethrpc.NewWithClient(lo.Must(ethclient.Dial("https://bsc.kyberengineering.io"))).
				SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")),
		},
		config: &Config{
			DexID:              valueobject.ExchangeThenaFusionV3,
			AllowSubgraphError: true,
			UseBasePluginV2:    true,
		},
		graphqlClient: graphqlpkg.NewClient("https://thegraph.com/explorer/api/playground/QmWSzHwZY9ZMNYMVbQLyL276V1toR3iZsnYMfQut166yit"),
	}
	got, err := d.GetNewPoolState(context.Background(), thenaEp, pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	t.Log(string(lo.Must(json.Marshal(got))))
}

func TestPoolTracker_GetNewPoolState_TrebleSwap(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip()
	}

	d := &PoolTracker{
		PoolTracker: algebra.PoolTracker[Timepoint, TimepointRPC]{
			EthrpcClient: ethrpc.NewWithClient(lo.Must(ethclient.Dial("https://mainnet.base.org"))).
				SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")),
		},
		config: &Config{
			DexID:              valueobject.ExchangeTrebleSwap,
			AllowSubgraphError: true,
		},
		// Subgraph URL requires a valid API key: replace {key} with your The Graph API key.
		// Full spec: VITE_INFO_GRAPH=https://gateway.thegraph.com/api/{key}/subgraphs/id/3sThy2UsWd9X3D2M6MUQWzNUYrs8snMMhQKHSg9kUEAd
		// TickLens contract on Base: 0x195d7ACc03F4C77150b64300138AB837d77691BA
		graphqlClient: graphqlpkg.NewClient(
			"https://gateway.thegraph.com/api/{key}/subgraphs/id/3sThy2UsWd9X3D2M6MUQWzNUYrs8snMMhQKHSg9kUEAd",
		),
	}

	trebleSwapPool := entity.Pool{
		Address:  "0x256f399754f7ed5baa75b911ae6fd3c1a63b169c",
		Exchange: valueobject.ExchangeTrebleSwap,
		Type:     DexType,
		Tokens: []*entity.PoolToken{
			{Address: "0x4200000000000000000000000000000000000006", Decimals: 18, Swappable: true},
			{Address: "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913", Decimals: 6, Swappable: true},
		},
		Reserves: entity.PoolReserves{"0", "0"},
	}

	got, err := d.GetNewPoolState(context.Background(), trebleSwapPool, pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	t.Log(string(lo.Must(json.Marshal(got))))
}
