package crystalob

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// Level is [price, orderSize...] with orders in fill (FIFO) order.
// Bid sizes are in quote, ask sizes in base.
type Level []*uint256.Int

type Extra struct {
	Bids        []Level      `json:"b,omitempty"`
	Asks        []Level      `json:"a,omitempty"`
	ReserveQ    *uint256.Int `json:"rq,omitempty"` // nil when the AMM is disabled
	ReserveB    *uint256.Int `json:"rb,omitempty"`
	TakerFee    uint64       `json:"tf"`
	MakerRebate uint64       `json:"mr"`
}

type StaticExtra struct {
	MarketType  uint64       `json:"t"`
	ScaleFactor *uint256.Int `json:"s"`
	TickSize    *uint256.Int `json:"ts"`
	MaxPrice    *uint256.Int `json:"mp"`
	Router      string       `json:"r"`
	HasNative   bool         `json:"n,omitempty"` // a token is Crystal.weth, so swap() takes native
}

// MetaInfo keeps pool.MetaInfo's approvalAddress key, which the encoder reads as the router.
type MetaInfo struct {
	ApprovalAddress string `json:"approvalAddress"`
	HasNative       bool   `json:"n,omitempty"`
	BlockNumber     uint64 `json:"blockNumber"`
}

type SwapInfo struct {
	IsBuy      bool         `json:"-"`
	LevelsDone int          `json:"-"` // fully consumed levels
	OrdersDone int          `json:"-"` // fully consumed orders in the next level
	PartialRem *uint256.Int `json:"-"` // remaining size of the partially filled order
	ReserveQ   *uint256.Int `json:"-"`
	ReserveB   *uint256.Int `json:"-"`
}

type MarketInfoRPC struct {
	QuoteAsset   common.Address
	BaseAsset    common.Address
	MarketType   *big.Int
	HighestBid   *big.Int
	LowestAsk    *big.Int
	ScaleFactor  *big.Int
	TickSize     *big.Int
	MaxPrice     *big.Int
	MinSize      *big.Int
	TakerFee     *big.Int
	MakerRebate  *big.Int
	ReserveQuote *big.Int
	ReserveBase  *big.Int
	IsAMMEnabled bool
}

type PriceLevelsRPC struct {
	HighestBid *big.Int
	LowestAsk  *big.Int
	Bids       []byte
	Asks       []byte
}

type PriceLevelRPC struct {
	Size           *big.Int
	LatestNativeId *big.Int
	Latest         *big.Int
	FillNext       *big.Int
}

type OrderRPC struct {
	IsBuy      bool
	Market     common.Address
	Price      *big.Int
	Size       *big.Int
	OrderType  *big.Int
	UserId     *big.Int
	FillBefore *big.Int
	FillAfter  *big.Int
}

type Metadata struct {
	Offset  int              `json:"offset"`
	Pending []common.Address `json:"pending,omitempty"` // launchpad markets not yet graduated
}
