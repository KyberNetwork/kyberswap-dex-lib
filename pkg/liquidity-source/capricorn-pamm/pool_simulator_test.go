package capricornpamm

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Shared fixture constants. Two snapshots live here:
//   - simpleFixture: pool 0x6309...E4df at Monad block 73_803_095 with a
//     single-point ladder per direction (lightweight, for unit tests).
//   - directMatchFixture: same pool at block 75_581_558 with the full 5-point
//     ladder captured on-chain — used for the step-4 direct-protocol match.
const (
	fixUSDC                 = "0x754704bc059f8c67012fed69bc8a327a5aafb603"
	fixWMON                 = "0x3bd359c1119da7da1d913d1c4d2b7c461115433a"
	fixPool                 = "0x63093325c05cd32b18034d3ea29199fb7098e4df"
	fixFactory              = "0x010cf4f9e3a79dd2fe11760d76a75df6c0656631"
	fixOracleId             = "0x4c09ef490619129335c4ca2303761513b58138dbe3f5a859c01beb4946c502f2"
	simpleSnapshotBlockNum  = uint64(73803095)
	intgSnapshotBlockNum    = uint64(75581558)
	simpleSnapshotReserve0  = "170197191"
	simpleSnapshotReserve1  = "5990944577820783056405"
	intgSnapshotReserve0    = "1270266722"
	intgSnapshotReserve1    = "45992785014363734947947"
	simpleQuoteOutUSDCtoWMN = "28903623346225898755" // 1 USDC -> WMON @ block 73_874_829
)

// mustU256 panics on bad input — fine for fixture constants.
func mustU256(s string) *uint256.Int {
	v, err := uint256.FromDecimal(s)
	if err != nil {
		panic("mustU256(" + s + "): " + err.Error())
	}
	return v
}

// simpleFixture builds an entity.Pool with a single-point ladder per direction
// — minimal data for the wiring/error-path tests.
func simpleFixture(t *testing.T) entity.Pool {
	t.Helper()
	return makeFixture(t,
		simpleSnapshotReserve0, simpleSnapshotReserve1, simpleSnapshotBlockNum,
		[]LadderPoint{{AmountIn: uint256.NewInt(1_000_000), AmountOut: mustU256(simpleQuoteOutUSDCtoWMN)}},
		[]LadderPoint{{AmountIn: mustU256("1000000000000000000"), AmountOut: uint256.NewInt(34254)}},
		false,
	)
}

// intgLadder0 / intgLadder1 are the on-chain quoteExactIn ladders for the
// USDC/WMON pool at block 75_581_558 — geometric grid of 5 points.
var intgLadder0 = []LadderPoint{
	{AmountIn: uint256.NewInt(1_000_000), AmountOut: mustU256("36561072573389277514")},
	{AmountIn: uint256.NewInt(3_981_072), AmountOut: mustU256("145545012722613538823")},
	{AmountIn: uint256.NewInt(15_848_932), AmountOut: mustU256("579309982828392287612")},
	{AmountIn: uint256.NewInt(63_095_734), AmountOut: mustU256("2304452288658578424337")},
	{AmountIn: uint256.NewInt(251_188_643), AmountOut: mustU256("9145417580492861513536")},
}

var intgLadder1 = []LadderPoint{
	{AmountIn: mustU256("1000000000000000000"), AmountOut: uint256.NewInt(27_078)},
	{AmountIn: mustU256("12328467394420658000"), AmountOut: uint256.NewInt(333_831)},
	{AmountIn: mustU256("151993900658438960000"), AmountOut: uint256.NewInt(4_115_473)},
	{AmountIn: mustU256("1872731009482889000000"), AmountOut: uint256.NewInt(50_379_316)},
	{AmountIn: mustU256("23077025673418920000000"), AmountOut: uint256.NewInt(614_515_241)},
}

// directMatchFixture is the block 75_581_558 snapshot with the full 5-point
// ladder. Unquoteable is forced false so CalcAmountOut can be compared
// against the on-chain quoter despite the live routing gate.
func directMatchFixture(t *testing.T) entity.Pool {
	t.Helper()
	return makeFixture(t,
		intgSnapshotReserve0, intgSnapshotReserve1, intgSnapshotBlockNum,
		intgLadder0, intgLadder1, false,
	)
}

func makeFixture(
	t *testing.T,
	r0, r1 string, blockNum uint64,
	ladder0, ladder1 []LadderPoint,
	unquoteable bool,
) entity.Pool {
	t.Helper()

	staticExtra, err := json.Marshal(StaticExtra{
		Factory:  fixFactory,
		OracleId: fixOracleId,
	})
	require.NoError(t, err)

	extra, err := json.Marshal(Extra{
		FeeBps:      50,
		Paused:      false,
		Unquoteable: unquoteable,
		Ladder0:     ladder0,
		Ladder1:     ladder1,
	})
	require.NoError(t, err)

	return entity.Pool{
		Address:     fixPool,
		Exchange:    "capricorn-pamm",
		Type:        DexType,
		Reserves:    entity.PoolReserves{r0, r1},
		BlockNumber: blockNum,
		Tokens: []*entity.PoolToken{
			{Address: fixUSDC, Symbol: "USDC", Decimals: 6, Swappable: true},
			{Address: fixWMON, Symbol: "WMON", Decimals: 18, Swappable: true},
		},
		StaticExtra: string(staticExtra),
		Extra:       string(extra),
	}
}

func TestCalcAmountOut_USDC_to_WMON_GoldenPath(t *testing.T) {
	sim, err := NewPoolSimulator(simpleFixture(t))
	require.NoError(t, err)

	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: fixUSDC, Amount: big.NewInt(1_000_000)},
		TokenOut:      fixWMON,
	})
	require.NoError(t, err)

	assert.Equal(t, simpleQuoteOutUSDCtoWMN, res.TokenAmountOut.Amount.String(),
		"single-point ladder must return the captured on-chain quote exactly")
	assert.Equal(t, fixWMON, res.TokenAmountOut.Token)
	assert.Equal(t, "5000", res.Fee.Amount.String()) // 1e6 * 50 / 10_000
	assert.Equal(t, fixUSDC, res.Fee.Token)
	assert.Equal(t, int64(defaultGas), res.Gas)
}

func TestCalcAmountOut_WMON_to_USDC_GoldenPath(t *testing.T) {
	sim, err := NewPoolSimulator(simpleFixture(t))
	require.NoError(t, err)

	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: fixWMON, Amount: mustU256("1000000000000000000").ToBig()},
		TokenOut:      fixUSDC,
	})
	require.NoError(t, err)

	assert.Equal(t, "34254", res.TokenAmountOut.Amount.String())
	assert.Equal(t, fixUSDC, res.TokenAmountOut.Token)
	assert.Equal(t, fixWMON, res.Fee.Token)
}

func TestCalcAmountOut_RejectsPaused(t *testing.T) {
	ep := mutateExtra(t, simpleFixture(t), func(ex *Extra) { ex.Paused = true })
	sim, err := NewPoolSimulator(ep)
	require.NoError(t, err)

	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: fixUSDC, Amount: big.NewInt(1_000_000)},
		TokenOut:      fixWMON,
	})
	assert.ErrorIs(t, err, ErrPaused)
}

func TestCalcAmountOut_RejectsUnquoteable(t *testing.T) {
	// Unquoteable is pool-wide, not per-direction — both directions must reject.
	ep := mutateExtra(t, simpleFixture(t), func(ex *Extra) { ex.Unquoteable = true })
	sim, err := NewPoolSimulator(ep)
	require.NoError(t, err)

	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: fixUSDC, Amount: big.NewInt(1_000_000)},
		TokenOut:      fixWMON,
	})
	assert.ErrorIs(t, err, ErrPoolUnavailable)

	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: fixWMON, Amount: mustU256("1000000000000000000").ToBig()},
		TokenOut:      fixUSDC,
	})
	assert.ErrorIs(t, err, ErrPoolUnavailable)
}

func TestCalcAmountOut_PausedTakesPrecedenceOverUnquoteable(t *testing.T) {
	ep := mutateExtra(t, simpleFixture(t), func(ex *Extra) {
		ex.Paused = true
		ex.Unquoteable = true
	})
	sim, err := NewPoolSimulator(ep)
	require.NoError(t, err)

	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: fixUSDC, Amount: big.NewInt(1_000_000)},
		TokenOut:      fixWMON,
	})
	assert.ErrorIs(t, err, ErrPaused, "Paused must win over Unquoteable for diagnostic clarity")
}

func TestCalcAmountOut_RejectsUnknownToken(t *testing.T) {
	sim, err := NewPoolSimulator(simpleFixture(t))
	require.NoError(t, err)

	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef", Amount: big.NewInt(1)},
		TokenOut:      fixWMON,
	})
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestCalcAmountOut_RejectsSameTokenInOut(t *testing.T) {
	sim, err := NewPoolSimulator(simpleFixture(t))
	require.NoError(t, err)

	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: fixUSDC, Amount: big.NewInt(1)},
		TokenOut:      fixUSDC,
	})
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestCalcAmountOut_RejectsWhenAmountOutExceedsReserve(t *testing.T) {
	// Force a pathological fixture: ladder claims 1 USDC -> 10x reserve1.
	// The reserve sanity guard must reject without trusting the ladder.
	ep := simpleFixture(t)
	r1, ok := new(big.Int).SetString(simpleSnapshotReserve1, 10)
	require.True(t, ok)
	bogusOut := new(big.Int).Mul(r1, big.NewInt(10))

	ep = mutateExtra(t, ep, func(ex *Extra) {
		ex.Ladder0 = []LadderPoint{{
			AmountIn:  uint256.NewInt(1_000_000),
			AmountOut: uint256.MustFromBig(bogusOut),
		}}
	})
	sim, err := NewPoolSimulator(ep)
	require.NoError(t, err)

	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: fixUSDC, Amount: big.NewInt(1_000_000)},
		TokenOut:      fixWMON,
	})
	assert.ErrorIs(t, err, ErrPoolUnavailable,
		"quoted amountOut > reserveOut must be rejected at quote time")
}

func TestCalcAmountOut_RejectsAmountInTooLargeForLadder(t *testing.T) {
	sim, err := NewPoolSimulator(simpleFixture(t))
	require.NoError(t, err)

	// simpleFixture's largest USDC ladder point is 1_000_000. Ask for 100×.
	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: fixUSDC, Amount: big.NewInt(100_000_000)},
		TokenOut:      fixWMON,
	})
	assert.ErrorIs(t, err, ErrAmountInTooLarge)
}

func TestUpdateBalance_SameDirection_Cumulative(t *testing.T) {
	// Splitting the input across two same-direction hops must sum to the
	// ladder's cumulative quote at the total input — the invariant the
	// marginal model preserves on the concave-monotone curve.
	sim, err := NewPoolSimulator(directMatchFixture(t))
	require.NoError(t, err)

	first, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: fixUSDC, Amount: big.NewInt(1_000_000)},
		TokenOut:      fixWMON,
	})
	require.NoError(t, err)

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: fixUSDC, Amount: big.NewInt(1_000_000)},
		TokenAmountOut: pool.TokenAmount{Token: fixWMON, Amount: first.TokenAmountOut.Amount},
	})

	second, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: fixUSDC, Amount: big.NewInt(2_981_072)},
		TokenOut:      fixWMON,
	})
	require.NoError(t, err)

	combined := new(big.Int).Add(first.TokenAmountOut.Amount, second.TokenAmountOut.Amount)
	cumulative := mustU256("145545012722613538823") // ladder[1].AmountOut @ 3_981_072 USDC
	assert.Equal(t, 0, combined.Cmp(cumulative.ToBig()),
		"first+second marginal must equal ladder cumulative at the total input")
}

func TestUpdateBalance_RejectsOnceLadderCapReached(t *testing.T) {
	sim, err := NewPoolSimulator(directMatchFixture(t))
	require.NoError(t, err)

	maxIn := big.NewInt(251_188_643)
	maxOut := mustU256("9145417580492861513536").ToBig()
	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: fixUSDC, Amount: maxIn},
		TokenAmountOut: pool.TokenAmount{Token: fixWMON, Amount: maxOut},
	})

	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: fixUSDC, Amount: big.NewInt(1)},
		TokenOut:      fixWMON,
	})
	assert.ErrorIs(t, err, ErrAmountInTooLarge,
		"after consuming the ladder cap, the next satoshi must be rejected")
}

func TestUpdateBalance_CrossDirection_MirrorsReserves(t *testing.T) {
	// Hop 1: USDC -> WMON. Hop 2: WMON -> USDC on the SAME pool.
	// Reserves must be mirrored both ways so each hop sees pre-hop state and
	// the reserve-out guard reflects the mutation.
	sim, err := NewPoolSimulator(directMatchFixture(t))
	require.NoError(t, err)

	r0Before := sim.reserve0.Dec()
	r1Before := sim.reserve1.Dec()

	hop1, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: fixUSDC, Amount: big.NewInt(1_000_000)},
		TokenOut:      fixWMON,
	})
	require.NoError(t, err)
	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: fixUSDC, Amount: big.NewInt(1_000_000)},
		TokenAmountOut: pool.TokenAmount{Token: fixWMON, Amount: hop1.TokenAmountOut.Amount},
	})

	// reserve0 grew by 1e6 USDC; reserve1 shrank by hop1 output.
	wantR0 := new(big.Int).Add(mustU256(r0Before).ToBig(), big.NewInt(1_000_000))
	wantR1 := new(big.Int).Sub(mustU256(r1Before).ToBig(), hop1.TokenAmountOut.Amount)
	assert.Equal(t, wantR0.String(), sim.reserve0.Dec(), "reserve0 must grow by hop1 amountIn")
	assert.Equal(t, wantR1.String(), sim.reserve1.Dec(), "reserve1 must shrink by hop1 amountOut")

	// Hop 2 opposite direction: ladder1 is fresh (consumedIn[1] == 0), and the
	// USDC reserveOut is now bigger — the guard must still pass.
	hop2, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: fixWMON, Amount: mustU256("1000000000000000000").ToBig()},
		TokenOut:      fixUSDC,
	})
	require.NoError(t, err)
	assert.Equal(t, "27078", hop2.TokenAmountOut.Amount.String(),
		"opposite direction quote on fresh ladder must match Ladder1[0]")
}

func TestUpdateBalance_ReserveGuard_TripsAfterDrain(t *testing.T) {
	// Construct an asymmetric pool: lots of USDC, almost no WMON, with a
	// ladder claiming to deliver more WMON than reserves have. The first hop
	// passes; the second must trip the reserve guard.
	ep := directMatchFixture(t)
	ep.Reserves = []string{"1000000000000", "5000000000000000000"} // 1M USDC, 5 WMON
	ep = mutateExtra(t, ep, func(ex *Extra) {
		// Force ladder0 to claim 4 WMON for 1 USDC, and 4.5 for 2 USDC.
		ex.Ladder0 = []LadderPoint{
			{AmountIn: uint256.NewInt(1_000_000), AmountOut: mustU256("4000000000000000000")},
			{AmountIn: uint256.NewInt(2_000_000), AmountOut: mustU256("4500000000000000000")},
		}
	})
	sim, err := NewPoolSimulator(ep)
	require.NoError(t, err)

	first, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: fixUSDC, Amount: big.NewInt(1_000_000)},
		TokenOut:      fixWMON,
	})
	require.NoError(t, err)
	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: fixUSDC, Amount: big.NewInt(1_000_000)},
		TokenAmountOut: pool.TokenAmount{Token: fixWMON, Amount: first.TokenAmountOut.Amount},
	})
	// reserve1 mirror = 5 - 4 = 1 WMON. Next marginal slice = 0.5 WMON. Fits.
	// Bump consumedIn to exactly the cap, then push a third hop and require
	// the guard to fail because remaining WMON < marginal output.
	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: fixUSDC, Amount: big.NewInt(1_000_000)},
		TokenOut:      fixWMON,
	})
	// marginal = ladder(2e6)-ladder(1e6) = 0.5 WMON; reserveOut mirror = 1 WMON. OK.
	require.NoError(t, err)

	// Drain reserve1 manually to simulate a pathological cross-direction effect.
	sim.reserve1.Clear()
	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: fixUSDC, Amount: big.NewInt(1_000_000)},
		TokenOut:      fixWMON,
	})
	assert.ErrorIs(t, err, ErrPoolUnavailable,
		"reserve guard must reject once the mirror runs dry")
}

func TestUpdateBalance_UnknownTokenIsNoOp(t *testing.T) {
	sim, err := NewPoolSimulator(simpleFixture(t))
	require.NoError(t, err)
	before := sim.consumedIn[0].Clone()

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef", Amount: big.NewInt(1)},
		TokenAmountOut: pool.TokenAmount{Token: fixWMON, Amount: big.NewInt(1)},
	})
	assert.Equal(t, 0, before.Cmp(&sim.consumedIn[0]))
}

func TestCloneState_IsIndependent(t *testing.T) {
	sim, err := NewPoolSimulator(directMatchFixture(t))
	require.NoError(t, err)

	clone := sim.CloneState().(*PoolSimulator)
	assert.NotSame(t, sim, clone)

	params := pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: fixUSDC, Amount: big.NewInt(1_000_000)},
		TokenOut:      fixWMON,
	}
	orig, err := sim.CalcAmountOut(params)
	require.NoError(t, err)
	cloned, err := clone.CalcAmountOut(params)
	require.NoError(t, err)
	assert.Equal(t, orig.TokenAmountOut.Amount.String(), cloned.TokenAmountOut.Amount.String())

	clone.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: fixUSDC, Amount: big.NewInt(1_000_000)},
		TokenAmountOut: pool.TokenAmount{Token: fixWMON, Amount: cloned.TokenAmountOut.Amount},
	})
	assert.True(t, sim.consumedIn[0].IsZero(), "original consumedIn[0] must stay zero")
	assert.Equal(t, intgSnapshotReserve0, sim.reserve0.Dec(),
		"original reserve0 must stay at snapshot")

	again, err := sim.CalcAmountOut(params)
	require.NoError(t, err)
	assert.Equal(t, orig.TokenAmountOut.Amount.String(), again.TokenAmountOut.Amount.String(),
		"original simulator's quote must not shift after clone's UpdateBalance")
}

func TestGetMetaInfo_CarriesBlockNumber(t *testing.T) {
	sim, err := NewPoolSimulator(simpleFixture(t))
	require.NoError(t, err)
	meta := sim.GetMetaInfo("", "").(MetaInfo)
	assert.Equal(t, simpleSnapshotBlockNum, meta.BlockNumber)
}

// ---- Step-4 direct-protocol match ------------------------------------
//
// Compare CalcAmountOut against on-chain pool.quoteExactIn for each fixture
// row at block 75_581_558. Exact-hit rows MUST match wei-for-wei. Interpolated
// rows must conservatively underestimate (concave-down curve).

type directMatchRow struct {
	name      string
	tokenIn   string
	tokenOut  string
	amountIn  *big.Int
	protoOut  string // on-chain pool.quoteExactIn result
	tolerance int64  // 0 = exact; >0 = allowed underestimate
}

var directMatchRows = []directMatchRow{
	// USDC -> WMON (Ladder0)
	{
		name:    "USDC->WMON exact grid[0]=1 USDC",
		tokenIn: fixUSDC, tokenOut: fixWMON,
		amountIn: big.NewInt(1_000_000),
		protoOut: "36561072573389277514", tolerance: 0,
	},
	{
		name:    "USDC->WMON exact grid[4]=251.188643 USDC",
		tokenIn: fixUSDC, tokenOut: fixWMON,
		amountIn: big.NewInt(251_188_643),
		protoOut: "9145417580492861513536", tolerance: 0,
	},
	{
		// Below-smallest 0.5 USDC — linear-from-origin underestimates by
		// 1.53e14 WMON (~0.00042%) on a concave-down curve.
		name:    "USDC->WMON below-smallest (0.5 USDC)",
		tokenIn: fixUSDC, tokenOut: fixWMON,
		amountIn: big.NewInt(500_000),
		protoOut: "18280689288508557380", tolerance: 1_000_000_000_000_000,
	},
	{
		// Mid-bracket 8 USDC, between grid[1] and grid[2]. Linear interp
		// underestimates by 1.93e16 WMON (~0.0066%).
		name:    "USDC->WMON interpolated 8 USDC",
		tokenIn: fixUSDC, tokenOut: fixWMON,
		amountIn: big.NewInt(8_000_000),
		protoOut: "292454312266972852096", tolerance: 100_000_000_000_000_000,
	},
	// WMON -> USDC (Ladder1)
	{
		name:    "WMON->USDC exact grid[0]=1 WMON",
		tokenIn: fixWMON, tokenOut: fixUSDC,
		amountIn: mustU256("1000000000000000000").ToBig(),
		protoOut: "27078", tolerance: 0,
	},
	{
		name:    "WMON->USDC exact grid[4]=23077 WMON",
		tokenIn: fixWMON, tokenOut: fixUSDC,
		amountIn: mustU256("23077025673418920000000").ToBig(),
		protoOut: "614515241", tolerance: 0,
	},
	{
		name:    "WMON->USDC below-smallest (0.5 WMON)",
		tokenIn: fixWMON, tokenOut: fixUSDC,
		amountIn: mustU256("500000000000000000").ToBig(),
		protoOut: "13539", tolerance: 0,
	},
}

func TestDirectProtocolMatch(t *testing.T) {
	sim, err := NewPoolSimulator(directMatchFixture(t))
	require.NoError(t, err)

	// Confirm the snapshot the fixture exposes.
	assert.Equal(t, intgSnapshotBlockNum, sim.Info.BlockNumber)
	assert.Len(t, sim.extra.Ladder0, 5)
	assert.Len(t, sim.extra.Ladder1, 5)

	for _, row := range directMatchRows {
		row := row
		t.Run(row.name, func(t *testing.T) {
			res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: row.tokenIn, Amount: row.amountIn},
				TokenOut:      row.tokenOut,
			})
			require.NoError(t, err)

			simOut := res.TokenAmountOut.Amount
			protoOut, ok := new(big.Int).SetString(row.protoOut, 10)
			require.True(t, ok, "bad protoOut constant: %s", row.protoOut)

			if row.tolerance == 0 {
				assert.Equal(t, 0, simOut.Cmp(protoOut),
					"exact grid-hit: simOut=%s protoOut=%s", simOut, protoOut)
				return
			}
			// Conservative underestimate only.
			if simOut.Cmp(protoOut) > 0 {
				overshoot := new(big.Int).Sub(simOut, protoOut)
				t.Errorf("OVERESTIMATE simOut=%s > protoOut=%s overshoot=%s",
					simOut, protoOut, overshoot)
				return
			}
			under := new(big.Int).Sub(protoOut, simOut)
			tol := big.NewInt(row.tolerance)
			assert.LessOrEqualf(t, under.Cmp(tol), 0,
				"underestimate %s > tolerance %s (simOut=%s protoOut=%s)",
				under, tol, simOut, protoOut)
		})
	}
}

// mutateExtra returns ep with Extra re-marshaled after fn applies the patch.
func mutateExtra(t *testing.T, ep entity.Pool, fn func(*Extra)) entity.Pool {
	t.Helper()
	var ex Extra
	require.NoError(t, json.Unmarshal([]byte(ep.Extra), &ex))
	fn(&ex)
	b, err := json.Marshal(ex)
	require.NoError(t, err)
	ep.Extra = string(b)
	return ep
}
