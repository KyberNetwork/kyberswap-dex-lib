// Package fakerpc is a tiny JSON-RPC node for tracker regression tests: it answers multicall
// aggregate and direct eth_call from a table, plus eth_blockNumber and eth_getLogs.
package fakerpc

import (
	"encoding/hex"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

const Multicall = "0xca11bde05977b3631167028862be2a173976ca11"

var aggABI = func() abi.ABI {
	a, err := abi.JSON(strings.NewReader(`[{"name":"aggregate","type":"function","inputs":[{"name":"calls","type":"tuple[]",
"components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],
"outputs":[{"name":"blockNumber","type":"uint256"},{"name":"returnData","type":"bytes[]"}]}]`))
	if err != nil {
		panic(err)
	}
	return a
}()

type Server struct {
	Calls map[string][]byte // Key(target, abi, method) -> return data
	Head  uint64            // eth_blockNumber and the block reported by aggregate
	Logs  []ethtypes.Log    // returned for every eth_getLogs
	// Handler, if set and non-nil for a call, overrides Calls (for per-argument answers).
	Handler func(to string, data []byte) []byte

	mu     sync.Mutex
	blocks []string // block tag of every multicall eth_call
	srv    *httptest.Server
}

func New(t *testing.T) *Server {
	s := &Server{Calls: map[string][]byte{}, Head: 1234}
	s.srv = httptest.NewServer(http.HandlerFunc(s.serve))
	t.Cleanup(s.srv.Close)
	return s
}

func (s *Server) Client() *ethrpc.Client {
	return ethrpc.New(s.srv.URL).SetMulticallContract(common.HexToAddress(Multicall))
}

// Blocks returns the block tag of each multicall eth_call so far ("latest" or a hex block).
func (s *Server) Blocks() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.blocks...)
}

func Key(target string, a abi.ABI, method string) string {
	return strings.ToLower(target) + hex.EncodeToString(a.Methods[method].ID)
}

func (s *Server) call(to string, data []byte) []byte {
	if len(data) < 4 {
		return nil
	}
	if to == Multicall {
		args, err := aggABI.Methods["aggregate"].Inputs.Unpack(data[4:])
		if err != nil {
			panic(err)
		}
		var rets [][]byte
		for _, c := range args[0].([]struct {
			Target   common.Address `json:"target"`
			CallData []byte         `json:"callData"`
		}) {
			rets = append(rets, s.call(strings.ToLower(c.Target.Hex()), c.CallData))
		}
		out, err := aggABI.Methods["aggregate"].Outputs.Pack(new(big.Int).SetUint64(s.Head), rets)
		if err != nil {
			panic(err)
		}
		return out
	}
	if s.Handler != nil {
		if out := s.Handler(to, data); out != nil {
			return out
		}
	}
	return s.Calls[to+hex.EncodeToString(data[:4])]
}

func (s *Server) handle(req map[string]any) map[string]any {
	res := map[string]any{"jsonrpc": "2.0", "id": req["id"]}
	params, _ := req["params"].([]any)
	switch req["method"] {
	case "eth_call":
		p := params[0].(map[string]any)
		d := p["data"]
		if d == nil {
			d = p["input"]
		}
		raw, _ := hex.DecodeString(strings.TrimPrefix(d.(string), "0x"))
		to := strings.ToLower(p["to"].(string))
		if to == Multicall {
			tag, _ := params[1].(string)
			s.mu.Lock()
			s.blocks = append(s.blocks, tag)
			s.mu.Unlock()
		}
		res["result"] = "0x" + hex.EncodeToString(s.call(to, raw))
	case "eth_blockNumber":
		res["result"] = "0x" + new(big.Int).SetUint64(s.Head).Text(16)
	case "eth_getLogs":
		logs := s.Logs
		if logs == nil {
			logs = []ethtypes.Log{}
		}
		res["result"] = logs
	default:
		res["result"] = "0x1"
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
