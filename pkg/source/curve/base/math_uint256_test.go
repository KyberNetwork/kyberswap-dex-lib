package base

import (
	"testing"
	"time"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// golden values were captured from the big.Int math before the uint256 port. They pin the uint256 path to the
// legacy results. The pool is the two-coin fixture below.
func goldenP2(t *testing.T) *PoolSimulator {
	t.Helper()
	p, err := NewPoolSimulator(entity.Pool{
		Reserves: entity.PoolReserves{"101940884", "107546110", "208092128367874420986"},
		Tokens:   []*entity.PoolToken{{Address: "T0"}, {Address: "T1"}},
		Extra:    `{"swapFee":"3000000","adminFee":"5000000000","initialA":"150000","futureA":"150000"}`,
		StaticExtra: `{"lpToken":"LP","aPrecision":"100","precisionMultipliers":["1000000000000","1000000000000"],` +
			`"rates":["1000000000000000000000000000000","1000000000000000000000000000000"]}`,
	})
	require.NoError(t, err)
	return p
}

func TestGoldenVirtualPrice(t *testing.T) {
	t.Parallel()
	p := goldenP2(t)
	v, d, err := p.GetVirtualPrice()
	require.NoError(t, err)
	require.Equal(t, "1006702875537675272", v.Dec())
	require.Equal(t, "209486944004694229067", d.Dec())
}

func TestGoldenGetDy(t *testing.T) {
	t.Parallel()
	p := goldenP2(t)
	for _, tc := range []struct {
		i, j        int
		dx, dy, fee string
	}{
		{0, 1, "1000", "999", "0"},
		{0, 1, "5000000", "4998519", "1500"},
		{0, 1, "100000000000", "107513846", "32263"},
		{1, 0, "1000", "999", "0"},
		{1, 0, "5000000", "4998160", "1499"},
		{1, 0, "100000000000", "101910301", "30582"},
	} {
		var dx uint256.Int
		require.NoError(t, dx.SetFromDecimal(tc.dx))
		dy, fee, err := p.GetDy(tc.i, tc.j, &dx, nil)
		require.NoError(t, err)
		require.Equal(t, tc.dy, dy.Dec(), "dy %d->%d dx=%s", tc.i, tc.j, tc.dx)
		require.Equal(t, tc.fee, fee.Dec(), "fee %d->%d dx=%s", tc.i, tc.j, tc.dx)
	}
}

func TestGoldenCalculateTokenAmount(t *testing.T) {
	t.Parallel()
	p := goldenP2(t)
	deposit := []*uint256.Int{uint256.NewInt(1000000), uint256.NewInt(2000000)}
	got, err := p.CalculateTokenAmount(deposit, true)
	require.NoError(t, err)
	require.Equal(t, "2980006917492527526", got.Dec())

	got, err = p.CalculateTokenAmount(deposit, false)
	require.NoError(t, err)
	require.Equal(t, "2980009601820768067", got.Dec())
}

func TestGoldenCalculateWithdrawOneCoin(t *testing.T) {
	t.Parallel()
	p := goldenP2(t)
	burn, err := uint256.FromDecimal("50000000000000000000")
	require.NoError(t, err)
	for _, tc := range []struct {
		i       int
		dy, fee string
	}{
		{0, "50319975", "7749"},
		{1, "50323302", "7346"},
	} {
		dy, fee, err := p.CalculateWithdrawOneCoin(burn, tc.i)
		require.NoError(t, err)
		require.Equal(t, tc.dy, dy.Dec(), "dy coin %d", tc.i)
		require.Equal(t, tc.fee, fee.Dec(), "fee coin %d", tc.i)
	}
}

func TestGoldenAddLiquidity(t *testing.T) {
	t.Parallel()
	p := goldenP2(t)
	mint, err := p.AddLiquidity([]*uint256.Int{uint256.NewInt(5000000), uint256.NewInt(10000000)})
	require.NoError(t, err)
	require.Equal(t, "14899326398466847830", mint.Dec())
	require.Equal(t, "106940712", p.Info.Reserves[0].Dec())
	require.Equal(t, "117545938", p.Info.Reserves[1].Dec())
	require.Equal(t, "222991454766341268816", p.LpSupply.Dec())
}

func TestGoldenRemoveLiquidityOneCoin(t *testing.T) {
	t.Parallel()
	p := goldenP2(t)
	burn, err := uint256.FromDecimal("50000000000000000000")
	require.NoError(t, err)
	dy, err := p.RemoveLiquidityOneCoin(burn, 0)
	require.NoError(t, err)
	require.Equal(t, "50319975", dy.Dec())
	require.Equal(t, "51617035", p.Info.Reserves[0].Dec())
	require.Equal(t, "107546110", p.Info.Reserves[1].Dec())
	require.Equal(t, "158092128367874420986", p.LpSupply.Dec())
}

// TestAu256Ramp pins the A ramp to the on-chain interpolation. A moves linearly from initialA to futureA over
// [initialATime, futureATime], and the uint256 port must match it exactly.
func TestAu256Ramp(t *testing.T) {
	t.Parallel()
	now := time.Now().Unix()
	p := &PoolSimulator{
		InitialA:     uint256.NewInt(100000),
		FutureA:      uint256.NewInt(200000),
		InitialATime: 0,
		FutureATime:  now * 2,
	}
	got, err := p._AU256(now)
	require.NoError(t, err)
	require.Equal(t, "150000", got.Dec())

	// Ramp down: 200k at zero to 100k at 2*now, so the midpoint is 150k.
	p.InitialA, p.FutureA = uint256.NewInt(200000), uint256.NewInt(100000)
	got, err = p._AU256(now)
	require.NoError(t, err)
	require.Equal(t, "150000", got.Dec())

	// A ramp that has not started yet fails on chain (block.timestamp - initial_A_time underflows).
	p.InitialATime = now + 10
	_, err = p._AU256(now)
	require.ErrorIs(t, err, errU256Underflow)
}
