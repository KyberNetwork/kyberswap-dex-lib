package swapdata

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var packMaverickV1SwapPairs = []struct {
	data       MaverickV1Swap
	packedData string
}{
	{
		data: MaverickV1Swap{
			Pool:              common.HexToAddress("0xedf1335a6f016d7e2c0d80082688cd582e48a6fc"),
			TokenIn:           common.HexToAddress("0x5faa989af96af85384b8a938c2ede4a7378d9875"),
			TokenOut:          common.HexToAddress("0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0"),
			Recipient:         common.HexToAddress("0x3fd899eaf2dda35cf2c7bfcdb27a23d727d9a67c"),
			SwapAmount:        big.NewInt(100000),
			SqrtPriceLimitD18: big.NewInt(0),
		},
		packedData: "000000000000000000000000edf1335a6f016d7e2c0d80082688cd582e48a6fc0000000000000000000000005faa989af96af85384b8a938c2ede4a7378d98750000000000000000000000007f39c581f595b53c5cb19bd0b3f8da6c935e2ca00000000000000000000000003fd899eaf2dda35cf2c7bfcdb27a23d727d9a67c00000000000000000000000000000000000000000000000000000000000186a00000000000000000000000000000000000000000000000000000000000000000",
	},
}

func TestPackMaverickV1(t *testing.T) {
	t.Parallel()

	for idx, pair := range packMaverickV1SwapPairs {
		t.Run(fmt.Sprintf("it should pack correctly %d", idx), func(t *testing.T) {
			result, err := packMaverickV1(pair.data)

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.packedData, common.Bytes2Hex(result))
		})
	}
}

func TestUnpackMaverickV1(t *testing.T) {
	t.Parallel()

	for idx, pair := range packMaverickV1SwapPairs {
		t.Run(fmt.Sprintf("it should decode correctly %d", idx), func(t *testing.T) {
			result, err := UnpackMaverickV1(common.Hex2Bytes(pair.packedData))

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.data.Pool, result.Pool)
			assert.Equal(t, pair.data.TokenIn, result.TokenIn)
			assert.Equal(t, pair.data.TokenOut, result.TokenOut)
			assert.Equal(t, pair.data.Recipient, result.Recipient)
			assert.Equal(t, pair.data.SwapAmount.String(), result.SwapAmount.String())
			assert.Equal(t, pair.data.SqrtPriceLimitD18.String(), result.SqrtPriceLimitD18.String())
		})
	}
}
