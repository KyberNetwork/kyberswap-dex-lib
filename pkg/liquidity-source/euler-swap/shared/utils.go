package shared

import (
	"math/big"

	"github.com/holiman/uint256"

	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

var (
	sixtyThree = uint256.NewInt(63)
)

func ConvertToAssets(shares, totalAssets, totalSupply *big.Int) *big.Int {
	if totalSupply.Sign() == 0 {
		return shares
	}
	// (shares * (totalAssets + VirtualAmount)) / (totalSupply + VirtualAmount)
	return new(big.Int).Div(new(big.Int).Mul(shares, new(big.Int).Add(totalAssets, VirtualAmount)),
		new(big.Int).Add(totalSupply, VirtualAmount))
}

func SubTill0(amt, sub *uint256.Int) *uint256.Int {
	if sub == nil || sub.Sign() == 0 {
		return amt
	}
	if amt == nil || sub.Cmp(amt) >= 0 {
		return big256.U0
	}
	return new(uint256.Int).Sub(amt, sub)
}

func DecodeCap(amountCap *uint256.Int) *uint256.Int {
	if amountCap.IsZero() {
		return new(uint256.Int).Set(big256.UMax)
	}

	var powerBits, tenToPower, multiplier uint256.Int
	powerBits.And(amountCap, sixtyThree)
	tenToPower.Exp(big256.U10, &powerBits)
	multiplier.Rsh(amountCap, 6)

	amountCap.Mul(&tenToPower, &multiplier)
	return amountCap.Div(amountCap, big256.U100)
}
