package uniswaplo

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
)

const (
	DexType = "uniswap-lo"
)

type OrderStatus string

type OrderType string

type SortKey string

type SwapSide int

const (
	OpenOrderStatus OrderStatus = "open"

	DutchV2OrderType OrderType = "Dutch_V2"

	CreatedAtSortKey SortKey = "createdAt"
)

type DutchOrderQuery struct {
	Limit       uint        `url:"limit,omitempty"`
	OrderStatus OrderStatus `url:"orderStatus,omitempty"`
	OrderType   OrderType   `url:"orderType,omitempty"`
	OrderHash   string      `url:"orderHash,omitempty"`
	Swapper     string      `url:"swapper,omitempty"`
	Filler      string      `url:"filler,omitempty"`
	Cursor      string      `url:"cursor,omitempty"`
	ChainID     int         `url:"chainId"`
	SortKey     SortKey     `url:"sortKey,omitempty"`
	Sort        string      `url:"sort,omitempty"`
}

func (d *DutchOrderQuery) AddSortByCreatedAtGreaterThan(time int64) {
	d.SortKey = CreatedAtSortKey
	d.Sort = fmt.Sprintf("gt(%v)", time)
}

type DutchOrdersResponse struct {
	Orders []DutchOrder `json:"orders"`
	Cursor string       `json:"cursor"`
}

type DutchOrder struct {
	Type           string          `json:"type"`
	OrderStatus    OrderStatus     `json:"orderStatus"`
	EncodedOrder   hexutil.Bytes   `json:"encodedOrder"`
	Signature      hexutil.Bytes   `json:"signature"`
	Nonce          string          `json:"nonce"`
	OrderHash      string          `json:"orderHash"`
	ChainID        int             `json:"chainId"`
	Swapper        common.Address  `json:"swapper"`
	Reactor        string          `json:"reactor"`
	DecayStartTime int             `json:"decayStartTime"`
	DecayEndTime   int             `json:"decayEndTime"`
	Deadline       int             `json:"deadline"`
	Input          Input           `json:"input"`
	Outputs        []Output        `json:"outputs"`
	Filler         common.Address  `json:"filler"`
	QuoteID        string          `json:"quoteId"`
	TxHash         string          `json:"txHash"`
	SettledAmounts []SettledAmount `json:"settledAmounts"`
	Cosignature    string          `json:"cosignature"`
	CosignerData   CosignerData    `json:"cosignerData"`
	CreatedAt      uint64          `json:"createdAt"`
}

type Input struct {
	Token       common.Address `json:"token"`
	StartAmount *uint256.Int   `json:"startAmount"`
	EndAmount   *uint256.Int   `json:"endAmount"`
}

type Output struct {
	Token       common.Address `json:"token"`
	StartAmount *uint256.Int   `json:"startAmount"`
	EndAmount   *uint256.Int   `json:"endAmount"`
	Recipient   common.Address `json:"recipient"`
}

type SettledAmount struct {
	TokenOut  common.Address `json:"tokenOut"`
	AmountOut *uint256.Int   `json:"amountOut"`
	TokenIn   common.Address `json:"tokenIn"`
	AmountIn  *uint256.Int   `json:"amountIn"`
}

type CosignerData struct {
	DecayStartTime  *uint256.Int   `json:"decayStartTime"`
	DecayEndTime    *uint256.Int   `json:"decayEndTime"`
	ExclusiveFiller string         `json:"exclusiveFiller"`
	InputOverride   *uint256.Int   `json:"inputOverride"`
	OutputOverrides []*uint256.Int `json:"outputOverrides"`
}

type SwapInfo struct {
	AmountIn     string        `json:"amountIn"`
	SwapSide     SwapSide      `json:"swapSide"`
	FilledOrders []*DutchOrder `json:"filledOrders"`
}
