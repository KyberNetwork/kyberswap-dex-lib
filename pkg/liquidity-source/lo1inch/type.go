package lo1inch

import (
	"math/big"

	"github.com/holiman/uint256"
)

type ChainID uint

type SwapSide int

const (
	SwapSideTakeToken0 SwapSide = iota
	SwapSideTakeToken1
	SwapSideUnknown
)

type Order struct {
	Signature            string       `json:"signature"`
	OrderHash            string       `json:"orderHash"`
	RemainingMakerAmount *uint256.Int `json:"remainingMakerAmount"`
	MakerBalance         *uint256.Int `json:"makerBalance"`
	MakerAllowance       *uint256.Int `json:"makerAllowance"`
	MakerAsset           string       `json:"makerAsset"`
	TakerAsset           string       `json:"takerAsset"`
	Salt                 string       `json:"salt"`
	Receiver             string       `json:"receiver"`
	MakingAmount         *uint256.Int `json:"makingAmount"`
	TakingAmount         *uint256.Int `json:"takingAmount"`
	Maker                string       `json:"maker"`
	Extension            string       `json:"extension"`
	MakerTraits          string       `json:"makerTraits"`
	IsMakerContract      bool         `json:"isMakerContract"`
	TakerRate            float64      `json:"-"` // We will not save this field in the datastore, but we need it for filtering the orders

	RemainingTakerAmount *uint256.Int `json:"-"`
	RateWithGasFee       float64      `json:"-"`
	Rate                 float64      `json:"-"`
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
	Signature            string       `json:"signature"`
	OrderHash            string       `json:"orderHash"`
	RemainingMakerAmount *uint256.Int `json:"remainingMakerAmount"`
	MakerBalance         *uint256.Int `json:"makerBalance"`
	MakerAllowance       *uint256.Int `json:"makerAllowance"`
	MakerAsset           string       `json:"makerAsset"`
	TakerAsset           string       `json:"takerAsset"`
	Salt                 string       `json:"salt"`
	Receiver             string       `json:"receiver"`
	MakingAmount         *uint256.Int `json:"makingAmount"`
	TakingAmount         *uint256.Int `json:"takingAmount"`
	Maker                string       `json:"maker"`
	Extension            string       `json:"extension"`
	MakerTraits          string       `json:"makerTraits"`
	IsMakerContract      bool         `json:"isMakerContract"`

	// Some extra fields compared to Order

	// FilledMakingAmount is the amount of maker asset that has been filled
	// But keep in mind that this is just the amount that has been filled after ONE CalcAmountOut call, not the total amount that has been filled in this order
	FilledMakingAmount *uint256.Int `json:"filledMakingAmount"`

	// FilledTakingAmount is the amount of taker asset that has been filled
	// But keep in mind that this is just the amount that has been filled after ONE CalcAmountOut call, not the total amount that has been filled in this order
	FilledTakingAmount *uint256.Int `json:"filledTakingAmount"`

	IsBackup bool `json:"isBackup"`
}

func (o *Order) GetMakerAsset() string {
	return o.MakerAsset
}

func (o *Order) GetTakerAsset() string {
	return o.TakerAsset
}

func (o *Order) GetMakingAmount() *big.Int {
	return o.MakingAmount.ToBig()
}

func (o *Order) GetTakingAmount() *big.Int {
	return o.TakingAmount.ToBig()
}

func (o *Order) GetAvailableMakingAmount() *big.Int {
	return o.RemainingMakerAmount.ToBig()
}

func (o *Order) SetAvailableMakingAmount(amount *big.Int) {
	o.RemainingMakerAmount = uint256.MustFromBig(amount)
}

func (o *Order) GetRemainingTakingAmount() *big.Int {
	return o.RemainingTakerAmount.ToBig()
}

func (o *Order) SetRemainingTakingAmount(amount *big.Int) {
	o.RemainingTakerAmount = uint256.MustFromBig(amount)
}

func (o *Order) GetFilledMakingAmount() *big.Int {
	return big.NewInt(0)
}

func (o *Order) GetRateWithGasFee() float64 {
	return o.RateWithGasFee
}

func (o *Order) SetRateWithGasFee(r float64) {
	o.RateWithGasFee = r
}

func (o *Order) GetRate() float64 {
	return o.Rate
}

func (o *Order) SetRate(r float64) {
	o.Rate = r
}
