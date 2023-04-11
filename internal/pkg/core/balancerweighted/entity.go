package balancerweighted

import (
	"math/big"
)

type Meta struct {
	VaultAddress string `json:"vault"`
	PoolId       string `json:"poolId"`
}

type StaticExtra struct {
	VaultAddress  string `json:"vaultAddress"`
	PoolId        string `json:"poolId"`
	TokenDecimals []uint `json:"tokenDecimals"`
}
type Extra struct {
	AmplificationParameter AmplificationParameter `json:"amplificationParameter"`
}
type AmplificationParameter struct {
	Value      *big.Int `json:"value"`
	IsUpdating bool     `json:"isUpdating"`
	Precision  *big.Int `json:"precision"`
}

type Gas struct {
	Swap int64
}
