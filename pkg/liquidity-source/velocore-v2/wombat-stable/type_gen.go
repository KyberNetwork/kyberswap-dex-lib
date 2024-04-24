package wombatstable

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *tokenInfo) DecodeMsg(dc *msgp.Reader) (err error) {
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
	z.IndexPlus1, err = dc.ReadUint8()
	if err != nil {
		err = msgp.WrapError(err, "IndexPlus1")
		return
	}
	z.Scale, err = dc.ReadUint8()
	if err != nil {
		err = msgp.WrapError(err, "Scale")
		return
	}
	{
		var zb0002 []byte
		zb0002, err = dc.ReadBytes((common.Address).Bytes(z.Gauge))
		if err != nil {
			err = msgp.WrapError(err, "Gauge")
			return
		}
		z.Gauge = common.BytesToAddress(zb0002)
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *tokenInfo) EncodeMsg(en *msgp.Writer) (err error) {
	// array header, size 3
	err = en.Append(0x93)
	if err != nil {
		return
	}
	err = en.WriteUint8(z.IndexPlus1)
	if err != nil {
		err = msgp.WrapError(err, "IndexPlus1")
		return
	}
	err = en.WriteUint8(z.Scale)
	if err != nil {
		err = msgp.WrapError(err, "Scale")
		return
	}
	err = en.WriteBytes((common.Address).Bytes(z.Gauge))
	if err != nil {
		err = msgp.WrapError(err, "Gauge")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *tokenInfo) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 3
	o = append(o, 0x93)
	o = msgp.AppendUint8(o, z.IndexPlus1)
	o = msgp.AppendUint8(o, z.Scale)
	o = msgp.AppendBytes(o, (common.Address).Bytes(z.Gauge))
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *tokenInfo) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
	z.IndexPlus1, bts, err = msgp.ReadUint8Bytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "IndexPlus1")
		return
	}
	z.Scale, bts, err = msgp.ReadUint8Bytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "Scale")
		return
	}
	{
		var zb0002 []byte
		zb0002, bts, err = msgp.ReadBytesBytes(bts, (common.Address).Bytes(z.Gauge))
		if err != nil {
			err = msgp.WrapError(err, "Gauge")
			return
		}
		z.Gauge = common.BytesToAddress(zb0002)
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *tokenInfo) Msgsize() (s int) {
	s = 1 + msgp.Uint8Size + msgp.Uint8Size + msgp.BytesPrefixSize + len((common.Address).Bytes(z.Gauge))
	return
}