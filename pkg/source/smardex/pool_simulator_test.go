package smardex

import (
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCalcAmountOut(t *testing.T) {
	extra := SmardexPair{
		PairFee: PairFee{
			FeesLP:   feesLP,
			FeesPool: feesPool,
			FeesBase: FEES_BASE,
		},
		FictiveReserve: FictiveReserve{
			FictiveReserve0: resFicT0,
			FictiveReserve1: resFicT1,
		},
		PriceAverage: PriceAverage{
			PriceAverage0:             priceAvT0,
			PriceAverage1:             priceAvT1,
			PriceAverageLastTimestamp: big.NewInt(TIMESTAMP_JAN_2020),
		},
		FeeToAmount: FeeToAmount{
			Fees0: big.NewInt(0),
			Fees1: big.NewInt(0),
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
	now = func() time.Time {
		return time.Unix(TIMESTAMP_JAN_2020, 0)
	}
	result, err := testutil.MustConcurrentSafe[*poolpkg.CalcAmountOutResult](t, func() (any, error) {
		return poolSimulator.CalcAmountOut(
			poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Token:  "token0",
					Amount: amountInT0,
				},
				TokenOut: "token1",
				Limit:    nil,
			})
	})

	if err != nil {
		t.Fatalf(`Error thrown %v`, err)
	}
	if result.TokenAmountOut.Amount.Cmp(expectedAmountOutT0) != 0 {
		t.Fatalf(`Invalid value = %d, expected: %d`, result.TokenAmountOut.Amount, expectedAmountOutT0)
	}

	newState, ok := result.SwapInfo.(SwapInfo)
	if !ok {
		t.Fatal(`Swapinfo is nil`)
	}
	if newState.newReserveIn.Cmp(new(big.Int).Sub(expectedResT0, newState.feeToAmount0)) != 0 {
		t.Fatalf(`Invalid value = %d, expected: %d`, newState.newReserveIn, expectedResT0)
	}
	if newState.newReserveOut.Cmp(expectedResT1) != 0 {
		t.Fatalf(`Invalid value = %d, expected: %d`, newState.newReserveOut, expectedResT1)
	}
	if newState.newFictiveReserveIn.Cmp(expectedResFicT0) != 0 {
		t.Fatalf(`Invalid value = %d, expected: %d`, newState.newFictiveReserveIn, expectedResFicT0)
	}
	if newState.newFictiveReserveOut.Cmp(expectedResFicT1) != 0 {
		t.Fatalf(`Invalid value = %d, expected: %d`, newState.newFictiveReserveOut, expectedResFicT1)
	}
}

func TestGetAmountOut(t *testing.T) {
	testCases := []struct {
		name                string
		amountParams        GetAmountParameters
		expectedAmountOutT0 *big.Int
		expectedReserve0    *big.Int
		expectedReserve1    *big.Int
		expectedResFictive0 *big.Int
		expectedResFictive1 *big.Int
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
				feesBase:          FEES_BASE},
			expectedAmountOutT0: expectedAmountOutT0,
			expectedReserve0:    expectedResT0,
			expectedReserve1:    expectedResT1,
			expectedResFictive0: expectedResFicT0,
			expectedResFictive1: expectedResFicT1,
		},
		{
			name: "Test case 2",
			amountParams: GetAmountParameters{
				amount:            big.NewInt(42),
				reserveIn:         parseString("161897635415"),
				reserveOut:        parseString("15369827327148701303864657"),
				fictiveReserveIn:  parseString("76745457210"),
				fictiveReserveOut: parseString("6535835031490019911286921"),
				priceAverageIn:    parseString("76745457210"),
				priceAverageOut:   parseString("6535835031490019911286921"),
				feesLP:            big.NewInt(1500),
				feesPool:          big.NewInt(900),
				feesBase:          FEES_BASE},
			expectedAmountOutT0: parseString("3483282525323441"),
			expectedReserve0:    parseString("161897635455"),
			expectedReserve1:    parseString("15369827323665418778541216"),
			expectedResFictive0: parseString("85593526029"),
			expectedResFictive1: parseString("7289358689105108450240064"),
		},
		{
			name: "Test case 3",
			amountParams: GetAmountParameters{
				amount:            big.NewInt(2000000000000000000),
				reserveIn:         parseString("3278796445628485066"),
				reserveOut:        parseString("6213633437"),
				fictiveReserveIn:  parseString("1602466039436492633"),
				fictiveReserveOut: parseString("3179127537"),
				priceAverageIn:    parseString("1602466039436492633"),
				priceAverageOut:   parseString("3179127537"),
				feesLP:            FEES_LP_DEFAULT_ETHEREUM,
				feesPool:          FEES_POOL_DEFAULT_ETHEREUM,
				feesBase:          FEES_BASE_ETHEREUM},
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
				result, err := testutil.MustConcurrentSafe[GetAmountResult](t, func() (any, error) {
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
	extra := SmardexPair{
		PairFee: PairFee{
			FeesLP:   feesLP,
			FeesPool: feesPool,
			FeesBase: FEES_BASE,
		},
		FictiveReserve: FictiveReserve{
			FictiveReserve0: resFicT0,
			FictiveReserve1: resFicT1,
		},
		PriceAverage: PriceAverage{
			PriceAverage0:             priceAvT0,
			PriceAverage1:             priceAvT1,
			PriceAverageLastTimestamp: big.NewInt(TIMESTAMP_JAN_2020),
		},
		FeeToAmount: FeeToAmount{
			Fees0: big.NewInt(0),
			Fees1: big.NewInt(0),
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
		Amount: amountInT0,
	}
	now = func() time.Time {
		return time.Unix(TIMESTAMP_JAN_2020, 0)
	}
	result, _ := testutil.MustConcurrentSafe[*poolpkg.CalcAmountOutResult](t, func() (any, error) {
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
	assert.Equal(t, poolSimulator.Info.Reserves[0].Cmp(new(big.Int).Sub(expectedResT0, poolSimulator.FeeToAmount.Fees0)), 0)
	assert.Equal(t, poolSimulator.Info.Reserves[1].Cmp(new(big.Int).Sub(expectedResT1, poolSimulator.FeeToAmount.Fees1)), 0)
	assert.Equal(t, poolSimulator.PriceAverage.PriceAverage0.Cmp(priceAvT0), 0)
	assert.Equal(t, poolSimulator.PriceAverage.PriceAverage1.Cmp(priceAvT1), 0)
	assert.Equal(t, poolSimulator.PriceAverage.PriceAverageLastTimestamp.Cmp(big.NewInt(TIMESTAMP_JAN_2020)), 0)
}
