package balancerv1

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpencode"
	"github.com/holiman/uint256"
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
	if zb0001 != 7 {
		err = msgp.ArrayError{Wanted: 7, Got: zb0001}
		return
	}
	err = z.Pool.DecodeMsg(dc)
	if err != nil {
		err = msgp.WrapError(err, "Pool")
		return
	}
	var zb0002 uint32
	zb0002, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err, "records")
		return
	}
	if z.records == nil {
		z.records = make(map[string]Record, zb0002)
	} else if len(z.records) > 0 {
		for key := range z.records {
			delete(z.records, key)
		}
	}
	var field []byte
	_ = field
	for zb0002 > 0 {
		zb0002--
		var za0001 string
		var za0002 Record
		za0001, err = dc.ReadString()
		if err != nil {
			err = msgp.WrapError(err, "records")
			return
		}
		err = za0002.DecodeMsg(dc)
		if err != nil {
			err = msgp.WrapError(err, "records", za0001)
			return
		}
		z.records[za0001] = za0002
	}
	z.publicSwap, err = dc.ReadBool()
	if err != nil {
		err = msgp.WrapError(err, "publicSwap")
		return
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "swapFee")
			return
		}
		z.swapFee = nil
	} else {
		{
			var zb0003 []byte
			zb0003, err = dc.ReadBytes(msgpencode.EncodeUint256(z.swapFee))
			if err != nil {
				err = msgp.WrapError(err, "swapFee")
				return
			}
			z.swapFee = msgpencode.DecodeUint256(zb0003)
		}
	}
	var zb0004 uint32
	zb0004, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err, "totalAmountsIn")
		return
	}
	if z.totalAmountsIn == nil {
		z.totalAmountsIn = make(map[string]*uint256.Int, zb0004)
	} else if len(z.totalAmountsIn) > 0 {
		for key := range z.totalAmountsIn {
			delete(z.totalAmountsIn, key)
		}
	}
	for zb0004 > 0 {
		zb0004--
		var za0003 string
		var za0004 *uint256.Int
		za0003, err = dc.ReadString()
		if err != nil {
			err = msgp.WrapError(err, "totalAmountsIn")
			return
		}
		if dc.IsNil() {
			err = dc.ReadNil()
			if err != nil {
				err = msgp.WrapError(err, "totalAmountsIn", za0003)
				return
			}
			za0004 = nil
		} else {
			{
				var zb0005 []byte
				zb0005, err = dc.ReadBytes(msgpencode.EncodeUint256(za0004))
				if err != nil {
					err = msgp.WrapError(err, "totalAmountsIn", za0003)
					return
				}
				za0004 = msgpencode.DecodeUint256(zb0005)
			}
		}
		z.totalAmountsIn[za0003] = za0004
	}
	var zb0006 uint32
	zb0006, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err, "maxTotalAmountsIn")
		return
	}
	if z.maxTotalAmountsIn == nil {
		z.maxTotalAmountsIn = make(map[string]*uint256.Int, zb0006)
	} else if len(z.maxTotalAmountsIn) > 0 {
		for key := range z.maxTotalAmountsIn {
			delete(z.maxTotalAmountsIn, key)
		}
	}
	for zb0006 > 0 {
		zb0006--
		var za0005 string
		var za0006 *uint256.Int
		za0005, err = dc.ReadString()
		if err != nil {
			err = msgp.WrapError(err, "maxTotalAmountsIn")
			return
		}
		if dc.IsNil() {
			err = dc.ReadNil()
			if err != nil {
				err = msgp.WrapError(err, "maxTotalAmountsIn", za0005)
				return
			}
			za0006 = nil
		} else {
			{
				var zb0007 []byte
				zb0007, err = dc.ReadBytes(msgpencode.EncodeUint256(za0006))
				if err != nil {
					err = msgp.WrapError(err, "maxTotalAmountsIn", za0005)
					return
				}
				za0006 = msgpencode.DecodeUint256(zb0007)
			}
		}
		z.maxTotalAmountsIn[za0005] = za0006
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
	err = en.WriteMapHeader(uint32(len(z.records)))
	if err != nil {
		err = msgp.WrapError(err, "records")
		return
	}
	for za0001, za0002 := range z.records {
		err = en.WriteString(za0001)
		if err != nil {
			err = msgp.WrapError(err, "records")
			return
		}
		err = za0002.EncodeMsg(en)
		if err != nil {
			err = msgp.WrapError(err, "records", za0001)
			return
		}
	}
	err = en.WriteBool(z.publicSwap)
	if err != nil {
		err = msgp.WrapError(err, "publicSwap")
		return
	}
	if z.swapFee == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = en.WriteBytes(msgpencode.EncodeUint256(z.swapFee))
		if err != nil {
			err = msgp.WrapError(err, "swapFee")
			return
		}
	}
	err = en.WriteMapHeader(uint32(len(z.totalAmountsIn)))
	if err != nil {
		err = msgp.WrapError(err, "totalAmountsIn")
		return
	}
	for za0003, za0004 := range z.totalAmountsIn {
		err = en.WriteString(za0003)
		if err != nil {
			err = msgp.WrapError(err, "totalAmountsIn")
			return
		}
		if za0004 == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = en.WriteBytes(msgpencode.EncodeUint256(za0004))
			if err != nil {
				err = msgp.WrapError(err, "totalAmountsIn", za0003)
				return
			}
		}
	}
	err = en.WriteMapHeader(uint32(len(z.maxTotalAmountsIn)))
	if err != nil {
		err = msgp.WrapError(err, "maxTotalAmountsIn")
		return
	}
	for za0005, za0006 := range z.maxTotalAmountsIn {
		err = en.WriteString(za0005)
		if err != nil {
			err = msgp.WrapError(err, "maxTotalAmountsIn")
			return
		}
		if za0006 == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = en.WriteBytes(msgpencode.EncodeUint256(za0006))
			if err != nil {
				err = msgp.WrapError(err, "maxTotalAmountsIn", za0005)
				return
			}
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
	// array header, size 7
	o = append(o, 0x97)
	o, err = z.Pool.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "Pool")
		return
	}
	o = msgp.AppendMapHeader(o, uint32(len(z.records)))
	for za0001, za0002 := range z.records {
		o = msgp.AppendString(o, za0001)
		o, err = za0002.MarshalMsg(o)
		if err != nil {
			err = msgp.WrapError(err, "records", za0001)
			return
		}
	}
	o = msgp.AppendBool(o, z.publicSwap)
	if z.swapFee == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeUint256(z.swapFee))
	}
	o = msgp.AppendMapHeader(o, uint32(len(z.totalAmountsIn)))
	for za0003, za0004 := range z.totalAmountsIn {
		o = msgp.AppendString(o, za0003)
		if za0004 == nil {
			o = msgp.AppendNil(o)
		} else {
			o = msgp.AppendBytes(o, msgpencode.EncodeUint256(za0004))
		}
	}
	o = msgp.AppendMapHeader(o, uint32(len(z.maxTotalAmountsIn)))
	for za0005, za0006 := range z.maxTotalAmountsIn {
		o = msgp.AppendString(o, za0005)
		if za0006 == nil {
			o = msgp.AppendNil(o)
		} else {
			o = msgp.AppendBytes(o, msgpencode.EncodeUint256(za0006))
		}
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
	if zb0001 != 7 {
		err = msgp.ArrayError{Wanted: 7, Got: zb0001}
		return
	}
	bts, err = z.Pool.UnmarshalMsg(bts)
	if err != nil {
		err = msgp.WrapError(err, "Pool")
		return
	}
	var zb0002 uint32
	zb0002, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "records")
		return
	}
	if z.records == nil {
		z.records = make(map[string]Record, zb0002)
	} else if len(z.records) > 0 {
		for key := range z.records {
			delete(z.records, key)
		}
	}
	var field []byte
	_ = field
	for zb0002 > 0 {
		var za0001 string
		var za0002 Record
		zb0002--
		za0001, bts, err = msgp.ReadStringBytes(bts)
		if err != nil {
			err = msgp.WrapError(err, "records")
			return
		}
		bts, err = za0002.UnmarshalMsg(bts)
		if err != nil {
			err = msgp.WrapError(err, "records", za0001)
			return
		}
		z.records[za0001] = za0002
	}
	z.publicSwap, bts, err = msgp.ReadBoolBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "publicSwap")
		return
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.swapFee = nil
	} else {
		{
			var zb0003 []byte
			zb0003, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeUint256(z.swapFee))
			if err != nil {
				err = msgp.WrapError(err, "swapFee")
				return
			}
			z.swapFee = msgpencode.DecodeUint256(zb0003)
		}
	}
	var zb0004 uint32
	zb0004, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "totalAmountsIn")
		return
	}
	if z.totalAmountsIn == nil {
		z.totalAmountsIn = make(map[string]*uint256.Int, zb0004)
	} else if len(z.totalAmountsIn) > 0 {
		for key := range z.totalAmountsIn {
			delete(z.totalAmountsIn, key)
		}
	}
	for zb0004 > 0 {
		var za0003 string
		var za0004 *uint256.Int
		zb0004--
		za0003, bts, err = msgp.ReadStringBytes(bts)
		if err != nil {
			err = msgp.WrapError(err, "totalAmountsIn")
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
				zb0005, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeUint256(za0004))
				if err != nil {
					err = msgp.WrapError(err, "totalAmountsIn", za0003)
					return
				}
				za0004 = msgpencode.DecodeUint256(zb0005)
			}
		}
		z.totalAmountsIn[za0003] = za0004
	}
	var zb0006 uint32
	zb0006, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "maxTotalAmountsIn")
		return
	}
	if z.maxTotalAmountsIn == nil {
		z.maxTotalAmountsIn = make(map[string]*uint256.Int, zb0006)
	} else if len(z.maxTotalAmountsIn) > 0 {
		for key := range z.maxTotalAmountsIn {
			delete(z.maxTotalAmountsIn, key)
		}
	}
	for zb0006 > 0 {
		var za0005 string
		var za0006 *uint256.Int
		zb0006--
		za0005, bts, err = msgp.ReadStringBytes(bts)
		if err != nil {
			err = msgp.WrapError(err, "maxTotalAmountsIn")
			return
		}
		if msgp.IsNil(bts) {
			bts, err = msgp.ReadNilBytes(bts)
			if err != nil {
				return
			}
			za0006 = nil
		} else {
			{
				var zb0007 []byte
				zb0007, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeUint256(za0006))
				if err != nil {
					err = msgp.WrapError(err, "maxTotalAmountsIn", za0005)
					return
				}
				za0006 = msgpencode.DecodeUint256(zb0007)
			}
		}
		z.maxTotalAmountsIn[za0005] = za0006
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
	s = 1 + z.Pool.Msgsize() + msgp.MapHeaderSize
	if z.records != nil {
		for za0001, za0002 := range z.records {
			_ = za0002
			s += msgp.StringPrefixSize + len(za0001) + za0002.Msgsize()
		}
	}
	s += msgp.BoolSize
	if z.swapFee == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeUint256(z.swapFee))
	}
	s += msgp.MapHeaderSize
	if z.totalAmountsIn != nil {
		for za0003, za0004 := range z.totalAmountsIn {
			_ = za0004
			s += msgp.StringPrefixSize + len(za0003)
			if za0004 == nil {
				s += msgp.NilSize
			} else {
				s += msgp.BytesPrefixSize + len(msgpencode.EncodeUint256(za0004))
			}
		}
	}
	s += msgp.MapHeaderSize
	if z.maxTotalAmountsIn != nil {
		for za0005, za0006 := range z.maxTotalAmountsIn {
			_ = za0006
			s += msgp.StringPrefixSize + len(za0005)
			if za0006 == nil {
				s += msgp.NilSize
			} else {
				s += msgp.BytesPrefixSize + len(msgpencode.EncodeUint256(za0006))
			}
		}
	}
	s += z.gas.Msgsize()
	return
}