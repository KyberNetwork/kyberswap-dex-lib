package litepsm

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type DexConfig struct {
	Dai  Token       `json:"dai"`
	PSMs []PSMConfig `json:"psms"`
}

type PSMConfig struct {
	Address string `json:"address"`
	Gem     Token  `json:"gem"`
}

type Token struct {
	Address  string `json:"address"`
	Decimals uint8  `json:"decimals"`
}

type LitePSM struct {
	TIn  *uint256.Int `json:"tIn"`
	TOut *uint256.Int `json:"tOut"`

	To18ConversionFactor *uint256.Int `json:"to18ConversionFactor,omitempty"`
	DaiBalance           *uint256.Int `json:"daiBalance,omitempty"`
	GemBalance           *uint256.Int `json:"gemBalance,omitempty"`
}

type StaticExtra struct {
	Gem    Token          `json:"gem"`
	Pocket common.Address `json:"pocket"` // The ultimate holder of the gems
}

type Extra struct {
	LitePSM LitePSM `json:"litePSM"`
}

type Gas struct {
	BuyGem  int64
	SellGem int64
}
