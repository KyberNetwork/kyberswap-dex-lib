package aegisprop

import (
	"context"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
)

func ladderHookExtra(t *testing.T, legacyTrackedAt int64) uniswapv4.HookExtra {
	lvl := []Level{{Price: uint256.NewInt(1e18), Amplitude: uint256.NewInt(1e18)}}
	raw, err := json.Marshal(&Hook{Bids: lvl, Asks: lvl, FreshThresholdSec: 3600, StaleThresholdSec: 86400,
		HaircutBpsAtStaleThreshold: 1000, TrackedAt: uint64(legacyTrackedAt)})
	require.NoError(t, err)
	return raw
}

func quote(t *testing.T, extra uniswapv4.HookExtra, lv int64) (*big.Int, error) {
	h, ok := uniswapv4.GetHook(HookAddresses[0], &uniswapv4.HookParam{HookExtra: extra, TrackedAt: lv})
	require.True(t, ok)
	res, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true, AmountSpecified: big.NewInt(1e15)})
	if err != nil {
		return nil, err
	}
	return res.DeltaUnspecified, nil
}

// The tracker's Extra lv is the staleness-ramp age baseline once a pool has one: a pool whose lv
// aged past the ramp is refused, and one inside the ramp pays a haircut. The legacy hX clock decides
// only while lv is 0.
func TestStalenessUsesLiveness(t *testing.T) {
	now := time.Now().Unix()
	fresh, err := quote(t, ladderHookExtra(t, 0), now-100)
	require.NoError(t, err)

	_, err = quote(t, ladderHookExtra(t, 0), now-100_000)
	assert.ErrorIs(t, err, ErrStaleLadder, "lv older than the stale threshold")

	ramp, err := quote(t, ladderHookExtra(t, 0), now-40_000)
	require.NoError(t, err)
	assert.Equal(t, 1, ramp.Cmp(fresh), "inside the ramp the haircut shrinks the output (DeltaUnspecified is -out)")

	got, err := quote(t, ladderHookExtra(t, now-100_000), now-100)
	require.NoError(t, err)
	assert.Equal(t, fresh, got, "lv beats an old legacy clock")

	_, err = quote(t, ladderHookExtra(t, now-100), 0)
	assert.NoError(t, err, "no lv: fresh legacy clock")
	_, err = quote(t, ladderHookExtra(t, now-100_000), 0)
	assert.ErrorIs(t, err, ErrStaleLadder, "no lv: old legacy clock")
	_, err = quote(t, ladderHookExtra(t, 0), 0)
	assert.ErrorIs(t, err, ErrStaleLadder, "no clock at all")
}

func TestLivenessInterval(t *testing.T) {
	var _ uniswapv4.HookWithLiveness = (*Hook)(nil)
	var _ uniswapv4.HookTracksActivity = (*Hook)(nil)
	assert.Equal(t, 5*time.Minute, (&Hook{}).LivenessInterval())
	assert.Less(t, uint64((&Hook{}).LivenessInterval()/time.Second), uint64(3600), "well inside the fresh threshold")
}

// Track writes no clock: two Tracks of the same chain state at different times, starting from a
// legacy hX that still carries "t", give byte-identical output without it.
func TestTrackWritesNoClock(t *testing.T) {
	var ladder LadderRPC
	for i := range ladder.Bids {
		ladder.Bids[i] = LevelRPC{Price: big.NewInt(0), Amplitude: big.NewInt(0)}
		ladder.Asks[i] = LevelRPC{Price: big.NewInt(0), Amplitude: big.NewInt(0)}
	}
	ladder.Bids[0] = LevelRPC{Price: big.NewInt(1e18), Amplitude: big.NewInt(1e18)}
	ladder.Asks[0] = ladder.Bids[0]
	ladder.BidLen, ladder.AskLen, ladder.Expiration = 1, 1, 4_000_000_000
	ladderOut, err := aegisPropHookABI.Methods["ladder"].Outputs.Pack(ladder)
	require.NoError(t, err)
	cfgOut, err := aegisPropHookABI.Methods["poolConfig"].Outputs.Pack(uint16(1), uint8(18), uint8(18), uint16(0),
		uint16(10), uint16(1000), uint32(3600), uint32(86400), OracleConfigRPC{})
	require.NoError(t, err)

	uint256T, _ := abi.NewType("uint256", "", nil)
	bytesArrT, _ := abi.NewType("bytes[]", "", nil)
	agg, err := abi.Arguments{{Type: uint256T}, {Type: bytesArrT}}.Pack(big.NewInt(100), [][]byte{ladderOut, cfgOut})
	require.NoError(t, err)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID json.RawMessage `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		resp, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": hexutil.Encode(agg)})
		_, _ = w.Write(resp)
	}))
	defer srv.Close()

	track := func() []byte {
		extra := ladderHookExtra(t, 1_700_000_000) // legacy hX with a "t"
		hook, ok := uniswapv4.GetHook(HookAddresses[0], &uniswapv4.HookParam{HookExtra: extra})
		require.True(t, ok)
		raw, err := hook.Track(context.Background(), &uniswapv4.HookParam{
			RpcClient: ethrpc.New(srv.URL), Pool: &entity.Pool{Address: "0xabc"}, HookAddress: HookAddresses[0]})
		require.NoError(t, err)
		return raw
	}
	first := track()
	time.Sleep(1100 * time.Millisecond) // a Unix-second clock in hX would differ now
	assert.Equal(t, string(first), string(track()))
	var m map[string]any
	require.NoError(t, json.Unmarshal(first, &m))
	assert.NotContains(t, m, "t")
}
