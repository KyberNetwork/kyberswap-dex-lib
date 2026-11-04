// Package rpcreplay serves a tracker's multicall batches from a fixture of single calls, for tracker tests that
// replay a pool at one block. A fixture maps each (target, calldata) to its return data, or to a revert; a call the
// fixture does not hold answers as a revert, so the tracker sees what a node gives for a failing call.
//
// With Record set to a node URL, a call the fixture does not hold is sent to that node at the fixture's block and
// added; Save writes the fixture back.
package rpcreplay

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

const multicall = "0xca11bde05977b3631167028862be2a173976ca11"

var multicallABI = func() abi.ABI {
	a, err := abi.JSON(strings.NewReader(`[
{"name":"aggregate","type":"function","inputs":[{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],
 "outputs":[{"name":"blockNumber","type":"uint256"},{"name":"returnData","type":"bytes[]"}]},
{"name":"tryAggregate","type":"function","inputs":[{"name":"requireSuccess","type":"bool"},{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],
 "outputs":[{"name":"returnData","type":"tuple[]","components":[{"name":"success","type":"bool"},{"name":"returnData","type":"bytes"}]}]},
{"name":"tryBlockAndAggregate","type":"function","inputs":[{"name":"requireSuccess","type":"bool"},{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],
 "outputs":[{"name":"blockNumber","type":"uint256"},{"name":"blockHash","type":"bytes32"},{"name":"returnData","type":"tuple[]","components":[{"name":"success","type":"bool"},{"name":"returnData","type":"bytes"}]}]}]`))
	if err != nil {
		panic(err)
	}
	return a
}()

type subCall = struct {
	Target   common.Address `json:"target"`
	CallData []byte         `json:"callData"`
}

type result struct {
	Success    bool   `json:"success"`
	ReturnData []byte `json:"returnData"`
}

// Fixture is the state of one chain at one block, as the answers to single calls.
type Fixture struct {
	Block uint64             `json:"block"`
	Calls map[string]*string `json:"calls"` // "target:calldata" (lower-case hex) -> return data, null for a revert
}

type Server struct {
	t      *testing.T
	file   string
	record string
	mu     sync.Mutex
	fx     Fixture
	srv    *httptest.Server
}

// New serves file. With record non-empty, unknown calls go to that node at the fixture's block.
func New(t *testing.T, file, record string) *Server {
	t.Helper()
	s := &Server{t: t, file: file, record: record, fx: Fixture{Calls: map[string]*string{}}}
	if b, err := os.ReadFile(file); err == nil {
		if err := json.Unmarshal(b, &s.fx); err != nil {
			t.Fatalf("fixture %s: %v", file, err)
		}
	} else if record == "" {
		t.Fatalf("fixture %s: %v", file, err)
	}
	if record != "" && s.fx.Block == 0 {
		s.fx.Block = s.head()
	}
	s.srv = httptest.NewServer(http.HandlerFunc(s.serve))
	t.Cleanup(s.srv.Close)
	return s
}

func (s *Server) Client() *ethrpc.Client {
	return ethrpc.New(s.srv.URL).SetMulticallContract(common.HexToAddress(multicall))
}

// Save writes the fixture, keys sorted, when recording.
func (s *Server) Save() {
	if s.record == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	keys := make([]string, 0, len(s.fx.Calls))
	for k := range s.fx.Calls {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "{\n  \"block\": %d,\n  \"calls\": {", s.fx.Block)
	for i, k := range keys {
		v, _ := json.Marshal(s.fx.Calls[k])
		sep := ","
		if i == len(keys)-1 {
			sep = ""
		}
		fmt.Fprintf(&buf, "\n    %q: %s%s", k, v, sep)
	}
	buf.WriteString("\n  }\n}\n")
	if err := os.WriteFile(s.file, buf.Bytes(), 0o644); err != nil {
		s.t.Error(err)
	}
}

func (s *Server) answer(target common.Address, data []byte) result {
	key := strings.ToLower(target.Hex()) + ":" + hex.EncodeToString(data)
	s.mu.Lock()
	v, ok := s.fx.Calls[key]
	s.mu.Unlock()
	if !ok && s.record != "" {
		v = s.fetch(target, data)
		s.mu.Lock()
		s.fx.Calls[key] = v
		s.mu.Unlock()
	}
	if v == nil {
		return result{}
	}
	out, err := hexutil.Decode(*v)
	if err != nil {
		s.t.Errorf("fixture %s: %v", key, err)
	}
	return result{Success: true, ReturnData: out}
}

func (s *Server) head() uint64 {
	resp, err := http.Post(s.record, "application/json",
		strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}`))
	if err != nil {
		s.t.Fatalf("record: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	var r struct {
		Result hexutil.Uint64 `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		s.t.Fatalf("record: %v", err)
	}
	return uint64(r.Result)
}

func (s *Server) fetch(target common.Address, data []byte) *string {
	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "eth_call",
		"params": []any{map[string]string{"to": target.Hex(), "data": hexutil.Encode(data)},
			hexutil.EncodeUint64(s.fx.Block)}})
	resp, err := http.Post(s.record, "application/json", bytes.NewReader(body))
	if err != nil {
		s.t.Fatalf("record: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	var r struct {
		Result *string         `json:"result"`
		Error  json.RawMessage `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		s.t.Fatalf("record: %v", err)
	}
	if r.Error != nil {
		return nil
	}
	return r.Result
}

func (s *Server) multicall(data []byte) ([]byte, error) {
	m, err := multicallABI.MethodById(data[:4])
	if err != nil {
		return nil, err
	}
	args, err := m.Inputs.Unpack(data[4:])
	if err != nil {
		return nil, err
	}
	calls := *abi.ConvertType(args[len(args)-1], new([]subCall)).(*[]subCall)
	results := make([]result, len(calls))
	for i, c := range calls {
		results[i] = s.answer(c.Target, c.CallData)
	}
	switch m.Name {
	case "aggregate":
		returnData := make([][]byte, len(results))
		for i, r := range results {
			if !r.Success {
				return nil, fmt.Errorf("execution reverted: call %d", i)
			}
			returnData[i] = r.ReturnData
		}
		return m.Outputs.Pack(new(big.Int).SetUint64(s.fx.Block), returnData)
	case "tryAggregate":
		return m.Outputs.Pack(results)
	}
	return m.Outputs.Pack(new(big.Int).SetUint64(s.fx.Block), common.Hash{}, results)
}

func (s *Server) handle(req map[string]any) map[string]any {
	res := map[string]any{"jsonrpc": "2.0", "id": req["id"]}
	params, _ := req["params"].([]any)
	switch req["method"] {
	case "eth_call":
		p, _ := params[0].(map[string]any)
		d, _ := p["data"].(string)
		if d == "" {
			d, _ = p["input"].(string)
		}
		to, _ := p["to"].(string)
		data, err := hexutil.Decode(d)
		if err != nil || len(data) < 4 {
			res["error"] = map[string]any{"code": -32602, "message": "bad call data"}
			return res
		}
		var out []byte
		if strings.EqualFold(to, multicall) {
			out, err = s.multicall(data)
		} else if r := s.answer(common.HexToAddress(to), data); r.Success {
			out = r.ReturnData
		} else {
			err = fmt.Errorf("execution reverted")
		}
		if err != nil {
			res["error"] = map[string]any{"code": 3, "message": err.Error()}
			return res
		}
		res["result"] = hexutil.Encode(out)
	case "eth_blockNumber":
		res["result"] = hexutil.EncodeUint64(s.fx.Block)
	default:
		s.t.Errorf("rpcreplay: unexpected method %v", req["method"])
		res["error"] = map[string]any{"code": -32601, "message": "unsupported"}
	}
	return res
}

func (s *Server) serve(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	w.Header().Set("Content-Type", "application/json")
	var batch []map[string]any
	if json.Unmarshal(body, &batch) == nil {
		outs := make([]map[string]any, 0, len(batch))
		for _, q := range batch {
			outs = append(outs, s.handle(q))
		}
		_ = json.NewEncoder(w).Encode(outs)
		return
	}
	var one map[string]any
	_ = json.Unmarshal(body, &one)
	_ = json.NewEncoder(w).Encode(s.handle(one))
}
