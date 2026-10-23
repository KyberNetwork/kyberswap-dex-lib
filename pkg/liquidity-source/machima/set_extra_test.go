package machima

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// TestSetExtraClearsWhatExtraLeavesOut: a price floor gone back to nil, which Extra omits, is
// cleared from the stored Extra, while a key Extra does not model, such as the UniV3 layer's
// lensDue, is kept.
func TestSetExtraClearsWhatExtraLeavesOut(t *testing.T) {
	p := entity.Pool{Extra: `{"xmaSellSqrtPriceLimit":12345,"lensDue":77,"hasTax":true}`}
	require.NoError(t, setExtra(&p, Extra{}))

	var got map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &got))
	assert.NotContains(t, got, "xmaSellSqrtPriceLimit", "the floor is gone")
	assert.Equal(t, "77", string(got["lensDue"]), "a key Extra does not model is kept")
	assert.Equal(t, "false", string(got["hasTax"]), "a modeled key takes the new value")
}
