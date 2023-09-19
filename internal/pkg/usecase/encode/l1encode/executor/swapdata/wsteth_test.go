package swapdata

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var packWSTETHPairs = []struct {
	data       WSTETH
	packedData string
}{
	{
		data: WSTETH{
			Pool:       common.HexToAddress("0x60beef336a37618bf66373361ace074b0b34bc0b"),
			Amount:     big.NewInt(2),
			IsWrapping: true,
		},
		packedData: "00000000000000000000000060beef336a37618bf66373361ace074b0b34bc0b00000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000001",
	},
}

func Test_packWSTETH(t *testing.T) {
	t.Parallel()

	for idx, pair := range packWSTETHPairs {
		t.Run(fmt.Sprintf("it should pack correctly %d", idx), func(t *testing.T) {
			result, err := packWSTETH(pair.data)

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.packedData, common.Bytes2Hex(result))
		})
	}
}

func TestUnpackWSTETH(t *testing.T) {
	t.Parallel()

	for idx, pair := range packWSTETHPairs {
		t.Run(fmt.Sprintf("it should decode correctly %d", idx), func(t *testing.T) {
			result, err := UnpackWSTETH(common.Hex2Bytes(pair.packedData))

			assert.ErrorIs(t, err, nil)
			assert.EqualValues(t, pair.data, result)
		})
	}
}
