package swapdata

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var packDODOPairs = []struct {
	data       DODO
	packedData string
}{
	{
		data: DODO{
			Recipient:     common.HexToAddress("0x671276fee1ee7355fe3b5e5e033ebc11c12f2934"),
			Pool:          common.HexToAddress("0x254e21a226ff58f5a0d99919e2cc75e66dc86d6a"),
			Amount:        big.NewInt(100000),
			SellHelper:    common.HexToAddress(""),
			IsSellBase:    true,
			IsVersion2:    true,
			isFirstSwap:   true,
			recipientFlag: 0,
		},
		packedData: "00671276fee1ee7355fe3b5e5e033ebc11c12f2934000000254e21a226ff58f5a0d99919e2cc75e66dc86d6a000000000000000000000000000186a000000000000000000000000000000000000000000101",
	},
}

func TestPackDODO(t *testing.T) {
	t.Parallel()

	for idx, pair := range packDODOPairs {
		t.Run(fmt.Sprintf("it should pack correctly %d", idx), func(t *testing.T) {
			result, err := packDODO(pair.data)

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.packedData, common.Bytes2Hex(result))
		})
	}
}

func TestUnpackDODO(t *testing.T) {
	t.Parallel()

	for idx, pair := range packDODOPairs {
		t.Run(fmt.Sprintf("it should decode correctly %d", idx), func(t *testing.T) {
			result, err := UnpackDODO(
				common.Hex2Bytes(pair.packedData),
				pair.data.isFirstSwap,
			)

			assert.ErrorIs(t, err, nil)
			assert.EqualValues(t, pair.data, result)
		})
	}
}
