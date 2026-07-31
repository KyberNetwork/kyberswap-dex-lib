package shared

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertToAssets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		shares      *big.Int
		totalAssets *big.Int
		totalSupply *big.Int
		want        *big.Int
	}{
		{
			// Controller-only vault appended by v2's pool_tracker: EulerAccountBalance
			// is set to 0 but totalAssets/totalSupply are never fetched (nil). Must not
			// panic — this is the exact prod SIGSEGV at utils.go:16.
			name:        "nil totals with zero shares -> 0 (controller-only vault)",
			shares:      big.NewInt(0),
			totalAssets: nil,
			totalSupply: nil,
			want:        big.NewInt(0),
		},
		{
			name:        "nil shares -> 0",
			shares:      nil,
			totalAssets: big.NewInt(100),
			totalSupply: big.NewInt(100),
			want:        big.NewInt(0),
		},
		{
			name:        "nil totalSupply -> shares 1:1",
			shares:      big.NewInt(42),
			totalAssets: big.NewInt(100),
			totalSupply: nil,
			want:        big.NewInt(42),
		},
		{
			name:        "nil totalAssets -> shares 1:1",
			shares:      big.NewInt(42),
			totalAssets: nil,
			totalSupply: big.NewInt(100),
			want:        big.NewInt(42),
		},
		{
			name:        "zero supply -> shares 1:1",
			shares:      big.NewInt(42),
			totalAssets: big.NewInt(0),
			totalSupply: big.NewInt(0),
			want:        big.NewInt(42),
		},
		{
			// (shares * (totalAssets + VirtualAmount)) / (totalSupply + VirtualAmount)
			// VirtualAmount = 1e6: (2 * (2e6 + 1e6)) / (1e6 + 1e6) = 6e6 / 2e6 = 3
			name:        "normal conversion",
			shares:      big.NewInt(2),
			totalAssets: big.NewInt(2_000_000),
			totalSupply: big.NewInt(1_000_000),
			want:        big.NewInt(3),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.NotPanics(t, func() {
				got := ConvertToAssets(tt.shares, tt.totalAssets, tt.totalSupply)
				assert.Zero(t, got.Cmp(tt.want), "got %s want %s", got, tt.want)
			})
		})
	}
}
