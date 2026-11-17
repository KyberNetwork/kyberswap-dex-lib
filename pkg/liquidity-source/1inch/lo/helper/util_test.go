//nolint:testpackage
package helper

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBitMask(t *testing.T) {
	tests := []struct {
		start  uint
		end    uint
		expect *big.Int
	}{
		{
			start:  0,
			end:    10,
			expect: big.NewInt(0b1111111111),
		},
		{
			start:  5,
			end:    17,
			expect: big.NewInt(0b11111111111100000),
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.expect, newBitMask(test.start, test.end))
	}
}

func TestSetMask(t *testing.T) {
	tests := []struct {
		n      *big.Int
		mask   *big.Int
		value  *big.Int
		expect *big.Int
	}{
		{
			n:      big.NewInt(0b1111111111111110101111111111100111),
			mask:   newBitMask(0, 10),
			value:  big.NewInt(0b0011110101),
			expect: big.NewInt(0b1111111111111110101111110011110101),
		},
		{
			n:      big.NewInt(0b1111111111111110101111111111100111),
			mask:   newBitMask(0, 10),
			value:  big.NewInt(0b11110011110101),
			expect: big.NewInt(0b1111111111111110101111110011110101),
		},
		{
			n:      big.NewInt(0b1111111111111110101111111111100111),
			mask:   newBitMask(5, 15),
			value:  big.NewInt(0b0011110101),
			expect: big.NewInt(0b1111111111111110101001111010100111),
		},
		{
			n:      big.NewInt(0b1111111111111110101111111111100111),
			mask:   newBitMask(5, 15),
			value:  big.NewInt(0b11110011110101),
			expect: big.NewInt(0b1111111111111110101001111010100111),
		},
	}

	for _, test := range tests {
		setMask(test.n, test.mask, test.value)
		assert.Equal(t, test.expect, test.n, "expect: %s, actual: %s", test.expect.Text(2), test.n.Text(2))
	}
}
