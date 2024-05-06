package elasticmsgp

import (
	"bytes"

	elasticentities "github.com/KyberNetwork/elastic-go-sdk/v2/entities"
	"github.com/tinylib/msgp/msgp"
)

func EncodePool(p *elasticentities.Pool) []byte {
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

func DecodePool(buf []byte) *elasticentities.Pool {
	if buf == nil {
		return nil
	}
	p := new(Pool)
	if err := msgp.Decode(bytes.NewReader(buf), p); err != nil {
		panic(err)
	}
	return p.toSdk()
}
