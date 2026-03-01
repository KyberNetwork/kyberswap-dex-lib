package wcm

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/ethereum/go-ethereum/common"
)

const (
	priceScaleBits = 5
	priceScaleMask = (1 << priceScaleBits) - 1
)

func decodePrice(packed *big.Int) *big.Int {
	if packed == nil || packed.Sign() < 0 {
		return Zero
	}
	u := packed.Uint64()
	scale := u & priceScaleMask
	mantissa := u >> priceScaleBits
	if scale > 18 {
		return Zero
	}
	out := new(big.Int).SetUint64(mantissa)
	exp := 18 - int(scale)
	if exp > 0 {
		out.Mul(out, bignumber.TenPowInt(exp))
	} else if exp < 0 {
		out.Div(out, bignumber.TenPowInt(-exp))
	}
	return out
}

func decodeDepthChartBucket(packed *big.Int) (
	price1, qty1, price2, qty2 *big.Int,
) {
	if packed == nil {
		return Zero, Zero, Zero, Zero
	}

	qty1 = new(big.Int).Rsh(packed, uint(DepthQty1Shift))
	qty1.And(qty1, Mask64)
	price1Packed := new(big.Int).Rsh(packed, uint(DepthPrice1Shift))
	price1Packed.And(price1Packed, Mask64)
	price1 = decodePrice(price1Packed)

	qty2 = new(big.Int).Rsh(packed, uint(DepthQty2Shift))
	qty2.And(qty2, Mask64)
	price2 = decodePrice(new(big.Int).And(new(big.Int).Set(packed), Mask64))

	return price1, qty1, price2, qty2
}

func decodeBestBidOffer(packed *big.Int) (sellPrice, sellQty, buyPrice, buyQty *big.Int) {
	if packed == nil {
		return Zero, Zero, Zero, Zero
	}
	sellPrice = decodePrice(new(big.Int).And(packed, Mask64))
	sellQty = new(big.Int).Rsh(packed, uint(DepthQty2Shift))
	sellQty.And(sellQty, Mask64)
	buyPricePacked := new(big.Int).Rsh(packed, uint(DepthPrice1Shift))
	buyPricePacked.And(buyPricePacked, Mask64)
	buyPrice = decodePrice(buyPricePacked)
	buyQty = new(big.Int).Rsh(packed, uint(DepthQty1Shift))
	buyQty.And(buyQty, Mask64)
	return sellPrice, sellQty, buyPrice, buyQty
}

func UnpackVaultTokenConfig(packed *big.Int) (
	tokenId uint32,
	positionDecimals uint8,
	erc20Decimals uint8,
	tokenAddress string,
) {
	if packed == nil || packed.Sign() < 0 {
		return 0, 0, 0, ""
	}
	tokenIdBig := new(big.Int).Rsh(packed, uint(VaultTokenIdShift))
	tokenIdBig.And(tokenIdBig, Mask32)
	tokenId = uint32(tokenIdBig.Uint64())

	posDecBig := new(big.Int).Rsh(packed, uint(VaultPositionDecimalsShift))
	posDecBig.And(posDecBig, Mask8)
	positionDecimals = uint8(posDecBig.Uint64())

	erc20DecBig := new(big.Int).Rsh(packed, uint(VaultErc20DecimalsShift))
	erc20DecBig.And(erc20DecBig, Mask8)
	erc20Decimals = uint8(erc20DecBig.Uint64())

	addrBig := new(big.Int).And(packed, Mask160)
	addrBytes := make([]byte, 20)
	addrBig.FillBytes(addrBytes)
	tokenAddress = common.BytesToAddress(addrBytes).Hex()
	return tokenId, positionDecimals, erc20Decimals, tokenAddress
}

func UnpackOrderBookConfig(packed *big.Int) (takerFeeRaw, fromMaxFee, toMaxFee *big.Int) {
	if packed == nil || packed.Sign() < 0 {
		return nil, nil, nil
	}

	takerFeeRaw = new(big.Int).Rsh(packed, uint(ConfigTakerFeeShift))
	takerFeeRaw.And(takerFeeRaw, Mask16)

	fromMaxFee = new(big.Int).Rsh(packed, uint(ConfigFromMaxFeeShift))
	fromMaxFee.And(fromMaxFee, Mask64)

	toMaxFee = new(big.Int).Rsh(packed, uint(ConfigToMaxFeeShift))
	toMaxFee.And(toMaxFee, Mask64)

	return takerFeeRaw, fromMaxFee, toMaxFee
}

func scaleAmountDecimals(amount *big.Int, fromDecimals, toDecimals uint8) *big.Int {
	return scaleAmountDecimalsRounding(amount, fromDecimals, toDecimals, false)
}

func scaleAmountDecimalsRounding(amount *big.Int, fromDecimals, toDecimals uint8, roundUp bool) *big.Int {
	if amount == nil {
		return Zero
	}
	out := new(big.Int).Set(amount)
	if toDecimals == fromDecimals {
		return out
	}
	if toDecimals > fromDecimals {
		diff := int(toDecimals - fromDecimals)
		out.Mul(out, bignumber.TenPowInt(diff))
		return out
	}
	diff := int(fromDecimals - toDecimals)
	divisor := bignumber.TenPowInt(diff)

	if roundUp {
		out.Add(out, new(big.Int).Sub(divisor, One))
	}
	out.Div(out, divisor)
	return out
}
