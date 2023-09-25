package swapdata

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var packPancakeStableSwapPairs = []struct {
	data       CurveSwap
	packedData string
}{
	{
		data: CurveSwap{
			Pool:              common.HexToAddress("0xafe57004eca7b85ba711130cb1b551d3d0b3c623"),
			TokenFrom:         common.HexToAddress("0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c"), // WBNB
			TokenTo:           common.HexToAddress("0x563d45366a1266076c6b81c3e984d3f52c4f6c76"),
			TokenIndexFrom:    big.NewInt(1),
			TokenIndexTo:      big.NewInt(2),
			Dx:                big.NewInt(1),
			UsePoolUnderlying: true,
			UseTriCrypto:      false,
		},
		packedData: "000000000000000000000000afe57004eca7b85ba711130cb1b551d3d0b3c623000000000000000000000000bb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c000000000000000000000000563d45366a1266076c6b81c3e984d3f52c4f6c7600000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000000",
	},
}

func Test_packPancakeStableSwap(t *testing.T) {
	t.Parallel()

	for idx, pair := range packPancakeStableSwapPairs {
		t.Run(fmt.Sprintf("it should pack correctly %d", idx), func(t *testing.T) {
			result, err := packPancakeStableSwap(pair.data)

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.packedData, common.Bytes2Hex(result))
		})
	}
}

func TestUnpackPancakeStableSwap(t *testing.T) {
	t.Parallel()

	for idx, pair := range packPancakeStableSwapPairs {
		t.Run(fmt.Sprintf("it should decode correctly %d", idx), func(t *testing.T) {
			result, err := UnpackPancakeStableSwap(common.Hex2Bytes(pair.packedData))

			assert.ErrorIs(t, err, nil)
			assert.EqualValues(t, pair.data, result)
		})
	}
}
