package twocryptong_test

import (
	"context"
	"math/big"
	"sync/atomic"
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
	// v21dOnlyViews are the v2.1.0d views a v2.0.0 pool lacks; v3OnlyViews are the v3.0.0 views a v2.x pool lacks.
	v21dOnlyViews = map[string]bool{"last_timestamp": true, "donation_shares": true, "donation_duration": true,
		"last_donation_release_ts": true, "donation_protection_expiry_ts": true, "donation_protection_period": true}
	v3OnlyViews = map[string]bool{"lp_xcp_profit": true, "reserved_profit_fraction": true, "admin_fee": true,
		"POLICY": true}
	// preV3Views are the views v3.0.0 dropped.
	preV3Views = map[string]bool{"allowed_extra_profit": true}
)

// versionCounter counts the pool calls the stub serves, by method, and how many of them revert.
type versionCounter struct {
	calls, reverts atomic.Int64
	byMethod       map[string]int
}

// versionHandler serves a pool of the given version(): each version reverts the views it lacks, as the
// deployed contracts do. Every other view returns zero, except last_timestamp (7) and version().
func versionHandler(version string, c *versionCounter) rpcstub.Handler {
	return func(target common.Address, data []byte) ([]byte, bool) {
		m, err := twocryptong.CurveTwocryptoNGABI.MethodById(data[:4])
		if target != stubPool || err != nil {
			return nil, false
		}
		c.calls.Add(1)
		c.byMethod[m.Name]++
		lacks := (version == "v2.0.0" && (v21dOnlyViews[m.Name] || v3OnlyViews[m.Name])) ||
			(version == twocryptong.VersionV21d && v3OnlyViews[m.Name]) ||
			(version == twocryptong.VersionV3 && preV3Views[m.Name])
		if lacks {
			c.reverts.Add(1)
			return nil, false
		}
		switch m.Name {
		case "version":
			out, _ := m.Outputs.Pack(version)
			return out, true
		case "last_timestamp":
			out, _ := m.Outputs.Pack(big.NewInt(7))
			return out, true
		}
		return rpcstub.ZeroReturn(twocryptong.CurveTwocryptoNGABI, data)
	}
}

// refreshFn runs one refresh of p through one of the trackers.
type refreshFn func(t *testing.T, s *rpcstub.Stub, p entity.Pool) entity.Pool

var versionTrackers = map[string]refreshFn{
	"tracker": func(t *testing.T, s *rpcstub.Stub, p entity.Pool) entity.Pool {
		tr, err := twocryptong.NewPoolTracker(&shared.Config{DexID: "test"}, s.Client)
		require.NoError(t, err)
		out, err := tr.GetNewPoolState(context.Background(), p, poolpkg.GetNewPoolStateParams{})
		require.NoError(t, err)
		return out
	},
	"lazy": func(t *testing.T, s *rpcstub.Stub, p entity.Pool) entity.Pool {
		tr, err := lazy.NewPoolTracker(&shared.Config{DexID: "test"}, s.Client)
		require.NoError(t, err)
		out, err := tr.GetNewPoolState(context.Background(), p, poolpkg.GetNewPoolStateParams{})
		require.NoError(t, err)
		return out
	},
	"lazybatch": func(t *testing.T, s *rpcstub.Stub, p entity.Pool) entity.Pool {
		tr, err := lazy.NewPoolTracker(&shared.Config{DexID: "test"}, s.Client)
		require.NoError(t, err)
		req, apply, err := tr.LazyNewPoolState(context.Background(), p, poolpkg.GetNewPoolStateParams{})
		require.NoError(t, err)
		res, err := req.(*poolpkg.LazyRequest).TryBlockAndAggregate()
		require.NoError(t, err)
		out, err := apply(res.BlockNumber)
		require.NoError(t, err)
		return out
	},
}

// baseCalls is the version-independent read of a 2-coin pool: 14 views, 2 balances, 3 price views. Every version
// reads one of policy() (v3.0.0) and allowed_extra_profit() (the others) on top, and a pool of no stored version
// both.
const baseCalls = 19

func withExtra(extra string) entity.Pool {
	p := stubEntityPool()
	p.Extra = extra
	return p
}

// A refresh adds version(), the v2.1.0d views and the v3.0.0 views only while the stored state has no
// version; once it has, only that version's views.
func TestVersionGateCallCounts(t *testing.T) {
	for _, tc := range []struct {
		name, stored, poolVersion string
		want                      int
	}{
		{"empty", "", "v2.0.0", baseCalls + 2 + 1 + 6 + 3},
		{"no version", `{"D":"1"}`, twocryptong.VersionV21d, baseCalls + 2 + 1 + 6 + 3},
		{"v2.0.0", `{"Version":"v2.0.0"}`, "v2.0.0", baseCalls + 1},
		{"v2.1.0d", `{"Version":"v2.1.0d"}`, twocryptong.VersionV21d, baseCalls + 1 + 6},
		{"v3.0.0", `{"Version":"v3.0.0"}`, twocryptong.VersionV3, baseCalls + 1 + 6 + 3},
		{"other", `{"Version":"v9.9.9"}`, "v9.9.9", baseCalls + 1},
	} {
		for name, refresh := range versionTrackers {
			t.Run(name+"/"+tc.name, func(t *testing.T) {
				c := &versionCounter{byMethod: map[string]int{}}
				s := rpcstub.New(t, versionHandler(tc.poolVersion, c))
				p := refresh(t, s, withExtra(tc.stored))
				require.EqualValues(t, tc.want, c.calls.Load())
				var extra twocryptong.Extra
				require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
				require.Equal(t, tc.poolVersion, extra.Version, "the stored version is kept")
				if tc.stored != "" && tc.stored != `{"D":"1"}` {
					require.Zero(t, c.byMethod["version"], "version() is not read again")
				}
				if tc.poolVersion == twocryptong.VersionV3 || tc.poolVersion == twocryptong.VersionV21d {
					// Both simulators read LastTimestamp (v3.0.0's ramp test among them): every refresh reads it.
					require.Equal(t, 1, c.byMethod["last_timestamp"], "last_timestamp is read on every refresh")
				}
			})
		}
	}
}

// A pool of any version, after the refresh that stores its version, issues no call that reverts: v2.x no
// policy(), v3.0.0 no allowed_extra_profit().
func TestVersionGateNoRevertsAfterBootstrap(t *testing.T) {
	for _, tc := range []struct {
		version          string
		bootstrapReverts int64
	}{{"v2.0.0", 10}, {twocryptong.VersionV21d, 4}, {twocryptong.VersionV3, 1}} {
		for name, refresh := range versionTrackers {
			t.Run(name+"/"+tc.version, func(t *testing.T) {
				c := &versionCounter{byMethod: map[string]int{}}
				s := rpcstub.New(t, versionHandler(tc.version, c))
				p := refresh(t, s, stubEntityPool())
				require.EqualValues(t, tc.bootstrapReverts, c.reverts.Load(), "bootstrap probes every version's views")
				c.calls.Store(0)
				c.reverts.Store(0)
				refresh(t, s, p)
				require.Zero(t, c.reverts.Load())
			})
		}
	}
}

// Gated refreshes store the same extra as the full read: the v2.1.0d views keep last_timestamp, and the
// version survives. Bootstrap and the second refresh agree field by field.
func TestVersionGateExtraUnchanged(t *testing.T) {
	for _, v := range []string{"v2.0.0", twocryptong.VersionV21d, twocryptong.VersionV3} {
		for name, refresh := range versionTrackers {
			t.Run(name+"/"+v, func(t *testing.T) {
				c := &versionCounter{byMethod: map[string]int{}}
				s := rpcstub.New(t, versionHandler(v, c))
				first := refresh(t, s, stubEntityPool())
				second := refresh(t, s, first)
				var e1, e2 map[string]any
				require.NoError(t, json.Unmarshal([]byte(first.Extra), &e1))
				require.NoError(t, json.Unmarshal([]byte(second.Extra), &e2))
				delete(e1, "PriceOracleTimestamp")
				delete(e2, "PriceOracleTimestamp")
				require.Equal(t, e1, e2)
				if v != "v2.0.0" {
					require.EqualValues(t, 7, e2["LastTimestamp"])
				}
			})
		}
	}
}

// Extra stored before the gate (any field set, no Version) still decodes and still bootstraps.
func TestStoredVersionOldExtra(t *testing.T) {
	require.Equal(t, "", twocryptong.StoredVersion(""))
	require.Equal(t, "", twocryptong.StoredVersion("not json"))
	require.Equal(t, "", twocryptong.StoredVersion(`{"D":"1","PolicyAddress":"0xab"}`))
	require.Equal(t, "v2.1.0d", twocryptong.StoredVersion(`{"Version":"v2.1.0d","LastTimestamp":3}`))
	old := `{"InitialA":"400000","D":"1000","PriceScale":["1"],"PriceOracle":["1"],"LastPrices":["1"],` +
		`"FeeGamma":"1","MidFee":"1","OutFee":"1","LpSupply":"1","XcpProfit":"1","VirtualPrice":"1",` +
		`"AllowedExtraProfit":"1","AdjustmentStep":"1"}`
	var e twocryptong.Extra
	require.NoError(t, json.Unmarshal([]byte(old), &e))
	require.Equal(t, "", e.Version)
	require.Equal(t, "1000", e.D.Dec())
}
