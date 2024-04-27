package kyberpmm

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpencode"
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *Inventory) DecodeMsg(dc *msgp.Reader) (err error) {
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
	var zb0002 uint32
	zb0002, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err, "Balance")
		return
	}
	if z.Balance == nil {
		z.Balance = make(map[string]*big.Int, zb0002)
	} else if len(z.Balance) > 0 {
		for key := range z.Balance {
			delete(z.Balance, key)
		}
	}
	for zb0002 > 0 {
		zb0002--
		var za0001 string
		var za0002 *big.Int
		za0001, err = dc.ReadString()
		if err != nil {
			err = msgp.WrapError(err, "Balance")
			return
		}
		if dc.IsNil() {
			err = dc.ReadNil()
			if err != nil {
				err = msgp.WrapError(err, "Balance", za0001)
				return
			}
			za0002 = nil
		} else {
			{
				var zb0003 []byte
				zb0003, err = dc.ReadBytes(msgpencode.EncodeInt(za0002))
				if err != nil {
					err = msgp.WrapError(err, "Balance", za0001)
					return
				}
				za0002 = msgpencode.DecodeInt(zb0003)
			}
		}
		z.Balance[za0001] = za0002
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *Inventory) EncodeMsg(en *msgp.Writer) (err error) {
	// array header, size 1
	err = en.Append(0x91)
	if err != nil {
		return
	}
	err = en.WriteMapHeader(uint32(len(z.Balance)))
	if err != nil {
		err = msgp.WrapError(err, "Balance")
		return
	}
	for za0001, za0002 := range z.Balance {
		err = en.WriteString(za0001)
		if err != nil {
			err = msgp.WrapError(err, "Balance")
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
				err = msgp.WrapError(err, "Balance", za0001)
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *Inventory) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 1
	o = append(o, 0x91)
	o = msgp.AppendMapHeader(o, uint32(len(z.Balance)))
	for za0001, za0002 := range z.Balance {
		o = msgp.AppendString(o, za0001)
		if za0002 == nil {
			o = msgp.AppendNil(o)
		} else {
			o = msgp.AppendBytes(o, msgpencode.EncodeInt(za0002))
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Inventory) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
	var zb0002 uint32
	zb0002, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "Balance")
		return
	}
	if z.Balance == nil {
		z.Balance = make(map[string]*big.Int, zb0002)
	} else if len(z.Balance) > 0 {
		for key := range z.Balance {
			delete(z.Balance, key)
		}
	}
	for zb0002 > 0 {
		var za0001 string
		var za0002 *big.Int
		zb0002--
		za0001, bts, err = msgp.ReadStringBytes(bts)
		if err != nil {
			err = msgp.WrapError(err, "Balance")
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
					err = msgp.WrapError(err, "Balance", za0001)
					return
				}
				za0002 = msgpencode.DecodeInt(zb0003)
			}
		}
		z.Balance[za0001] = za0002
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *Inventory) Msgsize() (s int) {
	s = 1 + msgp.MapHeaderSize
	if z.Balance != nil {
		for za0001, za0002 := range z.Balance {
			_ = za0002
			s += msgp.StringPrefixSize + len(za0001)
			if za0002 == nil {
				s += msgp.NilSize
			} else {
				s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(za0002))
			}
		}
	}
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
	if zb0001 != 9 {
		err = msgp.ArrayError{Wanted: 9, Got: zb0001}
		return
	}
	err = z.Pool.DecodeMsg(dc)
	if err != nil {
		err = msgp.WrapError(err, "Pool")
		return
	}
	err = z.baseToken.DecodeMsg(dc)
	if err != nil {
		err = msgp.WrapError(err, "baseToken")
		return
	}
	err = z.quoteToken.DecodeMsg(dc)
	if err != nil {
		err = msgp.WrapError(err, "quoteToken")
		return
	}
	var zb0002 uint32
	zb0002, err = dc.ReadArrayHeader()
	if err != nil {
		err = msgp.WrapError(err, "baseToQuotePriceLevels")
		return
	}
	if cap(z.baseToQuotePriceLevels) >= int(zb0002) {
		z.baseToQuotePriceLevels = (z.baseToQuotePriceLevels)[:zb0002]
	} else {
		z.baseToQuotePriceLevels = make([]PriceLevel, zb0002)
	}
	for za0001 := range z.baseToQuotePriceLevels {
		err = z.baseToQuotePriceLevels[za0001].DecodeMsg(dc)
		if err != nil {
			err = msgp.WrapError(err, "baseToQuotePriceLevels", za0001)
			return
		}
	}
	var zb0003 uint32
	zb0003, err = dc.ReadArrayHeader()
	if err != nil {
		err = msgp.WrapError(err, "quoteToBasePriceLevels")
		return
	}
	if cap(z.quoteToBasePriceLevels) >= int(zb0003) {
		z.quoteToBasePriceLevels = (z.quoteToBasePriceLevels)[:zb0003]
	} else {
		z.quoteToBasePriceLevels = make([]PriceLevel, zb0003)
	}
	for za0002 := range z.quoteToBasePriceLevels {
		err = z.quoteToBasePriceLevels[za0002].DecodeMsg(dc)
		if err != nil {
			err = msgp.WrapError(err, "quoteToBasePriceLevels", za0002)
			return
		}
	}
	err = z.gas.DecodeMsg(dc)
	if err != nil {
		err = msgp.WrapError(err, "gas")
		return
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "QuoteBalance")
			return
		}
		z.QuoteBalance = nil
	} else {
		{
			var zb0004 []byte
			zb0004, err = dc.ReadBytes(msgpencode.EncodeInt(z.QuoteBalance))
			if err != nil {
				err = msgp.WrapError(err, "QuoteBalance")
				return
			}
			z.QuoteBalance = msgpencode.DecodeInt(zb0004)
		}
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "BaseBalance")
			return
		}
		z.BaseBalance = nil
	} else {
		{
			var zb0005 []byte
			zb0005, err = dc.ReadBytes(msgpencode.EncodeInt(z.BaseBalance))
			if err != nil {
				err = msgp.WrapError(err, "BaseBalance")
				return
			}
			z.BaseBalance = msgpencode.DecodeInt(zb0005)
		}
	}
	z.timestamp, err = dc.ReadInt64()
	if err != nil {
		err = msgp.WrapError(err, "timestamp")
		return
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *PoolSimulator) EncodeMsg(en *msgp.Writer) (err error) {
	// array header, size 9
	err = en.Append(0x99)
	if err != nil {
		return
	}
	err = z.Pool.EncodeMsg(en)
	if err != nil {
		err = msgp.WrapError(err, "Pool")
		return
	}
	err = z.baseToken.EncodeMsg(en)
	if err != nil {
		err = msgp.WrapError(err, "baseToken")
		return
	}
	err = z.quoteToken.EncodeMsg(en)
	if err != nil {
		err = msgp.WrapError(err, "quoteToken")
		return
	}
	err = en.WriteArrayHeader(uint32(len(z.baseToQuotePriceLevels)))
	if err != nil {
		err = msgp.WrapError(err, "baseToQuotePriceLevels")
		return
	}
	for za0001 := range z.baseToQuotePriceLevels {
		err = z.baseToQuotePriceLevels[za0001].EncodeMsg(en)
		if err != nil {
			err = msgp.WrapError(err, "baseToQuotePriceLevels", za0001)
			return
		}
	}
	err = en.WriteArrayHeader(uint32(len(z.quoteToBasePriceLevels)))
	if err != nil {
		err = msgp.WrapError(err, "quoteToBasePriceLevels")
		return
	}
	for za0002 := range z.quoteToBasePriceLevels {
		err = z.quoteToBasePriceLevels[za0002].EncodeMsg(en)
		if err != nil {
			err = msgp.WrapError(err, "quoteToBasePriceLevels", za0002)
			return
		}
	}
	err = z.gas.EncodeMsg(en)
	if err != nil {
		err = msgp.WrapError(err, "gas")
		return
	}
	if z.QuoteBalance == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = en.WriteBytes(msgpencode.EncodeInt(z.QuoteBalance))
		if err != nil {
			err = msgp.WrapError(err, "QuoteBalance")
			return
		}
	}
	if z.BaseBalance == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = en.WriteBytes(msgpencode.EncodeInt(z.BaseBalance))
		if err != nil {
			err = msgp.WrapError(err, "BaseBalance")
			return
		}
	}
	err = en.WriteInt64(z.timestamp)
	if err != nil {
		err = msgp.WrapError(err, "timestamp")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *PoolSimulator) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 9
	o = append(o, 0x99)
	o, err = z.Pool.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "Pool")
		return
	}
	o, err = z.baseToken.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "baseToken")
		return
	}
	o, err = z.quoteToken.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "quoteToken")
		return
	}
	o = msgp.AppendArrayHeader(o, uint32(len(z.baseToQuotePriceLevels)))
	for za0001 := range z.baseToQuotePriceLevels {
		o, err = z.baseToQuotePriceLevels[za0001].MarshalMsg(o)
		if err != nil {
			err = msgp.WrapError(err, "baseToQuotePriceLevels", za0001)
			return
		}
	}
	o = msgp.AppendArrayHeader(o, uint32(len(z.quoteToBasePriceLevels)))
	for za0002 := range z.quoteToBasePriceLevels {
		o, err = z.quoteToBasePriceLevels[za0002].MarshalMsg(o)
		if err != nil {
			err = msgp.WrapError(err, "quoteToBasePriceLevels", za0002)
			return
		}
	}
	o, err = z.gas.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "gas")
		return
	}
	if z.QuoteBalance == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeInt(z.QuoteBalance))
	}
	if z.BaseBalance == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeInt(z.BaseBalance))
	}
	o = msgp.AppendInt64(o, z.timestamp)
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
	if zb0001 != 9 {
		err = msgp.ArrayError{Wanted: 9, Got: zb0001}
		return
	}
	bts, err = z.Pool.UnmarshalMsg(bts)
	if err != nil {
		err = msgp.WrapError(err, "Pool")
		return
	}
	bts, err = z.baseToken.UnmarshalMsg(bts)
	if err != nil {
		err = msgp.WrapError(err, "baseToken")
		return
	}
	bts, err = z.quoteToken.UnmarshalMsg(bts)
	if err != nil {
		err = msgp.WrapError(err, "quoteToken")
		return
	}
	var zb0002 uint32
	zb0002, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "baseToQuotePriceLevels")
		return
	}
	if cap(z.baseToQuotePriceLevels) >= int(zb0002) {
		z.baseToQuotePriceLevels = (z.baseToQuotePriceLevels)[:zb0002]
	} else {
		z.baseToQuotePriceLevels = make([]PriceLevel, zb0002)
	}
	for za0001 := range z.baseToQuotePriceLevels {
		bts, err = z.baseToQuotePriceLevels[za0001].UnmarshalMsg(bts)
		if err != nil {
			err = msgp.WrapError(err, "baseToQuotePriceLevels", za0001)
			return
		}
	}
	var zb0003 uint32
	zb0003, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "quoteToBasePriceLevels")
		return
	}
	if cap(z.quoteToBasePriceLevels) >= int(zb0003) {
		z.quoteToBasePriceLevels = (z.quoteToBasePriceLevels)[:zb0003]
	} else {
		z.quoteToBasePriceLevels = make([]PriceLevel, zb0003)
	}
	for za0002 := range z.quoteToBasePriceLevels {
		bts, err = z.quoteToBasePriceLevels[za0002].UnmarshalMsg(bts)
		if err != nil {
			err = msgp.WrapError(err, "quoteToBasePriceLevels", za0002)
			return
		}
	}
	bts, err = z.gas.UnmarshalMsg(bts)
	if err != nil {
		err = msgp.WrapError(err, "gas")
		return
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.QuoteBalance = nil
	} else {
		{
			var zb0004 []byte
			zb0004, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.QuoteBalance))
			if err != nil {
				err = msgp.WrapError(err, "QuoteBalance")
				return
			}
			z.QuoteBalance = msgpencode.DecodeInt(zb0004)
		}
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.BaseBalance = nil
	} else {
		{
			var zb0005 []byte
			zb0005, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.BaseBalance))
			if err != nil {
				err = msgp.WrapError(err, "BaseBalance")
				return
			}
			z.BaseBalance = msgpencode.DecodeInt(zb0005)
		}
	}
	z.timestamp, bts, err = msgp.ReadInt64Bytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "timestamp")
		return
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *PoolSimulator) Msgsize() (s int) {
	s = 1 + z.Pool.Msgsize() + z.baseToken.Msgsize() + z.quoteToken.Msgsize() + msgp.ArrayHeaderSize
	for za0001 := range z.baseToQuotePriceLevels {
		s += z.baseToQuotePriceLevels[za0001].Msgsize()
	}
	s += msgp.ArrayHeaderSize
	for za0002 := range z.quoteToBasePriceLevels {
		s += z.quoteToBasePriceLevels[za0002].Msgsize()
	}
	s += z.gas.Msgsize()
	if z.QuoteBalance == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(z.QuoteBalance))
	}
	if z.BaseBalance == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(z.BaseBalance))
	}
	s += msgp.Int64Size
	return
}
