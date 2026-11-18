package tickerdualpool

import (
	"testing"

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
