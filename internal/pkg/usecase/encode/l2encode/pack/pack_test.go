package pack

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestCompressPack(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		input  []interface{}
		result string
		err    error
	}{
		{
			name: "it should pack correctly in simple case",
			input: []interface{}{
				big.NewInt(256),
				common.HexToAddress("0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"),
				[]uint8{1, 2, 3},
			},
			result: "00000000000000000000000000000100eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee00000003010203",
			err:    nil,
		},
		{
			name: "it should pack with custom types",
			input: []interface{}{
				UInt24(27),
				UInt160(big.NewInt(4)),
			},
			result: "00001b0000000000000000000000000000000000000004",
			err:    nil,
		},
		{
			name: "it should keep raw bytes vs slice of bytes",
			input: []interface{}{
				RawBytes([]byte{2}),
				[]byte{7},
			},
			result: "020000000107",
			err:    nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Pack(tc.input...)
			assert.Equal(t, tc.result, hex.EncodeToString(result))
			assert.ErrorIs(t, err, tc.err)
		})
	}
}
