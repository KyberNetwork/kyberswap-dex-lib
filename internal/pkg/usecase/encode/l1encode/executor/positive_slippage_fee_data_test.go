package executor

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestPackPositiveSlippageFee(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		data           PositiveSlippageFeeData
		expectedResult string
		expectedError  error
	}{
		{
			name: "it should pack PositiveSlippageFeeData correctly",
			data: PositiveSlippageFeeData{
				MinimumPSAmount:      big.NewInt(30),
				ExpectedReturnAmount: big.NewInt(1000),
			},
			expectedResult: "0000000000000000000000000000001e000000000000000000000000000003e8",
			expectedError:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := PackPositiveSlippageFeeData(tc.data)

			assert.Equal(t, tc.expectedResult, common.Bytes2Hex(result))
			assert.ErrorIs(t, err, tc.expectedError)
		})
	}
}

func TestUnpackPositiveSlippageFee(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		encodedData    string
		expectedResult PositiveSlippageFeeData
		expectedError  error
	}{
		{
			name:        "it should unpack PositiveSlippageFeeData correctly",
			encodedData: "0000000000000000000000000000001e000000000000000000000000000003e8",
			expectedResult: PositiveSlippageFeeData{
				MinimumPSAmount:      big.NewInt(30),
				ExpectedReturnAmount: big.NewInt(1000),
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := UnpackPositiveSlippageFeeData(common.Hex2Bytes(tc.encodedData))

			assert.Equal(t, tc.expectedResult, result)
			assert.ErrorIs(t, err, tc.expectedError)
		})
	}
}
