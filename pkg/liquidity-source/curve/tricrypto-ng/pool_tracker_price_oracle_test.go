package tricryptong_test

import (
	"context"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/shared"
	tricryptong "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/tricrypto-ng"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/tricrypto-ng/lazy"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const tryBlockAndAggregateABI = `[{"name":"tryBlockAndAggregate","type":"function","stateMutability":"nonpayable",
"inputs":[{"name":"requireSuccess","type":"bool"},{"name":"calls","type":"tuple[]","components":[
{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],
"outputs":[{"name":"blockNumber","type":"uint256"},{"name":"blockHash","type":"bytes32"},
{"name":"returnData","type":"tuple[]","components":[{"name":"success","type":"bool"},{"name":"returnData","type":"bytes"}]}]}]`

type multicallResult struct {
	Success    bool
	ReturnData []byte
}

// poolServer answers tryBlockAndAggregate like a tricrypto-ng pool at block 12: each call's answer comes
// from its method name, every view returns value(method), and the methods in reverts revert.
func poolServer(t *testing.T, value func(method string) *big.Int, reverts ...string) *httptest.Server {
	t.Helper()
	mc, err := gethabi.JSON(strings.NewReader(tryBlockAndAggregateABI))
	require.NoError(t, err)
	method := mc.Methods["tryBlockAndAggregate"]
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID     json.RawMessage   `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		require.NoError(t, json.Unmarshal(body, &req))
		require.Equal(t, "eth_call", req.Method)
		var msg struct {
			Data  hexutil.Bytes `json:"data"`
			Input hexutil.Bytes `json:"input"`
		}
		require.NoError(t, json.Unmarshal(req.Params[0], &msg))
		data := msg.Data
		if len(data) == 0 {
			data = msg.Input
		}
		args, err := method.Inputs.Unpack(data[4:])
		require.NoError(t, err)
		calls := args[1].([]struct {
			Target   common.Address `json:"target"`
			CallData []byte         `json:"callData"`
		})
		results := make([]multicallResult, len(calls))
		for i, c := range calls {
			m, err := tricryptong.CurveTricryptoNGABI.MethodById(c.CallData[:4])
			require.NoError(t, err)
			if slices.Contains(reverts, m.RawName) {
				continue
			}
			out, err := m.Outputs.Pack(value(m.RawName))
			require.NoError(t, err)
			results[i] = multicallResult{Success: true, ReturnData: out}
		}
		out, err := method.Outputs.Pack(big.NewInt(12), common.Hash{}, results)
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": hexutil.Encode(out)})
	}))
}

// packedRebalancingParams is [allowed_extra_profit, adjustment_step, ma_time], 64 bits each, as the
// pool stores them: 0x2889302a holds 1e8, 1e11 and 866, and its ma_time() view says 601.
func packedRebalancingParams(allowedExtraProfit, adjustmentStep, maTime int64) *big.Int {
	packed := new(big.Int).Lsh(big.NewInt(allowedExtraProfit), 128)
	packed.Or(packed, new(big.Int).Lsh(big.NewInt(adjustmentStep), 64))
	return packed.Or(packed, big.NewInt(maTime))
}

func viewValues(packed *big.Int) func(string) *big.Int {
	return func(method string) *big.Int {
		switch method {
		case "packed_rebalancing_params":
			return packed
		case "ma_time":
			return big.NewInt(601)
		case "initial_A_gamma", "future_A_gamma":
			return new(big.Int).Or(new(big.Int).Lsh(big.NewInt(540000), 128), big.NewInt(80500000000000))
		case "initial_A_gamma_time", "future_A_gamma_time":
			return big.NewInt(1705051559)
		default:
			return big.NewInt(1e18)
		}
	}
}

type tracker interface {
	GetNewPoolState(context.Context, entity.Pool, pool.GetNewPoolStateParams) (entity.Pool, error)
}

func trackers(t *testing.T, url string) map[string]tracker {
	t.Helper()
	client := ethrpc.New(url).SetMulticallContract(common.HexToAddress("0xca11bde05977b3631167028862be2a173976ca11"))
	cfg := &shared.Config{DexID: tricryptong.DexType}
	main, err := tricryptong.NewPoolTracker(cfg, client)
	require.NoError(t, err)
	lz, err := lazy.NewPoolTracker(cfg, client)
	require.NoError(t, err)
	return map[string]tracker{"tracker": main, "lazy tracker": lz}
}

// track runs tr once and returns the Extra it stored, the pool, and the time window it ran in.
func track(t *testing.T, tr tracker) (tricryptong.Extra, entity.Pool, int64, int64) {
	t.Helper()
	p := entity.Pool{
		Address: "0x2889302a794da87fbf1d6db415c1492194663d13",
		Tokens: []*entity.PoolToken{{Address: "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e"},
			{Address: "0x18084fba666a33d37592fa2633fd49a74dd93a88"},
			{Address: "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0"}},
		Reserves: entity.PoolReserves{"0", "0", "0"},
		Extra:    "{}",
	}
	before := time.Now().Unix()
	got, err := tr.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
	after := time.Now().Unix()
	require.NoError(t, err)
	var extra tricryptong.Extra
	require.NoError(t, json.Unmarshal([]byte(got.Extra), &extra))
	return extra, got, before, after
}

// The trackers store the EMA's time constant from packed_rebalancing_params (not the ma_time() view)
// and the time they read price_oracle, which is also the pool's timestamp.
func TestPoolTrackers_StorePriceOracleReadTimeAndMaTime(t *testing.T) {
	srv := poolServer(t, viewValues(packedRebalancingParams(1e8, 1e11, 866)))
	defer srv.Close()
	for name, tr := range trackers(t, srv.URL) {
		t.Run(name, func(t *testing.T) {
			extra, got, before, after := track(t, tr)
			require.NotNil(t, extra.MaTime)
			assert.Equal(t, uint64(866), extra.MaTime.Uint64())
			assert.GreaterOrEqual(t, extra.PriceOracleTimestamp, before)
			assert.LessOrEqual(t, extra.PriceOracleTimestamp, after)
			assert.Equal(t, got.Timestamp, extra.PriceOracleTimestamp)
			assert.Equal(t, uint64(12), got.BlockNumber)
			assert.Equal(t, uint64(1e18), extra.AllowedExtraProfit.Uint64(), "allowed_extra_profit still comes from its own view")
		})
	}
}

// A pool without packed_rebalancing_params, or with ma_time 0 in it, leaves both fields out, so the
// simulator uses the stored price_oracle as it did before (TestPriceOracleNotCarriedForward_OldState).
func TestPoolTrackers_NoMaTimeLeavesTheFieldsOut(t *testing.T) {
	for name, value := range map[string]struct {
		values  func(string) *big.Int
		reverts []string
	}{
		"packed_rebalancing_params reverts": {viewValues(nil), []string{"packed_rebalancing_params"}},
		"ma_time 0":                         {viewValues(packedRebalancingParams(1e8, 1e11, 0)), nil},
	} {
		srv := poolServer(t, value.values, value.reverts...)
		for trName, tr := range trackers(t, srv.URL) {
			t.Run(name+"/"+trName, func(t *testing.T) {
				extra, got, _, _ := track(t, tr)
				assert.Nil(t, extra.MaTime)
				assert.Zero(t, extra.PriceOracleTimestamp)
				assert.NotContains(t, got.Extra, "MaTime")
				assert.NotContains(t, got.Extra, "PriceOracleTimestamp")
				assert.Equal(t, uint64(12), got.BlockNumber)
			})
		}
		srv.Close()
	}
}
