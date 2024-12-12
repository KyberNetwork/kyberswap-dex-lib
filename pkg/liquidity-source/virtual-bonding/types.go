package virtualfun

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type StaticExtra struct {
	BondingAddress string `json:"bondingAddress"`
}

type Extra struct {
	KLast    *big.Int `json:"kLast"`
	BuyTax   *big.Int `json:"buyTax"`
	SellTax  *big.Int `json:"sellTax"`
	ReserveA *big.Int `json:"reserveA"`
	ReserveB *big.Int `json:"reserveB"`
}

type SwapInfo struct {
	IsBuy          bool         `json:"isBuy"`
	BondingAddress string       `json:"bondingAddress"`
	NewReserveA    *uint256.Int `json:"-"`
	NewReserveB    *uint256.Int `json:"-"`
	NewBalanceA    *uint256.Int `json:"-"`
	NewBalanceB    *uint256.Int `json:"-"`
}

type TokenInfo struct {
	Creator          common.Address `json:"creator"`
	Token            common.Address `json:"token"`
	Pair             common.Address `json:"pair"`
	AgentToken       common.Address `json:"agent_token"`
	Data             Data           `json:"data"`
	Description      string         `json:"description"`
	Cores            []uint8        `json:"cores"`
	Image            string         `json:"image"`
	Twitter          string         `json:"twitter"`
	Telegram         string         `json:"telegram"`
	Youtube          string         `json:"youtube"`
	Website          string         `json:"website"`
	Trading          bool           `json:"trading"`
	TradingOnUniswap bool           `json:"trading_on_uniswap"`
}

type Data struct {
	Token       common.Address `json:"token"`
	Name        string         `json:"name"`
	ShortName   string         `json:"short_name"`
	Ticker      string         `json:"ticker"`
	Supply      *big.Int       `json:"supply"`
	Price       *big.Int       `json:"price"`
	MarketCap   *big.Int       `json:"market_cap"`
	Liquidity   *big.Int       `json:"liquidity"`
	Volume      *big.Int       `json:"volume"`
	Volume24H   *big.Int       `json:"volume_24h"`
	PrevPrice   *big.Int       `json:"prev_price"`
	LastUpdated *big.Int       `json:"last_updated"`
}

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}
