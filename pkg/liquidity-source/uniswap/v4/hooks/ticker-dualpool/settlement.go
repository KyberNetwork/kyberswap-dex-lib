package tickerdualpool

import "github.com/holiman/uint256"

// inventoryAfterJIT handles the side held directly by the hook, without a vault.
// Old claims are redeemed to cash before minting liquidity. Only the net JIT
// delta is settled: a positive delta becomes a new claim; a negative one uses cash.
func inventoryAfterJIT(raw, claims, minted, returned *uint256.Int) (*uint256.Int, *uint256.Int, error) {
	if raw == nil || claims == nil || minted == nil || returned == nil {
		return nil, nil, ErrInvalidState
	}
	cash, err := addChecked(raw, claims)
	if err != nil {
		return nil, nil, err
	}
	if cash.BitLen() > 128 || cash.Cmp(minted) < 0 {
		return nil, nil, ErrInvalidState
	}
	nextClaims := new(uint256.Int)
	if returned.Cmp(minted) >= 0 {
		nextClaims.Sub(returned, minted)
	} else {
		cash.Sub(cash, new(uint256.Int).Sub(minted, returned))
	}
	if nextClaims.BitLen() > 128 {
		return nil, nil, ErrInvalidState
	}
	return cash, nextClaims, nil
}

// managerCashAfterJIT stops before the router pays input or takes output.
// This helper stops before the exact-input hook fee take; the caller subtracts
// those fees before recording the state visible to a subsequent swap.
func managerCashAfterJIT(cash, oldClaims, minted, returned *uint256.Int) (*uint256.Int, error) {
	if cash == nil || oldClaims == nil || minted == nil || returned == nil {
		return nil, ErrInvalidState
	}
	if cash.Cmp(oldClaims) < 0 {
		return nil, ErrVaultIlliquid
	}
	next := new(uint256.Int).Sub(cash, oldClaims)
	if minted.Cmp(returned) > 0 {
		return addChecked(next, new(uint256.Int).Sub(minted, returned))
	}
	return next, nil
}
func unbackedClaims(claims, cash *uint256.Int) *uint256.Int {
	if claims.Cmp(cash) > 0 {
		return new(uint256.Int).Sub(claims, cash)
	}
	return new(uint256.Int)
}
