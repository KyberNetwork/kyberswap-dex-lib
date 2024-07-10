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
	var (
		metadataBytes, _ = json.Marshal(ambient.PoolListUpdaterMetadata{LastCreateTime: 0})
	)

	pu := ambient.NewPoolsListUpdater(ambient.Config{
		DexID:                  "ambient",
		SubgraphURL:            "https://api.studio.thegraph.com/query/47610/croc-mainnet/version/latest",
		SubgraphRequestTimeout: durationjson.Duration{Duration: time.Second * 10},
	})
	pools, meta, err := pu.GetNewPools(context.Background(), metadataBytes)
	require.NoError(t, err)

	t.Logf("%s\n", string(meta))
	for i := range pools {
		t.Logf("%+v\n", pools[i])
	}
}
