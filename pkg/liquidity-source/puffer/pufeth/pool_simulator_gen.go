package pufeth

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpencode"
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *Gas) DecodeMsg(dc *msgp.Reader) (err error) {
	var zb0001 uint32
	zb0001, err = dc.ReadArrayHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	if zb0001 != 2 {
		err = msgp.ArrayError{Wanted: 2, Got: zb0001}
		return
	}
	z.depositStETH, err = dc.ReadInt64()
	if err != nil {
		err = msgp.WrapError(err, "depositStETH")
		return
	}
	z.depositWstETH, err = dc.ReadInt64()
	if err != nil {
		err = msgp.WrapError(err, "depositWstETH")
		return
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z Gas) EncodeMsg(en *msgp.Writer) (err error) {
	// array header, size 2
	err = en.Append(0x92)
	if err != nil {
		return
	}
	err = en.WriteInt64(z.depositStETH)
	if err != nil {
		err = msgp.WrapError(err, "depositStETH")
		return
	}
	err = en.WriteInt64(z.depositWstETH)
	if err != nil {
		err = msgp.WrapError(err, "depositWstETH")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z Gas) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 2
	o = append(o, 0x92)
	o = msgp.AppendInt64(o, z.depositStETH)
	o = msgp.AppendInt64(o, z.depositWstETH)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Gas) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	if zb0001 != 2 {
		err = msgp.ArrayError{Wanted: 2, Got: zb0001}
		return
	}
	z.depositStETH, bts, err = msgp.ReadInt64Bytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "depositStETH")
		return
	}
	z.depositWstETH, bts, err = msgp.ReadInt64Bytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "depositWstETH")
		return
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z Gas) Msgsize() (s int) {
	s = 1 + msgp.Int64Size + msgp.Int64Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *PoolSimulator) DecodeMsg(dc *msgp.Reader) (err error) {
	var zb0001 uint32
	zb0001, err = dc.ReadArrayHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	if zb0001 != 6 {
		err = msgp.ArrayError{Wanted: 6, Got: zb0001}
		return
	}
	err = z.Pool.DecodeMsg(dc)
	if err != nil {
		err = msgp.WrapError(err, "Pool")
		return
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "totalSupply")
			return
		}
		z.totalSupply = nil
	} else {
		{
			var zb0002 []byte
			zb0002, err = dc.ReadBytes(msgpencode.EncodeUint256(z.totalSupply))
			if err != nil {
				err = msgp.WrapError(err, "totalSupply")
				return
			}
			z.totalSupply = msgpencode.DecodeUint256(zb0002)
		}
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "totalAssets")
			return
		}
		z.totalAssets = nil
	} else {
		{
			var zb0003 []byte
			zb0003, err = dc.ReadBytes(msgpencode.EncodeUint256(z.totalAssets))
			if err != nil {
				err = msgp.WrapError(err, "totalAssets")
				return
			}
			z.totalAssets = msgpencode.DecodeUint256(zb0003)
		}
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "totalPooledEther")
			return
		}
		z.totalPooledEther = nil
	} else {
		{
			var zb0004 []byte
			zb0004, err = dc.ReadBytes(msgpencode.EncodeUint256(z.totalPooledEther))
			if err != nil {
				err = msgp.WrapError(err, "totalPooledEther")
				return
			}
			z.totalPooledEther = msgpencode.DecodeUint256(zb0004)
		}
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "totalShares")
			return
		}
		z.totalShares = nil
	} else {
		{
			var zb0005 []byte
			zb0005, err = dc.ReadBytes(msgpencode.EncodeUint256(z.totalShares))
			if err != nil {
				err = msgp.WrapError(err, "totalShares")
				return
			}
			z.totalShares = msgpencode.DecodeUint256(zb0005)
		}
	}
	var zb0006 uint32
	zb0006, err = dc.ReadArrayHeader()
	if err != nil {
		err = msgp.WrapError(err, "gas")
		return
	}
	if zb0006 != 2 {
		err = msgp.ArrayError{Wanted: 2, Got: zb0006}
		return
	}
	z.gas.depositStETH, err = dc.ReadInt64()
	if err != nil {
		err = msgp.WrapError(err, "gas", "depositStETH")
		return
	}
	z.gas.depositWstETH, err = dc.ReadInt64()
	if err != nil {
		err = msgp.WrapError(err, "gas", "depositWstETH")
		return
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *PoolSimulator) EncodeMsg(en *msgp.Writer) (err error) {
	// array header, size 6
	err = en.Append(0x96)
	if err != nil {
		return
	}
	err = z.Pool.EncodeMsg(en)
	if err != nil {
		err = msgp.WrapError(err, "Pool")
		return
	}
	if z.totalSupply == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = en.WriteBytes(msgpencode.EncodeUint256(z.totalSupply))
		if err != nil {
			err = msgp.WrapError(err, "totalSupply")
			return
		}
	}
	if z.totalAssets == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = en.WriteBytes(msgpencode.EncodeUint256(z.totalAssets))
		if err != nil {
			err = msgp.WrapError(err, "totalAssets")
			return
		}
	}
	if z.totalPooledEther == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = en.WriteBytes(msgpencode.EncodeUint256(z.totalPooledEther))
		if err != nil {
			err = msgp.WrapError(err, "totalPooledEther")
			return
		}
	}
	if z.totalShares == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = en.WriteBytes(msgpencode.EncodeUint256(z.totalShares))
		if err != nil {
			err = msgp.WrapError(err, "totalShares")
			return
		}
	}
	// array header, size 2
	err = en.Append(0x92)
	if err != nil {
		return
	}
	err = en.WriteInt64(z.gas.depositStETH)
	if err != nil {
		err = msgp.WrapError(err, "gas", "depositStETH")
		return
	}
	err = en.WriteInt64(z.gas.depositWstETH)
	if err != nil {
		err = msgp.WrapError(err, "gas", "depositWstETH")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *PoolSimulator) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 6
	o = append(o, 0x96)
	o, err = z.Pool.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "Pool")
		return
	}
	if z.totalSupply == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeUint256(z.totalSupply))
	}
	if z.totalAssets == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeUint256(z.totalAssets))
	}
	if z.totalPooledEther == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeUint256(z.totalPooledEther))
	}
	if z.totalShares == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeUint256(z.totalShares))
	}
	// array header, size 2
	o = append(o, 0x92)
	o = msgp.AppendInt64(o, z.gas.depositStETH)
	o = msgp.AppendInt64(o, z.gas.depositWstETH)
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
	if zb0001 != 6 {
		err = msgp.ArrayError{Wanted: 6, Got: zb0001}
		return
	}
	bts, err = z.Pool.UnmarshalMsg(bts)
	if err != nil {
		err = msgp.WrapError(err, "Pool")
		return
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.totalSupply = nil
	} else {
		{
			var zb0002 []byte
			zb0002, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeUint256(z.totalSupply))
			if err != nil {
				err = msgp.WrapError(err, "totalSupply")
				return
			}
			z.totalSupply = msgpencode.DecodeUint256(zb0002)
		}
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.totalAssets = nil
	} else {
		{
			var zb0003 []byte
			zb0003, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeUint256(z.totalAssets))
			if err != nil {
				err = msgp.WrapError(err, "totalAssets")
				return
			}
			z.totalAssets = msgpencode.DecodeUint256(zb0003)
		}
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.totalPooledEther = nil
	} else {
		{
			var zb0004 []byte
			zb0004, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeUint256(z.totalPooledEther))
			if err != nil {
				err = msgp.WrapError(err, "totalPooledEther")
				return
			}
			z.totalPooledEther = msgpencode.DecodeUint256(zb0004)
		}
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.totalShares = nil
	} else {
		{
			var zb0005 []byte
			zb0005, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeUint256(z.totalShares))
			if err != nil {
				err = msgp.WrapError(err, "totalShares")
				return
			}
			z.totalShares = msgpencode.DecodeUint256(zb0005)
		}
	}
	var zb0006 uint32
	zb0006, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "gas")
		return
	}
	if zb0006 != 2 {
		err = msgp.ArrayError{Wanted: 2, Got: zb0006}
		return
	}
	z.gas.depositStETH, bts, err = msgp.ReadInt64Bytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "gas", "depositStETH")
		return
	}
	z.gas.depositWstETH, bts, err = msgp.ReadInt64Bytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "gas", "depositWstETH")
		return
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *PoolSimulator) Msgsize() (s int) {
	s = 1 + z.Pool.Msgsize()
	if z.totalSupply == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeUint256(z.totalSupply))
	}
	if z.totalAssets == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeUint256(z.totalAssets))
	}
	if z.totalPooledEther == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeUint256(z.totalPooledEther))
	}
	if z.totalShares == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeUint256(z.totalShares))
	}
	s += 1 + msgp.Int64Size + msgp.Int64Size
	return
}
