package litepsm

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type PSMConfig struct {
	DaiSelector uint32 `json:"daiSelector"`
	IsMint      bool   `json:"isMint"`
}

type StaticExtra struct {
	IsMint  bool            `json:"mint,omitempty"`
	Pocket  *common.Address `json:"poc,omitempty"` // gem holder, if exist
	GemJoin *common.Address `json:"gJ,omitempty"`  // gemJoin, if different from this psm
	Dai     *common.Address `json:"dai,omitempty"` // inner psm's dai, if different from this dai
}

type Extra struct {
	TIn  *uint256.Int `json:"tIn,omitempty"`
	TOut *uint256.Int `json:"tOut,omitempty"`
}

type MetaInfo struct {
	IsBuyGem         bool   `json:"bG,omitempty"`
	TokenDecimalDiff int8   `json:"tDD,omitempty"`
	PrecisionDecimal uint8  `json:"pD,omitempty"`
	ApprovalAddress  string `json:"approvalAddress,omitempty"`
}
