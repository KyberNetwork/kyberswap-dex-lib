package swapdata

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var packSTETHPairs = []struct {
	data       *big.Int
	packedData string
}{
	{
		data:       big.NewInt(1001),
		packedData: "00000000000000000000000000000000000000000000000000000000000003e9",
	},
	{
		data:       new(big.Int).Exp(big.NewInt(2), big.NewInt(60), nil),
		packedData: "0000000000000000000000000000000000000000000000001000000000000000",
	},
}

func Test_packSTETH(t *testing.T) {
	t.Parallel()

	for idx, pair := range packSTETHPairs {
		t.Run(fmt.Sprintf("it should pack correctly %d", idx), func(t *testing.T) {
			result, err := PackStETH(0, types.EncodingSwap{SwapAmount: pair.data})

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.packedData, common.Bytes2Hex(result))
		})
	}
}

func TestUnpackSTETH(t *testing.T) {
	t.Parallel()

	for idx, pair := range packSTETHPairs {
		t.Run(fmt.Sprintf("it should decode correctly %d", idx), func(t *testing.T) {
			result, err := UnpackStETH(common.Hex2Bytes(pair.packedData))

			assert.ErrorIs(t, err, nil)
			assert.EqualValues(t, pair.data, result)
		})
	}
}
