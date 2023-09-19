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
			Vault:       common.HexToAddress("0xafe57004eca7b85ba711130cb1b551d3d0b3c623"),
			PoolId:      [32]byte{1, 2, 3, 4, 5},
			AssetOut:    1,
			Amount:      big.NewInt(100000),
			isFirstSwap: true,
		},
		packedData: "000000afe57004eca7b85ba711130cb1b551d3d0b3c623010203040500000000000000000000000000000000000000000000000000000001000000000000000000000000000186a0",
	},
}

func TestPackBalancerV2(t *testing.T) {
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
			result, err := UnpackBalancerV2(
				common.Hex2Bytes(pair.packedData),
				pair.data.isFirstSwap,
			)

			assert.ErrorIs(t, err, nil)
			assert.EqualValues(t, pair.data, result)
		})
	}
}
