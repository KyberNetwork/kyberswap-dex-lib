package gmxglp

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpencode"
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *USDG) DecodeMsg(dc *msgp.Reader) (err error) {
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
	z.Address, err = dc.ReadString()
	if err != nil {
		err = msgp.WrapError(err, "Address")
		return
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "TotalSupply")
			return
		}
		z.TotalSupply = nil
	} else {
		{
			var zb0002 []byte
			zb0002, err = dc.ReadBytes(msgpencode.EncodeInt(z.TotalSupply))
			if err != nil {
				err = msgp.WrapError(err, "TotalSupply")
				return
			}
			z.TotalSupply = msgpencode.DecodeInt(zb0002)
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *USDG) EncodeMsg(en *msgp.Writer) (err error) {
	// array header, size 2
	err = en.Append(0x92)
	if err != nil {
		return
	}
	err = en.WriteString(z.Address)
	if err != nil {
		err = msgp.WrapError(err, "Address")
		return
	}
	if z.TotalSupply == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = en.WriteBytes(msgpencode.EncodeInt(z.TotalSupply))
		if err != nil {
			err = msgp.WrapError(err, "TotalSupply")
			return
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *USDG) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 2
	o = append(o, 0x92)
	o = msgp.AppendString(o, z.Address)
	if z.TotalSupply == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeInt(z.TotalSupply))
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *USDG) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
	z.Address, bts, err = msgp.ReadStringBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "Address")
		return
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.TotalSupply = nil
	} else {
		{
			var zb0002 []byte
			zb0002, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.TotalSupply))
			if err != nil {
				err = msgp.WrapError(err, "TotalSupply")
				return
			}
			z.TotalSupply = msgpencode.DecodeInt(zb0002)
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *USDG) Msgsize() (s int) {
	s = 1 + msgp.StringPrefixSize + len(z.Address)
	if z.TotalSupply == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(z.TotalSupply))
	}
	return
}
