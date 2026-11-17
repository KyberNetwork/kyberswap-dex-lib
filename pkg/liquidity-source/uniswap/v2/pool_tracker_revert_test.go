package uniswapv2

import (
	"context"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

const tryBlockAndAggregateABI = `[{"name":"tryBlockAndAggregate","type":"function","stateMutability":"nonpayable",
"inputs":[{"name":"requireSuccess","type":"bool"},{"name":"calls","type":"tuple[]","components":[
{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],
"outputs":[{"name":"blockNumber","type":"uint256"},{"name":"blockHash","type":"bytes32"},
{"name":"returnData","type":"tuple[]","components":[{"name":"success","type":"bool"},{"name":"returnData","type":"bytes"}]}]}]`

// multicallResult is one call's (success, returnData) in a tryBlockAndAggregate answer.
type multicallResult struct {
	Success    bool
	ReturnData []byte
}

// multicallServer answers every eth_call with a tryBlockAndAggregate result at block 12 carrying results, the
// calls' answers in order (a failed call has success false, as a reverting call gives; a call to an address with
// no code succeeds with no return data).
func multicallServer(t *testing.T, results []multicallResult) *httptest.Server {
	t.Helper()
	mc, err := gethabi.JSON(strings.NewReader(tryBlockAndAggregateABI))
	require.NoError(t, err)
	out, err := mc.Methods["tryBlockAndAggregate"].Outputs.Pack(big.NewInt(12), common.Hash{}, results)
	require.NoError(t, err)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		if err := json.Unmarshal(body, &req); err != nil || req.Method != "eth_call" {
			http.Error(w, "only eth_call", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": hexutil.Encode(out)})
	}))
}

func reservesResult(t *testing.T, r0, r1 int64) multicallResult {
	t.Helper()
	data, err := uniswapV2PairABI.Methods[pairMethodGetReserves].Outputs.Pack(big.NewInt(r0), big.NewInt(r1), uint32(300))
	require.NoError(t, err)
	return multicallResult{Success: true, ReturnData: data}
}

func balanceResult(t *testing.T, bal int64) multicallResult {
	t.Helper()
	data, err := abi.Erc20ABI.Methods[abi.Erc20BalanceOfMethod].Outputs.Pack(big.NewInt(bal))
	require.NoError(t, err)
	return multicallResult{Success: true, ReturnData: data}
}

var (
	reverted = multicallResult{Success: false}
	// noData is a balanceOf of a token with no code (one that self-destructed or was never deployed there): the
	// call succeeds and returns nothing, so it does not decode either.
	noData = multicallResult{Success: true}
)

// trackOnce runs GetNewPoolState without logs (reserves and balances from the multicall) against results.
func trackOnce(t *testing.T, results []multicallResult) (entity.Pool, error) {
	t.Helper()
	srv := multicallServer(t, results)
	defer srv.Close()
	tracker := &PoolTracker{
		config: &Config{FactoryAddress: "0x5c69bee701ef814a2b6a3edd4b1652cb9cc5aa6f", Fee: 30,
			FeePrecision: 10000}, // ChainID unset: no tax calls, the request holds the three reserve calls only
		ethrpcClient: ethrpc.New(srv.URL).SetMulticallContract(common.HexToAddress("0xca11bde05977b3631167028862be2a173976ca11")),
		logDecoder:   NewLogDecoder(),
	}
	p := entity.Pool{
		Address: "0x0000000000000000000000000000000000000001",
		Tokens: []*entity.PoolToken{{Address: "0x1111111111111111111111111111111111111111"},
			{Address: "0x2222222222222222222222222222222222222222"}},
		Reserves:    entity.PoolReserves{"900", "1900"},
		Extra:       `{"fee":30,"feePrecision":10000}`,
		BlockNumber: 10,
	}
	var (
		got  entity.Pool
		err  error
		pnc  any
		done = make(chan struct{})
	)
	go func() {
		defer close(done)
		defer func() { pnc = recover() }()
		got, err = tracker.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
	}()
	<-done
	require.Nil(t, pnc, "GetNewPoolState panicked")
	return got, err
}

// The healthy pair: reserves within the balances are kept.
func TestPoolTracker_ReservesWithinBalances(t *testing.T) {
	got, err := trackOnce(t, []multicallResult{reservesResult(t, 1000, 2000), balanceResult(t, 1000),
		balanceResult(t, 2500)})
	require.NoError(t, err)
	assert.Equal(t, entity.PoolReserves{"1000", "2000"}, got.Reserves)
	assert.Equal(t, uint64(12), got.BlockNumber)
}

// A token whose balanceOf does not decode (it reverts, or the token has no code and returns nothing, as token0
// 0x06cd2a24a126c3dc0fbe8798d43049c8b03a2a0c of pair 0x1628c593040f734838bb482d2579cc1d0da35c93 on Ethereum does)
// must not panic the tracker. The pair's balance cannot be checked, so the pool is marked unusable the way a short
// balance is: zero reserves.
func TestPoolTracker_UndecodedBalanceMarksThePoolEmpty(t *testing.T) {
	for name, results := range map[string][]multicallResult{
		"token0 reverts": {reservesResult(t, 1000, 2000), reverted, balanceResult(t, 2500)},
		"token1 reverts": {reservesResult(t, 1000, 2000), balanceResult(t, 1000), reverted},
		"both revert":    {reservesResult(t, 1000, 2000), reverted, reverted},
		"token0 no code": {reservesResult(t, 1000, 2000), noData, balanceResult(t, 2500)},
		"no code, empty": {reservesResult(t, 0, 0), noData, balanceResult(t, 0)},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := trackOnce(t, results)
			require.NoError(t, err)
			assert.Equal(t, entity.PoolReserves{"0", "0"}, got.Reserves)
			assert.Equal(t, uint64(12), got.BlockNumber)
		})
	}
}

// A pair whose getReserves reverts must not panic either. It is marked empty too: failing the task would only
// retry the same revert until the task is archived.
func TestPoolTracker_GetReservesRevertsMarksThePoolEmpty(t *testing.T) {
	got, err := trackOnce(t, []multicallResult{reverted, balanceResult(t, 1000), balanceResult(t, 2500)})
	require.NoError(t, err)
	assert.Equal(t, entity.PoolReserves{"0", "0"}, got.Reserves)
	assert.Equal(t, uint64(12), got.BlockNumber)
}
