package ambient_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ambient"
	"github.com/stretchr/testify/require"
)

func TestPoolListUpdater(t *testing.T) {
	t.Skip()

	pu := ambient.NewPoolsListUpdater(ambient.Config{
		DexID:                  "ambient",
		SubgraphURL:            "https://api.studio.thegraph.com/query/47610/croc-mainnet/version/latest",
		SubgraphRequestTimeout: durationjson.Duration{Duration: time.Second * 10},
	})

	t.Run("nil metadataBytes, default config", func(t *testing.T) {
		pools, meta, err := pu.GetNewPools(context.Background(), nil)
		require.NoError(t, err)

		t.Logf("%s\n", string(meta))
		for i := range pools {
			t.Logf("%+v\n", pools[i])
		}
	})

	t.Run("with metadataBytes, specified lastCreateTime", func(t *testing.T) {
		metadataBytes, _ := json.Marshal(ambient.PoolListUpdaterMetadata{LastCreateTime: 1697392211})

		pools, meta, err := pu.GetNewPools(context.Background(), metadataBytes)
		require.NoError(t, err)

		t.Logf("%s\n", string(meta))
		for i := range pools {
			t.Logf("%+v\n", pools[i])
		}
	})
}
