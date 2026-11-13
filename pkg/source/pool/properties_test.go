package pool_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	fluiddexv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/fluid/dex-v2"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/liquiditybookv21"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// The list, tracker and pool factories set "DexID", and the chains' configs spell "chainId", in
// configs that tag them dexID and chainID. Each lands in the configs the lens settings grow.
func TestPropertiesToStructMatchesAnyCase(t *testing.T) {
	properties := map[string]any{"DexID": "uniswap-v4", "chainId": 1, "useLens": true}

	var v4 uniswapv4.Config
	require.NoError(t, pool.PropertiesToStruct(properties, &v4))
	require.Equal(t, "uniswap-v4", v4.DexID)
	require.Equal(t, valueobject.ChainIDEthereum, v4.ChainID)
	require.True(t, v4.UseLens)

	var fluid fluiddexv2.Config
	require.NoError(t, pool.PropertiesToStruct(properties, &fluid))
	require.Equal(t, valueobject.ChainIDEthereum, fluid.ChainID)

	var lb liquiditybookv21.Config
	require.NoError(t, pool.PropertiesToStruct(map[string]any{"DexID": "traderjoe-v21"}, &lb))
	require.Equal(t, "traderjoe-v21", lb.DexID)
}
