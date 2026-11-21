package mhocdesk

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

const tPool = "0x00000000000000000000000000000000000000a1"

func bi(s string) *big.Int { b, _ := new(big.Int).SetString(s, 10); return b }

// stubFor answers the tracker's and the list updater's reads from a vector snapshot, as the pool would.
func stubFor(t *testing.T, p vecPool, s vecSnapshot) func(common.Address, []byte) (bool, []byte) {
	return func(to common.Address, data []byte) (bool, []byte) {
		target := strings.ToLower(to.Hex())
		if target == tBase || target == tQuote {
			m, err := abi.Erc20ABI.MethodById(data)
			require.NoError(t, err)
			switch {
			case m.Name == "balanceOf" && target == tBase:
				return true, must(m.Outputs.Pack(bi(s.BalanceBase)))
			case m.Name == "balanceOf":
				return true, must(m.Outputs.Pack(bi(s.BalanceQuote)))
			case m.Name == "decimals" && target == tBase:
				return true, must(m.Outputs.Pack(p.BaseDecimals))
			case m.Name == "decimals":
				return true, must(m.Outputs.Pack(p.QuoteDecimals))
			}
			return false, nil
		}
		require.Equal(t, tPool, target)
		m, err := propAMMABI.MethodById(data)
		require.NoError(t, err)
		if m.Name == "counters" {
			return true, must(m.Outputs.Pack(s.CardSeq, bi(s.CardFilled), s.FillBlock, bi(s.BlockFilled), s.WindowID,
				bi(s.WindowFilled)))
		}
		out := map[string]any{
			"stateWord": bi(s.StateWord), "floorBase": bi(s.FloorBase), "floorQuote": bi(s.FloorQuote),
			"maxNotional": bi(s.MaxNotional).Uint64(), "maxNotionalPerUpdate": bi(s.MaxNotionalPerUpdate).Uint64(),
			"maxNotionalPerWindow": uint32(bi(s.MaxNotionalPerWindow).Uint64()), "BASE": common.HexToAddress(tBase),
			"QUOTE": common.HexToAddress(tQuote), "PRICE_SCALE": bi(p.PriceScale),
			"K_SCALE": big.NewInt(int64(p.KScale)),
		}[m.Name]
		require.NotNil(t, out, m.Name)
		return true, must(m.Outputs.Pack(out))
	}
}

func must(b []byte, err error) []byte {
	if err != nil {
		panic(err)
	}
	return b
}

// The list updater emits each configured pool once, with its immutables; the tracker then reads the snapshot
// through the views, and the simulator built from the tracked entity reproduces the scenario's contract quotes in
// the snapshot's block and in the next block.
func TestListUpdaterTrackerSimulator(t *testing.T) {
	t.Parallel()
	v := loadVectors(t)
	for _, p := range v.Pools {
		for _, sc := range p.Scenarios[:3] {
			stub := testutil.NewMulticallStub(t, stubFor(t, p, sc.Snapshot)).
				At(sc.Snapshot.BlockNumber, sc.Snapshot.ReadTimestamp)

			lu := NewPoolsListUpdater(&Config{DexID: DexType, Pools: []string{tPool, "not-an-address"}}, stub.Client())
			pools, md, err := lu.GetNewPools(t.Context(), nil)
			require.NoError(t, err)
			require.Len(t, pools, 1)
			require.Equal(t, tBase, pools[0].Tokens[0].Address)
			require.Equal(t, tQuote, pools[0].Tokens[1].Address)
			again, _, err := lu.GetNewPools(t.Context(), md)
			require.NoError(t, err)
			require.Empty(t, again, "a pool is emitted once")

			tr, _ := NewPoolTracker(&Config{DexID: DexType}, stub.Client())
			tracked, err := tr.GetNewPoolState(t.Context(), pools[0], pool.GetNewPoolStateParams{})
			require.NoError(t, err)
			require.Equal(t, sc.Snapshot.BlockNumber, tracked.BlockNumber)
			require.Equal(t, entity.PoolReserves{sc.Snapshot.BalanceBase, sc.Snapshot.BalanceQuote}, tracked.Reserves)
			var got Extra
			require.NoError(t, json.Unmarshal([]byte(tracked.Extra), &got))
			require.Equal(t, snapshotExtra(sc.Snapshot), got)

			sim, err := NewPoolSimulator(tracked)
			require.NoError(t, err)
			label := sc.Name + " tracked"
			checkQuotes(t, sim, label+" same block", sc.Quotes, sc.QuoteBlockNumber, sc.QuoteTimestamp)
			checkQuotes(t, sim, label+" next block", sc.NextQuotes, sc.NextBlockNumber, sc.NextTimestamp)
		}
	}
}

// TestListUpdaterManyPools: the list updater takes a list of pools (one PropAMM per pair) and, when a registry is
// configured, merges registry.pools() into it. Each pool is emitted once with its own immutables; a pool added to
// the configuration or to the registry later is picked up on the next run; a pool in both is emitted once.
func TestListUpdaterManyPools(t *testing.T) {
	t.Parallel()
	const (
		a1, a2, a3, a4 = "0x00000000000000000000000000000000000000a1", "0x00000000000000000000000000000000000000a2",
			"0x00000000000000000000000000000000000000a3", "0x00000000000000000000000000000000000000a4"
		c1, c2, c3 = "0x00000000000000000000000000000000000000c1", "0x00000000000000000000000000000000000000c2",
			"0x00000000000000000000000000000000000000c3"
		registry = "0x00000000000000000000000000000000000000f1"
	)
	pools := map[string][2]string{a1: {tBase, tQuote}, a2: {c1, tQuote}, a3: {c2, tQuote}, a4: {c3, tQuote}}
	kScale := map[string]int64{a1: 4, a2: 4, a3: 1, a4: 256}
	decimals := map[string]uint8{tBase: 18, tQuote: 6, c1: 8, c2: 18, c3: 6}
	listed := []common.Address{common.HexToAddress(a2)} // what registry.pools() returns; grows below
	stub := testutil.NewMulticallStub(t, func(to common.Address, data []byte) (bool, []byte) {
		target := strings.ToLower(to.Hex())
		if target == registry {
			m, err := registryABI.MethodById(data)
			require.NoError(t, err)
			require.Equal(t, "pools", m.Name)
			return true, must(m.Outputs.Pack(listed))
		}
		if d, ok := decimals[target]; ok {
			m, err := abi.Erc20ABI.MethodById(data)
			require.NoError(t, err)
			require.Equal(t, "decimals", m.Name)
			return true, must(m.Outputs.Pack(d))
		}
		toks, ok := pools[target]
		require.True(t, ok, target)
		m, err := propAMMABI.MethodById(data)
		require.NoError(t, err)
		out := map[string]any{"BASE": common.HexToAddress(toks[0]), "QUOTE": common.HexToAddress(toks[1]),
			"PRICE_SCALE": big.NewInt(1e8), "K_SCALE": big.NewInt(kScale[target])}[m.Name]
		require.NotNil(t, out, m.Name)
		return true, must(m.Outputs.Pack(out))
	}).At(100, 1_800_000_000)

	check := func(got []entity.Pool, want ...string) {
		t.Helper()
		require.Len(t, got, len(want))
		for i, p := range got {
			require.Equal(t, want[i], p.Address)
			require.Equal(t, pools[p.Address][0], p.Tokens[0].Address)
			require.Equal(t, pools[p.Address][1], p.Tokens[1].Address)
			require.Equal(t, decimals[p.Tokens[0].Address], p.Tokens[0].Decimals)
			var se StaticExtra
			require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &se))
			require.EqualValues(t, kScale[p.Address], se.KScale)
			require.Equal(t, "100000000", se.PriceScale)
		}
	}

	// static list only (mixed-case address), then one more configured pool
	cfg := &Config{DexID: DexType, Pools: []string{"0x00000000000000000000000000000000000000A1"}}
	lu := NewPoolsListUpdater(cfg, stub.Client())
	got, md, err := lu.GetNewPools(t.Context(), nil)
	require.NoError(t, err)
	check(got, a1)
	cfg.Pools = append(cfg.Pools, a3)
	got, md, err = lu.GetNewPools(t.Context(), md)
	require.NoError(t, err)
	check(got, a3)

	// with a registry: its pools are merged with the static list (a2 is new; a1 and a3 were emitted already)
	cfg.Registry = registry
	got, md, err = lu.GetNewPools(t.Context(), md)
	require.NoError(t, err)
	check(got, a2)
	// the maker lists a new pair, and one that is also in the static list: only the new one is emitted
	listed = append(listed, common.HexToAddress(a4), common.HexToAddress(a1))
	got, md, err = lu.GetNewPools(t.Context(), md)
	require.NoError(t, err)
	check(got, a4)
	got, _, err = lu.GetNewPools(t.Context(), md)
	require.NoError(t, err)
	require.Empty(t, got)

	// a registry and no static list at all, from scratch: every listed pool, once
	lu = NewPoolsListUpdater(&Config{DexID: DexType, Registry: registry}, stub.Client())
	got, _, err = lu.GetNewPools(t.Context(), nil)
	require.NoError(t, err)
	check(got, a2, a4, a1)
}
