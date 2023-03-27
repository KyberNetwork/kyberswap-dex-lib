package swapdata

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var packBiswapPairs = []struct {
	data       UniSwap
	packedData string
}{
	{
		data: UniSwap{
			Pool:              common.HexToAddress("0xafe57004eca7b85ba711130cb1b551d3d0b3c623"),
			TokenIn:           common.HexToAddress("0xa66cc4b4c17361532f0baba708941b7b8cdf7aa0"),
			TokenOut:          common.HexToAddress("0x2771a9fdbaf7d37679116191007c4829cf7616d2"),
			Recipient:         common.HexToAddress("0xdeea7249f436cfdb360a8b6725aca01c604735b1"),
			CollectAmount:     big.NewInt(10000000),
			LimitReturnAmount: big.NewInt(1),
			SwapFee:           2,
			FeePrecision:      1000,
			TokenWeightInput:  50,
		},
		packedData: "000000000000000000000000afe57004eca7b85ba711130cb1b551d3d0b3c623000000000000000000000000a66cc4b4c17361532f0baba708941b7b8cdf7aa00000000000000000000000002771a9fdbaf7d37679116191007c4829cf7616d2000000000000000000000000deea7249f436cfdb360a8b6725aca01c604735b100000000000000000000000000000000000000000000000000000000009896800000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000003e80000000000000000000000000000000000000000000000000000000000000032",
	},
}

func Test_packBiswap(t *testing.T) {
	t.Parallel()

	for idx, pair := range packBiswapPairs {
		t.Run(fmt.Sprintf("it should pack correctly %d", idx), func(t *testing.T) {
			result, err := packBiswap(pair.data)

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.packedData, common.Bytes2Hex(result))
		})
	}
}

func TestUnpackBiwap(t *testing.T) {
	t.Parallel()

	for idx, pair := range packBiswapPairs {
		t.Run(fmt.Sprintf("it should decode correctly %d", idx), func(t *testing.T) {
			result, err := UnpackUniSwap(common.Hex2Bytes(pair.packedData))

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.data, result)
		})
	}
}
