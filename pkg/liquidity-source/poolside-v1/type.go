package poolsidev1

import "math/big"

type Gas struct {
	Swap int64
}

type Extra struct {
	Fee          uint64 `json:"fee"`
	FeePrecision uint64 `json:"feePrecision"`
}

type PoolMeta struct {
	Fee          uint64 `json:"fee"`
	FeePrecision uint64 `json:"feePrecision"`
	BlockNumber  uint64 `json:"blockNumber"`
}

type PoolsListUpdaterMetadata struct {
	Offset int `json:"offset"`
}

type GetReservesResult struct {
	Pool0              *big.Int
	Pool1              *big.Int
	BlockTimestampLast uint32
}

type ReserveData struct {
	Pool0 *big.Int
	Pool1 *big.Int
}
