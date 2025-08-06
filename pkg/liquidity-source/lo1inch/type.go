package lo1inch

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/lo1inch/helper"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/lo1inch/helper/utils"
	"github.com/ethereum/go-ethereum/common"
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

	// for calc taking & making amount after fee
	ExtensionInstance *helper.Extension         `json:"-"`
	FeeTakerExtension *helper.FeeTakerExtension `json:"-"`

	MakerTraitsInstance *helper.MakerTraits `json:"-"`
}

type StaticExtra struct {
	Token0        string `json:"token0"`
	Token1        string `json:"token1"`
	RouterAddress string `json:"routerAddress"`
	TakerAddress  string `json:"takerAddress"`
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

type MetaInfo struct {
	ApprovalAddress string `json:"approvalAddress"`
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

func (o *Order) CalcTakingAmount(
	taker common.Address,
	makingAmount *uint256.Int,
) (*uint256.Int, error) {
	// if the order only allow full fill, we need to return error
	if o.MakerTraitsInstance != nil && !o.MakerTraitsInstance.IsPartialFillAllowed() {
		if makingAmount.Cmp(o.MakingAmount) != 0 {
			return nil, ErrOnlyAllowFullFill
		}

		return o.TakingAmount, nil
	}

	takingAmount := utils.CalcTakingAmount(
		makingAmount,
		o.MakingAmount,
		o.TakingAmount,
	)

	if o.ExtensionInstance.IsEmpty() {
		return takingAmount, nil
	}

	// in case of having fee logic, we need to check if the fee taker extension is not nil
	if o.FeeTakerExtension == nil {
		return nil, ErrFeeTakerExtensionNotFound
	}

	return uint256.MustFromBig(
		o.FeeTakerExtension.GetTakingAmount(
			taker,
			takingAmount.ToBig(),
		),
	), nil
}

func (o *Order) CalcMakingAmount(
	taker common.Address,
	takingAmount *uint256.Int,
) (*uint256.Int, error) {
	// if the order only allow full fill, we need to return error
	if o.MakerTraitsInstance != nil && !o.MakerTraitsInstance.IsPartialFillAllowed() {
		if takingAmount.Cmp(o.TakingAmount) != 0 {
			return nil, ErrOnlyAllowFullFill
		}

		return o.MakingAmount, nil
	}

	makingAmount := utils.CalcMakingAmount(
		takingAmount,
		o.MakingAmount,
		o.TakingAmount,
	)

	// if there is no extension, we can return the making amount trivially
	if o.ExtensionInstance.IsEmpty() {
		return makingAmount, nil
	}

	// in case of having fee logic, we need to check if the fee taker extension is not nil
	if o.FeeTakerExtension == nil {
		return nil, ErrFeeTakerExtensionNotFound
	}

	return uint256.MustFromBig(
		o.FeeTakerExtension.GetMakingAmount(
			taker,
			makingAmount.ToBig(),
		),
	), nil
}
