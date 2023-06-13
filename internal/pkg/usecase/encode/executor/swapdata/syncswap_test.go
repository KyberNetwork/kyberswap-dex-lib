package swapdata

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
)

var packSyncSwapPairs = []struct {
	swap               types.EncodingSwap
	data               SyncSwap
	packedData         string
	packedSyncSwapData string
}{
	{
		swap: types.EncodingSwap{
			TokenIn:   "0x5aea5775959fbc2557cc8789bc1bf90a239d9a91",
			Recipient: "0x1111111111111111111111111111111111111100",
		},
		data: SyncSwap{
			Data: common.Hex2Bytes("0000000000000000000000005aea5775959fbc2557cc8789bc1bf90a239d9a91" +
				"0000000000000000000000001111111111111111111111111111111111111100" +
				"0000000000000000000000000000000000000000000000000000000000000000"),
			TokenIn:       common.HexToAddress("0x5aea5775959fbc2557cc8789bc1bf90a239d9a91"),
			Pool:          common.HexToAddress("0x323415fff51c2660348f27c2047a50834ad67ad5"),
			CollectAmount: big.NewInt(6000),
		},
		packedData: "0000000000000000000000000000000000000000000000000000000000000020" +
			"0000000000000000000000000000000000000000000000000000000000000080" +
			"0000000000000000000000005aea5775959fbc2557cc8789bc1bf90a239d9a91" +
			"000000000000000000000000323415fff51c2660348f27c2047a50834ad67ad5" +
			"0000000000000000000000000000000000000000000000000000000000001770" +
			"0000000000000000000000000000000000000000000000000000000000000060" +
			"0000000000000000000000005aea5775959fbc2557cc8789bc1bf90a239d9a91" +
			"0000000000000000000000001111111111111111111111111111111111111100" +
			"0000000000000000000000000000000000000000000000000000000000000000",
		packedSyncSwapData: "0000000000000000000000005aea5775959fbc2557cc8789bc1bf90a239d9a91" +
			"0000000000000000000000001111111111111111111111111111111111111100" +
			"0000000000000000000000000000000000000000000000000000000000000000",
	},
}

func TestBuildSyncSwap(t *testing.T) {
	t.Parallel()

	for idx, pair := range packSyncSwapPairs {
		t.Run(fmt.Sprintf("it should build correctly %d", idx), func(t *testing.T) {
			result, err := buildSyncSwap(pair.swap)

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.packedSyncSwapData, common.Bytes2Hex(result.Data))
		})
	}

}

func Test_packSyncSwap(t *testing.T) {
	t.Parallel()

	for idx, pair := range packSyncSwapPairs {
		t.Run(fmt.Sprintf("it should pack correctly %d", idx), func(t *testing.T) {
			result, err := packSyncSwap(pair.data)

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.packedData, common.Bytes2Hex(result))
		})
	}
}

func TestUnpackSyncSwap(t *testing.T) {
	t.Parallel()

	for idx, pair := range packSyncSwapPairs {
		t.Run(fmt.Sprintf("it should decode correctly %d", idx), func(t *testing.T) {
			result, err := UnpackSyncSwap(common.Hex2Bytes(pair.packedData))

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.data, result)
		})
	}
}
