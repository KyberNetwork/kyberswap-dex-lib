package gmx

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpencode"
	"github.com/ethereum/go-ethereum/common"
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *Vault) DecodeMsg(dc *msgp.Reader) (err error) {
	var zb0001 uint32
	zb0001, err = dc.ReadArrayHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	if zb0001 != 23 {
		err = msgp.ArrayError{Wanted: 23, Got: zb0001}
		return
	}
	z.HasDynamicFees, err = dc.ReadBool()
	if err != nil {
		err = msgp.WrapError(err, "HasDynamicFees")
		return
	}
	z.IncludeAmmPrice, err = dc.ReadBool()
	if err != nil {
		err = msgp.WrapError(err, "IncludeAmmPrice")
		return
	}
	z.IsSwapEnabled, err = dc.ReadBool()
	if err != nil {
		err = msgp.WrapError(err, "IsSwapEnabled")
		return
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "StableSwapFeeBasisPoints")
			return
		}
		z.StableSwapFeeBasisPoints = nil
	} else {
		{
			var zb0002 []byte
			zb0002, err = dc.ReadBytes(msgpencode.EncodeInt(z.StableSwapFeeBasisPoints))
			if err != nil {
				err = msgp.WrapError(err, "StableSwapFeeBasisPoints")
				return
			}
			z.StableSwapFeeBasisPoints = msgpencode.DecodeInt(zb0002)
		}
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "StableTaxBasisPoints")
			return
		}
		z.StableTaxBasisPoints = nil
	} else {
		{
			var zb0003 []byte
			zb0003, err = dc.ReadBytes(msgpencode.EncodeInt(z.StableTaxBasisPoints))
			if err != nil {
				err = msgp.WrapError(err, "StableTaxBasisPoints")
				return
			}
			z.StableTaxBasisPoints = msgpencode.DecodeInt(zb0003)
		}
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "SwapFeeBasisPoints")
			return
		}
		z.SwapFeeBasisPoints = nil
	} else {
		{
			var zb0004 []byte
			zb0004, err = dc.ReadBytes(msgpencode.EncodeInt(z.SwapFeeBasisPoints))
			if err != nil {
				err = msgp.WrapError(err, "SwapFeeBasisPoints")
				return
			}
			z.SwapFeeBasisPoints = msgpencode.DecodeInt(zb0004)
		}
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "TaxBasisPoints")
			return
		}
		z.TaxBasisPoints = nil
	} else {
		{
			var zb0005 []byte
			zb0005, err = dc.ReadBytes(msgpencode.EncodeInt(z.TaxBasisPoints))
			if err != nil {
				err = msgp.WrapError(err, "TaxBasisPoints")
				return
			}
			z.TaxBasisPoints = msgpencode.DecodeInt(zb0005)
		}
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "TotalTokenWeights")
			return
		}
		z.TotalTokenWeights = nil
	} else {
		{
			var zb0006 []byte
			zb0006, err = dc.ReadBytes(msgpencode.EncodeInt(z.TotalTokenWeights))
			if err != nil {
				err = msgp.WrapError(err, "TotalTokenWeights")
				return
			}
			z.TotalTokenWeights = msgpencode.DecodeInt(zb0006)
		}
	}
	var zb0007 uint32
	zb0007, err = dc.ReadArrayHeader()
	if err != nil {
		err = msgp.WrapError(err, "WhitelistedTokens")
		return
	}
	if cap(z.WhitelistedTokens) >= int(zb0007) {
		z.WhitelistedTokens = (z.WhitelistedTokens)[:zb0007]
	} else {
		z.WhitelistedTokens = make([]string, zb0007)
	}
	for za0001 := range z.WhitelistedTokens {
		z.WhitelistedTokens[za0001], err = dc.ReadString()
		if err != nil {
			err = msgp.WrapError(err, "WhitelistedTokens", za0001)
			return
		}
	}
	var zb0008 uint32
	zb0008, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err, "PoolAmounts")
		return
	}
	if z.PoolAmounts == nil {
		z.PoolAmounts = make(map[string]*big.Int, zb0008)
	} else if len(z.PoolAmounts) > 0 {
		for key := range z.PoolAmounts {
			delete(z.PoolAmounts, key)
		}
	}
	var field []byte
	_ = field
	for zb0008 > 0 {
		zb0008--
		var za0002 string
		var za0003 *big.Int
		za0002, err = dc.ReadString()
		if err != nil {
			err = msgp.WrapError(err, "PoolAmounts")
			return
		}
		if dc.IsNil() {
			err = dc.ReadNil()
			if err != nil {
				err = msgp.WrapError(err, "PoolAmounts", za0002)
				return
			}
			za0003 = nil
		} else {
			{
				var zb0009 []byte
				zb0009, err = dc.ReadBytes(msgpencode.EncodeInt(za0003))
				if err != nil {
					err = msgp.WrapError(err, "PoolAmounts", za0002)
					return
				}
				za0003 = msgpencode.DecodeInt(zb0009)
			}
		}
		z.PoolAmounts[za0002] = za0003
	}
	var zb0010 uint32
	zb0010, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err, "BufferAmounts")
		return
	}
	if z.BufferAmounts == nil {
		z.BufferAmounts = make(map[string]*big.Int, zb0010)
	} else if len(z.BufferAmounts) > 0 {
		for key := range z.BufferAmounts {
			delete(z.BufferAmounts, key)
		}
	}
	for zb0010 > 0 {
		zb0010--
		var za0004 string
		var za0005 *big.Int
		za0004, err = dc.ReadString()
		if err != nil {
			err = msgp.WrapError(err, "BufferAmounts")
			return
		}
		if dc.IsNil() {
			err = dc.ReadNil()
			if err != nil {
				err = msgp.WrapError(err, "BufferAmounts", za0004)
				return
			}
			za0005 = nil
		} else {
			{
				var zb0011 []byte
				zb0011, err = dc.ReadBytes(msgpencode.EncodeInt(za0005))
				if err != nil {
					err = msgp.WrapError(err, "BufferAmounts", za0004)
					return
				}
				za0005 = msgpencode.DecodeInt(zb0011)
			}
		}
		z.BufferAmounts[za0004] = za0005
	}
	var zb0012 uint32
	zb0012, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err, "ReservedAmounts")
		return
	}
	if z.ReservedAmounts == nil {
		z.ReservedAmounts = make(map[string]*big.Int, zb0012)
	} else if len(z.ReservedAmounts) > 0 {
		for key := range z.ReservedAmounts {
			delete(z.ReservedAmounts, key)
		}
	}
	for zb0012 > 0 {
		zb0012--
		var za0006 string
		var za0007 *big.Int
		za0006, err = dc.ReadString()
		if err != nil {
			err = msgp.WrapError(err, "ReservedAmounts")
			return
		}
		if dc.IsNil() {
			err = dc.ReadNil()
			if err != nil {
				err = msgp.WrapError(err, "ReservedAmounts", za0006)
				return
			}
			za0007 = nil
		} else {
			{
				var zb0013 []byte
				zb0013, err = dc.ReadBytes(msgpencode.EncodeInt(za0007))
				if err != nil {
					err = msgp.WrapError(err, "ReservedAmounts", za0006)
					return
				}
				za0007 = msgpencode.DecodeInt(zb0013)
			}
		}
		z.ReservedAmounts[za0006] = za0007
	}
	var zb0014 uint32
	zb0014, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err, "TokenDecimals")
		return
	}
	if z.TokenDecimals == nil {
		z.TokenDecimals = make(map[string]*big.Int, zb0014)
	} else if len(z.TokenDecimals) > 0 {
		for key := range z.TokenDecimals {
			delete(z.TokenDecimals, key)
		}
	}
	for zb0014 > 0 {
		zb0014--
		var za0008 string
		var za0009 *big.Int
		za0008, err = dc.ReadString()
		if err != nil {
			err = msgp.WrapError(err, "TokenDecimals")
			return
		}
		if dc.IsNil() {
			err = dc.ReadNil()
			if err != nil {
				err = msgp.WrapError(err, "TokenDecimals", za0008)
				return
			}
			za0009 = nil
		} else {
			{
				var zb0015 []byte
				zb0015, err = dc.ReadBytes(msgpencode.EncodeInt(za0009))
				if err != nil {
					err = msgp.WrapError(err, "TokenDecimals", za0008)
					return
				}
				za0009 = msgpencode.DecodeInt(zb0015)
			}
		}
		z.TokenDecimals[za0008] = za0009
	}
	var zb0016 uint32
	zb0016, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err, "StableTokens")
		return
	}
	if z.StableTokens == nil {
		z.StableTokens = make(map[string]bool, zb0016)
	} else if len(z.StableTokens) > 0 {
		for key := range z.StableTokens {
			delete(z.StableTokens, key)
		}
	}
	for zb0016 > 0 {
		zb0016--
		var za0010 string
		var za0011 bool
		za0010, err = dc.ReadString()
		if err != nil {
			err = msgp.WrapError(err, "StableTokens")
			return
		}
		za0011, err = dc.ReadBool()
		if err != nil {
			err = msgp.WrapError(err, "StableTokens", za0010)
			return
		}
		z.StableTokens[za0010] = za0011
	}
	var zb0017 uint32
	zb0017, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err, "USDGAmounts")
		return
	}
	if z.USDGAmounts == nil {
		z.USDGAmounts = make(map[string]*big.Int, zb0017)
	} else if len(z.USDGAmounts) > 0 {
		for key := range z.USDGAmounts {
			delete(z.USDGAmounts, key)
		}
	}
	for zb0017 > 0 {
		zb0017--
		var za0012 string
		var za0013 *big.Int
		za0012, err = dc.ReadString()
		if err != nil {
			err = msgp.WrapError(err, "USDGAmounts")
			return
		}
		if dc.IsNil() {
			err = dc.ReadNil()
			if err != nil {
				err = msgp.WrapError(err, "USDGAmounts", za0012)
				return
			}
			za0013 = nil
		} else {
			{
				var zb0018 []byte
				zb0018, err = dc.ReadBytes(msgpencode.EncodeInt(za0013))
				if err != nil {
					err = msgp.WrapError(err, "USDGAmounts", za0012)
					return
				}
				za0013 = msgpencode.DecodeInt(zb0018)
			}
		}
		z.USDGAmounts[za0012] = za0013
	}
	var zb0019 uint32
	zb0019, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err, "MaxUSDGAmounts")
		return
	}
	if z.MaxUSDGAmounts == nil {
		z.MaxUSDGAmounts = make(map[string]*big.Int, zb0019)
	} else if len(z.MaxUSDGAmounts) > 0 {
		for key := range z.MaxUSDGAmounts {
			delete(z.MaxUSDGAmounts, key)
		}
	}
	for zb0019 > 0 {
		zb0019--
		var za0014 string
		var za0015 *big.Int
		za0014, err = dc.ReadString()
		if err != nil {
			err = msgp.WrapError(err, "MaxUSDGAmounts")
			return
		}
		if dc.IsNil() {
			err = dc.ReadNil()
			if err != nil {
				err = msgp.WrapError(err, "MaxUSDGAmounts", za0014)
				return
			}
			za0015 = nil
		} else {
			{
				var zb0020 []byte
				zb0020, err = dc.ReadBytes(msgpencode.EncodeInt(za0015))
				if err != nil {
					err = msgp.WrapError(err, "MaxUSDGAmounts", za0014)
					return
				}
				za0015 = msgpencode.DecodeInt(zb0020)
			}
		}
		z.MaxUSDGAmounts[za0014] = za0015
	}
	var zb0021 uint32
	zb0021, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err, "TokenWeights")
		return
	}
	if z.TokenWeights == nil {
		z.TokenWeights = make(map[string]*big.Int, zb0021)
	} else if len(z.TokenWeights) > 0 {
		for key := range z.TokenWeights {
			delete(z.TokenWeights, key)
		}
	}
	for zb0021 > 0 {
		zb0021--
		var za0016 string
		var za0017 *big.Int
		za0016, err = dc.ReadString()
		if err != nil {
			err = msgp.WrapError(err, "TokenWeights")
			return
		}
		if dc.IsNil() {
			err = dc.ReadNil()
			if err != nil {
				err = msgp.WrapError(err, "TokenWeights", za0016)
				return
			}
			za0017 = nil
		} else {
			{
				var zb0022 []byte
				zb0022, err = dc.ReadBytes(msgpencode.EncodeInt(za0017))
				if err != nil {
					err = msgp.WrapError(err, "TokenWeights", za0016)
					return
				}
				za0017 = msgpencode.DecodeInt(zb0022)
			}
		}
		z.TokenWeights[za0016] = za0017
	}
	{
		var zb0023 []byte
		zb0023, err = dc.ReadBytes((common.Address).Bytes(z.PriceFeedAddress))
		if err != nil {
			err = msgp.WrapError(err, "PriceFeedAddress")
			return
		}
		z.PriceFeedAddress = common.BytesToAddress(zb0023)
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "PriceFeed")
			return
		}
		z.PriceFeed = nil
	} else {
		if z.PriceFeed == nil {
			z.PriceFeed = new(VaultPriceFeed)
		}
		err = z.PriceFeed.DecodeMsg(dc)
		if err != nil {
			err = msgp.WrapError(err, "PriceFeed")
			return
		}
	}
	{
		var zb0024 []byte
		zb0024, err = dc.ReadBytes((common.Address).Bytes(z.USDGAddress))
		if err != nil {
			err = msgp.WrapError(err, "USDGAddress")
			return
		}
		z.USDGAddress = common.BytesToAddress(zb0024)
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "USDG")
			return
		}
		z.USDG = nil
	} else {
		if z.USDG == nil {
			z.USDG = new(USDG)
		}
		err = z.USDG.DecodeMsg(dc)
		if err != nil {
			err = msgp.WrapError(err, "USDG")
			return
		}
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "WhitelistedTokensCount")
			return
		}
		z.WhitelistedTokensCount = nil
	} else {
		{
			var zb0025 []byte
			zb0025, err = dc.ReadBytes(msgpencode.EncodeInt(z.WhitelistedTokensCount))
			if err != nil {
				err = msgp.WrapError(err, "WhitelistedTokensCount")
				return
			}
			z.WhitelistedTokensCount = msgpencode.DecodeInt(zb0025)
		}
	}
	z.UseSwapPricing, err = dc.ReadBool()
	if err != nil {
		err = msgp.WrapError(err, "UseSwapPricing")
		return
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *Vault) EncodeMsg(en *msgp.Writer) (err error) {
	// array header, size 23
	err = en.Append(0xdc, 0x0, 0x17)
	if err != nil {
		return
	}
	err = en.WriteBool(z.HasDynamicFees)
	if err != nil {
		err = msgp.WrapError(err, "HasDynamicFees")
		return
	}
	err = en.WriteBool(z.IncludeAmmPrice)
	if err != nil {
		err = msgp.WrapError(err, "IncludeAmmPrice")
		return
	}
	err = en.WriteBool(z.IsSwapEnabled)
	if err != nil {
		err = msgp.WrapError(err, "IsSwapEnabled")
		return
	}
	if z.StableSwapFeeBasisPoints == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = en.WriteBytes(msgpencode.EncodeInt(z.StableSwapFeeBasisPoints))
		if err != nil {
			err = msgp.WrapError(err, "StableSwapFeeBasisPoints")
			return
		}
	}
	if z.StableTaxBasisPoints == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = en.WriteBytes(msgpencode.EncodeInt(z.StableTaxBasisPoints))
		if err != nil {
			err = msgp.WrapError(err, "StableTaxBasisPoints")
			return
		}
	}
	if z.SwapFeeBasisPoints == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = en.WriteBytes(msgpencode.EncodeInt(z.SwapFeeBasisPoints))
		if err != nil {
			err = msgp.WrapError(err, "SwapFeeBasisPoints")
			return
		}
	}
	if z.TaxBasisPoints == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = en.WriteBytes(msgpencode.EncodeInt(z.TaxBasisPoints))
		if err != nil {
			err = msgp.WrapError(err, "TaxBasisPoints")
			return
		}
	}
	if z.TotalTokenWeights == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = en.WriteBytes(msgpencode.EncodeInt(z.TotalTokenWeights))
		if err != nil {
			err = msgp.WrapError(err, "TotalTokenWeights")
			return
		}
	}
	err = en.WriteArrayHeader(uint32(len(z.WhitelistedTokens)))
	if err != nil {
		err = msgp.WrapError(err, "WhitelistedTokens")
		return
	}
	for za0001 := range z.WhitelistedTokens {
		err = en.WriteString(z.WhitelistedTokens[za0001])
		if err != nil {
			err = msgp.WrapError(err, "WhitelistedTokens", za0001)
			return
		}
	}
	err = en.WriteMapHeader(uint32(len(z.PoolAmounts)))
	if err != nil {
		err = msgp.WrapError(err, "PoolAmounts")
		return
	}
	for za0002, za0003 := range z.PoolAmounts {
		err = en.WriteString(za0002)
		if err != nil {
			err = msgp.WrapError(err, "PoolAmounts")
			return
		}
		if za0003 == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = en.WriteBytes(msgpencode.EncodeInt(za0003))
			if err != nil {
				err = msgp.WrapError(err, "PoolAmounts", za0002)
				return
			}
		}
	}
	err = en.WriteMapHeader(uint32(len(z.BufferAmounts)))
	if err != nil {
		err = msgp.WrapError(err, "BufferAmounts")
		return
	}
	for za0004, za0005 := range z.BufferAmounts {
		err = en.WriteString(za0004)
		if err != nil {
			err = msgp.WrapError(err, "BufferAmounts")
			return
		}
		if za0005 == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = en.WriteBytes(msgpencode.EncodeInt(za0005))
			if err != nil {
				err = msgp.WrapError(err, "BufferAmounts", za0004)
				return
			}
		}
	}
	err = en.WriteMapHeader(uint32(len(z.ReservedAmounts)))
	if err != nil {
		err = msgp.WrapError(err, "ReservedAmounts")
		return
	}
	for za0006, za0007 := range z.ReservedAmounts {
		err = en.WriteString(za0006)
		if err != nil {
			err = msgp.WrapError(err, "ReservedAmounts")
			return
		}
		if za0007 == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = en.WriteBytes(msgpencode.EncodeInt(za0007))
			if err != nil {
				err = msgp.WrapError(err, "ReservedAmounts", za0006)
				return
			}
		}
	}
	err = en.WriteMapHeader(uint32(len(z.TokenDecimals)))
	if err != nil {
		err = msgp.WrapError(err, "TokenDecimals")
		return
	}
	for za0008, za0009 := range z.TokenDecimals {
		err = en.WriteString(za0008)
		if err != nil {
			err = msgp.WrapError(err, "TokenDecimals")
			return
		}
		if za0009 == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = en.WriteBytes(msgpencode.EncodeInt(za0009))
			if err != nil {
				err = msgp.WrapError(err, "TokenDecimals", za0008)
				return
			}
		}
	}
	err = en.WriteMapHeader(uint32(len(z.StableTokens)))
	if err != nil {
		err = msgp.WrapError(err, "StableTokens")
		return
	}
	for za0010, za0011 := range z.StableTokens {
		err = en.WriteString(za0010)
		if err != nil {
			err = msgp.WrapError(err, "StableTokens")
			return
		}
		err = en.WriteBool(za0011)
		if err != nil {
			err = msgp.WrapError(err, "StableTokens", za0010)
			return
		}
	}
	err = en.WriteMapHeader(uint32(len(z.USDGAmounts)))
	if err != nil {
		err = msgp.WrapError(err, "USDGAmounts")
		return
	}
	for za0012, za0013 := range z.USDGAmounts {
		err = en.WriteString(za0012)
		if err != nil {
			err = msgp.WrapError(err, "USDGAmounts")
			return
		}
		if za0013 == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = en.WriteBytes(msgpencode.EncodeInt(za0013))
			if err != nil {
				err = msgp.WrapError(err, "USDGAmounts", za0012)
				return
			}
		}
	}
	err = en.WriteMapHeader(uint32(len(z.MaxUSDGAmounts)))
	if err != nil {
		err = msgp.WrapError(err, "MaxUSDGAmounts")
		return
	}
	for za0014, za0015 := range z.MaxUSDGAmounts {
		err = en.WriteString(za0014)
		if err != nil {
			err = msgp.WrapError(err, "MaxUSDGAmounts")
			return
		}
		if za0015 == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = en.WriteBytes(msgpencode.EncodeInt(za0015))
			if err != nil {
				err = msgp.WrapError(err, "MaxUSDGAmounts", za0014)
				return
			}
		}
	}
	err = en.WriteMapHeader(uint32(len(z.TokenWeights)))
	if err != nil {
		err = msgp.WrapError(err, "TokenWeights")
		return
	}
	for za0016, za0017 := range z.TokenWeights {
		err = en.WriteString(za0016)
		if err != nil {
			err = msgp.WrapError(err, "TokenWeights")
			return
		}
		if za0017 == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = en.WriteBytes(msgpencode.EncodeInt(za0017))
			if err != nil {
				err = msgp.WrapError(err, "TokenWeights", za0016)
				return
			}
		}
	}
	err = en.WriteBytes((common.Address).Bytes(z.PriceFeedAddress))
	if err != nil {
		err = msgp.WrapError(err, "PriceFeedAddress")
		return
	}
	if z.PriceFeed == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = z.PriceFeed.EncodeMsg(en)
		if err != nil {
			err = msgp.WrapError(err, "PriceFeed")
			return
		}
	}
	err = en.WriteBytes((common.Address).Bytes(z.USDGAddress))
	if err != nil {
		err = msgp.WrapError(err, "USDGAddress")
		return
	}
	if z.USDG == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = z.USDG.EncodeMsg(en)
		if err != nil {
			err = msgp.WrapError(err, "USDG")
			return
		}
	}
	if z.WhitelistedTokensCount == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = en.WriteBytes(msgpencode.EncodeInt(z.WhitelistedTokensCount))
		if err != nil {
			err = msgp.WrapError(err, "WhitelistedTokensCount")
			return
		}
	}
	err = en.WriteBool(z.UseSwapPricing)
	if err != nil {
		err = msgp.WrapError(err, "UseSwapPricing")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *Vault) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 23
	o = append(o, 0xdc, 0x0, 0x17)
	o = msgp.AppendBool(o, z.HasDynamicFees)
	o = msgp.AppendBool(o, z.IncludeAmmPrice)
	o = msgp.AppendBool(o, z.IsSwapEnabled)
	if z.StableSwapFeeBasisPoints == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeInt(z.StableSwapFeeBasisPoints))
	}
	if z.StableTaxBasisPoints == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeInt(z.StableTaxBasisPoints))
	}
	if z.SwapFeeBasisPoints == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeInt(z.SwapFeeBasisPoints))
	}
	if z.TaxBasisPoints == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeInt(z.TaxBasisPoints))
	}
	if z.TotalTokenWeights == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeInt(z.TotalTokenWeights))
	}
	o = msgp.AppendArrayHeader(o, uint32(len(z.WhitelistedTokens)))
	for za0001 := range z.WhitelistedTokens {
		o = msgp.AppendString(o, z.WhitelistedTokens[za0001])
	}
	o = msgp.AppendMapHeader(o, uint32(len(z.PoolAmounts)))
	for za0002, za0003 := range z.PoolAmounts {
		o = msgp.AppendString(o, za0002)
		if za0003 == nil {
			o = msgp.AppendNil(o)
		} else {
			o = msgp.AppendBytes(o, msgpencode.EncodeInt(za0003))
		}
	}
	o = msgp.AppendMapHeader(o, uint32(len(z.BufferAmounts)))
	for za0004, za0005 := range z.BufferAmounts {
		o = msgp.AppendString(o, za0004)
		if za0005 == nil {
			o = msgp.AppendNil(o)
		} else {
			o = msgp.AppendBytes(o, msgpencode.EncodeInt(za0005))
		}
	}
	o = msgp.AppendMapHeader(o, uint32(len(z.ReservedAmounts)))
	for za0006, za0007 := range z.ReservedAmounts {
		o = msgp.AppendString(o, za0006)
		if za0007 == nil {
			o = msgp.AppendNil(o)
		} else {
			o = msgp.AppendBytes(o, msgpencode.EncodeInt(za0007))
		}
	}
	o = msgp.AppendMapHeader(o, uint32(len(z.TokenDecimals)))
	for za0008, za0009 := range z.TokenDecimals {
		o = msgp.AppendString(o, za0008)
		if za0009 == nil {
			o = msgp.AppendNil(o)
		} else {
			o = msgp.AppendBytes(o, msgpencode.EncodeInt(za0009))
		}
	}
	o = msgp.AppendMapHeader(o, uint32(len(z.StableTokens)))
	for za0010, za0011 := range z.StableTokens {
		o = msgp.AppendString(o, za0010)
		o = msgp.AppendBool(o, za0011)
	}
	o = msgp.AppendMapHeader(o, uint32(len(z.USDGAmounts)))
	for za0012, za0013 := range z.USDGAmounts {
		o = msgp.AppendString(o, za0012)
		if za0013 == nil {
			o = msgp.AppendNil(o)
		} else {
			o = msgp.AppendBytes(o, msgpencode.EncodeInt(za0013))
		}
	}
	o = msgp.AppendMapHeader(o, uint32(len(z.MaxUSDGAmounts)))
	for za0014, za0015 := range z.MaxUSDGAmounts {
		o = msgp.AppendString(o, za0014)
		if za0015 == nil {
			o = msgp.AppendNil(o)
		} else {
			o = msgp.AppendBytes(o, msgpencode.EncodeInt(za0015))
		}
	}
	o = msgp.AppendMapHeader(o, uint32(len(z.TokenWeights)))
	for za0016, za0017 := range z.TokenWeights {
		o = msgp.AppendString(o, za0016)
		if za0017 == nil {
			o = msgp.AppendNil(o)
		} else {
			o = msgp.AppendBytes(o, msgpencode.EncodeInt(za0017))
		}
	}
	o = msgp.AppendBytes(o, (common.Address).Bytes(z.PriceFeedAddress))
	if z.PriceFeed == nil {
		o = msgp.AppendNil(o)
	} else {
		o, err = z.PriceFeed.MarshalMsg(o)
		if err != nil {
			err = msgp.WrapError(err, "PriceFeed")
			return
		}
	}
	o = msgp.AppendBytes(o, (common.Address).Bytes(z.USDGAddress))
	if z.USDG == nil {
		o = msgp.AppendNil(o)
	} else {
		o, err = z.USDG.MarshalMsg(o)
		if err != nil {
			err = msgp.WrapError(err, "USDG")
			return
		}
	}
	if z.WhitelistedTokensCount == nil {
		o = msgp.AppendNil(o)
	} else {
		o = msgp.AppendBytes(o, msgpencode.EncodeInt(z.WhitelistedTokensCount))
	}
	o = msgp.AppendBool(o, z.UseSwapPricing)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Vault) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	if zb0001 != 23 {
		err = msgp.ArrayError{Wanted: 23, Got: zb0001}
		return
	}
	z.HasDynamicFees, bts, err = msgp.ReadBoolBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "HasDynamicFees")
		return
	}
	z.IncludeAmmPrice, bts, err = msgp.ReadBoolBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "IncludeAmmPrice")
		return
	}
	z.IsSwapEnabled, bts, err = msgp.ReadBoolBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "IsSwapEnabled")
		return
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.StableSwapFeeBasisPoints = nil
	} else {
		{
			var zb0002 []byte
			zb0002, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.StableSwapFeeBasisPoints))
			if err != nil {
				err = msgp.WrapError(err, "StableSwapFeeBasisPoints")
				return
			}
			z.StableSwapFeeBasisPoints = msgpencode.DecodeInt(zb0002)
		}
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.StableTaxBasisPoints = nil
	} else {
		{
			var zb0003 []byte
			zb0003, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.StableTaxBasisPoints))
			if err != nil {
				err = msgp.WrapError(err, "StableTaxBasisPoints")
				return
			}
			z.StableTaxBasisPoints = msgpencode.DecodeInt(zb0003)
		}
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.SwapFeeBasisPoints = nil
	} else {
		{
			var zb0004 []byte
			zb0004, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.SwapFeeBasisPoints))
			if err != nil {
				err = msgp.WrapError(err, "SwapFeeBasisPoints")
				return
			}
			z.SwapFeeBasisPoints = msgpencode.DecodeInt(zb0004)
		}
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.TaxBasisPoints = nil
	} else {
		{
			var zb0005 []byte
			zb0005, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.TaxBasisPoints))
			if err != nil {
				err = msgp.WrapError(err, "TaxBasisPoints")
				return
			}
			z.TaxBasisPoints = msgpencode.DecodeInt(zb0005)
		}
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.TotalTokenWeights = nil
	} else {
		{
			var zb0006 []byte
			zb0006, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.TotalTokenWeights))
			if err != nil {
				err = msgp.WrapError(err, "TotalTokenWeights")
				return
			}
			z.TotalTokenWeights = msgpencode.DecodeInt(zb0006)
		}
	}
	var zb0007 uint32
	zb0007, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "WhitelistedTokens")
		return
	}
	if cap(z.WhitelistedTokens) >= int(zb0007) {
		z.WhitelistedTokens = (z.WhitelistedTokens)[:zb0007]
	} else {
		z.WhitelistedTokens = make([]string, zb0007)
	}
	for za0001 := range z.WhitelistedTokens {
		z.WhitelistedTokens[za0001], bts, err = msgp.ReadStringBytes(bts)
		if err != nil {
			err = msgp.WrapError(err, "WhitelistedTokens", za0001)
			return
		}
	}
	var zb0008 uint32
	zb0008, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "PoolAmounts")
		return
	}
	if z.PoolAmounts == nil {
		z.PoolAmounts = make(map[string]*big.Int, zb0008)
	} else if len(z.PoolAmounts) > 0 {
		for key := range z.PoolAmounts {
			delete(z.PoolAmounts, key)
		}
	}
	var field []byte
	_ = field
	for zb0008 > 0 {
		var za0002 string
		var za0003 *big.Int
		zb0008--
		za0002, bts, err = msgp.ReadStringBytes(bts)
		if err != nil {
			err = msgp.WrapError(err, "PoolAmounts")
			return
		}
		if msgp.IsNil(bts) {
			bts, err = msgp.ReadNilBytes(bts)
			if err != nil {
				return
			}
			za0003 = nil
		} else {
			{
				var zb0009 []byte
				zb0009, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(za0003))
				if err != nil {
					err = msgp.WrapError(err, "PoolAmounts", za0002)
					return
				}
				za0003 = msgpencode.DecodeInt(zb0009)
			}
		}
		z.PoolAmounts[za0002] = za0003
	}
	var zb0010 uint32
	zb0010, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "BufferAmounts")
		return
	}
	if z.BufferAmounts == nil {
		z.BufferAmounts = make(map[string]*big.Int, zb0010)
	} else if len(z.BufferAmounts) > 0 {
		for key := range z.BufferAmounts {
			delete(z.BufferAmounts, key)
		}
	}
	for zb0010 > 0 {
		var za0004 string
		var za0005 *big.Int
		zb0010--
		za0004, bts, err = msgp.ReadStringBytes(bts)
		if err != nil {
			err = msgp.WrapError(err, "BufferAmounts")
			return
		}
		if msgp.IsNil(bts) {
			bts, err = msgp.ReadNilBytes(bts)
			if err != nil {
				return
			}
			za0005 = nil
		} else {
			{
				var zb0011 []byte
				zb0011, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(za0005))
				if err != nil {
					err = msgp.WrapError(err, "BufferAmounts", za0004)
					return
				}
				za0005 = msgpencode.DecodeInt(zb0011)
			}
		}
		z.BufferAmounts[za0004] = za0005
	}
	var zb0012 uint32
	zb0012, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "ReservedAmounts")
		return
	}
	if z.ReservedAmounts == nil {
		z.ReservedAmounts = make(map[string]*big.Int, zb0012)
	} else if len(z.ReservedAmounts) > 0 {
		for key := range z.ReservedAmounts {
			delete(z.ReservedAmounts, key)
		}
	}
	for zb0012 > 0 {
		var za0006 string
		var za0007 *big.Int
		zb0012--
		za0006, bts, err = msgp.ReadStringBytes(bts)
		if err != nil {
			err = msgp.WrapError(err, "ReservedAmounts")
			return
		}
		if msgp.IsNil(bts) {
			bts, err = msgp.ReadNilBytes(bts)
			if err != nil {
				return
			}
			za0007 = nil
		} else {
			{
				var zb0013 []byte
				zb0013, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(za0007))
				if err != nil {
					err = msgp.WrapError(err, "ReservedAmounts", za0006)
					return
				}
				za0007 = msgpencode.DecodeInt(zb0013)
			}
		}
		z.ReservedAmounts[za0006] = za0007
	}
	var zb0014 uint32
	zb0014, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "TokenDecimals")
		return
	}
	if z.TokenDecimals == nil {
		z.TokenDecimals = make(map[string]*big.Int, zb0014)
	} else if len(z.TokenDecimals) > 0 {
		for key := range z.TokenDecimals {
			delete(z.TokenDecimals, key)
		}
	}
	for zb0014 > 0 {
		var za0008 string
		var za0009 *big.Int
		zb0014--
		za0008, bts, err = msgp.ReadStringBytes(bts)
		if err != nil {
			err = msgp.WrapError(err, "TokenDecimals")
			return
		}
		if msgp.IsNil(bts) {
			bts, err = msgp.ReadNilBytes(bts)
			if err != nil {
				return
			}
			za0009 = nil
		} else {
			{
				var zb0015 []byte
				zb0015, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(za0009))
				if err != nil {
					err = msgp.WrapError(err, "TokenDecimals", za0008)
					return
				}
				za0009 = msgpencode.DecodeInt(zb0015)
			}
		}
		z.TokenDecimals[za0008] = za0009
	}
	var zb0016 uint32
	zb0016, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "StableTokens")
		return
	}
	if z.StableTokens == nil {
		z.StableTokens = make(map[string]bool, zb0016)
	} else if len(z.StableTokens) > 0 {
		for key := range z.StableTokens {
			delete(z.StableTokens, key)
		}
	}
	for zb0016 > 0 {
		var za0010 string
		var za0011 bool
		zb0016--
		za0010, bts, err = msgp.ReadStringBytes(bts)
		if err != nil {
			err = msgp.WrapError(err, "StableTokens")
			return
		}
		za0011, bts, err = msgp.ReadBoolBytes(bts)
		if err != nil {
			err = msgp.WrapError(err, "StableTokens", za0010)
			return
		}
		z.StableTokens[za0010] = za0011
	}
	var zb0017 uint32
	zb0017, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "USDGAmounts")
		return
	}
	if z.USDGAmounts == nil {
		z.USDGAmounts = make(map[string]*big.Int, zb0017)
	} else if len(z.USDGAmounts) > 0 {
		for key := range z.USDGAmounts {
			delete(z.USDGAmounts, key)
		}
	}
	for zb0017 > 0 {
		var za0012 string
		var za0013 *big.Int
		zb0017--
		za0012, bts, err = msgp.ReadStringBytes(bts)
		if err != nil {
			err = msgp.WrapError(err, "USDGAmounts")
			return
		}
		if msgp.IsNil(bts) {
			bts, err = msgp.ReadNilBytes(bts)
			if err != nil {
				return
			}
			za0013 = nil
		} else {
			{
				var zb0018 []byte
				zb0018, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(za0013))
				if err != nil {
					err = msgp.WrapError(err, "USDGAmounts", za0012)
					return
				}
				za0013 = msgpencode.DecodeInt(zb0018)
			}
		}
		z.USDGAmounts[za0012] = za0013
	}
	var zb0019 uint32
	zb0019, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "MaxUSDGAmounts")
		return
	}
	if z.MaxUSDGAmounts == nil {
		z.MaxUSDGAmounts = make(map[string]*big.Int, zb0019)
	} else if len(z.MaxUSDGAmounts) > 0 {
		for key := range z.MaxUSDGAmounts {
			delete(z.MaxUSDGAmounts, key)
		}
	}
	for zb0019 > 0 {
		var za0014 string
		var za0015 *big.Int
		zb0019--
		za0014, bts, err = msgp.ReadStringBytes(bts)
		if err != nil {
			err = msgp.WrapError(err, "MaxUSDGAmounts")
			return
		}
		if msgp.IsNil(bts) {
			bts, err = msgp.ReadNilBytes(bts)
			if err != nil {
				return
			}
			za0015 = nil
		} else {
			{
				var zb0020 []byte
				zb0020, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(za0015))
				if err != nil {
					err = msgp.WrapError(err, "MaxUSDGAmounts", za0014)
					return
				}
				za0015 = msgpencode.DecodeInt(zb0020)
			}
		}
		z.MaxUSDGAmounts[za0014] = za0015
	}
	var zb0021 uint32
	zb0021, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "TokenWeights")
		return
	}
	if z.TokenWeights == nil {
		z.TokenWeights = make(map[string]*big.Int, zb0021)
	} else if len(z.TokenWeights) > 0 {
		for key := range z.TokenWeights {
			delete(z.TokenWeights, key)
		}
	}
	for zb0021 > 0 {
		var za0016 string
		var za0017 *big.Int
		zb0021--
		za0016, bts, err = msgp.ReadStringBytes(bts)
		if err != nil {
			err = msgp.WrapError(err, "TokenWeights")
			return
		}
		if msgp.IsNil(bts) {
			bts, err = msgp.ReadNilBytes(bts)
			if err != nil {
				return
			}
			za0017 = nil
		} else {
			{
				var zb0022 []byte
				zb0022, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(za0017))
				if err != nil {
					err = msgp.WrapError(err, "TokenWeights", za0016)
					return
				}
				za0017 = msgpencode.DecodeInt(zb0022)
			}
		}
		z.TokenWeights[za0016] = za0017
	}
	{
		var zb0023 []byte
		zb0023, bts, err = msgp.ReadBytesBytes(bts, (common.Address).Bytes(z.PriceFeedAddress))
		if err != nil {
			err = msgp.WrapError(err, "PriceFeedAddress")
			return
		}
		z.PriceFeedAddress = common.BytesToAddress(zb0023)
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.PriceFeed = nil
	} else {
		if z.PriceFeed == nil {
			z.PriceFeed = new(VaultPriceFeed)
		}
		bts, err = z.PriceFeed.UnmarshalMsg(bts)
		if err != nil {
			err = msgp.WrapError(err, "PriceFeed")
			return
		}
	}
	{
		var zb0024 []byte
		zb0024, bts, err = msgp.ReadBytesBytes(bts, (common.Address).Bytes(z.USDGAddress))
		if err != nil {
			err = msgp.WrapError(err, "USDGAddress")
			return
		}
		z.USDGAddress = common.BytesToAddress(zb0024)
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.USDG = nil
	} else {
		if z.USDG == nil {
			z.USDG = new(USDG)
		}
		bts, err = z.USDG.UnmarshalMsg(bts)
		if err != nil {
			err = msgp.WrapError(err, "USDG")
			return
		}
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.WhitelistedTokensCount = nil
	} else {
		{
			var zb0025 []byte
			zb0025, bts, err = msgp.ReadBytesBytes(bts, msgpencode.EncodeInt(z.WhitelistedTokensCount))
			if err != nil {
				err = msgp.WrapError(err, "WhitelistedTokensCount")
				return
			}
			z.WhitelistedTokensCount = msgpencode.DecodeInt(zb0025)
		}
	}
	z.UseSwapPricing, bts, err = msgp.ReadBoolBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "UseSwapPricing")
		return
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *Vault) Msgsize() (s int) {
	s = 3 + msgp.BoolSize + msgp.BoolSize + msgp.BoolSize
	if z.StableSwapFeeBasisPoints == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(z.StableSwapFeeBasisPoints))
	}
	if z.StableTaxBasisPoints == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(z.StableTaxBasisPoints))
	}
	if z.SwapFeeBasisPoints == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(z.SwapFeeBasisPoints))
	}
	if z.TaxBasisPoints == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(z.TaxBasisPoints))
	}
	if z.TotalTokenWeights == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(z.TotalTokenWeights))
	}
	s += msgp.ArrayHeaderSize
	for za0001 := range z.WhitelistedTokens {
		s += msgp.StringPrefixSize + len(z.WhitelistedTokens[za0001])
	}
	s += msgp.MapHeaderSize
	if z.PoolAmounts != nil {
		for za0002, za0003 := range z.PoolAmounts {
			_ = za0003
			s += msgp.StringPrefixSize + len(za0002)
			if za0003 == nil {
				s += msgp.NilSize
			} else {
				s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(za0003))
			}
		}
	}
	s += msgp.MapHeaderSize
	if z.BufferAmounts != nil {
		for za0004, za0005 := range z.BufferAmounts {
			_ = za0005
			s += msgp.StringPrefixSize + len(za0004)
			if za0005 == nil {
				s += msgp.NilSize
			} else {
				s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(za0005))
			}
		}
	}
	s += msgp.MapHeaderSize
	if z.ReservedAmounts != nil {
		for za0006, za0007 := range z.ReservedAmounts {
			_ = za0007
			s += msgp.StringPrefixSize + len(za0006)
			if za0007 == nil {
				s += msgp.NilSize
			} else {
				s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(za0007))
			}
		}
	}
	s += msgp.MapHeaderSize
	if z.TokenDecimals != nil {
		for za0008, za0009 := range z.TokenDecimals {
			_ = za0009
			s += msgp.StringPrefixSize + len(za0008)
			if za0009 == nil {
				s += msgp.NilSize
			} else {
				s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(za0009))
			}
		}
	}
	s += msgp.MapHeaderSize
	if z.StableTokens != nil {
		for za0010, za0011 := range z.StableTokens {
			_ = za0011
			s += msgp.StringPrefixSize + len(za0010) + msgp.BoolSize
		}
	}
	s += msgp.MapHeaderSize
	if z.USDGAmounts != nil {
		for za0012, za0013 := range z.USDGAmounts {
			_ = za0013
			s += msgp.StringPrefixSize + len(za0012)
			if za0013 == nil {
				s += msgp.NilSize
			} else {
				s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(za0013))
			}
		}
	}
	s += msgp.MapHeaderSize
	if z.MaxUSDGAmounts != nil {
		for za0014, za0015 := range z.MaxUSDGAmounts {
			_ = za0015
			s += msgp.StringPrefixSize + len(za0014)
			if za0015 == nil {
				s += msgp.NilSize
			} else {
				s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(za0015))
			}
		}
	}
	s += msgp.MapHeaderSize
	if z.TokenWeights != nil {
		for za0016, za0017 := range z.TokenWeights {
			_ = za0017
			s += msgp.StringPrefixSize + len(za0016)
			if za0017 == nil {
				s += msgp.NilSize
			} else {
				s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(za0017))
			}
		}
	}
	s += msgp.BytesPrefixSize + len((common.Address).Bytes(z.PriceFeedAddress))
	if z.PriceFeed == nil {
		s += msgp.NilSize
	} else {
		s += z.PriceFeed.Msgsize()
	}
	s += msgp.BytesPrefixSize + len((common.Address).Bytes(z.USDGAddress))
	if z.USDG == nil {
		s += msgp.NilSize
	} else {
		s += z.USDG.Msgsize()
	}
	if z.WhitelistedTokensCount == nil {
		s += msgp.NilSize
	} else {
		s += msgp.BytesPrefixSize + len(msgpencode.EncodeInt(z.WhitelistedTokensCount))
	}
	s += msgp.BoolSize
	return
}
