package metavault

import "github.com/holiman/uint256"

type VaultAddress struct {
	Vault string `json:"vault"`
}

type Extra struct {
	Vault *Vault `json:"vault"`
}

// SwapInfo is what CalcAmountOut hands to UpdateBalance: the two amounts swap computed from the state before the
// swap (UpdateBalance must not compute them again from the state it is updating). The fields are internal state, so
// they are not part of the swap extra in the route response.
type SwapInfo struct {
	// CreditedAmountIn is the amount in the vault credits to the swapper, see Vault.CreditedAmountIn.
	CreditedAmountIn uint256.Int `json:"-"`
	// UsdmAmount is CreditedAmountIn valued in USDM at the min price of the token in.
	UsdmAmount uint256.Int `json:"-"`
}
