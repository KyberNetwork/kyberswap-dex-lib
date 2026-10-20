package pool

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// Indexing must price at the snapshot, not the wall clock: a DB row older than the tracked state
// is still quotable as of the state it holds. Route finding keeps the wall clock.
func TestQuoteTime(t *testing.T) {
	ep := entity.Pool{Timestamp: 1700000000}

	assert.Equal(t, int64(1700000000), QuoteTime(false, ep.Timestamp).Unix())
	assert.True(t, QuoteTime(true, ep.Timestamp).IsZero())
	assert.True(t, QuoteTime(false, 0).IsZero(), "no snapshot means the caller's clock")
}
