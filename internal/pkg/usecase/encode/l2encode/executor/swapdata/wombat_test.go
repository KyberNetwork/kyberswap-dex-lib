package swapdata

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var packWombatPairs = []struct {
	data       Wombat
	packedData string
}{
	{
		data: Wombat{
			Pool:          common.HexToAddress("0xf8e32ca46ac28799c8fb7dce1ac11a4541160734"),
			TokenOut:      common.HexToAddress("0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0"),
			Recipient:     common.HexToAddress("0xdeea7249f436cfdb360a8b6725aca01c604735b1"),
			Amount:        big.NewInt(3792000000956209),
			isFirstSwap:   true,
			recipientFlag: 0,
		},
		packedData: "000000f8e32ca46ac28799c8fb7dce1ac11a45411607347f39c581f595b53c5cb19bd0b3f8da6c935e2ca00000000000000000000d78cdcd0b973100deea7249f436cfdb360a8b6725aca01c604735b1",
	},
}

func TestPackWombat(t *testing.T) {
	t.Parallel()

	for idx, pair := range packWombatPairs {
		t.Run(fmt.Sprintf("it should pack correctly %d", idx), func(t *testing.T) {
			result, err := packWombat(pair.data)

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.packedData, common.Bytes2Hex(result))
		})
	}
}

func TestUnpackWombat(t *testing.T) {
	t.Parallel()

	for idx, pair := range packWombatPairs {
		t.Run(fmt.Sprintf("it should decode correctly %d", idx), func(t *testing.T) {
			result, err := UnpackWombat(
				common.Hex2Bytes(pair.packedData),
				pair.data.isFirstSwap,
			)

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.data, result)
		})
	}
}
