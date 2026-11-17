package smardex

import (
	"math/big"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

var (
	TIMESTAMP_JAN_2020 int64 = 1577833200
	amountInT0               = parseString("1000000000000000000")
	resT0                    = parseString("13847262709278700000")
	resT1                    = parseString("119700592015995000000000")
	resFicT0                 = parseString("6441406027101710000")
	resFicT1                 = parseString("53094867866428500000000")
	priceAvT0                = parseString("1000000000000000000")
	priceAvT1                = parseString("8197837914161090000000")
	feesLP                   = uint256.NewInt(500)
	feesPool                 = uint256.NewInt(200)

	expectedResT0 = parseString("14847062709278699999")
	// the pair's reserve: balance less feeToAmount, one floor of the pool fee; the library's own is 1 wei lower
	expectedPairResT0   = parseString("14847062709278700000")
	expectedResT1       = parseString("112484184376480628646478")
	expectedResFicT0    = parseString("8094353523617659658")
	expectedResFicT1    = parseString("51232857537391979202756")
	expectedAmountOutT0 = parseString("7216407639514371353522")

	_ = func() bool { now = func() time.Time { return time.Unix(TIMESTAMP_JAN_2020, 0) }; return true }()
)

func parseString(value string) *uint256.Int {
	res := uint256.MustFromDecimal(value)
	return res
}

func TestCalcAmountOut(t *testing.T) {
	t.Parallel()
	extra := SmardexPair{
		PairFee: PairFee{
			FeesLP:   feesLP,
			FeesPool: feesPool,
			FeesBase: uint256.MustFromBig(FEES_BASE),
		},
		FictiveReserve: FictiveReserve{
			FictiveReserve0: resFicT0,
			FictiveReserve1: resFicT1,
		},
		PriceAverage: PriceAverage{
			PriceAverage0:             priceAvT0,
			PriceAverage1:             priceAvT1,
			PriceAverageLastTimestamp: uint256.NewInt(uint64(TIMESTAMP_JAN_2020)),
		},
		FeeToAmount: FeeToAmount{
			Fees0: uint256.NewInt(0),
			Fees1: uint256.NewInt(0),
		},
	}
	extraJson, _ := json.Marshal(extra)

	token0 := entity.PoolToken{
		Address:   "token0",
		Swappable: true,
	}
	token1 := entity.PoolToken{
		Address:   "token1",
		Swappable: true,
	}

	pool := entity.Pool{
		Reserves: entity.PoolReserves{resT0.String(), resT1.String()},
		Tokens:   []*entity.PoolToken{&token0, &token1},
		Extra:    string(extraJson),
	}
	poolSimulator, _ := NewPoolSimulator(pool)
	result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
		return poolSimulator.CalcAmountOut(
			poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Token:  "token0",
					Amount: amountInT0.ToBig(),
				},
				TokenOut: "token1",
			})
	})

	if err != nil {
		t.Fatalf(`Error thrown %v`, err)
	}
	if result.TokenAmountOut.Amount.Cmp(expectedAmountOutT0.ToBig()) != 0 {
		t.Fatalf(`Invalid value = %d, expected: %d`, result.TokenAmountOut.Amount, expectedAmountOutT0)
	}

	newState, ok := result.SwapInfo.(SwapInfo)
	if !ok {
		t.Fatal(`Swapinfo is nil`)
	}
	if newState.newReserve0.Cmp(expectedPairResT0) != 0 {
		t.Fatalf(`Invalid value = %d, expected: %d`, newState.newReserve0, expectedPairResT0)
	}
	if newState.newReserve1.Cmp(expectedResT1) != 0 {
		t.Fatalf(`Invalid value = %d, expected: %d`, newState.newReserve1, expectedResT1)
	}
	if newState.newFictiveReserve0.Cmp(expectedResFicT0) != 0 {
		t.Fatalf(`Invalid value = %d, expected: %d`, newState.newFictiveReserve0, expectedResFicT0)
	}
	if newState.newFictiveReserve1.Cmp(expectedResFicT1) != 0 {
		t.Fatalf(`Invalid value = %d, expected: %d`, newState.newFictiveReserve1, expectedResFicT1)
	}
}

func TestGetAmountOut(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name                string
		amountParams        GetAmountParameters
		expectedAmountOutT0 *uint256.Int
		expectedReserve0    *uint256.Int
		expectedReserve1    *uint256.Int
		expectedResFictive0 *uint256.Int
		expectedResFictive1 *uint256.Int
	}{
		{
			name: "Test case 1",
			amountParams: GetAmountParameters{
				amount:            amountInT0,
				reserveIn:         resT0,
				reserveOut:        resT1,
				fictiveReserveIn:  resFicT0,
				fictiveReserveOut: resFicT1,
				priceAverageIn:    priceAvT0,
				priceAverageOut:   priceAvT1,
				feesLP:            feesLP,
				feesPool:          feesPool,
				feesBase:          uint256.MustFromBig(FEES_BASE)},
			expectedAmountOutT0: expectedAmountOutT0,
			expectedReserve0:    expectedResT0,
			expectedReserve1:    expectedResT1,
			expectedResFictive0: expectedResFicT0,
			expectedResFictive1: expectedResFicT1,
		},
		{
			name: "Test case 2",
			amountParams: GetAmountParameters{
				amount:            uint256.NewInt(42),
				reserveIn:         parseString("161897635415"),
				reserveOut:        parseString("15369827327148701303864657"),
				fictiveReserveIn:  parseString("76745457210"),
				fictiveReserveOut: parseString("6535835031490019911286921"),
				priceAverageIn:    parseString("76745457210"),
				priceAverageOut:   parseString("6535835031490019911286921"),
				feesLP:            uint256.NewInt(1500),
				feesPool:          uint256.NewInt(900),
				feesBase:          uint256.MustFromBig(FEES_BASE)},
			expectedAmountOutT0: parseString("3483282525323441"),
			expectedReserve0:    parseString("161897635455"),
			expectedReserve1:    parseString("15369827323665418778541216"),
			expectedResFictive0: parseString("85593526029"),
			expectedResFictive1: parseString("7289358689105108450240064"),
		},
		{
			name: "Test case 3",
			amountParams: GetAmountParameters{
				amount:            uint256.NewInt(2000000000000000000),
				reserveIn:         parseString("3278796445628485066"),
				reserveOut:        parseString("6213633437"),
				fictiveReserveIn:  parseString("1602466039436492633"),
				fictiveReserveOut: parseString("3179127537"),
				priceAverageIn:    parseString("1602466039436492633"),
				priceAverageOut:   parseString("3179127537"),
				feesLP:            uint256.MustFromBig(FEES_LP_DEFAULT_ETHEREUM),
				feesPool:          uint256.MustFromBig(FEES_POOL_DEFAULT_ETHEREUM),
				feesBase:          uint256.MustFromBig(FEES_BASE_ETHEREUM)},
			expectedAmountOutT0: parseString("1719846589"),
			expectedReserve0:    parseString("5278396445628485066"),
			expectedReserve1:    parseString("4493786848"),
			expectedResFictive0: parseString("3530568934927624489"),
			expectedResFictive1: parseString("1317438058"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name,
			func(t *testing.T) {
				result, err := testutil.MustConcurrentSafe(t, func() (*GetAmountResult, error) {
					return getAmountOut(tc.amountParams)
				})
				if err != nil {
					t.Fatalf(`Error thrown %v`, err)
				}
				if result.amountOut.Cmp(tc.expectedAmountOutT0) != 0 {
					t.Fatalf(`Invalid value = %d, expected: %d`, result.amountOut, tc.expectedAmountOutT0)
				}
				if result.newReserveIn.Cmp(tc.expectedReserve0) != 0 {
					t.Fatalf(`Invalid value = %d, expected: %d`, result.newReserveIn, tc.expectedReserve0)
				}
				if result.newReserveOut.Cmp(tc.expectedReserve1) != 0 {
					t.Fatalf(`Invalid value = %d, expected: %d`, result.newReserveOut, tc.expectedReserve1)
				}
				if result.newFictiveReserveIn.Cmp(tc.expectedResFictive0) != 0 {
					t.Fatalf(`Invalid value = %d, expected: %d`, result.newFictiveReserveIn, tc.expectedResFictive0)
				}
				if result.newFictiveReserveOut.Cmp(tc.expectedResFictive1) != 0 {
					t.Fatalf(`Invalid value = %d, expected: %d`, result.newFictiveReserveOut, tc.expectedResFictive1)
				}
			})
	}

}

func TestComputeFictiveReservesTrueOeGT1(t *testing.T) {
	t.Parallel()
	resT0 := parseString("13873434733749100000")
	resT1 := parseString("119492838392173000000000")
	resFicT0 := parseString("7120725548088060000")
	resFicT1 := parseString("58241511553084200000000")
	expectedResFicT0 := parseString("6761986430618317504")
	expectedResFicT1 := parseString("55307329030031163856016")

	newResFicIn, newResFicOut := computeFictiveReserves(
		resT1,
		resT0,
		resFicT1,
		resFicT0,
	)

	if newResFicIn.Cmp(expectedResFicT1) != 0 {
		t.Fatalf(`Invalid value = %d, expected: %d`, newResFicIn, expectedResFicT1)
	}
	if newResFicOut.Cmp(expectedResFicT0) != 0 {
		t.Fatalf(`Invalid value = %d, expected: %d`, newResFicOut, expectedResFicT0)
	}
}

func TestComputeFictiveReservesEthInTrueOeLT1(t *testing.T) {
	t.Parallel()
	resT0 := parseString("13864885801349700000")
	resT1 := parseString("119555797951391000000000")
	resFicT0 := parseString("6459029119172690000")
	resFicT1 := parseString("52950073801824400000000")
	expectedResFicT0 := parseString("7112176615688650553")
	expectedResFicT1 := parseString("58304471112302341135376")

	newResFicIn, newResFicOut := computeFictiveReserves(
		resT0,
		resT1,
		resFicT0,
		resFicT1,
	)

	if newResFicIn.Cmp(expectedResFicT0) != 0 {
		t.Fatalf(`Invalid value = %d, expected: %d`, newResFicIn, expectedResFicT0)
	}
	if newResFicOut.Cmp(expectedResFicT1) != 0 {
		t.Fatalf(`Invalid value = %d, expected: %d`, newResFicOut, expectedResFicT1)
	}
}

func TestComputeFictiveReservesEthInTrueOeGT1(t *testing.T) {
	t.Parallel()
	// ETH_in, oe > 1, line 23
	resT0 := parseString("12668420462955600000")
	resT1 := parseString("103877534648498000000000")
	resFicT0 := parseString("6332837569656430000")
	resFicT1 := parseString("51951123826036400000000")
	expectedResFicT0 := parseString("6329892508211233858")
	expectedResFicT1 := parseString("51926964158252125695036")

	newResFicIn, newResFicOut := computeFictiveReserves(
		resT0,
		resT1,
		resFicT0,
		resFicT1,
	)

	if newResFicIn.Cmp(expectedResFicT0) != 0 {
		t.Fatalf(`Invalid value = %d, expected: %d`, newResFicIn, expectedResFicT0)
	}
	if newResFicOut.Cmp(expectedResFicT1) != 0 {
		t.Fatalf(`Invalid value = %d, expected: %d`, newResFicOut, expectedResFicT1)
	}
}

func TestUpdateBalance(t *testing.T) {
	t.Parallel()
	extra := SmardexPair{
		PairFee: PairFee{
			FeesLP:   feesLP,
			FeesPool: feesPool,
			FeesBase: uint256.MustFromBig(FEES_BASE),
		},
		FictiveReserve: FictiveReserve{
			FictiveReserve0: resFicT0,
			FictiveReserve1: resFicT1,
		},
		PriceAverage: PriceAverage{
			PriceAverage0:             priceAvT0,
			PriceAverage1:             priceAvT1,
			PriceAverageLastTimestamp: uint256.NewInt(uint64(TIMESTAMP_JAN_2020)),
		},
		FeeToAmount: FeeToAmount{
			Fees0: uint256.NewInt(0),
			Fees1: uint256.NewInt(0),
		},
	}
	extraJson, _ := json.Marshal(extra)

	token0 := entity.PoolToken{
		Address:   "token0",
		Swappable: true,
	}
	token1 := entity.PoolToken{
		Address:   "token1",
		Swappable: true,
	}

	pool := entity.Pool{
		Reserves: entity.PoolReserves{resT0.String(), resT1.String()},
		Tokens:   []*entity.PoolToken{&token0, &token1},
		Extra:    string(extraJson),
	}
	poolSimulator, _ := NewPoolSimulator(pool)
	tokenAmountIn := poolpkg.TokenAmount{
		Token:  "token0",
		Amount: amountInT0.ToBig(),
	}
	result, _ := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
		return poolSimulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: tokenAmountIn,
			TokenOut:      "token1",
			Limit:         nil,
		})
	})

	poolSimulator.UpdateBalance(poolpkg.UpdateBalanceParams{
		TokenAmountIn:  tokenAmountIn,
		TokenAmountOut: *result.TokenAmountOut,
		Fee:            *result.Fee,
		SwapInfo:       result.SwapInfo,
	})
	assert.Equal(t, poolSimulator.FictiveReserve.FictiveReserve0.Cmp(expectedResFicT0), 0)
	assert.Equal(t, poolSimulator.FictiveReserve.FictiveReserve1.Cmp(expectedResFicT1), 0)
	// getReserves leaves feeToAmount out already: the input less one pool fee in, getAmountOut's output out
	assert.Equal(t, poolSimulator.Info.Reserves[0].Cmp(expectedPairResT0.ToBig()), 0)
	assert.Equal(t, poolSimulator.Info.Reserves[1].Cmp(expectedResT1.ToBig()), 0)
	assert.Equal(t, poolSimulator.PriceAverage.PriceAverage0.Cmp(priceAvT0), 0)
	assert.Equal(t, poolSimulator.PriceAverage.PriceAverage1.Cmp(priceAvT1), 0)
	assert.Equal(t, poolSimulator.PriceAverage.PriceAverageLastTimestamp.Cmp(uint256.NewInt(uint64(TIMESTAMP_JAN_2020))), 0)
}

// The WETH/USDT pair at block 26329765 warped to backTime, 2,338 s after its last trade, and what the pair did on a
// fork of that block (forge, storage read after each swap): 0.2 WETH in, then 500 USDT back at the same timestamp.
// The swap back crosses the price average, so the pair runs it as two legs.
const (
	backTime  = 1793656605
	backPair  = "0x9fcf8f5bd54db123470c96620441ca5c342a8bd4"
	backExtra = `{"PairFee":{"FeesLP":"5","FeesPool":"2","FeesBase":"10000"},` +
		`"FictiveReserve":{"FictiveReserve0":"1373596388601278951","FictiveReserve1":"3094586360"},` +
		`"PriceAverage":{"PriceAverage0":"1375769401236588899","PriceAverage1":"3089511082","PriceAverageLastTimestamp":"1793654267"},` +
		`"FeeToAmount":{"Fees0":"3708471625987616","Fees1":"8999569"}}`
	afterAExtra = `{"PairFee":{"FeesLP":"5","FeesPool":"2","FeesBase":"10000"},` +
		`"FictiveReserve":{"FictiveReserve0":"1607392492532892891","FictiveReserve1":"2776538479"},` +
		`"PriceAverage":{"PriceAverage0":"1373596388601278951","PriceAverage1":"3094586360","PriceAverageLastTimestamp":"1793656605"},` +
		`"FeeToAmount":{"Fees0":"3748471625987616","Fees1":"8999569"}}`
)

func backSimulator(t *testing.T, reserve0, reserve1, extra string) *PoolSimulator {
	t.Helper()
	s, err := NewPoolSimulator(entity.Pool{
		Address: backPair, Exchange: DexTypeSmardex, Type: DexTypeSmardex,
		Reserves: entity.PoolReserves{reserve0, reserve1},
		Tokens:   []*entity.PoolToken{{Address: liveWETH, Decimals: 18}, {Address: liveUSDT, Decimals: 6}},
		Extra:    extra,
	})
	require.NoError(t, err)
	return s
}

func atTime(t *testing.T, unix int64) {
	t.Helper()
	prev := now
	now = func() time.Time { return time.Unix(unix, 0) }
	t.Cleanup(func() { now = prev })
}

// The pair runs the second leg whenever the first one does not take the whole input (SmardexLibrary.getAmountOut).
func TestBackSwapRunsBothLegs(t *testing.T) {
	atTime(t, backTime)
	s := backSimulator(t, "2782911624945216931", "6413619964", backExtra)
	state := func() []string {
		return []string{s.Info.Reserves[0].String(),
			s.FictiveReserve.FictiveReserve0.String(), s.FictiveReserve.FictiveReserve1.String(),
			s.PriceAverage.PriceAverage0.String(), s.PriceAverage.PriceAverage1.String(),
			s.PriceAverage.PriceAverageLastTimestamp.String(), s.FeeToAmount.Fees0.String(), s.FeeToAmount.Fees1.String()}
	}

	require.Equal(t, "394277511", liveSwap(t, s, liveWETH, liveUSDT, "200000000000000000"))
	require.Equal(t, "6019342453", s.Info.Reserves[1].String())
	require.Equal(t, []string{"2982871624945216931", "1607392492532892891", "2776538479",
		"1373596388601278951", "3094586360", "1793656605", "3748471625987616", "8999569"}, state())

	require.Equal(t, "245106512054878313", liveSwap(t, s, liveUSDT, liveWETH, "500000000"))
	require.Equal(t, []string{"2737765112890338618", "1330647478097017393", "3205159841",
		"1373596388601278951", "3094586360", "1793656605", "3748471625987616", "9099569"}, state())
}

// The pair's reserves are its balances less feeToAmount, so the input side of a swap moves by the input less
// one floor of the pool fee: reserve + feeToAmount grows by the whole input, for any input and any number of legs.
func TestReserveInPlusFeeTakesTheWholeInput(t *testing.T) {
	atTime(t, backTime)

	t.Run("pair's value after the two legs", func(t *testing.T) {
		s := backSimulator(t, "2982871624945216931", "6019342453", afterAExtra)
		liveSwap(t, s, liveUSDT, liveWETH, "500000000")
		require.Equal(t, "6519242453", s.Info.Reserves[1].String())
	})

	for _, tc := range []struct {
		name, in, out string
		i             int
		amounts       []string
	}{
		{"usdt in", liveUSDT, liveWETH, 1, []string{"1", "4999", "12345", "12345678", "500000000", "500000001"}},
		{"weth in", liveWETH, liveUSDT, 0, []string{"4999", "12345", "200000000000000001", "1234567890123456789"}},
	} {
		for _, amount := range tc.amounts {
			t.Run(tc.name+" "+amount, func(t *testing.T) {
				s := backSimulator(t, "2982871624945216931", "6019342453", afterAExtra)
				before := new(big.Int).Set(s.Info.Reserves[tc.i])
				fee := func() *big.Int {
					if tc.i == 0 {
						return s.FeeToAmount.Fees0.ToBig()
					}
					return s.FeeToAmount.Fees1.ToBig()
				}
				feeBefore := fee()
				liveSwap(t, s, tc.in, tc.out, amount)
				in, ok := new(big.Int).SetString(amount, 10)
				require.True(t, ok)
				got := new(big.Int).Sub(s.Info.Reserves[tc.i], before)
				got.Add(got, new(big.Int).Sub(fee(), feeBefore))
				require.Equal(t, in.String(), got.String())
			})
		}
	}
}

// SmardexLibrary.getUpdatedPriceAverage in the pair's verified source. Inside its 300 s window the new price
// average is the time-weighted mean, ((300-dt) * paOut * fIn / paIn + dt * fOut) / 300, with dt = min(elapsed, 300);
// at the pair's own timestamp it stays as stored.
func TestUpdatedPriceAverageMatchesThePairsFormula(t *testing.T) {
	fIn, fOut := big.NewInt(1373596388601278951), big.NewInt(3094586360)
	paIn, paOut := big.NewInt(1375769401236588899), big.NewInt(3089511082)
	const last = 1793654267

	solidity := func(elapsed int64) (in, out *big.Int) {
		if elapsed == 0 {
			return paIn, paOut
		}
		dt := min(elapsed, 300)
		out = new(big.Int).Mul(big.NewInt(300-dt), paOut)
		out.Mul(out, fIn).Quo(out, paIn)
		out.Add(out, new(big.Int).Mul(big.NewInt(dt), fOut))
		return fIn, out.Quo(out, big.NewInt(300))
	}

	for _, elapsed := range []int64{0, 1, 100, 299, 300, 2338} {
		wantIn, wantOut := solidity(elapsed)
		in, out, err := getUpdatedPriceAverage(uint256.MustFromBig(fIn), uint256.MustFromBig(fOut),
			uint256.NewInt(last), uint256.MustFromBig(paIn), uint256.MustFromBig(paOut),
			uint256.NewInt(uint64(last+elapsed)))
		require.NoError(t, err)
		require.Equal(t, wantIn.String(), in.String(), "elapsed %d", elapsed)
		require.Equal(t, wantOut.String(), out.String(), "elapsed %d", elapsed)
	}
}

// The same fork warped to withinTime, 100 s after the pair's last trade, so swap A updates the price average inside
// its window. The values are forge reads of the pair on a mainnet fork.
const withinTime = 1793654367

func pairState(s *PoolSimulator) []string {
	return []string{s.Info.Reserves[0].String(), s.Info.Reserves[1].String(),
		s.FictiveReserve.FictiveReserve0.String(), s.FictiveReserve.FictiveReserve1.String(),
		s.PriceAverage.PriceAverage0.String(), s.PriceAverage.PriceAverage1.String(),
		s.PriceAverage.PriceAverageLastTimestamp.String(), s.FeeToAmount.Fees0.String(), s.FeeToAmount.Fees1.String()}
}

// The price average fix on the chain's numbers: the quote and the price average the swap stores. Reserve0 follows the input reserve fix, checked below.
func TestSwapWithinTheWindowMatchesThePair(t *testing.T) {
	atTime(t, withinTime)
	s := backSimulator(t, "2782911624945216931", "6413619964", backExtra)
	require.Equal(t, "394258918", liveSwap(t, s, liveWETH, liveUSDT, "200000000000000000"))
	require.Equal(t, []string{"6019361046", "1607446233976928452", "2776514403",
		"1373596388601278951", "3087949608", "1793654367", "3748471625987616", "8999569"}, pairState(s)[1:])
}

// All three fixes together: both swaps, wei for wei, the swap back running two legs against that stored price average.
func TestBackSwapWithinTheWindowMatchesThePair(t *testing.T) {
	atTime(t, withinTime)
	s := backSimulator(t, "2782911624945216931", "6413619964", backExtra)
	liveSwap(t, s, liveWETH, liveUSDT, "200000000000000000")
	require.Equal(t, []string{"2982871624945216931", "6019361046", "1607446233976928452", "2776514403",
		"1373596388601278951", "3087949608", "1793654367", "3748471625987616", "8999569"}, pairState(s))
	require.Equal(t, "245112510658742292", liveSwap(t, s, liveUSDT, liveWETH, "500000000"))
	require.Equal(t, []string{"2737759114286474639", "6519261046", "1329148118412164447", "3201810627",
		"1373596388601278951", "3087949608", "1793654367", "3748471625987616", "9099569"}, pairState(s))
}

// The WETH/USDT pair 0x9fcf8f5bd54db123470c96620441ca5c342a8bd4 on Ethereum at block 26320030 (timestamp
// 1793537411), and what the pair did on a fork of that block: one swap, then a second in the same direction
// or a swap back. Its feeToAmount (3698290308571691 WETH) is larger than its USDT reserve.
const (
	liveWETH  = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
	liveUSDT  = "0xdac17f958d2ee523a2206206994597c13d831ec7"
	liveTime  = 1793537411
	liveExtra = `{"PairFee":{"FeesLP":"5","FeesPool":"2","FeesBase":"10000"},` +
		`"FictiveReserve":{"FictiveReserve0":"1373845172096677391","FictiveReserve1":"3106315442"},` +
		`"PriceAverage":{"PriceAverage0":"1374834110173688166","PriceAverage1":"3104325840","PriceAverageLastTimestamp":"1793536943"},` +
		`"FeeToAmount":{"Fees0":"3698290308571691","Fees1":"8977894"}}`
)

func newLiveSimulator(t *testing.T) *PoolSimulator {
	t.Helper()
	s, err := NewPoolSimulator(entity.Pool{
		Address: "0x9fcf8f5bd54db123470c96620441ca5c342a8bd4", Exchange: DexTypeSmardex, Type: DexTypeSmardex,
		Reserves: entity.PoolReserves{"2779542133377466939", "6421126753"},
		Tokens:   []*entity.PoolToken{{Address: liveWETH, Decimals: 18}, {Address: liveUSDT, Decimals: 6}},
		Extra:    liveExtra,
	})
	require.NoError(t, err)
	return s
}

func liveSwap(t *testing.T, s *PoolSimulator, in, out, amount string) string {
	t.Helper()
	a, _ := new(big.Int).SetString(amount, 10)
	r, err := s.CalcAmountOut(poolpkg.CalcAmountOutParams{TokenAmountIn: poolpkg.TokenAmount{Token: in, Amount: a}, TokenOut: out})
	require.NoError(t, err)
	s.UpdateBalance(poolpkg.UpdateBalanceParams{TokenAmountIn: poolpkg.TokenAmount{Token: in, Amount: a},
		TokenAmountOut: *r.TokenAmountOut, Fee: *r.Fee, SwapInfo: r.SwapInfo})
	return r.TokenAmountOut.Amount.String()
}

// After a swap the simulator holds the pair's reserves in token order (balances less feeToAmount, as
// getReserves returns them), so the next swap, in either direction, returns what the pair returned.
func TestUpdateBalanceMatchesThePair(t *testing.T) {
	defer func(f func() time.Time) { now = f }(now)
	now = func() time.Time { return time.Unix(liveTime, 0) }

	for _, tc := range []struct {
		name, in, out, a, outA, res0, res1, b, outB, back, outBack string
	}{
		{"weth in", liveWETH, liveUSDT, "10000000000000000", "22434964", "2789540133377466939", "6398691789",
			"50000000000000000", "107582912", "100000000", "43433582686221945"},
		{"usdt in", liveUSDT, liveWETH, "20000000", "8782843803129020", "2770759289574337919", "6441122753",
			"100000000", "42283159516666707", "50000000000000000", "110448642"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newLiveSimulator(t)
			require.Equal(t, tc.outA, liveSwap(t, s, tc.in, tc.out, tc.a))
			require.Equal(t, tc.res0, s.GetReserves()[0].String(), "token0's reserve")
			require.Equal(t, tc.res1, s.GetReserves()[1].String(), "token1's reserve")
			require.Equal(t, tc.outB, liveSwap(t, s, tc.in, tc.out, tc.b), "a second swap in the same direction")

			s = newLiveSimulator(t)
			liveSwap(t, s, tc.in, tc.out, tc.a)
			require.Equal(t, tc.outBack, liveSwap(t, s, tc.out, tc.in, tc.back), "a swap back")
		})
	}
}
