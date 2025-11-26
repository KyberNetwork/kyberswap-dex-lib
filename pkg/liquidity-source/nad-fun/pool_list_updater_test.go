package nadfun

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestPoolListUpdater(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	plUpdater := NewPoolsListUpdater(&Config{
		DexID:               DexType,
		BondingCurveAddress: "0xa7283d07812a02afb7c09b60f8896bcea3f90ace",
		NewPoolLimit:        3000,
		ChainID:             valueobject.ChainIDMonad,
	}, ethrpc.New("https://rpc-mainnet.monadinfra.com/rpc/ICLJSp4IKDWLSpZ4laJATUQfL0ucwxiK").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")),
		graphqlpkg.NewClient("https://api.goldsky.com/api/public/project_cmcu5mf7qh3lx01ww9j049ln3/subgraphs/nadfun-bc-monad/latest/gn"))

	newPools, _, err := plUpdater.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Greater(t, len(newPools), 0)

	tracker, err := NewPoolTracker(plUpdater.config, plUpdater.ethrpcClient)
	require.NoError(t, err)

	for _, p := range newPools {
		if p.Tokens[1].Address == "0x193836f0701ac6efbb409991645aee846d2e7777" {
			log.Fatalln()
		}
		p, err = tracker.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
		require.NotNil(t, p)
	}
}
