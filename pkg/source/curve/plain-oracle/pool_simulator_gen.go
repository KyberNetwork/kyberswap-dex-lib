package plainoracle

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"math/big"

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
	z.Exchange, err = dc.ReadInt64()
	if err != nil {
		err = msgp.WrapError(err, "Exchange")
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
	err = en.WriteInt64(z.Exchange)
	if err != nil {
		err = msgp.WrapError(err, "Exchange")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z Gas) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 1
	o = append(o, 0x91)
	o = msgp.AppendInt64(o, z.Exchange)
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
	z.Exchange, bts, err = msgp.ReadInt64Bytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "Exchange")
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
func (z *Pool) DecodeMsg(dc *msgp.Reader) (err error) {
	var zb0001 uint32
	zb0001, err = dc.ReadArrayHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	if zb0001 != 12 {
		err = msgp.ArrayError{Wanted: 12, Got: zb0001}
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
		err = msgp.WrapError(err, "Multipliers")
		return
	}
	if cap(z.Multipliers) >= int(zb0002) {
		z.Multipliers = (z.Multipliers)[:zb0002]
	} else {
		z.Multipliers = make([]*big.Int, zb0002)
	}
	for za0001 := range z.Multipliers {
		if dc.IsNil() {
			err = dc.ReadNil()
			if err != nil {
				err = msgp.WrapError(err, "Multipliers", za0001)
				return
			}
			z.Multipliers[za0001] = nil
		} else {
			{
				var zb0003 []byte
				zb0003, err = dc.ReadBytes(msgpencode.EncodeInt(z.Multipliers[za0001]))
				if err != nil {
					err = msgp.WrapError(err, "Multipliers", za0001)
					return
				}
				z.Multipliers[za0001] = msgpencode.DecodeInt(zb0003)
			}
		}
	}
	var zb0004 uint32
	zb0004, err = dc.ReadArrayHeader()
	if err != nil {
		err = msgp.WrapError(err, "Rates")
		return
	}
	if cap(z.Rates) >= int(zb0004) {
		z.Rates = (z.Rates)[:zb0004]
	} else {
		z.Rates = make([]*big.Int, zb0004)
	}
	for za0002 := range z.Rates {
		if dc.IsNil() {
			err = dc.ReadNil()
			if err != nil {
				err = msgp.WrapError(err, "Rates", za0002)
				return
			}
			z.Rates[za0002] = nil
		} else {
			{
				var zb0005 []byte
				zb0005, err = dc.ReadBytes(msgpencode.EncodeInt(z.Rates[za0002]))
				if err != nil {
					err = msgp.WrapError(err, "Rates", za0002)
					return
				}
				z.Rates[za0002] = msgpencode.DecodeInt(zb0005)
			}
		}
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "InitialA")
			return
		}
		z.InitialA = nil
	} else {
		{
			var zb0006 []byte
			zb0006, err = dc.ReadBytes(msgpencode.EncodeInt(z.InitialA))
			if err != nil {
				err = msgp.WrapError(err, "InitialA")
				return
			}
			z.InitialA = msgpencode.DecodeInt(zb0006)
		}
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "FutureA")
			return
		}
		z.FutureA = nil
	} else {
		{
			var zb0007 []byte
			zb0007, err = dc.ReadBytes(msgpencode.EncodeInt(z.FutureA))
			if err != nil {
				err = msgp.WrapError(err, "FutureA")
				return
			}
			z.FutureA = msgpencode.DecodeInt(zb0007)
		}
	}
	z.InitialATime, err = dc.ReadInt64()
	if err != nil {
		err = msgp.WrapError(err, "InitialATime")
		return
	}
	z.FutureATime, err = dc.ReadInt64()
	if err != nil {
		err = msgp.WrapError(err, "FutureATime")
		return
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "AdminFee")
			return
		}
		z.AdminFee = nil
	} else {
		{
			var zb0008 []byte
			zb0008, err = dc.ReadBytes(msgpencode.EncodeInt(z.AdminFee))
			if err != nil {
				err = msgp.WrapError(err, "AdminFee")
				return
			}
			z.AdminFee = msgpencode.DecodeInt(zb0008)
		}
	}
	z.LpToken, err = dc.ReadString()
	if err != nil {
		err = msgp.WrapError(err, "LpToken")
		return
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "LpSupply")
			return
		}
		z.LpSupply = nil
	} else {
		{
			var zb0009 []byte
			zb0009, err = dc.ReadBytes(msgpencode.EncodeInt(z.LpSupply))
			if err != nil {
				err = msgp.WrapError(err, "LpSupply")
				return
			}
			z.LpSupply = msgpencode.DecodeInt(zb0009)
		}
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "APrecision")
			return
		}
		z.APrecision = nil
	} else {
		{
			var zb0010 []byte
			zb0010, err = dc.ReadBytes(msgpencode.EncodeInt(z.APrecision))
			if err != nil {
				err = msgp.WrapError(err, "APrecision")
				return
			}
			z.APrecision = msgpencode.DecodeInt(zb0010)
		}
	}
	var zb0011 uint32
	zb0011, err = dc.ReadArrayHeader()
	if err != nil {
		err = msgp.WrapError(err, "gas")
		return
	}
	if zb0011 != 1 {
		err = msgp.ArrayError{Wanted: 1, Got: zb0011}
		return
	}
	z.gas.Exchange, err = dc.ReadInt64()
	if err != nil {
		err = msgp.WrapError(err, "gas", "Exchange")
		return
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *Pool) EncodeMsg(en *msgp.Writer) (err error) {
	// array header, size 12
	err = en.Append(0x9c)
	if err != nil {
		return
	}
	err = z.Pool.EncodeMsg(en)
	if err != nil {
		err = msgp.WrapError(err, "Pool")
		return
	}
	err = en.WriteArrayHeader(uint32(len(z.Multipliers)))
	if err != nil {
		err = msgp.WrapError(err, "Multipliers")
		return
	}
	for za0001 := range z.Multipliers {
		if z.Multipliers[za0001] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = en.WriteBytes(msgpencode.EncodeInt(z.Multipliers[za0001]))
			if err != nil {
				err = msgp.WrapError(err, "Multipliers", za0001)
				return
			}
		}
	}
	err = en.WriteArrayHeader(uint32(len(z.Rates)))
	if err != nil {
		err = msgp.WrapError(err, "Rates")
		return
	}
	for za0002 := range z.Rates {
		if z.Rates[za0002] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = en.WriteBytes(msgpencode.EncodeInt(z.Rates[za0002]))
			if err != nil {
				err = msgp.WrapError(err, "Rates", za0002)
				return
			}
		}
	}
	if z.InitialA == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = en.WriteBytes(msgpencode.EncodeInt(z.InitialA))
		if err != nil {
			err = msgp.WrapError(err, "InitialA")
			return
		}
	}
	if z.FutureA == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = en.WriteBytes(msgpencode.EncodeInt(z.FutureA))
		if err != nil {
			err = msgp.WrapError(err, "FutureA")
			return
		}
	}
	err = en.WriteInt64(z.InitialATime)
	if err != nil {
		err = msgp.WrapError(err, "InitialATime")
		return
	}
	err = en.WriteInt64(z.FutureATime)
	if err != nil {
		err = msgp.WrapError(err, "FutureATime")
		return
	}
	if z.AdminFee == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = en.WriteBytes(msgpencode.EncodeInt(z.AdminFee))
		if err != nil {
			err = msgp.WrapError(err, "AdminFee")
			return
		}
	}
	err = en.WriteString(z.LpToken)
	if err != nil {
		err = msgp.WrapError(err, "LpToken")
		return
	}
	if z.LpSupply == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = en.WriteBytes(msgpencode.EncodeInt(z.LpSupply))
		if err != nil {
			err = msgp.WrapError(err, "LpSupply")
			return
		}
	}
	if z.APrecision == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = en.WriteBytes(msgpencode.EncodeInt(z.APrecision))
		if err != nil {
			err = msgp.WrapError(err, "APrecision")
			return
		}
	}
	// array header, size 1
	err = en.Append(0x91)
	if err != nil {
		return
	}
	err = en.WriteInt64(z.gas.Exchange)
	if err != nil {
		err = msgp.WrapError(err, "gas", "Exchange")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *Pool) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 12
	o = append(o, 0x9c)
	o, err = z.Pool.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "Pool")
		return
	}
	o = msgp.AppendArrayHeader(o, uint32(len(z.Multipliers)))
	for za0001 := range z.Multipliers {
		if z.Multipliers[za0001] == nil {
			o = msgp.AppendNil(o)
		} else {
			o = msgp.AppendBytes(o, msgpencode.EncodeInt(z.Multipliers[za0001]))
		}
	}
	o = msgp.AppendArrayHeader(o, uint32(len(z.Rates)))
	for za0002 := range z.Rates {
		if z.Rates[za0002] == nil {
			o = msgp.AppendNil(o)
		} else {
			o = msgp.AppendBytes(o, msgpencode.EncodeInt(z.Rates[za0002]))
		}
	}
	if z.InitialA == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeInt(z.InitialA))
	}
	if z.FutureA == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeInt(z.FutureA))
	}
	o = msgp.AppendInt64(o, z.InitialATime)
	o = msgp.AppendInt64(o, z.FutureATime)
	if z.AdminFee == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeInt(z.AdminFee))
	}
	o = msgp.AppendString(o, z.LpToken)
	if z.LpSupply == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeInt(z.LpSupply))
	}
	if z.APrecision == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeInt(z.APrecision))
	}
	// array header, size 1
	o = append(o, 0x91)
	o = msgp.AppendInt64(o, z.gas.Exchange)
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
	if zb0001 != 12 {
		err = msgp.ArrayError{Wanted: 12, Got: zb0001}
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
		err = msgp.WrapError(err, "Multipliers")
		return
	}
	if cap(z.Multipliers) >= int(zb0002) {
		z.Multipliers = (z.Multipliers)[:zb0002]
	} else {
		z.Multipliers = make([]*big.Int, zb0002)
	}
	for za0001 := range z.Multipliers {
		if msgp.IsNil(bts) {
			bts, err = msgp.ReadNilBytes(bts)
			if err != nil {
				return
			}
			z.Multipliers[za0001] = nil
		} else {
			{
				var zb0003 []byte
				zb0003, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.Multipliers[za0001]))
				if err != nil {
					err = msgp.WrapError(err, "Multipliers", za0001)
					return
				}
				z.Multipliers[za0001] = msgpencode.DecodeInt(zb0003)
			}
		}
	}
	var zb0004 uint32
	zb0004, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "Rates")
		return
	}
	if cap(z.Rates) >= int(zb0004) {
		z.Rates = (z.Rates)[:zb0004]
	} else {
		z.Rates = make([]*big.Int, zb0004)
	}
	for za0002 := range z.Rates {
		if msgp.IsNil(bts) {
			bts, err = msgp.ReadNilBytes(bts)
			if err != nil {
				return
			}
			z.Rates[za0002] = nil
		} else {
			{
				var zb0005 []byte
				zb0005, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.Rates[za0002]))
				if err != nil {
					err = msgp.WrapError(err, "Rates", za0002)
					return
				}
				z.Rates[za0002] = msgpencode.DecodeInt(zb0005)
			}
		}
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.InitialA = nil
	} else {
		{
			var zb0006 []byte
			zb0006, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.InitialA))
			if err != nil {
				err = msgp.WrapError(err, "InitialA")
				return
			}
			z.InitialA = msgpencode.DecodeInt(zb0006)
		}
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.FutureA = nil
	} else {
		{
			var zb0007 []byte
			zb0007, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.FutureA))
			if err != nil {
				err = msgp.WrapError(err, "FutureA")
				return
			}
			z.FutureA = msgpencode.DecodeInt(zb0007)
		}
	}
	z.InitialATime, bts, err = msgp.ReadInt64Bytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "InitialATime")
		return
	}
	z.FutureATime, bts, err = msgp.ReadInt64Bytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "FutureATime")
		return
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.AdminFee = nil
	} else {
		{
			var zb0008 []byte
			zb0008, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.AdminFee))
			if err != nil {
				err = msgp.WrapError(err, "AdminFee")
				return
			}
			z.AdminFee = msgpencode.DecodeInt(zb0008)
		}
	}
	z.LpToken, bts, err = msgp.ReadStringBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "LpToken")
		return
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.LpSupply = nil
	} else {
		{
			var zb0009 []byte
			zb0009, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.LpSupply))
			if err != nil {
				err = msgp.WrapError(err, "LpSupply")
				return
			}
			z.LpSupply = msgpencode.DecodeInt(zb0009)
		}
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.APrecision = nil
	} else {
		{
			var zb0010 []byte
			zb0010, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.APrecision))
			if err != nil {
				err = msgp.WrapError(err, "APrecision")
				return
			}
			z.APrecision = msgpencode.DecodeInt(zb0010)
		}
	}
	var zb0011 uint32
	zb0011, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "gas")
		return
	}
	if zb0011 != 1 {
		err = msgp.ArrayError{Wanted: 1, Got: zb0011}
		return
	}
	z.gas.Exchange, bts, err = msgp.ReadInt64Bytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "gas", "Exchange")
		return
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *Pool) Msgsize() (s int) {
	s = 1 + z.Pool.Msgsize() + msgp.ArrayHeaderSize
	for za0001 := range z.Multipliers {
		if z.Multipliers[za0001] == nil {
			s += msgp.NilSize
		} else {
			s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(z.Multipliers[za0001]))
		}
	}
	s += msgp.ArrayHeaderSize
	for za0002 := range z.Rates {
		if z.Rates[za0002] == nil {
			s += msgp.NilSize
		} else {
			s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(z.Rates[za0002]))
		}
	}
	if z.InitialA == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(z.InitialA))
	}
	if z.FutureA == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(z.FutureA))
	}
	s += msgp.Int64Size + msgp.Int64Size
	if z.AdminFee == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(z.AdminFee))
	}
	s += msgp.StringPrefixSize + len(z.LpToken)
	if z.LpSupply == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(z.LpSupply))
	}
	if z.APrecision == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(z.APrecision))
	}
	s += 1 + msgp.Int64Size
	return
}
