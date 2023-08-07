package traderjoev21

import "math/big"

// Reserves https://github.com/traderjoe-xyz/joe-v2/blob/v2.1.1/src/LBPair.sol#L160
type Reserves struct {
	ReserveX *big.Int
	ReserveY *big.Int
}

type Metadata struct {
	Offset int `json:"offset"`
}
