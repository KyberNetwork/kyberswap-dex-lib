package swapdata

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var packUniSwapPairs = []struct {
	data       Uniswap
	packedData string
}{
	{
		data: Uniswap{
			Pool:             common.HexToAddress("0xafe57004eca7b85ba711130cb1b551d3d0b3c623"),
			Recipient:        common.HexToAddress("0xdeea7249f436cfdb360a8b6725aca01c604735b1"),
			CollectAmount:    big.NewInt(10000000),
			SwapFee:          30,
			FeePrecision:     10000,
			TokenWeightInput: 50,
			isFirstSwap:      true,
			recipientFlag:    0,
		},
		packedData: "000000afe57004eca7b85ba711130cb1b551d3d0b3c62300deea7249f436cfdb360a8b6725aca01c604735b1000000000000000000000000009896800000001e0000271000000032",
	},
}

func TestPackUniswap(t *testing.T) {
	t.Parallel()

	for idx, pair := range packUniSwapPairs {
		t.Run(fmt.Sprintf("it should pack correctly %d", idx), func(t *testing.T) {
			result, err := packUniswap(pair.data)

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.packedData, common.Bytes2Hex(result))
		})
	}
}

func TestUnpackUniswap(t *testing.T) {
	t.Parallel()

	for idx, pair := range packUniSwapPairs {
		t.Run(fmt.Sprintf("it should decode correctly %d", idx), func(t *testing.T) {
			result, err := UnpackUniswap(
				common.Hex2Bytes(pair.packedData),
				pair.data.isFirstSwap,
			)

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.data, result)
		})
	}
}
