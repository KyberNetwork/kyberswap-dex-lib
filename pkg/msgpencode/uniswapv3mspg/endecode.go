package uniswapv3mspg

import (
	"bytes"

	uniswapv3entities "github.com/daoleno/uniswapv3-sdk/entities"
	"github.com/tinylib/msgp/msgp"
)

func EncodeTickListDataProvider(t *uniswapv3entities.TickListDataProvider) []byte {
	if t == nil {
		return nil
	}
	_t := new(TickListDataProvider).fromSdk(t)
	buf := new(bytes.Buffer)
	if err := msgp.Encode(buf, _t); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func DecodeTickListDataProvider(buf []byte) *uniswapv3entities.TickListDataProvider {
	if buf == nil {
		return nil
	}
	t := new(TickListDataProvider)
	if err := msgp.Decode(bytes.NewReader(buf), t); err != nil {
		panic(err)
	}
	return t.toSdk()
}

func EncodePool(p *uniswapv3entities.Pool) []byte {
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

func DecodePool(buf []byte) *uniswapv3entities.Pool {
	if buf == nil {
		return nil
	}
	p := new(Pool)
	if err := msgp.Decode(bytes.NewReader(buf), p); err != nil {
		panic(err)
	}
	return p.toSdk()
}
