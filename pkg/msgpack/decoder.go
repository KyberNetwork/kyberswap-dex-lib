package msgpack

import (
	"bytes"
	"io"
	"sync"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/msgpack/v5"
	"github.com/klauspost/compress/snappy"
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

// DecodePoolSimulatorsMap decodes an encoded and Snappy compressed map from pool ID to IPoolSimulator
func DecodePoolSimulatorsMap(encoded []byte) (map[string]pool.IPoolSimulator, error) {
	poolsMap := make(map[string]pool.IPoolSimulator)
	zw := snappy.NewReader(bytes.NewReader(encoded))
	de := NewDecoder(zw)
	defer PutDecoder(de)
	if err := de.Decode(&poolsMap); err != nil {
		return nil, err
	}
	return poolsMap, nil
}
