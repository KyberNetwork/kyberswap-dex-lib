package uniswapv4

import (
	"context"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type trackDueHook struct {
	*BaseHook
	due bool
}

func (h *trackDueHook) TrackDue(*HookParam) bool { return h.due }

var (
	notDueHookAddr = common.HexToAddress("0x00000000000000000000000000000000000d0001")
	dueHookAddr    = common.HexToAddress("0x00000000000000000000000000000000000d0002")
	noTrackDueAddr = common.HexToAddress("0x00000000000000000000000000000000000d0003")

	_ = RegisterHooks(&trackDueHook{BaseHook: &BaseHook{}, due: false}, notDueHookAddr)
	_ = RegisterHooks(&trackDueHook{BaseHook: &BaseHook{}, due: true}, dueHookAddr)
	_ = RegisterHooks(&BaseHook{}, noTrackDueAddr)
)

// An interval task (only the zero-value log pool-service substitutes) must cost no RPC when
// the hook is not due, but a real log, a due hook, or a hook without TrackDue must still fetch.
func TestGetNewPoolState_SkipsRPCWhenTrackNotDue(t *testing.T) {
	zeroLog := ethtypes.Log{}
	realLog := ethtypes.Log{Address: common.HexToAddress("0x1"), BlockNumber: 1}

	tests := []struct {
		name      string
		hook      common.Address
		logs      []ethtypes.Log
		overrides bool
		wantRPC   bool
	}{
		{"not due, zero log only", notDueHookAddr, []ethtypes.Log{zeroLog}, false, false},
		{"not due, no logs", notDueHookAddr, nil, false, false},
		{"not due, real log", notDueHookAddr, []ethtypes.Log{realLog}, false, true},
		{"not due, zero log plus real log", notDueHookAddr, []ethtypes.Log{zeroLog, realLog}, false, true},
		{"not due, overrides set", notDueHookAddr, []ethtypes.Log{zeroLog}, true, true},
		{"due, zero log", dueHookAddr, []ethtypes.Log{zeroLog}, false, true},
		{"no TrackDue, zero log", noTrackDueAddr, []ethtypes.Log{zeroLog}, false, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				w.WriteHeader(http.StatusInternalServerError)
			}))
			defer stub.Close()

			extra, err := json.Marshal(Extra{Extra: &uniswapv3.Extra{
				Liquidity: big.NewInt(1), SqrtPriceX96: big.NewInt(1), Tick: big.NewInt(0),
			}})
			require.NoError(t, err)
			p := entity.Pool{
				Address:     "0xpool",
				StaticExtra: `{"hooks":"` + tc.hook.Hex() + `"}`,
				Extra:       string(extra),
			}
			tracker := &PoolTracker{config: &Config{}, ethrpcClient: ethrpc.New(stub.URL)}

			var got entity.Pool
			if tc.overrides {
				got, err = tracker.GetNewPoolStateWithOverrides(context.Background(), p,
					poolpkg.GetNewPoolStateWithOverridesParams{Logs: tc.logs,
						Overrides: map[common.Address]gethclient.OverrideAccount{{1}: {}}})
			} else {
				got, err = tracker.GetNewPoolState(context.Background(), p,
					poolpkg.GetNewPoolStateParams{Logs: tc.logs})
			}

			if tc.wantRPC {
				assert.Positive(t, calls.Load(), "expected RPC")
				return
			}
			require.NoError(t, err)
			assert.Zero(t, calls.Load(), "expected no RPC")
			assert.Equal(t, p, got, "pool must be returned unchanged")
		})
	}
}
