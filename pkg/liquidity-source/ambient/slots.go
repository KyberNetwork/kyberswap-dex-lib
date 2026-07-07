// Package ambient provides pure-Go primitives for reading CrocSwapDex
// (Ambient Finance) on-chain state via the contract's readSlot(uint256) view.
//
// Reference source: context/crocswap-protocol/contracts/mixins/StorageLayout.sol
// at commit db94f6d (branch mainnetDeploy).
package ambient

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/crypto/sha3"
)

// Storage base slot indices for the mappings declared in StorageLayout.
// Values marked (CrocSlots) are exposed as named constants on-chain inside
// the CrocSlots library. Values marked (inferred) come from the declaration
// order in StorageLayout.sol — empirically verified in Phase 1/2 tests.
const (
	LevelsMapSlot   uint64 = 65538 // (CrocSlots) levels_
	MezzMapSlot     uint64 = 65542 // (inferred)  mezzanine_
	TerminusMapSlot uint64 = 65543 // (inferred)  terminus_
	PoolsMapSlot    uint64 = 65545 // (inferred)  pools_
	CurvesMapSlot   uint64 = 65551 // (CrocSlots) curves_
)

func keccak256(parts ...[]byte) common.Hash {
	h := sha3.NewLegacyKeccak256()
	for _, p := range parts {
		h.Write(p)
	}
	var out common.Hash
	copy(out[:], h.Sum(nil))
	return out
}

func leftPad32(b []byte) []byte {
	if len(b) >= 32 {
		return b[len(b)-32:]
	}
	out := make([]byte, 32)
	copy(out[32-len(b):], b)
	return out
}

func uint256BE(u uint64) []byte {
	var out [32]byte
	binary.BigEndian.PutUint64(out[24:], u)
	return out[:]
}

// EncodePoolHash replicates PoolSpecs.encodeKey:
//
//	require(tokenX < tokenY);
//	return keccak256(abi.encode(tokenX, tokenY, poolIdx));
func EncodePoolHash(base, quote common.Address, poolIdx uint64) common.Hash {
	return keccak256(
		leftPad32(base.Bytes()),
		leftPad32(quote.Bytes()),
		uint256BE(poolIdx),
	)
}

func LobbyKey(tick int32) int8 { return int8(tick >> 16) }
func MezzKey(tick int32) int16 { return int16(tick >> 8) }
func MezzBit(tick int32) uint8 { return uint8(uint16(MezzKey(tick)) & 0xff) }
func TermBit(tick int32) uint8 { return uint8(uint32(tick) & 0xff) }

func packedBytes32Int8(h common.Hash, k int8) []byte {
	out := make([]byte, 33)
	copy(out[:32], h[:])
	out[32] = byte(k)
	return out
}

func packedBytes32Int16(h common.Hash, k int16) []byte {
	out := make([]byte, 34)
	copy(out[:32], h[:])
	binary.BigEndian.PutUint16(out[32:], uint16(k))
	return out
}

func packedBytes32Int24(h common.Hash, k int32) []byte {
	out := make([]byte, 35)
	copy(out[:32], h[:])
	u := uint32(k) & 0xffffff
	out[32] = byte(u >> 16)
	out[33] = byte(u >> 8)
	out[34] = byte(u)
	return out
}

// MezzMapKey = keccak256(abi.encodePacked(poolHash, int8(lobbyKey(tick)))).
func MezzMapKey(poolHash common.Hash, tick int32) common.Hash {
	return keccak256(packedBytes32Int8(poolHash, LobbyKey(tick)))
}

// TermMapKey = keccak256(abi.encodePacked(poolHash, int16(mezzKey(tick)))).
func TermMapKey(poolHash common.Hash, tick int32) common.Hash {
	return keccak256(packedBytes32Int16(poolHash, MezzKey(tick)))
}

// LevelMapKey = keccak256(abi.encodePacked(poolHash, int24(tick))).
func LevelMapKey(poolHash common.Hash, tick int32) common.Hash {
	return keccak256(packedBytes32Int24(poolHash, tick))
}

func mappingSlot(key common.Hash, slot uint64) common.Hash {
	return keccak256(key[:], uint256BE(slot))
}

// CurveSlot — base slot of curves_[poolHash]. 2 slots wide.
func CurveSlot(poolHash common.Hash) common.Hash {
	return mappingSlot(poolHash, CurvesMapSlot)
}

// PoolSpecsSlot — 1-slot Pool struct at pools_[poolHash].
func PoolSpecsSlot(poolHash common.Hash) common.Hash {
	return mappingSlot(poolHash, PoolsMapSlot)
}

func LevelSlot(poolHash common.Hash, tick int32) common.Hash {
	return mappingSlot(LevelMapKey(poolHash, tick), LevelsMapSlot)
}

func MezzSlot(poolHash common.Hash, tick int32) common.Hash {
	return mappingSlot(MezzMapKey(poolHash, tick), MezzMapSlot)
}

func TerminusSlot(poolHash common.Hash, tick int32) common.Hash {
	return mappingSlot(TermMapKey(poolHash, tick), TerminusMapSlot)
}
