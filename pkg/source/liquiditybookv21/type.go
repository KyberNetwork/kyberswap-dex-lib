package liquiditybookv21

import "math/big"

type Metadata struct {
	Offset int `json:"offset"`
}

type Extra struct {
	RpcBlockTimestamp      uint64            `json:"rpcBlockTimestamp"`
	SubgraphBlockTimestamp uint64            `json:"subgraphBlockTimestamp"`
	StaticFeeParams        staticFeeParams   `json:"staticFeeParams"`
	VariableFeeParams      variableFeeParams `json:"variableFeeParams"`
	ActiveBinID            uint32            `json:"activeBinId"`
	BinStep                uint16            `json:"binStep"`
	Bins                   []bin             `json:"bins"`
}

type SwapInfo struct {
	AmountsInLeft      *big.Int            `json:"amountsInLeft"`
	NewParameters      *parameters         `json:"parameters"`
	NewActiveID        uint32              `json:"newActiveId"`
	BinsReserveChanges []binReserveChanges `json:"binsReserveChanges"`
}

type queryRpcPoolStateResult struct {
	BlockTimestamp    uint64            `json:"blockTimestamp"`
	StaticFeeParams   staticFeeParams   `json:"staticFeeParams"`
	VariableFeeParams variableFeeParams `json:"variableFeeParams"`
	Reserves          reserves          `json:"reserves"`
	ActiveBinID       uint32            `json:"activeBinId"`
	BinStep           uint16            `json:"binStep"`
}

type querySubgraphPoolStateResult struct {
	BlockTimestamp uint64 `json:"blockTimestamp"`
	Bins           []bin  `json:"bins"`
}

type staticFeeParams struct {
	BaseFactor               uint16 `json:"baseFactor"`
	FilterPeriod             uint16 `json:"filterPeriod"`
	DecayPeriod              uint16 `json:"decayPeriod"`
	ReductionFactor          uint16 `json:"reductionFactor"`
	VariableFeeControl       uint32 `json:"variableFeeControl"`
	ProtocolShare            uint16 `json:"protocolShare"`
	MaxVolatilityAccumulator uint32 `json:"maxVolatilityAccumulator"`
}

type variableFeeParams struct {
	VolatilityAccumulator uint32 `json:"volatilityAccumulator"`
	VolatilityReference   uint32 `json:"volatilityReference"`
	IdReference           uint32 `json:"idReference"`
	TimeOfLastUpdate      uint64 `json:"timeOfLastUpdate"`
}

type reserves struct {
	ReserveX *big.Int `json:"reserveX"`
	ReserveY *big.Int `json:"reserveY"`
}

type getSwapOutResult struct {
	AmountOut          *big.Int
	Fee                *big.Int
	BinsReserveChanges []binReserveChanges
	Parameters         *parameters
	NewActiveID        uint32
}

// subgraph

type lbpairSubgraphResp struct {
	ID     string            `json:"id"`
	TokenX tokenSubgraphResp `json:"tokenX"`
	TokenY tokenSubgraphResp `json:"tokenY"`
	Bins   []binSubgraphResp `json:"bins"`
}

type binSubgraphResp struct {
	ID          string `json:"id"`
	BinID       string `json:"binId"`
	ReserveX    string `json:"reserveX"`
	ReserveY    string `json:"reserveY"`
	TotalSupply string `json:"totalSupply"`
}

type tokenSubgraphResp struct {
	Decimals string `json:"decimals"`
}

// rpc

type staticFeeParamsResp struct {
	BaseFactor               uint16
	FilterPeriod             uint16
	DecayPeriod              uint16
	ReductionFactor          uint16
	VariableFeeControl       *big.Int
	ProtocolShare            uint16
	MaxVolatilityAccumulator *big.Int
}

type variableFeeParamsResp struct {
	VolatilityAccumulator *big.Int
	VolatilityReference   *big.Int
	IdReference           *big.Int
	TimeOfLastUpdate      *big.Int
}
