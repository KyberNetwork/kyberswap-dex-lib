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
			Recipient:       common.HexToAddress("0x671276fee1ee7355fe3b5e5e033ebc11c12f2934"),
			Pool:            common.HexToAddress("0x254e21a226ff58f5a0d99919e2cc75e66dc86d6a"),
			TokenFrom:       common.HexToAddress("0x164c24c62b91ec79a9be0cfb9004837536a6ec58"),
			TokenTo:         common.HexToAddress("0x318f2d17461063778ffc877a5f6ac5b4e062a3ec"),
			Amount:          big.NewInt(100000),
			MinReceiveQuote: big.NewInt(12445),
			SellHelper:      common.HexToAddress(""),
			IsSellBase:      true,
			IsVersion2:      true,
		},
		packedData: "000000000000000000000000671276fee1ee7355fe3b5e5e033ebc11c12f2934000000000000000000000000254e21a226ff58f5a0d99919e2cc75e66dc86d6a000000000000000000000000164c24c62b91ec79a9be0cfb9004837536a6ec58000000000000000000000000318f2d17461063778ffc877a5f6ac5b4e062a3ec00000000000000000000000000000000000000000000000000000000000186a0000000000000000000000000000000000000000000000000000000000000309d000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000001",
	},
}

func Test_packDODO(t *testing.T) {
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
			result, err := UnpackDODO(common.Hex2Bytes(pair.packedData))

			assert.ErrorIs(t, err, nil)
			assert.EqualValues(t, pair.data, result)
		})
	}
}
