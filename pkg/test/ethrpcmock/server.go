package ethrpcmock

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
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
	return NewWithResolverAtBlock(t, resolver, Block, abis...)
}

func NewAtBlock(t *testing.T, block uint64, abis ...abi.ABI) (*httptest.Server, *Recorder) {
	return NewWithResolverAtBlock(t, nil, block, abis...)
}

func NewWithResolverAtBlock(t *testing.T, resolver Resolver, snapshotBlock uint64, abis ...abi.ABI) (*httptest.Server, *Recorder) {
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
				block := new(big.Int).SetUint64(snapshotBlock)
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
	require.GreaterOrEqual(t, len(tags), callsMin)
	require.Equal(t, "latest", tags[0])
	for _, tag := range tags[1:] {
		require.Equal(t, "0x3039", tag)
	}
}

func RequireRequestsPinned(t *testing.T, recorder *Recorder, callsMin int) {
	t.Helper()
	requests := recorder.Snapshot()
	require.GreaterOrEqual(t, len(requests), callsMin)
	require.Equal(t, "latest", requests[0].BlockTag)
	for _, request := range requests[1:] {
		require.Equal(t, "0x3039", request.BlockTag)
	}
}

func CheckPool(t *testing.T, key string, pool entity.Pool) {
	t.Helper()
	pool.BlockNumber = 0
	data, err := json.Marshal(pool)
	require.NoError(t, err)
	checkSnapshot(t, key, data)
}

func CheckRequests(t *testing.T, key string, recorder *Recorder, firstWrapperChange ...[2]string) {
	t.Helper()
	requests := recorder.Snapshot()
	require.LessOrEqual(t, len(firstWrapperChange), 1)
	if len(firstWrapperChange) == 1 {
		require.NotEmpty(t, requests)
		change := firstWrapperChange[0]
		require.Equal(t, change[1], requests[0].OuterMethod)
		requests[0].OuterMethod = change[0]
	}
	CheckRequestList(t, key, requests)
}

func CheckRequestList(t *testing.T, key string, requests []RPCRequest) {
	t.Helper()
	requests = append([]RPCRequest(nil), requests...)
	for i := range requests {
		requests[i].BlockTag = ""
	}
	data, err := json.Marshal(requests)
	require.NoError(t, err)
	var normalized []RPCRequest
	require.NoError(t, json.Unmarshal(data, &normalized))
	data, err = json.Marshal(normalized)
	require.NoError(t, err)
	checkSnapshot(t, key+".rpc", data)
}

func checkSnapshot(t *testing.T, key string, data []byte) {
	t.Helper()
	expected, ok := snapshotHashes[key]
	require.True(t, ok, "missing snapshot: %s", key)
	require.Equal(t, expected, fmt.Sprintf("%x", sha256.Sum256(data)), "%s: %s", key, data)
}

var snapshotHashes = map[string]string{
	"chainlink-chainlink-overrides":                                    "a3a5753c4d407d52c01a2f483808ba1297df079a95dc4a59887b7fc5534c513f",
	"chainlink-chainlink-overrides.rpc":                                "d7cbab28d07b7c63e4d86691a83aba8053ba8c82d3077c4edb71e5588828c4e8",
	"chainlink-chainlink-regular":                                      "a3a5753c4d407d52c01a2f483808ba1297df079a95dc4a59887b7fc5534c513f",
	"chainlink-chainlink-regular.rpc":                                  "28889c7b5198e6d883954654a76d9c75eca08c9560b5f9180855b4f0a85b111a",
	"max-max-overrides":                                                "6e795f25bfa4b6580a2b394ac72677bc94d3e322dfb074f58a9208bc3a3ea21a",
	"max-max-overrides.rpc":                                            "3d5893865b310715ac1630896165e095ce84a10ec3d48b4b5d0c163247231c52",
	"max-max-regular":                                                  "6e795f25bfa4b6580a2b394ac72677bc94d3e322dfb074f58a9208bc3a3ea21a",
	"max-max-regular.rpc":                                              "9462eaf7f6affc3c0d2c366763be974cab625397a5b8dfd13206b96d32342dd4",
	"morpho-morpho-overrides":                                          "a0b87d2587efd3db2bcba2f40682606390ba354ce5d928b21cdaeb597e7eaced",
	"morpho-morpho-overrides.rpc":                                      "cb07f94d19e074f3dcd573b46c3e4f3dcb47250511f134adf6a0f97fe213d227",
	"morpho-morpho-regular":                                            "a0b87d2587efd3db2bcba2f40682606390ba354ce5d928b21cdaeb597e7eaced",
	"morpho-morpho-regular.rpc":                                        "6924be1f1bf69c026fd0962b98e2ad88bc6f1e8e3c2cb7f96ba43ad9869ad775",
	"pyth-pyth-overrides":                                              "5c4da2a6340f29b393ec9d5d5c51535c1be707e57c59dd92ad3cad5a361069fe",
	"pyth-pyth-overrides.rpc":                                          "0ef21cebdc9ca90e407360c3fd804e8c8414396ec7270616b6f9bf1956279f12",
	"pyth-pyth-regular":                                                "5c4da2a6340f29b393ec9d5d5c51535c1be707e57c59dd92ad3cad5a361069fe",
	"pyth-pyth-regular.rpc":                                            "39fb5a8582983c5ea8c6d49343109d1b2ed33375ebaa9abbd7a4ef6fa3c42e1d",
	"bancor-v21_registry_nonempty":                                     "ab1fccbf1fca72f1eacad9b0c6076b37bbe94c44856c73c99b5c84ba41b34018",
	"bancor-v21_registry_nonempty.rpc":                                 "5708bcda619cb33ecac6db27d066daa2b974fb55235b7d77754e34a6cc1e658c",
	"carbon-tracker-add":                                               "de220d2757e38f6a01d784f8daefef11d87e7d7b9150a98e78e644bc60e93909",
	"carbon-tracker-add.rpc":                                           "1bcfdaa678c0126368dade049d16949e432e938fbdf92f3482c6e885797e5857",
	"carbon-tracker-delete":                                            "c5ab83b49876a4c0f831067cde1d6918e5007c91a0a01061c3dd2b2abc746f4a",
	"carbon-tracker-delete.rpc":                                        "5de0b0c635d4685bcebfb1773b689477ab8f955ba7f6a2c0a6e1979f9038fce0",
	"carbon-tracker-fee-error":                                         "e25835c6b41384181b6bc3f4ffe56f8cf8d141ee1e366fc48439a3ab549625b4",
	"carbon-tracker-fee-error.rpc":                                     "5af421b77608b4414d6aa567f3fd7347247f69d2b6966a0aa93e410a7e54a47d",
	"carbon-tracker-fee-global":                                        "97b2bf94cead450ab3cc3d8d3213f90c7c57f74e40ec622ce4ec0451d5641fa2",
	"carbon-tracker-fee-global.rpc":                                    "52609ffd32b35ecc6df27918c842580cc1db2e1994ae6d99df0f89c72c680e65",
	"carbon-tracker-full":                                              "aa3c172b6a7c6a1d741183ed1652440af13f80e7716cd18185e90690408e6ff5",
	"carbon-tracker-full.rpc":                                          "3b84e8f97e19137f1033e80358afce854706da857d46760f9769dddf9492ad14",
	"carbon-tracker-update":                                            "de220d2757e38f6a01d784f8daefef11d87e7d7b9150a98e78e644bc60e93909",
	"carbon-tracker-update.rpc":                                        "b5104a37daaa91049c560a9a15ceabd71c383a4def385533ee1d92aa70b3fc95",
	"curve-llamma-tracker-pin-normal.rpc":                              "d785fc8636d5229a6f15a0532eacf61b423a514d6c29a3721391c6b77ebbd449",
	"curve-llamma-tracker-pin-overrides.rpc":                           "1d59d3a9d6eab8831b27322415f7b8f4640f5d29230d87ca591fa8aa136fbc5e",
	"curve-llamma-tracker-pin":                                         "13cf0a37104d508a4b381825c6ea9a39d59881dab4796e628d73655bec568c8b",
	"ether-vista-tracker-pin-normal.rpc":                               "b3d574954761a96b719e0b50c0d977203d841e0fdf5d6dba5f1865710a4c328c",
	"ether-vista-tracker-pin-overrides.rpc":                            "fccecd8f2a4cfbdc14ed4a497a74636f013918925f5cc4bc26994f03f6a587e6",
	"ether-vista-tracker-pin":                                          "a42a6d16928fbc94355bc2cb77ff748b5d0226455d1db7e433f74d1a689e5766",
	"etherfi-vampire-tracker-pin-with-overrides":                       "17d8a4795960fd0ff4bf99caebc147a522b3facac6ff47cd7e01b5fca54e2e82",
	"etherfi-vampire-tracker-pin-with-overrides.rpc":                   "e22a5d4185fe7992b8f9a428c8906e391414b6c1337293339a03e010c0a68bc4",
	"etherfi-vampire-tracker-pin-without-overrides":                    "17d8a4795960fd0ff4bf99caebc147a522b3facac6ff47cd7e01b5fca54e2e82",
	"etherfi-vampire-tracker-pin-without-overrides.rpc":                "1b069a352a64df0b94bf6dbfaee3d3faf4b5d90c618db036781b9738aa19a227",
	"generic-arm-pricable":                                             "3efab9847f534722497f1d415ae1844d18660ce83e7afa3b8cd16127db88fd73",
	"generic-arm-pricable.rpc":                                         "6e2a73f7e5ae15e7de06fe88ebd0c36e48663ec04aac296dc4f787e0f24b3ebc",
	"generic-arm-pricable4626":                                         "592d6394bd7d10b1cabbae6b7ea3f6736a5ba36aea3a182db2ba35d2400f7864",
	"generic-arm-pricable4626.rpc":                                     "9c67951cfa4813fc9780f119a144a591f3e849c391f799fb1d47a3c4134bec14",
	"generic-arm-tracker-pin":                                          "2f7b69d49795ee460f9ee1f7ce0d7fabd752ed4eaed3a08e6c38f053cbbd1d51",
	"generic-arm-tracker-pin.rpc":                                      "4796df05300ca8c3eb78121a4af37160835d8fa37b44bb910f48cd7874276965",
	"generic-arm-withdrawal-direct":                                    "e0030678f005e96237b89e67717de4c3b3db672fd3c3a739df56e71fb6ddd388",
	"generic-arm-withdrawal-direct.rpc":                                "93947629d62bc5ca55ab84a3b19d78ef00e87f7833f32a7f2b75de4368d67e14",
	"generic-arm-withdrawal-shares":                                    "d54e3948aa03ed71bd7421b6b866eeb38d3b6c94c6abf3423bbaa6adbdffb543",
	"generic-arm-withdrawal-shares.rpc":                                "06830b6a269c5066c244c68005f2066a75d6eb6785c42e7975ed362da2c15266",
	"ghost_linear":                                                     "50e15f10c030e5c240832e1a161f1f9a7639b28b18b317c8c0549613d4aec2f6",
	"ghost_linear.rpc":                                                 "3b8f0f288c9f1ba749a397753de1f7ed291987ceeffde58ea4773ea8f404e518",
	"ghost_routing_direct":                                             "50e15f10c030e5c240832e1a161f1f9a7639b28b18b317c8c0549613d4aec2f6",
	"ghost_routing_direct.rpc":                                         "3b8f0f288c9f1ba749a397753de1f7ed291987ceeffde58ea4773ea8f404e518",
	"ghost_routing_fallback":                                           "50e15f10c030e5c240832e1a161f1f9a7639b28b18b317c8c0549613d4aec2f6",
	"ghost_routing_fallback.rpc":                                       "3b8f0f288c9f1ba749a397753de1f7ed291987ceeffde58ea4773ea8f404e518",
	"gohm-tracker-pin":                                                 "b03cd161b241f01b5d4c6c18c564d4feec20e875eb7a2bf0bb151ef0055b3b58",
	"gohm-tracker-pin.rpc":                                             "c700ddf790551e732a23d133934f84d51048f536164bc321a95b294d233d1577",
	"honey-tracker-new-assets":                                         "7b8b4d241f706eedf3ec54536fd279dfc3a31c3502165e3d7863cac5be783129",
	"honey-tracker-new-assets.rpc":                                     "99216f3a4139fd95ea6b417517cd59c43929f82439c9370c8ca6f2b38278f522",
	"honey-tracker-pin":                                                "27a832f5362b0e4a13bacfa054cb990a519583c881b1ce60631fcd9752efc901",
	"honey-tracker-pin.rpc":                                            "5104a0496bed501d18eb150d5e3f8283fa8b1eaacd28ad72bf4c3eab72fa409f",
	"infinifi-gateway-tracker-pin-existing-bucket":                     "611efd2ce4abaa5d5635da9c9958f5796a87e414d80f59691be5c0186564f573",
	"infinifi-gateway-tracker-pin-existing-bucket.rpc":                 "f6de88b42ffbcbd44c1d3b1401b263684e855bbdb55d96a7310cb5de78444f33",
	"infinifi-gateway-tracker-pin-new-bucket":                          "f798425c3d9b5c04fa81b8ef6dc72c7323c7619f28e7c9fa1c3dfdb8f4f28a95",
	"infinifi-gateway-tracker-pin-new-bucket.rpc":                      "15e66ecedef053f8cdb568ab28fd1509d56fa340c5b23d9c07ec9bafabdeac15",
	"kelp-rseth-tracker-pin-with-overrides":                            "30ec0d78d41bab5a33f54c178d0728163c907446c507498d68eb110d31bdd835",
	"kelp-rseth-tracker-pin-with-overrides.rpc":                        "e199d773a7ba990edcb124f6086f6e6c96662fb54aedde03947867bf742f4459",
	"kelp-rseth-tracker-pin-without-overrides":                         "30ec0d78d41bab5a33f54c178d0728163c907446c507498d68eb110d31bdd835",
	"kelp-rseth-tracker-pin-without-overrides.rpc":                     "e79e425de0468fcafea7c64c27b78c8284af69e16e1135156743ca74a55c2093",
	"lfj-poe-tracker-pin":                                              "47d3ed53b09802f31518ab61ec4c5cf56f6f4591cd8d5c5ca2f1ad209bdef077",
	"lfj-poe-tracker-pin.rpc":                                          "71a341539e4a5c990909a0ffe62ef2d8750000b40b212308cbaaa0f139b53850",
	"lista-stable-tracker-pin":                                         "40ae4089e7e4c3ddb3b86b2efe9ce3de51c0b18a3de6ca967b56547913e4324e",
	"lista-stable-tracker-pin.rpc":                                     "75c6027f8295d5da71e04f29ef071171c193299aea47f48272357b3041b2eb0a",
	"savingsdai_TestGetNewPoolStatePinsTimestamp":                      "8c63e1de69e43069c312eb63247406fd5396e719358f3389d6993c8d551b3e75",
	"savingsdai_TestGetNewPoolStatePinsTimestamp.rpc":                  "0437fb2720f3cf30754b364c5e205c1310f76e89956d31b259dbaa4050946196",
	"maverick-v1-overrides":                                            "fed324a103325717570d8bd82504effe67026e75b2f0f3b5afe49968c6c503ca",
	"maverick-v1-overrides.rpc":                                        "c1eec0458fbb5163dd72e9ea3b9bf0e4ec2f76174a29c6d6928904b93a1e4591",
	"maverick-v1-regular":                                              "fed324a103325717570d8bd82504effe67026e75b2f0f3b5afe49968c6c503ca",
	"maverick-v1-regular.rpc":                                          "15c7af7e1bf102e031b9cd766df03184864c538657422cb3ca2c530212df1f5d",
	"maverick-v2-lens-failure":                                         "034c404e20e2c597c60a14434f4de730a3e89515e10ce72fde9dbeecfe0eb1f7",
	"maverick-v2-lens-failure.rpc":                                     "01fbc8495c1037def63e7bb665abe074123b47978bfc8d73740032bfff1cb56e",
	"maverick-v2-overrides":                                            "e2e9d78370958704f9e27bd114cd8d6a3c580ac0568424f254bdc2e078316cbe",
	"maverick-v2-overrides.rpc":                                        "e983b1c32e409ac08abaaed37c0708cc38dee33de5a59784ac1d66093edc4a78",
	"maverick-v2-regular":                                              "e2e9d78370958704f9e27bd114cd8d6a3c580ac0568424f254bdc2e078316cbe",
	"maverick-v2-regular.rpc":                                          "474425d15496151b950196cec559ecb67bcd3e92f36dc3b45189c82c14b97b1c",
	"metronome-swap-round-two":                                         "cf73a5d30e7ab028605829bc367a29e27cb11a4073c1a5e130467c249acb571e",
	"metronome-swap-round-two.rpc":                                     "7f77aa6e1014528ed5581eae7992ba52107e03e8db5092ae025581e3a92e1a74",
	"net-staking-unwrapped-overrides":                                  "002bf0b58831a29dd4c6f44cfde81acf0e0b2ff84ee3db8f2f10007fb308c1a8",
	"net-staking-unwrapped-overrides.rpc":                              "06dd689a7a924386626a2148ee98b53ce2cd259d13c37ae93d5ee43dccb52fb1",
	"net-staking-unwrapped":                                            "002bf0b58831a29dd4c6f44cfde81acf0e0b2ff84ee3db8f2f10007fb308c1a8",
	"net-staking-unwrapped.rpc":                                        "b7fb85e02af062fe5218bf766110a8d72aa777141ba0fc12fa62ccb10de16d74",
	"net-staking-wrapped-overrides":                                    "0e7fdca2c76bb65b3da8a22bd9f0a04991c0fbae7fb5935485c1fa24cb5acb23",
	"net-staking-wrapped-overrides.rpc":                                "ec6986b0bdab24fa7c017e85e871e93458313bece2cedfa8f3c27e95a47551b4",
	"net-staking-wrapped":                                              "0e7fdca2c76bb65b3da8a22bd9f0a04991c0fbae7fb5935485c1fa24cb5acb23",
	"net-staking-wrapped.rpc":                                          "7ad770794d7c290c522c450e48d3c9c29e4b46999e924d195b391a4b78e0e968",
	"virtual-fun-v2_TestGetNewPoolStatePinsTaxReads":                   "9286a451f3ceb8c537640bf055fed954a885a7881e96e3e7df38102bec4ed053",
	"virtual-fun-v2_TestGetNewPoolStatePinsTaxReads.rpc":               "7eea497fe2b0642f1ad017bf6b5ef908538da171892fae05c986c812d7f8a6ef",
	"wildcard-native-balance":                                          "a3fd69555d16df903d95724adeb4ae122146add1f6f2080c6c3e3da2edda354f",
	"wildcard-native-balance.rpc":                                      "7b4169ece2a23bfe36917d25ecc3c7d714a8fad23c5c445133e4c70c1cdede49",
	"wildcard_TestTrackPoolsPinsSamplesForBatch.rpc":                   "0f63aab69f439101f64d867bdf7a92c4f889fbef13dbc8b8f91a6e816173454d",
	"wildcard_TestTrackPoolsPinsSamplesForBatch_0":                     "6a76eccda5afec9b21d7a38cea61d2c0ba00abc2068509916df6f40bb90ae55e",
	"wildcard_TestTrackPoolsPinsSamplesForBatch_1":                     "97aa86185d5aec2f9be83d81e172c784ca4f3a7d477315c28508795e6049cc5b",
	"fulcrom_TestGetNewPoolStateSnapshotEquivalent":                    "f506ce6169b02a4ddd40fea152d4e56e167fdc2f214dd2dec790a456eb9cdf9d",
	"fulcrom_TestGetNewPoolStateSnapshotEquivalent.rpc":                "2917ab5741db3ce29097380495a147195fa10d3110987c3cfc7611c66eb96727",
	"fxdx_TestGetNewPoolStatePinsAllCallsToFirstBlock":                 "fe0f3e783afefa22473de19413259d0f87960072993b431b0cb3c5422458d013",
	"fxdx_TestGetNewPoolStatePinsAllCallsToFirstBlock.rpc":             "0c66335dec41ad400978e2eaf4969b88e841565b2faf179022efb3f595088d6c",
	"gmx-glp_TestGetNewPoolStatePinsAllCallsToFirstBlock":              "24defcb688ef2a81aaa895d6af09814584e8946b0983393b67831f45f01bd2c9",
	"gmx-glp_TestGetNewPoolStatePinsAllCallsToFirstBlock.rpc":          "3e9beee193ff2e9a848eea9d7594491398f4fa4956cc0b32cff247bcd19f868d",
	"gmx-glp_TestGetNewPoolStatePinsAllCallsToFirstBlockV1":            "250657c4647f669f15a6a1f2aa0598ecb2f2322d934b45b6a8c488be8a608472",
	"gmx-glp_TestGetNewPoolStatePinsAllCallsToFirstBlockV1.rpc":        "9d108e09ab6565dd1ee91b634ebf9fbdd9b9e217c9da3d9e5992da39c1487354",
	"gmx_TestGetNewPoolStatePinsAllCallsToFirstBlock":                  "27fdc8183d1b128896a73ff2d4420e6805383da58e02a770038314c6104fb2d3",
	"gmx_TestGetNewPoolStatePinsAllCallsToFirstBlock.rpc":              "0bddf7be39585811a882441b395da100c4c7ba05563e5621d870b9df14d48bf5",
	"gmx_TestGetNewPoolStatePinsAllCallsToFirstBlockV1":                "2c68a873ff9088dca898c690eb3871a59d5baeddd2af77963352ecb88b242882",
	"gmx_TestGetNewPoolStatePinsAllCallsToFirstBlockV1.rpc":            "55c6a39cc865f84e271b6b7740bd065fd84e5a4b750d3e3a6c8ff6efb598457d",
	"gmx_TestGetNewPoolStatePriceFeedType_direct":                      "32feff233b86a1e32efab6d7c88528cda606da2f998d5ff306bc0482d6f7892a",
	"gmx_TestGetNewPoolStatePriceFeedType_direct.rpc":                  "c2f6ae04b05258a8e1b65953397e1d96b9309fbf38caa22773a20287a36ab96d",
	"gmx_TestGetNewPoolStatePriceFeedType_latest_round_answer":         "b88b614d0572fb78df59395c36f24ca1b326961e75923b224b1e3bc1ab18288f",
	"gmx_TestGetNewPoolStatePriceFeedType_latest_round_answer.rpc":     "41b6cb8ac3ef36ad26e50f76d64cbd60221a506c9d6a8f1d7a80c607cdff8a1e",
	"madmex_TestGetNewPoolStatePinsAllCallsToFirstBlock":               "ba3c2c4e702fa77d7eb3f8216469ac46bbbbc8e6497eb7556b67824ffc9a3d18",
	"madmex_TestGetNewPoolStatePinsAllCallsToFirstBlock.rpc":           "d384045fda66e5b587989c857cd8a2f6e778568e756eba74a74ce2debfa942f6",
	"madmex_TestGetNewPoolStatePinsAllCallsToFirstBlockV1":             "59b481913439c36e42f3095305f84705a3f37cc166014bef1d77de98a72edc26",
	"madmex_TestGetNewPoolStatePinsAllCallsToFirstBlockV1.rpc":         "500803538e736fa3cf8b97409448ae04fa8cac4b2344e7f54fd2f97ed270e123",
	"makerpsm_TestGetNewPoolStatePinsVatRead":                          "2a5189e81d9e48c85a49456bb1af059592808a61da2fdfddbd5188e029008b0b",
	"makerpsm_TestGetNewPoolStatePinsVatRead.rpc":                      "543a5aa50691c7816ba6f94532dfb9d5c3b2e453dc8d95a533a7d9b213baa318",
	"metavault_TestGetNewPoolStatePinsAllCallsToFirstBlock":            "fbeacc5ee262eb9980b7f5507f03283d30bf0b50117e91d9dab755b9fd5b4c5e",
	"metavault_TestGetNewPoolStatePinsAllCallsToFirstBlock.rpc":        "1c67346a73b7d310aa645f0e6de171b6b2404acec7af0f6d8ed96e71464df304",
	"oneswap_TestGetNewPoolStatePinsTotalSupply":                       "237afb03150682f7bb14db6996ba7fe4a03422d8d11a7c7bdf5f7e673fb19825",
	"oneswap_TestGetNewPoolStatePinsTotalSupply.rpc":                   "9b5ecdfaaf709299a51899b845f3070122b0b55fe4f3bb08914a12988a79b3fa",
	"quickperps_TestGetNewPoolStatePinsAllCallsToFirstBlock":           "b4f419ce044202a4225a192e6283288d16bb2dc8cabd8a6add0c8ccbf3074a47",
	"quickperps_TestGetNewPoolStatePinsAllCallsToFirstBlock.rpc":       "798cb7b4937267c2595c0d3b484cd10dd083b1203ea5ec193f75f30e6592fc11",
	"quickperps_TestGetNewPoolStatePinsAllCallsToFirstBlockV1":         "6af134893280db90d8944aa8fd6db551d5c0e2b8c0c163214e4852c722d0bab7",
	"quickperps_TestGetNewPoolStatePinsAllCallsToFirstBlockV1.rpc":     "3123868c32e5ddf4863cae61799491ad3c31cd1bcd5d4cc581ad5170e8efeecc",
	"swapbased-perp_TestGetNewPoolStatePinsAllCallsToFirstBlock":       "e644ce0ee285ce0fc55cb303db96d50600dbf7f15a5c5c5830093e090762d9d0",
	"swapbased-perp_TestGetNewPoolStatePinsAllCallsToFirstBlock.rpc":   "7e730143540e8afeab1ea1ad82e5a732f2dc6108a4161c5c7f56f618694d2f52",
	"swapbased-perp_TestGetNewPoolStatePinsAllCallsToFirstBlockV1":     "7ae7fa85480c07d30d91486497d270f41ef1dc9108a55415df0f4d606ff73434",
	"swapbased-perp_TestGetNewPoolStatePinsAllCallsToFirstBlockV1.rpc": "4a269cc6e69b53244cf54a61d264e7a44eacce79539479f3b0252b04364979e5",
}
