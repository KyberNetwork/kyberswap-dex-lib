package stableng

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const tryAggregateABI = `[{"name":"tryAggregate","type":"function","stateMutability":"nonpayable",
"inputs":[{"name":"requireSuccess","type":"bool"},{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],
"outputs":[{"name":"returnData","type":"tuple[]","components":[{"name":"success","type":"bool"},{"name":"returnData","type":"bytes"}]}]}]`

// listingStub serves the Curve API pool list and answers every eth_call (a tryAggregate of
// A/A_precise) with A=100 and A_precise=10000, except that calls to reverting pools fail.
type listingStub struct {
	mu        sync.Mutex
	pools     []shared.CurvePool
	reverting map[string]bool
	probes    [][]string // pools probed by each eth_call
}

func (s *listingStub) setPools(pools []shared.CurvePool, reverting map[string]bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pools, s.reverting = pools, reverting
}

func (s *listingStub) getProbes() [][]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([][]string(nil), s.probes...)
}

func (s *listingStub) curveAPI() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		var res shared.GetPoolsResult
		res.Success, res.Data.PoolData = true, s.pools
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	}))
}

func (s *listingStub) rpc(t *testing.T) *httptest.Server {
	multicallABI, err := abi.JSON(strings.NewReader(tryAggregateABI))
	require.NoError(t, err)
	tryAggregate := multicallABI.Methods["tryAggregate"]
	selectorA := CurveStableNGABI.Methods[poolMethodA].ID

	type call struct {
		Target   common.Address
		CallData []byte
	}
	type result struct {
		Success    bool
		ReturnData []byte
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     json.RawMessage   `json:"id"`
			Params []json.RawMessage `json:"params"`
		}
		var arg struct {
			Input hexutil.Bytes `json:"input"`
		}
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&req)) || !assert.NoError(t, json.Unmarshal(req.Params[0], &arg)) {
			return
		}
		args, err := tryAggregate.Inputs.Unpack(arg.Input[4:])
		if !assert.NoError(t, err) {
			return
		}
		calls := *abi.ConvertType(args[1], new([]call)).(*[]call)

		s.mu.Lock()
		defer s.mu.Unlock()
		results := make([]result, len(calls))
		var probed []string
		for i, c := range calls {
			pool := strings.ToLower(c.Target.Hex())
			if !lo.Contains(probed, pool) {
				probed = append(probed, pool)
			}
			if s.reverting[pool] {
				continue
			}
			v := big.NewInt(10000)
			if bytes.Equal(c.CallData[:4], selectorA) {
				v = big.NewInt(100)
			}
			results[i] = result{Success: true, ReturnData: common.LeftPadBytes(v.Bytes(), 32)}
		}
		s.probes = append(s.probes, probed)

		ret, err := tryAggregate.Outputs.Pack(results)
		if !assert.NoError(t, err) {
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": hexutil.Encode(ret)})
	}))
}

func stubPool(i int) shared.CurvePool {
	addr := common.BigToAddress(big.NewInt(int64(0x5100 + i))).Hex()
	return shared.CurvePool{
		Id:             fmt.Sprintf("factory-stable-ng-%d", i),
		Address:        addr,
		LpTokenAddress: addr,
		Coins: []shared.CurveCoin{
			{Address: common.BigToAddress(big.NewInt(0xc0)).Hex()},
			{Address: common.BigToAddress(big.NewInt(0xc1)).Hex()},
		},
	}
}

func poolAddr(i int) string { return strings.ToLower(stubPool(i).Address) }

// The Curve API returns every pool on every run. Each run must probe only pools not listed
// before, and must not call the RPC at all when there is nothing new to probe. A pool whose
// probe fails is not recorded as listed, so it is retried on the next run.
func TestPoolsListUpdater_GetNewPools_ProbesOnlyUnlistedPools(t *testing.T) {
	stub := &listingStub{}
	api, rpc := stub.curveAPI(), stub.rpc(t)
	defer api.Close()
	defer rpc.Close()

	run := func(metadata []byte, minDuration time.Duration) ([]string, []byte) {
		u := NewPoolsListUpdater(&shared.Config{
			DexID:                 DexType,
			ChainCode:             "ethereum",
			ChainID:               valueobject.ChainIDEthereum,
			DataSources:           []shared.CurveDataSource{shared.CURVE_DATASOURCE_FACTORY_STABLE_NG},
			HTTPConfig:            shared.HTTPConfig{BaseURL: api.URL, Timeout: durationjson.Duration{Duration: 5 * time.Second}},
			FetchPoolsMinDuration: durationjson.Duration{Duration: minDuration},
		}, ethrpc.New(rpc.URL).SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")))
		pools, newMetadata, err := u.GetNewPools(context.Background(), metadata)
		require.NoError(t, err)
		return lo.Map(pools, func(p entity.Pool, _ int) string { return p.Address }), newMetadata
	}

	// first run probes every pool in one call; pool 3 reverts so it is not listed
	stub.setPools([]shared.CurvePool{stubPool(1), stubPool(2), stubPool(3)}, map[string]bool{poolAddr(3): true})
	listed, metadata := run(nil, 0)
	assert.Equal(t, []string{poolAddr(1), poolAddr(2)}, listed)
	assert.Equal(t, [][]string{{poolAddr(1), poolAddr(2), poolAddr(3)}}, stub.getProbes())

	// within FetchPoolsMinDuration: no RPC call, and the listed pools are kept
	listed, skippedMetadata := run(metadata, time.Hour)
	assert.Empty(t, listed)
	assert.Len(t, stub.getProbes(), 1)
	assert.JSONEq(t, string(metadata), string(skippedMetadata))

	// pool 4 appears and pool 3 recovers: only those two are probed
	stub.setPools([]shared.CurvePool{stubPool(1), stubPool(2), stubPool(3), stubPool(4)}, nil)
	listed, metadata = run(metadata, 0)
	assert.Equal(t, []string{poolAddr(3), poolAddr(4)}, listed)
	assert.Equal(t, []string{poolAddr(3), poolAddr(4)}, stub.getProbes()[1])

	// nothing new: no RPC call
	listed, _ = run(metadata, 0)
	assert.Empty(t, listed)
	assert.Len(t, stub.getProbes(), 2)
}
