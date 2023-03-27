package executor

import (
	"fmt"
	"testing"

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
					FunctionSelector: [4]byte{2, 2, 3, 4},
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
