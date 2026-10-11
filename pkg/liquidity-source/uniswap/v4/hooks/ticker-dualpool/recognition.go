package tickerdualpool

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type runtimeFingerprint struct {
	Size             int
	Hash             common.Hash
	ImmutableOffsets []int
}

// matches checks executable bytecode, while allowing the compiler's immutable
// constructor arguments and metadata to differ between otherwise identical builds.
// The actual immutable settings must still be fetched/validated by the tracker.
func (f runtimeFingerprint) matches(code []byte) bool {
	if len(code) < 2 {
		return false
	}
	metadata := int(binary.BigEndian.Uint16(code[len(code)-2:]))
	end := len(code) - metadata - 2
	if end != f.Size || end < 0 {
		return false
	}
	executable := append([]byte(nil), code[:end]...)
	for _, offset := range f.ImmutableOffsets {
		if offset < 0 || offset+32 > len(executable) {
			return false
		}
		clear(executable[offset : offset+32])
	}
	return crypto.Keccak256Hash(executable) == f.Hash
}
