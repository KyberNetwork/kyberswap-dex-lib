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
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.NotPanics(t, func() {
				got := ConvertToAssets(tt.shares, tt.totalAssets, tt.totalSupply)
				assert.Zero(t, got.Cmp(tt.want), "got %s want %s", got, tt.want)
			})
		})
	}
}

func TestUnitPrice(t *testing.T) {
	t.Parallel()

	e := func(n int64) *big.Int { return new(big.Int).Exp(big.NewInt(10), big.NewInt(n), nil) }
	mul := func(a, b *big.Int) *big.Int { return new(big.Int).Mul(a, b) }
	tests := []struct {
		name               string
		quote, quoteAmount *big.Int
		want               *big.Int
	}{
		{
			// BSC USDT (18 decimals) at $0.99848 in an 18-decimal USD unit of account: quote / 1e18
			// rounds to 0 without the scale, and the solvency check then values the debt at 0.
			name:        "18-decimal token under $1",
			quote:       big.NewInt(998480000000000000),
			quoteAmount: e(18),
			want:        big.NewInt(998480000000000000),
		},
		{
			name:        "18-decimal token at $2.5 keeps its fraction",
			quote:       mul(big.NewInt(25), e(17)),
			quoteAmount: e(18),
			want:        mul(big.NewInt(25), e(17)),
		},
		{
			name:        "6-decimal token at $1",
			quote:       e(18),
			quoteAmount: e(6),
			want:        e(30),
		},
		{
			name:        "nil quote (failed call) -> 0",
			quote:       nil,
			quoteAmount: e(18),
			want:        big.NewInt(0),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, 0, UnitPrice(tt.quote, tt.quoteAmount).Cmp(tt.want), "got %s", UnitPrice(tt.quote, tt.quoteAmount))
		})
	}
}

func TestWithVaultDecimals(t *testing.T) {
	vaults := []VaultInfo{{VaultAddress: "0xa", QuoteAmount: big.NewInt(1)}, {VaultAddress: "0xb", QuoteAmount: big.NewInt(1e6)}}
	got := WithVaultDecimals(vaults, []uint8{18, 0})
	assert.Equal(t, "1000000000000000000", got[0].QuoteAmount.String())
	assert.Equal(t, "1000000", got[1].QuoteAmount.String(), "a vault whose decimals did not come back keeps its amount")
	assert.Equal(t, "1", vaults[0].QuoteAmount.String(), "the input is left unchanged")
}
