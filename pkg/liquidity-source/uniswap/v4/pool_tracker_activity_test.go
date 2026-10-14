package uniswapv4

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

// Test hooks return hX / reserves from these vars; optedHook also implements HookTracksActivity.
var (
	testHX       = `{"a":1}`
	testReserves entity.PoolReserves
)

type plainHook struct{ *BaseHook }

func (*plainHook) Track(context.Context, *HookParam) (json.RawMessage, error) {
	return json.RawMessage(testHX), nil
}

func (*plainHook) GetReserves(context.Context, *HookParam) (entity.PoolReserves, error) {
	return testReserves, nil
}

type optedHook struct{ plainHook }

func (*optedHook) TracksActivity() {}

var (
	plainHookAddr = common.HexToAddress("0x00000000000000000000000000000000000d0021")
	optedHookAddr = common.HexToAddress("0x00000000000000000000000000000000000d0022")

	_ = RegisterHooks(&plainHook{&BaseHook{}}, plainHookAddr)
	_ = RegisterHooks(&optedHook{plainHook{&BaseHook{}}}, optedHookAddr)
)

const stubBlock = 100 // block returned by aggregateStub

func newActivityTracker(t *testing.T) *PoolTracker {
	var calls atomic.Int32
	stub := aggregateStub(t, &calls)
	t.Cleanup(stub.Close)
	gql := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"ticks":[]}}`))
	}))
	t.Cleanup(gql.Close)

	orig := NowFn
	t.Cleanup(func() { NowFn = orig; testHX = `{"a":1}`; testReserves = nil })
	return &PoolTracker{config: &Config{}, ethrpcClient: ethrpc.New(stub.URL), graphqlClient: graphqlpkg.NewClient(gql.URL)}
}

func (tr *PoolTracker) refresh(t *testing.T, p entity.Pool, now int64, hx string, reserves entity.PoolReserves,
	logs []ethtypes.Log, headers map[uint64]entity.BlockHeader) entity.Pool {
	NowFn = func() int64 { return now }
	testHX, testReserves = hx, reserves
	got, err := tr.GetNewPoolState(context.Background(), p, poolpkg.GetNewPoolStateParams{Logs: logs, BlockHeaders: headers})
	require.NoError(t, err)
	return got
}

// settledPool is a pool whose stored state equals what the tracker would produce for hX {"a":1}.
func (tr *PoolTracker) settledPool(t *testing.T, hook common.Address, ts int64, block uint64) entity.Pool {
	p := tr.refresh(t, schedulePool(t, hook, 0), 1, `{"a":1}`, nil, nil, nil)
	p.Timestamp, p.BlockNumber = ts, block
	return p
}

// An opted-in hook's Timestamp/BlockNumber follow the pool state: they move only when Extra,
// SwapFee or Reserves change, never to a clock or block older than the stored one, and not for
// hooks that did not opt in.
func TestActivityBump(t *testing.T) {
	const ts, block, now = int64(1_000), uint64(50), int64(5_000)
	for name, tc := range map[string]struct {
		hook      common.Address
		hx        string
		reserves  entity.PoolReserves
		swapFee   float64 // 0: keep the settled fee
		ts        int64
		block     uint64
		wantTs    int64
		wantBlock uint64
	}{
		"hX changed bumps":              {optedHookAddr, `{"a":2}`, nil, 0, ts, block, now, stubBlock},
		"identical state keeps":         {optedHookAddr, `{"a":1}`, nil, 0, ts, block, ts, block},
		"non-opted hX change keeps":     {plainHookAddr, `{"a":2}`, nil, 0, ts, block, ts, block},
		"reserves changed bumps":        {optedHookAddr, `{"a":1}`, entity.PoolReserves{"7", "8"}, 0, ts, block, now, stubBlock},
		"swap fee changed bumps":        {optedHookAddr, `{"a":1}`, nil, 1, ts, block, now, stubBlock},
		"never moves back":              {optedHookAddr, `{"a":2}`, nil, 0, now + 4_000, 500, now + 4_000, 500},
		"non-opted reserves change":     {plainHookAddr, `{"a":1}`, entity.PoolReserves{"7", "8"}, 0, ts, block, ts, block},
		"non-opted swap fee change":     {plainHookAddr, `{"a":1}`, nil, 1, ts, block, ts, block},
		"opted + changed + stale fetch": {optedHookAddr, `{"a":3}`, nil, 0, ts, stubBlock + 1, now, stubBlock + 1},
	} {
		t.Run(name, func(t *testing.T) {
			tr := newActivityTracker(t)
			p := tr.settledPool(t, tc.hook, tc.ts, tc.block)
			if tc.swapFee != 0 {
				p.SwapFee = tc.swapFee
			}

			got := tr.refresh(t, p, now, tc.hx, tc.reserves, nil, nil)

			require.Equal(t, tc.wantTs, got.Timestamp)
			require.Equal(t, tc.wantBlock, got.BlockNumber)
		})
	}
}

// A real log keeps the header timestamp (EstimateLastActivityTime) when the state did not change:
// the wall-clock bump applies to a changed state only.
func TestActivityBump_LogHeaderTimestampWhenUnchanged(t *testing.T) {
	tr := newActivityTracker(t)
	p := tr.settledPool(t, optedHookAddr, 1_000, 50)
	logs := []ethtypes.Log{{Address: common.HexToAddress("0x1"), BlockNumber: stubBlock}}
	headers := map[uint64]entity.BlockHeader{stubBlock: {Timestamp: 2_000}}

	got := tr.refresh(t, p, 5_000, `{"a":1}`, nil, logs, headers)
	require.Equal(t, int64(2_000), got.Timestamp)

	got = tr.refresh(t, p, 5_000, `{"a":2}`, nil, logs, headers)
	require.Equal(t, int64(5_000), got.Timestamp)
}

// Bootstrap starts from an empty Extra, so an opted-in pool is bumped once; re-bootstrapping the
// same state does not bump again.
func TestActivityBump_Bootstrap(t *testing.T) {
	for name, tc := range map[string]struct {
		hook      common.Address
		wantTs    int64
		wantBlock uint64
	}{
		"opted":     {optedHookAddr, 5_000, stubBlock},
		"non-opted": {plainHookAddr, 0, 0},
	} {
		t.Run(name, func(t *testing.T) {
			tr := newActivityTracker(t)
			NowFn = func() int64 { return 5_000 }
			p := entity.Pool{Address: "0xpool", StaticExtra: `{"hooks":"` + tc.hook.Hex() + `"}`}
			got, err := tr.BootstrapPoolState(context.Background(), p, poolpkg.GetNewPoolStateParams{})
			require.NoError(t, err)
			require.Equal(t, tc.wantTs, got.Timestamp)
			require.Equal(t, tc.wantBlock, got.BlockNumber)

			NowFn = func() int64 { return 9_000 }
			again, err := tr.BootstrapPoolState(context.Background(), got, poolpkg.GetNewPoolStateParams{})
			require.NoError(t, err)
			require.Equal(t, tc.wantTs, again.Timestamp, "same state must not bump again")
			require.Equal(t, tc.wantBlock, again.BlockNumber)
		})
	}
}
