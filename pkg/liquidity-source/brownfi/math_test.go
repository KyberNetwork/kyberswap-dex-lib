package brownfi

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

func s(s string) *uint256.Int {
	res, _ := uint256.FromDecimal(s)
	return res
}

var testcases = []struct {
	amountIn        *uint256.Int
	reserveOut      *uint256.Int
	kappa           *uint256.Int
	oPrice          *uint256.Int
	isSell          bool
	fee             *uint256.Int
	feePrecision    *uint256.Int
	expectErr       error
	expectDelta     *uint256.Int
	expectSqrtDelta *uint256.Int
	expectAmountOut *uint256.Int
}{
	{
		// https://berascan.com/tx/0x2bdfdfc8945cca2822b492514610b374a8b758b1d99c20c6b838fb80fad3c84f
		amountIn:        s("500000000000000000"),
		reserveOut:      s("1055599299877346666213"),
		kappa:           s("340282366920938463463374607431768211"),
		oPrice:          s("1352423513467265103735019722083234772018"),
		isSell:          false,
		fee:             s("15"),
		feePrecision:    s("10000"),
		expectErr:       nil,
		expectDelta:     s("17597108146711889152683044192958681422771568"),
		expectSqrtDelta: s("4194890719281241199075"),
		expectAmountOut: s("125615947866181366"),
	},
	{
		// https://berascan.com/tx/0xe799229f925aed7e6ecb06f31998905dd67f089838e9a0aaefdb6fbecb875e47
		amountIn:        s("999999999999999998"),
		reserveOut:      s("844393591061170837668"),
		kappa:           s("340282366920938463463374607431768211"),
		oPrice:          s("1355831955198352353063685248472321152625"),
		isSell:          true,
		fee:             s("15"),
		feePrecision:    s("10000"),
		expectErr:       nil,
		expectDelta:     s("706294283474802688273732335883746562347584"),
		expectSqrtDelta: s("840413162364085471177"),
		expectAmountOut: s("3978445930977858145"),
	},
}

func TestAmountOut(t *testing.T) {
	for _, testcase := range testcases {
		delta, err := delta(
			testcase.amountIn,
			testcase.reserveOut,
			testcase.kappa,
			testcase.oPrice,
			testcase.isSell,
		)
		assert.Nil(t, err)
		assert.Equal(t, testcase.expectDelta, delta)
		assert.Equal(t, testcase.expectSqrtDelta, sqrt(delta))
		amountOut := getAmountOut(
			testcase.amountIn,
			testcase.reserveOut,
			testcase.kappa,
			testcase.oPrice,
			testcase.isSell,
			testcase.fee,
			testcase.feePrecision,
		)
		assert.Equal(t, testcase.expectAmountOut, amountOut)
	}
}
