package lunarbase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const verifiedPoolAddress = "0x00007904d186680c709519e71f4dc3e2df8f1b99"

type verifiedRequest struct {
	Method string
	Params []json.RawMessage
}

type verifiedFixture struct {
	expectedTarget  common.Address
	header, recheck *types.Header
	items           []ethrpc.TryAggregateResultItem
	aggregateBlock  uint64
	requests        []verifiedRequest
	failMethod      string
	outerMalformed  bool
}

type verifiedTransport func(*http.Request) (*http.Response, error)

func (f verifiedTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func verifiedWord(v *big.Int) []byte { return common.LeftPadBytes(v.Bytes(), 32) }
func verifiedWords(values ...*big.Int) []byte {
	var out []byte
	for _, v := range values {
		out = append(out, verifiedWord(v)...)
	}
	return out
}

func newVerifiedFixture() *verifiedFixture {
	h := &types.Header{Number: big.NewInt(900), Difficulty: big.NewInt(1), GasLimit: 30000000, Time: 1700000000, Extra: []byte{1}}
	anchor := new(big.Int).Lsh(big.NewInt(1), 96)
	data := [][]byte{
		common.LeftPadBytes(common.HexToAddress("0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c").Bytes(), 32),
		common.LeftPadBytes(common.HexToAddress("0x55d398326f99059ff775485246999027b3197955").Bytes(), 32),
		verifiedWord(big.NewInt(25)), verifiedWord(big.NewInt(0)), verifiedWord(big.NewInt(125000)),
		verifiedWord(big.NewInt(1230000000)), verifiedWord(big.NewInt(4560000000)), verifiedWord(big.NewInt(0)),
		verifiedWords(anchor, big.NewInt(15), big.NewInt(25), big.NewInt(899)), verifiedWord(anchor),
	}
	f := &verifiedFixture{expectedTarget: common.HexToAddress(verifiedPoolAddress), header: h, recheck: h, aggregateBlock: 900}
	for i, b := range data {
		f.items = append(f.items, ethrpc.TryAggregateResultItem{Success: i != 3, ReturnData: b})
	}
	return f
}

func (f *verifiedFixture) client(t *testing.T) *ethrpc.Client {
	t.Helper()
	outer, err := abi.JSON(strings.NewReader(`[{"name":"tryBlockAndAggregate","type":"function","inputs":[{"name":"requireSuccess","type":"bool"},{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],"outputs":[{"name":"blockNumber","type":"uint256"},{"name":"blockHash","type":"bytes32"},{"name":"returnData","type":"tuple[]","components":[{"name":"success","type":"bool"},{"name":"returnData","type":"bytes"}]}]}]`))
	if err != nil {
		t.Fatal(err)
	}
	transport := verifiedTransport(func(r *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		var req struct {
			ID     json.RawMessage   `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		if err = json.Unmarshal(body, &req); err != nil {
			return nil, err
		}
		f.requests = append(f.requests, verifiedRequest{req.Method, req.Params})
		response := map[string]any{"jsonrpc": "2.0", "id": req.ID}
		if req.Method == f.failMethod {
			response["error"] = map[string]any{"code": -32000, "message": "intentional fixture failure"}
		} else {
			switch req.Method {
			case "eth_getBlockByNumber":
				var selector string
				if err = json.Unmarshal(req.Params[0], &selector); err != nil {
					return nil, err
				}
				if selector == "latest" {
					response["result"] = f.header
				} else {
					response["result"] = f.recheck
				}
			case "eth_call":
				var call map[string]json.RawMessage
				if err = json.Unmarshal(req.Params[0], &call); err != nil {
					return nil, err
				}
				var data hexutil.Bytes
				field := call["input"]
				if field == nil {
					field = call["data"]
				}
				if err = json.Unmarshal(field, &data); err != nil {
					return nil, err
				}
				if len(data) < 4 || !bytes.Equal(data[:4], outer.Methods["tryBlockAndAggregate"].ID) {
					return nil, fmt.Errorf("unexpected multicall selector")
				}
				decoded, err := outer.Methods["tryBlockAndAggregate"].Inputs.Unpack(data[4:])
				if err != nil {
					return nil, err
				}
				calls := *abi.ConvertType(decoded[1], new([]ethrpc.MultiCallParam)).(*[]ethrpc.MultiCallParam)
				if len(calls) != 10 {
					return nil, fmt.Errorf("want ten getters, got %d", len(calls))
				}
				for _, c := range calls {
					if c.Target != f.expectedTarget {
						return nil, fmt.Errorf("cross-pool getter target: %s", c.Target)
					}
				}
				encoded, err := outer.Methods["tryBlockAndAggregate"].Outputs.Pack(new(big.Int).SetUint64(f.aggregateBlock), [32]byte{}, f.items)
				if err != nil {
					return nil, err
				}
				if f.outerMalformed {
					encoded = []byte{1}
				}
				response["result"] = hexutil.Encode(encoded)
			default:
				return nil, fmt.Errorf("unexpected RPC method %s", req.Method)
			}
		}
		encoded, err := json.Marshal(response)
		if err != nil {
			return nil, err
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(bytes.NewReader(encoded)), Request: r}, nil
	})
	rpcClient, err := rpc.DialOptions(context.Background(), "http://verified-snapshot.invalid", rpc.WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(rpcClient.Close)
	return ethrpc.NewWithClient(ethclient.NewClient(rpcClient)).SetMulticallContract(common.HexToAddress("0xca11bde05977b3631167028862be2a173976ca11"))
}

func verifiedEntity() entity.Pool {
	return entity.Pool{Address: verifiedPoolAddress, BlockNumber: 800, Extra: "{}", StaticExtra: "{}", Reserves: entity.PoolReserves{"1", "2"}, Tokens: []*entity.PoolToken{{Address: "0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c"}, {Address: "0x55d398326f99059ff775485246999027b3197955"}}}
}

func TestVerifiedSnapshotPinsAndRecordsHash(t *testing.T) {
	for _, withOverrides := range []bool{false, true} {
		t.Run(fmt.Sprintf("overrides-%t", withOverrides), func(t *testing.T) {
			f := newVerifiedFixture()
			var overrides map[common.Address]gethclient.OverrideAccount
			if withOverrides {
				overrides = map[common.Address]gethclient.OverrideAccount{common.HexToAddress(verifiedPoolAddress): {Balance: big.NewInt(1)}}
			}
			state, err := fetchRPCState(context.Background(), verifiedPoolAddress, valueobject.ChainIDBSC, f.client(t), overrides)
			if err != nil {
				t.Fatal(err)
			}
			if state.blockNumber != 900 || state.extra.BlockHash != f.header.Hash().Hex() {
				t.Fatalf("wrong snapshot identity: block=%d hash=%s", state.blockNumber, state.extra.BlockHash)
			}
			if len(f.requests) != 3 || f.requests[0].Method != "eth_getBlockByNumber" || f.requests[1].Method != "eth_call" || f.requests[2].Method != "eth_getBlockByNumber" {
				t.Fatalf("wrong RPC budget or order: %+v", f.requests)
			}
			if withOverrides {
				if string(f.requests[1].Params[1]) != `"0x384"` || len(f.requests[1].Params) != 3 {
					t.Fatalf("overrides must use pinned number and explicit override object: %+v", f.requests[1])
				}
			} else {
				var pin struct {
					BlockHash string `json:"blockHash"`
				}
				if err = json.Unmarshal(f.requests[1].Params[1], &pin); err != nil {
					t.Fatal(err)
				}
				if pin.BlockHash != f.header.Hash().Hex() {
					t.Fatalf("call did not pin header hash: %s", pin.BlockHash)
				}
			}
			if string(f.requests[2].Params[0]) != `"0x384"` {
				t.Fatal("canonical check did not address selected height")
			}
			p := verifiedEntity()
			got, err := buildEntityPool(&p, state)
			if err != nil {
				t.Fatal(err)
			}
			if got.Reserves[0] != "1230000000" || got.Reserves[1] != "4560000000" {
				t.Fatal("snapshot reserves differ")
			}
		})
	}
}

func TestVerifiedSnapshotRejectsBrokenResults(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*verifiedFixture)
	}{
		{"required_revert", func(f *verifiedFixture) { f.items[2].Success = false }},
		{"required_malformed_swallowed_by_ethrpc", func(f *verifiedFixture) { f.items[2].ReturnData = []byte{1} }},
		{"optional_success_malformed", func(f *verifiedFixture) { f.items[4].ReturnData = []byte{1} }},
		{"state_malformed", func(f *verifiedFixture) { f.items[8].ReturnData = []byte{1} }},
		{"missing_result", func(f *verifiedFixture) { f.items = f.items[:9] }},
		{"extra_result", func(f *verifiedFixture) { f.items = append(f.items, f.items[0]) }},
		{"outer_malformed", func(f *verifiedFixture) { f.outerMalformed = true }},
		{"neither_model", func(f *verifiedFixture) { f.items[4].Success = false }},
		{"both_models", func(f *verifiedFixture) { f.items[3].Success = true }},
		{"number_mismatch", func(f *verifiedFixture) { f.aggregateBlock = 901 }},
		{"anchor_disagreement", func(f *verifiedFixture) { f.items[9].ReturnData = verifiedWord(big.NewInt(999)) }},
		{"zero_delay", func(f *verifiedFixture) { f.items[2].ReturnData = verifiedWord(big.NewInt(0)) }},
		{"future_update_block", func(f *verifiedFixture) {
			f.items[8].ReturnData = verifiedWords(new(big.Int).Lsh(big.NewInt(1), 96), big.NewInt(15), big.NewInt(25), big.NewInt(901))
		}},
		{"identical_tokens", func(f *verifiedFixture) { f.items[1].ReturnData = f.items[0].ReturnData }},
		{"wrapped_token_alias", func(f *verifiedFixture) {
			f.items[1].ReturnData = f.items[0].ReturnData
			f.items[0].ReturnData = make([]byte, 32)
		}},
		{"max_out_of_uint24_range", func(f *verifiedFixture) { f.items[4].ReturnData = verifiedWord(new(big.Int).Lsh(big.NewInt(1), 24)) }},
		{"reserve_out_of_uint112_range", func(f *verifiedFixture) { f.items[5].ReturnData = verifiedWord(new(big.Int).Lsh(big.NewInt(1), 112)) }},
		{"noncanonical_address", func(f *verifiedFixture) {
			f.items[0].ReturnData = append([]byte{}, f.items[0].ReturnData...)
			f.items[0].ReturnData[0] = 1
		}},
		{"trailing_getter_data", func(f *verifiedFixture) { f.items[2].ReturnData = append(f.items[2].ReturnData, make([]byte, 32)...) }},
		{"same_height_replacement", func(f *verifiedFixture) {
			copyHeader := *f.header
			copyHeader.Extra = []byte{2}
			f.recheck = &copyHeader
		}},
		{"aggregate_rpc_failure", func(f *verifiedFixture) { f.failMethod = "eth_call" }},
		{"header_rpc_failure", func(f *verifiedFixture) { f.failMethod = "eth_getBlockByNumber" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newVerifiedFixture()
			tc.mutate(f)
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("bad RPC data caused panic: %v", r)
				}
			}()
			state, err := fetchRPCState(context.Background(), verifiedPoolAddress, valueobject.ChainIDBSC, f.client(t), nil)
			if err == nil || state != nil {
				t.Fatalf("bad snapshot was accepted: state=%+v err=%v", state, err)
			}
		})
	}
}

func TestVerifiedSnapshotAllowsValidEdgeStates(t *testing.T) {
	for _, name := range []string{"zero_max", "zero_reserves", "paused", "legacy_model", "legacy_zero_model", "zero_price", "native_token"} {
		t.Run(name, func(t *testing.T) {
			f := newVerifiedFixture()
			p := verifiedEntity()
			switch name {
			case "zero_max":
				f.items[4].ReturnData = verifiedWord(big.NewInt(0))
			case "zero_reserves":
				f.items[5].ReturnData = verifiedWord(big.NewInt(0))
				f.items[6].ReturnData = verifiedWord(big.NewInt(0))
			case "paused":
				f.items[7].ReturnData = verifiedWord(big.NewInt(1))
			case "legacy_model":
				f.items[3].Success = true
				f.items[3].ReturnData = verifiedWord(big.NewInt(4096))
				f.items[4].Success = false
			case "legacy_zero_model":
				f.items[3].Success = true
				f.items[3].ReturnData = verifiedWord(big.NewInt(0))
				f.items[4].Success = false
			case "zero_price":
				f.items[8].ReturnData = verifiedWords(big.NewInt(0), big.NewInt(15), big.NewInt(25), big.NewInt(899))
				f.items[9].ReturnData = verifiedWord(big.NewInt(0))
			case "native_token":
				f.items[0].ReturnData = make([]byte, 32)
				p.StaticExtra = `{"n":true}`
			}
			state, err := fetchRPCState(context.Background(), verifiedPoolAddress, valueobject.ChainIDBSC, f.client(t), nil)
			if err != nil {
				t.Fatal(err)
			}
			if state.extra.ConcentrationModel != (name == "legacy_model" || name == "legacy_zero_model") {
				t.Fatal("pricing model identity was lost, particularly for a zero model parameter")
			}
			if _, err = buildEntityPool(&p, state); err != nil {
				t.Fatal(err)
			}
			if name == "paused" || name == "zero_price" || name == "zero_reserves" {
				sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p, ChainID: valueobject.ChainIDBSC})
				if err != nil {
					t.Fatal(err)
				}
				_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: p.Tokens[0].Address, Amount: big.NewInt(1000)}, TokenOut: p.Tokens[1].Address})
				if err == nil {
					t.Fatal("disabled or empty snapshot yielded an executable quote")
				}
			}
		})
	}
}

func TestVerifiedSnapshotRejectsWrongEntityIdentity(t *testing.T) {
	for _, name := range []string{"swapped_tokens", "foreign_token", "missing_token", "native_metadata"} {
		t.Run(name, func(t *testing.T) {
			f := newVerifiedFixture()
			state, err := fetchRPCState(context.Background(), verifiedPoolAddress, valueobject.ChainIDBSC, f.client(t), nil)
			if err != nil {
				t.Fatal(err)
			}
			p := verifiedEntity()
			switch name {
			case "swapped_tokens":
				p.Tokens[0], p.Tokens[1] = p.Tokens[1], p.Tokens[0]
			case "foreign_token":
				p.Tokens[0].Address = "0x0000000000000000000000000000000000000001"
			case "missing_token":
				p.Tokens = p.Tokens[:1]
			case "native_metadata":
				p.StaticExtra = `{"n":true}`
			}
			before, _ := json.Marshal(p)
			if _, err = buildEntityPool(&p, state); err == nil {
				t.Fatal("mismatched entity accepted")
			}
			after, _ := json.Marshal(p)
			if !bytes.Equal(before, after) {
				t.Fatal("failed validation mutated caller entity")
			}
		})
	}
}
