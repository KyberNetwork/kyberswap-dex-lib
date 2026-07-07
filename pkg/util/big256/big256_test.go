package big256

import (
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
