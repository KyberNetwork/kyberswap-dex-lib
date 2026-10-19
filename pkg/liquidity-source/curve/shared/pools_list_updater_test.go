package shared

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
	"github.com/KyberNetwork/logger"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// GetNewPools must skip pools recorded by AddListedPools, while a lister that never records
// pools (plain, tricrypto-ng, twocrypto-ng) keeps getting every pool on every run.
func TestPoolsListUpdater_GetNewPools_SkipsListedPools(t *testing.T) {
	const poolA, poolB = "0x000000000000000000000000000000000000a001", "0x000000000000000000000000000000000000b002"
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		var res GetPoolsResult
		res.Success, res.Data.PoolData = true, []CurvePool{{Address: poolA}, {Address: poolB}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	}))
	defer api.Close()

	u := NewPoolsListUpdater(&Config{
		ChainCode:   "ethereum",
		ChainID:     valueobject.ChainIDEthereum,
		DataSources: []CurveDataSource{CURVE_DATASOURCE_FACTORY_STABLE_NG},
		HTTPConfig:  HTTPConfig{BaseURL: api.URL, Timeout: durationjson.Duration{Duration: 5 * time.Second}},
	}, nil, logger.WithFields(logger.Fields{}))
	getNewPools := func(metadata []byte) ([]string, []byte) {
		pools, newMetadata, err := u.GetNewPools(context.Background(), metadata,
			mapset.NewSet(CURVE_POOL_TYPE_STABLE_NG_PLAIN))
		require.NoError(t, err)
		return lo.Map(pools, func(p CurvePoolWithType, _ int) string { return p.Address }), newMetadata
	}

	// metadata written before ListedPools existed, and runs that record nothing: every pool, every run
	listed, metadata := getNewPools([]byte(`{"LastRun":"2026-01-01T00:00:00Z"}`))
	assert.Equal(t, []string{poolA, poolB}, listed)
	listed, metadata = getNewPools(metadata)
	assert.Equal(t, []string{poolA, poolB}, listed)

	// once poolA is recorded it is skipped, and the record is kept across runs
	metadata, err := AddListedPools(metadata, []entity.Pool{{Address: poolA}})
	require.NoError(t, err)
	listed, metadata = getNewPools(metadata)
	assert.Equal(t, []string{poolB}, listed)
	var m PoolListUpdaterMetadata
	require.NoError(t, json.Unmarshal(metadata, &m))
	assert.Equal(t, []string{poolA}, m.ListedPools)
}
