package fulcrom

import (
	"bytes"
	"context"
	stdjson "encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test/ethrpcmock"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCalcAmountOutConcurrentSafe(t *testing.T) {
	t.Parallel()
	type testcase struct {
		name        string
		poolEncoded string
		tokenIn     string
		amountIn    string
		tokenOut    string
	}
	testcases := []testcase{
		{
			name: "swap WETH for USDT",
			poolEncoded: `{
				"address": "0x8c7ef34aa54210c76d6d5e475f43e0c11f876098",
				"type": "fulcrom",
				"timestamp": 1705352300,
				"reserves": [
					"3164844253",
					"407981862705453089405",
					"1488648645459",
					"628292027378",
					"11981305446",
					"261209766075",
					"280620655518",
					"37075925310",
					"9333383502",
					"977067545087"
				],
				"tokens": [
					{
						"address": "0x062e66477faf219f25d27dced647bf57c3107d52",
						"swappable": true
					},
					{
						"address": "0xe44fd7fcb2b1581822d0c862b68222998a0c299a",
						"swappable": true
					},
					{
						"address": "0xc21223249ca28397b4b6541dffaecc539bff0c59",
						"swappable": true
					},
					{
						"address": "0x66e428c3f67a68878562e79a0234c1f83c208770",
						"swappable": true
					},
					{
						"address": "0xb888d8dd1733d72681b30c00ee76bde93ae7aa93",
						"swappable": true
					},
					{
						"address": "0x0e517979c2c1c1522ddb0c73905e0d39b3f990c0",
						"swappable": true
					},
					{
						"address": "0xb9ce0dd29c91e02d4620f57a66700fc5e41d6d15",
						"swappable": true
					},
					{
						"address": "0x9d97be214b68c7051215bb61059b4e299cd792c3",
						"swappable": true
					},
					{
						"address": "0x7589b70abb83427bb7049e08ee9fc6479ccb7a23",
						"swappable": true
					},
					{
						"address": "0xc9de0f3e08162312528ff72559db82590b481800",
						"swappable": true
					}
				],
				"extra": "{\"vault\":{\"hasDynamicFees\":true,\"includeAmmPrice\":false,\"isSwapEnabled\":true,\"stableSwapFeeBasisPoints\":1,\"stableTaxBasisPoints\":5,\"swapFeeBasisPoints\":30,\"taxBasisPoints\":50,\"totalTokenWeights\":100000,\"whitelistedTokens\":[\"0x062e66477faf219f25d27dced647bf57c3107d52\",\"0xe44fd7fcb2b1581822d0c862b68222998a0c299a\",\"0xc21223249ca28397b4b6541dffaecc539bff0c59\",\"0x66e428c3f67a68878562e79a0234c1f83c208770\",\"0xb888d8dd1733d72681b30c00ee76bde93ae7aa93\",\"0x0e517979c2c1c1522ddb0c73905e0d39b3f990c0\",\"0xb9ce0dd29c91e02d4620f57a66700fc5e41d6d15\",\"0x9d97be214b68c7051215bb61059b4e299cd792c3\",\"0x7589b70abb83427bb7049e08ee9fc6479ccb7a23\",\"0xc9de0f3e08162312528ff72559db82590b481800\"],\"poolAmounts\":{\"0x062e66477faf219f25d27dced647bf57c3107d52\":3164844253,\"0x0e517979c2c1c1522ddb0c73905e0d39b3f990c0\":261209766075,\"0x66e428c3f67a68878562e79a0234c1f83c208770\":628292027378,\"0x7589b70abb83427bb7049e08ee9fc6479ccb7a23\":9333383502,\"0x9d97be214b68c7051215bb61059b4e299cd792c3\":37075925310,\"0xb888d8dd1733d72681b30c00ee76bde93ae7aa93\":11981305446,\"0xb9ce0dd29c91e02d4620f57a66700fc5e41d6d15\":280620655518,\"0xc21223249ca28397b4b6541dffaecc539bff0c59\":1488648645459,\"0xc9de0f3e08162312528ff72559db82590b481800\":977067545087,\"0xe44fd7fcb2b1581822d0c862b68222998a0c299a\":407981862705453089405},\"bufferAmounts\":{\"0x062e66477faf219f25d27dced647bf57c3107d52\":1600000000,\"0x0e517979c2c1c1522ddb0c73905e0d39b3f990c0\":160000000000,\"0x66e428c3f67a68878562e79a0234c1f83c208770\":570000000000,\"0x7589b70abb83427bb7049e08ee9fc6479ccb7a23\":6200000000,\"0x9d97be214b68c7051215bb61059b4e299cd792c3\":20000000000,\"0xb888d8dd1733d72681b30c00ee76bde93ae7aa93\":12000000000,\"0xb9ce0dd29c91e02d4620f57a66700fc5e41d6d15\":200000000000,\"0xc21223249ca28397b4b6541dffaecc539bff0c59\":910000000000,\"0xc9de0f3e08162312528ff72559db82590b481800\":590000000000,\"0xe44fd7fcb2b1581822d0c862b68222998a0c299a\":310000000000000000000},\"reservedAmounts\":{\"0x062e66477faf219f25d27dced647bf57c3107d52\":1483801599,\"0x0e517979c2c1c1522ddb0c73905e0d39b3f990c0\":152785893326,\"0x66e428c3f67a68878562e79a0234c1f83c208770\":8530356764,\"0x7589b70abb83427bb7049e08ee9fc6479ccb7a23\":4819564425,\"0x9d97be214b68c7051215bb61059b4e299cd792c3\":7157579282,\"0xb888d8dd1733d72681b30c00ee76bde93ae7aa93\":1181604923,\"0xb9ce0dd29c91e02d4620f57a66700fc5e41d6d15\":132962863475,\"0xc21223249ca28397b4b6541dffaecc539bff0c59\":58591018662,\"0xc9de0f3e08162312528ff72559db82590b481800\":694007455781,\"0xe44fd7fcb2b1581822d0c862b68222998a0c299a\":240424920555819866828},\"tokenDecimals\":{\"0x062e66477faf219f25d27dced647bf57c3107d52\":8,\"0x0e517979c2c1c1522ddb0c73905e0d39b3f990c0\":6,\"0x66e428c3f67a68878562e79a0234c1f83c208770\":6,\"0x7589b70abb83427bb7049e08ee9fc6479ccb7a23\":8,\"0x9d97be214b68c7051215bb61059b4e299cd792c3\":8,\"0xb888d8dd1733d72681b30c00ee76bde93ae7aa93\":6,\"0xb9ce0dd29c91e02d4620f57a66700fc5e41d6d15\":6,\"0xc21223249ca28397b4b6541dffaecc539bff0c59\":6,\"0xc9de0f3e08162312528ff72559db82590b481800\":9,\"0xe44fd7fcb2b1581822d0c862b68222998a0c299a\":18},\"stableTokens\":{\"0x062e66477faf219f25d27dced647bf57c3107d52\":false,\"0x0e517979c2c1c1522ddb0c73905e0d39b3f990c0\":false,\"0x66e428c3f67a68878562e79a0234c1f83c208770\":true,\"0x7589b70abb83427bb7049e08ee9fc6479ccb7a23\":false,\"0x9d97be214b68c7051215bb61059b4e299cd792c3\":false,\"0xb888d8dd1733d72681b30c00ee76bde93ae7aa93\":false,\"0xb9ce0dd29c91e02d4620f57a66700fc5e41d6d15\":false,\"0xc21223249ca28397b4b6541dffaecc539bff0c59\":true,\"0xc9de0f3e08162312528ff72559db82590b481800\":false,\"0xe44fd7fcb2b1581822d0c862b68222998a0c299a\":false},\"usdgAmounts\":{\"0x062e66477faf219f25d27dced647bf57c3107d52\":1269253204177016042299857,\"0x0e517979c2c1c1522ddb0c73905e0d39b3f990c0\":148762964598771913035464,\"0x66e428c3f67a68878562e79a0234c1f83c208770\":628555183346144671484622,\"0x7589b70abb83427bb7049e08ee9fc6479ccb7a23\":22389985595290798631700,\"0x9d97be214b68c7051215bb61059b4e299cd792c3\":25012737783739541468619,\"0xb888d8dd1733d72681b30c00ee76bde93ae7aa93\":113265269274853567141994,\"0xb9ce0dd29c91e02d4620f57a66700fc5e41d6d15\":160062949314803878462094,\"0xc21223249ca28397b4b6541dffaecc539bff0c59\":1469458366089667194649382,\"0xc9de0f3e08162312528ff72559db82590b481800\":84568490519676064583638,\"0xe44fd7fcb2b1581822d0c862b68222998a0c299a\":938836986036312645429339},\"maxUsdgAmounts\":{\"0x062e66477faf219f25d27dced647bf57c3107d52\":1500000000000000000000000,\"0x0e517979c2c1c1522ddb0c73905e0d39b3f990c0\":230000000000000000000000,\"0x66e428c3f67a68878562e79a0234c1f83c208770\":1000000000000000000000000,\"0x7589b70abb83427bb7049e08ee9fc6479ccb7a23\":59000000000000000000000,\"0x9d97be214b68c7051215bb61059b4e299cd792c3\":59000000000000000000000,\"0xb888d8dd1733d72681b30c00ee76bde93ae7aa93\":250000000000000000000000,\"0xb9ce0dd29c91e02d4620f57a66700fc5e41d6d15\":290000000000000000000000,\"0xc21223249ca28397b4b6541dffaecc539bff0c59\":1500000000000000000000000,\"0xc9de0f3e08162312528ff72559db82590b481800\":170000000000000000000000,\"0xe44fd7fcb2b1581822d0c862b68222998a0c299a\":1200000000000000000000000},\"tokenWeights\":{\"0x062e66477faf219f25d27dced647bf57c3107d52\":20000,\"0x0e517979c2c1c1522ddb0c73905e0d39b3f990c0\":3000,\"0x66e428c3f67a68878562e79a0234c1f83c208770\":17000,\"0x7589b70abb83427bb7049e08ee9fc6479ccb7a23\":1000,\"0x9d97be214b68c7051215bb61059b4e299cd792c3\":1000,\"0xb888d8dd1733d72681b30c00ee76bde93ae7aa93\":4000,\"0xb9ce0dd29c91e02d4620f57a66700fc5e41d6d15\":5000,\"0xc21223249ca28397b4b6541dffaecc539bff0c59\":25000,\"0xc9de0f3e08162312528ff72559db82590b481800\":3000,\"0xe44fd7fcb2b1581822d0c862b68222998a0c299a\":21000},\"priceFeed\":{\"minPrices\":{\"0x062e66477faf219f25d27dced647bf57c3107d52\":42858111666670000000000000000000000,\"0x0e517979c2c1c1522ddb0c73905e0d39b3f990c0\":531590000000000000000000000000,\"0x66e428c3f67a68878562e79a0234c1f83c208770\":1000000000000000000000000000000,\"0x7589b70abb83427bb7049e08ee9fc6479ccb7a23\":251272000000000000000000000000000,\"0x9d97be214b68c7051215bb61059b4e299cd792c3\":70023600000000000000000000000000,\"0xb888d8dd1733d72681b30c00ee76bde93ae7aa93\":10259160000000000000000000000000,\"0xb9ce0dd29c91e02d4620f57a66700fc5e41d6d15\":578210000000000000000000000000,\"0xc21223249ca28397b4b6541dffaecc539bff0c59\":1000000000000000000000000000000,\"0xc9de0f3e08162312528ff72559db82590b481800\":95079800000000000000000000000000,\"0xe44fd7fcb2b1581822d0c862b68222998a0c299a\":2526105000000000000000000000000000},\"maxPrices\":{\"0x062e66477faf219f25d27dced647bf57c3107d52\":42858111666670000000000000000000000,\"0x0e517979c2c1c1522ddb0c73905e0d39b3f990c0\":531590000000000000000000000000,\"0x66e428c3f67a68878562e79a0234c1f83c208770\":1000000000000000000000000000000,\"0x7589b70abb83427bb7049e08ee9fc6479ccb7a23\":251272000000000000000000000000000,\"0x9d97be214b68c7051215bb61059b4e299cd792c3\":70023600000000000000000000000000,\"0xb888d8dd1733d72681b30c00ee76bde93ae7aa93\":10259160000000000000000000000000,\"0xb9ce0dd29c91e02d4620f57a66700fc5e41d6d15\":578210000000000000000000000000,\"0xc21223249ca28397b4b6541dffaecc539bff0c59\":1000000000000000000000000000000,\"0xc9de0f3e08162312528ff72559db82590b481800\":95079800000000000000000000000000,\"0xe44fd7fcb2b1581822d0c862b68222998a0c299a\":2526105000000000000000000000000000}},\"usdg\":{\"address\":\"0xB09BD2bAf03e19550473a5DC1D5023805E04a4f5\",\"totalSupply\":4208732677493008283439248},\"UseSwapPricing\":false}}"
			}`,
			tokenIn:  "0xe44fd7fcb2b1581822d0c862b68222998a0c299a", // WETH
			amountIn: "10000000000000000000",                       // 10 WETH
			tokenOut: "0x66e428c3f67a68878562e79a0234c1f83c208770", // USDT
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			poolEntity := new(entity.Pool)
			err := json.Unmarshal([]byte(tc.poolEncoded), poolEntity)
			require.NoError(t, err)

			poolSim, err := NewPoolSimulator(*poolEntity)
			require.NoError(t, err)

			result, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return poolSim.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{
						Token:  tc.tokenIn,
						Amount: bignumber.NewBig10(tc.amountIn),
					},
					TokenOut: tc.tokenOut,
				})
			})
			require.NoError(t, err)
			_ = result
		})
	}
}

func TestGetNewPoolStatePinsAllCallsToFirstBlock(t *testing.T) {
	const snapshotBlock = uint64(12345)
	const vaultAddress = "0x0000000000000000000000000000000000000001"
	const usdgAddress = "0x0000000000000000000000000000000000000002"
	const feedAddress = "0x0000000000000000000000000000000000000003"
	const tokenAddress = "0x0000000000000000000000000000000000000004"

	multicallABI, err := abi.JSON(strings.NewReader(`[
		{"inputs":[{"name":"requireSuccess","type":"bool"},{"components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}],"name":"calls","type":"tuple[]"}],"name":"tryAggregate","outputs":[{"components":[{"name":"success","type":"bool"},{"name":"returnData","type":"bytes"}],"type":"tuple[]"}],"stateMutability":"payable","type":"function"},
		{"inputs":[{"name":"requireSuccess","type":"bool"},{"components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}],"name":"calls","type":"tuple[]"}],"name":"tryBlockAndAggregate","outputs":[{"name":"blockNumber","type":"uint256"},{"name":"blockHash","type":"bytes32"},{"components":[{"name":"success","type":"bool"},{"name":"returnData","type":"bytes"}],"type":"tuple[]"}],"stateMutability":"payable","type":"function"}
	]`))
	if err != nil {
		t.Fatal(err)
	}
	contractABIs := map[common.Address]abi.ABI{
		common.HexToAddress(vaultAddress):                                 vaultABI,
		common.HexToAddress(feedAddress):                                  vaultPriceFeedABI,
		common.HexToAddress(usdgAddress):                                  erc20ABI,
		common.HexToAddress("0x0000000000000000000000000000000000000005"): multicallABI,
	}
	var tags []string
	var firstRequireSuccess bool
	var firstCallFailed bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID     stdjson.RawMessage   `json:"id"`
			Method string               `json:"method"`
			Params []stdjson.RawMessage `json:"params"`
		}
		if err := stdjson.Unmarshal(body, &req); err != nil {
			t.Errorf("decode rpc request: %v", err)
			return
		}
		var result string
		switch req.Method {
		case "eth_chainId":
			result = "0x1"
		case "eth_blockNumber":
			result = "0x3039"
		case "eth_call":
			var call struct {
				To    string `json:"to"`
				Data  string `json:"data"`
				Input string `json:"input"`
			}
			var tag string
			if err := stdjson.Unmarshal(req.Params[0], &call); err != nil {
				t.Errorf("decode eth_call: %v", err)
				return
			}
			_ = stdjson.Unmarshal(req.Params[1], &tag)
			tags = append(tags, tag)
			input, _ := hexutil.Decode(call.Data)
			if len(input) == 0 {
				input, _ = hexutil.Decode(call.Input)
			}
			target := common.HexToAddress(call.To)
			a, ok := contractABIs[target]
			if !ok {
				t.Errorf("unexpected call target %s", call.To)
				return
			}
			method, err := a.MethodById(input[:4])
			if err != nil {
				t.Errorf("unknown method selector %x for %s", input[:4], call.To)
				return
			}
			if target == common.HexToAddress("0x0000000000000000000000000000000000000005") {
				decoded, err := method.Inputs.Unpack(input[4:])
				if err != nil {
					t.Errorf("unpack multicall: %v", err)
					return
				}
				if method.Name == "tryBlockAndAggregate" {
					firstRequireSuccess = decoded[0].(bool)
				}
				calls := decoded[1].([]struct {
					Target   common.Address `json:"target"`
					CallData []byte         `json:"callData"`
				})
				items := make([]struct {
					Success    bool   `abi:"success"`
					ReturnData []byte `abi:"returnData"`
				}, len(calls))
				for i, inner := range calls {
					innerABI, ok := contractABIs[inner.Target]
					if !ok {
						t.Errorf("unexpected inner target %s", inner.Target)
						return
					}
					innerMethod, err := innerABI.MethodById(inner.CallData[:4])
					if err != nil {
						t.Errorf("unknown inner selector %x", inner.CallData[:4])
						return
					}
					values := make([]any, len(innerMethod.Outputs))
					for j, output := range innerMethod.Outputs {
						switch output.Type.T {
						case abi.AddressTy:
							switch innerMethod.Name {
							case vaultMethodUSDG:
								values[j] = common.HexToAddress(usdgAddress)
							case vaultMethodPriceFeed:
								values[j] = common.HexToAddress(feedAddress)
							case vaultMethodWhitelistedTokens:
								values[j] = common.HexToAddress(tokenAddress)
							default:
								values[j] = common.Address{}
							}
						case abi.BoolTy:
							values[j] = true
						case abi.UintTy:
							values[j] = big.NewInt(1)
						default:
							t.Errorf("unsupported output type %s for %s", output.Type, innerMethod.Name)
							return
						}
					}
					items[i].Success = innerMethod.Name != vaultMethodHasDynamicFees
					if !items[i].Success {
						firstCallFailed = true
						continue
					}
					items[i].ReturnData, err = innerMethod.Outputs.Pack(values...)
					if err != nil {
						t.Errorf("pack %s result: %v", innerMethod.Name, err)
						return
					}
				}
				var packed []byte
				if method.Name == "tryAggregate" {
					packed, err = method.Outputs.Pack(items)
				} else {
					packed, err = method.Outputs.Pack(new(big.Int).SetUint64(snapshotBlock), [32]byte{}, items)
				}
				if err != nil {
					t.Errorf("pack multicall result: %v", err)
					return
				}
				result = hexutil.Encode(packed)
			} else {
				method, err := a.MethodById(input[:4])
				if err != nil {
					t.Errorf("unknown direct method: %v", err)
					return
				}
				packed, err := method.Outputs.Pack(big.NewInt(99))
				if err != nil {
					t.Errorf("pack direct response: %v", err)
					return
				}
				result = hexutil.Encode(packed)
			}
		default:
			t.Errorf("unexpected rpc method %s", req.Method)
			return
		}
		response, _ := stdjson.Marshal(map[string]any{"jsonrpc": "2.0", "id": stdjson.RawMessage(req.ID), "result": result})
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.Copy(w, bytes.NewReader(response))
	})
	server := httptest.NewServer(handler)
	defer server.Close()
	client := ethrpc.New(server.URL).SetMulticallContract(common.HexToAddress("0x0000000000000000000000000000000000000005"))
	tracker, err := NewPoolTracker(&Config{DexID: DexTypeFulcrom}, client)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := tracker.GetNewPoolState(context.Background(), entity.Pool{Address: vaultAddress}, pool.GetNewPoolStateParams{})
	if err != nil {
		t.Fatal(err)
	}
	if updated.BlockNumber != snapshotBlock {
		t.Fatalf("BlockNumber = %d, want %d", updated.BlockNumber, snapshotBlock)
	}
	if firstRequireSuccess || !firstCallFailed {
		t.Fatalf("first call failure semantics changed: requireSuccess=%v callFailed=%v", firstRequireSuccess, firstCallFailed)
	}
	if len(updated.Tokens) != 1 || len(updated.Reserves) != 1 || updated.Tokens[0].Address != tokenAddress || updated.Reserves[0] != "1" {
		t.Fatalf("unexpected pool state: tokens=%+v reserves=%v", updated.Tokens, updated.Reserves)
	}

	if len(tags) < 5 || tags[0] != "latest" {
		t.Fatalf("unexpected eth_call block tags: %v", tags)
	}
	for i, tag := range tags[1:] {
		if tag != "0x3039" {
			t.Fatalf("eth_call %d block tag = %s, want 0x3039; all tags: %v", i+1, tag, tags)
		}
	}
}

func TestGetNewPoolStateSnapshotEquivalent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		vaultAddress := common.HexToAddress("0x1000000000000000000000000000000000000001")
		usdgAddress := common.HexToAddress("0x2000000000000000000000000000000000000002")
		feedAddress := common.HexToAddress("0x3000000000000000000000000000000000000003")
		tokenAddress := common.HexToAddress("0x4000000000000000000000000000000000000004")
		resolver := func(_ common.Address, method abi.Method, inputs []any) ([]any, error) {
			switch method.Name {
			case vaultMethodUSDG:
				return []any{usdgAddress}, nil
			case vaultMethodPriceFeed:
				return []any{feedAddress}, nil
			case vaultMethodWhitelistedTokenCount:
				return []any{big.NewInt(1)}, nil
			case vaultMethodWhitelistedTokens:
				return []any{tokenAddress}, nil
			}
			values := make([]any, len(method.Outputs))
			for i, output := range method.Outputs {
				values[i] = ethrpcmock.Value(output.Type)
			}
			return values, nil
		}
		server, recorder := ethrpcmock.NewWithResolver(t, resolver, vaultABI, vaultPriceFeedABI, erc20ABI)
		client := ethrpc.New(server.URL).SetMulticallContract(common.HexToAddress("0x5000000000000000000000000000000000000005"))
		tracker, err := NewPoolTracker(&Config{DexID: DexTypeFulcrom}, client)
		require.NoError(t, err)
		updated, err := tracker.GetNewPoolState(context.Background(), entity.Pool{Address: vaultAddress.Hex(), Exchange: DexTypeFulcrom, Type: DexTypeFulcrom, Extra: "{}"}, pool.GetNewPoolStateParams{})
		require.NoError(t, err)

		require.Equal(t, uint64(ethrpcmock.Block), updated.BlockNumber)

		require.Len(t, updated.Tokens, 1)
		require.Equal(t, hexutil.Encode(tokenAddress[:]), updated.Tokens[0].Address)
		require.Equal(t, entity.PoolReserves{"1"}, updated.Reserves)
		ethrpcmock.CheckPool(t, "fulcrom_TestGetNewPoolStateSnapshotEquivalent", updated)
		ethrpcmock.CheckRequests(t, "fulcrom_TestGetNewPoolStateSnapshotEquivalent", recorder, [2]string{"tryAggregate", "tryBlockAndAggregate"})
		ethrpcmock.RequireRequestsPinned(t, recorder, 3)
	})
}
