package quoting

import "math/big"

type Pool interface {
	Quote(amount *big.Int, isToken1 bool) (*Quote, error)
	GetKey() *PoolKey
}
