package shared

import (
	"math/big"
	"slices"

	"github.com/holiman/uint256"

	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	sixtyThree = uint256.NewInt(63)

	// PriceScale scales every tracked price (DebtPrice, ValuePrices, VaultValuePrices): a price is
	// the unit of account per smallest unit of the asset, times PriceScale. The solvency check
	// compares values priced on one scale, so the factor cancels; it keeps the fraction that an
	// integer price per smallest unit loses (an 18-decimal asset under $1 in an 18-decimal unit
	// of account is 0 per unit).
	PriceScale = big.NewInt(1e18)
)

// UnitPrice is an oracle quote for quoteAmount of an asset as a price per smallest unit, scaled by
// PriceScale; 0 for a quote the call did not return.
func UnitPrice(quote, quoteAmount *big.Int) *big.Int {
	if quote == nil || quoteAmount == nil || quoteAmount.Sign() == 0 {
		return big.NewInt(0)
	}
	return new(big.Int).Div(new(big.Int).Mul(quote, PriceScale), quoteAmount)
}

func ConvertToAssets(shares, totalAssets, totalSupply *big.Int) *big.Int {
	if shares == nil {
		return big.NewInt(0)
	}
	// A nil total means the vault's balances were never fetched (e.g. the
	// controller-only vault v2's pool_tracker appends past the batch RPC has no
	// totalAssets/totalSupply call), and a zero supply means no shares are
	// outstanding — in both cases assets == shares 1:1. Guarding here avoids the
	// nil *big.Int deref in totalSupply.Sign()/Add that otherwise panics.
	if totalSupply == nil || totalAssets == nil || totalSupply.Sign() == 0 {
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

// WithVaultDecimals returns a copy of vaults whose QuoteAmount is one whole unit of each vault's asset,
// 10^decimals, from the decimals the vault reports (an EVK vault has its asset's decimals). The pool's
// tokens carry no decimals, so a quote amount built from them is 1 smallest unit, and the oracle prices
// 1 smallest unit of an 18-decimal asset under $1 at 0. A vault whose decimals were not returned (0)
// keeps its QuoteAmount.
func WithVaultDecimals(vaults []VaultInfo, decimals []uint8) []VaultInfo {
	out := slices.Clone(vaults)
	for i := range out {
		if i < len(decimals) && decimals[i] > 0 {
			out[i].QuoteAmount = bignumber.TenPowInt(decimals[i])
		}
	}
	return out
}
