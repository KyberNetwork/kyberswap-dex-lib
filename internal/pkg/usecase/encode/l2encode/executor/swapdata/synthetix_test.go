package swapdata

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var packSynthetixPairs = []struct {
	data       Synthetix
	packedData string
}{
	{
		data: Synthetix{
			SynthetixProxy:         common.HexToAddress("0x60beef336a37618bf66373361ace074b0b34bc0b"),
			SourceCurrencyKey:      [32]byte{1, 2, 3},
			SourceAmount:           big.NewInt(1),
			DestinationCurrencyKey: [32]byte{2, 3, 4},
			UseAtomicExchange:      true,
			isFirstSwap:            true,
		},
		packedData: "00000060beef336a37618bf66373361ace074b0b34bc0b0000000000000000000000000000000000000000010203000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001020304000000000000000000000000000000000000000000000000000000000001",
	},
}

func TestPackSynthetix(t *testing.T) {
	t.Parallel()

	for idx, pair := range packSynthetixPairs {
		t.Run(fmt.Sprintf("it should pack correctly %d", idx), func(t *testing.T) {
			result, err := packSynthetix(pair.data)

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.packedData, common.Bytes2Hex(result))
		})
	}
}

func TestUnpackSynthetix(t *testing.T) {
	t.Parallel()

	for idx, pair := range packSynthetixPairs {
		t.Run(fmt.Sprintf("it should decode correctly %d", idx), func(t *testing.T) {
			result, err := UnpackSynthetix(
				common.Hex2Bytes(pair.packedData),
				pair.data.isFirstSwap,
			)

			assert.ErrorIs(t, err, nil)
			assert.EqualValues(t, pair.data, result)
		})
	}
}
