package eth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
)

func TestCallWithBlockRetry(t *testing.T) {
	methods := []string{"Aggregate", "TryAggregate", "TryBlockAndAggregate", "Call"}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				attempts := 0
				result, err := eth.CallWithBlockRetry(context.Background(), func() (string, error) {
					attempts++
					if attempts == 1 {
						return "", errors.New("RPC: UNKNOWN BLOCK 0x3039")
					}
					return method, nil
				})
				if err != nil || result != method || attempts != 2 {
					t.Fatalf("result=%q err=%v attempts=%d", result, err, attempts)
				}
			})
		})
	}
}

func TestCallWithBlockRetryEthRPCMethods(t *testing.T) {
	innerABI, err := abi.JSON(strings.NewReader(`[{"type":"function","name":"value","inputs":[],"outputs":[{"type":"uint256"}]}]`))
	if err != nil {
		t.Fatal(err)
	}
	outerABI, err := abi.JSON(strings.NewReader(`[
		{"type":"function","name":"aggregate","inputs":[{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],"outputs":[{"name":"blockNumber","type":"uint256"},{"name":"returnData","type":"bytes[]"}]},
		{"type":"function","name":"tryAggregate","inputs":[{"name":"requireSuccess","type":"bool"},{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],"outputs":[{"name":"returnData","type":"tuple[]","components":[{"name":"success","type":"bool"},{"name":"returnData","type":"bytes"}]}]},
		{"type":"function","name":"tryBlockAndAggregate","inputs":[{"name":"requireSuccess","type":"bool"},{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],"outputs":[{"name":"blockNumber","type":"uint256"},{"name":"blockHash","type":"bytes32"},{"name":"returnData","type":"tuple[]","components":[{"name":"success","type":"bool"},{"name":"returnData","type":"bytes"}]}]}
	]`))
	if err != nil {
		t.Fatal(err)
	}
	for _, methodName := range []string{"Aggregate", "TryAggregate", "TryBlockAndAggregate", "Call"} {
		t.Run(methodName, func(t *testing.T) {
			var mu sync.Mutex
			var gotTags []string
			var gotCalls []string
			attempt := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				var request struct {
					ID     json.RawMessage   `json:"id"`
					Method string            `json:"method"`
					Params []json.RawMessage `json:"params"`
				}
				if err := json.Unmarshal(body, &request); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				if request.Method != "eth_call" {
					http.Error(w, "unexpected rpc method", http.StatusBadRequest)
					return
				}
				var call struct {
					To    string `json:"to"`
					Input string `json:"input"`
					Data  string `json:"data"`
				}
				if err := json.Unmarshal(request.Params[0], &call); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				input := call.Input
				if input == "" {
					input = call.Data
				}
				var tag string
				_ = json.Unmarshal(request.Params[1], &tag)
				mu.Lock()
				attempt++
				gotTags = append(gotTags, tag)
				gotCalls = append(gotCalls, call.To+":"+input)
				currentAttempt := attempt
				mu.Unlock()
				if currentAttempt == 1 {
					w.Header().Set("Content-Type", "application/json")
					_, _ = io.Copy(w, bytes.NewReader([]byte(fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"error":{"code":-32000,"message":"unknown block 0x3039"}}`, request.ID))))
					return
				}
				data, err := hexutil.Decode(input)
				if err != nil || len(data) < 4 {
					http.Error(w, "invalid calldata", http.StatusBadRequest)
					return
				}
				var result []byte
				if methodName == "Call" {
					result, err = innerABI.Methods["value"].Outputs.Pack(big.NewInt(7))
				} else {
					outerName := strings.ToLower(methodName[:1]) + methodName[1:]
					method := outerABI.Methods[outerName]
					value, packErr := innerABI.Methods["value"].Outputs.Pack(big.NewInt(7))
					if packErr != nil {
						err = packErr
					} else if outerName == "aggregate" {
						result, err = method.Outputs.Pack(big.NewInt(12345), [][]byte{value})
					} else {
						items := []struct {
							Success    bool   `abi:"success"`
							ReturnData []byte `abi:"returnData"`
						}{{Success: true, ReturnData: value}}
						if outerName == "tryAggregate" {
							result, err = method.Outputs.Pack(items)
						} else {
							result, err = method.Outputs.Pack(big.NewInt(12345), [32]byte{}, items)
						}
					}
				}
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				response, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": json.RawMessage(request.ID), "result": hexutil.Encode(result)})
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write(response)
			}))
			defer server.Close()
			client := ethrpc.New(server.URL).SetMulticallContract(common.HexToAddress("0x0000000000000000000000000000000000000005"))
			var output *big.Int
			request := client.NewRequest().SetContext(context.Background()).SetBlockNumber(big.NewInt(12345))
			request.AddCall(&ethrpc.Call{ABI: innerABI, Target: "0x0000000000000000000000000000000000000001", Method: "value"}, []any{&output})
			call := map[string]func() (*ethrpc.Response, error){
				"Aggregate":            request.Aggregate,
				"TryAggregate":         request.TryAggregate,
				"TryBlockAndAggregate": request.TryBlockAndAggregate,
				"Call":                 request.Call,
			}[methodName]
			_, err := eth.CallWithBlockRetry(context.Background(), call)
			if err != nil || output == nil || output.Uint64() != 7 {
				t.Fatalf("output=%v err=%v", output, err)
			}
			if len(gotTags) != 2 || gotTags[0] != "0x3039" || gotTags[1] != "0x3039" {
				t.Fatalf("block tags: %v", gotTags)
			}
			if gotCalls[0] != gotCalls[1] {
				t.Fatalf("retried request changed: %v", gotCalls)
			}
		})
	}
}

func TestCallWithBlockRetryExhaustionAndOrdinaryError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		attempts := 0
		started := time.Now()
		result, err := eth.CallWithBlockRetry(context.Background(), func() (int, error) {
			attempts++
			return 0, errors.New("header not found")
		})
		if result != 0 || err == nil || attempts != 3 || time.Since(started) != 1800*time.Millisecond {
			t.Fatalf("result=%d err=%v attempts=%d elapsed=%s", result, err, attempts, time.Since(started))
		}
	})

	attempts := 0
	wantErr := errors.New("connection reset")
	result, err := eth.CallWithBlockRetry(context.Background(), func() (int, error) {
		attempts++
		return 7, wantErr
	})
	if result != 7 || !errors.Is(err, wantErr) || attempts != 1 {
		t.Fatalf("result=%d err=%v attempts=%d", result, err, attempts)
	}
}

func TestCallWithBlockRetryCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		attempts := 0
		finished := make(chan error, 1)
		go func() {
			_, err := eth.CallWithBlockRetry(ctx, func() (int, error) {
				attempts++
				return 0, fmt.Errorf("unknown block")
			})
			finished <- err
		}()
		synctest.Wait()
		cancel()
		select {
		case err := <-finished:
			if !errors.Is(err, context.Canceled) || attempts != 1 {
				t.Fatalf("err=%v attempts=%d", err, attempts)
			}
		case <-time.After(time.Second):
			t.Fatal("call did not stop after cancellation")
		}
	})
}
