package libv2

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

func TestSolveQuadraticFunctionForTarget(t *testing.T) {
	t.Parallel()
	type args struct {
		V1    *uint256.Int
		delta *uint256.Int
		i     *uint256.Int
		k     *uint256.Int
	}
	tests := []struct {
		name string
		args args
		want *uint256.Int
	}{
		{
			name: "case 1",
			args: args{
				V1:    uint256.NewInt(5293182),
				delta: uint256.NewInt(20407),
				i:     uint256.NewInt(92184012410),
				k:     uint256.NewInt(300000000000000000),
			},
			want: uint256.NewInt(5293182),
		},
		{
			name: "case 2",
			args: args{
				V1:    uint256.NewInt(432422342),
				delta: uint256.NewInt(32131),
				i:     uint256.NewInt(48930284098),
				k:     uint256.NewInt(300000000000000000),
			},
			want: uint256.NewInt(432422342),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SolveQuadraticFunctionForTarget(tt.args.V1, tt.args.delta, tt.args.i, tt.args.k)
			assert.Equalf(t, tt.want, got, "SolveQuadraticFunctionForTarget() got = %v, want %v", got, tt.want)
		})
	}
}

func TestSolveQuadraticFunctionForTrade(t *testing.T) {
	t.Parallel()
	type args struct {
		V0    *uint256.Int
		V1    *uint256.Int
		delta *uint256.Int
		i     *uint256.Int
		k     *uint256.Int
	}
	tests := []struct {
		name string
		args args
		want *uint256.Int
	}{
		{
			name: "case 1",
			args: args{
				V0:    uint256.NewInt(10388770142),
				V1:    uint256.NewInt(10388770142),
				delta: uint256.NewInt(664906786),
				i:     uint256.NewInt(92184012410),
				k:     uint256.NewInt(300000000000000000),
			},
			want: uint256.NewInt(62),
		},
		{
			name: "case 2",
			args: args{
				V0:    uint256.NewInt(31232114321321),
				V1:    uint256.NewInt(45433543332131),
				delta: uint256.NewInt(3213211123),
				i:     uint256.NewInt(43543636664323),
				k:     uint256.NewInt(300000000000000000),
			},
			want: uint256.NewInt(166217),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SolveQuadraticFunctionForTrade(tt.args.V0, tt.args.V1, tt.args.delta, tt.args.i, tt.args.k)
			assert.Equalf(t, tt.want, got, "SolveQuadraticFunctionForTrade() got = %v, want %v", got, tt.want)
		})
	}
}
