package lo1inch

import "math/big"

type ChainID uint

type SwapSide int

const (
	SwapSideTakeToken0 SwapSide = iota
	SwapSideTakeToken1
	SwapSideUnknown
)

type Order struct {
	Signature            string   `json:"signature"`
	OrderHash            string   `json:"orderHash"`
	CreateDateTime       string   `json:"createDateTime"`
	RemainingMakerAmount *big.Int `json:"remainingMakerAmount"`
	MakerBalance         *big.Int `json:"makerBalance"`
	MakerAllowance       *big.Int `json:"makerAllowance"`
	MakerAsset           string   `json:"makerAsset"`
	TakerAsset           string   `json:"takerAsset"`
	Salt                 string   `json:"salt"`
	Receiver             string   `json:"receiver"`
	MakingAmount         *big.Int `json:"makingAmount"`
	TakingAmount         *big.Int `json:"takingAmount"`
	Maker                string   `json:"maker"`
	MakerRate            string   `json:"makerRate"`
	TakerRate            string   `json:"takerRate"`
}

type StaticExtra struct {
	Token0        string `json:"token0"`
	Token1        string `json:"token1"`
	RouterAddress string `json:"routerAddress"`
}

type Extra struct {
	TakeToken0Orders []*Order `json:"takeToken0Orders"`
	TakeToken1Orders []*Order `json:"takeToken1Orders"`
}

type SwapInfo struct {
	AmountIn     string             `json:"amountIn"`
	SwapSide     SwapSide           `json:"swapSide"`
	FilledOrders []*FilledOrderInfo `json:"filledOrders"`
}

type FilledOrderInfo struct {
	Signature            string   `json:"signature"`
	OrderHash            string   `json:"orderHash"`
	CreateDateTime       string   `json:"createDateTime"`
	RemainingMakerAmount *big.Int `json:"remainingMakerAmount"`
	MakerBalance         *big.Int `json:"makerBalance"`
	MakerAllowance       *big.Int `json:"makerAllowance"`
	MakerAsset           string   `json:"makerAsset"`
	TakerAsset           string   `json:"takerAsset"`
	Salt                 string   `json:"salt"`
	Receiver             string   `json:"receiver"`
	MakingAmount         *big.Int `json:"makingAmount"`
	TakingAmount         *big.Int `json:"takingAmount"`
	Maker                string   `json:"maker"`
	MakerRate            string   `json:"makerRate"`
	TakerRate            string   `json:"takerRate"`

	// Some extra fields compared to Order

	// FilledMakingAmount is the amount of maker asset that has been filled
	// But keep in mind that this is just the amount that has been filled after ONE CalcAmountOut call, not the total amount that has been filled in this order
	FilledMakingAmount *big.Int `json:"filledMakingAmount"`

	// FilledTakingAmount is the amount of taker asset that has been filled
	// But keep in mind that this is just the amount that has been filled after ONE CalcAmountOut call, not the total amount that has been filled in this order
	FilledTakingAmount *big.Int `json:"filledTakingAmount"`

	IsBackup bool `json:"isBackup"`
}
