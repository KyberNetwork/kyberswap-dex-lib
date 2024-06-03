package syncswapv2aqua

import (
	"github.com/holiman/uint256"
)

type ExtraAquaPool struct {
	SwapFee0To1Min            *uint256.Int `json:"swapFee0To1Min"`
	SwapFee0To1Max            *uint256.Int `json:"swapFee0To1Max"`
	SwapFee0To1Gamma          *uint256.Int `json:"swapFee0To1Gamma"`
	SwapFee1To0Min            *uint256.Int `json:"swapFee1To0Min"`
	SwapFee1To0Max            *uint256.Int `json:"swapFee1To0Max"`
	SwapFee1To0Gamma          *uint256.Int `json:"swapFee1To0Gamma"`
	Token0PrecisionMultiplier *uint256.Int `json:"token0PrecisionMultiplier"`
	Token1PrecisionMultiplier *uint256.Int `json:"token1PrecisionMultiplier"`
	VaultAddress              string       `json:"vaultAddress"`
	PriceScale                *uint256.Int `json:"priceScale"`
	A                         *uint256.Int `json:"a"`
	D                         *uint256.Int `json:"d"`
	Gamma                     *uint256.Int `json:"gamma"`
	LastPrices                *uint256.Int `json:"lastPrices"`
	PriceOracle               *uint256.Int `json:"priceOracle"`
	LastPricesTimestamp       int64        `json:"lastPricesTimestamp"`
	LpSupply                  *uint256.Int `json:"lpSupply"`
	XcpProfit                 *uint256.Int `json:"xcpProfit"`
	VirtualPrice              *uint256.Int `json:"virtualPrice"`
	AllowedExtraProfit        *uint256.Int `json:"allowedExtraProfit"`
	AdjustmentStep            *uint256.Int `json:"adjustmentStep"`
	MaHalfTime                *uint256.Int `json:"maHalfTime"`
	InitialTime               int64        `json:"initialTime"`
	FutureTime                int64        `json:"futureTime"`
	InitialA                  int64        `json:"initialA"`
	FutureA                   int64        `json:"futureA"`
	InitialGamma              int64        `json:"initialGamma"`
	FutureGamma               int64        `json:"futureGamma"`
	FeeManagerAddress         string       `json:"feeManagerAddress"`
}
