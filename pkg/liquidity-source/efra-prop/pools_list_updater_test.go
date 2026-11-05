package efraprop

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

const testOther = "0x4444444444444444444444444444444444444444"

// Every market of this venue shares one contract address, so the pool id has to be derived
// from the pair or two markets would collide in the store.
func TestBuildPools(t *testing.T) {
	u := &PoolsListUpdater{cfg: &Config{
		DexID: DexType,
		Venue: "0xABCDEF0123456789012345678901234567890123",
		Tokens: []string{
			testNVDA,
			"0x5555555555555555555555555555555555555555",
			testNVDA, // duplicate
			testUSDC, // the quote token is not a market of its own
		},
	}}

	pools, err := u.buildPools(common.HexToAddress(testUSDC), 1_700_000_000)
	require.NoError(t, err)
	require.Len(t, pools, 2)

	assert.Equal(t, syntheticPoolAddress(DexType, testNVDA, testUSDC), pools[0].Address)
	assert.Equal(t, syntheticPoolAddress(DexType, "0x5555555555555555555555555555555555555555", testUSDC), pools[1].Address)
	assert.NotEqual(t, pools[0].Address, pools[1].Address)

	// Stock token first, USDC second: the tracker and simulator index directions by token
	// order, and every market is token <-> USDC.
	assert.Equal(t, testNVDA, pools[0].Tokens[0].Address)
	assert.Equal(t, testUSDC, pools[0].Tokens[1].Address)
	assert.Equal(t, entity.PoolReserves{"0", "0"}, pools[0].Reserves, "discovery never sets reserves")

	var staticExtra StaticExtra
	require.NoError(t, json.Unmarshal([]byte(pools[0].StaticExtra), &staticExtra))
	assert.Equal(t, "0xabcdef0123456789012345678901234567890123", staticExtra.Venue)
	assert.Equal(t, strings.ToLower(defaultDest.Hex()), staticExtra.Taker,
		"probes use the executor's address so the venue's gap discount is included")
}

func TestBuildPools_InvalidToken(t *testing.T) {
	u := &PoolsListUpdater{cfg: &Config{DexID: DexType, Venue: testVenue, Tokens: []string{"not-an-address"}}}
	_, err := u.buildPools(common.HexToAddress(testUSDC), 0)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

// The venue's fee is flat USDC and comes off the output, so a haircut must scale the probed
// amountOut only; a zero buffer must leave probes untouched.
func TestApplyBuffer(t *testing.T) {
	outs := []*big.Int{big.NewInt(180_000_000), big.NewInt(356_400_000)}
	(&PoolTracker{cfg: &Config{Buffer: 9_990}}).applyBuffer(outs)
	assert.Equal(t, []*big.Int{big.NewInt(179_820_000), big.NewInt(356_043_600)}, outs)

	untouched := []*big.Int{big.NewInt(180_000_000)}
	(&PoolTracker{cfg: &Config{}}).applyBuffer(untouched)
	assert.Equal(t, []*big.Int{big.NewInt(180_000_000)}, untouched)
}
