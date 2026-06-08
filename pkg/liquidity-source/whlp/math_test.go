package whlp

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuoteToShare(t *testing.T) {
	rate := big.NewInt(1_136_850)
	oneShare := big.NewInt(1_000_000)

	out, err := quoteToShare(big.NewInt(1_000_000), rate, oneShare)
	require.NoError(t, err)
	assert.Equal(t, int64(879_623), out.Int64())
}

func TestShareToQuote(t *testing.T) {
	rate := big.NewInt(1_136_850)
	oneShare := big.NewInt(1_000_000)

	out, err := shareToQuote(big.NewInt(1_000_000), rate, oneShare)
	require.NoError(t, err)
	assert.Equal(t, int64(1_136_850), out.Int64())
}

func TestQuoteToShareZeroAmount(t *testing.T) {
	_, err := quoteToShare(big.NewInt(0), big.NewInt(1_136_850), big.NewInt(1_000_000))
	assert.ErrorIs(t, err, ErrZeroAmount)
}
