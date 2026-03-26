package canonic

import "github.com/holiman/uint256"

type Extra struct {
	MidPrice   *uint256.Int   `json:"mp"`
	MidPrec    *uint256.Int   `json:"mprec"`
	TakerFee   *uint256.Int   `json:"tf"`
	BaseScale  *uint256.Int   `json:"bs"`
	QuoteScale *uint256.Int   `json:"qs"`
	AskBps     []uint16       `json:"abps"`
	AskVols    []*uint256.Int `json:"avol"`
	BidBps     []uint16       `json:"bbps"`
	BidVols    []*uint256.Int `json:"bvol"`
	Active     bool           `json:"act"`
}

type StaticExtra struct {
	BaseToken  string `json:"bt"`
	QuoteToken string `json:"qt"`
}

type PoolMeta struct {
	BlockNumber uint64 `json:"bN"`
	MAOB        string `json:"maob"`
	IsSellBase  bool   `json:"isSell,omitempty"`
}
