package tickerdualpool

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestRuntimeFingerprintChecksInstructionsAndAllowsImmutables(t *testing.T) {
	executable := make([]byte, 64)
	executable[0] = 0x60
	fingerprint := runtimeFingerprint{Size: 64, Hash: crypto.Keccak256Hash(executable), ImmutableOffsets: []int{16}}
	code := append(append([]byte(nil), executable...), 0xa0, 0x00, 0x01)
	code[20] = 123
	require.True(t, fingerprint.matches(code))
	code[0] ^= 1
	require.False(t, fingerprint.matches(code), "changed executable logic must be rejected")
	require.False(t, fingerprint.matches(nil))
	require.False(t, fingerprint.matches([]byte{0xff, 0xff}))
	fingerprint.ImmutableOffsets = []int{40}
	require.False(t, fingerprint.matches(code))
}

// Source-path changes alter Solidity's metadata, not this proxy's executable.
func TestProxyFingerprintAllowsMetadataButRejectsChangedInstructions(t *testing.T) {
	code := common.FromHex("0x60806040527f360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc545f9081906001600160a01b0316368280378136915af43d5f803e156048573d5ff35b3d5ffdfea264697066735822122044c8b9786344fee6be53958ad44cfa4e69b9f47919040c0a4950bfcf47095b0f64736f6c634300081a0033")
	require.True(t, tickerProxyFingerprint.matches(code))
	code[90] ^= 1
	require.True(t, tickerProxyFingerprint.matches(code), "metadata is not executable")
	code[20] ^= 1
	require.False(t, tickerProxyFingerprint.matches(code), "implementation-slot instructions must match")
}
