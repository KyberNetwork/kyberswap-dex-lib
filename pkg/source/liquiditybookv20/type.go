package liquiditybookv20

import "math/big"

type Metadata struct {
	Offset int `json:"offset"`
}

type Extra struct {
	RpcBlockTimestamp      uint64        `json:"rpcBlockTimestamp"`
	SubgraphBlockTimestamp uint64        `json:"subgraphBlockTimestamp,omitempty"`
	FeeParameters          feeParameters `json:"feeParameters"`
	ActiveBinID            uint32        `json:"activeBinId"`
	Bins                   []Bin         `json:"bins"`
	Liquidity              *big.Int      `json:"liquidity"`
	PriceX128              *big.Int      `json:"priceX128"`
}

type QueryRpcPoolStateResult struct {
	BlockTimestamp uint64        `json:"blockTimestamp"`
	FeeParameters  feeParameters `json:"feeParameters"`
	ReservesAndID  reservesAndID `json:"reserves"`
	Liquidity      *big.Int      `json:"liquidity"`
	PriceX128      *big.Int      `json:"priceX128"`
}

type reservesAndID struct {
	ReserveX *big.Int `json:"reserveX"`
	ReserveY *big.Int `json:"reserveY"`
	ActiveId *big.Int `json:"activeId"`
}

type querySubgraphPoolStateResult struct {
	BlockTimestamp uint64 `json:"blockTimestamp"`
	Bins           []Bin  `json:"bins"`
}

type getSwapOutResult struct {
	AmountOut          *big.Int
	Fee                *big.Int
	BinsReserveChanges []binReserveChanges
	FeeParameters      feeParameters
	NewActiveID        uint32
}

type SwapInfo struct {
	BinsReserveChanges []binReserveChanges `json:"-"`
	NewFeeParameters   feeParameters       `json:"-"`
	NewActiveID        uint32              `json:"-"`
}

// rpc

type feeParametersRpcResp struct {
	State feeParametersRpc
}

type feeParametersRpc struct {
	BinStep                  uint16
	BaseFactor               uint16
	FilterPeriod             uint16
	DecayPeriod              uint16
	ReductionFactor          uint16
	VariableFeeControl       *big.Int
	ProtocolShare            uint16
	MaxVolatilityAccumulated *big.Int
	VolatilityAccumulated    *big.Int
	VolatilityReference      *big.Int
	IndexRef                 *big.Int
	Time                     *big.Int
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
