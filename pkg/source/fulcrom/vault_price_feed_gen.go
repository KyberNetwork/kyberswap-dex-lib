package fulcrom

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpencode"
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *VaultPriceFeed) DecodeMsg(dc *msgp.Reader) (err error) {
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
	var zb0002 uint32
	zb0002, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err, "MinPrices")
		return
	}
	if z.MinPrices == nil {
		z.MinPrices = make(map[string]*big.Int, zb0002)
	} else if len(z.MinPrices) > 0 {
		for key := range z.MinPrices {
			delete(z.MinPrices, key)
		}
	}
	var field []byte
	_ = field
	for zb0002 > 0 {
		zb0002--
		var za0001 string
		var za0002 *big.Int
		za0001, err = dc.ReadString()
		if err != nil {
			err = msgp.WrapError(err, "MinPrices")
			return
		}
		if dc.IsNil() {
			err = dc.ReadNil()
			if err != nil {
				err = msgp.WrapError(err, "MinPrices", za0001)
				return
			}
			za0002 = nil
		} else {
			{
				var zb0003 []byte
				zb0003, err = dc.ReadBytes(msgpencode.EncodeInt(za0002))
				if err != nil {
					err = msgp.WrapError(err, "MinPrices", za0001)
					return
				}
				za0002 = msgpencode.DecodeInt(zb0003)
			}
		}
		z.MinPrices[za0001] = za0002
	}
	var zb0004 uint32
	zb0004, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err, "MaxPrices")
		return
	}
	if z.MaxPrices == nil {
		z.MaxPrices = make(map[string]*big.Int, zb0004)
	} else if len(z.MaxPrices) > 0 {
		for key := range z.MaxPrices {
			delete(z.MaxPrices, key)
		}
	}
	for zb0004 > 0 {
		zb0004--
		var za0003 string
		var za0004 *big.Int
		za0003, err = dc.ReadString()
		if err != nil {
			err = msgp.WrapError(err, "MaxPrices")
			return
		}
		if dc.IsNil() {
			err = dc.ReadNil()
			if err != nil {
				err = msgp.WrapError(err, "MaxPrices", za0003)
				return
			}
			za0004 = nil
		} else {
			{
				var zb0005 []byte
				zb0005, err = dc.ReadBytes(msgpencode.EncodeInt(za0004))
				if err != nil {
					err = msgp.WrapError(err, "MaxPrices", za0003)
					return
				}
				za0004 = msgpencode.DecodeInt(zb0005)
			}
		}
		z.MaxPrices[za0003] = za0004
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *VaultPriceFeed) EncodeMsg(en *msgp.Writer) (err error) {
	// array header, size 2
	err = en.Append(0x92)
	if err != nil {
		return
	}
	err = en.WriteMapHeader(uint32(len(z.MinPrices)))
	if err != nil {
		err = msgp.WrapError(err, "MinPrices")
		return
	}
	for za0001, za0002 := range z.MinPrices {
		err = en.WriteString(za0001)
		if err != nil {
			err = msgp.WrapError(err, "MinPrices")
			return
		}
		if za0002 == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = en.WriteBytes(msgpencode.EncodeInt(za0002))
			if err != nil {
				err = msgp.WrapError(err, "MinPrices", za0001)
				return
			}
		}
	}
	err = en.WriteMapHeader(uint32(len(z.MaxPrices)))
	if err != nil {
		err = msgp.WrapError(err, "MaxPrices")
		return
	}
	for za0003, za0004 := range z.MaxPrices {
		err = en.WriteString(za0003)
		if err != nil {
			err = msgp.WrapError(err, "MaxPrices")
			return
		}
		if za0004 == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = en.WriteBytes(msgpencode.EncodeInt(za0004))
			if err != nil {
				err = msgp.WrapError(err, "MaxPrices", za0003)
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *VaultPriceFeed) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 2
	o = append(o, 0x92)
	o = msgp.AppendMapHeader(o, uint32(len(z.MinPrices)))
	for za0001, za0002 := range z.MinPrices {
		o = msgp.AppendString(o, za0001)
		if za0002 == nil {
			o = msgp.AppendNil(o)
		} else {
			o = msgp.AppendBytes(o, msgpencode.EncodeInt(za0002))
		}
	}
	o = msgp.AppendMapHeader(o, uint32(len(z.MaxPrices)))
	for za0003, za0004 := range z.MaxPrices {
		o = msgp.AppendString(o, za0003)
		if za0004 == nil {
			o = msgp.AppendNil(o)
		} else {
			o = msgp.AppendBytes(o, msgpencode.EncodeInt(za0004))
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *VaultPriceFeed) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
	var zb0002 uint32
	zb0002, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "MinPrices")
		return
	}
	if z.MinPrices == nil {
		z.MinPrices = make(map[string]*big.Int, zb0002)
	} else if len(z.MinPrices) > 0 {
		for key := range z.MinPrices {
			delete(z.MinPrices, key)
		}
	}
	var field []byte
	_ = field
	for zb0002 > 0 {
		var za0001 string
		var za0002 *big.Int
		zb0002--
		za0001, bts, err = msgp.ReadStringBytes(bts)
		if err != nil {
			err = msgp.WrapError(err, "MinPrices")
			return
		}
		if msgp.IsNil(bts) {
			bts, err = msgp.ReadNilBytes(bts)
			if err != nil {
				return
			}
			za0002 = nil
		} else {
			{
				var zb0003 []byte
				zb0003, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(za0002))
				if err != nil {
					err = msgp.WrapError(err, "MinPrices", za0001)
					return
				}
				za0002 = msgpencode.DecodeInt(zb0003)
			}
		}
		z.MinPrices[za0001] = za0002
	}
	var zb0004 uint32
	zb0004, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "MaxPrices")
		return
	}
	if z.MaxPrices == nil {
		z.MaxPrices = make(map[string]*big.Int, zb0004)
	} else if len(z.MaxPrices) > 0 {
		for key := range z.MaxPrices {
			delete(z.MaxPrices, key)
		}
	}
	for zb0004 > 0 {
		var za0003 string
		var za0004 *big.Int
		zb0004--
		za0003, bts, err = msgp.ReadStringBytes(bts)
		if err != nil {
			err = msgp.WrapError(err, "MaxPrices")
			return
		}
		if msgp.IsNil(bts) {
			bts, err = msgp.ReadNilBytes(bts)
			if err != nil {
				return
			}
			za0004 = nil
		} else {
			{
				var zb0005 []byte
				zb0005, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(za0004))
				if err != nil {
					err = msgp.WrapError(err, "MaxPrices", za0003)
					return
				}
				za0004 = msgpencode.DecodeInt(zb0005)
			}
		}
		z.MaxPrices[za0003] = za0004
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *VaultPriceFeed) Msgsize() (s int) {
	s = 1 + msgp.MapHeaderSize
	if z.MinPrices != nil {
		for za0001, za0002 := range z.MinPrices {
			_ = za0002
			s += msgp.StringPrefixSize + len(za0001)
			if za0002 == nil {
				s += msgp.NilSize
			} else {
				s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(za0002))
			}
		}
	}
	s += msgp.MapHeaderSize
	if z.MaxPrices != nil {
		for za0003, za0004 := range z.MaxPrices {
			_ = za0004
			s += msgp.StringPrefixSize + len(za0003)
			if za0004 == nil {
				s += msgp.NilSize
			} else {
				s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(za0004))
			}
		}
	}
	return
}
