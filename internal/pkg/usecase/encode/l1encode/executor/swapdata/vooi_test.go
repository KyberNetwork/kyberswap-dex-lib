package swapdata

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var packVooiPairs = []struct {
	data       Vooi
	packedData string
}{
	{
		data: Vooi{
			Pool:       common.HexToAddress("0xbc7f67fa9c72f9fccf917cbcee2a50deb031462a"),
			FromToken:  common.HexToAddress("0x176211869ca2b568f2a7d4ee941e073a821ee1ff"),
			ToToken:    common.HexToAddress("0x4af15ec2a0bd43db75dd04e62faa3b8ef36b00d5"),
			FromID:     big.NewInt(1),
			ToID:       big.NewInt(2),
			FromAmount: big.NewInt(1000000),
			To:         common.HexToAddress("0x9206ccEf3362A31f97FbCa8bc21407bD00edDbb4"),
		},
		packedData: "000000000000000000000000bc7f67fa9c72f9fccf917cbcee2a50deb031462a" +
			"000000000000000000000000176211869ca2b568f2a7d4ee941e073a821ee1ff" +
			"0000000000000000000000004af15ec2a0bd43db75dd04e62faa3b8ef36b00d5" +
			"0000000000000000000000000000000000000000000000000000000000000001" +
			"0000000000000000000000000000000000000000000000000000000000000002" +
			"00000000000000000000000000000000000000000000000000000000000f4240" +
			"0000000000000000000000009206ccef3362a31f97fbca8bc21407bd00eddbb4",
	},
}

func TestPackVooi(t *testing.T) {
	t.Parallel()

	for idx, pair := range packVooiPairs {
		t.Run(fmt.Sprintf("it should pack correctly %d", idx), func(t *testing.T) {
			result, err := packVooi(pair.data)

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.packedData, common.Bytes2Hex(result))
		})
	}
}

func TestUnpackVooi(t *testing.T) {
	t.Parallel()

	for idx, pair := range packVooiPairs {
		t.Run(fmt.Sprintf("it should decode correctly %d", idx), func(t *testing.T) {
			result, err := UnpackVooi(common.Hex2Bytes(pair.packedData))

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.data, result)
		})
	}
}
