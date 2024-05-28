package syncswapv2

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/syncswap"
)

type ExtraStablePool struct {
	syncswap.ExtraStablePool
	A *big.Int `json:"a"`
}

type ExtraAquaPool struct {
	SwapFee0To1Min            *big.Int `json:"swapFee0To1Min"`
	SwapFee0To1Max            *big.Int `json:"swapFee0To1Max"`
	SwapFee0To1Gamma          *big.Int `json:"swapFee0To1Gamma"`
	SwapFee1To0Min            *big.Int `json:"swapFee1To0Min"`
	SwapFee1To0Max            *big.Int `json:"swapFee1To0Max"`
	SwapFee1To0Gamma          *big.Int `json:"swapFee1To0Gamma"`
	Token0PrecisionMultiplier *big.Int `json:"token0PrecisionMultiplier"`
	Token1PrecisionMultiplier *big.Int `json:"token1PrecisionMultiplier"`
	VaultAddress              string   `json:"vaultAddress"`
	PriceScale                *big.Int `json:"priceScale"`
	A                         *big.Int `json:"a"`
	D                         *big.Int `json:"d"`
	Gamma                     *big.Int `json:"gamma"`
	LastPrices                *big.Int `json:"lastPrices"`
	PriceOracle               *big.Int `json:"priceOracle"`
	LastPricesTimestamp       int64    `json:"lastPricesTimestamp"`
	LpSupply                  *big.Int `json:"lpSupply"`
	XcpProfit                 *big.Int `json:"xcpProfit"`
	VirtualPrice              *big.Int `json:"virtualPrice"`
	AllowedExtraProfit        *big.Int `json:"allowedExtraProfit"`
	AdjustmentStep            *big.Int `json:"adjustmentStep"`
	MaHalfTime                *big.Int `json:"maHalfTime"`
	InitialTime               int64    `json:"initialTime"`
	FutureTime                int64    `json:"futureTime"`
	InitialA                  int64    `json:"initialA"`
	FutureA                   int64    `json:"futureA"`
	InitialGamma              int64    `json:"initialGamma"`
	FutureGamma               int64    `json:"futureGamma"`
	FeeManagerAddress         string   `json:"feeManagerAddress"`
}
