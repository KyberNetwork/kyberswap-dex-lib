package traderjoev20

import "math/big"

// Reserves https://github.com/traderjoe-xyz/joe-v2/blob/v2.0.0/src/LBPair.sol#L151
type Reserves struct {
	ReserveX *big.Int
	ReserveY *big.Int
	//revive:disable:var-naming
	ActiveId *big.Int
}

type Metadata struct {
	Offset int `json:"offset"`
}
