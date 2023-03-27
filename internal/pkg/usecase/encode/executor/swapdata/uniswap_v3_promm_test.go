package swapdata

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var packUniSwapV3ProMMPairs = []struct {
	data       UniSwapV3ProMM
	packedData string
}{
	{
		data: UniSwapV3ProMM{
			Recipient:         common.HexToAddress("0xafe57004eca7b85ba711130cb1b551d3d0b3c623"),
			Pool:              common.HexToAddress("0x8211c35c244e3a849ffa9e53b8cd17d80caa68a4"),
			TokenIn:           common.HexToAddress("0x563d45366a1266076c6b81c3e984d3f52c4f6c76"),
			TokenOut:          common.HexToAddress("0xcb5ca33cc86a8b070654c6c8d5d29089b16028fd"),
			SwapAmount:        big.NewInt(10000),
			LimitReturnAmount: big.NewInt(1),
			SqrtPriceLimitX96: big.NewInt(100000),
			IsUniV3:           true,
		},
		packedData: "000000000000000000000000afe57004eca7b85ba711130cb1b551d3d0b3c6230000000000000000000000008211c35c244e3a849ffa9e53b8cd17d80caa68a4000000000000000000000000563d45366a1266076c6b81c3e984d3f52c4f6c76000000000000000000000000cb5ca33cc86a8b070654c6c8d5d29089b16028fd0000000000000000000000000000000000000000000000000000000000002710000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000186a00000000000000000000000000000000000000000000000000000000000000001",
	},
}

func Test_packUniSwapV3ProMM(t *testing.T) {
	t.Parallel()

	for idx, pair := range packUniSwapV3ProMMPairs {
		t.Run(fmt.Sprintf("it should pack correctly %d", idx), func(t *testing.T) {
			result, err := packUniSwapV3ProMM(pair.data)

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.packedData, common.Bytes2Hex(result))
		})
	}
}

func TestUnpackUniSwapV3ProMM(t *testing.T) {
	t.Parallel()

	for idx, pair := range packUniSwapV3ProMMPairs {
		t.Run(fmt.Sprintf("it should decode correctly %d", idx), func(t *testing.T) {
			result, err := UnpackUniSwapV3ProMM(common.Hex2Bytes(pair.packedData))

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.data, result)
		})
	}
}
