package pancakev3msgp

import (
	"bytes"

	pancakev3entities "github.com/KyberNetwork/pancake-v3-sdk/entities"
	"github.com/tinylib/msgp/msgp"
)

func EncodePool(p *pancakev3entities.Pool) []byte {
	if p == nil {
		return nil
	}
	_p := new(Pool).fromSdk(p)
	buf := new(bytes.Buffer)
	if err := msgp.Encode(buf, _p); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func DecodePool(buf []byte) *pancakev3entities.Pool {
	if buf == nil {
		return nil
	}
	p := new(Pool)
	if err := msgp.Decode(bytes.NewReader(buf), p); err != nil {
		panic(err)
	}
	return p.toSdk()
}
