package clear

import (
	"math/big"
)

type Metadata struct {
	Offset map[string]int `json:"offset"`
}

type Extra struct {
	SwapAddress string                             `json:"swapAddress"`
	IOUs        []string                           `json:"ious"`     // token address -> iou token address
	Reserves    map[int]map[int]*PreviewSwapResult `json:"reserves"` // token address -> reserve
}
type SwapInfo struct {
	SwapAddress string `json:"swapAddress"`
	IOU         string `json:"iou"`
	ReceiveIOU  bool   `json:"receiveIOU"`
}

// PreviewSwapResult from ClearSwap.previewSwap()
type PreviewSwapResult struct {
	AmountIn  *big.Int
	AmountOut *big.Int
	IOUs      *big.Int `json:"-"`
}

// Gas costs for different operations
type Gas struct {
	Swap int64
}

var DefaultGas = Gas{
	Swap: defaultGas,
}
