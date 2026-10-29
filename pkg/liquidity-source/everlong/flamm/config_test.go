package everlongflamm

import (
	"testing"

	"github.com/stretchr/testify/require"

	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// TestConfigThroughFactories decodes Config the way pool-service does: through the registered lister and tracker
// factories, which inject properties["DexID"] and decode with pool.PropertiesToStruct. Every documented key must
// land, in the exact spelling of its tag and in the Go field-name spelling the README's table uses.
func TestConfigThroughFactories(t *testing.T) {
	t.Parallel()
	factory := c104.Factory.Hex()
	for _, c := range []struct {
		name  string
		props map[string]any
		want  Config
	}{
		{"minimal", map[string]any{"chainID": 8453, "factory": factory},
			Config{DexID: DexType, ChainID: valueobject.ChainIDBase, Factory: factory}},
		{"tag-spelling", map[string]any{"chainID": 8453, "factory": factory, "leverRouting": true,
			"leverMinEdgeBps": 25, "priceBandMarginBps": 35, "maxSnapshotAgeSec": 120, "quoteDonatedVenues": true,
			"gas": map[string]any{"swapSell": 1_400_000, "leverDown": 4_200_000}},
			Config{DexID: DexType, ChainID: valueobject.ChainIDBase, Factory: factory, LeverRouting: true,
				LeverMinEdgeBps: new(uint64(25)), PriceBandMarginBps: new(uint64(35)), MaxSnapshotAgeSec: new(uint64(120)),
				QuoteDonatedVenues: true, Gas: GasConfig{SwapSell: 1_400_000, LeverDown: 4_200_000}}},
		{"field-name-spelling", map[string]any{"ChainID": 8453, "Factory": factory, "LeverRouting": true,
			"LeverMinEdgeBps": 0, "PriceBandMarginBps": 0, "PriceAgeMarginSec": 90, "SpreadAgeMarginSec": 15,
			"DebtDriftSec": 0, "MaxSnapshotAgeSec": 300, "Pools": []string{c104.Pool.Hex()},
			"Gas": map[string]any{"SwapSellPass": 7}},
			Config{DexID: DexType, ChainID: valueobject.ChainIDBase, Factory: factory, LeverRouting: true,
				LeverMinEdgeBps: new(uint64(0)), PriceBandMarginBps: new(uint64(0)), PriceAgeMarginSec: new(uint64(90)),
				SpreadAgeMarginSec: new(uint64(15)), DebtDriftSec: new(uint64(0)), MaxSnapshotAgeSec: new(uint64(300)),
				Pools: []string{c104.Pool.Hex()}, Gas: GasConfig{SwapSellPass: 7}}},
		{"chainid-lowercase-d", map[string]any{"chainId": 8453, "factory": factory},
			Config{DexID: DexType, ChainID: valueobject.ChainIDBase, Factory: factory}},
		// A configuration that spells the dex id itself does not displace the one the factory injects.
		{"dexid-spelled", map[string]any{"chainID": 8453, "factory": factory, "dexId": DexType},
			Config{DexID: DexType, ChainID: valueobject.ChainIDBase, Factory: factory}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			lister, err := poollist.Factory(DexType)(DexType, poollist.FactoryParams{Exchange: DexType,
				Properties: c.props})
			require.NoError(t, err)
			require.Equal(t, c.want, *lister.(*PoolsListUpdater).config)

			tracker, err := pooltrack.Factory(DexType)(DexType, pooltrack.FactoryParams{Exchange: DexType,
				Properties: c.props})
			require.NoError(t, err)
			require.Equal(t, c.want, *tracker.(*PoolTracker).config)
		})
	}
}
