package swapdata

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var packCurveSwapPairs = []struct {
	data       CurveSwap
	packedData string
}{
	{
		data: CurveSwap{
			CanGetToken:       true,
			Pool:              common.HexToAddress("0xafe57004eca7b85ba711130cb1b551d3d0b3c623"),
			TokenIndexTo:      2,
			Dx:                big.NewInt(1),
			UsePoolUnderlying: true,
			UseTriCrypto:      true,
			isFirstSwap:       true,
		},
		packedData: "01000000afe57004eca7b85ba711130cb1b551d3d0b3c62302000000000000000000000000000000010101",
	},
}

func TestPackCurveSwap(t *testing.T) {
	t.Parallel()

	for idx, pair := range packCurveSwapPairs {
		t.Run(fmt.Sprintf("it should pack correctly %d", idx), func(t *testing.T) {
			result, err := packCurveSwap(pair.data)

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.packedData, common.Bytes2Hex(result))
		})
	}
}

func TestUnpackCurveSwap(t *testing.T) {
	t.Parallel()

	for idx, pair := range packCurveSwapPairs {
		t.Run(fmt.Sprintf("it should decode correctly %d", idx), func(t *testing.T) {
			result, err := UnpackCurveSwap(
				common.Hex2Bytes(pair.packedData),
				pair.data.isFirstSwap,
			)

			assert.ErrorIs(t, err, nil)
			assert.EqualValues(t, pair.data, result)
		})
	}
}
