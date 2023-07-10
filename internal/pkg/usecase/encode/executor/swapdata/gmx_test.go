package swapdata

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var packGMXPairs = []struct {
	data       GMX
	packedData string
}{
	{
		data: GMX{
			Vault:    common.HexToAddress("0x60beef336a37618bf66373361ace074b0b34bc0b"),
			TokenIn:  common.HexToAddress("0x80e526c863632a31ac0787cf7cef32c0adcddfe0"),
			TokenOut: common.HexToAddress("0x0d9a92ffd756d8587541614800c06feb7ae2ee1a"),
			Amount:   big.NewInt(10000),
			Receiver: common.HexToAddress("0x428d69ccc61c8c15993bf6925172dc85047a6abf"),
		},
		packedData: "00000000000000000000000060beef336a37618bf66373361ace074b0b34bc0b00000000000000000000000080e526c863632a31ac0787cf7cef32c0adcddfe00000000000000000000000000d9a92ffd756d8587541614800c06feb7ae2ee1a0000000000000000000000000000000000000000000000000000000000002710000000000000000000000000428d69ccc61c8c15993bf6925172dc85047a6abf",
	},
}

func Test_packGMX(t *testing.T) {
	t.Parallel()

	for idx, pair := range packGMXPairs {
		t.Run(fmt.Sprintf("it should pack correctly %d", idx), func(t *testing.T) {
			result, err := packGMX(pair.data)

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.packedData, common.Bytes2Hex(result))
		})
	}
}

func TestUnpackGMX(t *testing.T) {
	t.Parallel()

	for idx, pair := range packGMXPairs {
		t.Run(fmt.Sprintf("it should decode correctly %d", idx), func(t *testing.T) {
			result, err := UnpackGMX(common.Hex2Bytes(pair.packedData))

			assert.ErrorIs(t, err, nil)
			assert.EqualValues(t, pair.data, result)
		})
	}
}
