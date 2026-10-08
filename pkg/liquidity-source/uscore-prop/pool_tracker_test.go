package uscoreprop

import (
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

func testEntity() entity.Pool {
	return entity.Pool{Address: testPool, Exchange: DexType, Type: DexType, Reserves: entity.PoolReserves{"0", "0"}, Extra: "{}",
		Tokens: []*entity.PoolToken{{Address: testQuote, Decimals: 6, Swappable: true}, {Address: testBase, Decimals: 18, Swappable: true}}}
}

var testUnits = [2]*uint256.Int{big256.TenPow(6), big256.TenPow(18)}

// Pools shaped like the live ones: USDC/WHYPE at ~91.7 USDC per WHYPE with a fee that peaks the
// output, a USDT0 side too shallow to pay out one WHYPE (exercises the halving probe), a pool
// that can't pay out one direction at all, and a paused pool.
var testPools = map[string]mockPool{
	"deep": {reserves: [2]*uint256.Int{uint256.NewInt(1e9), big256.TenPow(19)},
		curves: [2]mockCurve{{num: 109e8, den: 1, peak: 8e8}, {num: 917, den: 1e13, peak: 9e18}}},
	"shallow": {reserves: [2]*uint256.Int{uint256.NewInt(50e6), big256.TenPow(20)},
		curves: [2]mockCurve{{num: 109e8, den: 1, peak: 8e9}, {num: 917, den: 1e13, peak: 1e18}}},
	"one-sided": {reserves: [2]*uint256.Int{uint256.NewInt(0), big256.TenPow(24)},
		curves: [2]mockCurve{{num: 109e8, den: 1, peak: 5e11}, {num: 917, den: 1e13, peak: 8e18}}},
	"paused": {reserves: [2]*uint256.Int{uint256.NewInt(1e12), big256.TenPow(24)},
		curves: [2]mockCurve{{num: 109e8, den: 1, peak: 5e11}, {num: 917, den: 1e13, peak: 8e18}}, paused: true},
}

// USCoreSampler must sample exactly what the Go reference does (same points, same block), in one
// eth_call: that's what lets a refresh cost one RPC call without losing accuracy.
func TestSamplerMatchesReference(t *testing.T) {
	for name, m := range testPools {
		t.Run(name, func(t *testing.T) {
			rpc := newEVMRPC(t, m)
			s, err := sample(t.Context(), rpc.client, common.HexToAddress(testPool),
				[2]common.Address{common.HexToAddress(testQuote), common.HexToAddress(testBase)}, testUnits, nil)
			require.NoError(t, err)
			require.EqualValues(t, 1, rpc.calls.Load())
			ref := referenceSample(mockQuoter{m}, testUnits)
			require.Equal(t, uint64(evmBlock), s.BlockNumber.Uint64())
			require.Equal(t, int64(evmTimestamp), s.Timestamp.Int64())
			require.Equal(t, ref.reserves[0].ToBig(), s.ReserveQuote)
			require.Equal(t, ref.reserves[1].ToBig(), s.ReserveBase)
			require.Equal(t, ref.rounds, int(s.Rounds))
			for i, got := range [2]struct {
				points, outs []*big.Int
				status       uint8
			}{{s.Points0, s.Outs0, s.Status0}, {s.Points1, s.Outs1, s.Status1}} {
				require.Equal(t, ref.points[i], big256.MustFromBigs(got.points), "points dir %d", i)
				require.Equal(t, ref.quotes[i].status, got.status, "status dir %d", i)
				require.Equal(t, ref.quotes[i].outs, big256.MustFromBigs(got.outs), "outs dir %d", i)
			}
			t.Logf("rounds=%d points=%d/%d gas=%d", s.Rounds, len(s.Points0), len(s.Points1), rpc.gasUsed.Load())
		})
	}
}

// The tracker turns the sampler's last round into the ladder, the reserves and the block, and the
// simulator then reproduces the pool's quote exactly at every sampled amount.
func TestTrackerOneCallLadder(t *testing.T) {
	m := testPools["deep"]
	rpc := newEVMRPC(t, m)
	p, err := NewPoolTracker(&Config{}, rpc.client).GetNewPoolState(t.Context(), testEntity(), pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	require.EqualValues(t, 1, rpc.calls.Load())
	require.Equal(t, uint64(evmBlock), p.BlockNumber)
	require.Equal(t, int64(evmTimestamp), p.Timestamp)
	require.Equal(t, entity.PoolReserves{"1000000000", "10000000000000000000"}, p.Reserves)
	sim, err := NewPoolSimulatorWith(p, math.MaxInt64)
	require.NoError(t, err)
	require.Equal(t, ladder.PoolMeta{BlockNumber: evmBlock}, sim.GetMetaInfo(testQuote, testBase))
	tokens := [2]string{testQuote, testBase}
	for dir, points := range ladders(t, p) {
		require.Greater(t, len(points), ladder.SampleSize)
		for _, pt := range points {
			want, status := m.quote(dir, &pt[0])
			require.Equal(t, statusOK, int(status))
			got, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: tokens[dir], Amount: pt[0].ToBig()}, TokenOut: tokens[1-dir]})
			require.NoError(t, err)
			require.Equal(t, want.ToBig(), got.TokenAmountOut.Amount)
		}
	}
}

func TestTrackerOverridesReachTheCall(t *testing.T) {
	rpc := newEVMRPC(t, testPools["deep"])
	_, err := NewPoolTracker(&Config{}, rpc.client).GetNewPoolStateWithOverrides(t.Context(), testEntity(),
		pool.GetNewPoolStateWithOverridesParams{
			Overrides: map[common.Address]gethclient.OverrideAccount{common.HexToAddress(testPool): {Nonce: 1}}})
	require.NoError(t, err)
	require.EqualValues(t, 1, rpc.overrides.Load())
}

// An unquotable direction offers no liquidity: its ladder is empty and its output reserve is 0.
func TestTrackerUnquotableDirections(t *testing.T) {
	for name, want := range map[string]entity.PoolReserves{"paused": {"0", "0"}, "one-sided": {"0", "1000000000000000000000000"}} {
		t.Run(name, func(t *testing.T) {
			p, err := NewPoolTracker(&Config{}, newEVMRPC(t, testPools[name]).client).GetNewPoolState(t.Context(),
				testEntity(), pool.GetNewPoolStateParams{})
			require.NoError(t, err)
			require.Equal(t, want, p.Reserves)
			l := ladders(t, p)
			require.Empty(t, l[1])
			require.Equal(t, name == "paused", len(l[0]) == 0)
			sim, err := NewPoolSimulatorWith(p, math.MaxInt64)
			require.NoError(t, err)
			_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: testBase, Amount: big.NewInt(1e18)}, TokenOut: testQuote})
			require.ErrorIs(t, err, ladder.ErrNoQuote)
		})
	}
}

// Sampling depends only on the pool, not on what was stored before, and identical quotes give an
// identical Extra, so pool-service can skip the write (its heartbeat keeps the Timestamp fresh).
func TestTrackerRefreshIsDeterministic(t *testing.T) {
	tracker := NewPoolTracker(&Config{}, newEVMRPC(t, testPools["deep"]).client)
	first, err := tracker.GetNewPoolState(t.Context(), testEntity(), pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	for _, prior := range []entity.Pool{first, {Extra: "{", Reserves: entity.PoolReserves{"1", "1"}}} {
		p := testEntity()
		p.Extra, p.Reserves = prior.Extra, prior.Reserves
		again, err := tracker.GetNewPoolState(t.Context(), p, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
		require.Equal(t, first.Extra, again.Extra)
		require.Equal(t, first.Reserves, again.Reserves)
	}
}

func TestTrackerFactory(t *testing.T) {
	factory := pooltrack.Factory(DexType)
	require.NotNil(t, factory)
	tracker, err := factory(DexType, pooltrack.FactoryParams{
		Dependencies: pooltrack.Dependencies{EthrpcClient: newEVMRPC(t, testPools["deep"]).client}})
	require.NoError(t, err)
	_, err = tracker.GetNewPoolState(t.Context(), testEntity(), pool.GetNewPoolStateParams{})
	require.NoError(t, err)
}

func TestDecodeSnapshotRejectsOtherReverts(t *testing.T) {
	for _, data := range [][]byte{nil, {1, 2, 3}, {0xde, 0xad, 0xbe, 0xef, 0}} {
		_, err := decodeSnapshot(data)
		require.ErrorIs(t, err, ErrUnexpectedSamplerRevert)
	}
	_, err := toLadderQuote(testAmounts(1), nil, 0)
	require.ErrorIs(t, err, ErrInvalidState)
	_, err = toLadderQuote(testAmounts(1), []*big.Int{big.NewInt(1)}, maxStatus+1)
	require.ErrorIs(t, err, ErrInvalidState)
}

// A revert without data is a transient node failure on HyperEVM: retried once, then reported.
func TestTrackerRetriesBareRevertOnce(t *testing.T) {
	for reverts, wantErr := range map[int32]bool{1: false, 2: true} {
		rpc := newEVMRPC(t, testPools["deep"])
		rpc.bareReverts.Store(reverts)
		_, err := NewPoolTracker(&Config{}, rpc.client).GetNewPoolState(t.Context(), testEntity(), pool.GetNewPoolStateParams{})
		require.Equal(t, wantErr, err != nil, "reverts=%d err=%v", reverts, err)
		require.EqualValues(t, 2, rpc.calls.Load())
	}
}
