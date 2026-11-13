package everlongflamm

import (
	"math/big"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	testPool = "0x00000000000000000000000000000000000000aa"
	cbBTC    = "0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf"
	usdc     = "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"
	usdt     = "0xfde4c96c8593536e31f229ea8f37b2ada2699bb2"
)

// testEntity is a three-token pool: cbBTC (the pool asset), USDC and USDT. cbBTC and USDT both buy USDC; USDC
// buys cbBTC (capped); USDT is paid by nothing.
func testEntity(ts int64) entity.Pool {
	extra, _ := json.Marshal(Extra{HaircutBps: 1, Ladders: []Ladder{
		{In: 0, Out: 1, Cells: []Cell{{Gas: 600_000}}, Second: 300_000, Points: pts(100_000, 99_000_000, 1_000_000, 985_000_000, 10_000_000, 9_700_000_000)},
		{In: 2, Out: 1, Cells: []Cell{{Gas: 600_000}}, Second: 300_000, Points: pts(1_000_000, 999_000, 1_000_000_000, 998_000_000, 5_000_000_000, 4_980_000_000)},
		{In: 1, Out: 0, Cells: []Cell{{Gas: 650_000}}, Second: 300_000, Points: pts(100_000_000, 99_000, 1_000_000_000, 985_000), Capped: true},
	}})
	return entity.Pool{
		Address: testPool, Exchange: DexType, Type: DexType, Timestamp: ts, BlockNumber: 42,
		Reserves: entity.PoolReserves{"985000", "9700000000", "0"},
		Tokens: []*entity.PoolToken{{Address: cbBTC, Decimals: 8, Swappable: true},
			{Address: usdc, Decimals: 6, Swappable: true}, {Address: usdt, Decimals: 6, Swappable: true}},
		Extra:       string(extra),
		StaticExtra: `{"factory":"0x00000000000000000000000000000000000000ff"}`,
	}
}

func newTestSim(t *testing.T) *PoolSimulator {
	t.Helper()
	s, err := NewPoolSimulator(pool.FactoryParams{EntityPool: testEntity(time.Now().Unix()),
		Opts: pool.FactoryOpts{StaleCheck: true}})
	require.NoError(t, err)
	return s
}

func calc(t *testing.T, s *PoolSimulator, in string, amount int64, out string) (*pool.CalcAmountOutResult, error) {
	t.Helper()
	return s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: in, Amount: big.NewInt(amount)}, TokenOut: out})
}

func fill(t *testing.T, s *PoolSimulator, in string, amount int64, out string) *pool.CalcAmountOutResult {
	t.Helper()
	res, err := calc(t, s, in, amount, out)
	require.NoError(t, err)
	s.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: in, Amount: big.NewInt(amount)},
		TokenAmountOut: *res.TokenAmountOut, Fee: *res.Fee, SwapInfo: res.SwapInfo,
	})
	return res
}

func TestSimulatorQuotesProbesExactly(t *testing.T) {
	s := newTestSim(t)
	res, err := calc(t, s, cbBTC, 1_000_000, usdc)
	require.NoError(t, err)
	require.Equal(t, int64(985_000_000), res.TokenAmountOut.Amount.Int64())
	require.Zero(t, res.RemainingTokenAmountIn.Amount.Sign())
	require.Equal(t, int64(600_000), res.Gas)

	b, err := json.Marshal(res.SwapInfo)
	require.NoError(t, err)
	require.JSONEq(t, `{"venue":0,"poolAssetIn":true,"amountInUsed":"1000000","amountOut":"985000000"}`, string(b))
	require.Equal(t, PoolMeta{ApprovalAddress: testPool, BlockNumber: 42}, s.GetMetaInfo(cbBTC, usdc))

	// between probes: the chord less 1 bp
	res, err = calc(t, s, cbBTC, 5_500_000, usdc)
	require.NoError(t, err)
	require.Equal(t, int64((985_000_000+(9_700_000_000-985_000_000)*4_500_000/9_000_000)*9_999/10_000),
		res.TokenAmountOut.Amount.Int64())
}

func TestSimulatorFillsAtTheCeilingAndReturnsTheRest(t *testing.T) {
	s := newTestSim(t)
	res, err := calc(t, s, usdc, 3_000_000_000, cbBTC)
	require.NoError(t, err)
	require.Equal(t, int64(985_000), res.TokenAmountOut.Amount.Int64())
	require.Equal(t, int64(2_000_000_000), res.RemainingTokenAmountIn.Amount.Int64())
	require.Equal(t, "1000000000", res.SwapInfo.(SwapInfo).AmountInUsed)
	require.False(t, res.SwapInfo.(SwapInfo).PoolAssetIn)

	_, err = calc(t, s, cbBTC, 10_000_001, usdc)
	require.ErrorIs(t, err, ladder.ErrAmountInTooLarge)
	_, err = calc(t, s, usdc, 1_000_000, usdt)
	require.ErrorIs(t, err, ladder.ErrNoQuote, "a direction with no ladder")
	_, err = calc(t, s, usdc, 0, cbBTC)
	require.Error(t, err)
}

func TestSimulatorWalksTheLadderAndSharesThePoolsRoom(t *testing.T) {
	s := newTestSim(t)
	before := s.CloneState().(*PoolSimulator)

	first := fill(t, s, cbBTC, 1_000_000, usdc)
	require.Equal(t, int64(985_000_000), first.TokenAmountOut.Amount.Int64())

	// the same direction takes no second fill in the route
	_, err := calc(t, s, cbBTC, 1_000_000, usdc)
	require.ErrorIs(t, err, ladder.ErrNoQuote)

	// USDT also pays out of the USDC room the cbBTC fill drew down
	require.Equal(t, "8715000000", s.GetReserves()[1].String())
	_, err = calc(t, s, usdt, 5_000_000_000, usdc)
	require.NoError(t, err)
	drained := before.CloneState().(*PoolSimulator)
	fill(t, drained, cbBTC, 10_000_000, usdc) // the cbBTC ladder's top: the room left is 0
	_, err = calc(t, drained, usdt, 1_000_000, usdc)
	require.ErrorIs(t, err, ladder.ErrInsufficientLiquidity)

	// the clone taken before is untouched
	res, err := calc(t, before, cbBTC, 1_000_000, usdc)
	require.NoError(t, err)
	require.Equal(t, int64(985_000_000), res.TokenAmountOut.Amount.Int64())
	require.Equal(t, "9700000000", before.GetReserves()[1].String())

	// the capped direction, once at its ceiling, quotes nothing more
	fill(t, s, usdc, 2_000_000_000, cbBTC)
	_, err = calc(t, s, usdc, 1_000_000, cbBTC)
	require.ErrorIs(t, err, ladder.ErrNoQuote)
}

// The chain charges a second fill in a direction its own fee on the book the first moved, so a route's direction takes
// one fill, on a ladder of flat spans, of chords or of steeper steps alike; the other direction keeps its own first, and
// two fills one each way are both quoted. The fill is carried by a clone.
func TestSimulatorTakesOneFillPerDirection(t *testing.T) {
	for name, tc := range map[string]struct {
		points []Point
		flat   uint64
	}{
		"a flat span":               {points: pts(100, 990, 200, 1970, 400, 3900), flat: 1 << 2},
		"chords":                    {points: pts(100, 990, 200, 1970, 400, 3900)},
		"a span only steeper steps": {points: pts(100, 900, 200, 1900, 400, 3900)},
	} {
		t.Run(name, func(t *testing.T) {
			p := testEntity(time.Now().Unix())
			extra, err := json.Marshal(Extra{HaircutBps: 1, Ladders: []Ladder{
				{In: 0, Out: 1, Cells: []Cell{{Gas: 600_000}}, Second: 300_000, Points: tc.points, Flat: tc.flat},
				{In: 1, Out: 0, Cells: []Cell{{Gas: 650_000}}, Second: 300_000, Points: pts(100_000_000, 99_000, 1_000_000_000, 985_000)},
			}})
			require.NoError(t, err)
			p.Extra = string(extra)
			s, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
			require.NoError(t, err)
			first := fill(t, s, cbBTC, 150, usdc)
			require.Positive(t, first.TokenAmountOut.Amount.Sign())
			for _, amount := range []int64{100, 150, 200} {
				_, err = calc(t, s, cbBTC, amount, usdc)
				require.ErrorIs(t, err, ladder.ErrNoQuote, "a second fill in the direction, at %d", amount)
			}
			_, err = calc(t, s.CloneState().(*PoolSimulator), cbBTC, 100, usdc)
			require.ErrorIs(t, err, ladder.ErrNoQuote, "the clone carries the fill")
			back := fill(t, s, usdc, 100_000_000, cbBTC)
			require.Positive(t, back.TokenAmountOut.Amount.Sign(), "the other direction keeps its first fill")
			_, err = calc(t, s, usdc, 100_000_000, cbBTC)
			require.ErrorIs(t, err, ladder.ErrNoQuote, "and takes no second")
		})
	}
}

// A refused second fill leaves the pool as it was: the quote, the room and the other direction's quote stand.
func TestSimulatorRefusedSecondFillChangesNothing(t *testing.T) {
	s := newTestSim(t)
	fill(t, s, cbBTC, 1_000_000, usdc)
	room := s.GetReserves()[1].String()
	_, err := calc(t, s, cbBTC, 1_000_000, usdc)
	require.ErrorIs(t, err, ladder.ErrNoQuote)
	require.Equal(t, room, s.GetReserves()[1].String())
	back, err := calc(t, s, usdc, 1_000_000_000, cbBTC)
	require.NoError(t, err)
	require.Positive(t, back.TokenAmountOut.Amount.Sign())
}

func TestSimulatorRoutesOnlyLadderedDirections(t *testing.T) {
	s := newTestSim(t)
	require.ElementsMatch(t, []string{cbBTC, usdt}, s.CanSwapTo(usdc))
	require.ElementsMatch(t, []string{usdc}, s.CanSwapTo(cbBTC))
	require.Empty(t, s.CanSwapTo(usdt))
	require.ElementsMatch(t, []string{cbBTC}, s.CanSwapFrom(usdc))
	require.ElementsMatch(t, []string{usdc}, s.CanSwapFrom(usdt))
	require.Nil(t, s.CanSwapTo("0x0000000000000000000000000000000000000001"))
}

func TestSimulatorStaleOnlyWithStaleCheck(t *testing.T) {
	old := testEntity(time.Now().Add(-2 * MaxAge).Unix())
	_, err := NewPoolSimulator(pool.FactoryParams{EntityPool: old, Opts: pool.FactoryOpts{StaleCheck: true}})
	require.ErrorIs(t, err, ladder.ErrStale)
	_, err = NewPoolSimulator(pool.FactoryParams{EntityPool: old})
	require.NoError(t, err)
}

func TestSimulatorRefusesMalformedState(t *testing.T) {
	for name, extra := range map[string]string{
		"a token past the list": `{"l":[{"i":0,"j":3,"p":[["1","1"]],"g":1}],"h":1}`,
		"a pair twice":          `{"l":[{"i":0,"j":1,"p":[["1","1"]],"g":1},{"i":0,"j":1,"p":[["2","2"]],"g":1}],"h":1}`,
		"a haircut past 100 %":  `{"l":[],"h":10001}`,
		"no json":               `x`,
	} {
		p := testEntity(time.Now().Unix())
		p.Extra = extra
		_, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
		require.Error(t, err, name)
	}
	p := testEntity(time.Now().Unix())
	p.Reserves = p.Reserves[:2]
	_, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
	require.ErrorIs(t, err, ErrInvalidState)
}

// The pool pulls the input with transferFrom, so it is the address a taker approves.
func TestSimulatorApprovalAddressIsThePool(t *testing.T) {
	s := newTestSim(t)
	require.NotEmpty(t, s.Info.Address)
	require.Equal(t, s.Info.Address, s.GetApprovalAddress(cbBTC, usdc))
}

// A fill whose SwapInfo cannot be read still happened: the direction takes no second fill.
func TestSimulatorUpdateBalanceWithWrongSwapInfoStillMarksTheDirection(t *testing.T) {
	s := newTestSim(t)
	res, err := calc(t, s, cbBTC, 1_000_000, usdc)
	require.NoError(t, err)
	s.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: cbBTC, Amount: big.NewInt(1_000_000)},
		TokenAmountOut: *res.TokenAmountOut, Fee: *res.Fee, SwapInfo: "not a SwapInfo",
	})
	_, err = calc(t, s, cbBTC, 1_000_000, usdc)
	require.ErrorIs(t, err, ladder.ErrNoQuote)
	back, err := calc(t, s, usdc, 1_000_000_000, cbBTC)
	require.NoError(t, err, "the other direction keeps its first fill")
	require.Positive(t, back.TokenAmountOut.Amount.Sign())
}
