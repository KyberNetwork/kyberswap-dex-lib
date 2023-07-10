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
			TokenIn:                common.HexToAddress("0x80e526c863632a31ac0787cf7cef32c0adcddfe0"),
			TokenOut:               common.HexToAddress("0x0d9a92ffd756d8587541614800c06feb7ae2ee1a"),
			SourceCurrencyKey:      [32]byte{1, 2, 3},
			SourceAmount:           big.NewInt(1),
			DestinationCurrencyKey: [32]byte{2, 3, 4},
			UseAtomicExchange:      true,
		},
		packedData: "00000000000000000000000060beef336a37618bf66373361ace074b0b34bc0b00000000000000000000000080e526c863632a31ac0787cf7cef32c0adcddfe00000000000000000000000000d9a92ffd756d8587541614800c06feb7ae2ee1a0102030000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000102030400000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001",
	},
}

func Test_packSynthetix(t *testing.T) {
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
			result, err := UnpackSynthetix(common.Hex2Bytes(pair.packedData))

			assert.ErrorIs(t, err, nil)
			assert.EqualValues(t, pair.data, result)
		})
	}
}
