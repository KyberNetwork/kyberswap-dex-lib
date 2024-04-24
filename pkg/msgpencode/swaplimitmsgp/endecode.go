package swaplimitmsgp

import (
	"bytes"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/tinylib/msgp/msgp"
)

func EncodeSwapLimit(limit pool.SwapLimit) []byte {
	if limit == nil {
		return nil
	}
	enum := &SwapLimitEnum{}
	if err := enum.set(limit); err != nil {
		panic(err)
	}
	buf := new(bytes.Buffer)
	if err := msgp.Encode(buf, enum); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func DecodeSwapLimit(buf []byte) pool.SwapLimit {
	if buf == nil {
		return nil
	}
	enum := &SwapLimitEnum{}
	if err := msgp.Decode(bytes.NewReader(buf), enum); err != nil {
		panic(err)
	}
	return enum.get()
}
