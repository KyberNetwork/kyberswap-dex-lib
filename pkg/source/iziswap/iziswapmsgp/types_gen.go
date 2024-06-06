package iziswapmsgp

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpencode"
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *LimitOrderPoint) DecodeMsg(dc *msgp.Reader) (err error) {
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
		case "SellingX":
			if dc.IsNil() {
				err = dc.ReadNil()
				if err != nil {
					err = msgp.WrapError(err, "SellingX")
					return
				}
				z.SellingX = nil
			} else {
				{
					var zb0002 []byte
					zb0002, err = dc.ReadBytes(msgpencode.EncodeInt(z.SellingX))
					if err != nil {
						err = msgp.WrapError(err, "SellingX")
						return
					}
					z.SellingX = msgpencode.DecodeInt(zb0002)
				}
			}
		case "SellingY":
			if dc.IsNil() {
				err = dc.ReadNil()
				if err != nil {
					err = msgp.WrapError(err, "SellingY")
					return
				}
				z.SellingY = nil
			} else {
				{
					var zb0003 []byte
					zb0003, err = dc.ReadBytes(msgpencode.EncodeInt(z.SellingY))
					if err != nil {
						err = msgp.WrapError(err, "SellingY")
						return
					}
					z.SellingY = msgpencode.DecodeInt(zb0003)
				}
			}
		case "Point":
			z.Point, err = dc.ReadInt()
			if err != nil {
				err = msgp.WrapError(err, "Point")
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
func (z *LimitOrderPoint) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 3
	// write "SellingX"
	err = en.Append(0x83, 0xa8, 0x53, 0x65, 0x6c, 0x6c, 0x69, 0x6e, 0x67, 0x58)
	if err != nil {
		return
	}
	if z.SellingX == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = en.WriteBytes(msgpencode.EncodeInt(z.SellingX))
		if err != nil {
			err = msgp.WrapError(err, "SellingX")
			return
		}
	}
	// write "SellingY"
	err = en.Append(0xa8, 0x53, 0x65, 0x6c, 0x6c, 0x69, 0x6e, 0x67, 0x59)
	if err != nil {
		return
	}
	if z.SellingY == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = en.WriteBytes(msgpencode.EncodeInt(z.SellingY))
		if err != nil {
			err = msgp.WrapError(err, "SellingY")
			return
		}
	}
	// write "Point"
	err = en.Append(0xa5, 0x50, 0x6f, 0x69, 0x6e, 0x74)
	if err != nil {
		return
	}
	err = en.WriteInt(z.Point)
	if err != nil {
		err = msgp.WrapError(err, "Point")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *LimitOrderPoint) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 3
	// string "SellingX"
	o = append(o, 0x83, 0xa8, 0x53, 0x65, 0x6c, 0x6c, 0x69, 0x6e, 0x67, 0x58)
	if z.SellingX == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeInt(z.SellingX))
	}
	// string "SellingY"
	o = append(o, 0xa8, 0x53, 0x65, 0x6c, 0x6c, 0x69, 0x6e, 0x67, 0x59)
	if z.SellingY == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeInt(z.SellingY))
	}
	// string "Point"
	o = append(o, 0xa5, 0x50, 0x6f, 0x69, 0x6e, 0x74)
	o = msgp.AppendInt(o, z.Point)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *LimitOrderPoint) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "SellingX":
			if msgp.IsNil(bts) {
				bts, err = msgp.ReadNilBytes(bts)
				if err != nil {
					return
				}
				z.SellingX = nil
			} else {
				{
					var zb0002 []byte
					zb0002, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.SellingX))
					if err != nil {
						err = msgp.WrapError(err, "SellingX")
						return
					}
					z.SellingX = msgpencode.DecodeInt(zb0002)
				}
			}
		case "SellingY":
			if msgp.IsNil(bts) {
				bts, err = msgp.ReadNilBytes(bts)
				if err != nil {
					return
				}
				z.SellingY = nil
			} else {
				{
					var zb0003 []byte
					zb0003, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.SellingY))
					if err != nil {
						err = msgp.WrapError(err, "SellingY")
						return
					}
					z.SellingY = msgpencode.DecodeInt(zb0003)
				}
			}
		case "Point":
			z.Point, bts, err = msgp.ReadIntBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Point")
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
func (z *LimitOrderPoint) Msgsize() (s int) {
	s = 1 + 9
	if z.SellingX == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(z.SellingX))
	}
	s += 9
	if z.SellingY == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(z.SellingY))
	}
	s += 6 + msgp.IntSize
	return
}

// DecodeMsg implements msgp.Decodable
func (z *LiquidityPoint) DecodeMsg(dc *msgp.Reader) (err error) {
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
		case "LiqudityDelta":
			if dc.IsNil() {
				err = dc.ReadNil()
				if err != nil {
					err = msgp.WrapError(err, "LiqudityDelta")
					return
				}
				z.LiqudityDelta = nil
			} else {
				{
					var zb0002 []byte
					zb0002, err = dc.ReadBytes(msgpencode.EncodeInt(z.LiqudityDelta))
					if err != nil {
						err = msgp.WrapError(err, "LiqudityDelta")
						return
					}
					z.LiqudityDelta = msgpencode.DecodeInt(zb0002)
				}
			}
		case "Point":
			z.Point, err = dc.ReadInt()
			if err != nil {
				err = msgp.WrapError(err, "Point")
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
func (z *LiquidityPoint) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 2
	// write "LiqudityDelta"
	err = en.Append(0x82, 0xad, 0x4c, 0x69, 0x71, 0x75, 0x64, 0x69, 0x74, 0x79, 0x44, 0x65, 0x6c, 0x74, 0x61)
	if err != nil {
		return
	}
	if z.LiqudityDelta == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = en.WriteBytes(msgpencode.EncodeInt(z.LiqudityDelta))
		if err != nil {
			err = msgp.WrapError(err, "LiqudityDelta")
			return
		}
	}
	// write "Point"
	err = en.Append(0xa5, 0x50, 0x6f, 0x69, 0x6e, 0x74)
	if err != nil {
		return
	}
	err = en.WriteInt(z.Point)
	if err != nil {
		err = msgp.WrapError(err, "Point")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *LiquidityPoint) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 2
	// string "LiqudityDelta"
	o = append(o, 0x82, 0xad, 0x4c, 0x69, 0x71, 0x75, 0x64, 0x69, 0x74, 0x79, 0x44, 0x65, 0x6c, 0x74, 0x61)
	if z.LiqudityDelta == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeInt(z.LiqudityDelta))
	}
	// string "Point"
	o = append(o, 0xa5, 0x50, 0x6f, 0x69, 0x6e, 0x74)
	o = msgp.AppendInt(o, z.Point)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *LiquidityPoint) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "LiqudityDelta":
			if msgp.IsNil(bts) {
				bts, err = msgp.ReadNilBytes(bts)
				if err != nil {
					return
				}
				z.LiqudityDelta = nil
			} else {
				{
					var zb0002 []byte
					zb0002, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.LiqudityDelta))
					if err != nil {
						err = msgp.WrapError(err, "LiqudityDelta")
						return
					}
					z.LiqudityDelta = msgpencode.DecodeInt(zb0002)
				}
			}
		case "Point":
			z.Point, bts, err = msgp.ReadIntBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Point")
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
func (z *LiquidityPoint) Msgsize() (s int) {
	s = 1 + 14
	if z.LiqudityDelta == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(z.LiqudityDelta))
	}
	s += 6 + msgp.IntSize
	return
}

// DecodeMsg implements msgp.Decodable
func (z *PoolInfo) DecodeMsg(dc *msgp.Reader) (err error) {
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
		case "CurrentPoint":
			z.CurrentPoint, err = dc.ReadInt()
			if err != nil {
				err = msgp.WrapError(err, "CurrentPoint")
				return
			}
		case "PointDelta":
			z.PointDelta, err = dc.ReadInt()
			if err != nil {
				err = msgp.WrapError(err, "PointDelta")
				return
			}
		case "LeftMostPt":
			z.LeftMostPt, err = dc.ReadInt()
			if err != nil {
				err = msgp.WrapError(err, "LeftMostPt")
				return
			}
		case "RightMostPt":
			z.RightMostPt, err = dc.ReadInt()
			if err != nil {
				err = msgp.WrapError(err, "RightMostPt")
				return
			}
		case "Fee":
			z.Fee, err = dc.ReadInt()
			if err != nil {
				err = msgp.WrapError(err, "Fee")
				return
			}
		case "Liquidity":
			if dc.IsNil() {
				err = dc.ReadNil()
				if err != nil {
					err = msgp.WrapError(err, "Liquidity")
					return
				}
				z.Liquidity = nil
			} else {
				{
					var zb0002 []byte
					zb0002, err = dc.ReadBytes(msgpencode.EncodeInt(z.Liquidity))
					if err != nil {
						err = msgp.WrapError(err, "Liquidity")
						return
					}
					z.Liquidity = msgpencode.DecodeInt(zb0002)
				}
			}
		case "LiquidityX":
			if dc.IsNil() {
				err = dc.ReadNil()
				if err != nil {
					err = msgp.WrapError(err, "LiquidityX")
					return
				}
				z.LiquidityX = nil
			} else {
				{
					var zb0003 []byte
					zb0003, err = dc.ReadBytes(msgpencode.EncodeInt(z.LiquidityX))
					if err != nil {
						err = msgp.WrapError(err, "LiquidityX")
						return
					}
					z.LiquidityX = msgpencode.DecodeInt(zb0003)
				}
			}
		case "Liquidities":
			var zb0004 uint32
			zb0004, err = dc.ReadArrayHeader()
			if err != nil {
				err = msgp.WrapError(err, "Liquidities")
				return
			}
			if cap(z.Liquidities) >= int(zb0004) {
				z.Liquidities = (z.Liquidities)[:zb0004]
			} else {
				z.Liquidities = make([]LiquidityPoint, zb0004)
			}
			for za0001 := range z.Liquidities {
				var zb0005 uint32
				zb0005, err = dc.ReadMapHeader()
				if err != nil {
					err = msgp.WrapError(err, "Liquidities", za0001)
					return
				}
				for zb0005 > 0 {
					zb0005--
					field, err = dc.ReadMapKeyPtr()
					if err != nil {
						err = msgp.WrapError(err, "Liquidities", za0001)
						return
					}
					switch msgp.UnsafeString(field) {
					case "LiqudityDelta":
						if dc.IsNil() {
							err = dc.ReadNil()
							if err != nil {
								err = msgp.WrapError(err, "Liquidities", za0001, "LiqudityDelta")
								return
							}
							z.Liquidities[za0001].LiqudityDelta = nil
						} else {
							{
								var zb0006 []byte
								zb0006, err = dc.ReadBytes(msgpencode.EncodeInt(z.Liquidities[za0001].LiqudityDelta))
								if err != nil {
									err = msgp.WrapError(err, "Liquidities", za0001, "LiqudityDelta")
									return
								}
								z.Liquidities[za0001].LiqudityDelta = msgpencode.DecodeInt(zb0006)
							}
						}
					case "Point":
						z.Liquidities[za0001].Point, err = dc.ReadInt()
						if err != nil {
							err = msgp.WrapError(err, "Liquidities", za0001, "Point")
							return
						}
					default:
						err = dc.Skip()
						if err != nil {
							err = msgp.WrapError(err, "Liquidities", za0001)
							return
						}
					}
				}
			}
		case "LimitOrders":
			var zb0007 uint32
			zb0007, err = dc.ReadArrayHeader()
			if err != nil {
				err = msgp.WrapError(err, "LimitOrders")
				return
			}
			if cap(z.LimitOrders) >= int(zb0007) {
				z.LimitOrders = (z.LimitOrders)[:zb0007]
			} else {
				z.LimitOrders = make([]LimitOrderPoint, zb0007)
			}
			for za0002 := range z.LimitOrders {
				err = z.LimitOrders[za0002].DecodeMsg(dc)
				if err != nil {
					err = msgp.WrapError(err, "LimitOrders", za0002)
					return
				}
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
func (z *PoolInfo) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 9
	// write "CurrentPoint"
	err = en.Append(0x89, 0xac, 0x43, 0x75, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x50, 0x6f, 0x69, 0x6e, 0x74)
	if err != nil {
		return
	}
	err = en.WriteInt(z.CurrentPoint)
	if err != nil {
		err = msgp.WrapError(err, "CurrentPoint")
		return
	}
	// write "PointDelta"
	err = en.Append(0xaa, 0x50, 0x6f, 0x69, 0x6e, 0x74, 0x44, 0x65, 0x6c, 0x74, 0x61)
	if err != nil {
		return
	}
	err = en.WriteInt(z.PointDelta)
	if err != nil {
		err = msgp.WrapError(err, "PointDelta")
		return
	}
	// write "LeftMostPt"
	err = en.Append(0xaa, 0x4c, 0x65, 0x66, 0x74, 0x4d, 0x6f, 0x73, 0x74, 0x50, 0x74)
	if err != nil {
		return
	}
	err = en.WriteInt(z.LeftMostPt)
	if err != nil {
		err = msgp.WrapError(err, "LeftMostPt")
		return
	}
	// write "RightMostPt"
	err = en.Append(0xab, 0x52, 0x69, 0x67, 0x68, 0x74, 0x4d, 0x6f, 0x73, 0x74, 0x50, 0x74)
	if err != nil {
		return
	}
	err = en.WriteInt(z.RightMostPt)
	if err != nil {
		err = msgp.WrapError(err, "RightMostPt")
		return
	}
	// write "Fee"
	err = en.Append(0xa3, 0x46, 0x65, 0x65)
	if err != nil {
		return
	}
	err = en.WriteInt(z.Fee)
	if err != nil {
		err = msgp.WrapError(err, "Fee")
		return
	}
	// write "Liquidity"
	err = en.Append(0xa9, 0x4c, 0x69, 0x71, 0x75, 0x69, 0x64, 0x69, 0x74, 0x79)
	if err != nil {
		return
	}
	if z.Liquidity == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = en.WriteBytes(msgpencode.EncodeInt(z.Liquidity))
		if err != nil {
			err = msgp.WrapError(err, "Liquidity")
			return
		}
	}
	// write "LiquidityX"
	err = en.Append(0xaa, 0x4c, 0x69, 0x71, 0x75, 0x69, 0x64, 0x69, 0x74, 0x79, 0x58)
	if err != nil {
		return
	}
	if z.LiquidityX == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = en.WriteBytes(msgpencode.EncodeInt(z.LiquidityX))
		if err != nil {
			err = msgp.WrapError(err, "LiquidityX")
			return
		}
	}
	// write "Liquidities"
	err = en.Append(0xab, 0x4c, 0x69, 0x71, 0x75, 0x69, 0x64, 0x69, 0x74, 0x69, 0x65, 0x73)
	if err != nil {
		return
	}
	err = en.WriteArrayHeader(uint32(len(z.Liquidities)))
	if err != nil {
		err = msgp.WrapError(err, "Liquidities")
		return
	}
	for za0001 := range z.Liquidities {
		// map header, size 2
		// write "LiqudityDelta"
		err = en.Append(0x82, 0xad, 0x4c, 0x69, 0x71, 0x75, 0x64, 0x69, 0x74, 0x79, 0x44, 0x65, 0x6c, 0x74, 0x61)
		if err != nil {
			return
		}
		if z.Liquidities[za0001].LiqudityDelta == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = en.WriteBytes(msgpencode.EncodeInt(z.Liquidities[za0001].LiqudityDelta))
			if err != nil {
				err = msgp.WrapError(err, "Liquidities", za0001, "LiqudityDelta")
				return
			}
		}
		// write "Point"
		err = en.Append(0xa5, 0x50, 0x6f, 0x69, 0x6e, 0x74)
		if err != nil {
			return
		}
		err = en.WriteInt(z.Liquidities[za0001].Point)
		if err != nil {
			err = msgp.WrapError(err, "Liquidities", za0001, "Point")
			return
		}
	}
	// write "LimitOrders"
	err = en.Append(0xab, 0x4c, 0x69, 0x6d, 0x69, 0x74, 0x4f, 0x72, 0x64, 0x65, 0x72, 0x73)
	if err != nil {
		return
	}
	err = en.WriteArrayHeader(uint32(len(z.LimitOrders)))
	if err != nil {
		err = msgp.WrapError(err, "LimitOrders")
		return
	}
	for za0002 := range z.LimitOrders {
		err = z.LimitOrders[za0002].EncodeMsg(en)
		if err != nil {
			err = msgp.WrapError(err, "LimitOrders", za0002)
			return
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *PoolInfo) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 9
	// string "CurrentPoint"
	o = append(o, 0x89, 0xac, 0x43, 0x75, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x50, 0x6f, 0x69, 0x6e, 0x74)
	o = msgp.AppendInt(o, z.CurrentPoint)
	// string "PointDelta"
	o = append(o, 0xaa, 0x50, 0x6f, 0x69, 0x6e, 0x74, 0x44, 0x65, 0x6c, 0x74, 0x61)
	o = msgp.AppendInt(o, z.PointDelta)
	// string "LeftMostPt"
	o = append(o, 0xaa, 0x4c, 0x65, 0x66, 0x74, 0x4d, 0x6f, 0x73, 0x74, 0x50, 0x74)
	o = msgp.AppendInt(o, z.LeftMostPt)
	// string "RightMostPt"
	o = append(o, 0xab, 0x52, 0x69, 0x67, 0x68, 0x74, 0x4d, 0x6f, 0x73, 0x74, 0x50, 0x74)
	o = msgp.AppendInt(o, z.RightMostPt)
	// string "Fee"
	o = append(o, 0xa3, 0x46, 0x65, 0x65)
	o = msgp.AppendInt(o, z.Fee)
	// string "Liquidity"
	o = append(o, 0xa9, 0x4c, 0x69, 0x71, 0x75, 0x69, 0x64, 0x69, 0x74, 0x79)
	if z.Liquidity == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeInt(z.Liquidity))
	}
	// string "LiquidityX"
	o = append(o, 0xaa, 0x4c, 0x69, 0x71, 0x75, 0x69, 0x64, 0x69, 0x74, 0x79, 0x58)
	if z.LiquidityX == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeInt(z.LiquidityX))
	}
	// string "Liquidities"
	o = append(o, 0xab, 0x4c, 0x69, 0x71, 0x75, 0x69, 0x64, 0x69, 0x74, 0x69, 0x65, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Liquidities)))
	for za0001 := range z.Liquidities {
		// map header, size 2
		// string "LiqudityDelta"
		o = append(o, 0x82, 0xad, 0x4c, 0x69, 0x71, 0x75, 0x64, 0x69, 0x74, 0x79, 0x44, 0x65, 0x6c, 0x74, 0x61)
		if z.Liquidities[za0001].LiqudityDelta == nil {
			o = msgp.AppendNil(o)
		} else {
			o = msgp.AppendBytes(o, msgpencode.EncodeInt(z.Liquidities[za0001].LiqudityDelta))
		}
		// string "Point"
		o = append(o, 0xa5, 0x50, 0x6f, 0x69, 0x6e, 0x74)
		o = msgp.AppendInt(o, z.Liquidities[za0001].Point)
	}
	// string "LimitOrders"
	o = append(o, 0xab, 0x4c, 0x69, 0x6d, 0x69, 0x74, 0x4f, 0x72, 0x64, 0x65, 0x72, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.LimitOrders)))
	for za0002 := range z.LimitOrders {
		o, err = z.LimitOrders[za0002].MarshalMsg(o)
		if err != nil {
			err = msgp.WrapError(err, "LimitOrders", za0002)
			return
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *PoolInfo) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "CurrentPoint":
			z.CurrentPoint, bts, err = msgp.ReadIntBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "CurrentPoint")
				return
			}
		case "PointDelta":
			z.PointDelta, bts, err = msgp.ReadIntBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "PointDelta")
				return
			}
		case "LeftMostPt":
			z.LeftMostPt, bts, err = msgp.ReadIntBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "LeftMostPt")
				return
			}
		case "RightMostPt":
			z.RightMostPt, bts, err = msgp.ReadIntBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "RightMostPt")
				return
			}
		case "Fee":
			z.Fee, bts, err = msgp.ReadIntBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Fee")
				return
			}
		case "Liquidity":
			if msgp.IsNil(bts) {
				bts, err = msgp.ReadNilBytes(bts)
				if err != nil {
					return
				}
				z.Liquidity = nil
			} else {
				{
					var zb0002 []byte
					zb0002, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.Liquidity))
					if err != nil {
						err = msgp.WrapError(err, "Liquidity")
						return
					}
					z.Liquidity = msgpencode.DecodeInt(zb0002)
				}
			}
		case "LiquidityX":
			if msgp.IsNil(bts) {
				bts, err = msgp.ReadNilBytes(bts)
				if err != nil {
					return
				}
				z.LiquidityX = nil
			} else {
				{
					var zb0003 []byte
					zb0003, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.LiquidityX))
					if err != nil {
						err = msgp.WrapError(err, "LiquidityX")
						return
					}
					z.LiquidityX = msgpencode.DecodeInt(zb0003)
				}
			}
		case "Liquidities":
			var zb0004 uint32
			zb0004, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Liquidities")
				return
			}
			if cap(z.Liquidities) >= int(zb0004) {
				z.Liquidities = (z.Liquidities)[:zb0004]
			} else {
				z.Liquidities = make([]LiquidityPoint, zb0004)
			}
			for za0001 := range z.Liquidities {
				var zb0005 uint32
				zb0005, bts, err = msgp.ReadMapHeaderBytes(bts)
				if err != nil {
					err = msgp.WrapError(err, "Liquidities", za0001)
					return
				}
				for zb0005 > 0 {
					zb0005--
					field, bts, err = msgp.ReadMapKeyZC(bts)
					if err != nil {
						err = msgp.WrapError(err, "Liquidities", za0001)
						return
					}
					switch msgp.UnsafeString(field) {
					case "LiqudityDelta":
						if msgp.IsNil(bts) {
							bts, err = msgp.ReadNilBytes(bts)
							if err != nil {
								return
							}
							z.Liquidities[za0001].LiqudityDelta = nil
						} else {
							{
								var zb0006 []byte
								zb0006, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.Liquidities[za0001].LiqudityDelta))
								if err != nil {
									err = msgp.WrapError(err, "Liquidities", za0001, "LiqudityDelta")
									return
								}
								z.Liquidities[za0001].LiqudityDelta = msgpencode.DecodeInt(zb0006)
							}
						}
					case "Point":
						z.Liquidities[za0001].Point, bts, err = msgp.ReadIntBytes(bts)
						if err != nil {
							err = msgp.WrapError(err, "Liquidities", za0001, "Point")
							return
						}
					default:
						bts, err = msgp.Skip(bts)
						if err != nil {
							err = msgp.WrapError(err, "Liquidities", za0001)
							return
						}
					}
				}
			}
		case "LimitOrders":
			var zb0007 uint32
			zb0007, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "LimitOrders")
				return
			}
			if cap(z.LimitOrders) >= int(zb0007) {
				z.LimitOrders = (z.LimitOrders)[:zb0007]
			} else {
				z.LimitOrders = make([]LimitOrderPoint, zb0007)
			}
			for za0002 := range z.LimitOrders {
				bts, err = z.LimitOrders[za0002].UnmarshalMsg(bts)
				if err != nil {
					err = msgp.WrapError(err, "LimitOrders", za0002)
					return
				}
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
func (z *PoolInfo) Msgsize() (s int) {
	s = 1 + 13 + msgp.IntSize + 11 + msgp.IntSize + 11 + msgp.IntSize + 12 + msgp.IntSize + 4 + msgp.IntSize + 10
	if z.Liquidity == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(z.Liquidity))
	}
	s += 11
	if z.LiquidityX == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(z.LiquidityX))
	}
	s += 12 + msgp.ArrayHeaderSize
	for za0001 := range z.Liquidities {
		s += 1 + 14
		if z.Liquidities[za0001].LiqudityDelta == nil {
			s += msgp.NilSize
		} else {
			s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(z.Liquidities[za0001].LiqudityDelta))
		}
		s += 6 + msgp.IntSize
	}
	s += 12 + msgp.ArrayHeaderSize
	for za0002 := range z.LimitOrders {
		s += z.LimitOrders[za0002].Msgsize()
	}
	return
}