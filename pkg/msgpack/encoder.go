package msgpack

import (
	"bytes"
	"io"
	"sync"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/msgpack/v5"
	"github.com/klauspost/compress/snappy"
)

var encoderPool = sync.Pool{
	New: func() any {
		en := msgpack.NewEncoder(nil)
		return en
	},
}

func NewEncoder(w io.Writer) *msgpack.Encoder {
	en := encoderPool.Get().(*msgpack.Encoder)
	en.Reset(w)
	en.IncludeUnexported(true)
	en.SetForceAsArray(true)
	return en
}

func PutEncoder(en *msgpack.Encoder) {
	encoderPool.Put(en)
}

// EncodePoolSimulatorsMap encode a map from pool ID to IPoolSimulator with Snappy compression
func EncodePoolSimulatorsMap(poolsMap map[string]pool.IPoolSimulator) ([]byte, error) {
	var (
		buf bytes.Buffer
		zw  = snappy.NewBufferedWriter(&buf)
	)
	en := NewEncoder(zw)
	defer PutEncoder(en)
	if err := en.Encode(poolsMap); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
