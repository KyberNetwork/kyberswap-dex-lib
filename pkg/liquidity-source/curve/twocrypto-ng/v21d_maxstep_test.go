package twocryptong

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The earlier-build (max-step) v2.1.0d pools of every chain, by runtime code size (23655 bytes against the later
// build's 23613) over each chain's factory-twocrypto pools: three on Ethereum, two on BSC, none elsewhere.
func TestV21dMaxStepPoolsAllChains(t *testing.T) {
	want := []string{
		"0x83f24023d15d835a213df24fd309c47dab5beb32", // ethereum
		"0xd9ff8396554a0d18b2cfbec53e1979b7ecce8373", // ethereum
		"0xf1f435b05d255a5dbde37333c0f61da6f69c6127", // ethereum
		"0x8c65cec3847ad99bdc02621bdbc89f2ace56934b", // bsc
		"0xe4251b067c541f12c5032198884306b66122e44e", // bsc
	}
	require.Len(t, v21dMaxStepPools, len(want))
	for _, a := range want {
		_, ok := v21dMaxStepPools[a]
		require.True(t, ok, a)
	}
	for a := range v21dMaxStepPools {
		require.Equal(t, strings.ToLower(a), a, "keys are lower-case, as pool addresses are stored")
	}
}
