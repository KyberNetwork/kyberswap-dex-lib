package ethrpcmock

import (
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

const Block uint64 = 12345

var ErrRevert = errors.New("ethrpcmock: call reverted")

var ErrRPCError = errors.New("ethrpcmock: rpc call failed")

type Call struct {
	Target   common.Address
	CallData []byte
}

type InnerCall struct {
	Target   common.Address
	Method   string
	CallData string
	Inputs   []any
}

type RPCRequest struct {
	OuterMethod string
	BlockTag    string
	Overrides   json.RawMessage
	Calls       []InnerCall
}

type Resolver func(target common.Address, method abi.Method, inputs []any) ([]any, error)

type Recorder struct {
	mu       sync.Mutex
	Requests []RPCRequest
}

func (r *Recorder) Snapshot() []RPCRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]RPCRequest, len(r.Requests))
	copy(result, r.Requests)
	return result
}

type Result struct {
	Success    bool
	ReturnData []byte
}

var outer = func() abi.ABI {
	const schema = `[
	{"type":"function","name":"aggregate","inputs":[{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],"outputs":[{"name":"blockNumber","type":"uint256"},{"name":"returnData","type":"bytes[]"}]},
	{"type":"function","name":"tryAggregate","inputs":[{"name":"requireSuccess","type":"bool"},{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],"outputs":[{"name":"returnData","type":"tuple[]","components":[{"name":"success","type":"bool"},{"name":"returnData","type":"bytes"}]}]},
	{"type":"function","name":"tryBlockAndAggregate","inputs":[{"name":"requireSuccess","type":"bool"},{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],"outputs":[{"name":"blockNumber","type":"uint256"},{"name":"blockHash","type":"bytes32"},{"name":"returnData","type":"tuple[]","components":[{"name":"success","type":"bool"},{"name":"returnData","type":"bytes"}]}]},
	{"type":"function","name":"getCurrentBlockTimestamp","inputs":[],"outputs":[{"name":"timestamp","type":"uint256"}]}
	]`
	a, err := abi.JSON(strings.NewReader(schema))
	if err != nil {
		panic(err)
	}
	return a
}()

func New(t *testing.T, abis ...abi.ABI) (*httptest.Server, *Recorder) {
	return NewWithResolver(t, nil, abis...)
}

func NewWithResolver(t *testing.T, resolver Resolver, abis ...abi.ABI) (*httptest.Server, *Recorder) {
	t.Helper()
	selectors := map[[4]byte]abi.Method{}
	for _, a := range abis {
		for _, m := range a.Methods {
			selectors[[4]byte(m.ID)] = m
		}
	}
	recorder := new(Recorder)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     json.RawMessage   `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		var result any = "0x1"
		if req.Method == "eth_call" {
			var call struct {
				Input string         `json:"input"`
				Data  string         `json:"data"`
				To    common.Address `json:"to"`
			}
			if err := json.Unmarshal(req.Params[0], &call); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			input := call.Input
			if input == "" {
				input = call.Data
			}
			data, err := hexutil.Decode(input)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			var tag string
			if len(req.Params) > 1 {
				_ = json.Unmarshal(req.Params[1], &tag)
			}
			entry := RPCRequest{BlockTag: tag}
			if len(req.Params) > 2 {
				entry.Overrides = append(json.RawMessage(nil), req.Params[2]...)
			}
			if len(data) < 4 {
				http.Error(w, "short eth_call data", 400)
				return
			}
			name := ""
			var directMethod abi.Method
			directCall := false
			for n, m := range outer.Methods {
				if string(m.ID) == string(data[:4]) {
					name = n
					break
				}
			}
			if name == "" {
				if m, ok := selectors[[4]byte(data[:4])]; ok {
					directMethod = m
					directCall = true
				}
			}
			switch name {
			case "aggregate", "tryAggregate", "tryBlockAndAggregate":
				entry.OuterMethod = name
				method := outer.Methods[name]
				args, unpackErr := method.Inputs.Unpack(data[4:])
				if unpackErr != nil {
					http.Error(w, unpackErr.Error(), 400)
					return
				}
				var rawCalls []Call
				callArg := args[len(args)-1]
				v := reflect.ValueOf(callArg)
				for i := 0; i < v.Len(); i++ {
					item := v.Index(i)
					rawCalls = append(rawCalls, Call{Target: item.Field(0).Interface().(common.Address), CallData: item.Field(1).Bytes()})
				}
				outs := make([][]byte, len(rawCalls))
				successes := make([]bool, len(rawCalls))
				var rpcErr error
				for i, c := range rawCalls {
					successes[i] = true
					if len(c.CallData) < 4 {
						http.Error(w, "short inner calldata", 400)
						return
					}
					m, ok := selectors[[4]byte(c.CallData[:4])]
					if !ok {
						http.Error(w, "unknown inner selector", 400)
						return
					}
					inputs, inputErr := m.Inputs.Unpack(c.CallData[4:])
					if inputErr != nil {
						http.Error(w, inputErr.Error(), 400)
						return
					}
					entry.Calls = append(entry.Calls, InnerCall{Target: c.Target, Method: m.RawName, CallData: hexutil.Encode(c.CallData), Inputs: inputs})
					var values []any
					if resolver != nil {
						values, err = resolver(c.Target, m, inputs)
						if err != nil {
							if errors.Is(err, ErrRevert) {
								successes[i] = false
								continue
							}
							if errors.Is(err, ErrRPCError) {
								rpcErr = err
								values = make([]any, len(m.Outputs))
								for j, output := range m.Outputs {
									values[j] = value(output.Type)
								}
							} else {
								http.Error(w, err.Error(), 400)
								return
							}
						}
					} else {
						values = make([]any, len(m.Outputs))
						for j, arg := range m.Outputs {
							values[j] = value(arg.Type)
						}
					}
					outs[i], err = m.Outputs.Pack(values...)
					if err != nil {
						http.Error(w, err.Error(), 400)
						return
					}
				}
				block := new(big.Int).SetUint64(Block)
				recorder.mu.Lock()
				recorder.Requests = append(recorder.Requests, entry)
				recorder.mu.Unlock()
				if rpcErr != nil {
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "error": map[string]any{"code": -32000, "message": rpcErr.Error()}})
					return
				}
				switch name {
				case "aggregate":
					for _, success := range successes {
						if !success {
							w.Header().Set("Content-Type", "application/json")
							_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "error": map[string]any{"code": 3, "message": "execution reverted"}})
							return
						}
					}
					resultBytes, e := method.Outputs.Pack(block, outs)
					if e != nil {
						http.Error(w, e.Error(), 400)
						return
					}
					result = hexutil.Encode(resultBytes)
				case "tryAggregate":
					items := make([]Result, len(outs))
					for i := range outs {
						items[i] = Result{successes[i], outs[i]}
					}
					resultBytes, e := method.Outputs.Pack(items)
					if e != nil {
						http.Error(w, e.Error(), 400)
						return
					}
					result = hexutil.Encode(resultBytes)
				case "tryBlockAndAggregate":
					items := make([]Result, len(outs))
					for i := range outs {
						items[i] = Result{successes[i], outs[i]}
					}
					resultBytes, e := method.Outputs.Pack(block, [32]byte{}, items)
					if e != nil {
						http.Error(w, e.Error(), 400)
						return
					}
					result = hexutil.Encode(resultBytes)
				}
			case "getCurrentBlockTimestamp":
				entry.OuterMethod = name
				recorder.mu.Lock()
				recorder.Requests = append(recorder.Requests, entry)
				recorder.mu.Unlock()
				resultBytes, e := outer.Methods[name].Outputs.Pack(big.NewInt(1700000000))
				if e != nil {
					http.Error(w, e.Error(), 400)
					return
				}
				result = hexutil.Encode(resultBytes)
			case "":
				if !directCall {
					http.Error(w, "unknown multicall selector", 400)
					return
				}
				inputs, e := directMethod.Inputs.Unpack(data[4:])
				if e != nil {
					http.Error(w, e.Error(), 400)
					return
				}
				entry.OuterMethod = "call"
				entry.Calls = []InnerCall{{Target: call.To, Method: directMethod.RawName, CallData: hexutil.Encode(data), Inputs: inputs}}
				var values []any
				if resolver != nil {
					values, e = resolver(call.To, directMethod, inputs)
					if e != nil {
						http.Error(w, e.Error(), 400)
						return
					}
				} else {
					values = make([]any, len(directMethod.Outputs))
					for i, arg := range directMethod.Outputs {
						values[i] = value(arg.Type)
					}
				}
				resultBytes, e := directMethod.Outputs.Pack(values...)
				if e != nil {
					http.Error(w, e.Error(), 400)
					return
				}
				recorder.mu.Lock()
				recorder.Requests = append(recorder.Requests, entry)
				recorder.mu.Unlock()
				result = hexutil.Encode(resultBytes)
			default:
				http.Error(w, "unknown multicall selector", 400)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
	}))
	t.Cleanup(srv.Close)
	return srv, recorder
}

func value(t abi.Type) any {
	switch t.T {
	case abi.UintTy, abi.IntTy:
		if t.Size > 64 {
			return big.NewInt(1)
		}
		v := reflect.New(t.GetType()).Elem()
		if t.T == abi.UintTy {
			v.SetUint(1)
		} else {
			v.SetInt(1)
		}
		return v.Interface()
	case abi.BoolTy:
		return false
	case abi.AddressTy:
		return common.Address{}
	case abi.StringTy:
		return ""
	case abi.BytesTy:
		return []byte{}
	case abi.FixedBytesTy:
		return reflect.Zero(t.GetType()).Interface()
	case abi.SliceTy:
		v := reflect.MakeSlice(t.GetType(), 2, 2)
		for i := 0; i < v.Len(); i++ {
			v.Index(i).Set(reflect.ValueOf(value(*t.Elem)))
		}
		return v.Interface()
	case abi.ArrayTy:
		v := reflect.New(t.GetType()).Elem()
		for i := 0; i < v.Len(); i++ {
			v.Index(i).Set(reflect.ValueOf(value(*t.Elem)))
		}
		return v.Interface()
	case abi.TupleTy:
		v := reflect.New(t.GetType()).Elem()
		for i := 0; i < v.NumField(); i++ {
			v.Field(i).Set(reflect.ValueOf(value(*t.TupleElems[i])))
		}
		return v.Interface()
	default:
		panic("unsupported ABI output type")
	}
}

func Value(t abi.Type) any {
	return value(t)
}

func RequirePinned(t *testing.T, tags []string, callsMin int) {
	t.Helper()
	if os.Getenv("BLOCKPIN_BASELINE") == "1" {
		return
	}
	require.GreaterOrEqual(t, len(tags), callsMin)
	require.Equal(t, "latest", tags[0])
	for _, tag := range tags[1:] {
		require.Equal(t, "0x3039", tag)
	}
}

func RequireRequestsPinned(t *testing.T, recorder *Recorder, callsMin int) {
	t.Helper()
	if os.Getenv("BLOCKPIN_BASELINE") == "1" {
		return
	}
	requests := recorder.Snapshot()
	require.GreaterOrEqual(t, len(requests), callsMin)
	require.Equal(t, "latest", requests[0].BlockTag)
	for _, request := range requests[1:] {
		require.Equal(t, "0x3039", request.BlockTag)
	}
}

func CheckPool(t *testing.T, key string, pool entity.Pool) {
	t.Helper()
	dir := filepath.Join("testdata")
	if os.Getenv("BLOCKPIN_BASELINE") == "1" {
		dir = os.Getenv("BLOCKPIN_EQ_DIR")
		require.NotEmpty(t, dir, "BLOCKPIN_EQ_DIR must point to the baseline pool directory")
	}
	path := filepath.Join(dir, key+".json")
	if os.Getenv("BLOCKPIN_BASELINE") == "1" {
		data, err := json.Marshal(pool)
		require.NoError(t, err)
		require.NoError(t, os.MkdirAll(dir, 0o755))
		require.NoError(t, os.WriteFile(path, data, 0o600))
		return
	}
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var baseline entity.Pool
	require.NoError(t, json.Unmarshal(data, &baseline))
	baseline.BlockNumber = pool.BlockNumber
	require.Equal(t, baseline, pool)
}

func CheckRequests(t *testing.T, key string, recorder *Recorder) {
	t.Helper()
	dir := filepath.Join("testdata")
	if os.Getenv("BLOCKPIN_BASELINE") == "1" {
		dir = os.Getenv("BLOCKPIN_EQ_DIR")
		require.NotEmpty(t, dir, "BLOCKPIN_EQ_DIR must point to the baseline pool directory")
	}
	path := filepath.Join(dir, key+".rpc.json")
	requests := recorder.Snapshot()
	for i := range requests {
		requests[i].BlockTag = ""
	}
	data, err := json.Marshal(requests)
	require.NoError(t, err)
	if os.Getenv("BLOCKPIN_BASELINE") == "1" {
		require.NoError(t, os.MkdirAll(dir, 0o755))
		require.NoError(t, os.WriteFile(path, data, 0o600))
		return
	}
	baseline, err := os.ReadFile(path)
	require.NoError(t, err)
	var expected []RPCRequest
	var actual []RPCRequest
	require.NoError(t, json.Unmarshal(baseline, &expected))
	require.NoError(t, json.Unmarshal(data, &actual))
	if len(expected) > 0 && len(actual) > 0 && ((expected[0].OuterMethod == "tryAggregate" && actual[0].OuterMethod == "tryBlockAndAggregate") || (expected[0].OuterMethod == "tryBlockAndAggregate" && actual[0].OuterMethod == "tryAggregate")) {
		actual[0].OuterMethod = expected[0].OuterMethod
	}
	if len(expected) > 0 && len(actual) > 0 && expected[0].OuterMethod == "call" && actual[0].OuterMethod == "aggregate" {
		actual[0].OuterMethod = "call"
	}
	expectedJSON, err := json.Marshal(expected)
	require.NoError(t, err)
	actualJSON, err := json.Marshal(actual)
	require.NoError(t, err)
	require.JSONEq(t, string(expectedJSON), string(actualJSON))
}
