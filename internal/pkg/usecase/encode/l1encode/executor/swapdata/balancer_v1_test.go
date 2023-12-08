package swapdata

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var packBalancerV1Pairs = []struct {
	data       BalancerV1
	packedData string
}{
	{
		data: BalancerV1{
			Pool:     common.HexToAddress("0xa8501eae18f4ec4d27063a659873c9f88726ec3b"),
			TokenIn:  common.HexToAddress("0x563d45366a1266076c6b81c3e984d3f52c4f6c76"),
			TokenOut: common.HexToAddress("0x8211c35c244e3a849ffa9e53b8cd17d80caa68a4"),
			Amount:   big.NewInt(100000),
		},
		packedData: "000000000000000000000000a8501eae18f4ec4d27063a659873c9f88726ec3b" +
			"000000000000000000000000563d45366a1266076c6b81c3e984d3f52c4f6c76" +
			"0000000000000000000000008211c35c244e3a849ffa9e53b8cd17d80caa68a4" +
			"00000000000000000000000000000000000000000000000000000000000186a0",
	},
}

func Test_packBalancerV1(t *testing.T) {
	t.Parallel()

	for idx, pair := range packBalancerV1Pairs {
		t.Run(fmt.Sprintf("it should pack correctly %d", idx), func(t *testing.T) {
			result, err := packBalancerV1(pair.data)

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.packedData, common.Bytes2Hex(result))
		})
	}
}

func TestUnpackBalancerV1(t *testing.T) {
	t.Parallel()

	for idx, pair := range packBalancerV1Pairs {
		t.Run(fmt.Sprintf("it should decode correctly %d", idx), func(t *testing.T) {
			result, err := UnpackBalancerV1(common.Hex2Bytes(pair.packedData))

			assert.ErrorIs(t, err, nil)
			assert.EqualValues(t, pair.data, result)
		})
	}
}
