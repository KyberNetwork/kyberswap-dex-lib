package velocimeter

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpencode"
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *Pool) DecodeMsg(dc *msgp.Reader) (err error) {
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
	var zb0002 uint32
	zb0002, err = dc.ReadArrayHeader()
	if err != nil {
		err = msgp.WrapError(err, "Decimals")
		return
	}
	if cap(z.Decimals) >= int(zb0002) {
		z.Decimals = (z.Decimals)[:zb0002]
	} else {
		z.Decimals = make([]*big.Int, zb0002)
	}
	for za0001 := range z.Decimals {
		if dc.IsNil() {
			err = dc.ReadNil()
			if err != nil {
				err = msgp.WrapError(err, "Decimals", za0001)
				return
			}
			z.Decimals[za0001] = nil
		} else {
			{
				var zb0003 []byte
				zb0003, err = dc.ReadBytes(msgpencode.EncodeInt(z.Decimals[za0001]))
				if err != nil {
					err = msgp.WrapError(err, "Decimals", za0001)
					return
				}
				z.Decimals[za0001] = msgpencode.DecodeInt(zb0003)
			}
		}
	}
	z.stable, err = dc.ReadBool()
	if err != nil {
		err = msgp.WrapError(err, "stable")
		return
	}
	err = z.gas.DecodeMsg(dc)
	if err != nil {
		err = msgp.WrapError(err, "gas")
		return
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *Pool) EncodeMsg(en *msgp.Writer) (err error) {
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
	err = en.WriteArrayHeader(uint32(len(z.Decimals)))
	if err != nil {
		err = msgp.WrapError(err, "Decimals")
		return
	}
	for za0001 := range z.Decimals {
		if z.Decimals[za0001] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = en.WriteBytes(msgpencode.EncodeInt(z.Decimals[za0001]))
			if err != nil {
				err = msgp.WrapError(err, "Decimals", za0001)
				return
			}
		}
	}
	err = en.WriteBool(z.stable)
	if err != nil {
		err = msgp.WrapError(err, "stable")
		return
	}
	err = z.gas.EncodeMsg(en)
	if err != nil {
		err = msgp.WrapError(err, "gas")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *Pool) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 4
	o = append(o, 0x94)
	o, err = z.Pool.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "Pool")
		return
	}
	o = msgp.AppendArrayHeader(o, uint32(len(z.Decimals)))
	for za0001 := range z.Decimals {
		if z.Decimals[za0001] == nil {
			o = msgp.AppendNil(o)
		} else {
			o = msgp.AppendBytes(o, msgpencode.EncodeInt(z.Decimals[za0001]))
		}
	}
	o = msgp.AppendBool(o, z.stable)
	o, err = z.gas.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "gas")
		return
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Pool) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
	var zb0002 uint32
	zb0002, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "Decimals")
		return
	}
	if cap(z.Decimals) >= int(zb0002) {
		z.Decimals = (z.Decimals)[:zb0002]
	} else {
		z.Decimals = make([]*big.Int, zb0002)
	}
	for za0001 := range z.Decimals {
		if msgp.IsNil(bts) {
			bts, err = msgp.ReadNilBytes(bts)
			if err != nil {
				return
			}
			z.Decimals[za0001] = nil
		} else {
			{
				var zb0003 []byte
				zb0003, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.Decimals[za0001]))
				if err != nil {
					err = msgp.WrapError(err, "Decimals", za0001)
					return
				}
				z.Decimals[za0001] = msgpencode.DecodeInt(zb0003)
			}
		}
	}
	z.stable, bts, err = msgp.ReadBoolBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "stable")
		return
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
func (z *Pool) Msgsize() (s int) {
	s = 1 + z.Pool.Msgsize() + msgp.ArrayHeaderSize
	for za0001 := range z.Decimals {
		if z.Decimals[za0001] == nil {
			s += msgp.NilSize
		} else {
			s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(z.Decimals[za0001]))
		}
	}
	s += msgp.BoolSize + z.gas.Msgsize()
	return
}
