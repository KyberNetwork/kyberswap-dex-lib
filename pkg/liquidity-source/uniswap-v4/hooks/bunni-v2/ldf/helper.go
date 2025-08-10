package ldf

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
