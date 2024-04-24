package limitorder

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
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
	if zb0001 != 6 {
		err = msgp.ArrayError{Wanted: 6, Got: zb0001}
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
		err = msgp.WrapError(err, "tokens")
		return
	}
	if cap(z.tokens) >= int(zb0002) {
		z.tokens = (z.tokens)[:zb0002]
	} else {
		z.tokens = make([]*entity.PoolToken, zb0002)
	}
	for za0001 := range z.tokens {
		if dc.IsNil() {
			err = dc.ReadNil()
			if err != nil {
				err = msgp.WrapError(err, "tokens", za0001)
				return
			}
			z.tokens[za0001] = nil
		} else {
			if z.tokens[za0001] == nil {
				z.tokens[za0001] = new(entity.PoolToken)
			}
			err = z.tokens[za0001].DecodeMsg(dc)
			if err != nil {
				err = msgp.WrapError(err, "tokens", za0001)
				return
			}
		}
	}
	var zb0003 uint32
	zb0003, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err, "ordersMapping")
		return
	}
	if z.ordersMapping == nil {
		z.ordersMapping = make(map[string]*order, zb0003)
	} else if len(z.ordersMapping) > 0 {
		for key := range z.ordersMapping {
			delete(z.ordersMapping, key)
		}
	}
	for zb0003 > 0 {
		zb0003--
		var za0002 string
		var za0003 *order
		za0002, err = dc.ReadString()
		if err != nil {
			err = msgp.WrapError(err, "ordersMapping")
			return
		}
		if dc.IsNil() {
			err = dc.ReadNil()
			if err != nil {
				err = msgp.WrapError(err, "ordersMapping", za0002)
				return
			}
			za0003 = nil
		} else {
			if za0003 == nil {
				za0003 = new(order)
			}
			err = za0003.DecodeMsg(dc)
			if err != nil {
				err = msgp.WrapError(err, "ordersMapping", za0002)
				return
			}
		}
		z.ordersMapping[za0002] = za0003
	}
	var zb0004 uint32
	zb0004, err = dc.ReadArrayHeader()
	if err != nil {
		err = msgp.WrapError(err, "sellOrderIDs")
		return
	}
	if cap(z.sellOrderIDs) >= int(zb0004) {
		z.sellOrderIDs = (z.sellOrderIDs)[:zb0004]
	} else {
		z.sellOrderIDs = make([]int64, zb0004)
	}
	for za0004 := range z.sellOrderIDs {
		z.sellOrderIDs[za0004], err = dc.ReadInt64()
		if err != nil {
			err = msgp.WrapError(err, "sellOrderIDs", za0004)
			return
		}
	}
	var zb0005 uint32
	zb0005, err = dc.ReadArrayHeader()
	if err != nil {
		err = msgp.WrapError(err, "buyOrderIDs")
		return
	}
	if cap(z.buyOrderIDs) >= int(zb0005) {
		z.buyOrderIDs = (z.buyOrderIDs)[:zb0005]
	} else {
		z.buyOrderIDs = make([]int64, zb0005)
	}
	for za0005 := range z.buyOrderIDs {
		z.buyOrderIDs[za0005], err = dc.ReadInt64()
		if err != nil {
			err = msgp.WrapError(err, "buyOrderIDs", za0005)
			return
		}
	}
	z.contractAddress, err = dc.ReadString()
	if err != nil {
		err = msgp.WrapError(err, "contractAddress")
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
	err = en.WriteArrayHeader(uint32(len(z.tokens)))
	if err != nil {
		err = msgp.WrapError(err, "tokens")
		return
	}
	for za0001 := range z.tokens {
		if z.tokens[za0001] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.tokens[za0001].EncodeMsg(en)
			if err != nil {
				err = msgp.WrapError(err, "tokens", za0001)
				return
			}
		}
	}
	err = en.WriteMapHeader(uint32(len(z.ordersMapping)))
	if err != nil {
		err = msgp.WrapError(err, "ordersMapping")
		return
	}
	for za0002, za0003 := range z.ordersMapping {
		err = en.WriteString(za0002)
		if err != nil {
			err = msgp.WrapError(err, "ordersMapping")
			return
		}
		if za0003 == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = za0003.EncodeMsg(en)
			if err != nil {
				err = msgp.WrapError(err, "ordersMapping", za0002)
				return
			}
		}
	}
	err = en.WriteArrayHeader(uint32(len(z.sellOrderIDs)))
	if err != nil {
		err = msgp.WrapError(err, "sellOrderIDs")
		return
	}
	for za0004 := range z.sellOrderIDs {
		err = en.WriteInt64(z.sellOrderIDs[za0004])
		if err != nil {
			err = msgp.WrapError(err, "sellOrderIDs", za0004)
			return
		}
	}
	err = en.WriteArrayHeader(uint32(len(z.buyOrderIDs)))
	if err != nil {
		err = msgp.WrapError(err, "buyOrderIDs")
		return
	}
	for za0005 := range z.buyOrderIDs {
		err = en.WriteInt64(z.buyOrderIDs[za0005])
		if err != nil {
			err = msgp.WrapError(err, "buyOrderIDs", za0005)
			return
		}
	}
	err = en.WriteString(z.contractAddress)
	if err != nil {
		err = msgp.WrapError(err, "contractAddress")
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
	o = msgp.AppendArrayHeader(o, uint32(len(z.tokens)))
	for za0001 := range z.tokens {
		if z.tokens[za0001] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.tokens[za0001].MarshalMsg(o)
			if err != nil {
				err = msgp.WrapError(err, "tokens", za0001)
				return
			}
		}
	}
	o = msgp.AppendMapHeader(o, uint32(len(z.ordersMapping)))
	for za0002, za0003 := range z.ordersMapping {
		o = msgp.AppendString(o, za0002)
		if za0003 == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = za0003.MarshalMsg(o)
			if err != nil {
				err = msgp.WrapError(err, "ordersMapping", za0002)
				return
			}
		}
	}
	o = msgp.AppendArrayHeader(o, uint32(len(z.sellOrderIDs)))
	for za0004 := range z.sellOrderIDs {
		o = msgp.AppendInt64(o, z.sellOrderIDs[za0004])
	}
	o = msgp.AppendArrayHeader(o, uint32(len(z.buyOrderIDs)))
	for za0005 := range z.buyOrderIDs {
		o = msgp.AppendInt64(o, z.buyOrderIDs[za0005])
	}
	o = msgp.AppendString(o, z.contractAddress)
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
	var zb0002 uint32
	zb0002, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "tokens")
		return
	}
	if cap(z.tokens) >= int(zb0002) {
		z.tokens = (z.tokens)[:zb0002]
	} else {
		z.tokens = make([]*entity.PoolToken, zb0002)
	}
	for za0001 := range z.tokens {
		if msgp.IsNil(bts) {
			bts, err = msgp.ReadNilBytes(bts)
			if err != nil {
				return
			}
			z.tokens[za0001] = nil
		} else {
			if z.tokens[za0001] == nil {
				z.tokens[za0001] = new(entity.PoolToken)
			}
			bts, err = z.tokens[za0001].UnmarshalMsg(bts)
			if err != nil {
				err = msgp.WrapError(err, "tokens", za0001)
				return
			}
		}
	}
	var zb0003 uint32
	zb0003, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "ordersMapping")
		return
	}
	if z.ordersMapping == nil {
		z.ordersMapping = make(map[string]*order, zb0003)
	} else if len(z.ordersMapping) > 0 {
		for key := range z.ordersMapping {
			delete(z.ordersMapping, key)
		}
	}
	for zb0003 > 0 {
		var za0002 string
		var za0003 *order
		zb0003--
		za0002, bts, err = msgp.ReadStringBytes(bts)
		if err != nil {
			err = msgp.WrapError(err, "ordersMapping")
			return
		}
		if msgp.IsNil(bts) {
			bts, err = msgp.ReadNilBytes(bts)
			if err != nil {
				return
			}
			za0003 = nil
		} else {
			if za0003 == nil {
				za0003 = new(order)
			}
			bts, err = za0003.UnmarshalMsg(bts)
			if err != nil {
				err = msgp.WrapError(err, "ordersMapping", za0002)
				return
			}
		}
		z.ordersMapping[za0002] = za0003
	}
	var zb0004 uint32
	zb0004, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "sellOrderIDs")
		return
	}
	if cap(z.sellOrderIDs) >= int(zb0004) {
		z.sellOrderIDs = (z.sellOrderIDs)[:zb0004]
	} else {
		z.sellOrderIDs = make([]int64, zb0004)
	}
	for za0004 := range z.sellOrderIDs {
		z.sellOrderIDs[za0004], bts, err = msgp.ReadInt64Bytes(bts)
		if err != nil {
			err = msgp.WrapError(err, "sellOrderIDs", za0004)
			return
		}
	}
	var zb0005 uint32
	zb0005, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "buyOrderIDs")
		return
	}
	if cap(z.buyOrderIDs) >= int(zb0005) {
		z.buyOrderIDs = (z.buyOrderIDs)[:zb0005]
	} else {
		z.buyOrderIDs = make([]int64, zb0005)
	}
	for za0005 := range z.buyOrderIDs {
		z.buyOrderIDs[za0005], bts, err = msgp.ReadInt64Bytes(bts)
		if err != nil {
			err = msgp.WrapError(err, "buyOrderIDs", za0005)
			return
		}
	}
	z.contractAddress, bts, err = msgp.ReadStringBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "contractAddress")
		return
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *PoolSimulator) Msgsize() (s int) {
	s = 1 + z.Pool.Msgsize() + msgp.ArrayHeaderSize
	for za0001 := range z.tokens {
		if z.tokens[za0001] == nil {
			s += msgp.NilSize
		} else {
			s += z.tokens[za0001].Msgsize()
		}
	}
	s += msgp.MapHeaderSize
	if z.ordersMapping != nil {
		for za0002, za0003 := range z.ordersMapping {
			_ = za0003
			s += msgp.StringPrefixSize + len(za0002)
			if za0003 == nil {
				s += msgp.NilSize
			} else {
				s += za0003.Msgsize()
			}
		}
	}
	s += msgp.ArrayHeaderSize + (len(z.sellOrderIDs) * (msgp.Int64Size)) + msgp.ArrayHeaderSize + (len(z.buyOrderIDs) * (msgp.Int64Size)) + msgp.StringPrefixSize + len(z.contractAddress)
	return
}
