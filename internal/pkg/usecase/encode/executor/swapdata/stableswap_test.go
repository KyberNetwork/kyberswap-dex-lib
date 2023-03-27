package swapdata

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var packStableSwapPairs = []struct {
	data       StableSwap
	packedData string
}{
	{
		data: StableSwap{
			Pool:           common.HexToAddress("0xafe57004eca7b85ba711130cb1b551d3d0b3c623"),
			TokenFrom:      common.HexToAddress("0x8211c35c244e3a849ffa9e53b8cd17d80caa68a4"),
			TokenTo:        common.HexToAddress("0x563d45366a1266076c6b81c3e984d3f52c4f6c76"),
			TokenIndexFrom: 1,
			TokenIndexTo:   0,
			Dx:             big.NewInt(1),
			MinDy:          big.NewInt(2),
			PoolLength:     big.NewInt(2),
			PoolLp:         common.HexToAddress("0xcb5ca33cc86a8b070654c6c8d5d29089b16028fd"),
			IsSaddle:       true,
		},
		packedData: "000000000000000000000000afe57004eca7b85ba711130cb1b551d3d0b3c6230000000000000000000000008211c35c244e3a849ffa9e53b8cd17d80caa68a4000000000000000000000000563d45366a1266076c6b81c3e984d3f52c4f6c7600000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000002000000000000000000000000cb5ca33cc86a8b070654c6c8d5d29089b16028fd0000000000000000000000000000000000000000000000000000000000000001",
	},
}

func Test_packStableSwap(t *testing.T) {
	t.Parallel()

	for idx, pair := range packStableSwapPairs {
		t.Run(fmt.Sprintf("it should pack correctly %d", idx), func(t *testing.T) {
			result, err := packStableSwap(pair.data)

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.packedData, common.Bytes2Hex(result))
		})
	}
}

func TestUnpackStableSwap(t *testing.T) {
	t.Parallel()

	for idx, pair := range packStableSwapPairs {
		t.Run(fmt.Sprintf("it should decode correctly %d", idx), func(t *testing.T) {
			result, err := UnpackStableSwap(common.Hex2Bytes(pair.packedData))

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.data, result)
		})
	}
}
