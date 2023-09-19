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
			Pool:         common.HexToAddress("0xafe57004eca7b85ba711130cb1b551d3d0b3c623"),
			TokenIndexTo: 0,
			Dx:           big.NewInt(1),
			PoolLp:       common.HexToAddress("0xcb5ca33cc86a8b070654c6c8d5d29089b16028fd"),
			IsSaddle:     true,
			isFirstSwap:  true,
		},
		packedData: "000000afe57004eca7b85ba711130cb1b551d3d0b3c6230000000000000000000000000000000001cb5ca33cc86a8b070654c6c8d5d29089b16028fd01",
	},
}

func TestPackStableSwap(t *testing.T) {
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
			result, err := UnpackStableSwap(
				common.Hex2Bytes(pair.packedData),
				pair.data.isFirstSwap,
			)

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.data, result)
		})
	}
}
