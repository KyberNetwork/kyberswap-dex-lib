package libv1

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

func Test__SolveQuadraticFunctionForTrade(t *testing.T) {
	t.Parallel()
	type args struct {
		Q0        *uint256.Int
		Q1        *uint256.Int
		ideltaB   *uint256.Int
		deltaBSig bool
		k         *uint256.Int
	}
	tests := []struct {
		name string
		args args
		want *uint256.Int
	}{
		{
			name: "case 1",
			args: args{
				Q0:        uint256.NewInt(10388770142),
				Q1:        uint256.NewInt(10388770142),
				ideltaB:   uint256.NewInt(664906786),
				deltaBSig: false,
				k:         uint256.NewInt(300000000000000000),
			},
			want: uint256.NewInt(9736953633),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SolveQuadraticFunctionForTrade(tt.args.Q0, tt.args.Q1, tt.args.ideltaB, tt.args.deltaBSig, tt.args.k)
			assert.Equalf(t, tt.want, got, "SolveQuadraticFunctionForTrade got = %v, want %v", got, tt.want)
		})
	}
}

func Test__SolveQuadraticFunctionForTarget(t *testing.T) {
	t.Parallel()
	type args struct {
		V1         *uint256.Int
		k          *uint256.Int
		fairAmount *uint256.Int
	}
	tests := []struct {
		name string
		args args
		want *uint256.Int
	}{
		{
			name: "case 1",
			args: args{
				V1:         uint256.NewInt(5293182),
				k:          uint256.NewInt(300000000000000000),
				fairAmount: uint256.NewInt(20407),
			},
			want: uint256.NewInt(5313565),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SolveQuadraticFunctionForTarget(tt.args.V1, tt.args.k, tt.args.fairAmount)
			assert.Equalf(t, tt.want, got, "SolveQuadraticFunctionForTarget() got = %v, want %v", got, tt.want)
		})
	}
}
