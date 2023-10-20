package sdk

import (
	"math/big"
	"testing"
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
	feesLP                   = big.NewInt(500)
	feesPool                 = big.NewInt(200)

	expectedResT0       = parseString("14847062709278699999")
	expectedResT1       = parseString("112484184376480628646478")
	expectedResFicT0    = parseString("8094353523617659658")
	expectedResFicT1    = parseString("51232857537391979202756")
	expectedAmountOutT0 = parseString("7216407639514371353522")
)

func TestComputeAmountOut(t *testing.T) {

	computed, err := ComputeAmountOut(
		"token0",
		"token1",
		resT0,
		resT1,
		resFicT0,
		resFicT1,
		amountInT0,
		"token0",
		TIMESTAMP_JAN_2020,
		priceAvT0,
		priceAvT1,
		feesLP,
		feesPool,
		TIMESTAMP_JAN_2020,
		300,
	)
	if err != nil {
		t.Fatalf(`Error thrown %v`, err)
	}

	if computed.currency == "token0" {
		t.Fatalf(`Invalid value = %s, expected: %s`, "token0", computed.currency)
	}
	if computed.amount.Cmp(expectedAmountOutT0) != 0 {
		t.Fatalf(`Invalid value = %d, expected: %d`, computed.amount, expectedAmountOutT0)
	}
	if computed.newRes0.Cmp(expectedResT0) != 0 {
		t.Fatalf(`Invalid value = %d, expected: %d`, computed.newRes0, expectedResT0)
	}
	if computed.newRes1.Cmp(expectedResT1) != 0 {
		t.Fatalf(`Invalid value = %d, expected: %d`, computed.newRes1, expectedResT1)
	}
	if computed.newRes0Fic.Cmp(expectedResFicT0) != 0 {
		t.Fatalf(`Invalid value = %d, expected: %d`, computed.newRes0Fic, expectedResFicT0)
	}
	if computed.newRes1Fic.Cmp(expectedResFicT1) != 0 {
		t.Fatalf(`Invalid value = %d, expected: %d`, computed.newRes1Fic, expectedResFicT1)
	}
	if computed.amountMax.Cmp(computed.amount) == -1 {
		t.Fatalf(`Invalid value = %d, expected: > %d`, computed.amountMax, computed.amount)
	}
	if computed.amountMax.Cmp(resFicT0) != 1 {
		t.Fatalf(`Invalid value = %d, expected: < %d`, computed.amountMax, resFicT0)
	}
}
func TestGetAmountOut(t *testing.T) {

	amountOut, newResIn, newResOut, newResInFic, newResOutFic, err := getAmountOut(
		amountInT0,
		resT0,
		resT1,
		resFicT0,
		resFicT1,
		priceAvT0,
		priceAvT1,
		feesLP,
		feesPool,
	)
	if err != nil {
		t.Fatalf(`Error thrown %v`, err)
	}
	if amountOut.Cmp(expectedAmountOutT0) != 0 {
		t.Fatalf(`Invalid value = %d, expected: %d`, amountOut, expectedAmountOutT0)
	}
	if newResIn.Cmp(expectedResT0) != 0 {
		t.Fatalf(`Invalid value = %d, expected: %d`, newResIn, expectedAmountOutT0)
	}
	if newResOut.Cmp(expectedResT1) != 0 {
		t.Fatalf(`Invalid value = %d, expected: %d`, newResOut, expectedAmountOutT0)
	}
	if newResInFic.Cmp(expectedResFicT0) != 0 {
		t.Fatalf(`Invalid value = %d, expected: %d`, newResInFic, expectedAmountOutT0)
	}
	if newResOutFic.Cmp(expectedResFicT1) != 0 {
		t.Fatalf(`Invalid value = %d, expected: %d`, newResOutFic, expectedAmountOutT0)
	}
}

func TestComputeReserveFicEthOutTrueOeGT1(t *testing.T) {
	resT0 := parseString("13873434733749100000")
	resT1 := parseString("119492838392173000000000")
	resFicT0 := parseString("7120725548088060000")
	resFicT1 := parseString("58241511553084200000000")
	expectedResFicT0 := parseString("6761986430618317504")
	expectedResFicT1 := parseString("55307329030031163856016")

	newResFicIn, newResFicOut := computeReserveFic(
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
func TestComputeReserveFicEthInTrueOeLT1(t *testing.T) {
	resT0 := parseString("13864885801349700000")
	resT1 := parseString("119555797951391000000000")
	resFicT0 := parseString("6459029119172690000")
	resFicT1 := parseString("52950073801824400000000")
	expectedResFicT0 := parseString("7112176615688650553")
	expectedResFicT1 := parseString("58304471112302341135376")

	newResFicIn, newResFicOut := computeReserveFic(
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

func TestComputeReserveFicEthInTrueOeGT1(t *testing.T) {
	// ETH_in, oe > 1, line 23
	resT0 := parseString("12668420462955600000")
	resT1 := parseString("103877534648498000000000")
	resFicT0 := parseString("6332837569656430000")
	resFicT1 := parseString("51951123826036400000000")
	expectedResFicT0 := parseString("6329892508211233858")
	expectedResFicT1 := parseString("51926964158252125695036")

	newResFicIn, newResFicOut := computeReserveFic(
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

func parseString(value string) *big.Int {
	newValue := new(big.Int)
	newValue.SetString(value, 10)
	return newValue
}
