package usdfi

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
			Name:           "Test volatile: normal case",
			amountIn:       setBigValue("1000000"),
			reserveIn:      setBigValue("734057310691"),
			reserveOut:     setBigValue("619890288342243016669"),
			decimalIn:      setBigValue("1000000"),
			decimalOut:     setBigValue("1000000000000000000"),
			swapFee:        setBigValue("200000000000000"),
			stable:         false,
			expectedOutput: setBigValue("844301197094305"),
		},
		{
			Name:           "Test volatile: small input case",
			amountIn:       setBigValue("1"),
			reserveIn:      setBigValue("734158559194"),
			reserveOut:     setBigValue("619810940630240220120"),
			decimalIn:      setBigValue("1000000"),
			decimalOut:     setBigValue("1000000000000000000"),
			swapFee:        setBigValue("200000000000000"),
			stable:         false,
			expectedOutput: setBigValue("844246699"),
		},
		{
			Name:           "Test volatile: big input case",
			amountIn:       setBigValue("1000000000000000000000000"),
			reserveIn:      setBigValue("734158559194"),
			reserveOut:     setBigValue("619810940630240220120"),
			decimalIn:      setBigValue("1000000"),
			decimalOut:     setBigValue("1000000000000000000"),
			swapFee:        setBigValue("200000000000000"),
			stable:         false,
			expectedOutput: setBigValue("619810940629785089586"),
		},
		{
			Name:           "Test stable: normal case",
			amountIn:       setBigValue("1000000"),
			reserveIn:      setBigValue("1597357611912"),
			reserveOut:     setBigValue("1069785590448332953758441"),
			decimalIn:      setBigValue("1000000"),
			decimalOut:     setBigValue("1000000000000000000"),
			swapFee:        setBigValue("200000000000000"),
			stable:         true,
			expectedOutput: setBigValue("984443098823156057"),
		},
		{
			Name:           "Test stable: small input case",
			amountIn:       setBigValue("1"),
			reserveIn:      setBigValue("1597357611912"),
			reserveOut:     setBigValue("1069785590448332953758441"),
			decimalIn:      setBigValue("1000000"),
			decimalOut:     setBigValue("1000000000000000000"),
			swapFee:        setBigValue("200000000000000"),
			stable:         true,
			expectedOutput: setBigValue("984640112686"),
		},
		{
			Name:           "Test stable: big input case",
			amountIn:       setBigValue("1000000000000000000000000"),
			reserveIn:      setBigValue("1597357611912"),
			reserveOut:     setBigValue("1069785590448332953758441"),
			decimalIn:      setBigValue("1000000"),
			decimalOut:     setBigValue("1000000000000000000"),
			swapFee:        setBigValue("200000000000000"),
			stable:         true,
			expectedOutput: setBigValue("1069785590448332953758440"),
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
