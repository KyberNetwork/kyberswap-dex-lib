package uniswapv4

import (
	"context"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

func TestPoolTracker_GetNewPoolState(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip()
	}

	pt := &PoolTracker{
		config:        &Config{DexID: DexType, StateViewAddress: "0x7fFE42C4a5DEeA5b0feC41C94C136Cf115597227"},
		ethrpcClient:  ethrpc.New("https://ethereum.kyberengineering.io").SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")),
		graphqlClient: graphqlpkg.NewClient(os.ExpandEnv("https://gateway.thegraph.com/api/$THEGRAPH_API_KEY/subgraphs/id/DiYPVdygkfjDWhbxGSqAQxwBKmfKnkWQojqeM2rkLb3G")),
	}
	got, err := pt.GetNewPoolState(context.Background(),
		entity.Pool{Address: "0x6b77c5119ea25b4b46ec79166075eed433bf8ad4bfe907490bb06305e3c0012a",
			StaticExtra: `{"tS":200}`},
		pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	t.Log(got)
}
