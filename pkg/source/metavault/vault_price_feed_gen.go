package metavault

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpencode"
	"github.com/ethereum/go-ethereum/common"
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
	if zb0001 != 13 {
		err = msgp.ArrayError{Wanted: 13, Got: zb0001}
		return
	}
	z.IsSecondaryPriceEnabled, err = dc.ReadBool()
	if err != nil {
		err = msgp.WrapError(err, "IsSecondaryPriceEnabled")
		return
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "MaxStrictPriceDeviation")
			return
		}
		z.MaxStrictPriceDeviation = nil
	} else {
		{
			var zb0002 []byte
			zb0002, err = dc.ReadBytes(msgpencode.EncodeInt(z.MaxStrictPriceDeviation))
			if err != nil {
				err = msgp.WrapError(err, "MaxStrictPriceDeviation")
				return
			}
			z.MaxStrictPriceDeviation = msgpencode.DecodeInt(zb0002)
		}
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "PriceSampleSpace")
			return
		}
		z.PriceSampleSpace = nil
	} else {
		{
			var zb0003 []byte
			zb0003, err = dc.ReadBytes(msgpencode.EncodeInt(z.PriceSampleSpace))
			if err != nil {
				err = msgp.WrapError(err, "PriceSampleSpace")
				return
			}
			z.PriceSampleSpace = msgpencode.DecodeInt(zb0003)
		}
	}
	var zb0004 uint32
	zb0004, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err, "PriceDecimals")
		return
	}
	if z.PriceDecimals == nil {
		z.PriceDecimals = make(map[string]*big.Int, zb0004)
	} else if len(z.PriceDecimals) > 0 {
		for key := range z.PriceDecimals {
			delete(z.PriceDecimals, key)
		}
	}
	for zb0004 > 0 {
		zb0004--
		var za0001 string
		var za0002 *big.Int
		za0001, err = dc.ReadString()
		if err != nil {
			err = msgp.WrapError(err, "PriceDecimals")
			return
		}
		if dc.IsNil() {
			err = dc.ReadNil()
			if err != nil {
				err = msgp.WrapError(err, "PriceDecimals", za0001)
				return
			}
			za0002 = nil
		} else {
			{
				var zb0005 []byte
				zb0005, err = dc.ReadBytes(msgpencode.EncodeInt(za0002))
				if err != nil {
					err = msgp.WrapError(err, "PriceDecimals", za0001)
					return
				}
				za0002 = msgpencode.DecodeInt(zb0005)
			}
		}
		z.PriceDecimals[za0001] = za0002
	}
	var zb0006 uint32
	zb0006, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err, "SpreadBasisPoints")
		return
	}
	if z.SpreadBasisPoints == nil {
		z.SpreadBasisPoints = make(map[string]*big.Int, zb0006)
	} else if len(z.SpreadBasisPoints) > 0 {
		for key := range z.SpreadBasisPoints {
			delete(z.SpreadBasisPoints, key)
		}
	}
	for zb0006 > 0 {
		zb0006--
		var za0003 string
		var za0004 *big.Int
		za0003, err = dc.ReadString()
		if err != nil {
			err = msgp.WrapError(err, "SpreadBasisPoints")
			return
		}
		if dc.IsNil() {
			err = dc.ReadNil()
			if err != nil {
				err = msgp.WrapError(err, "SpreadBasisPoints", za0003)
				return
			}
			za0004 = nil
		} else {
			{
				var zb0007 []byte
				zb0007, err = dc.ReadBytes(msgpencode.EncodeInt(za0004))
				if err != nil {
					err = msgp.WrapError(err, "SpreadBasisPoints", za0003)
					return
				}
				za0004 = msgpencode.DecodeInt(zb0007)
			}
		}
		z.SpreadBasisPoints[za0003] = za0004
	}
	var zb0008 uint32
	zb0008, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err, "AdjustmentBasisPoints")
		return
	}
	if z.AdjustmentBasisPoints == nil {
		z.AdjustmentBasisPoints = make(map[string]*big.Int, zb0008)
	} else if len(z.AdjustmentBasisPoints) > 0 {
		for key := range z.AdjustmentBasisPoints {
			delete(z.AdjustmentBasisPoints, key)
		}
	}
	for zb0008 > 0 {
		zb0008--
		var za0005 string
		var za0006 *big.Int
		za0005, err = dc.ReadString()
		if err != nil {
			err = msgp.WrapError(err, "AdjustmentBasisPoints")
			return
		}
		if dc.IsNil() {
			err = dc.ReadNil()
			if err != nil {
				err = msgp.WrapError(err, "AdjustmentBasisPoints", za0005)
				return
			}
			za0006 = nil
		} else {
			{
				var zb0009 []byte
				zb0009, err = dc.ReadBytes(msgpencode.EncodeInt(za0006))
				if err != nil {
					err = msgp.WrapError(err, "AdjustmentBasisPoints", za0005)
					return
				}
				za0006 = msgpencode.DecodeInt(zb0009)
			}
		}
		z.AdjustmentBasisPoints[za0005] = za0006
	}
	var zb0010 uint32
	zb0010, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err, "StrictStableTokens")
		return
	}
	if z.StrictStableTokens == nil {
		z.StrictStableTokens = make(map[string]bool, zb0010)
	} else if len(z.StrictStableTokens) > 0 {
		for key := range z.StrictStableTokens {
			delete(z.StrictStableTokens, key)
		}
	}
	for zb0010 > 0 {
		zb0010--
		var za0007 string
		var za0008 bool
		za0007, err = dc.ReadString()
		if err != nil {
			err = msgp.WrapError(err, "StrictStableTokens")
			return
		}
		za0008, err = dc.ReadBool()
		if err != nil {
			err = msgp.WrapError(err, "StrictStableTokens", za0007)
			return
		}
		z.StrictStableTokens[za0007] = za0008
	}
	var zb0011 uint32
	zb0011, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err, "IsAdjustmentAdditive")
		return
	}
	if z.IsAdjustmentAdditive == nil {
		z.IsAdjustmentAdditive = make(map[string]bool, zb0011)
	} else if len(z.IsAdjustmentAdditive) > 0 {
		for key := range z.IsAdjustmentAdditive {
			delete(z.IsAdjustmentAdditive, key)
		}
	}
	for zb0011 > 0 {
		zb0011--
		var za0009 string
		var za0010 bool
		za0009, err = dc.ReadString()
		if err != nil {
			err = msgp.WrapError(err, "IsAdjustmentAdditive")
			return
		}
		za0010, err = dc.ReadBool()
		if err != nil {
			err = msgp.WrapError(err, "IsAdjustmentAdditive", za0009)
			return
		}
		z.IsAdjustmentAdditive[za0009] = za0010
	}
	{
		var zb0012 []byte
		zb0012, err = dc.ReadBytes((common.Address).Bytes(z.SecondaryPriceFeedAddress))
		if err != nil {
			err = msgp.WrapError(err, "SecondaryPriceFeedAddress")
			return
		}
		z.SecondaryPriceFeedAddress = common.BytesToAddress(zb0012)
	}
	z.SecondaryPriceFeedVersion, err = dc.ReadInt()
	if err != nil {
		err = msgp.WrapError(err, "SecondaryPriceFeedVersion")
		return
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "SecondaryPriceFeedEnum")
			return
		}
		z.SecondaryPriceFeedEnum = nil
	} else {
		if z.SecondaryPriceFeedEnum == nil {
			z.SecondaryPriceFeedEnum = new(PriceFeedEnum)
		}
		err = z.SecondaryPriceFeedEnum.DecodeMsg(dc)
		if err != nil {
			err = msgp.WrapError(err, "SecondaryPriceFeedEnum")
			return
		}
	}
	var zb0013 uint32
	zb0013, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err, "PriceFeedsAddresses")
		return
	}
	if z.PriceFeedsAddresses == nil {
		z.PriceFeedsAddresses = make(map[string]common.Address, zb0013)
	} else if len(z.PriceFeedsAddresses) > 0 {
		for key := range z.PriceFeedsAddresses {
			delete(z.PriceFeedsAddresses, key)
		}
	}
	for zb0013 > 0 {
		zb0013--
		var za0011 string
		var za0012 common.Address
		za0011, err = dc.ReadString()
		if err != nil {
			err = msgp.WrapError(err, "PriceFeedsAddresses")
			return
		}
		{
			var zb0014 []byte
			zb0014, err = dc.ReadBytes((common.Address).Bytes(za0012))
			if err != nil {
				err = msgp.WrapError(err, "PriceFeedsAddresses", za0011)
				return
			}
			za0012 = common.BytesToAddress(zb0014)
		}
		z.PriceFeedsAddresses[za0011] = za0012
	}
	var zb0015 uint32
	zb0015, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err, "PriceFeeds")
		return
	}
	if z.PriceFeeds == nil {
		z.PriceFeeds = make(map[string]*PriceFeed, zb0015)
	} else if len(z.PriceFeeds) > 0 {
		for key := range z.PriceFeeds {
			delete(z.PriceFeeds, key)
		}
	}
	for zb0015 > 0 {
		zb0015--
		var za0013 string
		var za0014 *PriceFeed
		za0013, err = dc.ReadString()
		if err != nil {
			err = msgp.WrapError(err, "PriceFeeds")
			return
		}
		if dc.IsNil() {
			err = dc.ReadNil()
			if err != nil {
				err = msgp.WrapError(err, "PriceFeeds", za0013)
				return
			}
			za0014 = nil
		} else {
			if za0014 == nil {
				za0014 = new(PriceFeed)
			}
			err = za0014.DecodeMsg(dc)
			if err != nil {
				err = msgp.WrapError(err, "PriceFeeds", za0013)
				return
			}
		}
		z.PriceFeeds[za0013] = za0014
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *VaultPriceFeed) EncodeMsg(en *msgp.Writer) (err error) {
	// array header, size 13
	err = en.Append(0x9d)
	if err != nil {
		return
	}
	err = en.WriteBool(z.IsSecondaryPriceEnabled)
	if err != nil {
		err = msgp.WrapError(err, "IsSecondaryPriceEnabled")
		return
	}
	if z.MaxStrictPriceDeviation == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = en.WriteBytes(msgpencode.EncodeInt(z.MaxStrictPriceDeviation))
		if err != nil {
			err = msgp.WrapError(err, "MaxStrictPriceDeviation")
			return
		}
	}
	if z.PriceSampleSpace == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = en.WriteBytes(msgpencode.EncodeInt(z.PriceSampleSpace))
		if err != nil {
			err = msgp.WrapError(err, "PriceSampleSpace")
			return
		}
	}
	err = en.WriteMapHeader(uint32(len(z.PriceDecimals)))
	if err != nil {
		err = msgp.WrapError(err, "PriceDecimals")
		return
	}
	for za0001, za0002 := range z.PriceDecimals {
		err = en.WriteString(za0001)
		if err != nil {
			err = msgp.WrapError(err, "PriceDecimals")
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
				err = msgp.WrapError(err, "PriceDecimals", za0001)
				return
			}
		}
	}
	err = en.WriteMapHeader(uint32(len(z.SpreadBasisPoints)))
	if err != nil {
		err = msgp.WrapError(err, "SpreadBasisPoints")
		return
	}
	for za0003, za0004 := range z.SpreadBasisPoints {
		err = en.WriteString(za0003)
		if err != nil {
			err = msgp.WrapError(err, "SpreadBasisPoints")
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
				err = msgp.WrapError(err, "SpreadBasisPoints", za0003)
				return
			}
		}
	}
	err = en.WriteMapHeader(uint32(len(z.AdjustmentBasisPoints)))
	if err != nil {
		err = msgp.WrapError(err, "AdjustmentBasisPoints")
		return
	}
	for za0005, za0006 := range z.AdjustmentBasisPoints {
		err = en.WriteString(za0005)
		if err != nil {
			err = msgp.WrapError(err, "AdjustmentBasisPoints")
			return
		}
		if za0006 == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = en.WriteBytes(msgpencode.EncodeInt(za0006))
			if err != nil {
				err = msgp.WrapError(err, "AdjustmentBasisPoints", za0005)
				return
			}
		}
	}
	err = en.WriteMapHeader(uint32(len(z.StrictStableTokens)))
	if err != nil {
		err = msgp.WrapError(err, "StrictStableTokens")
		return
	}
	for za0007, za0008 := range z.StrictStableTokens {
		err = en.WriteString(za0007)
		if err != nil {
			err = msgp.WrapError(err, "StrictStableTokens")
			return
		}
		err = en.WriteBool(za0008)
		if err != nil {
			err = msgp.WrapError(err, "StrictStableTokens", za0007)
			return
		}
	}
	err = en.WriteMapHeader(uint32(len(z.IsAdjustmentAdditive)))
	if err != nil {
		err = msgp.WrapError(err, "IsAdjustmentAdditive")
		return
	}
	for za0009, za0010 := range z.IsAdjustmentAdditive {
		err = en.WriteString(za0009)
		if err != nil {
			err = msgp.WrapError(err, "IsAdjustmentAdditive")
			return
		}
		err = en.WriteBool(za0010)
		if err != nil {
			err = msgp.WrapError(err, "IsAdjustmentAdditive", za0009)
			return
		}
	}
	err = en.WriteBytes((common.Address).Bytes(z.SecondaryPriceFeedAddress))
	if err != nil {
		err = msgp.WrapError(err, "SecondaryPriceFeedAddress")
		return
	}
	err = en.WriteInt(z.SecondaryPriceFeedVersion)
	if err != nil {
		err = msgp.WrapError(err, "SecondaryPriceFeedVersion")
		return
	}
	if z.SecondaryPriceFeedEnum == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = z.SecondaryPriceFeedEnum.EncodeMsg(en)
		if err != nil {
			err = msgp.WrapError(err, "SecondaryPriceFeedEnum")
			return
		}
	}
	err = en.WriteMapHeader(uint32(len(z.PriceFeedsAddresses)))
	if err != nil {
		err = msgp.WrapError(err, "PriceFeedsAddresses")
		return
	}
	for za0011, za0012 := range z.PriceFeedsAddresses {
		err = en.WriteString(za0011)
		if err != nil {
			err = msgp.WrapError(err, "PriceFeedsAddresses")
			return
		}
		err = en.WriteBytes((common.Address).Bytes(za0012))
		if err != nil {
			err = msgp.WrapError(err, "PriceFeedsAddresses", za0011)
			return
		}
	}
	err = en.WriteMapHeader(uint32(len(z.PriceFeeds)))
	if err != nil {
		err = msgp.WrapError(err, "PriceFeeds")
		return
	}
	for za0013, za0014 := range z.PriceFeeds {
		err = en.WriteString(za0013)
		if err != nil {
			err = msgp.WrapError(err, "PriceFeeds")
			return
		}
		if za0014 == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = za0014.EncodeMsg(en)
			if err != nil {
				err = msgp.WrapError(err, "PriceFeeds", za0013)
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *VaultPriceFeed) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 13
	o = append(o, 0x9d)
	o = msgp.AppendBool(o, z.IsSecondaryPriceEnabled)
	if z.MaxStrictPriceDeviation == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeInt(z.MaxStrictPriceDeviation))
	}
	if z.PriceSampleSpace == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeInt(z.PriceSampleSpace))
	}
	o = msgp.AppendMapHeader(o, uint32(len(z.PriceDecimals)))
	for za0001, za0002 := range z.PriceDecimals {
		o = msgp.AppendString(o, za0001)
		if za0002 == nil {
			o = msgp.AppendNil(o)
		} else {
			o = msgp.AppendBytes(o, msgpencode.EncodeInt(za0002))
		}
	}
	o = msgp.AppendMapHeader(o, uint32(len(z.SpreadBasisPoints)))
	for za0003, za0004 := range z.SpreadBasisPoints {
		o = msgp.AppendString(o, za0003)
		if za0004 == nil {
			o = msgp.AppendNil(o)
		} else {
			o = msgp.AppendBytes(o, msgpencode.EncodeInt(za0004))
		}
	}
	o = msgp.AppendMapHeader(o, uint32(len(z.AdjustmentBasisPoints)))
	for za0005, za0006 := range z.AdjustmentBasisPoints {
		o = msgp.AppendString(o, za0005)
		if za0006 == nil {
			o = msgp.AppendNil(o)
		} else {
			o = msgp.AppendBytes(o, msgpencode.EncodeInt(za0006))
		}
	}
	o = msgp.AppendMapHeader(o, uint32(len(z.StrictStableTokens)))
	for za0007, za0008 := range z.StrictStableTokens {
		o = msgp.AppendString(o, za0007)
		o = msgp.AppendBool(o, za0008)
	}
	o = msgp.AppendMapHeader(o, uint32(len(z.IsAdjustmentAdditive)))
	for za0009, za0010 := range z.IsAdjustmentAdditive {
		o = msgp.AppendString(o, za0009)
		o = msgp.AppendBool(o, za0010)
	}
	o = msgp.AppendBytes(o, (common.Address).Bytes(z.SecondaryPriceFeedAddress))
	o = msgp.AppendInt(o, z.SecondaryPriceFeedVersion)
	if z.SecondaryPriceFeedEnum == nil {
		o = msgp.AppendNil(o)
	} else {
		o, err = z.SecondaryPriceFeedEnum.MarshalMsg(o)
		if err != nil {
			err = msgp.WrapError(err, "SecondaryPriceFeedEnum")
			return
		}
	}
	o = msgp.AppendMapHeader(o, uint32(len(z.PriceFeedsAddresses)))
	for za0011, za0012 := range z.PriceFeedsAddresses {
		o = msgp.AppendString(o, za0011)
		o = msgp.AppendBytes(o, (common.Address).Bytes(za0012))
	}
	o = msgp.AppendMapHeader(o, uint32(len(z.PriceFeeds)))
	for za0013, za0014 := range z.PriceFeeds {
		o = msgp.AppendString(o, za0013)
		if za0014 == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = za0014.MarshalMsg(o)
			if err != nil {
				err = msgp.WrapError(err, "PriceFeeds", za0013)
				return
			}
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
	if zb0001 != 13 {
		err = msgp.ArrayError{Wanted: 13, Got: zb0001}
		return
	}
	z.IsSecondaryPriceEnabled, bts, err = msgp.ReadBoolBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "IsSecondaryPriceEnabled")
		return
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.MaxStrictPriceDeviation = nil
	} else {
		{
			var zb0002 []byte
			zb0002, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.MaxStrictPriceDeviation))
			if err != nil {
				err = msgp.WrapError(err, "MaxStrictPriceDeviation")
				return
			}
			z.MaxStrictPriceDeviation = msgpencode.DecodeInt(zb0002)
		}
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.PriceSampleSpace = nil
	} else {
		{
			var zb0003 []byte
			zb0003, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.PriceSampleSpace))
			if err != nil {
				err = msgp.WrapError(err, "PriceSampleSpace")
				return
			}
			z.PriceSampleSpace = msgpencode.DecodeInt(zb0003)
		}
	}
	var zb0004 uint32
	zb0004, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "PriceDecimals")
		return
	}
	if z.PriceDecimals == nil {
		z.PriceDecimals = make(map[string]*big.Int, zb0004)
	} else if len(z.PriceDecimals) > 0 {
		for key := range z.PriceDecimals {
			delete(z.PriceDecimals, key)
		}
	}
	for zb0004 > 0 {
		var za0001 string
		var za0002 *big.Int
		zb0004--
		za0001, bts, err = msgp.ReadStringBytes(bts)
		if err != nil {
			err = msgp.WrapError(err, "PriceDecimals")
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
				var zb0005 []byte
				zb0005, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(za0002))
				if err != nil {
					err = msgp.WrapError(err, "PriceDecimals", za0001)
					return
				}
				za0002 = msgpencode.DecodeInt(zb0005)
			}
		}
		z.PriceDecimals[za0001] = za0002
	}
	var zb0006 uint32
	zb0006, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "SpreadBasisPoints")
		return
	}
	if z.SpreadBasisPoints == nil {
		z.SpreadBasisPoints = make(map[string]*big.Int, zb0006)
	} else if len(z.SpreadBasisPoints) > 0 {
		for key := range z.SpreadBasisPoints {
			delete(z.SpreadBasisPoints, key)
		}
	}
	for zb0006 > 0 {
		var za0003 string
		var za0004 *big.Int
		zb0006--
		za0003, bts, err = msgp.ReadStringBytes(bts)
		if err != nil {
			err = msgp.WrapError(err, "SpreadBasisPoints")
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
				var zb0007 []byte
				zb0007, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(za0004))
				if err != nil {
					err = msgp.WrapError(err, "SpreadBasisPoints", za0003)
					return
				}
				za0004 = msgpencode.DecodeInt(zb0007)
			}
		}
		z.SpreadBasisPoints[za0003] = za0004
	}
	var zb0008 uint32
	zb0008, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "AdjustmentBasisPoints")
		return
	}
	if z.AdjustmentBasisPoints == nil {
		z.AdjustmentBasisPoints = make(map[string]*big.Int, zb0008)
	} else if len(z.AdjustmentBasisPoints) > 0 {
		for key := range z.AdjustmentBasisPoints {
			delete(z.AdjustmentBasisPoints, key)
		}
	}
	for zb0008 > 0 {
		var za0005 string
		var za0006 *big.Int
		zb0008--
		za0005, bts, err = msgp.ReadStringBytes(bts)
		if err != nil {
			err = msgp.WrapError(err, "AdjustmentBasisPoints")
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
				var zb0009 []byte
				zb0009, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(za0006))
				if err != nil {
					err = msgp.WrapError(err, "AdjustmentBasisPoints", za0005)
					return
				}
				za0006 = msgpencode.DecodeInt(zb0009)
			}
		}
		z.AdjustmentBasisPoints[za0005] = za0006
	}
	var zb0010 uint32
	zb0010, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "StrictStableTokens")
		return
	}
	if z.StrictStableTokens == nil {
		z.StrictStableTokens = make(map[string]bool, zb0010)
	} else if len(z.StrictStableTokens) > 0 {
		for key := range z.StrictStableTokens {
			delete(z.StrictStableTokens, key)
		}
	}
	for zb0010 > 0 {
		var za0007 string
		var za0008 bool
		zb0010--
		za0007, bts, err = msgp.ReadStringBytes(bts)
		if err != nil {
			err = msgp.WrapError(err, "StrictStableTokens")
			return
		}
		za0008, bts, err = msgp.ReadBoolBytes(bts)
		if err != nil {
			err = msgp.WrapError(err, "StrictStableTokens", za0007)
			return
		}
		z.StrictStableTokens[za0007] = za0008
	}
	var zb0011 uint32
	zb0011, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "IsAdjustmentAdditive")
		return
	}
	if z.IsAdjustmentAdditive == nil {
		z.IsAdjustmentAdditive = make(map[string]bool, zb0011)
	} else if len(z.IsAdjustmentAdditive) > 0 {
		for key := range z.IsAdjustmentAdditive {
			delete(z.IsAdjustmentAdditive, key)
		}
	}
	for zb0011 > 0 {
		var za0009 string
		var za0010 bool
		zb0011--
		za0009, bts, err = msgp.ReadStringBytes(bts)
		if err != nil {
			err = msgp.WrapError(err, "IsAdjustmentAdditive")
			return
		}
		za0010, bts, err = msgp.ReadBoolBytes(bts)
		if err != nil {
			err = msgp.WrapError(err, "IsAdjustmentAdditive", za0009)
			return
		}
		z.IsAdjustmentAdditive[za0009] = za0010
	}
	{
		var zb0012 []byte
		zb0012, bts, err = msgp.ReadBytesBytes(bts, (common.Address).Bytes(z.SecondaryPriceFeedAddress))
		if err != nil {
			err = msgp.WrapError(err, "SecondaryPriceFeedAddress")
			return
		}
		z.SecondaryPriceFeedAddress = common.BytesToAddress(zb0012)
	}
	z.SecondaryPriceFeedVersion, bts, err = msgp.ReadIntBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "SecondaryPriceFeedVersion")
		return
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.SecondaryPriceFeedEnum = nil
	} else {
		if z.SecondaryPriceFeedEnum == nil {
			z.SecondaryPriceFeedEnum = new(PriceFeedEnum)
		}
		bts, err = z.SecondaryPriceFeedEnum.UnmarshalMsg(bts)
		if err != nil {
			err = msgp.WrapError(err, "SecondaryPriceFeedEnum")
			return
		}
	}
	var zb0013 uint32
	zb0013, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "PriceFeedsAddresses")
		return
	}
	if z.PriceFeedsAddresses == nil {
		z.PriceFeedsAddresses = make(map[string]common.Address, zb0013)
	} else if len(z.PriceFeedsAddresses) > 0 {
		for key := range z.PriceFeedsAddresses {
			delete(z.PriceFeedsAddresses, key)
		}
	}
	for zb0013 > 0 {
		var za0011 string
		var za0012 common.Address
		zb0013--
		za0011, bts, err = msgp.ReadStringBytes(bts)
		if err != nil {
			err = msgp.WrapError(err, "PriceFeedsAddresses")
			return
		}
		{
			var zb0014 []byte
			zb0014, bts, err = msgp.ReadBytesBytes(bts, (common.Address).Bytes(za0012))
			if err != nil {
				err = msgp.WrapError(err, "PriceFeedsAddresses", za0011)
				return
			}
			za0012 = common.BytesToAddress(zb0014)
		}
		z.PriceFeedsAddresses[za0011] = za0012
	}
	var zb0015 uint32
	zb0015, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "PriceFeeds")
		return
	}
	if z.PriceFeeds == nil {
		z.PriceFeeds = make(map[string]*PriceFeed, zb0015)
	} else if len(z.PriceFeeds) > 0 {
		for key := range z.PriceFeeds {
			delete(z.PriceFeeds, key)
		}
	}
	for zb0015 > 0 {
		var za0013 string
		var za0014 *PriceFeed
		zb0015--
		za0013, bts, err = msgp.ReadStringBytes(bts)
		if err != nil {
			err = msgp.WrapError(err, "PriceFeeds")
			return
		}
		if msgp.IsNil(bts) {
			bts, err = msgp.ReadNilBytes(bts)
			if err != nil {
				return
			}
			za0014 = nil
		} else {
			if za0014 == nil {
				za0014 = new(PriceFeed)
			}
			bts, err = za0014.UnmarshalMsg(bts)
			if err != nil {
				err = msgp.WrapError(err, "PriceFeeds", za0013)
				return
			}
		}
		z.PriceFeeds[za0013] = za0014
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *VaultPriceFeed) Msgsize() (s int) {
	s = 1 + msgp.BoolSize
	if z.MaxStrictPriceDeviation == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(z.MaxStrictPriceDeviation))
	}
	if z.PriceSampleSpace == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(z.PriceSampleSpace))
	}
	s += msgp.MapHeaderSize
	if z.PriceDecimals != nil {
		for za0001, za0002 := range z.PriceDecimals {
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
	if z.SpreadBasisPoints != nil {
		for za0003, za0004 := range z.SpreadBasisPoints {
			_ = za0004
			s += msgp.StringPrefixSize + len(za0003)
			if za0004 == nil {
				s += msgp.NilSize
			} else {
				s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(za0004))
			}
		}
	}
	s += msgp.MapHeaderSize
	if z.AdjustmentBasisPoints != nil {
		for za0005, za0006 := range z.AdjustmentBasisPoints {
			_ = za0006
			s += msgp.StringPrefixSize + len(za0005)
			if za0006 == nil {
				s += msgp.NilSize
			} else {
				s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(za0006))
			}
		}
	}
	s += msgp.MapHeaderSize
	if z.StrictStableTokens != nil {
		for za0007, za0008 := range z.StrictStableTokens {
			_ = za0008
			s += msgp.StringPrefixSize + len(za0007) + msgp.BoolSize
		}
	}
	s += msgp.MapHeaderSize
	if z.IsAdjustmentAdditive != nil {
		for za0009, za0010 := range z.IsAdjustmentAdditive {
			_ = za0010
			s += msgp.StringPrefixSize + len(za0009) + msgp.BoolSize
		}
	}
	s += msgp.BytesPrefixSize + len((common.Address).Bytes(z.SecondaryPriceFeedAddress)) + msgp.IntSize
	if z.SecondaryPriceFeedEnum == nil {
		s += msgp.NilSize
	} else {
		s += z.SecondaryPriceFeedEnum.Msgsize()
	}
	s += msgp.MapHeaderSize
	if z.PriceFeedsAddresses != nil {
		for za0011, za0012 := range z.PriceFeedsAddresses {
			_ = za0012
			s += msgp.StringPrefixSize + len(za0011) + msgp.BytesPrefixSize + len((common.Address).Bytes(za0012))
		}
	}
	s += msgp.MapHeaderSize
	if z.PriceFeeds != nil {
		for za0013, za0014 := range z.PriceFeeds {
			_ = za0014
			s += msgp.StringPrefixSize + len(za0013)
			if za0014 == nil {
				s += msgp.NilSize
			} else {
				s += za0014.Msgsize()
			}
		}
	}
	return
}
