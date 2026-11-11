package plain

import (
	"bytes"
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func stubPool(i int64) shared.CurvePool {
	addr := common.BigToAddress(big.NewInt(0x5100 + i)).Hex()
	return shared.CurvePool{
		Address:        addr,
		LpTokenAddress: addr,
		Coins:          []shared.CurveCoin{{Address: "0x00000000000000000000000000000000000000c0"}, {Address: "0x00000000000000000000000000000000000000c1"}},
	}
}

func poolAddr(p shared.CurvePool) string { return strings.ToLower(p.Address) }

// getNewPools runs GetNewPools against a Curve API serving apiPools and an RPC answering A()=100,
// A_precise()=10000 and oracle()=zero address, except that calls to reverting pools fail. It returns
// the listed pools, the pools probed by each eth_call, and the new metadata.
func getNewPools(t *testing.T, metadata []byte, minDuration time.Duration, apiPools []shared.CurvePool,
	reverting ...shared.CurvePool) ([]string, [][]string, []byte) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		var res shared.GetPoolsResult
		res.Success, res.Data.PoolData = true, apiPools
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	}))
	defer api.Close()
	rpc := testutil.NewMulticallStub(t, func(target common.Address, callData []byte) (bool, []byte) {
		if lo.ContainsBy(reverting, func(p shared.CurvePool) bool { return poolAddr(p) == strings.ToLower(target.Hex()) }) {
			return false, nil
		}
		v := new(big.Int)
		switch {
		case bytes.Equal(callData[:4], CurvePlainABI.Methods[poolMethodA].ID):
			v.SetInt64(100)
		case bytes.Equal(callData[:4], CurvePlainABI.Methods[poolMethodAPrecise].ID):
			v.SetInt64(10000)
		}
		return true, common.LeftPadBytes(v.Bytes(), 32)
	})

	u := NewPoolsListUpdater(&shared.Config{
		DexID:                 DexType,
		ChainCode:             "ethereum",
		ChainID:               valueobject.ChainIDEthereum,
		DataSources:           []shared.CurveDataSource{shared.CURVE_DATASOURCE_FACTORY},
		HTTPConfig:            shared.HTTPConfig{BaseURL: api.URL, Timeout: durationjson.Duration{Duration: 5 * time.Second}},
		FetchPoolsMinDuration: durationjson.Duration{Duration: minDuration},
	}, ethrpc.New(rpc.URL).SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")))
	pools, newMetadata, err := u.GetNewPools(context.Background(), metadata)
	require.NoError(t, err)

	probes := lo.Map(rpc.Targets(), func(targets []common.Address, _ int) []string {
		return lo.Uniq(lo.Map(targets, func(a common.Address, _ int) string { return strings.ToLower(a.Hex()) }))
	})
	return lo.Map(pools, func(p entity.Pool, _ int) string { return p.Address }), probes, newMetadata
}

// The Curve API returns every pool on every run. Each run must probe only pools not listed before,
// and must not call the RPC at all when there is nothing new. A pool whose probe fails is not
// recorded as listed, so it is retried on the next run.
func TestPoolsListUpdater_GetNewPools_ProbesOnlyUnlistedPools(t *testing.T) {
	p1, p2, p3, p4 := stubPool(1), stubPool(2), stubPool(3), stubPool(4)

	// first run probes every pool in one call; pool 3 reverts so it is not listed
	listed, probes, metadata := getNewPools(t, nil, 0, []shared.CurvePool{p1, p2, p3}, p3)
	assert.Equal(t, []string{poolAddr(p1), poolAddr(p2)}, listed)
	assert.Equal(t, [][]string{{poolAddr(p1), poolAddr(p2), poolAddr(p3)}}, probes)

	// within FetchPoolsMinDuration: no RPC call, and the listed pools are kept
	listed, probes, skipped := getNewPools(t, metadata, time.Hour, []shared.CurvePool{p1, p2, p3})
	assert.Empty(t, listed)
	assert.Empty(t, probes)
	assert.JSONEq(t, string(metadata), string(skipped))

	// pool 4 appears and pool 3 recovers: only those two are probed
	listed, probes, metadata = getNewPools(t, metadata, 0, []shared.CurvePool{p1, p2, p3, p4})
	assert.Equal(t, []string{poolAddr(p3), poolAddr(p4)}, listed)
	assert.Equal(t, [][]string{{poolAddr(p3), poolAddr(p4)}}, probes)

	// nothing new: no RPC call
	listed, probes, _ = getNewPools(t, metadata, 0, []shared.CurvePool{p1, p2, p3, p4})
	assert.Empty(t, listed)
	assert.Empty(t, probes)
}
