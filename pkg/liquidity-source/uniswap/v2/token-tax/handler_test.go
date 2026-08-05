package tokentax

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

func TestNewHandler(t *testing.T) {
	t.Parallel()

	// A pool tracked by an older/removed detection mechanism must keep applying its last known
	// tax until the tracker refreshes it.
	handler := NewHandler(&TaxInfo{Token: "0xagent", BuyTaxBps: uint256.NewInt(100), Checked: true})
	assert.Equal(t, "0xagent", handler.TokenAddress)
	assert.Equal(t, uint256.NewInt(100), handler.BuyTaxBps)

	// nil (never tracked, or an ordinary non-taxed pool) yields a no-op handler.
	assert.Equal(t, Handler{}, NewHandler(nil))
}
