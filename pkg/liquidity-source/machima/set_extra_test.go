package machima

import (
	"testing"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
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

// TestTaxBlock: the tax of a pool the lens just gave its state is read at the snapshot's block,
// which dates the pool, not at its older logs'; any other at the logs' block, or the latest.
func TestTaxBlock(t *testing.T) {
	logs := []ethtypes.Log{{BlockNumber: 90}}
	read := entity.Pool{BlockNumber: 120, Extra: `{"liquidity":1,"sqrtPriceX96":1,"ticks":[]}`}
	assert.EqualValues(t, 120, taxBlock(false, read, logs), "read whole through the lens")
	assert.EqualValues(t, 90, taxBlock(true, read, logs), "updated from its logs")
	assert.EqualValues(t, 90, taxBlock(false, entity.Pool{BlockNumber: 120, Extra: `{}`}, logs),
		"left without state")
	assert.Zero(t, taxBlock(true, read, nil), "an interval refresh, at the latest")
}
