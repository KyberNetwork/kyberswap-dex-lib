package meta

import (
	"bytes"
	"fmt"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/plain"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve/aave"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve/base"
	plainoracle "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve/plain-oracle"
	"github.com/tinylib/msgp/msgp"
)

const (
	basePoolAave = iota
	basePoolBase
	basePoolPlainOracle
	basePoolPlain
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
	case *aave.AavePool:
		basePoolType = basePoolAave
		encodable = pool
	case *base.PoolBaseSimulator:
		basePoolType = basePoolBase
		encodable = pool
	case *plainoracle.Pool:
		basePoolType = basePoolPlainOracle
		encodable = pool
	case *plain.PoolSimulator:
		basePoolType = basePoolPlain
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
	case basePoolAave:
		impl := new(aave.AavePool)
		decodable = impl
		pool = impl
	case basePoolBase:
		impl := new(base.PoolBaseSimulator)
		decodable = impl
		pool = impl
	case basePoolPlainOracle:
		impl := new(plainoracle.Pool)
		decodable = impl
		pool = impl
	case basePoolPlain:
		impl := new(plain.PoolSimulator)
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
