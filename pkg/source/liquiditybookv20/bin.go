package liquiditybookv20

import "math/big"

type bin struct {
	ID          uint32   `json:"id"`
	ReserveX    *big.Int `json:"reserveX"`
	ReserveY    *big.Int `json:"reserveY"`
	TotalSupply *big.Int `json:"totalSupply"`
}
