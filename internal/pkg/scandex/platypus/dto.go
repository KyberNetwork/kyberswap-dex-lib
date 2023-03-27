package platypus

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type GetPoolsQueryParam struct {
	First int
	Skip  int
}

type GetPoolsResponse struct {
	Pools []struct {
		Id string `json:"id"`
	} `json:"pools"`
}

// PoolState represents data of pool smart contract
type PoolState struct {
	Address        string
	C1             *big.Int
	HaircutRate    *big.Int
	PriceOracle    common.Address
	RetentionRatio *big.Int
	SlippageParamK *big.Int
	SlippageParamN *big.Int
	TokenAddresses []common.Address
	XThreshold     *big.Int
	Paused         bool
}

// AssetState represents data of asset smart contract
type AssetState struct {
	Address          string
	Decimals         uint8
	Cash             *big.Int
	Liability        *big.Int
	UnderlyingToken  common.Address
	AggregateAccount common.Address
}
