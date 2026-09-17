package lunarbase_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"sort"
	"testing"

	"github.com/KyberNetwork/msgpack/v5"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	lb "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/lunarbase"
	wire "github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpack"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// These are actual Snappy frames emitted by a separate executable importing
// unmodified d3bf232d7b7c9df37c87a8de89c7743310e181c0, using that revision's
// pkg/msgpack.EncodePoolSimulatorsMap (unexported fields, ForceAsArray).
const upstreamKnownFrame = "/wYAAHNOYVBwWQD4AAD2Jb+N2QKggal0ZXN0LXBvb2zUf9lXKmdpdGh1Yi5jb20vS3liZXJOZXR3b3JrL2sBDfB7c3dhcC1kZXgtbGliL3BrZy9saXF1aWRpdHktc291cmNlL2x1bmFyYmFzZSBQb29sU2ltdWxhdG9ynJfZKjB4MDAwMDc5MDRkMTg2NjgwYzcwOTUxOWU3MWY0ZGMzZTJkZjhmMWI5OaCgkqF4oXmSksKSzzXJrcXeoAAANjINAAjAzwAJAQhkkpQVIxEUADYRCQAAFQmSJQAEOJQVCwEJAAEyHQAVCSTOAAAKJc4AAAMcERMEZMIRCgQZzgEIFM4AAUeuwg=="
const upstreamMixedFrame = "/wYAAHNOYVBwWQAZAQA3RrpAtgWggql0ZXN0LXBvb2zUf9lXKmdpdGh1Yi5jb20vS3liZXJOZXR3b3JrL2sBDfB7c3dhcC1kZXgtbGliL3BrZy9saXF1aWRpdHktc291cmNlL2x1bmFyYmFzZSBQb29sU2ltdWxhdG9ynJfZKjB4MDAwMDc5MDRkMTg2NjgwYzcwOTUxOWU3MWY0ZGMzZTJkZjhmMWI5OaCgkqF4oXmSksKSzzXJrcXeoAAANjINAAjAzwAJAQhkkpQVIxEUADYRCQAAFQmSJQAEOJQVCwEJAAEyHQAVCSTOAAAKJc4AAAMcERMEZMIRCgQZzgEIPM4AAUeuwq5hbWJpZ3VvdXPuXQHuXQHuXQHuXQHuXQGKXQEMAAAAwg=="

func wireQuoteParams() pool.CalcAmountOutParams {
	return pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: "x", Amount: big.NewInt(1000000000000000000)}, TokenOut: "y"}
}
func wireSimulator(t *testing.T, extra lb.Extra) *lb.PoolSimulator {
	t.Helper()
	raw, err := json.Marshal(extra)
	require.NoError(t, err)
	s, err := lb.NewPoolSimulator(pool.FactoryParams{ChainID: valueobject.ChainIDBSC, EntityPool: entity.Pool{
		Address: "0x00007904d186680c709519e71f4dc3e2df8f1b99", BlockNumber: 100,
		Extra: string(raw), StaticExtra: "{}", Tokens: []*entity.PoolToken{{Address: "x"}, {Address: "y"}},
		Reserves: entity.PoolReserves{"1000000000000000000000", "1000000000000000000000"},
	}})
	require.NoError(t, err)
	return s
}
func wireExtra() lb.Extra {
	return lb.Extra{SqrtPriceX96: new(uint256.Int).Lsh(uint256.NewInt(1), 96), FeeAskX24: 2597, FeeBidX24: 796, LatestUpdateBlock: 100, BlockDelay: 25, MaxPunishmentX24: 83886, BlockHash: "0x0123456789abcdef"}
}
func wireRaw(t *testing.T, value any) []byte {
	t.Helper()
	var b bytes.Buffer
	e := wire.NewEncoder(&b)
	defer wire.PutEncoder(e)
	require.NoError(t, e.Encode(value))
	return b.Bytes()
}
func wireReadRaw(data []byte, target any) error {
	d := wire.NewDecoder(bytes.NewReader(data))
	defer wire.PutDecoder(d)
	return d.Decode(target)
}
func wireMapRoundTrip(t *testing.T, simulators map[string]pool.IPoolSimulator) map[string]pool.IPoolSimulator {
	t.Helper()
	data, err := wire.EncodePoolSimulatorsMap(simulators)
	require.NoError(t, err)
	decoded, err := wire.DecodePoolSimulatorsMap(data)
	require.NoError(t, err)
	return decoded
}
func wireReadLegacy(t *testing.T, fixture string) map[string]pool.IPoolSimulator {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(fixture)
	require.NoError(t, err)
	decoded, err := wire.DecodePoolSimulatorsMap(raw)
	require.NoError(t, err)
	return decoded
}
func wireRequireKnownQuote(t *testing.T, sim pool.IPoolSimulator) {
	t.Helper()
	q, err := sim.CalcAmountOut(wireQuoteParams())
	require.NoError(t, err)
	require.Equal(t, "999950051307678223", q.TokenAmountOut.Amount.String())
	require.Equal(t, "49948692321777", q.Fee.Amount.String())
}

func TestLunarbaseWireActualUpstreamFrame(t *testing.T) {
	decoded := wireReadLegacy(t, upstreamKnownFrame)
	require.Len(t, decoded, 1)
	s := decoded["test-pool"].(*lb.PoolSimulator)
	wireRequireKnownQuote(t, s)
	require.Empty(t, s.BlockHash, "a legacy frame must not invent canonical provenance")
	require.False(t, s.ConcentrationModel)
	require.Equal(t, "1000000000000000000000", s.GetReserves()[0].String())
	wireRequireKnownQuote(t, wireMapRoundTrip(t, decoded)["test-pool"])
}

func TestLunarbaseWirePreservesModelsAndMetadata(t *testing.T) {
	for _, test := range []struct {
		name          string
		k, max, fee   uint32
		concentration bool
	}{
		{"punishment", 0, 83886, 796, false},
		{"concentration positive K", 4096, 0, 796, true},
		{"concentration zero K", 0, 0, 16777215, true},
		{"verified punishment zero maximum", 0, 0, 796, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			extra := wireExtra()
			extra.ConcentrationK = test.k
			extra.MaxPunishmentX24 = test.max
			extra.ConcentrationModel = test.concentration
			extra.FeeAskX24 = test.fee
			extra.FeeBidX24 = test.fee
			original := wireSimulator(t, extra)
			decoded := wireMapRoundTrip(t, map[string]pool.IPoolSimulator{"pool": original})["pool"].(*lb.PoolSimulator)
			require.Equal(t, original.Extra, decoded.Extra)
			require.Equal(t, original.StaticExtra, decoded.StaticExtra)
			require.Equal(t, original.GetMetaInfo("x", "y"), decoded.GetMetaInfo("x", "y"))
			require.Equal(t, original.GetReserves(), decoded.GetReserves())
			want, err := original.CalcAmountOut(wireQuoteParams())
			require.NoError(t, err)
			got, err := decoded.CalcAmountOut(wireQuoteParams())
			require.NoError(t, err)
			require.Equal(t, want.TokenAmountOut, got.TokenAmountOut)
			require.Equal(t, want.Fee, got.Fee)
			if test.concentration && test.k == 0 {
				require.Equal(t, "59604644776", got.TokenAmountOut.Amount.String())
			}
		})
	}
}

func TestLunarbaseWireAmbiguityIsPoolLocalAndPersistent(t *testing.T) {
	decoded := wireReadLegacy(t, upstreamMixedFrame)
	require.Len(t, decoded, 2)
	check := func(t *testing.T, pools map[string]pool.IPoolSimulator) {
		t.Helper()
		wireRequireKnownQuote(t, pools["test-pool"])
		ambiguous := pools["ambiguous-pool"]
		_, err := ambiguous.CalcAmountOut(wireQuoteParams())
		require.ErrorIs(t, err, lb.ErrStalePool)
		_, err = ambiguous.CloneState().CalcAmountOut(wireQuoteParams())
		require.ErrorIs(t, err, lb.ErrStalePool)
	}
	check(t, decoded)
	check(t, wireMapRoundTrip(t, decoded))
	refreshed := wireExtra()
	refreshed.MaxPunishmentX24 = 0
	_, err := wireSimulator(t, refreshed).CalcAmountOut(wireQuoteParams())
	require.NoError(t, err)
}

func TestLunarbaseWireTypedSwapInfoPreservesPostReserves(t *testing.T) {
	original := wireSimulator(t, wireExtra())
	q, err := original.CalcAmountOut(wireQuoteParams())
	require.NoError(t, err)
	var decodedInfo lb.SwapInfo
	require.NoError(t, wireReadRaw(wireRaw(t, q.SwapInfo.(lb.SwapInfo)), &decodedInfo))
	first := original.CloneState()
	second := original.CloneState()
	update := pool.UpdateBalanceParams{TokenAmountIn: wireQuoteParams().TokenAmountIn, TokenAmountOut: *q.TokenAmountOut, Fee: *q.Fee, SwapInfo: q.SwapInfo}
	first.UpdateBalance(update)
	update.SwapInfo = decodedInfo
	second.UpdateBalance(update)
	require.Equal(t, first.GetReserves(), second.GetReserves())
	require.Equal(t, "1001000000000000000000", second.GetReserves()[0].String())
	require.Equal(t, "999000000000000000000", second.GetReserves()[1].String(), "the full output fee leaves the tradable reserve")
	a, err := first.CalcAmountOut(wireQuoteParams())
	require.NoError(t, err)
	b, err := second.CalcAmountOut(wireQuoteParams())
	require.NoError(t, err)
	require.Equal(t, a.TokenAmountOut, b.TokenAmountOut)
	require.Equal(t, a.Fee, b.Fee)
	// SwapInfo inside a generic result was not a registered polymorphic wire
	// type in upstream either. A typed roundtrip is the supported test here.
	var generic pool.CalcAmountOutResult
	require.NoError(t, wireReadRaw(wireRaw(t, q), &generic))
	require.IsType(t, []any{}, generic.SwapInfo)
}

func TestLunarbaseWireRejectsMalformedWithoutMutatingTarget(t *testing.T) {
	original := wireSimulator(t, wireExtra())
	var fields map[string]msgpack.RawMessage
	require.NoError(t, wireReadRaw(wireRaw(t, original), &fields))
	cloneFields := func() map[string]msgpack.RawMessage {
		next := make(map[string]msgpack.RawMessage, len(fields))
		for k, v := range fields {
			next[k] = v
		}
		return next
	}
	mutate := func(key string, value any) []byte {
		next := cloneFields()
		next[key] = wireRaw(t, value)
		return wireRaw(t, next)
	}
	missing := func(key string) []byte { next := cloneFields(); delete(next, key); return wireRaw(t, next) }
	var duplicate bytes.Buffer
	enc := wire.NewEncoder(&duplicate)
	require.NoError(t, enc.EncodeMapLen(len(fields)+1))
	names := make([]string, 0, len(fields))
	for k := range fields {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		require.NoError(t, enc.EncodeString(k))
		require.NoError(t, enc.Encode(fields[k]))
	}
	require.NoError(t, enc.EncodeString("_lunarbaseWire"))
	require.NoError(t, enc.EncodeUint64(2))
	wire.PutEncoder(enc)
	for _, test := range []struct {
		name string
		data []byte
	}{
		{"unsupported version 258", mutate("_lunarbaseWire", uint64(258))},
		{"missing canonical metadata", missing("BlockHash")},
		{"missing model metadata", missing("ConcentrationModel")},
		{"missing refresh flag", missing("requiresRPCRefresh")},
		{"missing reserves", missing("reserves")},
		{"nil reserves", mutate("reserves", nil)},
		{"nil price", mutate("SqrtPriceX96", nil)},
		{"wrong fee type", mutate("FeeAskX24", "invalid")},
		{"duplicate field", duplicate.Bytes()},
		{"unsupported legacy length", wireRaw(t, make([]any, 11))},
		{"truncated map", wireRaw(t, original)[:17]},
	} {
		t.Run(test.name, func(t *testing.T) {
			target := wireSimulator(t, wireExtra())
			before := wireRaw(t, target)
			require.Error(t, wireReadRaw(test.data, target))
			require.Equal(t, before, wireRaw(t, target))
		})
	}
	t.Run("unknown named field is skipped", func(t *testing.T) {
		var target lb.PoolSimulator
		require.NoError(t, wireReadRaw(mutate("futureMetadata", []string{"value"}), &target))
		wireRequireKnownQuote(t, &target)
	})
	t.Run("direct codec rejects nil", func(t *testing.T) {
		// The generic library handles a nil code before invoking custom methods.
		// This assertion is about our explicit decoder's contract.
		target := wireSimulator(t, wireExtra())
		before := wireRaw(t, target)
		dec := wire.NewDecoder(bytes.NewReader(wireRaw(t, nil)))
		defer wire.PutDecoder(dec)
		require.Error(t, target.DecodeMsgpack(dec))
		require.Equal(t, before, wireRaw(t, target))
	})
}
