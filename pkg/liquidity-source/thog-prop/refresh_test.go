package thogprop

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func lensPayload(t *testing.T) (contractFixture, State, []byte) {
	t.Helper()
	f := fixture(t, "execution-model3.json.gz")
	sample := f.Snapshots[0]
	s := stateFrom(t, sample)
	issuers := make([]common.Address, 0, len(sample.ShareBalances))
	shares, multipliers := []*big.Int{}, []*big.Int{}
	for _, token := range s.Tokens {
		if holding, ok := sample.ShareBalances[token.Address]; ok {
			issuers = append(issuers, common.HexToAddress(token.Address))
			shares = append(shares, holding.Shares.ToBig())
			multipliers = append(multipliers, holding.Multiplier.ToBig())
		}
	}
	payload, err := snapshotLensResult.Pack([4]byte{'T', 'H', 'G', '8'}, new(big.Int).SetUint64(s.BlockNumber), big.NewInt(1793310000), raw(t, sample.Snapshot), issuers, shares, multipliers)
	require.NoError(t, err)
	return f, s, payload
}

func TestSnapshotLensLiveCapture(t *testing.T) {
	compressed, err := os.ReadFile("testdata/lens-monad.json.gz")
	require.NoError(t, err)
	z, err := gzip.NewReader(bytes.NewReader(compressed))
	require.NoError(t, err)
	defer func() { require.NoError(t, z.Close()) }()
	data, err := io.ReadAll(z)
	require.NoError(t, err)
	var capture struct {
		Proxy      string `json:"proxy"`
		LensSha256 string `json:"lensSha256"`
		Response   struct {
			Error struct {
				Data string `json:"data"`
			} `json:"error"`
		} `json:"response"`
	}
	require.NoError(t, json.Unmarshal(data, &capture))
	digest := sha256.Sum256(snapshotLensCode)
	require.Equal(t, hex.EncodeToString(digest[:]), capture.LensSha256)
	s, timestamp, err := decodeSnapshotLens(raw(t, capture.Response.Error.Data), common.HexToAddress(capture.Proxy))
	require.NoError(t, err)
	require.Greater(t, s.BlockNumber, uint64(117_000_000))
	require.Positive(t, timestamp)
	require.False(t, s.BaseFee.IsZero(), "nonzero simulation gas price must preserve BASEFEE")
	require.Len(t, s.Tokens, 13)
	count := 0
	for i, token := range s.Tokens {
		if isXStock(token.Address) {
			count++
			require.False(t, s.Shares[i].Multiplier.IsZero())
		}
	}
	require.Equal(t, 5, count)
	t.Logf("Live lens: block %d, timestamp %d, base fee %s, %d tokens, %d issuers", s.BlockNumber, timestamp, s.BaseFee.Dec(), len(s.Tokens), count)
}

func TestSnapshotLensDecoderRejectsMalformedAndIssuerFailures(t *testing.T) {
	f, _, payload := lensPayload(t)
	for _, data := range [][]byte{nil, payload[:40], append([]byte{0}, payload[1:]...)} {
		_, _, err := decodeSnapshotLens(data, common.HexToAddress(f.Proxy))
		require.Error(t, err)
	}
	failure := snapshotLensABI.Errors["ReadFailed"]
	encoded, err := failure.Inputs.Pack(common.HexToAddress("0xa753a7395cae905cd615da0b82a53e0560f250af"), [4]byte{1, 2, 3, 4}, []byte{5})
	require.NoError(t, err)
	_, _, err = decodeSnapshotLens(append(failure.ID[:4:4], encoded...), common.HexToAddress(f.Proxy))
	require.Error(t, err)
	require.Equal(t, "issuer", classifyRefreshError(err))
}

func TestTrackerUsesOneAtomicCallAndRetriesWholeRead(t *testing.T) {
	f, expected, payload := lensPayload(t)
	for _, mode := range []string{"success", "lag_then_success", "persistent_issuer"} {
		t.Run(mode, func(t *testing.T) {
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				n := calls.Add(1)
				var request rpcRequest
				require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
				require.Equal(t, "eth_call", request.Method)
				require.JSONEq(t, `"latest"`, string(request.Params[1]))
				var call map[string]string
				require.NoError(t, json.Unmarshal(request.Params[0], &call))
				require.NotContains(t, call, "to")
				require.Equal(t, "0x1", call["gasPrice"])
				require.Equal(t, hexutil.EncodeUint64(snapshotLensGas), call["gas"])
				reply := map[string]any{"jsonrpc": "2.0", "id": request.ID}
				switch {
				case mode == "lag_then_success" && n == 1:
					reply["error"] = map[string]any{"code": -32000, "message": "block not found"}
				case mode == "persistent_issuer":
					failure := snapshotLensABI.Errors["ReadFailed"]
					encoded, err := failure.Inputs.Pack(common.HexToAddress("0xa753a7395cae905cd615da0b82a53e0560f250af"), [4]byte{1}, []byte{})
					require.NoError(t, err)
					reply["error"] = map[string]any{"code": 3, "message": "execution reverted", "data": hexutil.Encode(append(failure.ID[:4:4], encoded...))}
				default:
					reply["error"] = map[string]any{"code": 3, "message": "execution reverted", "data": hexutil.Encode(payload)}
				}
				w.Header().Set("Content-Type", "application/json")
				require.NoError(t, json.NewEncoder(w).Encode(reply))
			}))
			defer srv.Close()
			tracker, err := NewPoolTracker(&Config{}, ethrpc.New(srv.URL))
			require.NoError(t, err)
			before := entity.Pool{Address: f.Proxy, Type: DexType, StaticExtra: `{}`}
			after, err := tracker.GetNewPoolState(context.Background(), before, pool.GetNewPoolStateParams{})
			if mode == "persistent_issuer" {
				require.Error(t, err)
				require.Equal(t, before, after)
				require.Equal(t, "issuer", classifyRefreshError(err))
				require.EqualValues(t, 2, calls.Load())
				return
			}
			require.NoError(t, err)
			var extra Extra
			require.NoError(t, json.Unmarshal([]byte(after.Extra), &extra))
			require.Equal(t, expected, extra.State)
			require.Positive(t, extra.CapturedAtMs)
			require.Equal(t, extra.CapturedAtMs/1000, after.Timestamp)
			require.Equal(t, defaultMaxSnapshotAgeMs, extra.MaxSnapshotAgeMs)
			want := int32(1)
			if mode == "lag_then_success" {
				want = 2
			}
			require.Equal(t, want, calls.Load())
		})
	}
}

func TestPageReaderPinsEveryDependentCallAndRejectsReorg(t *testing.T) {
	f := fixture(t, "execution-model3.json.gz")
	sample := f.Snapshots[0]
	expected := stateFrom(t, sample)
	for _, mode := range []string{"success", "reorg", "issuer_error", "lag", "short_issuer", "wrong_balance", "lens_gas"} {
		t.Run(mode, func(t *testing.T) {
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				var data json.RawMessage
				require.NoError(t, json.NewDecoder(r.Body).Decode(&data))
				isBatch := data[0] == '['
				var requests []rpcRequest
				if isBatch {
					require.NoError(t, json.Unmarshal(data, &requests))
				} else {
					requests = make([]rpcRequest, 1)
					require.NoError(t, json.Unmarshal(data, &requests[0]))
				}
				var replies []map[string]any
				for _, request := range requests {
					require.Equal(t, "eth_call", request.Method)
					var call map[string]string
					require.NoError(t, json.Unmarshal(request.Params[0], &call))
					require.Equal(t, "0x1", call["gasPrice"])
					var pin string
					require.NoError(t, json.Unmarshal(request.Params[1], &pin))
					if isBatch {
						require.Equal(t, hexutil.EncodeUint64(expected.BlockNumber), pin)
					} else {
						require.Equal(t, "latest", pin)
					}
					reply := map[string]any{"jsonrpc": "2.0", "id": request.ID}
					if call["to"] == "" {
						require.Equal(t, "lens_gas", mode)
						reply["error"] = map[string]any{"code": -32000, "message": "gas required exceeds allowance"}
						replies = append(replies, reply)
						continue
					}
					if call["to"] == f.Proxy {
						reply["result"] = sample.Snapshot
						if mode == "reorg" && isBatch {
							changed := append([]byte{}, raw(t, sample.Snapshot)...)
							changed[len(changed)-1] ^= 1
							reply["result"] = hexutil.Encode(changed)
						}
					} else {
						holding, ok := sample.ShareBalances[call["to"]]
						require.True(t, ok)
						if call["data"][:10] == "0xf5eb42dc" {
							reply["result"] = hexutil.Encode(encodedWords(holding.Shares))
						} else {
							if mode == "wrong_balance" {
								holding.Multiplier.Lsh(&holding.Multiplier, 1)
							}
							reply["result"] = hexutil.Encode(encodedWords(holding.Multiplier, u(0), u(0)))
						}
						if mode == "short_issuer" {
							reply["result"] = "0x"
						}
						if mode == "issuer_error" || mode == "lag" {
							delete(reply, "result")
							message := "issuer failed"
							if mode == "lag" {
								message = "block not found"
							}
							reply["error"] = map[string]any{"code": -32000, "message": message}
						}
					}
					replies = append(replies, reply)
				}
				w.Header().Set("Content-Type", "application/json")
				if isBatch {
					for i, j := 0, len(replies)-1; i < j; i, j = i+1, j-1 {
						replies[i], replies[j] = replies[j], replies[i]
					}
					require.NoError(t, json.NewEncoder(w).Encode(replies))
				} else {
					require.NoError(t, json.NewEncoder(w).Encode(replies[0]))
				}
			}))
			defer srv.Close()
			tracker, err := NewPoolTracker(&Config{DisableLens: mode != "lens_gas"}, ethrpc.New(srv.URL))
			require.NoError(t, err)
			state, _, err := tracker.readSnapshot(context.Background(), common.HexToAddress(f.Proxy))
			if mode != "success" && mode != "lens_gas" {
				require.Error(t, err)
				if mode == "reorg" {
					require.Equal(t, "reorg", classifyRefreshError(err))
				}
				if mode == "lag" {
					require.Equal(t, "provider_lag", classifyRefreshError(err))
				}
				return
			}
			require.NoError(t, err)
			require.Equal(t, expected, state)
			want := 2
			if mode == "lens_gas" {
				want = 3
			}
			require.EqualValues(t, want, calls.Load())
		})
	}
}

func TestTrackerCancellationAndConfig(t *testing.T) {
	_, err := NewPoolTracker(&Config{RequestTimeoutMs: -1}, nil)
	require.Error(t, err)
	tracker, err := NewPoolTracker(&Config{}, nil)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	before := entity.Pool{Address: "0x1"}
	after, err := tracker.GetNewPoolState(ctx, before, pool.GetNewPoolStateParams{})
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, before, after)
	for _, message := range []string{"gas required exceeds allowance", "out of gas", "contract creation not supported"} {
		require.True(t, lensUnsupported(errors.New(message)))
	}
	require.False(t, lensUnsupported(errors.New("issuer reverted")))
}

func TestTrackerAttemptTimeoutRetriesFromScratch(t *testing.T) {
	f, _, payload := lensPayload(t)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			time.Sleep(250 * time.Millisecond)
			return
		}
		var request rpcRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "error": map[string]any{"code": 3, "message": "execution reverted", "data": hexutil.Encode(payload)}}))
	}))
	defer server.Close()
	tracker, err := NewPoolTracker(&Config{RequestTimeoutMs: 100}, ethrpc.New(server.URL))
	require.NoError(t, err)
	after, err := tracker.GetNewPoolState(context.Background(), entity.Pool{Address: f.Proxy}, pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	require.NotEmpty(t, after.Extra)
	require.EqualValues(t, 2, calls.Load())
}
