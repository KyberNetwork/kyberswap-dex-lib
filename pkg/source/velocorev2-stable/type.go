package velocorev2stable

import "math/big"

type Metadata struct {
	Offset int `json:"offset"`
}

type bytes32 [32]byte

type Extra struct {
	Fee1e18         *big.Int            `json:"fee1e18"`
	LpTokenBalances map[string]*big.Int `json:"lpTokenBalances"`
}

type tokenInfo struct {
	Scale uint8 `json:"scale"`
}
