package executor

import (
	"fmt"
	"testing"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var packSwapSingleSequenceInputsPairs = []struct {
	data       SwapSingleSequenceInputs
	packedData string
}{
	{
		data: SwapSingleSequenceInputs{
			SwapData: []Swap{
				{
					Data:             []byte("data 1.2"),
					SelectorAndFlags: SwapSelectorAndFlags{2, 2, 3, 4},
				},
			},
		},
		packedData: "0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000040020203040000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000086461746120312e32000000000000000000000000000000000000000000000000",
	},
}

func TestPackSwapSingleSequenceInputs(t *testing.T) {
	t.Parallel()

	for idx, pair := range packSwapSingleSequenceInputsPairs {
		t.Run(fmt.Sprintf("it should pack correctly %d", idx), func(t *testing.T) {
			result, err := PackSwapSingleSequenceInputs(pair.data)

			assert.Nil(t, err)
			assert.Equal(t, pair.packedData, common.Bytes2Hex(result))
		})
	}
}

func TestUnpackSwapSingleSequenceInputs(t *testing.T) {
	for idx, pair := range packSwapSingleSequenceInputsPairs {
		t.Run(fmt.Sprintf("it should decode correctly %d", idx), func(t *testing.T) {
			result, err := UnpackSwapSingleSequenceInputs(common.Hex2Bytes(pair.packedData))

			assert.Nil(t, err)
			assert.Equal(t, pair.data, result)
		})
	}
}

func TestGetSwapFlags(t *testing.T) {
	testCases := []struct {
		name           string
		flags          []types.EncodingSwapFlag
		expectedResult SwapFlags
	}{
		{
			name: "it should parse correctly with using last byte only (current case)",
			flags: []types.EncodingSwapFlag{
				types.EncodingSwapFlagShouldNotKeepDustTokenOut,
				types.EncodingSwapFlagShouldApproveMax,
			},
			expectedResult: SwapFlags{3, 0, 0, 0},
		},
		{
			name: "it should parse correctly with any flag bits",
			flags: []types.EncodingSwapFlag{
				{Value: 256},
				{Value: 2147483648},
				types.EncodingSwapFlagShouldNotKeepDustTokenOut,
				types.EncodingSwapFlagShouldApproveMax,
			},
			expectedResult: SwapFlags{3, 1, 0, 128},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := getSwapFlags(tc.flags)

			assert.Equal(t, tc.expectedResult, result)
		})

	}

}

func TestBuildSelectorAndFlags(t *testing.T) {
	selector := [4]byte{1, 2, 3, 4}
	flags := [4]byte{5, 6, 7, 8}

	build := buildSelectorAndFlags(selector, flags)

	// The result should be {1, 2, 3, 4, 0, ..., 0, 8, 7, 6, 5}
	for idx := 0; idx < 4; idx++ {
		assert.Equal(t, selector[idx], build[idx])
		assert.Equal(t, flags[idx], build[len(build)-1-idx])
	}
}
