package ekubov3

import (
	"context"
	"slices"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/pools"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestPoolListUpdater(t *testing.T) {
	t.Parallel()
	test.SkipCI(t)

	plUpdater := NewPoolListUpdater(MainnetConfig, ethrpc.New("https://ethereum.drpc.org").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")),
		graphql.NewClient(MainnetConfig.SubgraphAPI))

	newPools, _, err := plUpdater.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Greater(t, len(newPools), 0)

	expected := []pools.AnyPoolKey{
		anyPoolKey(
			valueobject.ZeroAddress,
			"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			common.Address{}.Hex(),
			9223372036854775,
			pools.NewConcentratedPoolTypeConfig(1000),
		),
		// A stableswap `BoostedFees` pool, tracked through automatic support for extensions without `beforeSwap` and `afterSwap` call point
		anyPoolKey(
			"0x6440f144b7e50d6a8439336510312d2f54beb01d",
			"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			"0x948b9c2c99718034954110cb61a6e08e107745f9",
			3689348814741910,
			pools.NewStableswapPoolTypeConfig(-27631040, 14),
		),
	}

	filteredOut := []pools.AnyPoolKey{
		// An old `BoostedFees` extension
		anyPoolKey(
			valueobject.ZeroAddress,
			"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			"0xd48eb64c9c58cb3317f44551e80acc67b9f8ccae",
			9223372036854775,
			pools.NewConcentratedPoolTypeConfig(1000),
		),
	}

	containsPoolKey := func(testPk pools.AnyPoolKey) bool {
		return slices.ContainsFunc(newPools, func(el entity.Pool) bool {
			var staticExtra StaticExtra
			err := json.Unmarshal([]byte(el.StaticExtra), &staticExtra)
			require.NoError(t, err)

			pk := staticExtra.PoolKey

			return pk.Token0.Cmp(testPk.Token0) == 0 && pk.Token1.Cmp(testPk.Token1) == 0 &&
				pk.Config.Compressed() == testPk.Config.Compressed()
		})
	}

	for _, testPk := range expected {
		require.True(t, containsPoolKey(testPk), "missing pool key: %v", testPk)
	}

	for _, testPk := range filteredOut {
		require.False(t, containsPoolKey(testPk), "unexpected filtered pool key returned: %v", testPk)
	}
}
