package flap

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// totalSupply mirrors LibCurve.TOTAL_SUPPLY: 1,000,000,000 ether (18 decimals, WAD-scaled).
var totalSupply = new(uint256.Int).Mul(big256.TenPow(9), big256.BONE)

// Curve mirrors flap's LibCurve.Curve: a constant-product bonding curve with virtual reserves, where
//
//	(TOTAL_SUPPLY + h - s) * (reserve + r) = k
//
// r, h, k are all WAD-scaled (1e18), independent of the quote token's own decimals. r is the virtual
// reserve offset, h is the virtual supply offset (0 for the legacy "fromR" curves, where k = r *
// TOTAL_SUPPLY), k is the square of the virtual liquidity.
type Curve struct {
	R *uint256.Int `json:"r"`
	H *uint256.Int `json:"h"`
	K *uint256.Int `json:"k"`
}

// estimateSupplyV2 mirrors LibCurve.estimateSupplyV2: given a reserve amount denominated in the quote
// token's own decimals, returns the circulating supply (18 decimals) implied by the curve. Rounded
// down, matching the contract's protocol-favorable rounding for buys (s = TOTAL_SUPPLY + h -
// divWadUp(k, r + reserve)).
func estimateSupplyV2(curve Curve, reserve *uint256.Int, reserveDecimals uint8) (*uint256.Int, error) {
	scaledReserve, err := scaleTo18(reserve, reserveDecimals)
	if err != nil {
		return nil, err
	}

	denom := new(uint256.Int).Add(curve.R, scaledReserve)
	if denom.IsZero() {
		return nil, ErrDivByZero
	}

	var quotient uint256.Int
	big256.MulDivUp(&quotient, curve.K, big256.BONE, denom)

	supply := new(uint256.Int).Add(totalSupply, curve.H)
	if supply.Lt(&quotient) {
		return nil, ErrCurveUnderflow
	}
	supply.Sub(supply, &quotient)

	return supply, nil
}

// estimateReserveV2 mirrors LibCurve.estimateReserveV2: given a circulating supply (18 decimals),
// returns the reserve amount denominated in the quote token's own decimals implied by the curve.
// Rounded up, matching the contract's protocol-favorable rounding for sells (reserve = divWadUp(k, h +
// TOTAL_SUPPLY - s) - r).
func estimateReserveV2(curve Curve, supply *uint256.Int, reserveDecimals uint8) (*uint256.Int, error) {
	if supply.Gt(totalSupply) {
		return nil, ErrSupplyExceedsTotalSupply
	}

	denom := new(uint256.Int).Add(totalSupply, curve.H)
	if denom.Lt(supply) {
		return nil, ErrCurveUnderflow
	}
	denom.Sub(denom, supply)
	if denom.IsZero() {
		return nil, ErrDivByZero
	}

	var scaledReserve uint256.Int
	big256.MulDivUp(&scaledReserve, curve.K, big256.BONE, denom)
	if scaledReserve.Lt(curve.R) {
		return nil, ErrCurveUnderflow
	}
	scaledReserve.Sub(&scaledReserve, curve.R)

	return scaleFrom18Up(&scaledReserve, reserveDecimals), nil
}

// scaleTo18 mirrors the reserve-scaling half of estimateSupplyV2: upscale a reserve amount from
// reserveDecimals to 18 decimals. A no-op (same pointer) when reserveDecimals == 18.
func scaleTo18(amount *uint256.Int, reserveDecimals uint8) (*uint256.Int, error) {
	if reserveDecimals > 18 {
		return nil, ErrUnsupportedDecimals
	}
	if reserveDecimals == 18 {
		return amount, nil
	}
	scale := big256.TenPow(18 - reserveDecimals)
	return new(uint256.Int).Mul(amount, scale), nil
}

// scaleFrom18Up mirrors the reserve-scaling half of estimateReserveV2: downscale a WAD (18-decimal)
// amount to reserveDecimals, rounding up. A no-op (same pointer) when reserveDecimals == 18.
func scaleFrom18Up(amount *uint256.Int, reserveDecimals uint8) *uint256.Int {
	if reserveDecimals == 18 {
		return amount
	}
	scale := big256.TenPow(18 - reserveDecimals)
	return big256.DivUp(amount, scale)
}

// applyFeeDown returns amount with feeBps taken off, rounded down (fee rounded up in the protocol's
// favor), mirroring how a bps-based protocol fee is conventionally deducted.
func applyFeeDown(amount *uint256.Int, feeBps uint64) *uint256.Int {
	if feeBps == 0 {
		return new(uint256.Int).Set(amount)
	}
	fee := new(uint256.Int).Mul(amount, uint256.NewInt(feeBps))
	fee = big256.DivUp(fee, uint256.NewInt(bpsDenominator))
	return new(uint256.Int).Sub(amount, fee)
}

// growForFeeUp is the exact inverse of applyFeeDown: given a desired post-fee (net) amount, returns
// the smallest pre-fee (gross) amount that clears it after applyFeeDown is re-applied, rounded up.
// feeBps must be < bpsDenominator (guaranteed by the tracker/simulator construction checks).
func growForFeeUp(netAmount *uint256.Int, feeBps uint64) *uint256.Int {
	if feeBps == 0 {
		return new(uint256.Int).Set(netAmount)
	}
	num := new(uint256.Int).Mul(netAmount, uint256.NewInt(bpsDenominator))
	return big256.DivUp(num, uint256.NewInt(bpsDenominator-feeBps))
}

// floorFeeAmount returns floor(amount*feeBps/bpsDenominator), matching PortalTradeV2's sell-side fee
// computation exactly (verified bit-for-bit live: sell mismatched by 1 wei until this replaced the
// combined ceil-based deduction used for buy - sell computes the protocol-fee and token-tax *amounts*
// independently, each floored, rather than combining bps into one ceiling mulDiv like buy does).
func floorFeeAmount(amount *uint256.Int, feeBps uint64) *uint256.Int {
	if feeBps == 0 {
		return new(uint256.Int)
	}
	var fee uint256.Int
	big256.MulDivDown(&fee, amount, uint256.NewInt(feeBps), uint256.NewInt(bpsDenominator))
	return &fee
}
