package big256

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

func TestNewI(t *testing.T) {
	type args struct {
		i int64
	}
	tests := []struct {
		name string
		args args
		want *uint256.Int
	}{
		{
			"positive",
			args{
				i: 1,
			},
			uint256.NewInt(1),
		},
		{
			"negative",
			args{
				i: -1,
			},
			new(uint256.Int).SubUint64(U0, 1),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewI(tt.args.i)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestToBigBacked: quote results hand these big.Ints to callers, so each must equal ToBig for every
// limb count and own its words.
func TestToBigBacked(t *testing.T) {
	for _, s := range []string{"0", "1", "18446744073709551615", "18446744073709551616",
		"340282366920938463463374607431768211457", "6277101735386680763835789423207666416102355444464034512896",
		"115792089237316195423570985008687907853269984665640564039457584007913129639935"} {
		u := uint256.MustFromDecimal(s)
		var z big.Int
		var words [4]big.Word
		got := ToBigBacked(&z, &words, u)
		assert.Equal(t, 0, got.Cmp(u.ToBig()), s)
		u.Clear()
		assert.Equal(t, s, got.String(), "the result must not alias its source")
	}
}
