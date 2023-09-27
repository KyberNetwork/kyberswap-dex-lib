package swapdata

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/l2encode/pack"
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
			TokenOut:   common.HexToAddress("0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c"),
			Recipient:  common.HexToAddress("0xdeea7249f436cfdb360a8b6725aca01c604735b1"),
			SwapAmount: big.NewInt(10000000),
			LimitPoint: pack.Int24(51890), // Positive LimitPoint

			isFirstSwap: true,
		},
		packedData: "0000001ce3082de766ebfe1b4db39f616426631bbb29acbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c00deea7249f436cfdb360a8b6725aca01c604735b10000000000000000000000000098968000cab2",
	},
	{
		data: IZiSwap{
			Pool:       common.HexToAddress("0x1ce3082de766ebfe1b4db39f616426631bbb29ac"),
			TokenOut:   common.HexToAddress("0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c"),
			Recipient:  common.HexToAddress("0xdeea7249f436cfdb360a8b6725aca01c604735b1"),
			SwapAmount: big.NewInt(10000000),
			LimitPoint: pack.Int24(-51890), // Negative LimitPoint

			isFirstSwap: true,
		},
		packedData: "0000001ce3082de766ebfe1b4db39f616426631bbb29acbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c00deea7249f436cfdb360a8b6725aca01c604735b100000000000000000000000000989680ff354e",
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
			result, err := UnpackIZiSwap(
				common.Hex2Bytes(pair.packedData),
				pair.data.isFirstSwap,
			)

			assert.ErrorIs(t, err, nil)
			assert.EqualValues(t, pair.data, result)
		})
	}
}
