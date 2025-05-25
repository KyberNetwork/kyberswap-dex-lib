package hooks

import (
	"fmt"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

func TestNewStableSurgeHook(t *testing.T) {
	type args struct {
		maxSurgeFeePercentage *uint256.Int
		thresholdPercentage   *uint256.Int
	}
	tests := []struct {
		name    string
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			"valid",
			args{
				maxSurgeFeePercentage: uint256.NewInt(20000000000000000),
				thresholdPercentage:   uint256.NewInt(20000000000000000),
			},
			assert.NoError,
		},
		{
			"invalid maxSurgeFeePercentage",
			args{
				maxSurgeFeePercentage: uint256.NewInt(950000000000000000),
				thresholdPercentage:   uint256.NewInt(300000000000000000),
			},
			assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewStableSurgeHook(tt.args.maxSurgeFeePercentage, tt.args.thresholdPercentage)
			tt.wantErr(t, err, fmt.Sprintf("NewStableSurgeHook(%v, %v)",
				tt.args.maxSurgeFeePercentage, tt.args.thresholdPercentage))
		})
	}
}
