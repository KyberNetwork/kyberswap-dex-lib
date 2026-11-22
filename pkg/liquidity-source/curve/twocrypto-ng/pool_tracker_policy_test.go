package twocryptong_test

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/shared"
	twocryptong "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/twocrypto-ng"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/twocrypto-ng/internal/rpcstub"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/twocrypto-ng/lazy"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var (
	stubPool   = common.HexToAddress("0x862cb4e988fb66e72f128d1183829f8c05b6c6a0")
	stubPolicy = common.HexToAddress("0x57336cE442165ccDF19a820355a823238137170b")
)

func bi(v int64) *big.Int { return big.NewInt(v) }

// bp is m * 10^e.
func bp(m int64, e int64) *big.Int {
	return new(big.Int).Mul(bi(m), new(big.Int).Exp(bi(10), bi(e), nil))
}

// policyHandler serves the pool (zero state, POLICY = stubPolicy) and a policy in one of the modes:
// "bank" (get_report), "old" (report), "unknown" (every call reverts), "noreport" (config readable, report and state revert).
func policyHandler(mode string) rpcstub.Handler {
	return func(target common.Address, data []byte) ([]byte, bool) {
		if target == stubPool {
			if m, err := twocryptong.CurveTwocryptoNGABI.MethodById(data[:4]); err == nil && m.Name == "POLICY" {
				out, _ := m.Outputs.Pack(stubPolicy)
				return out, true
			}
			return rpcstub.ZeroReturn(twocryptong.CurveTwocryptoNGABI, data)
		}
		if mode == "unknown" {
			return nil, false
		}
		var vals []any
		m, err := twocryptong.YBBankPolicyABI.MethodById(data[:4])
		if err != nil {
			if m, err = twocryptong.YBPolicyABI.MethodById(data[:4]); err != nil {
				return nil, false
			}
		}
		switch m.Name {
		case "is_killed":
			vals = []any{false}
		case "config":
			vals = []any{bp(3, 7), bp(5, 17), bp(2, 8), bi(90), bi(1), bi(1), bi(1), bi(1), bi(1)}
		case "get_report":
			if mode != "bank" {
				return nil, false
			}
			vals = []any{bp(2, 21), bp(3, 21), bp(1, 15)}
		case "report":
			if mode != "old" {
				return nil, false
			}
			vals = []any{bp(3, 21), bp(1, 15)}
		case "state":
			if mode == "noreport" {
				return nil, false
			}
			vals = []any{bi(1), bi(1), bi(1), bp(2, 21), [2]*big.Int{bp(1, 24), bp(1, 24)}, bp(2, 24), bi(1)}
		case "cached_A":
			vals = []any{bi(50000)}
		default:
			return nil, false
		}
		out, err := m.Outputs.Pack(vals...)
		if err != nil {
			panic(err)
		}
		return out, true
	}
}

var policyModes = []struct {
	mode, reportPrice, fallbackFee string
}{
	{"bank", "3000000000000000000000", "200000000"},
	{"old", "3000000000000000000000", "200000000"},
	{"noreport", "0", "200000000"}, // report unreadable: the policy's own fallback_fee
	{"unknown", "0", "1000000000"}, // nothing readable: the 10% cap
}

func stubEntityPool() entity.Pool {
	return entity.Pool{
		Address: stubPool.Hex(), Exchange: "curve-twocrypto-ng", Type: twocryptong.DexType,
		Reserves: entity.PoolReserves{"0", "0"},
		Tokens:   []*entity.PoolToken{{Address: "0x01", Swappable: true}, {Address: "0x02", Swappable: true}},
	}
}

func requirePolicyState(t *testing.T, p entity.Pool, reportPrice, fallbackFee string) {
	t.Helper()
	require.NotEmpty(t, p.Reserves, "pool state must still update")
	require.Equal(t, uint64(12345), p.BlockNumber)
	var extra twocryptong.Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	require.NotNil(t, extra.Policy)
	require.Equal(t, fallbackFee, extra.Policy.FallbackFee.Dec())
	require.Equal(t, reportPrice, extra.Policy.ReportPrice.Dec())
}

// One multicall per policy fetch whatever the ABI (the old ABI is probed in the same batch), so a
// refresh costs 2 roundtrips: pool data, then policy. Pre-change code also took 2.
func TestFetchPolicyRoundtrips(t *testing.T) {
	for _, tc := range policyModes {
		s := rpcstub.New(t, policyHandler(tc.mode))
		st, err := twocryptong.FetchPolicy(context.Background(), s.Client, nil, stubPolicy, nil)
		require.NoError(t, err, tc.mode)
		require.Equal(t, tc.fallbackFee, st.FallbackFee.Dec(), tc.mode)
		require.Equal(t, tc.reportPrice, st.ReportPrice.Dec(), tc.mode)
		require.EqualValues(t, 1, s.Roundtrips.Load(), tc.mode)
	}
}

func TestTrackerPolicyFallback(t *testing.T) {
	cfg := &shared.Config{DexID: "test"}
	for _, tc := range policyModes {
		t.Run("tracker/"+tc.mode, func(t *testing.T) {
			s := rpcstub.New(t, policyHandler(tc.mode))
			tr, err := twocryptong.NewPoolTracker(cfg, s.Client)
			require.NoError(t, err)
			p, err := tr.GetNewPoolState(context.Background(), stubEntityPool(), poolpkg.GetNewPoolStateParams{})
			require.NoError(t, err)
			requirePolicyState(t, p, tc.reportPrice, tc.fallbackFee)
			require.EqualValues(t, 2, s.Roundtrips.Load())
		})
		t.Run("lazy/"+tc.mode, func(t *testing.T) {
			s := rpcstub.New(t, policyHandler(tc.mode))
			tr, err := lazy.NewPoolTracker(cfg, s.Client)
			require.NoError(t, err)
			p, err := tr.GetNewPoolState(context.Background(), stubEntityPool(), poolpkg.GetNewPoolStateParams{})
			require.NoError(t, err)
			requirePolicyState(t, p, tc.reportPrice, tc.fallbackFee)
			require.EqualValues(t, 2, s.Roundtrips.Load())
		})
		// Batched path: a worker runs the planned pool calls, then applies the result.
		t.Run("lazybatch/"+tc.mode, func(t *testing.T) {
			s := rpcstub.New(t, policyHandler(tc.mode))
			tr, err := lazy.NewPoolTracker(cfg, s.Client)
			require.NoError(t, err)
			req, apply, err := tr.LazyNewPoolState(context.Background(), stubEntityPool(), poolpkg.GetNewPoolStateParams{})
			require.NoError(t, err)
			res, err := req.(*poolpkg.LazyRequest).TryBlockAndAggregate()
			require.NoError(t, err)
			p, err := apply(res.BlockNumber)
			require.NoError(t, err)
			requirePolicyState(t, p, tc.reportPrice, tc.fallbackFee)
			require.EqualValues(t, 2, s.Roundtrips.Load())
		})
	}
}
