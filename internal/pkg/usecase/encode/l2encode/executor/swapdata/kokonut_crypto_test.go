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
			TokenIndexFrom: 1,
			isFirstSwap:    true,
		},
		packedData: "000000afe57004eca7b85ba711130cb1b551d3d0b3c6230000000000000000000000000000000101",
	},
}

func TestKokonutCrypto(t *testing.T) {
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
			result, err := UnpackKokonutCrypto(
				common.Hex2Bytes(pair.packedData),
				pair.data.isFirstSwap,
			)

			assert.ErrorIs(t, err, nil)
			assert.EqualValues(t, pair.data, result)
		})
	}
}
