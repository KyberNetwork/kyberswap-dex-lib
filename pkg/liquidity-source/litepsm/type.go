package litepsm

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type PSMConfig struct {
	PoolAddress string `json:"poolAddress"`
	PsmAddress  string `json:"psmAddress"`
	DebtToken   string `json:"debtToken"`
	GemToken    string `json:"gemToken"`
	DaiToken    string `json:"daiToken"`
}

type LitePSM struct {
	TIn  *uint256.Int `json:"tIn"`
	TOut *uint256.Int `json:"tOut"`

	To18ConversionFactor *uint256.Int `json:"to18ConversionFactor,omitempty"`
	DaiBalance           *uint256.Int `json:"daiBalance,omitempty"`
	GemBalance           *uint256.Int `json:"gemBalance,omitempty"`
}

type StaticExtra struct {
	Pocket common.Address `json:"pocket"` // The ultimate holder of the gems
	Psm    common.Address `json:"psm"`
	Dai    common.Address `json:"dai"`
}

type Extra struct {
	LitePSM LitePSM `json:"litePSM"`
}

type MetaInfo struct {
	IsSellGem       bool   `json:"isSellGem"`
	ApprovalAddress string `json:"approvalAddress,omitempty"`
}

type Gas struct {
	BuyGem  int64
	SellGem int64
}
