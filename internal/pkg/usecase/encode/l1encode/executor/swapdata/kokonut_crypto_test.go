package swapdata

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var packKokonutCryptoPairs = []struct {
	data       KokonutCrypto
	packedData string
}{
	{
		data: KokonutCrypto{
			Pool:           common.HexToAddress("0xafe57004eca7b85ba711130cb1b551d3d0b3c623"),
			Dx:             big.NewInt(1),
			TokenIndexFrom: big.NewInt(1),
			FromToken:      common.HexToAddress("0x8211c35c244e3a849ffa9e53b8cd17d80caa68a4"),
			ToToken:        common.HexToAddress("0x563d45366a1266076c6b81c3e984d3f52c4f6c76"),
		},
		packedData: "000000000000000000000000afe57004eca7b85ba711130cb1b551d3d0b3c623000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000010000000000000000000000008211c35c244e3a849ffa9e53b8cd17d80caa68a4000000000000000000000000563d45366a1266076c6b81c3e984d3f52c4f6c76",
	},
}

func TestPackKokonutCrypto(t *testing.T) {
	t.Parallel()

	for idx, pair := range packKokonutCryptoPairs {
		t.Run(fmt.Sprintf("it should pack correctly %d", idx), func(t *testing.T) {
			result, err := packKokonutCrypto(pair.data)

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.packedData, common.Bytes2Hex(result))
		})
	}
}

func TestUnpackKokonutCrypto(t *testing.T) {
	t.Parallel()

	for idx, pair := range packKokonutCryptoPairs {
		t.Run(fmt.Sprintf("it should decode correctly %d", idx), func(t *testing.T) {
			result, err := UnpackKokonutCrypto(common.Hex2Bytes(pair.packedData))

			assert.ErrorIs(t, err, nil)
			assert.EqualValues(t, pair.data, result)
		})
	}
}
