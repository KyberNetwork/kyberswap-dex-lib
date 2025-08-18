package ldf

import "encoding/binary"

const (
	QUERY_SCALE_SHIFT        = 4
	INITIALIZED_STATE uint32 = 1 << 24
)

// DecodeState decodes a bytes32 state into initialized flag and lastMinTick
func DecodeState(ldfState [32]byte) (initialized bool, lastMinTick int32) {
	// | initialized - 1 byte | lastMinTick - 3 bytes |
	initialized = ldfState[0] == 1

	lastMinTickRaw := uint32(ldfState[1])<<16 | uint32(ldfState[2])<<8 | uint32(ldfState[3])

	if lastMinTickRaw&0x800000 != 0 {
		lastMinTickRaw |= 0xFF000000
	}
	lastMinTick = int32(lastMinTickRaw)

	return
}

func EncodeState(twapTick int) (state [32]byte) {
	// | initialized - 1 byte | lastTickLower - 3 bytes |
	lastTickLowerUint24 := uint32(twapTick) & 0xFFFFFF

	combined := INITIALIZED_STATE + lastTickLowerUint24

	binary.BigEndian.PutUint32(state[:4], combined)

	return
}

func SignExtend24to32(val uint32) uint32 {
	if val&0x800000 != 0 {
		return val | 0xFF000000
	}
	return val & 0x00FFFFFF
}
