package hyperamm

import (
	"context"
	"math/big"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	valantisstex "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/valantis-stex"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const (
	testBlock     = 48709733
	testFairPrice = "80000000000000000000000"
)

// errStalePrice is Pyth's StalePrice() selector, which the fee module bubbles up once the oracle
// update is older than its max age.
var errStalePrice = hexutil.MustDecode("0x19abf40e")

var testMulticallABI = func() abi.ABI {
	const calls = `{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}`
	a, err := abi.JSON(strings.NewReader(`[
		{"type":"function","name":"aggregate","inputs":[` + calls + `],
		 "outputs":[{"name":"blockNumber","type":"uint256"},{"name":"returnData","type":"bytes[]"}]},
		{"type":"function","name":"tryBlockAndAggregate","inputs":[{"name":"requireSuccess","type":"bool"},` + calls + `],
		 "outputs":[{"name":"blockNumber","type":"uint256"},{"name":"blockHash","type":"bytes32"},
		  {"name":"returnData","type":"tuple[]","components":[{"name":"success","type":"bool"},{"name":"returnData","type":"bytes"}]}]}]`))
	if err != nil {
		panic(err)
	}
	return a
}()

type multicallCall struct {
	Target   common.Address
	CallData []byte
}

type multicallResult struct {
	Success    bool
	ReturnData []byte
}

// fakeHyperAMMRPC serves the tracker's multicalls for one pool, reverting the named methods with
// StalePrice. Like Multicall3, aggregate reverts as a whole while tryBlockAndAggregate reports
// each call's success.
func fakeHyperAMMRPC(t *testing.T, reverts ...string) (*ethrpc.Client, *atomic.Int32) {
	answer := func(data []byte) (bool, []byte) {
		for _, a := range []abi.ABI{hyperAMMABI, hyperAMMSwapFeeModuleABI, hyperAMMLensABI} {
			m, err := a.MethodById(data[:4])
			if err != nil {
				continue
			} else if slices.Contains(reverts, m.Name) {
				return false, errStalePrice
			}
			var out []byte
			switch m.Name {
			case "paused":
				out, err = m.Outputs.Pack(false)
			case "fairPriceInScale18":
				out, err = m.Outputs.Pack(bignumber.NewBig(testFairPrice))
			case "baseFeeBps":
				out, err = m.Outputs.Pack(uint16(6))
			case "getAvailableLiquidity":
				out, err = m.Outputs.Pack(big.NewInt(30218182), big.NewInt(76836))
			case "getSwapFeeInBips":
				out, err = m.Outputs.Pack(valantisstex.SwapFeeModuleData{FeeInBips: big.NewInt(6), InternalContext: []byte{}})
			}
			if err != nil {
				t.Errorf("pack %s: %v", m.Name, err)
			}
			return true, out
		}
		t.Errorf("unexpected call %x", data[:4])
		return false, nil
	}

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var req struct {
			ID     json.RawMessage   `json:"id"`
			Params []json.RawMessage `json:"params"`
		}
		var msg struct {
			Input hexutil.Bytes `json:"input"`
			Data  hexutil.Bytes `json:"data"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Params) == 0 ||
			json.Unmarshal(req.Params[0], &msg) != nil {
			t.Errorf("bad eth_call request: %v", err)
			return
		}
		data := msg.Input
		if len(data) == 0 {
			data = msg.Data
		}

		m, err := testMulticallABI.MethodById(data[:4])
		if err != nil {
			t.Errorf("unexpected multicall method %x", data[:4])
			return
		}
		args, err := m.Inputs.Unpack(data[4:])
		if err != nil {
			t.Errorf("unpack %s: %v", m.Name, err)
			return
		}
		mcCalls := *abi.ConvertType(args[len(args)-1], new([]multicallCall)).(*[]multicallCall)

		resp := map[string]any{"jsonrpc": "2.0", "id": req.ID}
		switch m.Name {
		case "tryBlockAndAggregate":
			results := make([]multicallResult, len(mcCalls))
			for i, c := range mcCalls {
				results[i].Success, results[i].ReturnData = answer(c.CallData)
			}
			result, err := m.Outputs.Pack(big.NewInt(testBlock), [32]byte{}, results)
			if err != nil {
				t.Errorf("pack %s: %v", m.Name, err)
			}
			resp["result"] = hexutil.Bytes(result)
		case "aggregate":
			returnData := make([][]byte, len(mcCalls))
			for i, c := range mcCalls {
				var ok bool
				if ok, returnData[i] = answer(c.CallData); !ok {
					resp["error"] = map[string]any{"code": 3, "message": "execution reverted",
						"data": hexutil.Bytes(errStalePrice)}
					break
				}
			}
			if resp["error"] == nil {
				result, err := m.Outputs.Pack(big.NewInt(testBlock), returnData)
				if err != nil {
					t.Errorf("pack %s: %v", m.Name, err)
				}
				resp["result"] = hexutil.Bytes(result)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)

	return ethrpc.New(srv.URL).SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")),
		&calls
}

// testPricedPool is the USDC/UBTC pool 0x0dbc588a… as last refreshed while its oracle was fresh.
func testPricedPool(t *testing.T) entity.Pool {
	fairPrice := uint256.MustFromDecimal(testFairPrice)
	extra, err := json.Marshal(Extra{FairPriceFrom: [2]*uint256.Int{fairPrice, fairPrice}, RefFeeFrom: [2]uint64{6, 6}})
	require.NoError(t, err)
	return entity.Pool{
		Address:  "0x0dbc588a63da9c93c68aba9d6c860fefab674b70",
		Exchange: DexType,
		Type:     DexType,
		Reserves: entity.PoolReserves{"30218182", "76836"},
		Tokens: []*entity.PoolToken{
			{Address: "0xb88339cb7199b77e23db6e890353e22632ba630f", Decimals: 6, Swappable: true},
			{Address: "0x9fdbda0a5e284c32744d2f17ee5c74b284993463", Decimals: 8, Swappable: true},
		},
		Extra:       string(extra),
		StaticExtra: `{"s":"0x63b4915a4c73face606b0e1846b9ecba0f8d76d3"}`,
	}
}

// A HyperAMM swap reverts whenever its fee module can't price the pool (e.g. Pyth StalePrice once
// the oracle stops being updated). The tracker must then publish the pool as unquotable: returning
// an error would leave the last priced state in place, and the pool would keep being quoted at a
// price no swap can execute.
func TestTrackerMarksOracleRevertUnquotable(t *testing.T) {
	for name, tc := range map[string]struct {
		reverts  []string
		calls    int32
		err      error
		quotable bool
	}{
		"healthy":            {calls: 2, quotable: true},
		"fair price reverts": {reverts: []string{"fairPriceInScale18", "getSwapFeeInBips"}, calls: 1},
		// An unreadable paused flag must not be published as unpaused.
		"paused reverts": {reverts: []string{"paused"}, calls: 1, err: ErrReadPoolState},
	} {
		t.Run(name, func(t *testing.T) {
			client, calls := fakeHyperAMMRPC(t, tc.reverts...)
			p, err := NewPoolTracker(&Config{Lens: "0xa099e8a1b9646e4b56b09644d0672d87ba3a0b60"}, client).
				GetNewPoolState(context.Background(), testPricedPool(t), poolpkg.GetNewPoolStateParams{})
			require.Equal(t, tc.calls, calls.Load())
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			require.EqualValues(t, testBlock, p.BlockNumber)

			sim, err := NewPoolSimulator(p)
			require.NoError(t, err)
			for i, amountIn := range []int64{50e6, 30000} {
				_, err := sim.CalcAmountOut(poolpkg.CalcAmountOutParams{
					TokenAmountIn: poolpkg.TokenAmount{Token: p.Tokens[i].Address, Amount: big.NewInt(amountIn)},
					TokenOut:      p.Tokens[1-i].Address,
				})
				if tc.quotable {
					require.NoError(t, err)
				} else {
					require.ErrorIs(t, err, ErrZeroFairPrice)
				}
			}
		})
	}
}
