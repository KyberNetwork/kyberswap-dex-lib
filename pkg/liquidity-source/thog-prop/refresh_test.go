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
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	abipkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

// Opt-in read-only check against an explicitly supplied RPC. Pin both real reader
// calls to the same canonical block so changing prices cannot mask a mismatch.
func TestLiveSnapshotReadersAgree(t *testing.T) {
	url := os.Getenv("THOG_PROP_READER_RPC_URL")
	if url == "" {
		t.Skip("set THOG_PROP_READER_RPC_URL to compare both readers on chain")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	upstream, err := rpc.DialContext(ctx, url)
	require.NoError(t, err)
	defer upstream.Close()
	var header struct {
		Hash   common.Hash    `json:"hash"`
		Number hexutil.Uint64 `json:"number"`
	}
	require.NoError(t, upstream.CallContext(ctx, &header, "eth_getBlockByNumber", "latest", false))
	require.NotZero(t, header.Hash)
	block := rpc.BlockNumberOrHashWithHash(header.Hash, true)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request rpcRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		require.Equal(t, "eth_call", request.Method)
		require.Len(t, request.Params, 2)
		var result json.RawMessage
		err := upstream.CallContext(ctx, &result, request.Method, request.Params[0], block)
		reply := map[string]any{"jsonrpc": "2.0", "id": request.ID}
		if err == nil {
			reply["result"] = result
		} else {
			var rpcErr rpc.Error
			require.ErrorAs(t, err, &rpcErr)
			failure := map[string]any{"code": rpcErr.ErrorCode(), "message": err.Error()}
			var dataErr rpc.DataError
			if errors.As(err, &dataErr) {
				failure["data"] = dataErr.ErrorData()
			}
			reply["error"] = failure
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(reply))
	}))
	defer srv.Close()
	client, err := rpc.DialContext(ctx, srv.URL)
	require.NoError(t, err)
	defer client.Close()
	proxy := common.HexToAddress(verifyProxy)
	initCode, err := lensInitCode(proxy)
	require.NoError(t, err)
	lens, lensTime, err := readSnapshotLens(ctx, client, proxy, initCode)
	require.NoError(t, err)
	// With no token list yet, the Multicall3 reader first learns the pool's list
	// and then reads it; both reads are pinned to the same block here.
	tracker, err := NewPoolTracker(&Config{DisableLens: true}, ethrpc.New(srv.URL))
	require.NoError(t, err)
	multi, multiTime, err := tracker.readSnapshot(ctx, proxy, nil)
	require.NoError(t, err)
	require.Equal(t, uint64(header.Number), lens.BlockNumber)
	require.Equal(t, lens, multi)
	require.Equal(t, lensTime, multiTime)
	shareBased := 0
	for i := range lens.Tokens {
		if lens.shareBased(i) {
			shareBased++
		}
	}
	t.Logf("lens and Multicall3 agree at block %d with %d tokens, %d share-based", lens.BlockNumber, len(lens.Tokens), shareBased)
}

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
	compressed, err := os.ReadFile("testdata/lens-stored-multipliers.json.gz")
	require.NoError(t, err)
	z, err := gzip.NewReader(bytes.NewReader(compressed))
	require.NoError(t, err)
	defer func() { require.NoError(t, z.Close()) }()
	data, err := io.ReadAll(z)
	require.NoError(t, err)
	type lensResponse struct {
		Error struct {
			Data string `json:"data"`
		} `json:"error"`
	}
	var capture struct {
		Proxy                 string       `json:"proxy"`
		LensSha256            string       `json:"lensSha256"`
		Response              lensResponse `json:"response"`
		IssuerGetterIsolation struct {
			Overrides map[string]struct {
				Code string `json:"code"`
			} `json:"overrides"`
			Response lensResponse `json:"response"`
		} `json:"issuerGetterIsolation"`
	}
	require.NoError(t, json.Unmarshal(data, &capture))
	digest := sha256.Sum256(snapshotLensCode)
	require.Equal(t, hex.EncodeToString(digest[:]), capture.LensSha256)
	s, timestamp, err := decodeSnapshotLens(raw(t, capture.Response.Error.Data), common.HexToAddress(capture.Proxy))
	require.NoError(t, err)
	require.Greater(t, s.BlockNumber, uint64(117_000_000))
	require.Positive(t, timestamp)
	require.False(t, s.BaseFee.IsZero(), "nonzero simulation gas price must preserve BASEFEE")
	require.Len(t, s.Tokens, 17)
	// The lens found the share-based tokens by asking the pool; the second response
	// was read with each of them overridden to answer sharesOf alone.
	shareBased := 0
	for i, token := range s.Tokens {
		_, overridden := capture.IssuerGetterIsolation.Overrides[token.Address]
		require.Equal(t, overridden, s.shareBased(i), token.Address)
		if s.shareBased(i) {
			shareBased++
		}
	}
	require.Equal(t, 9, shareBased)
	require.Len(t, capture.IssuerGetterIsolation.Overrides, shareBased)
	require.Equal(t, capture.Response.Error.Data, capture.IssuerGetterIsolation.Response.Error.Data,
		"a token that answers only sharesOf must give the same snapshot")
	t.Logf("Live lens: block %d, timestamp %d, base fee %s, %d tokens, %d share-based", s.BlockNumber, timestamp, s.BaseFee.Dec(), len(s.Tokens), shareBased)
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

func TestMulticallReaderReadsOneStateAtLatest(t *testing.T) {
	for _, file := range []string{"execution-model3.json.gz", "deployed-stored-multipliers.json.gz", "deployed-18.json.gz"} {
		t.Run(file, func(t *testing.T) { testMulticallReadsSnapshot(t, file) })
	}
}

func testMulticallReadsSnapshot(t *testing.T, file string) {
	f := fixture(t, file)
	sample := f.Snapshots[0]
	expected := stateFrom(t, sample)
	proxy := common.HexToAddress(f.Proxy)
	listed := pageTokens(t, sample)
	pageData, err := thogAMMABI.Pack(methodGetPoolData, uint8(0), uint8(maxTokens))
	require.NoError(t, err)
	sharesData, err := thogAMMABI.Pack("sharesOf", proxy)
	require.NoError(t, err)
	method := abipkg.Multicall3ABI.Methods[abipkg.Multicall3TryAggregate]
	const blockTime = 1793310000
	for _, mode := range []string{"success", "lens_gas", "first_read", "relisted", "page_reverts", "shares_revert",
		"short_shares", "multiplier_reverts", "multiplier_abi", "wrong_balance", "zero_base_fee", "no_timestamp", "short_reply", "lag"} {
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
				require.GreaterOrEqual(t, len(requested), multicallFixedCalls)
				require.Zero(t, (len(requested)-multicallFixedCalls)%2)
				require.Equal(t, tryCall{Target: proxy, CallData: pageData}, requested[0])
				require.Equal(t, tryCall{Target: common.HexToAddress(multicall3), CallData: common.FromHex(selectorGetCurrentBlockTimestamp)}, requested[1])
				page := raw(t, sample.Snapshot)
				if mode == "zero_base_fee" {
					clear(page[3*32 : 4*32]) // tuple head: offset, schema, block number, base fee
				}
				results := []tryResult{{Success: mode != "page_reverts", ReturnData: page},
					{Success: mode != "no_timestamp", ReturnData: encodedWords(u(blockTime))}}
				mutated := false
				for i := multicallFixedCalls; i < len(requested); i += 2 {
					token := requested[i+1].Target
					multiplierData, err := thogAMMABI.Pack("storedMultiplier", token)
					require.NoError(t, err)
					require.Equal(t, tryCall{Target: proxy, CallData: multiplierData}, requested[i])
					require.Equal(t, tryCall{Target: token, CallData: sharesData}, requested[i+1])
					holding, ok := sample.ShareBalances[hexutil.Encode(token[:])]
					if !ok {
						// An ordinary token has a zero stored multiplier. Its sharesOf reverts or,
						// through a permissive fallback, succeeds with no data; it is never read.
						results = append(results, tryResult{Success: true, ReturnData: encodedWords(u(0))}, tryResult{Success: i%4 == 0})
						continue
					}
					if mode == "wrong_balance" {
						holding.Multiplier.Lsh(&holding.Multiplier, 1)
					}
					multiplierResult := tryResult{Success: true, ReturnData: encodedWords(holding.Multiplier)}
					sharesResult := tryResult{Success: true, ReturnData: encodedWords(holding.Shares)}
					if !mutated {
						switch mode {
						case "shares_revert":
							sharesResult = tryResult{}
						case "short_shares":
							sharesResult.ReturnData = nil
						case "multiplier_reverts":
							multiplierResult.Success = false
						case "multiplier_abi":
							multiplierResult.ReturnData = encodedWords(holding.Multiplier, u(0), u(0))
						}
					}
					mutated = true
					results = append(results, multiplierResult, sharesResult)
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
			tokens, want := listed, int32(1)
			switch mode {
			case "lens_gas":
				want = 2 // the refused lens, then one batch
			case "first_read":
				tokens, want = nil, 2 // one batch to learn the pool's list, one to read it
			case "relisted":
				tokens, want = slices.Clone(listed[:len(listed)-1]), 2 // the pool appended a token
			}
			state, timestamp, err := tracker.readSnapshot(context.Background(), proxy, tokens)
			require.Equal(t, want, calls.Load())
			switch mode {
			case "success", "lens_gas", "first_read", "relisted":
				require.NoError(t, err)
				require.Equal(t, expected, state)
				require.EqualValues(t, blockTime, timestamp)
				return
			}
			require.Error(t, err)
			kind := map[string]string{"page_reverts": "contract", "shares_revert": "issuer", "multiplier_reverts": "contract", "lag": "provider_lag"}[mode]
			if kind == "" {
				kind = "validation"
			}
			require.Equal(t, kind, classifyRefreshError(err))
		})
	}
}

// No token address is compiled into the adapter: the pool's stored multiplier
// alone decides which listed tokens are share-based.
func TestShareBasedTokensComeFromStoredMultipliers(t *testing.T) {
	for _, file := range []string{"execution-model3.json.gz", "deployed-model3.json.gz", "one-block-model3.json.gz", "deployed-stored-multipliers.json.gz", "deployed-18.json.gz"} {
		for _, sample := range fixture(t, file).Snapshots {
			s := stateFrom(t, sample)
			for i, token := range s.Tokens {
				_, held := sample.ShareBalances[token.Address]
				require.Equal(t, held, s.shareBased(i), token.Address)
				require.False(t, bytes.Contains(snapshotLensCode, common.FromHex(token.Address)), "lens bytecode names %s", token.Address)
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
