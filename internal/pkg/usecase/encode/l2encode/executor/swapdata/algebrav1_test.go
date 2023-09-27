package swapdata

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var packAlgebraV1Pairs = []struct {
	data       AlgebraV1
	packedData string
}{
	{
		data: AlgebraV1{
			Recipient:           common.HexToAddress("0x8857d848e9094b473663F448134fd8a94e5C7C46"),
			Pool:                common.HexToAddress("0x99f74674bdb885ec5915fac225d069255cc9ae07"),
			TokenOut:            common.HexToAddress("0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270"),
			SwapAmount:          big.NewInt(96000000),
			SenderFeeOnTransfer: big.NewInt(1),

			isFirstSwap: true,
		},
		packedData: "008857d848e9094b473663f448134fd8a94e5c7c4600000099f74674bdb885ec5915fac225d069255cc9ae070d500b1d8e8ef31e21c99d1db9a6444d3adf127000000000000000000000000005b8d8000000000000000000000000000000000000000000000000000000000000000001",
	},
}

func TestPackAlgebraV1(t *testing.T) {
	t.Parallel()

	for idx, pair := range packAlgebraV1Pairs {
		t.Run(fmt.Sprintf("it should pack correctly %d", idx), func(t *testing.T) {
			result, err := packAlgebraV1(pair.data)

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.packedData, common.Bytes2Hex(result))
		})
	}
}

func TestUnpackAlgebraV1(t *testing.T) {
	t.Parallel()

	for idx, pair := range packAlgebraV1Pairs {
		t.Run(fmt.Sprintf("it should decode correctly %d", idx), func(t *testing.T) {
			result, err := UnpackAlgebraV1(
				common.Hex2Bytes(pair.packedData),
				pair.data.isFirstSwap,
			)

			assert.ErrorIs(t, err, nil)
			assert.EqualValues(t, pair.data, result)
		})
	}
}
