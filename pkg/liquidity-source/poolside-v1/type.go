package poolsidev1

import "math/big"

type Gas struct {
	Swap int64
}

type RebaseTokenInfo struct {
	UnderlyingToken string     `json:"underlyingToken"`
	WrapRatio       *big.Float `json:"wrapRatio"`
	UnwrapRatio     *big.Float `json:"unwrapRatio"`
	Decimals        uint8      `json:"decimals"`
}

type Extra struct {
	Fee                uint64                     `json:"fee"`
	FeePrecision       uint64                     `json:"feePrecision"`
	RebaseTokenInfoMap map[string]RebaseTokenInfo `json:"rebaseTokenInfoMap"`
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
