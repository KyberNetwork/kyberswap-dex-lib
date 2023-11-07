package swapdata

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var packSmardexPairs = []struct {
	data       Smardex
	packedData string
}{
	{
		data: Smardex{
			Recipient: common.HexToAddress("0x8857d848e9094b473663F448134fd8a94e5C7C46"),
			Pool:      common.HexToAddress("0x99f74674bdb885ec5915fac225d069255cc9ae07"),
			TokenIn:   common.HexToAddress("0x82af49447d8a07e3bd95bd0d56f35241523fbab1"),
			TokenOut:  common.HexToAddress("0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270"),
			Amount:    big.NewInt(96000000),

			isFirstSwap: true,
		},
		packedData: "008857d848e9094b473663f448134fd8a94e5c7c4600000099f74674bdb885ec5915fac225d069255cc9ae0782af49447d8a07e3bd95bd0d56f35241523fbab10d500b1d8e8ef31e21c99d1db9a6444d3adf127000000000000000000000000005b8d800",
	},
}

func TestPackSmardex(t *testing.T) {
	t.Parallel()

	for idx, pair := range packSmardexPairs {
		t.Run(fmt.Sprintf("it should pack correctly %d", idx), func(t *testing.T) {
			result, err := packSmardex(pair.data)

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.packedData, common.Bytes2Hex(result))
		})
	}
}

func TestUnpackSmardex(t *testing.T) {
	t.Parallel()

	for idx, pair := range packSmardexPairs {
		t.Run(fmt.Sprintf("it should decode correctly %d", idx), func(t *testing.T) {
			result, err := UnpackSmardex(
				common.Hex2Bytes(pair.packedData),
				pair.data.isFirstSwap,
			)

			assert.ErrorIs(t, err, nil)
			assert.EqualValues(t, pair.data, result)
		})
	}
}
