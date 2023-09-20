package swapdata

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var packIZiSwapPairs = []struct {
	data       IZiSwap
	packedData string
}{
	{
		data: IZiSwap{
			Pool:       common.HexToAddress("0x1ce3082de766ebfe1b4db39f616426631bbb29ac"),
			TokenIn:    common.HexToAddress("0x55d398326f99059ff775485246999027b3197955"),
			TokenOut:   common.HexToAddress("0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c"),
			Recipient:  common.HexToAddress("0xdeea7249f436cfdb360a8b6725aca01c604735b1"),
			SwapAmount: big.NewInt(10000000),
			LimitPoint: big.NewInt(-51890),
		},
		packedData: "0000000000000000000000001ce3082de766ebfe1b4db39f616426631bbb29ac00000000000000000000000055d398326f99059ff775485246999027b3197955000000000000000000000000bb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c000000000000000000000000deea7249f436cfdb360a8b6725aca01c604735b10000000000000000000000000000000000000000000000000000000000989680ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff354e",
	},
}

func TestPackIZiSwap(t *testing.T) {
	t.Parallel()

	for idx, pair := range packIZiSwapPairs {
		t.Run(fmt.Sprintf("it should pack correctly %d", idx), func(t *testing.T) {
			result, err := packIZiSwap(pair.data)

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.packedData, common.Bytes2Hex(result))
		})
	}
}

func TestUnpackIZiSwap(t *testing.T) {
	t.Parallel()

	for idx, pair := range packIZiSwapPairs {
		t.Run(fmt.Sprintf("it should decode correctly %d", idx), func(t *testing.T) {
			result, err := UnpackIZiSwap(common.Hex2Bytes(pair.packedData))

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.data, result)
		})
	}
}
