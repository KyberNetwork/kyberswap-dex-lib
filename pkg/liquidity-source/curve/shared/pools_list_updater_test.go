package shared

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
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

// `main` pools are classified with an on-chain multicall. Listed pools must not be classified again,
// and once every pool is listed a run must not call the RPC at all.
func TestPoolsListUpdater_GetNewPools_MainRegistrySkipsListedPools(t *testing.T) {
	const poolA, poolB = "0x000000000000000000000000000000000000a001", "0x000000000000000000000000000000000000b002"
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		var res GetPoolsResult
		res.Success, res.Data.PoolData = true, []CurvePool{
			{Address: poolA, Coins: []CurveCoin{{}, {}}}, {Address: poolB, Coins: []CurveCoin{{}, {}}}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	}))
	defer api.Close()

	getNewPools := func(listed ...string) ([]string, [][]common.Address) {
		// every call fails: no gamma() and no underlying coins, i.e. a plain pool
		rpc := testutil.NewMulticallStub(t, func(common.Address, []byte) (bool, []byte) { return false, nil })
		u := NewPoolsListUpdater(&Config{
			ChainCode:   "ethereum",
			ChainID:     valueobject.ChainIDEthereum,
			DataSources: []CurveDataSource{CURVE_DATASOURCE_MAIN},
			HTTPConfig:  HTTPConfig{BaseURL: api.URL, Timeout: durationjson.Duration{Duration: 5 * time.Second}},
		}, ethrpc.New(rpc.URL).SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")),
			logger.WithFields(logger.Fields{}))
		pools, _, err := u.GetNewPools(context.Background(), PoolListUpdaterMetadata{ListedPools: listed}.ToBytes(),
			mapset.NewSet(CURVE_POOL_TYPE_STABLE_PLAIN))
		require.NoError(t, err)
		return lo.Map(pools, func(p CurvePoolWithType, _ int) string { return p.Address }), rpc.Targets()
	}

	listed, calls := getNewPools(poolB)
	assert.Equal(t, []string{poolA}, listed)
	require.Len(t, calls, 1)
	assert.Equal(t, []common.Address{common.HexToAddress(poolA)}, lo.Uniq(calls[0]))

	listed, calls = getNewPools(poolA, poolB)
	assert.Empty(t, listed)
	assert.Empty(t, calls)
}
