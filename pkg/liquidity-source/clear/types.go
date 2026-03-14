package clear

import (
	"github.com/holiman/uint256"
)

type Metadata struct {
	Offset map[string]int `json:"offset"`
}

type Extra struct {
	SwapAddress string       `json:"s"`
	IOUs        []string     `json:"i"` // tokenIdx -> iou token address
	Rates       [][]AmtInOut `json:"p"` // tokenIn -> tokenOut -> [amtIn, amtOut]
}

type AmtInOut [2]*uint256.Int

type SwapInfo struct {
	SwapAddress string `json:"swapAddress"`
	IOU         string `json:"iou"`
	ReceiveIOU  bool   `json:"receiveIOU"`
}
