package savingsdai

import "github.com/holiman/uint256"

type Extra struct {
	BlockTimestamp *uint256.Int `json:"blockTimestamp"`
	DSR            *uint256.Int `json:"dsr"`
	RHO            *uint256.Int `json:"rho"`
	CHI            *uint256.Int `json:"chi"`
}
