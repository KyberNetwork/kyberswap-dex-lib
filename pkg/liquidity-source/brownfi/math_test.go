package brownfi

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func u256(s string) *uint256.Int {
	return big256.NewUint256(s)
}

var testcases = []struct {
	amountIn        *uint256.Int
	reserveOut      *uint256.Int
	kappa           *uint256.Int
	oPrice          *uint256.Int
	fee             *uint256.Int
	feePrecision    *uint256.Int
	isSell          bool
	expectErr       error
	expectDelta     *uint256.Int
	expectAmountOut *uint256.Int
}{
	{
		// https://berascan.com/tx/0x2bdfdfc8945cca2822b492514610b374a8b758b1d99c20c6b838fb80fad3c84f
		amountIn:        u256("500000000000000000"),
		reserveOut:      u256("1055599299877346666213"),
		kappa:           u256("340282366920938463463374607431768211"),
		oPrice:          u256("1352423513467265103735019722083234772018"),
		fee:             u256("15"),
		feePrecision:    u256("10000"),
		isSell:          false,
		expectErr:       nil,
		expectDelta:     u256("17597108146711889152683044192958681422771568"),
		expectAmountOut: u256("125615947866181366"),
	},
	{
		// https://berascan.com/tx/0xe799229f925aed7e6ecb06f31998905dd67f089838e9a0aaefdb6fbecb875e47
		amountIn:        u256("999999999999999998"),
		reserveOut:      u256("844393591061170837668"),
		kappa:           u256("340282366920938463463374607431768211"),
		oPrice:          u256("1355831955198352353063685248472321152625"),
		fee:             u256("15"),
		feePrecision:    u256("10000"),
		isSell:          true,
		expectErr:       nil,
		expectDelta:     u256("706294283474802688273732335883746562347584"),
		expectAmountOut: u256("3978445930977858145"),
	},
}

func TestAmountOut(t *testing.T) {
	for _, testcase := range testcases {
		delta := calcDelta(
			testcase.amountIn,
			testcase.reserveOut,
			testcase.kappa,
			testcase.oPrice,
			testcase.isSell,
		)
		assert.Equal(t, testcase.expectDelta, delta)
		amountOut := getAmountOut(
			testcase.amountIn,
			testcase.reserveOut,
			testcase.kappa,
			testcase.oPrice,
			testcase.fee,
			testcase.feePrecision,
			testcase.isSell,
		)
		assert.Equal(t, testcase.expectAmountOut, amountOut)
	}
}
