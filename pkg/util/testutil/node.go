package testutil

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	utilabi "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

// gasCapProbe is the init code pkg/util/ephemeral sends to learn a node's gas cap: it returns the
// gas it starts with, 53124 below the gas of the call.
var gasCapProbe = []byte{0x5a, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xf3}

// NodeGasCap is the gas a NodeStub runs a call with at most, as a node capping eth_call does.
const NodeGasCap = 50_000_000

// NodeCall is an eth_call a NodeStub received.
type NodeCall struct {
	To    *common.Address // nil for a deployless call, which runs Data as init code
	Data  []byte
	Gas   uint64
	Tag   string // the block argument: "latest" or a hex number
	Block uint64 // Tag's number, the stub's head for "latest"
}

// Revert is the error of a call that reverted with data, such as a lens's result; with none, a
// revert without data.
type Revert []byte

func (Revert) Error() string { return "execution reverted" }

// NodeStub is a JSON-RPC node at block head of a chain, for code that reads through ethrpc and
// through deployless calls. It answers eth_chainId, eth_blockNumber and the gas cap probe itself,
// and every other eth_call by its handler: a Revert error as a revert with data, any other error
// as a node's failure with that message.
type NodeStub struct {
	*httptest.Server
}

// NewNodeStub serves chainID at block head, answering each eth_call by call.
func NewNodeStub(t testing.TB, chainID, head uint64, call func(NodeCall) ([]byte, error)) *NodeStub {
	s := &NodeStub{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     json.RawMessage   `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&req)) {
			return
		}
		reply := map[string]any{"jsonrpc": "2.0", "id": req.ID}
		defer func() { _ = json.NewEncoder(w).Encode(reply) }()
		switch req.Method {
		case "eth_chainId":
			reply["result"] = hexutil.Uint64(chainID)
			return
		case "eth_blockNumber":
			reply["result"] = hexutil.Uint64(head)
			return
		case "eth_call":
		default:
			t.Errorf("unexpected %s", req.Method)
			reply["error"] = map[string]any{"code": -32601, "message": "method not found"}
			return
		}

		var msg struct {
			To    *common.Address `json:"to"`
			Data  *hexutil.Bytes  `json:"data"`
			Input *hexutil.Bytes  `json:"input"`
			Gas   hexutil.Uint64  `json:"gas"`
		}
		if !assert.NotEmpty(t, req.Params) || !assert.NoError(t, json.Unmarshal(req.Params[0], &msg)) {
			return
		}
		c := NodeCall{To: msg.To, Gas: uint64(msg.Gas), Tag: "latest", Block: head}
		if in := cmp.Or(msg.Input, msg.Data); in != nil {
			c.Data = *in
		}
		if len(req.Params) > 1 {
			_ = json.Unmarshal(req.Params[1], &c.Tag)
		}
		if n, err := hexutil.DecodeUint64(c.Tag); err == nil {
			c.Block = n
		}

		if c.To == nil && bytes.Equal(c.Data, gasCapProbe) {
			reply["result"] = hexutil.Bytes(common.BigToHash(new(big.Int).SetUint64(
				min(c.Gas, NodeGasCap) - 53_124)).Bytes())
			return
		}
		ret, err := call(c)
		if revert, ok := errors.AsType[Revert](err); ok {
			e := map[string]any{"code": 3, "message": "execution reverted"}
			if len(revert) > 0 {
				e["data"] = hexutil.Bytes(revert)
			}
			reply["error"] = e
		} else if err != nil {
			reply["error"] = map[string]any{"code": -32000, "message": err.Error()}
		} else {
			reply["result"] = hexutil.Bytes(ret)
		}
	}))
	t.Cleanup(s.Close)
	return s
}

// Client returns an ethrpc client reading from the stub, with Multicall3 as its multicall contract.
func (s *NodeStub) Client() *ethrpc.Client {
	return ethrpc.New(s.URL).SetMulticallContract(Multicall3)
}

// multicallMethods are the Multicall3 methods a stub answers itself, those ethrpc calls; any other
// call, a Multicall3 getter among them, goes to respond.
var multicallMethods = []string{"aggregate", "tryAggregate", "tryBlockAndAggregate", "getCurrentBlockTimestamp"}

type stubCall struct {
	Target   common.Address
	CallData []byte
}

type stubResult struct {
	Success    bool
	ReturnData []byte
}

// Multicall answers an eth_call to to, telling a multicall from another call by its selector. A
// multicall (aggregate, tryAggregate, tryBlockAndAggregate or getCurrentBlockTimestamp) answers each
// of its calls by respond and reports block and timestamp as its own; any other call is answered by
// respond directly. respond's ok=false fails a call: in a try multicall it fails alone, anywhere
// else it fails the eth_call. targets is the calls' targets.
func Multicall(to common.Address, data []byte, block, timestamp uint64,
	respond func(target common.Address, callData []byte) (ok bool, ret []byte)) (ret []byte,
	targets []common.Address, err error) {
	method, err := utilabi.Multicall3ABI.MethodById(data)
	if err != nil || !slices.Contains(multicallMethods, method.Name) {
		if ok, ret := respond(to, data); ok {
			return ret, []common.Address{to}, nil
		}
		return nil, []common.Address{to}, fmt.Errorf("call to %s failed", to)
	}
	if method.Name == "getCurrentBlockTimestamp" {
		ret, err := method.Outputs.Pack(new(big.Int).SetUint64(timestamp))
		return ret, nil, err
	}

	args, err := method.Inputs.Unpack(data[4:])
	if err != nil {
		return nil, nil, err
	}
	calls := *abi.ConvertType(args[len(args)-1], new([]stubCall)).(*[]stubCall)
	targets = make([]common.Address, len(calls))
	results := make([]stubResult, len(calls))
	for i, c := range calls {
		targets[i] = c.Target
		results[i].Success, results[i].ReturnData = respond(c.Target, c.CallData)
	}

	blockNumber := new(big.Int).SetUint64(block)
	switch method.Name {
	case "aggregate":
		rets := make([][]byte, len(results))
		for i, r := range results {
			if !r.Success {
				return nil, targets, fmt.Errorf("call %d to %s failed", i, calls[i].Target)
			}
			rets[i] = r.ReturnData
		}
		ret, err = method.Outputs.Pack(blockNumber, rets)
	case "tryAggregate":
		ret, err = method.Outputs.Pack(results)
	default:
		ret, err = method.Outputs.Pack(blockNumber, common.Hash{}, results)
	}
	return ret, targets, err
}

// MustPack packs values as args, failing t if they do not fit.
func MustPack(t testing.TB, args abi.Arguments, values ...any) []byte {
	t.Helper()
	out, err := args.Pack(values...)
	require.NoError(t, err)
	return out
}
