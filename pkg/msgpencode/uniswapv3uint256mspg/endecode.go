package uniswapv3uint256mspg

import (
	"bytes"

	uniswapv3entities "github.com/KyberNetwork/uniswapv3-sdk-uint256/entities"
	"github.com/tinylib/msgp/msgp"
)

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
