package uniswapv2

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

var (
	MaxUInt256 = new(uint256.Int).Sub(new(uint256.Int).Lsh(uint256.NewInt(1), 256), uint256.NewInt(1))
)

func TestSafeAdd(t *testing.T) {
	t.Parallel()
	type args struct {
		x *uint256.Int
		y *uint256.Int
	}
	tests := []struct {
		name        string
		args        args
		want        *uint256.Int
		shouldPanic bool
	}{
		{
			name: "normal case",
			args: args{
				x: uint256.NewInt(10),
				y: uint256.NewInt(5),
			},
			want: uint256.NewInt(15),
		},
		{
			name: "overflow case",
			args: args{
				x: new(uint256.Int).Set(MaxUInt256),
				y: uint256.NewInt(10),
			},
			want:        nil,
			shouldPanic: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldPanic {
				assert.PanicsWithError(t, ErrDSMathAddOverflow.Error(), func() {
					SafeAdd(tt.args.x, tt.args.y)
				})
			} else {
				got := SafeAdd(tt.args.x, tt.args.y)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestSafeSub(t *testing.T) {
	t.Parallel()
	type args struct {
		x *uint256.Int
		y *uint256.Int
	}
	tests := []struct {
		name        string
		args        args
		want        *uint256.Int
		shouldPanic bool
	}{
		{
			name: "normal case",
			args: args{
				x: uint256.NewInt(10),
				y: uint256.NewInt(5),
			},
			want: uint256.NewInt(5),
		},
		{
			name: "underflow case",
			args: args{
				x: uint256.NewInt(10),
				y: uint256.NewInt(15),
			},
			want:        nil,
			shouldPanic: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldPanic {
				assert.PanicsWithError(t, ErrDSMathSubUnderflow.Error(), func() {
					SafeSub(tt.args.x, tt.args.y)
				})
			} else {
				got := SafeSub(tt.args.x, tt.args.y)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestSafeMul(t *testing.T) {
	t.Parallel()
	type args struct {
		x *uint256.Int
		y *uint256.Int
	}
	tests := []struct {
		name        string
		args        args
		want        *uint256.Int
		shouldPanic bool
	}{
		{
			name: "normal case",
			args: args{
				x: uint256.NewInt(10),
				y: uint256.NewInt(5),
			},
			want: uint256.NewInt(50),
		},
		{
			name: "overflow case",
			args: args{
				x: new(uint256.Int).Set(MaxUInt256),
				y: uint256.NewInt(10),
			},
			want:        nil,
			shouldPanic: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldPanic {
				assert.PanicsWithError(t, ErrDSMathMulOverflow.Error(), func() {
					SafeMul(tt.args.x, tt.args.y)
				})
			} else {
				got := SafeMul(tt.args.x, tt.args.y)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
