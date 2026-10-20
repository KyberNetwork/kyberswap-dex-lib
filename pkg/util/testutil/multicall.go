package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const tryAggregateABI = `[{"name":"tryAggregate","type":"function","stateMutability":"nonpayable",
"inputs":[{"name":"requireSuccess","type":"bool"},{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],
"outputs":[{"name":"returnData","type":"tuple[]","components":[{"name":"success","type":"bool"},{"name":"returnData","type":"bytes"}]}]}]`

// TryAggregateStub is a JSON-RPC server answering every eth_call as a multicall tryAggregate,
// for testing code that batches calls with ethrpc's TryAggregate.
type TryAggregateStub struct {
	*httptest.Server
	mu      sync.Mutex
	targets [][]common.Address
}

// NewTryAggregateStub answers each sub-call with respond; ok=false makes that sub-call fail.
func NewTryAggregateStub(t testing.TB,
	respond func(target common.Address, callData []byte) (ok bool, ret []byte)) *TryAggregateStub {
	multicallABI, err := abi.JSON(strings.NewReader(tryAggregateABI))
	require.NoError(t, err)
	tryAggregate := multicallABI.Methods["tryAggregate"]

	type call struct {
		Target   common.Address
		CallData []byte
	}
	type result struct {
		Success    bool
		ReturnData []byte
	}

	s := &TryAggregateStub{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     json.RawMessage   `json:"id"`
			Params []json.RawMessage `json:"params"`
		}
		var arg struct {
			Input hexutil.Bytes `json:"input"`
		}
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&req)) || !assert.NotEmpty(t, req.Params) ||
			!assert.NoError(t, json.Unmarshal(req.Params[0], &arg)) {
			return
		}
		args, err := tryAggregate.Inputs.Unpack(arg.Input[4:])
		if !assert.NoError(t, err) {
			return
		}
		calls := *abi.ConvertType(args[1], new([]call)).(*[]call)

		results := make([]result, len(calls))
		targets := make([]common.Address, len(calls))
		for i, c := range calls {
			targets[i] = c.Target
			results[i].Success, results[i].ReturnData = respond(c.Target, c.CallData)
		}
		s.mu.Lock()
		s.targets = append(s.targets, targets)
		s.mu.Unlock()

		ret, err := tryAggregate.Outputs.Pack(results)
		if !assert.NoError(t, err) {
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": hexutil.Encode(ret)})
	}))
	t.Cleanup(s.Close)
	return s
}

// Targets returns the sub-call targets of each eth_call received so far.
func (s *TryAggregateStub) Targets() [][]common.Address {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([][]common.Address(nil), s.targets...)
}
