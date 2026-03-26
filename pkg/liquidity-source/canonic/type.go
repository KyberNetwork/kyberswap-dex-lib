package canonic

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Extra struct {
	MidPrice       string   `json:"mp"`
	MidPrecision   string   `json:"mpr"`
	OracleUpdAt    uint64   `json:"oua"`
	TakerFee       uint32   `json:"tf"`
	FeeDenom       string   `json:"fd"`
	MinQuoteTaker  string   `json:"mq"`
	MarketState    uint8    `json:"ms"`
	StateExpiresAt uint64   `json:"se"`
	RungDenom      string   `json:"rd"`
	PriceSigfigs   string   `json:"psf"`
	AskRungs       []uint16 `json:"ar"`
	AskVolumes     []string `json:"av"`
	BidRungs       []uint16 `json:"br"`
	BidVolumes     []string `json:"bv"`
}

type StaticExtra struct {
	Previewer     string `json:"prev"`
	BaseToken     string `json:"bt"`
	QuoteToken    string `json:"qt"`
	BaseDecimals  uint8  `json:"bd"`
	QuoteDecimals uint8  `json:"qd"`
	BaseScale     string `json:"bs"`
	QuoteScale    string `json:"qs"`
}

type PoolMeta struct {
	BlockNumber uint64 `json:"bN"`
	IsBuyBase   bool   `json:"isBuyBase,omitempty"`
}

type Config struct {
	DexId   string              `json:"dexId"`
	ChainId valueobject.ChainID `json:"chainId"`
	Pools   []string            `json:"pools"`
}
