package meta

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
	if zb0001 != 2 {
		err = msgp.ArrayError{Wanted: 2, Got: zb0001}
		return
	}
	z.Exchange, err = dc.ReadInt64()
	if err != nil {
		err = msgp.WrapError(err, "Exchange")
		return
	}
	z.ExchangeUnderlying, err = dc.ReadInt64()
	if err != nil {
		err = msgp.WrapError(err, "ExchangeUnderlying")
		return
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z Gas) EncodeMsg(en *msgp.Writer) (err error) {
	// array header, size 2
	err = en.Append(0x92)
	if err != nil {
		return
	}
	err = en.WriteInt64(z.Exchange)
	if err != nil {
		err = msgp.WrapError(err, "Exchange")
		return
	}
	err = en.WriteInt64(z.ExchangeUnderlying)
	if err != nil {
		err = msgp.WrapError(err, "ExchangeUnderlying")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z Gas) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 2
	o = append(o, 0x92)
	o = msgp.AppendInt64(o, z.Exchange)
	o = msgp.AppendInt64(o, z.ExchangeUnderlying)
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
	if zb0001 != 2 {
		err = msgp.ArrayError{Wanted: 2, Got: zb0001}
		return
	}
	z.Exchange, bts, err = msgp.ReadInt64Bytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "Exchange")
		return
	}
	z.ExchangeUnderlying, bts, err = msgp.ReadInt64Bytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "ExchangeUnderlying")
		return
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z Gas) Msgsize() (s int) {
	s = 1 + msgp.Int64Size + msgp.Int64Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *Pool) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "Pool":
			err = z.Pool.DecodeMsg(dc)
			if err != nil {
				err = msgp.WrapError(err, "Pool")
				return
			}
		case "BasePool":
			{
				var zb0002 []byte
				zb0002, err = dc.ReadBytes(encodeBasePool(z.BasePool))
				if err != nil {
					err = msgp.WrapError(err, "BasePool")
					return
				}
				z.BasePool = decodeBasePool(zb0002)
			}
		case "RateMultiplier":
			if dc.IsNil() {
				err = dc.ReadNil()
				if err != nil {
					err = msgp.WrapError(err, "RateMultiplier")
					return
				}
				z.RateMultiplier = nil
			} else {
				{
					var zb0003 []byte
					zb0003, err = dc.ReadBytes(msgpencode.EncodeInt(z.RateMultiplier))
					if err != nil {
						err = msgp.WrapError(err, "RateMultiplier")
						return
					}
					z.RateMultiplier = msgpencode.DecodeInt(zb0003)
				}
			}
		case "InitialA":
			if dc.IsNil() {
				err = dc.ReadNil()
				if err != nil {
					err = msgp.WrapError(err, "InitialA")
					return
				}
				z.InitialA = nil
			} else {
				{
					var zb0004 []byte
					zb0004, err = dc.ReadBytes(msgpencode.EncodeInt(z.InitialA))
					if err != nil {
						err = msgp.WrapError(err, "InitialA")
						return
					}
					z.InitialA = msgpencode.DecodeInt(zb0004)
				}
			}
		case "FutureA":
			if dc.IsNil() {
				err = dc.ReadNil()
				if err != nil {
					err = msgp.WrapError(err, "FutureA")
					return
				}
				z.FutureA = nil
			} else {
				{
					var zb0005 []byte
					zb0005, err = dc.ReadBytes(msgpencode.EncodeInt(z.FutureA))
					if err != nil {
						err = msgp.WrapError(err, "FutureA")
						return
					}
					z.FutureA = msgpencode.DecodeInt(zb0005)
				}
			}
		case "InitialATime":
			z.InitialATime, err = dc.ReadInt64()
			if err != nil {
				err = msgp.WrapError(err, "InitialATime")
				return
			}
		case "FutureATime":
			z.FutureATime, err = dc.ReadInt64()
			if err != nil {
				err = msgp.WrapError(err, "FutureATime")
				return
			}
		case "AdminFee":
			if dc.IsNil() {
				err = dc.ReadNil()
				if err != nil {
					err = msgp.WrapError(err, "AdminFee")
					return
				}
				z.AdminFee = nil
			} else {
				{
					var zb0006 []byte
					zb0006, err = dc.ReadBytes(msgpencode.EncodeInt(z.AdminFee))
					if err != nil {
						err = msgp.WrapError(err, "AdminFee")
						return
					}
					z.AdminFee = msgpencode.DecodeInt(zb0006)
				}
			}
		case "LpToken":
			z.LpToken, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "LpToken")
				return
			}
		case "LpSupply":
			if dc.IsNil() {
				err = dc.ReadNil()
				if err != nil {
					err = msgp.WrapError(err, "LpSupply")
					return
				}
				z.LpSupply = nil
			} else {
				{
					var zb0007 []byte
					zb0007, err = dc.ReadBytes(msgpencode.EncodeInt(z.LpSupply))
					if err != nil {
						err = msgp.WrapError(err, "LpSupply")
						return
					}
					z.LpSupply = msgpencode.DecodeInt(zb0007)
				}
			}
		case "APrecision":
			if dc.IsNil() {
				err = dc.ReadNil()
				if err != nil {
					err = msgp.WrapError(err, "APrecision")
					return
				}
				z.APrecision = nil
			} else {
				{
					var zb0008 []byte
					zb0008, err = dc.ReadBytes(msgpencode.EncodeInt(z.APrecision))
					if err != nil {
						err = msgp.WrapError(err, "APrecision")
						return
					}
					z.APrecision = msgpencode.DecodeInt(zb0008)
				}
			}
		case "gas":
			var zb0009 uint32
			zb0009, err = dc.ReadArrayHeader()
			if err != nil {
				err = msgp.WrapError(err, "gas")
				return
			}
			if zb0009 != 2 {
				err = msgp.ArrayError{Wanted: 2, Got: zb0009}
				return
			}
			z.gas.Exchange, err = dc.ReadInt64()
			if err != nil {
				err = msgp.WrapError(err, "gas", "Exchange")
				return
			}
			z.gas.ExchangeUnderlying, err = dc.ReadInt64()
			if err != nil {
				err = msgp.WrapError(err, "gas", "ExchangeUnderlying")
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *Pool) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 12
	// write "Pool"
	err = en.Append(0x8c, 0xa4, 0x50, 0x6f, 0x6f, 0x6c)
	if err != nil {
		return
	}
	err = z.Pool.EncodeMsg(en)
	if err != nil {
		err = msgp.WrapError(err, "Pool")
		return
	}
	// write "BasePool"
	err = en.Append(0xa8, 0x42, 0x61, 0x73, 0x65, 0x50, 0x6f, 0x6f, 0x6c)
	if err != nil {
		return
	}
	err = en.WriteBytes(encodeBasePool(z.BasePool))
	if err != nil {
		err = msgp.WrapError(err, "BasePool")
		return
	}
	// write "RateMultiplier"
	err = en.Append(0xae, 0x52, 0x61, 0x74, 0x65, 0x4d, 0x75, 0x6c, 0x74, 0x69, 0x70, 0x6c, 0x69, 0x65, 0x72)
	if err != nil {
		return
	}
	if z.RateMultiplier == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = en.WriteBytes(msgpencode.EncodeInt(z.RateMultiplier))
		if err != nil {
			err = msgp.WrapError(err, "RateMultiplier")
			return
		}
	}
	// write "InitialA"
	err = en.Append(0xa8, 0x49, 0x6e, 0x69, 0x74, 0x69, 0x61, 0x6c, 0x41)
	if err != nil {
		return
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
	// write "FutureA"
	err = en.Append(0xa7, 0x46, 0x75, 0x74, 0x75, 0x72, 0x65, 0x41)
	if err != nil {
		return
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
	// write "InitialATime"
	err = en.Append(0xac, 0x49, 0x6e, 0x69, 0x74, 0x69, 0x61, 0x6c, 0x41, 0x54, 0x69, 0x6d, 0x65)
	if err != nil {
		return
	}
	err = en.WriteInt64(z.InitialATime)
	if err != nil {
		err = msgp.WrapError(err, "InitialATime")
		return
	}
	// write "FutureATime"
	err = en.Append(0xab, 0x46, 0x75, 0x74, 0x75, 0x72, 0x65, 0x41, 0x54, 0x69, 0x6d, 0x65)
	if err != nil {
		return
	}
	err = en.WriteInt64(z.FutureATime)
	if err != nil {
		err = msgp.WrapError(err, "FutureATime")
		return
	}
	// write "AdminFee"
	err = en.Append(0xa8, 0x41, 0x64, 0x6d, 0x69, 0x6e, 0x46, 0x65, 0x65)
	if err != nil {
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
	// write "LpToken"
	err = en.Append(0xa7, 0x4c, 0x70, 0x54, 0x6f, 0x6b, 0x65, 0x6e)
	if err != nil {
		return
	}
	err = en.WriteString(z.LpToken)
	if err != nil {
		err = msgp.WrapError(err, "LpToken")
		return
	}
	// write "LpSupply"
	err = en.Append(0xa8, 0x4c, 0x70, 0x53, 0x75, 0x70, 0x70, 0x6c, 0x79)
	if err != nil {
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
	// write "APrecision"
	err = en.Append(0xaa, 0x41, 0x50, 0x72, 0x65, 0x63, 0x69, 0x73, 0x69, 0x6f, 0x6e)
	if err != nil {
		return
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
	// write "gas"
	err = en.Append(0xa3, 0x67, 0x61, 0x73)
	if err != nil {
		return
	}
	// array header, size 2
	err = en.Append(0x92)
	if err != nil {
		return
	}
	err = en.WriteInt64(z.gas.Exchange)
	if err != nil {
		err = msgp.WrapError(err, "gas", "Exchange")
		return
	}
	err = en.WriteInt64(z.gas.ExchangeUnderlying)
	if err != nil {
		err = msgp.WrapError(err, "gas", "ExchangeUnderlying")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *Pool) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 12
	// string "Pool"
	o = append(o, 0x8c, 0xa4, 0x50, 0x6f, 0x6f, 0x6c)
	o, err = z.Pool.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "Pool")
		return
	}
	// string "BasePool"
	o = append(o, 0xa8, 0x42, 0x61, 0x73, 0x65, 0x50, 0x6f, 0x6f, 0x6c)
	o = msgp.AppendBytes(o, encodeBasePool(z.BasePool))
	// string "RateMultiplier"
	o = append(o, 0xae, 0x52, 0x61, 0x74, 0x65, 0x4d, 0x75, 0x6c, 0x74, 0x69, 0x70, 0x6c, 0x69, 0x65, 0x72)
	if z.RateMultiplier == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeInt(z.RateMultiplier))
	}
	// string "InitialA"
	o = append(o, 0xa8, 0x49, 0x6e, 0x69, 0x74, 0x69, 0x61, 0x6c, 0x41)
	if z.InitialA == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeInt(z.InitialA))
	}
	// string "FutureA"
	o = append(o, 0xa7, 0x46, 0x75, 0x74, 0x75, 0x72, 0x65, 0x41)
	if z.FutureA == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeInt(z.FutureA))
	}
	// string "InitialATime"
	o = append(o, 0xac, 0x49, 0x6e, 0x69, 0x74, 0x69, 0x61, 0x6c, 0x41, 0x54, 0x69, 0x6d, 0x65)
	o = msgp.AppendInt64(o, z.InitialATime)
	// string "FutureATime"
	o = append(o, 0xab, 0x46, 0x75, 0x74, 0x75, 0x72, 0x65, 0x41, 0x54, 0x69, 0x6d, 0x65)
	o = msgp.AppendInt64(o, z.FutureATime)
	// string "AdminFee"
	o = append(o, 0xa8, 0x41, 0x64, 0x6d, 0x69, 0x6e, 0x46, 0x65, 0x65)
	if z.AdminFee == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeInt(z.AdminFee))
	}
	// string "LpToken"
	o = append(o, 0xa7, 0x4c, 0x70, 0x54, 0x6f, 0x6b, 0x65, 0x6e)
	o = msgp.AppendString(o, z.LpToken)
	// string "LpSupply"
	o = append(o, 0xa8, 0x4c, 0x70, 0x53, 0x75, 0x70, 0x70, 0x6c, 0x79)
	if z.LpSupply == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeInt(z.LpSupply))
	}
	// string "APrecision"
	o = append(o, 0xaa, 0x41, 0x50, 0x72, 0x65, 0x63, 0x69, 0x73, 0x69, 0x6f, 0x6e)
	if z.APrecision == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeInt(z.APrecision))
	}
	// string "gas"
	o = append(o, 0xa3, 0x67, 0x61, 0x73)
	// array header, size 2
	o = append(o, 0x92)
	o = msgp.AppendInt64(o, z.gas.Exchange)
	o = msgp.AppendInt64(o, z.gas.ExchangeUnderlying)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Pool) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "Pool":
			bts, err = z.Pool.UnmarshalMsg(bts)
			if err != nil {
				err = msgp.WrapError(err, "Pool")
				return
			}
		case "BasePool":
			{
				var zb0002 []byte
				zb0002, bts, err = msgp.ReadBytesBytes(bts, encodeBasePool(z.BasePool))
				if err != nil {
					err = msgp.WrapError(err, "BasePool")
					return
				}
				z.BasePool = decodeBasePool(zb0002)
			}
		case "RateMultiplier":
			if msgp.IsNil(bts) {
				bts, err = msgp.ReadNilBytes(bts)
				if err != nil {
					return
				}
				z.RateMultiplier = nil
			} else {
				{
					var zb0003 []byte
					zb0003, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.RateMultiplier))
					if err != nil {
						err = msgp.WrapError(err, "RateMultiplier")
						return
					}
					z.RateMultiplier = msgpencode.DecodeInt(zb0003)
				}
			}
		case "InitialA":
			if msgp.IsNil(bts) {
				bts, err = msgp.ReadNilBytes(bts)
				if err != nil {
					return
				}
				z.InitialA = nil
			} else {
				{
					var zb0004 []byte
					zb0004, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.InitialA))
					if err != nil {
						err = msgp.WrapError(err, "InitialA")
						return
					}
					z.InitialA = msgpencode.DecodeInt(zb0004)
				}
			}
		case "FutureA":
			if msgp.IsNil(bts) {
				bts, err = msgp.ReadNilBytes(bts)
				if err != nil {
					return
				}
				z.FutureA = nil
			} else {
				{
					var zb0005 []byte
					zb0005, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.FutureA))
					if err != nil {
						err = msgp.WrapError(err, "FutureA")
						return
					}
					z.FutureA = msgpencode.DecodeInt(zb0005)
				}
			}
		case "InitialATime":
			z.InitialATime, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "InitialATime")
				return
			}
		case "FutureATime":
			z.FutureATime, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "FutureATime")
				return
			}
		case "AdminFee":
			if msgp.IsNil(bts) {
				bts, err = msgp.ReadNilBytes(bts)
				if err != nil {
					return
				}
				z.AdminFee = nil
			} else {
				{
					var zb0006 []byte
					zb0006, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.AdminFee))
					if err != nil {
						err = msgp.WrapError(err, "AdminFee")
						return
					}
					z.AdminFee = msgpencode.DecodeInt(zb0006)
				}
			}
		case "LpToken":
			z.LpToken, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "LpToken")
				return
			}
		case "LpSupply":
			if msgp.IsNil(bts) {
				bts, err = msgp.ReadNilBytes(bts)
				if err != nil {
					return
				}
				z.LpSupply = nil
			} else {
				{
					var zb0007 []byte
					zb0007, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.LpSupply))
					if err != nil {
						err = msgp.WrapError(err, "LpSupply")
						return
					}
					z.LpSupply = msgpencode.DecodeInt(zb0007)
				}
			}
		case "APrecision":
			if msgp.IsNil(bts) {
				bts, err = msgp.ReadNilBytes(bts)
				if err != nil {
					return
				}
				z.APrecision = nil
			} else {
				{
					var zb0008 []byte
					zb0008, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.APrecision))
					if err != nil {
						err = msgp.WrapError(err, "APrecision")
						return
					}
					z.APrecision = msgpencode.DecodeInt(zb0008)
				}
			}
		case "gas":
			var zb0009 uint32
			zb0009, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "gas")
				return
			}
			if zb0009 != 2 {
				err = msgp.ArrayError{Wanted: 2, Got: zb0009}
				return
			}
			z.gas.Exchange, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "gas", "Exchange")
				return
			}
			z.gas.ExchangeUnderlying, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "gas", "ExchangeUnderlying")
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *Pool) Msgsize() (s int) {
	s = 1 + 5 + z.Pool.Msgsize() + 9 + msgp.BytesPrefixSize + len(encodeBasePool(z.BasePool)) + 15
	if z.RateMultiplier == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(z.RateMultiplier))
	}
	s += 9
	if z.InitialA == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(z.InitialA))
	}
	s += 8
	if z.FutureA == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(z.FutureA))
	}
	s += 13 + msgp.Int64Size + 12 + msgp.Int64Size + 9
	if z.AdminFee == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(z.AdminFee))
	}
	s += 8 + msgp.StringPrefixSize + len(z.LpToken) + 9
	if z.LpSupply == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(z.LpSupply))
	}
	s += 11
	if z.APrecision == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(z.APrecision))
	}
	s += 4 + 1 + msgp.Int64Size + msgp.Int64Size
	return
}