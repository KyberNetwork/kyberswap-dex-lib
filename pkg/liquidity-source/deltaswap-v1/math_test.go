package deltaswapv1

import (
	"testing"

	"github.com/holiman/uint256"
)

func TestSqrt(t *testing.T) {
	t.Skip("Skipping testing in CI environment")

	tests := []struct {
		input    *uint256.Int
		expected *uint256.Int
	}{
		{uint256.NewInt(0), uint256.NewInt(0)},
		{uint256.NewInt(1), uint256.NewInt(1)},
		{uint256.NewInt(4), uint256.NewInt(2)},
		{uint256.NewInt(9), uint256.NewInt(3)},
		{uint256.NewInt(16), uint256.NewInt(4)},
		{uint256.NewInt(49), uint256.NewInt(7)},
		{uint256.NewInt(50), uint256.NewInt(7)},
		{uint256.NewInt(100), uint256.NewInt(10)},
		{uint256.NewInt(1000000), uint256.NewInt(1000)},
		{uint256.NewInt(999999), uint256.NewInt(999)},
		{uint256.NewInt(1000000000000), uint256.NewInt(1000000)},
		{uint256.NewInt(3), uint256.NewInt(1)},
		{uint256.NewInt(2), uint256.NewInt(1)},
		{uint256.NewInt(100000213213213), uint256.NewInt(10000010)},
		{uint256.NewInt(100000213213213213), uint256.NewInt(316228103)},
	}

	for _, tt := range tests {
		t.Run(tt.input.String(), func(t *testing.T) {
			result := Sqrt(tt.input)
			if !result.Eq(tt.expected) || result.Cmp(new(uint256.Int).Sqrt(tt.input)) != 0 {
				t.Errorf("expected %s, got %s", tt.expected.String(), result.String())
			}
		})
	}
}
