package gsm4626

import "github.com/holiman/uint256"

type StaticExtra struct {
	PriceRatio *uint256.Int `json:"priceRatio"`
}

type Extra struct {
	CanSwap         bool         `json:"canSwap"`
	BuyFee          *uint256.Int `json:"buyFee"`
	SellFee         *uint256.Int `json:"sellFee"`
	CurrentExposure *uint256.Int `json:"currentExposure"`
	ExposureCap     *uint256.Int `json:"exposureCap"`
	Rate            *uint256.Int `json:"rate"`
}

type Meta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type SwapInfo struct {
	IsBuy bool `json:"isBuy"`
}
