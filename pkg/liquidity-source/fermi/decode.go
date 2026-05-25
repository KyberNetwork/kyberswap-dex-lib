package fermi

import (
	"encoding/binary"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// pairStateSlot is the Solidity slot index of FermiEngine's
// mapping(uint256 => PairState) _isActive (slot 6).
const pairStateSlot = 6

// pairBaseSlot returns the storage slot at which the PairState for
// pairKey begins: keccak256(pairKey . slot).
func pairBaseSlot(pairKey common.Hash) common.Hash {
	var buf [64]byte
	copy(buf[:32], pairKey[:])
	buf[63] = byte(pairStateSlot)
	return common.BytesToHash(crypto.Keccak256(buf[:]))
}

// pairKeyForTokens replicates FermiEngine's keccak256(tokenA, tokenB).
// Returns both directions because the canonical registration direction
// is not knowable off-chain.
func pairKeyForTokens(a, b common.Address) (forward, reverse common.Hash) {
	var buf [64]byte
	copy(buf[12:32], a.Bytes())
	copy(buf[44:64], b.Bytes())
	forward = common.BytesToHash(crypto.Keccak256(buf[:]))

	copy(buf[12:32], b.Bytes())
	copy(buf[44:64], a.Bytes())
	reverse = common.BytesToHash(crypto.Keccak256(buf[:]))
	return
}

// slotOffset returns base + n as a storage slot hash.
func slotOffset(base common.Hash, n uint64) common.Hash {
	b := new(big.Int).SetBytes(base[:])
	b.Add(b, new(big.Int).SetUint64(n))
	return common.BigToHash(b)
}

func decodeMidPrice(word common.Hash) *big.Int {
	return new(big.Int).SetBytes(word[:])
}

func decodeLastUpdatedBlock(word common.Hash) uint64 {
	return binary.BigEndian.Uint64(word[len(word)-9 : len(word)-1])
}
