package velocorev2stable

import "math/big"

type Metadata struct {
	Offset int `json:"offset"`
}

type bytes32 [32]byte

type Extra struct {
	LpTokenBalances []*big.Int `json:"lpTokenBalances"`
}
