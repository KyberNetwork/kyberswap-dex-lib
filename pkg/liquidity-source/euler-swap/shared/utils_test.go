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

// TestQuoteAmount: the quote amount does not depend on any decimals, is at least PriceScale (which bounds the
// stored price's precision, so a larger amount buys none), is below the uint128 limit that Euler's
// UniswapV3 oracle adapter puts on inAmount, and gives the same price per smallest unit as a whole-unit quote.
func TestQuoteAmount(t *testing.T) {
	t.Parallel()

	e := func(n int64) *big.Int { return new(big.Int).Exp(big.NewInt(10), big.NewInt(n), nil) }
	require.Equal(t, 0, QuoteAmount.Cmp(e(18)), "got %s", QuoteAmount)
	require.GreaterOrEqual(t, QuoteAmount.Cmp(PriceScale), 0, "below PriceScale the stored price loses digits")
	uint128Max := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1))
	require.Equal(t, -1, QuoteAmount.Cmp(uint128Max))

	// An oracle's quote is linear in inAmount: quoting QuoteAmount of an asset with d decimals, priced at p of a
	// unit of account with u decimals, answers p*10^u*QuoteAmount/10^d. The stored price must be the same as for
	// one whole unit, p*10^u/10^d*PriceScale.
	for _, tc := range []struct {
		name       string
		assetDec   int64
		unitDec    int64
		priceNum   int64 // price p = priceNum / 10^priceExp
		priceExp   int64
		wantPerUnt string // price per smallest unit, times PriceScale
	}{
		{"WETH 18 dec at 2315.96 in an 18-dec USD", 18, 18, 231596, 2, "2315960000000000000000"},
		{"USDC 6 dec at 1 in an 18-dec USD", 6, 18, 1, 0, "1000000000000000000000000000000"},
		{"18-dec token at 0.99848 in an 18-dec USD", 18, 18, 99848, 5, "998480000000000000"},
		{"0-dec token at 1 in an 8-dec USD", 0, 8, 1, 0, "100000000000000000000000000"},
		{"20-dec token at 1 in an 18-dec USD", 20, 18, 1, 0, "10000000000000000"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			quote := new(big.Int).Mul(big.NewInt(tc.priceNum), e(tc.unitDec))
			quote.Mul(quote, QuoteAmount).Div(quote, e(tc.priceExp)).Div(quote, e(tc.assetDec))
			require.Equal(t, tc.wantPerUnt, UnitPrice(quote, QuoteAmount).String())
		})
	}
}
