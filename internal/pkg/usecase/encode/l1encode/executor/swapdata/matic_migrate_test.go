package swapdata

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var packMaticMigratePairs = []struct {
	data       MaticMigrate
	packedData string
}{
	{
		data: MaticMigrate{
			Pool:      common.HexToAddress("0x550b7cdac6f5a0d9e840505c3df74ac045530446"),
			TokenIn:   common.HexToAddress("0x7d1afa7b718fb893db30a3abc0cfc608aacfebb0"),
			TokenOut:  common.HexToAddress("0x455e53cbb86018ac2b8092fdcd39d8444affc3f6"),
			Amount:    big.NewInt(1000000),
			Recipient: common.HexToAddress("0x9206ccef3362a31f97fbca8bc21407bd00eddbb4"),
			IsMigrate: true,
		},
		packedData: "000000000000000000000000550b7cdac6f5a0d9e840505c3df74ac045530446" +
			"0000000000000000000000007d1afa7b718fb893db30a3abc0cfc608aacfebb0" +
			"000000000000000000000000455e53cbb86018ac2b8092fdcd39d8444affc3f6" +
			"00000000000000000000000000000000000000000000000000000000000f4240" +
			"0000000000000000000000009206ccef3362a31f97fbca8bc21407bd00eddbb4" +
			"0000000000000000000000000000000000000000000000000000000000000001",
	},
}

func TestPackMaticMigrate(t *testing.T) {
	t.Parallel()

	for idx, pair := range packMaticMigratePairs {
		t.Run(fmt.Sprintf("it should pack correctly %d", idx), func(t *testing.T) {
			result, err := packMaticMigrate(pair.data)

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.packedData, common.Bytes2Hex(result))
		})
	}
}

func TestUnpackMaticMigrate(t *testing.T) {
	t.Parallel()

	for idx, pair := range packMaticMigratePairs {
		t.Run(fmt.Sprintf("it should decode correctly %d", idx), func(t *testing.T) {
			result, err := UnpackMaticMigrate(common.Hex2Bytes(pair.packedData))

			assert.ErrorIs(t, err, nil)
			assert.Equal(t, pair.data, result)
		})
	}
}
