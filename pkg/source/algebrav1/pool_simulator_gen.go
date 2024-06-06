package algebrav1

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpencode"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpencode/uniswapv3msgp"
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *PoolSimulator) DecodeMsg(dc *msgp.Reader) (err error) {
	var zb0001 uint32
	zb0001, err = dc.ReadArrayHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	if zb0001 != 8 {
		err = msgp.ArrayError{Wanted: 8, Got: zb0001}
		return
	}
	err = z.Pool.DecodeMsg(dc)
	if err != nil {
		err = msgp.WrapError(err, "Pool")
		return
	}
	err = z.globalState.DecodeMsg(dc)
	if err != nil {
		err = msgp.WrapError(err, "globalState")
		return
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "liquidity")
			return
		}
		z.liquidity = nil
	} else {
		{
			var zb0002 []byte
			zb0002, err = dc.ReadBytes(msgpencode.EncodeInt(z.liquidity))
			if err != nil {
				err = msgp.WrapError(err, "liquidity")
				return
			}
			z.liquidity = msgpencode.DecodeInt(zb0002)
		}
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "ticks")
			return
		}
		z.ticks = nil
	} else {
		{
			var zb0003 []byte
			zb0003, err = dc.ReadBytes(uniswapv3msgp.EncodeTickListDataProvider(z.ticks))
			if err != nil {
				err = msgp.WrapError(err, "ticks")
				return
			}
			z.ticks = uniswapv3msgp.DecodeTickListDataProvider(zb0003)
		}
	}
	z.gas, err = dc.ReadInt64()
	if err != nil {
		err = msgp.WrapError(err, "gas")
		return
	}
	z.tickMin, err = dc.ReadInt()
	if err != nil {
		err = msgp.WrapError(err, "tickMin")
		return
	}
	z.tickMax, err = dc.ReadInt()
	if err != nil {
		err = msgp.WrapError(err, "tickMax")
		return
	}
	z.tickSpacing, err = dc.ReadInt()
	if err != nil {
		err = msgp.WrapError(err, "tickSpacing")
		return
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *PoolSimulator) EncodeMsg(en *msgp.Writer) (err error) {
	// array header, size 8
	err = en.Append(0x98)
	if err != nil {
		return
	}
	err = z.Pool.EncodeMsg(en)
	if err != nil {
		err = msgp.WrapError(err, "Pool")
		return
	}
	err = z.globalState.EncodeMsg(en)
	if err != nil {
		err = msgp.WrapError(err, "globalState")
		return
	}
	if z.liquidity == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = en.WriteBytes(msgpencode.EncodeInt(z.liquidity))
		if err != nil {
			err = msgp.WrapError(err, "liquidity")
			return
		}
	}
	if z.ticks == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = en.WriteBytes(uniswapv3msgp.EncodeTickListDataProvider(z.ticks))
		if err != nil {
			err = msgp.WrapError(err, "ticks")
			return
		}
	}
	err = en.WriteInt64(z.gas)
	if err != nil {
		err = msgp.WrapError(err, "gas")
		return
	}
	err = en.WriteInt(z.tickMin)
	if err != nil {
		err = msgp.WrapError(err, "tickMin")
		return
	}
	err = en.WriteInt(z.tickMax)
	if err != nil {
		err = msgp.WrapError(err, "tickMax")
		return
	}
	err = en.WriteInt(z.tickSpacing)
	if err != nil {
		err = msgp.WrapError(err, "tickSpacing")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *PoolSimulator) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 8
	o = append(o, 0x98)
	o, err = z.Pool.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "Pool")
		return
	}
	o, err = z.globalState.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "globalState")
		return
	}
	if z.liquidity == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeInt(z.liquidity))
	}
	if z.ticks == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, uniswapv3msgp.EncodeTickListDataProvider(z.ticks))
	}
	o = msgp.AppendInt64(o, z.gas)
	o = msgp.AppendInt(o, z.tickMin)
	o = msgp.AppendInt(o, z.tickMax)
	o = msgp.AppendInt(o, z.tickSpacing)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *PoolSimulator) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	if zb0001 != 8 {
		err = msgp.ArrayError{Wanted: 8, Got: zb0001}
		return
	}
	bts, err = z.Pool.UnmarshalMsg(bts)
	if err != nil {
		err = msgp.WrapError(err, "Pool")
		return
	}
	bts, err = z.globalState.UnmarshalMsg(bts)
	if err != nil {
		err = msgp.WrapError(err, "globalState")
		return
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.liquidity = nil
	} else {
		{
			var zb0002 []byte
			zb0002, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.liquidity))
			if err != nil {
				err = msgp.WrapError(err, "liquidity")
				return
			}
			z.liquidity = msgpencode.DecodeInt(zb0002)
		}
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.ticks = nil
	} else {
		{
			var zb0003 []byte
			zb0003, bts, err = msgp.ReadBytesBytes(bts, uniswapv3msgp.EncodeTickListDataProvider(z.ticks))
			if err != nil {
				err = msgp.WrapError(err, "ticks")
				return
			}
			z.ticks = uniswapv3msgp.DecodeTickListDataProvider(zb0003)
		}
	}
	z.gas, bts, err = msgp.ReadInt64Bytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "gas")
		return
	}
	z.tickMin, bts, err = msgp.ReadIntBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "tickMin")
		return
	}
	z.tickMax, bts, err = msgp.ReadIntBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "tickMax")
		return
	}
	z.tickSpacing, bts, err = msgp.ReadIntBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "tickSpacing")
		return
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *PoolSimulator) Msgsize() (s int) {
	s = 1 + z.Pool.Msgsize() + z.globalState.Msgsize()
	if z.liquidity == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(z.liquidity))
	}
	if z.ticks == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(uniswapv3msgp.EncodeTickListDataProvider(z.ticks))
	}
	s += msgp.Int64Size + msgp.IntSize + msgp.IntSize + msgp.IntSize
	return
}