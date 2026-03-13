package smoothy

import "github.com/holiman/uint256"

type TokenInfo struct {
	SoftWeight         *uint256.Int `json:"sW"`
	HardWeight         *uint256.Int `json:"hW"`
	DecimalMulitiplier uint8        `json:"dM"`
	Balance            *uint256.Int `json:"b"`
}

type Extra struct {
	SwapFee      *uint256.Int `json:"sF"`
	AdminFeePct  *uint256.Int `json:"aFP"`
	TotalBalance *uint256.Int `json:"tB"`
	TokenInfos   []TokenInfo  `json:"tI"`
}

type SwapInfo struct {
	IdxIn  int `json:"idxIn"`
	IdxOut int `json:"idxOut"`

	amountIn  *uint256.Int
	amountOut *uint256.Int
	adminFee  *uint256.Int
	fee       *uint256.Int
}

type Meta struct {
	BlockNumber uint64 `json:"bN"`
}
