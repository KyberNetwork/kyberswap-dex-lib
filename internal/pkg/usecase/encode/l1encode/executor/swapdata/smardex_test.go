package swapdata

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var packSmardexPairs = []struct {
	data       Smardex
	packedData string
}{
	{
		data: Smardex{
			Pool:      common.HexToAddress("0xd4026c954d7d8d27fb72677c78e388c6969258c7"),
			TokenIn:   common.HexToAddress("0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"),
			TokenOut:  common.HexToAddress("0x4af15ec2a0bd43db75dd04e62faa3b8ef36b00d5"),
			Amount:    big.NewInt(1000000),
			Recipient: common.HexToAddress("0x85db3da60fec7d3d821bef7a95796578ded9f7bc"),
		},
		packedData: "000000000000000000000000d4026c954d7d8d27fb72677c78e388c6969258c7" +
			"000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2" +
			"0000000000000000000000004af15ec2a0bd43db75dd04e62faa3b8ef36b00d5" +
			"00000000000000000000000000000000000000000000000000000000000f4240" +
			"00000000000000000000000085db3da60fec7d3d821bef7a95796578ded9f7bc",
	},
}

func TestPackSmardex(t *testing.T) {
	t.Parallel()

	for idx, pair := range packSmardexPairs {
		t.Run(fmt.Sprintf("it should pack correctly %d", idx), func(t *testing.T) {
			result, err := packSmardex(pair.data)

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.packedData, common.Bytes2Hex(result))
		})
	}
}

func TestUnpackSmardex(t *testing.T) {
	t.Parallel()

	for idx, pair := range packSmardexPairs {
		t.Run(fmt.Sprintf("it should decode correctly %d", idx), func(t *testing.T) {
			result, err := UnpackSmardex(common.Hex2Bytes(pair.packedData))

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.data, result)
		})
	}
}
