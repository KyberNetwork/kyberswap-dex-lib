package parityprop

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const referenceRegistryAddress = "0xe9890479e4b7ba305b48d729073a13d17ff80597"

// TestGetNewPools_Live exercises registry.getPools() discovery against the
// real registry and confirms the known reference pool decodes with the
// correct base/quote/scale values, with reserves left at "0" (the tracker
// fills real reserves on the first refresh).
func TestGetNewPools_Live(t *testing.T) {
	test.SkipCI(t)

	cfg := &Config{ChainID: valueobject.ChainIDRobinhood, Registry: referenceRegistryAddress}
	updater := NewPoolsListUpdater(cfg, newLiveRPCClient())

	pools, metadataBytes, err := updater.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.NotEmpty(t, pools, "registry.getPools() should return at least the reference pool")

	var found *entity.Pool
	for i := range pools {
		if pools[i].Address == referencePoolAddress {
			found = &pools[i]
			break
		}
	}
	require.NotNil(t, found, "reference pool %s not discovered", referencePoolAddress)
	assert.Equal(t, DexType, found.Type)
	assert.EqualValues(t, []string{"0", "0"}, found.Reserves)
	require.Len(t, found.Tokens, 2)
	assert.Equal(t, weth, found.Tokens[0].Address)
	assert.True(t, found.Tokens[0].Swappable)
	assert.Equal(t, usdg, found.Tokens[1].Address)
	assert.True(t, found.Tokens[1].Swappable)

	var staticExtra StaticExtra
	require.NoError(t, json.Unmarshal([]byte(found.StaticExtra), &staticExtra))
	assert.Equal(t, "1000000000000000000", staticExtra.BaseScale)
	assert.Equal(t, "1000000", staticExtra.QuoteScale)

	// Metadata.Seen must record every pool already resolved, so a second
	// call with the returned metadata does not re-resolve (and therefore
	// does not re-return) the reference pool.
	var metadata Metadata
	require.NoError(t, json.Unmarshal(metadataBytes, &metadata))
	assert.True(t, metadata.Seen[referencePoolAddress])

	pools2, _, err := updater.GetNewPools(context.Background(), metadataBytes)
	require.NoError(t, err)
	for i := range pools2 {
		assert.NotEqual(t, referencePoolAddress, pools2[i].Address,
			"seen-set continuation re-resolved an already-seen pool")
	}
}

// TestUnseenAddrs_SwapAndPopDoesNotSkipPools guards the specific bug an
// index-based offset would hit: PmmRegistry.removePool() is documented as
// swap-and-pop, so a pool that was never resolved can be moved into a slot
// index-based tracking would treat as already scanned. Filtering by address
// must still pick it up regardless of its position in getPools().
func TestUnseenAddrs_SwapAndPopDoesNotSkipPools(t *testing.T) {
	addrA := common.HexToAddress("0x00000000000000000000000000000000000aaa")
	addrB := common.HexToAddress("0x00000000000000000000000000000000000bbb")
	addrC := common.HexToAddress("0x00000000000000000000000000000000000ccc")

	// A 2-element registry [A, B]; a prior refresh resolved both (an
	// index-based scheme would have advanced its offset to 2).
	seen := map[string]bool{
		hexutil.Encode(addrA[:]): true,
		hexutil.Encode(addrB[:]): true,
	}

	// Then B is removed via swap-and-pop: the registry now returns [A, C]
	// -- C (never resolved) has been swapped into index 1, the exact slot
	// an offset of 2 would treat as already scanned.
	poolAddrs := []common.Address{addrA, addrC}

	got := unseenAddrs(poolAddrs, seen)
	require.Len(t, got, 1)
	assert.Equal(t, addrC, got[0])
}
