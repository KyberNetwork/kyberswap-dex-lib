package rsweth

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpencode"
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
	if zb0001 != 4 {
		err = msgp.ArrayError{Wanted: 4, Got: zb0001}
		return
	}
	err = z.Pool.DecodeMsg(dc)
	if err != nil {
		err = msgp.WrapError(err, "Pool")
		return
	}
	z.paused, err = dc.ReadBool()
	if err != nil {
		err = msgp.WrapError(err, "paused")
		return
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "ethToRswETHRate")
			return
		}
		z.ethToRswETHRate = nil
	} else {
		{
			var zb0002 []byte
			zb0002, err = dc.ReadBytes(msgpencode.EncodeInt(z.ethToRswETHRate))
			if err != nil {
				err = msgp.WrapError(err, "ethToRswETHRate")
				return
			}
			z.ethToRswETHRate = msgpencode.DecodeInt(zb0002)
		}
	}
	err = z.gas.DecodeMsg(dc)
	if err != nil {
		err = msgp.WrapError(err, "gas")
		return
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *PoolSimulator) EncodeMsg(en *msgp.Writer) (err error) {
	// array header, size 4
	err = en.Append(0x94)
	if err != nil {
		return
	}
	err = z.Pool.EncodeMsg(en)
	if err != nil {
		err = msgp.WrapError(err, "Pool")
		return
	}
	err = en.WriteBool(z.paused)
	if err != nil {
		err = msgp.WrapError(err, "paused")
		return
	}
	if z.ethToRswETHRate == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = en.WriteBytes(msgpencode.EncodeInt(z.ethToRswETHRate))
		if err != nil {
			err = msgp.WrapError(err, "ethToRswETHRate")
			return
		}
	}
	err = z.gas.EncodeMsg(en)
	if err != nil {
		err = msgp.WrapError(err, "gas")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *PoolSimulator) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 4
	o = append(o, 0x94)
	o, err = z.Pool.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "Pool")
		return
	}
	o = msgp.AppendBool(o, z.paused)
	if z.ethToRswETHRate == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeInt(z.ethToRswETHRate))
	}
	o, err = z.gas.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "gas")
		return
	}
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
	if zb0001 != 4 {
		err = msgp.ArrayError{Wanted: 4, Got: zb0001}
		return
	}
	bts, err = z.Pool.UnmarshalMsg(bts)
	if err != nil {
		err = msgp.WrapError(err, "Pool")
		return
	}
	z.paused, bts, err = msgp.ReadBoolBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "paused")
		return
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.ethToRswETHRate = nil
	} else {
		{
			var zb0002 []byte
			zb0002, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.ethToRswETHRate))
			if err != nil {
				err = msgp.WrapError(err, "ethToRswETHRate")
				return
			}
			z.ethToRswETHRate = msgpencode.DecodeInt(zb0002)
		}
	}
	bts, err = z.gas.UnmarshalMsg(bts)
	if err != nil {
		err = msgp.WrapError(err, "gas")
		return
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *PoolSimulator) Msgsize() (s int) {
	s = 1 + z.Pool.Msgsize() + msgp.BoolSize
	if z.ethToRswETHRate == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(z.ethToRswETHRate))
	}
	s += z.gas.Msgsize()
	return
}
