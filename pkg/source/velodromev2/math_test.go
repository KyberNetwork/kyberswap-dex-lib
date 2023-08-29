package velodromev2

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAmountOut(t *testing.T) {
	testCases := []struct {
		Name           string
		amountIn       *big.Int
		reserveIn      *big.Int
		reserveOut     *big.Int
		decimalIn      *big.Int
		decimalOut     *big.Int
		swapFee        *big.Int
		stable         bool
		expectedOutput *big.Int
	}{
		{
			// https://optimistic.etherscan.io/address/0xC1E5b706d5b5FEc9aAca710fc03dcBD4356fb247#readContract
			Name:           "Test volatile",
			amountIn:       setBigValue("100000000000000"),
			reserveIn:      setBigValue("25331207543090011"),
			reserveOut:     setBigValue("3235407125910983309109"),
			decimalIn:      setBigValue("1000000000000000000"),
			decimalOut:     setBigValue("1000000000000000000"),
			swapFee:        setBigValue("3000000000000000"),
			stable:         false,
			expectedOutput: setBigValue("12684175344775399021"),
		},
		{
			// https://optimistic.etherscan.io/address/0x904f14F9ED81d0b0a40D8169B28592aac5687158#readContract
			Name:           "Test stable",
			amountIn:       setBigValue("100000000000000000000"),
			reserveIn:      setBigValue("15518240983398142455365"),
			reserveOut:     setBigValue("6346606574070997109228"),
			decimalIn:      setBigValue("1000000000000000000"),
			decimalOut:     setBigValue("1000000000000000000"),
			swapFee:        setBigValue("10000000000000000"),
			stable:         true,
			expectedOutput: setBigValue("85017301393639506495"),
		},
	}

	for _, tc := range testCases {
		test := tc
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()
			actualAmountOut := getAmountOut(test.amountIn, test.reserveIn, test.reserveOut, test.decimalIn, test.decimalOut, test.swapFee, test.stable)

			assert.Equal(t, test.expectedOutput, actualAmountOut)
		})
	}
}

func setBigValue(value string) *big.Int {
	bigValue, _ := new(big.Int).SetString(value, 10)
	return bigValue
}
