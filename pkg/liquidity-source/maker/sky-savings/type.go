package skysavings

import (
	"github.com/holiman/uint256"
)

type Extra struct {
	BlockTimestamp *uint256.Int `json:"blockTimestamp"`
	RHO            *uint256.Int `json:"rho"`
	CHI            *uint256.Int `json:"chi"`
	SavingsRate    *uint256.Int `json:"savingsRate"`
}

type StaticExtra struct {
	Pot string `json:"pot"`
	// "dsr" (DAI Savings Rate) or "ssr" (Sky Savings Rate)
	SavingsRateSymbol string `json:"savingsRateSymbol"`
}
