package testutil

import (
	"encoding/hex"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// HexFixture is the bytes of the hex file at path, such as a recorded lens reply.
func HexFixture(t testing.TB, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	b, err := hex.DecodeString(strings.TrimSpace(string(raw)))
	require.NoError(t, err)
	return b
}
