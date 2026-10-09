package uniswapv4

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	gojson "github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const scheduledAt = int64(1_800_000_000)

type scheduleHook struct{ *BaseHook }

func (*scheduleHook) NextTrackAt(*HookParam) int64 { return scheduledAt }

var (
	scheduleHookAddr   = common.HexToAddress("0x00000000000000000000000000000000000d0011")
	noScheduleHookAddr = common.HexToAddress("0x00000000000000000000000000000000000d0012")

	_ = RegisterHooks(&scheduleHook{BaseHook: &BaseHook{}}, scheduleHookAddr)
	_ = RegisterHooks(&BaseHook{}, noScheduleHookAddr)
)

// aggregateStub answers every eth_call with a multicall aggregate of liquidity and slot0.
func aggregateStub(t *testing.T, calls *atomic.Int32) *httptest.Server {
	uint256T, _ := abi.NewType("uint256", "", nil)
	bytesArrT, _ := abi.NewType("bytes[]", "", nil)
	outs := abi.Arguments{{Type: uint256T}, {Type: bytesArrT}}

	liq, err := stateViewABI.Methods["getLiquidity"].Outputs.Pack(big.NewInt(1e18))
	require.NoError(t, err)
	sqrtPrice, _ := new(big.Int).SetString("79228162514264337593543950336", 10)
	slot0, err := stateViewABI.Methods["getSlot0"].Outputs.Pack(sqrtPrice, big.NewInt(0), uint32(0), uint32(3000))
	require.NoError(t, err)
	ret, err := outs.Pack(big.NewInt(100), [][]byte{liq, slot0})
	require.NoError(t, err)

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var req struct {
			ID json.RawMessage `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		resp, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": hexutil.Encode(ret)})
		_, _ = w.Write(resp)
	}))
}

func schedulePool(t *testing.T, hook common.Address, timestamp int64) entity.Pool {
	extra, err := gojson.Marshal(Extra{Extra: &uniswapv3.Extra{
		Liquidity: big.NewInt(1), SqrtPriceX96: big.NewInt(1), Tick: big.NewInt(0),
	}})
	require.NoError(t, err)
	return entity.Pool{
		Address:     "0xpool",
		Timestamp:   timestamp,
		StaticExtra: `{"hooks":"` + hook.Hex() + `"}`,
		Extra:       string(extra),
	}
}

// NextUpdateAt must report the hook's schedule (0 without HookWithSchedule) and never call RPC.
func TestNextUpdateAt(t *testing.T) {
	for name, tc := range map[string]struct {
		hook common.Address
		want int64
	}{
		"hook with schedule":    {scheduleHookAddr, scheduledAt},
		"hook without schedule": {noScheduleHookAddr, 0},
	} {
		t.Run(name, func(t *testing.T) {
			var calls atomic.Int32
			stub := aggregateStub(t, &calls)
			defer stub.Close()
			tracker := &PoolTracker{config: &Config{}, ethrpcClient: ethrpc.New(stub.URL)}

			got := poolpkg.IPoolTrackerWithSchedule(tracker).NextUpdateAt(schedulePool(t, tc.hook, 0))

			assert.Equal(t, tc.want, got)
			assert.Zero(t, calls.Load(), "NextUpdateAt must not make RPC calls")
		})
	}
}

// pool-service sends only a zero-value log for scheduled updates. It must refresh state but
// not bump Timestamp, or the inactive-pool tracker would never inactivate an idle pool.
func TestGetNewPoolState_ZeroLogKeepsTimestamp(t *testing.T) {
	const oldTimestamp, headerTimestamp = int64(1_000), uint64(2_000)
	headers := map[uint64]entity.BlockHeader{100: {Timestamp: headerTimestamp}}

	for name, tc := range map[string]struct {
		log  ethtypes.Log
		want int64
	}{
		"zero log keeps timestamp": {ethtypes.Log{}, oldTimestamp},
		"real log bumps (control)": {ethtypes.Log{Address: common.HexToAddress("0x1"), BlockNumber: 100}, int64(headerTimestamp)},
	} {
		t.Run(name, func(t *testing.T) {
			var calls atomic.Int32
			stub := aggregateStub(t, &calls)
			defer stub.Close()
			tracker := &PoolTracker{config: &Config{}, ethrpcClient: ethrpc.New(stub.URL)}

			p := schedulePool(t, noScheduleHookAddr, oldTimestamp)
			got, err := tracker.GetNewPoolState(context.Background(), p, poolpkg.GetNewPoolStateParams{
				Logs: []ethtypes.Log{tc.log}, BlockHeaders: headers,
			})

			require.NoError(t, err)
			assert.Positive(t, calls.Load(), "state must still be refreshed")
			assert.NotEqual(t, p.Extra, got.Extra, "state must be refreshed")
			assert.Equal(t, tc.want, got.Timestamp)
		})
	}
}
