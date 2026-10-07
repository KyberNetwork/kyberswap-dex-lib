package uscoreprop

import (
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
)

const (
	evmBlock     = 100
	evmTimestamp = 1_800_000_000
	evmGasCap    = 50_000_000
)

var mockPoolArgs = lo.Must(abi.JSON(strings.NewReader(`[{"type":"constructor","inputs":[
 {"name":"t0","type":"address"},{"name":"reserves","type":"uint256[2]"},
 {"name":"c0","type":"uint256[3]"},{"name":"c1","type":"uint256[3]"},{"name":"pause","type":"bool"}]}]`)))

// mockCurve prices one direction of testdata/MockUSCorePool.sol: out = x*num/den - fee with
// fee = gross*x/(2*peak), so it peaks at x = peak; past 2*peak, the reserve or at zero out it's status 3.
type mockCurve struct{ num, den, peak uint64 }

type mockPool struct {
	reserves [2]*uint256.Int
	curves   [2]mockCurve
	paused   bool
}

func (m mockPool) quote(dir int, x *uint256.Int) (*uint256.Int, uint8) {
	if m.paused {
		return new(uint256.Int), 4
	}
	c := m.curves[dir]
	var twoPeak, gross, fee uint256.Int
	twoPeak.SetUint64(2 * c.peak)
	if x.Gt(&twoPeak) {
		return new(uint256.Int), statusInsufficient
	}
	gross.Mul(x, uint256.NewInt(c.num)).Div(&gross, uint256.NewInt(c.den))
	fee.Mul(&gross, x).Div(&fee, &twoPeak)
	out := new(uint256.Int).Sub(&gross, &fee)
	if out.IsZero() || out.Gt(m.reserves[1-dir]) {
		return new(uint256.Int), statusInsufficient
	}
	return out, statusOK
}

// quoter, so the Go reference samples the same pool the EVM does.
type mockQuoter struct{ mockPool }

func (q mockQuoter) reserves() [2]*uint256.Int { return q.mockPool.reserves }

func (q mockQuoter) quoteExactIn(dir int, x *uint256.Int) (*uint256.Int, uint8) {
	return q.quote(dir, x)
}

func (q mockQuoter) quoteLadder(dir int, xs []*uint256.Int) ([]*uint256.Int, uint8) {
	outs := make([]*uint256.Int, len(xs))
	var status uint8
	for i, x := range xs {
		var s uint8
		if outs[i], s = q.quote(dir, x); s != statusOK {
			status = s
		}
	}
	return outs, status
}

// evmRPC serves to-less eth_calls by running them in an in-process EVM where m is deployed at
// testPool, the way a node runs USCoreSampler. calls counts eth_calls; overrides counts the
// ones carrying a state override.
type evmRPC struct {
	client           *ethrpc.Client
	calls, overrides atomic.Int32
	gasUsed          atomic.Uint64
}

func newEVMRPC(t *testing.T, m mockPool) *evmRPC {
	t.Helper()
	statedb, err := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	require.NoError(t, err)
	cfg := &runtime.Config{State: statedb, BlockNumber: big.NewInt(evmBlock), Time: evmTimestamp, GasLimit: evmGasCap}
	bin, err := os.ReadFile("testdata/MockUSCorePool.bin")
	require.NoError(t, err)
	curve := func(c mockCurve) [3]*big.Int {
		return [3]*big.Int{new(big.Int).SetUint64(c.num), new(big.Int).SetUint64(c.den), new(big.Int).SetUint64(c.peak)}
	}
	args, err := mockPoolArgs.Pack("", common.HexToAddress(testQuote),
		[2]*big.Int{m.reserves[0].ToBig(), m.reserves[1].ToBig()}, curve(m.curves[0]), curve(m.curves[1]), m.paused)
	require.NoError(t, err)
	code, _, _, err := runtime.Create(append(common.FromHex(string(bin)), args...), cfg)
	require.NoError(t, err)
	statedb.SetCode(common.HexToAddress(testPool), code, tracing.CodeChangeUnspecified)

	r := &evmRPC{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			ID     json.RawMessage
			Method string
			Params []json.RawMessage
		}
		require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
		require.Equal(t, "eth_call", body.Method)
		r.calls.Add(1)
		if len(body.Params) > 2 {
			r.overrides.Add(1)
		}
		var call struct {
			To   *common.Address
			Data hexutil.Bytes
		}
		require.NoError(t, json.Unmarshal(body.Params[0], &call))
		require.Nil(t, call.To, "the tracker only sends deployless calls")
		run := *cfg
		run.State = statedb.Copy()
		ret, _, left, err := runtime.Create(call.Data, &run)
		r.gasUsed.Store(evmGasCap - left)
		resp := map[string]any{"jsonrpc": "2.0", "id": body.ID}
		switch {
		case errors.Is(err, vm.ErrExecutionReverted):
			resp["error"] = map[string]any{"code": 3, "message": "execution reverted", "data": hexutil.Encode(ret)}
		case err != nil:
			resp["error"] = map[string]any{"code": -32000, "message": err.Error()}
		default:
			resp["result"] = hexutil.Encode(ret)
		}
		w.Header().Set("content-type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
	t.Cleanup(server.Close)
	r.client = ethrpc.New(server.URL)
	return r
}
