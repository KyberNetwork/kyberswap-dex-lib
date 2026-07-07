package kuruob

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type MarketInfo struct {
	MarketAddress string    `json:"marketaddress"`
	BaseToken     TokenInfo `json:"baseToken"`
	QuoteToken    TokenInfo `json:"quoteToken"`
}

type TokenInfo struct {
	Address string `json:"address"`
	Decimal uint8  `json:"decimal"`
	Ticker  string `json:"ticker"`
}

type MarketParamsRPC struct {
	PricePrecision     uint32
	SizePrecision      *big.Int
	BaseAssetAddress   common.Address
	BaseAssetDecimals  *big.Int
	QuoteAssetAddress  common.Address
	QuoteAssetDecimals *big.Int
	TickSize           uint32
	MinSize            *big.Int
	MaxSize            *big.Int
	TakerFeeBps        *big.Int
	MakerFeeBps        *big.Int
}

type Metadata struct {
	LastCount         int            `json:"count"`
	LastPoolsChecksum common.Address `json:"poolsChecksum"`
}

type VaultParamsRPC struct {
	KuruAmmVault           common.Address
	VaultBestBid           *big.Int
	BidPartiallyFilledSize *big.Int
	VaultBestAsk           *big.Int
	AskPartiallyFilledSize *big.Int
	VaultBidOrderSize      *big.Int
	VaultAskOrderSize      *big.Int
	Spread                 *big.Int
}

type StaticExtra struct {
	PricePrecision int  `json:"p,omitempty"`
	SizePrecision  int  `json:"s,omitempty"`
	HasNative      bool `json:"n,omitempty"`
}

type MetaInfo struct {
	Decimals  uint8 `json:"d,omitempty"`
	Precision int   `json:"p,omitempty"`
	IdxIn     int   `json:"i,omitempty"`
	HasNative bool  `json:"n,omitempty"`
}
