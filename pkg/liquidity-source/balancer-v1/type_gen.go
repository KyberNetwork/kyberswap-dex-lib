package balancerv1

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
	if zb0001 != 1 {
		err = msgp.ArrayError{Wanted: 1, Got: zb0001}
		return
	}
	z.SwapExactAmountIn, err = dc.ReadInt64()
	if err != nil {
		err = msgp.WrapError(err, "SwapExactAmountIn")
		return
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z Gas) EncodeMsg(en *msgp.Writer) (err error) {
	// array header, size 1
	err = en.Append(0x91)
	if err != nil {
		return
	}
	err = en.WriteInt64(z.SwapExactAmountIn)
	if err != nil {
		err = msgp.WrapError(err, "SwapExactAmountIn")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z Gas) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 1
	o = append(o, 0x91)
	o = msgp.AppendInt64(o, z.SwapExactAmountIn)
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
	if zb0001 != 1 {
		err = msgp.ArrayError{Wanted: 1, Got: zb0001}
		return
	}
	z.SwapExactAmountIn, bts, err = msgp.ReadInt64Bytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "SwapExactAmountIn")
		return
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z Gas) Msgsize() (s int) {
	s = 1 + msgp.Int64Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *Record) DecodeMsg(dc *msgp.Reader) (err error) {
	var zb0001 uint32
	zb0001, err = dc.ReadArrayHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	if zb0001 != 3 {
		err = msgp.ArrayError{Wanted: 3, Got: zb0001}
		return
	}
	z.Bound, err = dc.ReadBool()
	if err != nil {
		err = msgp.WrapError(err, "Bound")
		return
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "Denorm")
			return
		}
		z.Denorm = nil
	} else {
		{
			var zb0002 []byte
			zb0002, err = dc.ReadBytes(msgpencode.EncodeUint256(z.Denorm))
			if err != nil {
				err = msgp.WrapError(err, "Denorm")
				return
			}
			z.Denorm = msgpencode.DecodeUint256(zb0002)
		}
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "Balance")
			return
		}
		z.Balance = nil
	} else {
		{
			var zb0003 []byte
			zb0003, err = dc.ReadBytes(msgpencode.EncodeUint256(z.Balance))
			if err != nil {
				err = msgp.WrapError(err, "Balance")
				return
			}
			z.Balance = msgpencode.DecodeUint256(zb0003)
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *Record) EncodeMsg(en *msgp.Writer) (err error) {
	// array header, size 3
	err = en.Append(0x93)
	if err != nil {
		return
	}
	err = en.WriteBool(z.Bound)
	if err != nil {
		err = msgp.WrapError(err, "Bound")
		return
	}
	if z.Denorm == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = en.WriteBytes(msgpencode.EncodeUint256(z.Denorm))
		if err != nil {
			err = msgp.WrapError(err, "Denorm")
			return
		}
	}
	if z.Balance == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = en.WriteBytes(msgpencode.EncodeUint256(z.Balance))
		if err != nil {
			err = msgp.WrapError(err, "Balance")
			return
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *Record) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 3
	o = append(o, 0x93)
	o = msgp.AppendBool(o, z.Bound)
	if z.Denorm == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeUint256(z.Denorm))
	}
	if z.Balance == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeUint256(z.Balance))
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Record) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	if zb0001 != 3 {
		err = msgp.ArrayError{Wanted: 3, Got: zb0001}
		return
	}
	z.Bound, bts, err = msgp.ReadBoolBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "Bound")
		return
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.Denorm = nil
	} else {
		{
			var zb0002 []byte
			zb0002, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeUint256(z.Denorm))
			if err != nil {
				err = msgp.WrapError(err, "Denorm")
				return
			}
			z.Denorm = msgpencode.DecodeUint256(zb0002)
		}
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.Balance = nil
	} else {
		{
			var zb0003 []byte
			zb0003, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeUint256(z.Balance))
			if err != nil {
				err = msgp.WrapError(err, "Balance")
				return
			}
			z.Balance = msgpencode.DecodeUint256(zb0003)
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *Record) Msgsize() (s int) {
	s = 1 + msgp.BoolSize
	if z.Denorm == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeUint256(z.Denorm))
	}
	if z.Balance == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeUint256(z.Balance))
	}
	return
}