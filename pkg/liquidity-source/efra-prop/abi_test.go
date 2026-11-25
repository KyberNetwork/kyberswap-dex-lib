package efraprop

import (
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The embedded ABI must stay the venue's own: a typo here would make every probe decode
// garbage (or revert), which looks exactly like an untradeable market.
func TestVenueABI(t *testing.T) {
	quote, ok := venueABI.Methods["quoteExactIn"]
	require.True(t, ok)
	require.Len(t, quote.Inputs, 4)
	assert.Equal(t, "address", quote.Inputs[0].Type.String())
	assert.Equal(t, "address", quote.Inputs[1].Type.String())
	assert.Equal(t, "uint256", quote.Inputs[2].Type.String())
	assert.Equal(t, "address", quote.Inputs[3].Type.String(), "taker, so a gap discount is included")
	require.Len(t, quote.Outputs, 1)
	assert.Equal(t, "uint256", quote.Outputs[0].Type.String())

	usdc, ok := venueABI.Methods["USDC"]
	require.True(t, ok)
	assert.Empty(t, usdc.Inputs)

	selector := crypto.Keccak256([]byte("quoteExactIn(address,address,uint256,address)"))[:4]
	assert.Equal(t, hex.EncodeToString(selector), hex.EncodeToString(venueABI.Methods["quoteExactIn"].ID))
}
