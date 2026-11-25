// Package rpcstub is a JSON-RPC test server that answers tryBlockAndAggregate multicalls for
// a twocrypto-ng pool and its policy, and counts the eth_call roundtrips it serves.
package rpcstub

import (
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

const multicallABIJSON = `[{"name":"tryBlockAndAggregate","type":"function","stateMutability":"payable",
"inputs":[{"name":"requireSuccess","type":"bool"},{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],
"outputs":[{"name":"blockNumber","type":"uint256"},{"name":"blockHash","type":"bytes32"},{"name":"returnData","type":"tuple[]","components":[{"name":"success","type":"bool"},{"name":"returnData","type":"bytes"}]}]}]`

// Handler answers one call: ok=false makes the call revert.
type Handler func(target common.Address, data []byte) (ret []byte, ok bool)

type Stub struct {
	Client *ethrpc.Client
	// Roundtrips counts eth_call requests; CallsPerBatch records the size of each multicall.
	Roundtrips    atomic.Int64
	CallsPerBatch []int
}

// New starts a stub whose multicalls are answered by h.
func New(t *testing.T, h Handler) *Stub {
	mcABI, err := abi.JSON(strings.NewReader(multicallABIJSON))
	require.NoError(t, err)
	s := &Stub{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID     json.RawMessage   `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		if json.Unmarshal(body, &req) != nil {
			http.Error(w, "batch not supported", 400)
			return
		}
		result := "0x"
		if req.Method == "eth_call" {
			s.Roundtrips.Add(1)
			var msg struct {
				Data  hexutil.Bytes `json:"data"`
				Input hexutil.Bytes `json:"input"`
			}
			_ = json.Unmarshal(req.Params[0], &msg)
			if len(msg.Data) == 0 {
				msg.Data = msg.Input
			}
			in, err := mcallInputs(mcABI, msg.Data)
			require.NoError(t, err)
			s.CallsPerBatch = append(s.CallsPerBatch, len(in))
			type item struct {
				Success    bool
				ReturnData []byte
			}
			items := make([]item, len(in))
			for i, c := range in {
				ret, ok := h(c.Target, c.CallData)
				items[i] = item{ok, ret}
			}
			out, err := mcABI.Methods["tryBlockAndAggregate"].Outputs.Pack(big.NewInt(12345), [32]byte{1}, items)
			require.NoError(t, err)
			result = hexutil.Encode(out)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
	}))
	t.Cleanup(srv.Close)
	s.Client = ethrpc.New(srv.URL).SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	return s
}

type mcall struct {
	Target   common.Address
	CallData []byte
}

func mcallInputs(a abi.ABI, data []byte) ([]mcall, error) {
	args, err := a.Methods["tryBlockAndAggregate"].Inputs.Unpack(data[4:])
	if err != nil {
		return nil, err
	}
	var out []mcall
	v := reflect.ValueOf(args[1])
	for i := 0; i < v.Len(); i++ {
		out = append(out, mcall{v.Index(i).FieldByName("Target").Interface().(common.Address),
			v.Index(i).FieldByName("CallData").Interface().([]byte)})
	}
	return out, nil
}

// ZeroReturn packs the zero value of every output of method m of the ABI that owns selector data.
func ZeroReturn(a abi.ABI, data []byte) ([]byte, bool) {
	m, err := a.MethodById(data[:4])
	if err != nil {
		return nil, false
	}
	vals := make([]any, len(m.Outputs))
	for i, o := range m.Outputs {
		vals[i] = zeroOf(o.Type.GetType()).Interface()
	}
	out, err := m.Outputs.Pack(vals...)
	return out, err == nil
}

// zeroOf is reflect.Zero with non-nil *big.Int (nil cannot be ABI-packed), also inside arrays.
func zeroOf(t reflect.Type) reflect.Value {
	switch {
	case t == reflect.TypeOf((*big.Int)(nil)):
		return reflect.ValueOf(new(big.Int))
	case t.Kind() == reflect.Array:
		v := reflect.New(t).Elem()
		for i := 0; i < t.Len(); i++ {
			v.Index(i).Set(zeroOf(t.Elem()))
		}
		return v
	}
	return reflect.Zero(t)
}
