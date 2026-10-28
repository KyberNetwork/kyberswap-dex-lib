package thogprop

import (
	"context"
	"encoding/json"
	"maps"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type rpcRequest struct {
	ID     json.RawMessage   `json:"id"`
	Method string            `json:"method"`
	Params []json.RawMessage `json:"params"`
}

func TestTrackerPinsPoolAndIssuerStateToOneCanonicalHash(t *testing.T) {
	f := fixture(t, "execution-model3.json.gz")
	sample := f.Snapshots[0]
	expected := stateFrom(t, sample)
	page := fixturePage(t, sample)
	data, err := thogAMMABI.Pack(methodGetPoolData, uint8(0), uint8(maxTokens))
	require.NoError(t, err)
	sharesData, err := thogAMMABI.Pack("sharesOf", common.HexToAddress(f.Proxy))
	require.NoError(t, err)
	multiplierData, err := thogAMMABI.Pack("getCurrentMultiplier")
	require.NoError(t, err)
	hash := common.HexToHash("0x1234")
	for _, mode := range []string{"success", "issuer_error", "missing_issuer", "short_issuer", "zero_multiplier", "wrong_balance", "wrong_block", "wrong_base_fee", "wrong_parent", "reorg", "missing_header"} {
		t.Run(mode, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				var payload json.RawMessage
				require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
				batch := len(payload) > 0 && payload[0] == '['
				var requests []rpcRequest
				if batch {
					require.NoError(t, json.Unmarshal(payload, &requests))
					require.Len(t, requests, 2*len(sample.ShareBalances))
				} else {
					requests = make([]rpcRequest, 1)
					require.NoError(t, json.Unmarshal(payload, &requests[0]))
				}
				responses := make([]map[string]any, 0, len(requests))
				for _, request := range requests {
					response := map[string]any{"jsonrpc": "2.0", "id": request.ID}
					if request.Method == "eth_getBlockByNumber" {
						require.JSONEq(t, `"latest"`, string(request.Params[0]))
						header := map[string]any{"number": hexutil.EncodeUint64(expected.BlockNumber), "hash": hash, "parentHash": common.Hash(page.ParentBlockHash), "baseFeePerGas": expected.BaseFee.Hex()}
						switch mode {
						case "wrong_block":
							header["number"] = hexutil.EncodeUint64(expected.BlockNumber + 1)
						case "wrong_base_fee":
							header["baseFeePerGas"] = "0x0"
						case "wrong_parent":
							header["parentHash"] = common.Hash{}
						}
						response["result"] = header
						if mode == "missing_header" {
							response["result"] = nil
						}
					} else {
						require.Equal(t, "eth_call", request.Method)
						var call map[string]string
						require.NoError(t, json.Unmarshal(request.Params[0], &call))
						require.Equal(t, "0x1", call["gasPrice"])
						var pinned struct {
							BlockHash        common.Hash `json:"blockHash"`
							RequireCanonical bool        `json:"requireCanonical"`
						}
						require.NoError(t, json.Unmarshal(request.Params[1], &pinned))
						require.Equal(t, hash, pinned.BlockHash)
						require.True(t, pinned.RequireCanonical)
						if call["to"] == f.Proxy {
							require.False(t, batch)
							require.Equal(t, hexutil.Encode(data), call["data"])
							response["result"] = sample.Snapshot
							if mode == "reorg" {
								delete(response, "result")
								response["error"] = map[string]any{"code": -32000, "message": "hash is not canonical"}
							}
						} else {
							require.True(t, batch)
							holding, ok := sample.ShareBalances[call["to"]]
							require.True(t, ok)
							switch call["data"] {
							case hexutil.Encode(sharesData):
								response["result"] = hexutil.Encode(encodedWords(holding.Shares))
							case hexutil.Encode(multiplierData):
								if mode == "zero_multiplier" {
									holding.Multiplier.Clear()
								}
								if mode == "wrong_balance" {
									holding.Multiplier.Lsh(&holding.Multiplier, 1)
								}
								response["result"] = hexutil.Encode(encodedWords(holding.Multiplier, u(0), u(0)))
							default:
								t.Errorf("unexpected issuer calldata %s", call["data"])
							}
							if mode == "short_issuer" {
								response["result"] = "0x"
							}
							if mode == "issuer_error" {
								delete(response, "result")
								response["error"] = map[string]any{"code": -32000, "message": "issuer read failed"}
							}
						}
					}
					responses = append(responses, response)
				}
				w.Header().Set("Content-Type", "application/json")
				if batch {
					if mode == "missing_issuer" {
						responses = responses[1:]
					}
					// Batch order is unrelated to call order.
					for i, j := 0, len(responses)-1; i < j; i, j = i+1, j-1 {
						responses[i], responses[j] = responses[j], responses[i]
					}
					require.NoError(t, json.NewEncoder(w).Encode(responses))
				} else {
					require.NoError(t, json.NewEncoder(w).Encode(responses[0]))
				}
			}))
			defer server.Close()
			tracker, err := NewPoolTracker(&Config{Proxy: f.Proxy}, ethrpc.New(server.URL))
			require.NoError(t, err)
			before := entity.Pool{Address: f.Proxy, Type: DexType, StaticExtra: `{"poolId":"0xce389e78282dedac7b18ba7f775b7602d2ab3ab171bbd6711eb0239be6ef4dcc"}`}
			after, err := tracker.GetNewPoolState(context.Background(), before, pool.GetNewPoolStateParams{})
			if mode != "success" {
				require.Error(t, err)
				require.Equal(t, before, after)
				return
			}
			require.NoError(t, err)
			require.EqualValues(t, 3, calls.Load())
			sim, err := NewPoolSimulator(after)
			require.NoError(t, err)
			require.Equal(t, expected, sim.state)
			require.Equal(t, expected.BlockNumber, sim.Info.BlockNumber)
			require.Equal(t, expected.BlockNumber, sim.GetMetaInfo("", "").(pool.MetaInfo).BlockNumber)
			require.Empty(t, before.Extra)
			require.Empty(t, before.Tokens)
		})
	}
}
func TestSnapshotRejectsIncompleteOrIncompatibleImages(t *testing.T) {
	f := fixture(t, "execution-model3.json.gz")
	sample := f.Snapshots[0]
	for _, mutate := range []func(*poolData){
		func(p *poolData) { p.SchemaVersion = 7 }, func(p *poolData) { p.Words = p.Words[:125] },
		func(p *poolData) { p.PortfolioBalances = p.PortfolioBalances[:7] }, func(p *poolData) { p.Words[1].Slot = 2 },
		func(p *poolData) { p.Tokens[0].VolumeRate = 0 }, func(p *poolData) { p.Categories[0].AnchorTokenIndex = 4 },
		func(p *poolData) { p.BaseFeePerGas = nil }, func(p *poolData) { p.Balances[0] = nil },
		func(p *poolData) { p.Tokens[1].Token = p.Tokens[0].Token },
		func(p *poolData) { p.Words[6].Value.Sub(p.Words[6].Value, new(big.Int).Lsh(big.NewInt(1), 176)) },
	} {
		p := fixturePage(t, sample)
		mutate(&p)
		_, err := snapshotState(p, sample.ShareBalances)
		require.Error(t, err)
	}
	p := fixturePage(t, sample)
	_, err := snapshotState(p, nil)
	require.Error(t, err)
	shares := maps.Clone(sample.ShareBalances)
	shares[f.Proxy] = ShareBalance{}
	_, err = snapshotState(p, shares)
	require.Error(t, err)
}
func TestTrackerSchedulesUpdatesWithoutMakerEvents(t *testing.T) {
	tracker := &PoolTracker{}
	require.Equal(t, int64(1001), tracker.NextUpdateAt(entity.Pool{Timestamp: 1000}))
	f := fixture(t, "execution-model3.json.gz")
	s := stateFrom(t, f.Snapshots[0])
	posted, maxAge := s.small(2, 36, 32), s.small(2, 138, 8)
	s.BlockNumber = posted + maxAge
	// The router's explicit one-block inclusion policy crosses the actual
	// contract cutoff; it does not silently clamp the age or use old quotes.
	sim, err := NewPoolSimulator(entityFrom(t, s))
	require.NoError(t, err)
	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: s.Tokens[0].Address, Amount: big.NewInt(1000000)}, TokenOut: s.Tokens[1].Address})
	require.ErrorIs(t, err, ErrUnavailable)
}
