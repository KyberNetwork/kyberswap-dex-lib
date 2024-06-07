package msgpack

import (
	"io"
	"sync"

	"github.com/KyberNetwork/msgpack/v5"
)

var decoderPool = sync.Pool{
	New: func() any {
		de := msgpack.NewDecoder(nil)
		return de
	},
}

func NewDecoder(r io.Reader) *msgpack.Decoder {
	de := decoderPool.Get().(*msgpack.Decoder)
	de.Reset(r)
	de.IncludeUnexported(true)
	de.SetForceAsArray(true)
	return de
}

func PutDecoder(en *msgpack.Decoder) {
	decoderPool.Put(en)
}
