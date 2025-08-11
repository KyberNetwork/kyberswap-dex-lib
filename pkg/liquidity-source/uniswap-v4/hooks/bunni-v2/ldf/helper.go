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
	lastMinTick = int32(uint32(ldfState[1])<<16 | uint32(ldfState[2])<<8 | uint32(ldfState[3]))
	return
}

func EncodeState(twapTick int) [32]byte {
	var state [32]byte
	twapTickUint24 := uint32(twapTick) & 0xFFFFFF
	combined := INITIALIZED_STATE + twapTickUint24

	binary.BigEndian.PutUint32(state[:4], combined)

	return state
}
