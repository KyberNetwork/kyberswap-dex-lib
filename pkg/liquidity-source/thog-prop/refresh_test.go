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
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	abipkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
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

func TestMulticallFallbackReadsOneStateAtLatest(t *testing.T) {
	f := fixture(t, "execution-model3.json.gz")
	sample := f.Snapshots[0]
	expected := stateFrom(t, sample)
	proxy := common.HexToAddress(f.Proxy)
	pageData, err := thogAMMABI.Pack(methodGetPoolData, uint8(0), uint8(maxTokens))
	require.NoError(t, err)
	sharesData, err := thogAMMABI.Pack("sharesOf", proxy)
	require.NoError(t, err)
	multiplierData, err := thogAMMABI.Pack("getCurrentMultiplier")
	require.NoError(t, err)
	method := abipkg.Multicall3ABI.Methods[abipkg.Multicall3TryAggregate]
	const blockTime = 1793310000
	for _, mode := range []string{"success", "lens_gas", "unlisted_issuer_reverts", "page_reverts", "issuer_reverts",
		"short_issuer", "wrong_balance", "zero_base_fee", "no_timestamp", "short_reply", "lag"} {
		t.Run(mode, func(t *testing.T) {
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				var request rpcRequest
				require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
				require.Equal(t, "eth_call", request.Method)
				require.JSONEq(t, `"latest"`, string(request.Params[1]))
				var call map[string]string
				require.NoError(t, json.Unmarshal(request.Params[0], &call))
				require.Equal(t, "0x1", call["gasPrice"])
				reply := map[string]any{"jsonrpc": "2.0", "id": request.ID}
				w.Header().Set("Content-Type", "application/json")
				if call["to"] == "" {
					require.Equal(t, "lens_gas", mode)
					reply["error"] = map[string]any{"code": -32000, "message": "user-specified gas exceeds provider limit"}
					require.NoError(t, json.NewEncoder(w).Encode(reply))
					return
				}
				require.Equal(t, strings.ToLower(multicall3), call["to"])
				if mode == "lag" {
					reply["error"] = map[string]any{"code": -32000, "message": "block not found"}
					require.NoError(t, json.NewEncoder(w).Encode(reply))
					return
				}
				in, err := method.Inputs.Unpack(hexutil.MustDecode(call["data"])[4:])
				require.NoError(t, err)
				require.Equal(t, false, in[0]) // requireSuccess
				requested := *abi.ConvertType(in[1], new([]tryCall)).(*[]tryCall)
				require.Len(t, requested, 2+2*len(xStocks))
				require.Equal(t, tryCall{Target: proxy, CallData: pageData}, requested[0])
				require.Equal(t, tryCall{Target: common.HexToAddress(multicall3), CallData: common.FromHex(selectorGetCurrentBlockTimestamp)}, requested[1])
				page := raw(t, sample.Snapshot)
				if mode == "zero_base_fee" {
					clear(page[3*32 : 4*32]) // tuple head: offset, schema, block number, base fee
				}
				results := []tryResult{{Success: mode != "page_reverts", ReturnData: page},
					{Success: mode != "no_timestamp", ReturnData: encodedWords(u(blockTime))}}
				listed := false
				for i, token := range xStocks {
					require.Equal(t, tryCall{Target: common.HexToAddress(token), CallData: sharesData}, requested[2+2*i])
					require.Equal(t, tryCall{Target: common.HexToAddress(token), CallData: multiplierData}, requested[3+2*i])
					holding, ok := sample.ShareBalances[token]
					if !ok {
						results = append(results, tryResult{Success: mode != "unlisted_issuer_reverts"}, tryResult{})
						continue
					}
					if mode == "wrong_balance" {
						holding.Multiplier.Lsh(&holding.Multiplier, 1)
					}
					sharesResult := tryResult{Success: true, ReturnData: encodedWords(holding.Shares)}
					if !listed && mode == "issuer_reverts" {
						sharesResult = tryResult{}
					}
					if !listed && mode == "short_issuer" {
						sharesResult.ReturnData = nil
					}
					listed = true
					results = append(results, sharesResult, tryResult{Success: true, ReturnData: encodedWords(holding.Multiplier, u(0), u(0))})
				}
				if mode == "short_reply" {
					results = results[:len(results)-1]
				}
				out, err := method.Outputs.Pack(results)
				require.NoError(t, err)
				reply["result"] = hexutil.Encode(out)
				require.NoError(t, json.NewEncoder(w).Encode(reply))
			}))
			defer srv.Close()
			tracker, err := NewPoolTracker(&Config{DisableLens: mode != "lens_gas"}, ethrpc.New(srv.URL))
			require.NoError(t, err)
			state, timestamp, err := tracker.readSnapshot(context.Background(), proxy)
			want := int32(1)
			if mode == "lens_gas" {
				want = 2
			}
			require.Equal(t, want, calls.Load())
			switch mode {
			case "success", "lens_gas", "unlisted_issuer_reverts":
				require.NoError(t, err)
				require.Equal(t, expected, state)
				require.EqualValues(t, blockTime, timestamp)
				return
			}
			require.Error(t, err)
			kind := map[string]string{"page_reverts": "contract", "issuer_reverts": "issuer", "lag": "provider_lag"}[mode]
			if kind == "" {
				kind = "validation"
			}
			require.Equal(t, kind, classifyRefreshError(err))
		})
	}
}

func TestIssuerCatalogMatchesQuotePathAndLens(t *testing.T) {
	require.Len(t, xStocks, 9)
	seen := map[string]bool{}
	for _, token := range xStocks {
		require.True(t, isXStock(token), token)
		require.False(t, seen[token], token)
		seen[token] = true
		// The lens hard-codes the same catalog as address literals.
		require.True(t, bytes.Contains(snapshotLensCode, common.FromHex(token)), "lens bytecode lacks %s", token)
	}
	for _, file := range []string{"execution-model3.json.gz", "deployed-model3.json.gz", "one-block-model3.json.gz"} {
		for _, sample := range fixture(t, file).Snapshots {
			for _, token := range fixturePage(t, sample).Tokens {
				address := hexutil.Encode(token.Token[:])
				require.Equal(t, seen[address], isXStock(address), address)
			}
		}
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

func TestLensCapabilityRefusalsPreserveContractAndTransportErrors(t *testing.T) {
	for _, message := range []string{
		"user-specified gas exceeds provider limit", "gas required exceeds allowance",
		"requested gas limit is too high", "out of gas", "gas exceeds the gas cap",
		"contract creation is not supported", "unsupported creation eth_call", "creation is disabled",
	} {
		require.True(t, lensUnsupported(errors.New(message)), message)
	}
	for _, err := range []error{
		context.Canceled, context.DeadlineExceeded, errors.New("connection reset by peer"),
		errors.New("HTTP 429 rate limit"), errors.New("HTTP 403 forbidden"),
		refreshFailure("issuer", "lens read", errors.New("out of gas")),
		refreshFailure("contract", "lens read", errors.New("gas limit exceeded")),
		refreshFailure("validation", "lens result", errors.New("unsupported creation payload")),
	} {
		require.False(t, lensUnsupported(err), err.Error())
	}
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
