package fxdx

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
	if zb0001 != 28 {
		err = msgp.ArrayError{Wanted: 28, Got: zb0001}
		return
	}
	z.Address, err = dc.ReadString()
	if err != nil {
		err = msgp.WrapError(err, "Address")
		return
	}
	z.BNB, err = dc.ReadString()
	if err != nil {
		err = msgp.WrapError(err, "BNB")
		return
	}
	z.BTC, err = dc.ReadString()
	if err != nil {
		err = msgp.WrapError(err, "BTC")
		return
	}
	z.ETH, err = dc.ReadString()
	if err != nil {
		err = msgp.WrapError(err, "ETH")
		return
	}
	{
		var zb0002 []byte
		zb0002, err = dc.ReadBytes((common.Address).Bytes(z.BNBBUSDAddress))
		if err != nil {
			err = msgp.WrapError(err, "BNBBUSDAddress")
			return
		}
		z.BNBBUSDAddress = common.BytesToAddress(zb0002)
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "BNBBUSD")
			return
		}
		z.BNBBUSD = nil
	} else {
		if z.BNBBUSD == nil {
			z.BNBBUSD = new(PancakePair)
		}
		err = z.BNBBUSD.DecodeMsg(dc)
		if err != nil {
			err = msgp.WrapError(err, "BNBBUSD")
			return
		}
	}
	{
		var zb0003 []byte
		zb0003, err = dc.ReadBytes((common.Address).Bytes(z.BTCBNBAddress))
		if err != nil {
			err = msgp.WrapError(err, "BTCBNBAddress")
			return
		}
		z.BTCBNBAddress = common.BytesToAddress(zb0003)
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "BTCBNB")
			return
		}
		z.BTCBNB = nil
	} else {
		if z.BTCBNB == nil {
			z.BTCBNB = new(PancakePair)
		}
		err = z.BTCBNB.DecodeMsg(dc)
		if err != nil {
			err = msgp.WrapError(err, "BTCBNB")
			return
		}
	}
	{
		var zb0004 []byte
		zb0004, err = dc.ReadBytes((common.Address).Bytes(z.ETHBNBAddress))
		if err != nil {
			err = msgp.WrapError(err, "ETHBNBAddress")
			return
		}
		z.ETHBNBAddress = common.BytesToAddress(zb0004)
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "ETHBNB")
			return
		}
		z.ETHBNB = nil
	} else {
		if z.ETHBNB == nil {
			z.ETHBNB = new(PancakePair)
		}
		err = z.ETHBNB.DecodeMsg(dc)
		if err != nil {
			err = msgp.WrapError(err, "ETHBNB")
			return
		}
	}
	{
		var zb0005 []byte
		zb0005, err = dc.ReadBytes((common.Address).Bytes(z.ChainlinkFlagsAddress))
		if err != nil {
			err = msgp.WrapError(err, "ChainlinkFlagsAddress")
			return
		}
		z.ChainlinkFlagsAddress = common.BytesToAddress(zb0005)
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "ChainlinkFlags")
			return
		}
		z.ChainlinkFlags = nil
	} else {
		if z.ChainlinkFlags == nil {
			z.ChainlinkFlags = new(ChainlinkFlags)
		}
		err = z.ChainlinkFlags.DecodeMsg(dc)
		if err != nil {
			err = msgp.WrapError(err, "ChainlinkFlags")
			return
		}
	}
	z.FavorPrimaryPrice, err = dc.ReadBool()
	if err != nil {
		err = msgp.WrapError(err, "FavorPrimaryPrice")
		return
	}
	z.IsAmmEnabled, err = dc.ReadBool()
	if err != nil {
		err = msgp.WrapError(err, "IsAmmEnabled")
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
			var zb0006 []byte
			zb0006, err = dc.ReadBytes(msgpencode.EncodeInt(z.MaxStrictPriceDeviation))
			if err != nil {
				err = msgp.WrapError(err, "MaxStrictPriceDeviation")
				return
			}
			z.MaxStrictPriceDeviation = msgpencode.DecodeInt(zb0006)
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
			var zb0007 []byte
			zb0007, err = dc.ReadBytes(msgpencode.EncodeInt(z.PriceSampleSpace))
			if err != nil {
				err = msgp.WrapError(err, "PriceSampleSpace")
				return
			}
			z.PriceSampleSpace = msgpencode.DecodeInt(zb0007)
		}
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "SpreadThresholdBasisPoints")
			return
		}
		z.SpreadThresholdBasisPoints = nil
	} else {
		{
			var zb0008 []byte
			zb0008, err = dc.ReadBytes(msgpencode.EncodeInt(z.SpreadThresholdBasisPoints))
			if err != nil {
				err = msgp.WrapError(err, "SpreadThresholdBasisPoints")
				return
			}
			z.SpreadThresholdBasisPoints = msgpencode.DecodeInt(zb0008)
		}
	}
	z.UseV2Pricing, err = dc.ReadBool()
	if err != nil {
		err = msgp.WrapError(err, "UseV2Pricing")
		return
	}
	var zb0009 uint32
	zb0009, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err, "PriceDecimals")
		return
	}
	if z.PriceDecimals == nil {
		z.PriceDecimals = make(map[string]*big.Int, zb0009)
	} else if len(z.PriceDecimals) > 0 {
		for key := range z.PriceDecimals {
			delete(z.PriceDecimals, key)
		}
	}
	var field []byte
	_ = field
	for zb0009 > 0 {
		zb0009--
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
				var zb0010 []byte
				zb0010, err = dc.ReadBytes(msgpencode.EncodeInt(za0002))
				if err != nil {
					err = msgp.WrapError(err, "PriceDecimals", za0001)
					return
				}
				za0002 = msgpencode.DecodeInt(zb0010)
			}
		}
		z.PriceDecimals[za0001] = za0002
	}
	var zb0011 uint32
	zb0011, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err, "SpreadBasisPoints")
		return
	}
	if z.SpreadBasisPoints == nil {
		z.SpreadBasisPoints = make(map[string]*big.Int, zb0011)
	} else if len(z.SpreadBasisPoints) > 0 {
		for key := range z.SpreadBasisPoints {
			delete(z.SpreadBasisPoints, key)
		}
	}
	for zb0011 > 0 {
		zb0011--
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
				var zb0012 []byte
				zb0012, err = dc.ReadBytes(msgpencode.EncodeInt(za0004))
				if err != nil {
					err = msgp.WrapError(err, "SpreadBasisPoints", za0003)
					return
				}
				za0004 = msgpencode.DecodeInt(zb0012)
			}
		}
		z.SpreadBasisPoints[za0003] = za0004
	}
	var zb0013 uint32
	zb0013, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err, "AdjustmentBasisPoints")
		return
	}
	if z.AdjustmentBasisPoints == nil {
		z.AdjustmentBasisPoints = make(map[string]*big.Int, zb0013)
	} else if len(z.AdjustmentBasisPoints) > 0 {
		for key := range z.AdjustmentBasisPoints {
			delete(z.AdjustmentBasisPoints, key)
		}
	}
	for zb0013 > 0 {
		zb0013--
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
				var zb0014 []byte
				zb0014, err = dc.ReadBytes(msgpencode.EncodeInt(za0006))
				if err != nil {
					err = msgp.WrapError(err, "AdjustmentBasisPoints", za0005)
					return
				}
				za0006 = msgpencode.DecodeInt(zb0014)
			}
		}
		z.AdjustmentBasisPoints[za0005] = za0006
	}
	var zb0015 uint32
	zb0015, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err, "StrictStableTokens")
		return
	}
	if z.StrictStableTokens == nil {
		z.StrictStableTokens = make(map[string]bool, zb0015)
	} else if len(z.StrictStableTokens) > 0 {
		for key := range z.StrictStableTokens {
			delete(z.StrictStableTokens, key)
		}
	}
	for zb0015 > 0 {
		zb0015--
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
	var zb0016 uint32
	zb0016, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err, "IsAdjustmentAdditive")
		return
	}
	if z.IsAdjustmentAdditive == nil {
		z.IsAdjustmentAdditive = make(map[string]bool, zb0016)
	} else if len(z.IsAdjustmentAdditive) > 0 {
		for key := range z.IsAdjustmentAdditive {
			delete(z.IsAdjustmentAdditive, key)
		}
	}
	for zb0016 > 0 {
		zb0016--
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
		var zb0017 []byte
		zb0017, err = dc.ReadBytes((common.Address).Bytes(z.SecondaryPriceFeedAddress))
		if err != nil {
			err = msgp.WrapError(err, "SecondaryPriceFeedAddress")
			return
		}
		z.SecondaryPriceFeedAddress = common.BytesToAddress(zb0017)
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "SecondaryPriceFeed")
			return
		}
		z.SecondaryPriceFeed = nil
	} else {
		if z.SecondaryPriceFeed == nil {
			z.SecondaryPriceFeed = new(FastPriceFeed)
		}
		err = z.SecondaryPriceFeed.DecodeMsg(dc)
		if err != nil {
			err = msgp.WrapError(err, "SecondaryPriceFeed")
			return
		}
	}
	var zb0018 uint32
	zb0018, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err, "PriceFeedsAddresses")
		return
	}
	if z.PriceFeedsAddresses == nil {
		z.PriceFeedsAddresses = make(map[string]common.Address, zb0018)
	} else if len(z.PriceFeedsAddresses) > 0 {
		for key := range z.PriceFeedsAddresses {
			delete(z.PriceFeedsAddresses, key)
		}
	}
	for zb0018 > 0 {
		zb0018--
		var za0011 string
		var za0012 common.Address
		za0011, err = dc.ReadString()
		if err != nil {
			err = msgp.WrapError(err, "PriceFeedsAddresses")
			return
		}
		{
			var zb0019 []byte
			zb0019, err = dc.ReadBytes((common.Address).Bytes(za0012))
			if err != nil {
				err = msgp.WrapError(err, "PriceFeedsAddresses", za0011)
				return
			}
			za0012 = common.BytesToAddress(zb0019)
		}
		z.PriceFeedsAddresses[za0011] = za0012
	}
	var zb0020 uint32
	zb0020, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err, "PriceFeeds")
		return
	}
	if z.PriceFeeds == nil {
		z.PriceFeeds = make(map[string]*PriceFeed, zb0020)
	} else if len(z.PriceFeeds) > 0 {
		for key := range z.PriceFeeds {
			delete(z.PriceFeeds, key)
		}
	}
	for zb0020 > 0 {
		zb0020--
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
	// array header, size 28
	err = en.Append(0xdc, 0x0, 0x1c)
	if err != nil {
		return
	}
	err = en.WriteString(z.Address)
	if err != nil {
		err = msgp.WrapError(err, "Address")
		return
	}
	err = en.WriteString(z.BNB)
	if err != nil {
		err = msgp.WrapError(err, "BNB")
		return
	}
	err = en.WriteString(z.BTC)
	if err != nil {
		err = msgp.WrapError(err, "BTC")
		return
	}
	err = en.WriteString(z.ETH)
	if err != nil {
		err = msgp.WrapError(err, "ETH")
		return
	}
	err = en.WriteBytes((common.Address).Bytes(z.BNBBUSDAddress))
	if err != nil {
		err = msgp.WrapError(err, "BNBBUSDAddress")
		return
	}
	if z.BNBBUSD == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = z.BNBBUSD.EncodeMsg(en)
		if err != nil {
			err = msgp.WrapError(err, "BNBBUSD")
			return
		}
	}
	err = en.WriteBytes((common.Address).Bytes(z.BTCBNBAddress))
	if err != nil {
		err = msgp.WrapError(err, "BTCBNBAddress")
		return
	}
	if z.BTCBNB == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = z.BTCBNB.EncodeMsg(en)
		if err != nil {
			err = msgp.WrapError(err, "BTCBNB")
			return
		}
	}
	err = en.WriteBytes((common.Address).Bytes(z.ETHBNBAddress))
	if err != nil {
		err = msgp.WrapError(err, "ETHBNBAddress")
		return
	}
	if z.ETHBNB == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = z.ETHBNB.EncodeMsg(en)
		if err != nil {
			err = msgp.WrapError(err, "ETHBNB")
			return
		}
	}
	err = en.WriteBytes((common.Address).Bytes(z.ChainlinkFlagsAddress))
	if err != nil {
		err = msgp.WrapError(err, "ChainlinkFlagsAddress")
		return
	}
	if z.ChainlinkFlags == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = z.ChainlinkFlags.EncodeMsg(en)
		if err != nil {
			err = msgp.WrapError(err, "ChainlinkFlags")
			return
		}
	}
	err = en.WriteBool(z.FavorPrimaryPrice)
	if err != nil {
		err = msgp.WrapError(err, "FavorPrimaryPrice")
		return
	}
	err = en.WriteBool(z.IsAmmEnabled)
	if err != nil {
		err = msgp.WrapError(err, "IsAmmEnabled")
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
	if z.SpreadThresholdBasisPoints == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = en.WriteBytes(msgpencode.EncodeInt(z.SpreadThresholdBasisPoints))
		if err != nil {
			err = msgp.WrapError(err, "SpreadThresholdBasisPoints")
			return
		}
	}
	err = en.WriteBool(z.UseV2Pricing)
	if err != nil {
		err = msgp.WrapError(err, "UseV2Pricing")
		return
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
	if z.SecondaryPriceFeed == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = z.SecondaryPriceFeed.EncodeMsg(en)
		if err != nil {
			err = msgp.WrapError(err, "SecondaryPriceFeed")
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
	// array header, size 28
	o = append(o, 0xdc, 0x0, 0x1c)
	o = msgp.AppendString(o, z.Address)
	o = msgp.AppendString(o, z.BNB)
	o = msgp.AppendString(o, z.BTC)
	o = msgp.AppendString(o, z.ETH)
	o = msgp.AppendBytes(o, (common.Address).Bytes(z.BNBBUSDAddress))
	if z.BNBBUSD == nil {
		o = msgp.AppendNil(o)
	} else {
		o, err = z.BNBBUSD.MarshalMsg(o)
		if err != nil {
			err = msgp.WrapError(err, "BNBBUSD")
			return
		}
	}
	o = msgp.AppendBytes(o, (common.Address).Bytes(z.BTCBNBAddress))
	if z.BTCBNB == nil {
		o = msgp.AppendNil(o)
	} else {
		o, err = z.BTCBNB.MarshalMsg(o)
		if err != nil {
			err = msgp.WrapError(err, "BTCBNB")
			return
		}
	}
	o = msgp.AppendBytes(o, (common.Address).Bytes(z.ETHBNBAddress))
	if z.ETHBNB == nil {
		o = msgp.AppendNil(o)
	} else {
		o, err = z.ETHBNB.MarshalMsg(o)
		if err != nil {
			err = msgp.WrapError(err, "ETHBNB")
			return
		}
	}
	o = msgp.AppendBytes(o, (common.Address).Bytes(z.ChainlinkFlagsAddress))
	if z.ChainlinkFlags == nil {
		o = msgp.AppendNil(o)
	} else {
		o, err = z.ChainlinkFlags.MarshalMsg(o)
		if err != nil {
			err = msgp.WrapError(err, "ChainlinkFlags")
			return
		}
	}
	o = msgp.AppendBool(o, z.FavorPrimaryPrice)
	o = msgp.AppendBool(o, z.IsAmmEnabled)
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
	if z.SpreadThresholdBasisPoints == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeInt(z.SpreadThresholdBasisPoints))
	}
	o = msgp.AppendBool(o, z.UseV2Pricing)
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
	if z.SecondaryPriceFeed == nil {
		o = msgp.AppendNil(o)
	} else {
		o, err = z.SecondaryPriceFeed.MarshalMsg(o)
		if err != nil {
			err = msgp.WrapError(err, "SecondaryPriceFeed")
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
	if zb0001 != 28 {
		err = msgp.ArrayError{Wanted: 28, Got: zb0001}
		return
	}
	z.Address, bts, err = msgp.ReadStringBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "Address")
		return
	}
	z.BNB, bts, err = msgp.ReadStringBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "BNB")
		return
	}
	z.BTC, bts, err = msgp.ReadStringBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "BTC")
		return
	}
	z.ETH, bts, err = msgp.ReadStringBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "ETH")
		return
	}
	{
		var zb0002 []byte
		zb0002, bts, err = msgp.ReadBytesBytes(bts, (common.Address).Bytes(z.BNBBUSDAddress))
		if err != nil {
			err = msgp.WrapError(err, "BNBBUSDAddress")
			return
		}
		z.BNBBUSDAddress = common.BytesToAddress(zb0002)
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.BNBBUSD = nil
	} else {
		if z.BNBBUSD == nil {
			z.BNBBUSD = new(PancakePair)
		}
		bts, err = z.BNBBUSD.UnmarshalMsg(bts)
		if err != nil {
			err = msgp.WrapError(err, "BNBBUSD")
			return
		}
	}
	{
		var zb0003 []byte
		zb0003, bts, err = msgp.ReadBytesBytes(bts, (common.Address).Bytes(z.BTCBNBAddress))
		if err != nil {
			err = msgp.WrapError(err, "BTCBNBAddress")
			return
		}
		z.BTCBNBAddress = common.BytesToAddress(zb0003)
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.BTCBNB = nil
	} else {
		if z.BTCBNB == nil {
			z.BTCBNB = new(PancakePair)
		}
		bts, err = z.BTCBNB.UnmarshalMsg(bts)
		if err != nil {
			err = msgp.WrapError(err, "BTCBNB")
			return
		}
	}
	{
		var zb0004 []byte
		zb0004, bts, err = msgp.ReadBytesBytes(bts, (common.Address).Bytes(z.ETHBNBAddress))
		if err != nil {
			err = msgp.WrapError(err, "ETHBNBAddress")
			return
		}
		z.ETHBNBAddress = common.BytesToAddress(zb0004)
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.ETHBNB = nil
	} else {
		if z.ETHBNB == nil {
			z.ETHBNB = new(PancakePair)
		}
		bts, err = z.ETHBNB.UnmarshalMsg(bts)
		if err != nil {
			err = msgp.WrapError(err, "ETHBNB")
			return
		}
	}
	{
		var zb0005 []byte
		zb0005, bts, err = msgp.ReadBytesBytes(bts, (common.Address).Bytes(z.ChainlinkFlagsAddress))
		if err != nil {
			err = msgp.WrapError(err, "ChainlinkFlagsAddress")
			return
		}
		z.ChainlinkFlagsAddress = common.BytesToAddress(zb0005)
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.ChainlinkFlags = nil
	} else {
		if z.ChainlinkFlags == nil {
			z.ChainlinkFlags = new(ChainlinkFlags)
		}
		bts, err = z.ChainlinkFlags.UnmarshalMsg(bts)
		if err != nil {
			err = msgp.WrapError(err, "ChainlinkFlags")
			return
		}
	}
	z.FavorPrimaryPrice, bts, err = msgp.ReadBoolBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "FavorPrimaryPrice")
		return
	}
	z.IsAmmEnabled, bts, err = msgp.ReadBoolBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "IsAmmEnabled")
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
			var zb0006 []byte
			zb0006, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.MaxStrictPriceDeviation))
			if err != nil {
				err = msgp.WrapError(err, "MaxStrictPriceDeviation")
				return
			}
			z.MaxStrictPriceDeviation = msgpencode.DecodeInt(zb0006)
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
			var zb0007 []byte
			zb0007, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.PriceSampleSpace))
			if err != nil {
				err = msgp.WrapError(err, "PriceSampleSpace")
				return
			}
			z.PriceSampleSpace = msgpencode.DecodeInt(zb0007)
		}
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.SpreadThresholdBasisPoints = nil
	} else {
		{
			var zb0008 []byte
			zb0008, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.SpreadThresholdBasisPoints))
			if err != nil {
				err = msgp.WrapError(err, "SpreadThresholdBasisPoints")
				return
			}
			z.SpreadThresholdBasisPoints = msgpencode.DecodeInt(zb0008)
		}
	}
	z.UseV2Pricing, bts, err = msgp.ReadBoolBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "UseV2Pricing")
		return
	}
	var zb0009 uint32
	zb0009, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "PriceDecimals")
		return
	}
	if z.PriceDecimals == nil {
		z.PriceDecimals = make(map[string]*big.Int, zb0009)
	} else if len(z.PriceDecimals) > 0 {
		for key := range z.PriceDecimals {
			delete(z.PriceDecimals, key)
		}
	}
	var field []byte
	_ = field
	for zb0009 > 0 {
		var za0001 string
		var za0002 *big.Int
		zb0009--
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
				var zb0010 []byte
				zb0010, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(za0002))
				if err != nil {
					err = msgp.WrapError(err, "PriceDecimals", za0001)
					return
				}
				za0002 = msgpencode.DecodeInt(zb0010)
			}
		}
		z.PriceDecimals[za0001] = za0002
	}
	var zb0011 uint32
	zb0011, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "SpreadBasisPoints")
		return
	}
	if z.SpreadBasisPoints == nil {
		z.SpreadBasisPoints = make(map[string]*big.Int, zb0011)
	} else if len(z.SpreadBasisPoints) > 0 {
		for key := range z.SpreadBasisPoints {
			delete(z.SpreadBasisPoints, key)
		}
	}
	for zb0011 > 0 {
		var za0003 string
		var za0004 *big.Int
		zb0011--
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
				var zb0012 []byte
				zb0012, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(za0004))
				if err != nil {
					err = msgp.WrapError(err, "SpreadBasisPoints", za0003)
					return
				}
				za0004 = msgpencode.DecodeInt(zb0012)
			}
		}
		z.SpreadBasisPoints[za0003] = za0004
	}
	var zb0013 uint32
	zb0013, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "AdjustmentBasisPoints")
		return
	}
	if z.AdjustmentBasisPoints == nil {
		z.AdjustmentBasisPoints = make(map[string]*big.Int, zb0013)
	} else if len(z.AdjustmentBasisPoints) > 0 {
		for key := range z.AdjustmentBasisPoints {
			delete(z.AdjustmentBasisPoints, key)
		}
	}
	for zb0013 > 0 {
		var za0005 string
		var za0006 *big.Int
		zb0013--
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
				var zb0014 []byte
				zb0014, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(za0006))
				if err != nil {
					err = msgp.WrapError(err, "AdjustmentBasisPoints", za0005)
					return
				}
				za0006 = msgpencode.DecodeInt(zb0014)
			}
		}
		z.AdjustmentBasisPoints[za0005] = za0006
	}
	var zb0015 uint32
	zb0015, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "StrictStableTokens")
		return
	}
	if z.StrictStableTokens == nil {
		z.StrictStableTokens = make(map[string]bool, zb0015)
	} else if len(z.StrictStableTokens) > 0 {
		for key := range z.StrictStableTokens {
			delete(z.StrictStableTokens, key)
		}
	}
	for zb0015 > 0 {
		var za0007 string
		var za0008 bool
		zb0015--
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
	var zb0016 uint32
	zb0016, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "IsAdjustmentAdditive")
		return
	}
	if z.IsAdjustmentAdditive == nil {
		z.IsAdjustmentAdditive = make(map[string]bool, zb0016)
	} else if len(z.IsAdjustmentAdditive) > 0 {
		for key := range z.IsAdjustmentAdditive {
			delete(z.IsAdjustmentAdditive, key)
		}
	}
	for zb0016 > 0 {
		var za0009 string
		var za0010 bool
		zb0016--
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
		var zb0017 []byte
		zb0017, bts, err = msgp.ReadBytesBytes(bts, (common.Address).Bytes(z.SecondaryPriceFeedAddress))
		if err != nil {
			err = msgp.WrapError(err, "SecondaryPriceFeedAddress")
			return
		}
		z.SecondaryPriceFeedAddress = common.BytesToAddress(zb0017)
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.SecondaryPriceFeed = nil
	} else {
		if z.SecondaryPriceFeed == nil {
			z.SecondaryPriceFeed = new(FastPriceFeed)
		}
		bts, err = z.SecondaryPriceFeed.UnmarshalMsg(bts)
		if err != nil {
			err = msgp.WrapError(err, "SecondaryPriceFeed")
			return
		}
	}
	var zb0018 uint32
	zb0018, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "PriceFeedsAddresses")
		return
	}
	if z.PriceFeedsAddresses == nil {
		z.PriceFeedsAddresses = make(map[string]common.Address, zb0018)
	} else if len(z.PriceFeedsAddresses) > 0 {
		for key := range z.PriceFeedsAddresses {
			delete(z.PriceFeedsAddresses, key)
		}
	}
	for zb0018 > 0 {
		var za0011 string
		var za0012 common.Address
		zb0018--
		za0011, bts, err = msgp.ReadStringBytes(bts)
		if err != nil {
			err = msgp.WrapError(err, "PriceFeedsAddresses")
			return
		}
		{
			var zb0019 []byte
			zb0019, bts, err = msgp.ReadBytesBytes(bts, (common.Address).Bytes(za0012))
			if err != nil {
				err = msgp.WrapError(err, "PriceFeedsAddresses", za0011)
				return
			}
			za0012 = common.BytesToAddress(zb0019)
		}
		z.PriceFeedsAddresses[za0011] = za0012
	}
	var zb0020 uint32
	zb0020, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "PriceFeeds")
		return
	}
	if z.PriceFeeds == nil {
		z.PriceFeeds = make(map[string]*PriceFeed, zb0020)
	} else if len(z.PriceFeeds) > 0 {
		for key := range z.PriceFeeds {
			delete(z.PriceFeeds, key)
		}
	}
	for zb0020 > 0 {
		var za0013 string
		var za0014 *PriceFeed
		zb0020--
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
	s = 3 + msgp.StringPrefixSize + len(z.Address) + msgp.StringPrefixSize + len(z.BNB) + msgp.StringPrefixSize + len(z.BTC) + msgp.StringPrefixSize + len(z.ETH) + msgp.BytesPrefixSize + len((common.Address).Bytes(z.BNBBUSDAddress))
	if z.BNBBUSD == nil {
		s += msgp.NilSize
	} else {
		s += z.BNBBUSD.Msgsize()
	}
	s += msgp.BytesPrefixSize + len((common.Address).Bytes(z.BTCBNBAddress))
	if z.BTCBNB == nil {
		s += msgp.NilSize
	} else {
		s += z.BTCBNB.Msgsize()
	}
	s += msgp.BytesPrefixSize + len((common.Address).Bytes(z.ETHBNBAddress))
	if z.ETHBNB == nil {
		s += msgp.NilSize
	} else {
		s += z.ETHBNB.Msgsize()
	}
	s += msgp.BytesPrefixSize + len((common.Address).Bytes(z.ChainlinkFlagsAddress))
	if z.ChainlinkFlags == nil {
		s += msgp.NilSize
	} else {
		s += z.ChainlinkFlags.Msgsize()
	}
	s += msgp.BoolSize + msgp.BoolSize + msgp.BoolSize
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
	if z.SpreadThresholdBasisPoints == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(z.SpreadThresholdBasisPoints))
	}
	s += msgp.BoolSize + msgp.MapHeaderSize
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
	s += msgp.BytesPrefixSize + len((common.Address).Bytes(z.SecondaryPriceFeedAddress))
	if z.SecondaryPriceFeed == nil {
		s += msgp.NilSize
	} else {
		s += z.SecondaryPriceFeed.Msgsize()
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
