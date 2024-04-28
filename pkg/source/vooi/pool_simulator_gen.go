package vooi

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
	z.Swap, err = dc.ReadInt64()
	if err != nil {
		err = msgp.WrapError(err, "Swap")
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
	err = en.WriteInt64(z.Swap)
	if err != nil {
		err = msgp.WrapError(err, "Swap")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z Gas) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 1
	o = append(o, 0x91)
	o = msgp.AppendInt64(o, z.Swap)
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
	z.Swap, bts, err = msgp.ReadInt64Bytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "Swap")
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
func (z *PoolSimulator) DecodeMsg(dc *msgp.Reader) (err error) {
	var zb0001 uint32
	zb0001, err = dc.ReadArrayHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	if zb0001 != 7 {
		err = msgp.ArrayError{Wanted: 7, Got: zb0001}
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
			err = msgp.WrapError(err, "a")
			return
		}
		z.a = nil
	} else {
		{
			var zb0002 []byte
			zb0002, err = dc.ReadBytes(msgpencode.EncodeInt(z.a))
			if err != nil {
				err = msgp.WrapError(err, "a")
				return
			}
			z.a = msgpencode.DecodeInt(zb0002)
		}
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "lpFee")
			return
		}
		z.lpFee = nil
	} else {
		{
			var zb0003 []byte
			zb0003, err = dc.ReadBytes(msgpencode.EncodeInt(z.lpFee))
			if err != nil {
				err = msgp.WrapError(err, "lpFee")
				return
			}
			z.lpFee = msgpencode.DecodeInt(zb0003)
		}
	}
	z.paused, err = dc.ReadBool()
	if err != nil {
		err = msgp.WrapError(err, "paused")
		return
	}
	var zb0004 uint32
	zb0004, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err, "assetByToken")
		return
	}
	if z.assetByToken == nil {
		z.assetByToken = make(map[string]Asset, zb0004)
	} else if len(z.assetByToken) > 0 {
		for key := range z.assetByToken {
			delete(z.assetByToken, key)
		}
	}
	var field []byte
	_ = field
	for zb0004 > 0 {
		zb0004--
		var za0001 string
		var za0002 Asset
		za0001, err = dc.ReadString()
		if err != nil {
			err = msgp.WrapError(err, "assetByToken")
			return
		}
		err = za0002.DecodeMsg(dc)
		if err != nil {
			err = msgp.WrapError(err, "assetByToken", za0001)
			return
		}
		z.assetByToken[za0001] = za0002
	}
	var zb0005 uint32
	zb0005, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err, "indexByToken")
		return
	}
	if z.indexByToken == nil {
		z.indexByToken = make(map[string]int, zb0005)
	} else if len(z.indexByToken) > 0 {
		for key := range z.indexByToken {
			delete(z.indexByToken, key)
		}
	}
	for zb0005 > 0 {
		zb0005--
		var za0003 string
		var za0004 int
		za0003, err = dc.ReadString()
		if err != nil {
			err = msgp.WrapError(err, "indexByToken")
			return
		}
		za0004, err = dc.ReadInt()
		if err != nil {
			err = msgp.WrapError(err, "indexByToken", za0003)
			return
		}
		z.indexByToken[za0003] = za0004
	}
	var zb0006 uint32
	zb0006, err = dc.ReadArrayHeader()
	if err != nil {
		err = msgp.WrapError(err, "gas")
		return
	}
	if zb0006 != 1 {
		err = msgp.ArrayError{Wanted: 1, Got: zb0006}
		return
	}
	z.gas.Swap, err = dc.ReadInt64()
	if err != nil {
		err = msgp.WrapError(err, "gas", "Swap")
		return
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *PoolSimulator) EncodeMsg(en *msgp.Writer) (err error) {
	// array header, size 7
	err = en.Append(0x97)
	if err != nil {
		return
	}
	err = z.Pool.EncodeMsg(en)
	if err != nil {
		err = msgp.WrapError(err, "Pool")
		return
	}
	if z.a == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = en.WriteBytes(msgpencode.EncodeInt(z.a))
		if err != nil {
			err = msgp.WrapError(err, "a")
			return
		}
	}
	if z.lpFee == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = en.WriteBytes(msgpencode.EncodeInt(z.lpFee))
		if err != nil {
			err = msgp.WrapError(err, "lpFee")
			return
		}
	}
	err = en.WriteBool(z.paused)
	if err != nil {
		err = msgp.WrapError(err, "paused")
		return
	}
	err = en.WriteMapHeader(uint32(len(z.assetByToken)))
	if err != nil {
		err = msgp.WrapError(err, "assetByToken")
		return
	}
	for za0001, za0002 := range z.assetByToken {
		err = en.WriteString(za0001)
		if err != nil {
			err = msgp.WrapError(err, "assetByToken")
			return
		}
		err = za0002.EncodeMsg(en)
		if err != nil {
			err = msgp.WrapError(err, "assetByToken", za0001)
			return
		}
	}
	err = en.WriteMapHeader(uint32(len(z.indexByToken)))
	if err != nil {
		err = msgp.WrapError(err, "indexByToken")
		return
	}
	for za0003, za0004 := range z.indexByToken {
		err = en.WriteString(za0003)
		if err != nil {
			err = msgp.WrapError(err, "indexByToken")
			return
		}
		err = en.WriteInt(za0004)
		if err != nil {
			err = msgp.WrapError(err, "indexByToken", za0003)
			return
		}
	}
	// array header, size 1
	err = en.Append(0x91)
	if err != nil {
		return
	}
	err = en.WriteInt64(z.gas.Swap)
	if err != nil {
		err = msgp.WrapError(err, "gas", "Swap")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *PoolSimulator) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 7
	o = append(o, 0x97)
	o, err = z.Pool.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "Pool")
		return
	}
	if z.a == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeInt(z.a))
	}
	if z.lpFee == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeInt(z.lpFee))
	}
	o = msgp.AppendBool(o, z.paused)
	o = msgp.AppendMapHeader(o, uint32(len(z.assetByToken)))
	for za0001, za0002 := range z.assetByToken {
		o = msgp.AppendString(o, za0001)
		o, err = za0002.MarshalMsg(o)
		if err != nil {
			err = msgp.WrapError(err, "assetByToken", za0001)
			return
		}
	}
	o = msgp.AppendMapHeader(o, uint32(len(z.indexByToken)))
	for za0003, za0004 := range z.indexByToken {
		o = msgp.AppendString(o, za0003)
		o = msgp.AppendInt(o, za0004)
	}
	// array header, size 1
	o = append(o, 0x91)
	o = msgp.AppendInt64(o, z.gas.Swap)
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
	if zb0001 != 7 {
		err = msgp.ArrayError{Wanted: 7, Got: zb0001}
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
		z.a = nil
	} else {
		{
			var zb0002 []byte
			zb0002, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.a))
			if err != nil {
				err = msgp.WrapError(err, "a")
				return
			}
			z.a = msgpencode.DecodeInt(zb0002)
		}
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.lpFee = nil
	} else {
		{
			var zb0003 []byte
			zb0003, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.lpFee))
			if err != nil {
				err = msgp.WrapError(err, "lpFee")
				return
			}
			z.lpFee = msgpencode.DecodeInt(zb0003)
		}
	}
	z.paused, bts, err = msgp.ReadBoolBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "paused")
		return
	}
	var zb0004 uint32
	zb0004, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "assetByToken")
		return
	}
	if z.assetByToken == nil {
		z.assetByToken = make(map[string]Asset, zb0004)
	} else if len(z.assetByToken) > 0 {
		for key := range z.assetByToken {
			delete(z.assetByToken, key)
		}
	}
	var field []byte
	_ = field
	for zb0004 > 0 {
		var za0001 string
		var za0002 Asset
		zb0004--
		za0001, bts, err = msgp.ReadStringBytes(bts)
		if err != nil {
			err = msgp.WrapError(err, "assetByToken")
			return
		}
		bts, err = za0002.UnmarshalMsg(bts)
		if err != nil {
			err = msgp.WrapError(err, "assetByToken", za0001)
			return
		}
		z.assetByToken[za0001] = za0002
	}
	var zb0005 uint32
	zb0005, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "indexByToken")
		return
	}
	if z.indexByToken == nil {
		z.indexByToken = make(map[string]int, zb0005)
	} else if len(z.indexByToken) > 0 {
		for key := range z.indexByToken {
			delete(z.indexByToken, key)
		}
	}
	for zb0005 > 0 {
		var za0003 string
		var za0004 int
		zb0005--
		za0003, bts, err = msgp.ReadStringBytes(bts)
		if err != nil {
			err = msgp.WrapError(err, "indexByToken")
			return
		}
		za0004, bts, err = msgp.ReadIntBytes(bts)
		if err != nil {
			err = msgp.WrapError(err, "indexByToken", za0003)
			return
		}
		z.indexByToken[za0003] = za0004
	}
	var zb0006 uint32
	zb0006, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "gas")
		return
	}
	if zb0006 != 1 {
		err = msgp.ArrayError{Wanted: 1, Got: zb0006}
		return
	}
	z.gas.Swap, bts, err = msgp.ReadInt64Bytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "gas", "Swap")
		return
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *PoolSimulator) Msgsize() (s int) {
	s = 1 + z.Pool.Msgsize()
	if z.a == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(z.a))
	}
	if z.lpFee == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(z.lpFee))
	}
	s += msgp.BoolSize + msgp.MapHeaderSize
	if z.assetByToken != nil {
		for za0001, za0002 := range z.assetByToken {
			_ = za0002
			s += msgp.StringPrefixSize + len(za0001) + za0002.Msgsize()
		}
	}
	s += msgp.MapHeaderSize
	if z.indexByToken != nil {
		for za0003, za0004 := range z.indexByToken {
			_ = za0004
			s += msgp.StringPrefixSize + len(za0003) + msgp.IntSize
		}
	}
	s += 1 + msgp.Int64Size
	return
}
