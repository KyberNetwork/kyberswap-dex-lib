package stablemetang

import (
	"bytes"
	"fmt"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/plain"
	stableng "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/stable-ng"
	"github.com/tinylib/msgp/msgp"
)

const (
	basePoolPlain = iota
	basePoolStable
	basePoolMeta
)

func encodeBasePool(pool ICurveBasePool) []byte {
	if pool == nil {
		return nil
	}

	var (
		basePoolType byte
		encodable    msgp.Encodable
		buf          = new(bytes.Buffer)
	)
	switch pool := pool.(type) {
	case *plain.PoolSimulator:
		basePoolType = basePoolPlain
		encodable = pool
	case *stableng.PoolSimulator:
		basePoolType = basePoolStable
		encodable = pool
	case *PoolSimulator:
		basePoolType = basePoolMeta
		encodable = pool
	default:
		panic("invalid ICurveBasePool concrete type")
	}

	if err := buf.WriteByte(basePoolType); err != nil {
		panic(fmt.Sprintf("could not encode pool type: %s", err))
	}

	if err := msgp.Encode(buf, encodable); err != nil {
		panic(fmt.Sprintf("could not encode ICurveBasePool: %s", err))
	}

	return buf.Bytes()
}

func decodeBasePool(encoded []byte) ICurveBasePool {
	if encoded == nil {
		return nil
	}

	var (
		buf          = bytes.NewBuffer(encoded)
		basePoolType byte
		decodable    msgp.Decodable
		pool         ICurveBasePool
		err          error
	)
	if basePoolType, err = buf.ReadByte(); err != nil {
		panic(fmt.Sprintf("could not read pool type: %s", err))
	}

	switch basePoolType {
	case basePoolPlain:
		impl := new(plain.PoolSimulator)
		decodable = impl
		pool = impl
	case basePoolStable:
		impl := new(stableng.PoolSimulator)
		decodable = impl
		pool = impl
	case basePoolMeta:
		impl := new(PoolSimulator)
		decodable = impl
		pool = impl
	default:
		panic(fmt.Sprintf("invalid pool type %d", basePoolType))
	}

	if err := msgp.Decode(buf, decodable); err != nil {
		panic(fmt.Sprintf("could not decode ICurveBasePool: %s", err))
	}

	return pool
}
