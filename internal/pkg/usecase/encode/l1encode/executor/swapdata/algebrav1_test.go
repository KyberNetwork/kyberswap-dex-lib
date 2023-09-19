package swapdata

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var packAlgebraV1SwapPairs = []struct {
	data       AlgebraV1
	packedData string
}{
	{
		data: AlgebraV1{
			Recipient:           common.HexToAddress("0x8857d848e9094b473663F448134fd8a94e5C7C46"),
			Pool:                common.HexToAddress("0x99f74674bdb885ec5915fac225d069255cc9ae07"),
			TokenIn:             common.HexToAddress("0xfcb5a415c5665a2868e9afc776fb506e127fd373"),
			TokenOut:            common.HexToAddress("0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270"),
			SwapAmount:          big.NewInt(96000000),
			SqrtPriceLimitX96:   big.NewInt(0),
			SenderFeeOnTransfer: big.NewInt(0),
		},
		packedData: "0000000000000000000000008857d848e9094b473663f448134fd8a94e5c7c4600000000000000000000000099f74674bdb885ec5915fac225d069255cc9ae07000000000000000000000000fcb5a415c5665a2868e9afc776fb506e127fd3730000000000000000000000000d500b1d8e8ef31e21c99d1db9a6444d3adf12700000000000000000000000000000000000000000000000000000000005b8d80000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
	},
}

func TestPackAlgebraV1(t *testing.T) {
	t.Parallel()

	for idx, pair := range packAlgebraV1SwapPairs {
		t.Run(fmt.Sprintf("it should pack correctly %d", idx), func(t *testing.T) {
			result, err := packAlgebraV1(pair.data)

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.packedData, common.Bytes2Hex(result))
		})
	}
}

func TestUnpackAlgebraV1(t *testing.T) {
	t.Parallel()

	for idx, pair := range packAlgebraV1SwapPairs {
		t.Run(fmt.Sprintf("it should decode correctly %d", idx), func(t *testing.T) {
			result, err := UnpackAlgebraV1(common.Hex2Bytes(pair.packedData))

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.data.Pool, result.Pool)
			assert.Equal(t, pair.data.TokenIn, result.TokenIn)
			assert.Equal(t, pair.data.TokenOut, result.TokenOut)
			assert.Equal(t, pair.data.Recipient, result.Recipient)
			assert.Equal(t, pair.data.SwapAmount.String(), result.SwapAmount.String())
			assert.Equal(t, pair.data.SqrtPriceLimitX96.String(), result.SqrtPriceLimitX96.String())
		})
	}
}
