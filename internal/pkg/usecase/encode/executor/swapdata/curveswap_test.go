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
			Pool:              common.HexToAddress("0xafe57004eca7b85ba711130cb1b551d3d0b3c623"),
			TokenFrom:         common.HexToAddress("0x8211c35c244e3a849ffa9e53b8cd17d80caa68a4"),
			TokenTo:           common.HexToAddress("0x563d45366a1266076c6b81c3e984d3f52c4f6c76"),
			TokenIndexFrom:    big.NewInt(1),
			TokenIndexTo:      big.NewInt(2),
			Dx:                big.NewInt(1),
			MinDy:             big.NewInt(2),
			UsePoolUnderlying: true,
			UseTriCrypto:      true,
		},
		packedData: "000000000000000000000000afe57004eca7b85ba711130cb1b551d3d0b3c6230000000000000000000000008211c35c244e3a849ffa9e53b8cd17d80caa68a4000000000000000000000000563d45366a1266076c6b81c3e984d3f52c4f6c76000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000001",
	},
}

func Test_packCurveSwap(t *testing.T) {
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
			result, err := UnpackCurveSwap(common.Hex2Bytes(pair.packedData))

			assert.ErrorIs(t, err, nil)
			assert.EqualValues(t, pair.data, result)
		})
	}
}
