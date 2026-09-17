package prmfun

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const testHead = uint64(65000000)

var multicall3 = common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")

const routerAddress = "0x08A59435c8359A45F4F5dC8D91DF893Cc33DaF29"
const factoryAddress = "0x34fb85aF7588dB97Fd6db6508Aa387D0f088564c"
const prmToken = "0xf24f8f6b08fe87cf062e833a732ad7f636064bc8"
const stockToken = "0x64e8bee350c0ba7c7601f7eca7f61ab965148b51"

type mockCall struct {
	Target   common.Address
	CallData []byte
}
type rpcRequest struct {
	ID     json.RawMessage   `json:"id"`
	Method string            `json:"method"`
	Params []json.RawMessage `json:"params"`
}

// A real ABI-encoding RPC stub catches tuple decoding, required-call failures,
// cursor loss and accidental use of the L1 block returned by Multicall.
func mockClient(t *testing.T, answer func(common.Address, *abi.Method, []any) ([]any, bool)) *ethrpc.Client {
	t.Helper()
	multi, err := abi.JSON(strings.NewReader(`[{"type":"function","name":"aggregate","inputs":[{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],"outputs":[{"name":"blockNumber","type":"uint256"},{"name":"returnData","type":"bytes[]"}]}]`))
	require.NoError(t, err)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			return
		}
		var result any
		switch req.Method {
		case "eth_blockNumber":
			result = hexutil.EncodeUint64(testHead)
		case "eth_call":
			var call struct {
				Input string `json:"input"`
				Data  string `json:"data"`
			}
			if err := json.Unmarshal(req.Params[0], &call); err != nil {
				t.Error(err)
				return
			}
			var tag string
			if err := json.Unmarshal(req.Params[1], &tag); err != nil {
				t.Error(err)
				return
			}
			if tag != hexutil.EncodeUint64(testHead) {
				t.Errorf("unexpected block %s", tag)
				return
			}
			input := call.Input
			if input == "" {
				input = call.Data
			}
			data, err := hexutil.Decode(input)
			if err != nil {
				t.Error(err)
				return
			}
			args, err := multi.Methods["aggregate"].Inputs.Unpack(data[4:])
			if err != nil {
				t.Error(err)
				return
			}
			calls := *abi.ConvertType(args[0], new([]mockCall)).(*[]mockCall)
			outputs := make([][]byte, len(calls))
			for i, c := range calls {
				contract := memeCurveABI
				if c.Target == common.HexToAddress(factoryAddress) {
					contract = memeFactoryABI
				}
				method, err := contract.MethodById(c.CallData[:4])
				if err != nil {
					t.Error(err)
					return
				}
				args, err := method.Inputs.Unpack(c.CallData[4:])
				if err != nil {
					t.Error(err)
					return
				}
				values, ok := answer(c.Target, method, args)
				if !ok {
					_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "error": map[string]any{"code": 3, "message": "required getter reverted"}})
					return
				}
				outputs[i], err = method.Outputs.Pack(values...)
				if err != nil {
					t.Error(err)
					return
				}
			}
			// Deliberately an L1 block, different from eth_blockNumber.
			encoded, err := multi.Methods["aggregate"].Outputs.Pack(big.NewInt(24000000), outputs)
			if err != nil {
				t.Error(err)
				return
			}
			result = hexutil.Encode(encoded)
		default:
			t.Errorf("unexpected method %s", req.Method)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
	}))
	t.Cleanup(server.Close)
	return ethrpc.New(server.URL).SetMulticallContract(multicall3)
}

func testPool(t *testing.T) entity.Pool {
	t.Helper()
	s := newTestSimulator(t)
	static, err := json.Marshal(StaticExtra{RouterAddress: routerAddress, CurveAddress: s.curveAddress, MemeToken: memeToken, GraduationDesk: s.graduationDesk.Dec(), IsNativeQuote: true})
	require.NoError(t, err)
	return entity.Pool{Address: s.curveAddress, Exchange: DexType, Type: DexType, Tokens: []*entity.PoolToken{{Address: testDeskToken}, {Address: memeToken}}, Reserves: []string{"0", "0"}, StaticExtra: string(static)}
}

func TestTrackerSnapshotsAndLifecycle(t *testing.T) {
	for _, phase := range []uint8{PhaseTrading, PhasePaused, PhaseGraduated} {
		t.Run(string(rune('0'+phase)), func(t *testing.T) {
			s := newTestSimulator(t)
			client := mockClient(t, func(_ common.Address, m *abi.Method, _ []any) ([]any, bool) {
				switch m.Name {
				case "phase":
					return []any{phase}, true
				case "getReserves":
					return []any{s.virtualDesk.ToBig(), s.virtualMeme.ToBig()}, true
				case "memeSold":
					return []any{s.memeSold.ToBig()}, true
				case "deskRaised":
					return []any{s.deskRaised.ToBig()}, true
				case "deskToken":
					return []any{common.HexToAddress(testDeskToken)}, true
				case "isNativeQuote":
					return []any{true}, true
				}
				return nil, false
			})
			tracker, err := NewPoolTracker(&Config{ChainId: 4663}, client)
			require.NoError(t, err)
			p, err := tracker.GetNewPoolState(context.Background(), testPool(t), pool.GetNewPoolStateParams{})
			require.NoError(t, err)
			require.Equal(t, testHead, p.BlockNumber)
			if phase == PhaseGraduated {
				require.EqualValues(t, 1, p.Timestamp)
			} else {
				require.Greater(t, p.Timestamp, int64(1))
			}
			if phase == PhaseTrading {
				require.Equal(t, s.virtualDesk.Dec(), p.Reserves[0])
			} else {
				require.Equal(t, entity.PoolReserves{"0", "0"}, p.Reserves)
			}
			sim, err := NewPoolSimulator(p)
			require.NoError(t, err)
			require.Equal(t, testHead, sim.GetMetaInfo(testDeskToken, memeToken).(PoolMeta).BlockNumber)
		})
	}
}

func TestTrackerRequiredReadFailurePreservesPool(t *testing.T) {
	client := mockClient(t, func(_ common.Address, _ *abi.Method, _ []any) ([]any, bool) { return nil, false })
	tracker, err := NewPoolTracker(&Config{}, client)
	require.NoError(t, err)
	before := testPool(t)
	after, err := tracker.GetNewPoolState(context.Background(), before, pool.GetNewPoolStateParams{})
	require.Error(t, err)
	require.Equal(t, before, after)
}
