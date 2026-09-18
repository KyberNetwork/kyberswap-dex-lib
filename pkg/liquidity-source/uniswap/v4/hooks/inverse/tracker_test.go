package inverse

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
)

// Exercise real ABI packing/unpacking and the ethrpc transport offline. This
// catches getter-width errors, block drift and missing override propagation.
func TestTrackPinnedSnapshot(t *testing.T) {
	s := vectors(t)[0].Before
	token := common.HexToAddress("0x0000000000000000000000000000000000000010")
	hook := common.HexToAddress("0x0000000000000000000000000000000000002aec")
	poolID := common.HexToHash("0x1234")
	multi, err := abi.JSON(strings.NewReader(`[{"type":"function","name":"aggregate","inputs":[{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],"outputs":[{"name":"blockNumber","type":"uint256"},{"name":"returnData","type":"bytes[]"}]}]`))
	require.NoError(t, err)
	tests := []struct {
		name        string
		override    bool
		fail        bool
		closed      bool
		unsupported bool
		feeWrap     bool
	}{{name: "ordinary"}, {name: "overrides", override: true}, {name: "rpc failure", fail: true}, {name: "closed", closed: true}, {name: "controller changed", unsupported: true}, {name: "fee growth wrap", feeWrap: true}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := s
			if tt.feeWrap {
				s = vectors(t)[0].After
			}
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request struct {
					ID     int               `json:"id"`
					Method string            `json:"method"`
					Params []json.RawMessage `json:"params"`
				}
				require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
				require.Equal(t, "eth_call", request.Method)
				require.Equal(t, `"0x1"`, string(request.Params[1]))
				if tt.override {
					require.Len(t, request.Params, 3)
					require.Contains(t, string(request.Params[2]), "code")
				}
				requests++
				if tt.fail && requests == 2 {
					_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%d,"error":{"code":-32000,"message":"fixture failure"}}`, request.ID)
					return
				}
				var call struct {
					Data  string `json:"data"`
					Input string `json:"input"`
				}
				require.NoError(t, json.Unmarshal(request.Params[0], &call))
				if call.Data == "" {
					call.Data = call.Input
				}
				raw := common.FromHex(call.Data)
				decoded, e := multi.Methods["aggregate"].Inputs.Unpack(raw[4:])
				require.NoError(t, e)
				type rpcCall struct {
					Target   common.Address
					CallData []byte
				}
				calls := *abi.ConvertType(decoded[0], new([]rpcCall)).(*[]rpcCall)
				results := make([][]byte, len(calls))
				for i, c := range calls {
					m, e := contractABI.MethodById(c.CallData[:4])
					require.NoError(t, e)
					var result []any
					switch m.Name {
					case "initialShares", "totalShares":
						result = []any{s.InitialShares.ToBig()}
					case "initialQuote":
						result = []any{s.InitialQuote.ToBig()}
					case "reserveShares":
						result = []any{s.ReserveShares.ToBig()}
					case "reserveQuote":
						result = []any{s.ReserveQuote.ToBig()}
					case "nativeQuote":
						result = []any{s.NativeQuote.ToBig()}
					case "roundingQuote":
						result = []any{s.RoundingQuote.ToBig()}
					case "sequence":
						result = []any{s.Sequence.ToBig()}
					case "liquidity", "getLiquidity":
						result = []any{s.Liquidity.ToBig()}
					case "token":
						result = []any{token}
					case "quoteToken":
						result = []any{quoteAddress}
					case "poolManager":
						result = []any{managerAddress}
					case "poolId":
						result = []any{poolID}
					case "initialized":
						result = []any{true}
					case "closed":
						result = []any{tt.closed}
					case "prepaymentGuard":
						result = []any{false}
					case "phase":
						result = []any{uint8(0)}
					case "protocolFeeController":
						address := controllerAddress
						if tt.unsupported {
							address = common.Address{}
						}
						result = []any{address}
					case "getSlot0":
						result = []any{s.SqrtPriceX96.ToBig(), n(int64(s.Tick)), n(int64(s.ProtocolFee)), n(3000)}
					case "getPositionInfo":
						result = []any{s.Liquidity.ToBig(), n(0), n(0)}
						if tt.feeWrap {
							result[2] = sub(new(big.Int).Lsh(n(1), 256), n(5))
						}
					case "getFeeGrowthInside":
						result = []any{n(0), n(0)}
						if tt.feeWrap {
							result[1] = sub(up(s.Fees1.ToBig(), q128, s.Liquidity.ToBig()), n(5))
						}
					case "indexRay", "workingIndexRay":
						result = []any{s.Index.ToBig()}
					case "custodiedShares":
						result = []any{s.CustodiedShares.ToBig()}
					case "nativeBalance":
						result = []any{s.NativeInverse.ToBig()}
					case "prepaidNominal", "prepaidShares", "protocolFeesAccrued":
						result = []any{n(0)}
					case "market":
						result = []any{hook}
					case "balanceOf":
						result = []any{s.HookQuote.ToBig()}
					default:
						t.Errorf("unexpected method %s", m.Name)
						return
					}
					results[i], e = m.Outputs.Pack(result...)
					require.NoError(t, e)
				}
				packed, e := multi.Methods["aggregate"].Outputs.Pack(n(1), results)
				require.NoError(t, e)
				_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%d,"result":"%s"}`, request.ID, hexutil.Encode(packed))
			}))
			defer server.Close()
			p := &uniswapv4.HookParam{Cfg: &uniswapv4.Config{ChainID: 4663}, RpcClient: ethrpc.New(server.URL).SetMulticallContract(common.HexToAddress("0xca11")), HookAddress: hook, BlockNumber: n(1), Pool: &entity.Pool{Address: poolID.Hex(), Tokens: []*entity.PoolToken{{Address: hexutil.Encode(token[:]), Decimals: 18}, {Address: hexutil.Encode(quoteAddress[:]), Decimals: 18}}, StaticExtra: fmt.Sprintf(`{"fee":3000,"tS":60,"hooks":"%s"}`, hook)}}
			if tt.override {
				p.Overrides = map[common.Address]gethclient.OverrideAccount{hook: {Code: []byte{0}}}
			}
			h := newHook(Extra{})
			data, e := h.Track(context.Background(), p)
			if tt.fail {
				require.Error(t, e)
				require.Equal(t, Extra{}, h.State)
				return
			}
			require.NoError(t, e)
			require.Equal(t, 2, requests)
			require.Equal(t, uint64(1), p.Pool.BlockNumber)
			want := s
			want.Live = !tt.closed
			want.FeeControllerSupported = !tt.unsupported
			require.Equal(t, want, h.State)
			var roundTrip Extra
			require.NoError(t, json.Unmarshal(data, &roundTrip))
			require.Equal(t, want, roundTrip)
			if tt.closed {
				r, e := h.GetReserves(context.Background(), p)
				require.NoError(t, e)
				require.Equal(t, entity.PoolReserves{"0", "0"}, r)
			}
		})
	}
}

func TestTrackRejectsUnpinnedState(t *testing.T) {
	h := newHook(Extra{})
	for _, p := range []*uniswapv4.HookParam{nil, {}, {BlockNumber: n(0)}, {Cfg: &uniswapv4.Config{ChainID: 1}}} {
		_, err := h.Track(context.Background(), p)
		require.ErrorIs(t, err, ErrState)
		require.Equal(t, Extra{}, h.State)
	}
}
