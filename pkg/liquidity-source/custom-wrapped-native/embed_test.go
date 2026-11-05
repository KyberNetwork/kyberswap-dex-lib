package customwrappednative

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Every pool must pair the chain's canonical wrapped native with the wrapper that is the pool id,
// otherwise the router's unwrap/wrap handling and the encoder's direction check break.
func TestElysiumPools(t *testing.T) {
	var pools []struct {
		ID     string `json:"id"`
		Tokens []struct {
			Address string `json:"address"`
		} `json:"tokens"`
	}
	require.NoError(t, json.Unmarshal(BytesByPath["pools/elysium.json"], &pools))
	require.NotEmpty(t, pools)
	canonical := valueobject.WrapNativeLower(valueobject.NativeAddress, valueobject.ChainIDElysium)
	for _, p := range pools {
		require.Len(t, p.Tokens, 2)
		require.Equal(t, canonical, p.Tokens[0].Address)
		require.Equal(t, p.ID, p.Tokens[1].Address)
	}
}
