package kuruobv2

import (
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type HTTPConfig struct {
	BaseURL    string                `mapstructure:"base_url" json:"base_url,omitempty"`
	Timeout    durationjson.Duration `mapstructure:"timeout" json:"timeout,omitempty"`
	RetryCount int                   `mapstructure:"retry_count" json:"retry_count,omitempty"`
}

type Config struct {
	DexID       string              `mapstructure:"dexID" json:"dexID,omitempty"`
	ChainId     valueobject.ChainID `mapstructure:"chain_id" json:"chain_id,omitempty"`
	AccountCore string              `mapstructure:"account_core" json:"account_core,omitempty"`
	HTTPConfig  HTTPConfig          `mapstructure:"http_config" json:"http_config,omitempty"`
}

type MarketInfo struct {
	MarketAddress string    `json:"marketAddress"`
	BaseToken     TokenInfo `json:"baseToken"`
	QuoteToken    TokenInfo `json:"quoteToken"`
}

type TokenInfo struct {
	TokenAddress string `json:"tokenAddress"`
}

type Metadata struct {
	LastCount         int            `json:"count"`
	LastPoolsChecksum common.Address `json:"poolsChecksum"`
}

type MarketParamsRPC struct {
	PricePrecision     uint32
	SizePrecision      *big.Int
	BaseSizeMultiplier *big.Int
	QuoteTokenDecimals uint8
}

type L2Book struct {
	BidPrices []uint32   `abi:"bidPrices"`
	BidSizes  []*big.Int `abi:"bidSizes"`
	AskPrices []uint32   `abi:"askPrices"`
	AskSizes  []*big.Int `abi:"askSizes"`
}

// StaticExtra holds immutable OrderBook params. Book quantity q is q*BaseSizeMultiplier base atoms;
// raw price p is p/PricePrecision quote per base.
type StaticExtra struct {
	PricePrecision     uint64       `json:"p"`
	SizePrecision      *uint256.Int `json:"s"`
	BaseSizeMultiplier *uint256.Int `json:"m"`
	QuoteDecimals      uint8        `json:"q"`
	HasNative          bool         `json:"n,omitempty"`
}

// Level is one aggregated L2 price level: raw price and book quantity.
type Level struct {
	Price uint64      `json:"p"`
	Size  uint256.Int `json:"s"`
}

// Extra is empty (no levels) when the market or AccountCore is paused.
type Extra struct {
	Bids        []Level `json:"b,omitempty"`
	Asks        []Level `json:"a,omitempty"`
	TakerFeePps uint64  `json:"f,omitempty"`
}

// SwapInfo tells UpdateBalance how far the swap walked the book on the consumed side.
type SwapInfo struct {
	FullLevels  int         `json:"-"`
	PartialFill uint256.Int `json:"-"`
}

// MetaInfo feeds the encoder: IdxIn 1 = quote in = OrderBook.swap isBuy.
type MetaInfo struct {
	IdxIn       int    `json:"i,omitempty"`
	HasNative   bool   `json:"n,omitempty"`
	BlockNumber uint64 `json:"blockNumber"`
}
