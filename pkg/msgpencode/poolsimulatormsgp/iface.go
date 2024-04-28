package poolsimulatormsgp

// MsgpHookable specifices hook functions to be called durint Msgpack encoding/decoding.
type MsgpHookable interface {
	AfterMsgpDecode() error
}
