package testutil

import (
	"cmp"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// multicallABI holds the Multicall3 methods ethrpc calls.
const multicallABI = `[
{"name":"aggregate","type":"function","stateMutability":"nonpayable",
"inputs":[{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],
"outputs":[{"name":"blockNumber","type":"uint256"},{"name":"returnData","type":"bytes[]"}]},
{"name":"tryAggregate","type":"function","stateMutability":"nonpayable",
"inputs":[{"name":"requireSuccess","type":"bool"},{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],
"outputs":[{"name":"returnData","type":"tuple[]","components":[{"name":"success","type":"bool"},{"name":"returnData","type":"bytes"}]}]},
{"name":"tryBlockAndAggregate","type":"function","stateMutability":"nonpayable",
"inputs":[{"name":"requireSuccess","type":"bool"},{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],
"outputs":[{"name":"blockNumber","type":"uint256"},{"name":"blockHash","type":"bytes32"},{"name":"returnData","type":"tuple[]","components":[{"name":"success","type":"bool"},{"name":"returnData","type":"bytes"}]}]},
{"name":"getCurrentBlockTimestamp","type":"function","stateMutability":"view","inputs":[],
"outputs":[{"name":"timestamp","type":"uint256"}]}]`

// MulticallStub is a JSON-RPC server answering eth_call for code that reads through ethrpc. A call
// is told from a multicall by its selector: a multicall (aggregate, tryAggregate,
// tryBlockAndAggregate or getCurrentBlockTimestamp) answers each of its calls by respond, and any
// other call is answered by respond directly.
type MulticallStub struct {
	*httptest.Server
	multicall abi.ABI

	mu               sync.Mutex
	block, timestamp uint64
	targets          [][]common.Address
}

// NewMulticallStub answers each call with respond; ok=false makes that call fail: in a try
// multicall it fails alone, anywhere else it reverts the eth_call.
func NewMulticallStub(t testing.TB,
	respond func(target common.Address, callData []byte) (ok bool, ret []byte)) *MulticallStub {
	multicall, err := abi.JSON(strings.NewReader(multicallABI))
	require.NoError(t, err)
	s := &MulticallStub{multicall: multicall}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     json.RawMessage   `json:"id"`
			Params []json.RawMessage `json:"params"`
		}
		var call struct {
			To    common.Address `json:"to"`
			Data  *hexutil.Bytes `json:"data"`
			Input *hexutil.Bytes `json:"input"`
		}
		var block string
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&req)) || !assert.NotEmpty(t, req.Params) ||
			!assert.NoError(t, json.Unmarshal(req.Params[0], &call)) {
			return
		}
		if len(req.Params) > 1 {
			_ = json.Unmarshal(req.Params[1], &block)
		}
		var data []byte
		if in := cmp.Or(call.Input, call.Data); in != nil {
			data = *in
		}

		ret, err := s.call(block, call.To, data, respond)
		reply := map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": hexutil.Encode(ret)}
		if err != nil {
			reply = map[string]any{"jsonrpc": "2.0", "id": req.ID,
				"error": map[string]any{"code": 3, "message": "execution reverted: " + err.Error()}}
		}
		_ = json.NewEncoder(w).Encode(reply)
	}))
	t.Cleanup(s.Close)
	return s
}

// At makes the stub a chain at block: a read at any other block fails, a multicall reports block
// as its own, and getCurrentBlockTimestamp returns timestamp. A read at "latest" is one at block.
func (s *MulticallStub) At(block, timestamp uint64) *MulticallStub {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.block, s.timestamp = block, timestamp
	return s
}

// Client returns an ethrpc client reading from the stub. Its multicall address is Multicall3's, though
// the stub takes any call with a multicall's selector for one.
func (s *MulticallStub) Client() *ethrpc.Client {
	return ethrpc.New(s.URL).SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
}

// Targets returns the targets of each eth_call received so far, a multicall's being its calls'.
func (s *MulticallStub) Targets() [][]common.Address {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([][]common.Address(nil), s.targets...)
}

type stubCall struct {
	Target   common.Address
	CallData []byte
}

type stubResult struct {
	Success    bool
	ReturnData []byte
}

func (s *MulticallStub) call(blockTag string, to common.Address, data []byte,
	respond func(common.Address, []byte) (bool, []byte)) ([]byte, error) {
	s.mu.Lock()
	block, timestamp := s.block, s.timestamp
	s.mu.Unlock()
	if n, err := hexutil.DecodeUint64(blockTag); block > 0 && err == nil && n != block {
		return nil, fmt.Errorf("a read at block %d of a chain at %d", n, block)
	}

	method, err := s.multicall.MethodById(data)
	if err != nil {
		s.record(to)
		if ok, ret := respond(to, data); ok {
			return ret, nil
		}
		return nil, fmt.Errorf("call to %s failed", to)
	}
	if method.Name == "getCurrentBlockTimestamp" {
		return method.Outputs.Pack(new(big.Int).SetUint64(timestamp))
	}

	args, err := method.Inputs.Unpack(data[4:])
	if err != nil {
		return nil, err
	}
	calls := *abi.ConvertType(args[len(args)-1], new([]stubCall)).(*[]stubCall)
	targets := make([]common.Address, len(calls))
	results := make([]stubResult, len(calls))
	for i, c := range calls {
		targets[i] = c.Target
		results[i].Success, results[i].ReturnData = respond(c.Target, c.CallData)
	}
	s.record(targets...)

	blockNumber := new(big.Int).SetUint64(block)
	switch method.Name {
	case "aggregate":
		rets := make([][]byte, len(results))
		for i, r := range results {
			if !r.Success {
				return nil, fmt.Errorf("call %d to %s failed", i, calls[i].Target)
			}
			rets[i] = r.ReturnData
		}
		return method.Outputs.Pack(blockNumber, rets)
	case "tryAggregate":
		return method.Outputs.Pack(results)
	default:
		return method.Outputs.Pack(blockNumber, common.Hash{}, results)
	}
}

func (s *MulticallStub) record(targets ...common.Address) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.targets = append(s.targets, targets)
}

// RecordedCalls answers each call from calls, which holds what each returned on chain keyed by
// "<to> <calldata>" in lower-case hex. A call it lacks fails the test.
func RecordedCalls(t testing.TB, calls map[string]string) func(common.Address, []byte) (bool, []byte) {
	return func(to common.Address, data []byte) (bool, []byte) {
		key := strings.ToLower(to.Hex()) + " " + hexutil.Encode(data)
		ret, ok := calls[key]
		assert.True(t, ok, "call not recorded: %s", key)
		return ok, common.FromHex(ret)
	}
}
