package swapdata

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var packUniSwapV3ProMMPairs = []struct {
	data       UniswapV3KSElastic
	packedData string
}{
	{
		data: UniswapV3KSElastic{
			Recipient:     common.HexToAddress("0xafe57004eca7b85ba711130cb1b551d3d0b3c623"),
			Pool:          common.HexToAddress("0x8211c35c244e3a849ffa9e53b8cd17d80caa68a4"),
			SwapAmount:    big.NewInt(10000),
			IsUniV3:       true,
			isFirstSwap:   true,
			recipientFlag: 0,
		},
		packedData: "00afe57004eca7b85ba711130cb1b551d3d0b3c6230000008211c35c244e3a849ffa9e53b8cd17d80caa68a40000000000000000000000000000271001",
	},
}

func TestPackUniSwapV3ProMM(t *testing.T) {
	t.Parallel()

	for idx, pair := range packUniSwapV3ProMMPairs {
		t.Run(fmt.Sprintf("it should pack correctly %d", idx), func(t *testing.T) {
			result, err := packUniswapV3KSElastic(pair.data)

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.packedData, common.Bytes2Hex(result))
		})
	}
}

func TestUnpackUniSwapV3ProMM(t *testing.T) {
	t.Parallel()

	for idx, pair := range packUniSwapV3ProMMPairs {
		t.Run(fmt.Sprintf("it should decode correctly %d", idx), func(t *testing.T) {
			result, err := UnpackUniswapV3KSElastic(
				common.Hex2Bytes(pair.packedData),
				pair.data.isFirstSwap,
			)

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.data, result)
		})
	}
}
