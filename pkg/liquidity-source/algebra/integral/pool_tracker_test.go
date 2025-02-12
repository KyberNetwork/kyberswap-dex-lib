package integral

import (
	"context"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/algebra"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestPoolTracker_GetNewPoolState(t *testing.T) {
	// t.Skip()

	d := &PoolTracker{
		PoolTracker: algebra.PoolTracker[Timepoint, TimepointRPC]{
			EthrpcClient: ethrpc.NewWithClient(lo.Must(ethclient.Dial("https://bsc.kyberengineering.io"))).
				SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")),
		},
		config: &Config{
			DexID:              string(valueobject.ExchangeThenaFusionV3),
			AllowSubgraphError: true,
			UseBasePluginV2:    true,
		},
		graphqlClient: graphqlpkg.NewClient("https://thegraph.com/explorer/api/playground/QmWSzHwZY9ZMNYMVbQLyL276V1toR3iZsnYMfQut166yit"),
	}
	got, err := d.GetNewPoolState(context.Background(), thenaEp, pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	t.Log(string(lo.Must(json.Marshal(got))))
}
