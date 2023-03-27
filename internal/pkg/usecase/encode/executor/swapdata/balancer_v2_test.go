package swapdata

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var packBalancerV2Pairs = []struct {
	data       BalancerV2
	packedData string
}{
	{
		data: BalancerV2{
			Vault:    common.HexToAddress("0xafe57004eca7b85ba711130cb1b551d3d0b3c623"),
			PoolId:   [32]byte{1, 2, 3, 4, 5},
			AssetIn:  common.HexToAddress("0x563d45366a1266076c6b81c3e984d3f52c4f6c76"),
			AssetOut: common.HexToAddress("0x8211c35c244e3a849ffa9e53b8cd17d80caa68a4"),
			Amount:   big.NewInt(100000),
			Limit:    big.NewInt(100000),
		},
		packedData: "000000000000000000000000afe57004eca7b85ba711130cb1b551d3d0b3c6230102030405000000000000000000000000000000000000000000000000000000000000000000000000000000563d45366a1266076c6b81c3e984d3f52c4f6c760000000000000000000000008211c35c244e3a849ffa9e53b8cd17d80caa68a400000000000000000000000000000000000000000000000000000000000186a000000000000000000000000000000000000000000000000000000000000186a0",
	},
}

func Test_packBalancerV2(t *testing.T) {
	t.Parallel()

	for idx, pair := range packBalancerV2Pairs {
		t.Run(fmt.Sprintf("it should pack correctly %d", idx), func(t *testing.T) {
			result, err := packBalancerV2(pair.data)

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.packedData, common.Bytes2Hex(result))
		})
	}
}

func TestUnpackBalancerV2(t *testing.T) {
	t.Parallel()

	for idx, pair := range packBalancerV2Pairs {
		t.Run(fmt.Sprintf("it should decode correctly %d", idx), func(t *testing.T) {
			result, err := UnpackBalancerV2(common.Hex2Bytes(pair.packedData))

			assert.ErrorIs(t, err, nil)
			assert.EqualValues(t, pair.data, result)
		})
	}
}
