package types

import (
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/business"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/eth"
	"github.com/KyberNetwork/router-service/internal/pkg/validator"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type EncodingDataBuilder struct {
	data EncodingData
}

func NewEncodingDataBuilder() *EncodingDataBuilder {
	return &EncodingDataBuilder{
		data: EncodingData{},
	}
}

func (b *EncodingDataBuilder) SetSlippageTolerance(slippageTolerance *big.Int) *EncodingDataBuilder {
	b.data.SlippageTolerance = slippageTolerance

	return b
}

func (b *EncodingDataBuilder) SetDeadline(deadline *big.Int) *EncodingDataBuilder {
	b.data.Deadline = deadline

	return b
}

func (b *EncodingDataBuilder) SetClientData(clientData []byte) *EncodingDataBuilder {
	b.data.ClientData = clientData

	return b
}

func (b *EncodingDataBuilder) SetPermit(permit []byte) *EncodingDataBuilder {
	b.data.Permit = permit

	return b
}

func (b *EncodingDataBuilder) SetRoute(
	routeSummary *valueobject.RouteSummary,
	executorAddress string,
	kyberLOAddress string,
	recipient string,
) *EncodingDataBuilder {
	encodingMode := getEncodingMode(routeSummary.TokenIn, routeSummary.Route)
	encodingRoute := transformRoute(routeSummary.Route, kyberLOAddress)
	encodingRoute = updateSwapRecipientAndCollectAmount(
		encodingRoute,
		encodingMode,
		executorAddress,
	)

	b.data.TokenIn = routeSummary.TokenIn
	b.data.TokenOut = routeSummary.TokenOut
	b.data.InputAmount = routeSummary.AmountIn
	b.data.OutputAmount = routeSummary.AmountOut
	b.data.TotalAmountOut = getTotalAmountOut(encodingRoute)
	b.data.ExtraFee = routeSummary.ExtraFee
	b.data.Recipient = recipient
	b.data.Route = encodingRoute
	b.data.EncodingMode = encodingMode
	b.data.Flags = getEncodingFlags(encodingMode, routeSummary.ExtraFee)

	return b
}

func (b *EncodingDataBuilder) GetData() EncodingData {
	return b.data
}

func getEncodingMode(tokenIn string, route [][]valueobject.Swap) EncodingMode {
	if canSwapSimpleMode(tokenIn, route) {
		return EncodingModeSimple
	}

	return EncodingModeNormal
}

func getEncodingFlags(mode EncodingMode, extraFee valueobject.ExtraFee) []EncodingFlag {
	var flags []EncodingFlag

	if mode.IsSimple() {
		flags = append(flags, EncodingFlagSimpleSwap)
	}

	if len(extraFee.FeeReceiver) > 0 && extraFee.FeeAmount != nil {
		if extraFee.IsInBps {
			flags = append(flags, EncodingFlagFeeInBps)
		}

		if extraFee.IsChargeFeeByCurrencyOut() {
			flags = append(flags, EncodingFlagFeeOnDst)
		}
	}

	return flags
}

func getEncodingSwapFlags(swap EncodingSwap, executorAddress string) []EncodingSwapFlag {
	var flags []EncodingSwapFlag

	// For now: always unset ShouldNotKeepDustTokenOut & set ShouldApproveMax
	flags = append(flags, EncodingSwapFlagShouldApproveMax)

	return flags
}

func transformRoute(route [][]valueobject.Swap, kyberLOAddress string) [][]EncodingSwap {
	encodingRoute := make([][]EncodingSwap, 0, len(route))

	for _, path := range route {
		encodingPath := make([]EncodingSwap, 0, len(path))

		for _, swap := range path {
			encodingPath = append(encodingPath, EncodingSwap{
				Pool:              getPool(&swap, kyberLOAddress),
				TokenIn:           swap.TokenIn,
				TokenOut:          swap.TokenOut,
				SwapAmount:        swap.SwapAmount,
				AmountOut:         swap.AmountOut,
				LimitReturnAmount: swap.LimitReturnAmount,
				Exchange:          swap.Exchange,
				PoolLength:        swap.PoolLength,
				PoolType:          swap.PoolType,
				PoolExtra:         swap.PoolExtra,
				Extra:             swap.Extra,
			})
		}

		encodingRoute = append(encodingRoute, encodingPath)
	}

	return encodingRoute
}

func getPool(swap *valueobject.Swap, kyberLOAddress string) string {
	if swap.Exchange == valueobject.ExchangeKyberSwapLimitOrder {
		if swap.PoolExtra != nil {
			if contractAddress, ok := swap.PoolExtra.(string); ok && validator.IsEthereumAddress(contractAddress) {
				return contractAddress
			} else {
				logger.Debugf("Invalid LO contract address %v %v", swap.PoolExtra, swap.Pool)
			}
		}
		return kyberLOAddress
	}
	return swap.Pool
}

func updateSwapRecipientAndCollectAmount(
	route [][]EncodingSwap,
	encodingMode EncodingMode,
	executorAddress string,
) [][]EncodingSwap {
	for pathIdx, path := range route {
		for swapIdx, swap := range path {
			var (
				nextSwap EncodingSwap
				prevSwap EncodingSwap
			)

			if swapIdx == len(path)-1 {
				nextSwap = ZeroEncodingSwap
			} else {
				nextSwap = path[swapIdx+1]
			}

			if swapIdx == 0 {
				prevSwap = ZeroEncodingSwap
			} else {
				prevSwap = path[swapIdx-1]
			}

			route[pathIdx][swapIdx].Flags = getEncodingSwapFlags(swap, executorAddress)
			route[pathIdx][swapIdx].Recipient = getRecipient(swap, nextSwap, executorAddress)
			route[pathIdx][swapIdx].CollectAmount = getCollectAmount(swap, prevSwap, encodingMode)
		}
	}

	return route
}

func getTotalAmountOut(route [][]EncodingSwap) *big.Int {
	totalAmountOut := big.NewInt(0)

	for _, path := range route {
		totalAmountOut.Add(totalAmountOut, path[len(path)-1].AmountOut)
	}

	return totalAmountOut
}

func getRecipient(
	curSwap EncodingSwap,
	nextSwap EncodingSwap,
	executorAddress string,
) string {
	// curSwap is the last swap
	if nextSwap.IsZero() {
		return executorAddress
	}

	if business.CanReceiveTokenBeforeSwap(curSwap.Exchange) && business.CanReceiveTokenBeforeSwap(nextSwap.Exchange) {
		return nextSwap.Pool
	}

	return executorAddress
}

func getCollectAmount(
	curSwap EncodingSwap,
	prevSwap EncodingSwap,
	encodingMode EncodingMode,
) *big.Int {
	if prevSwap.IsZero() && encodingMode.IsSimple() {
		return ZeroCollectAmount
	}

	if business.CanReceiveTokenBeforeSwap(prevSwap.Exchange) && business.CanReceiveTokenBeforeSwap(curSwap.Exchange) {
		return ZeroCollectAmount
	}

	return curSwap.SwapAmount
}

// canSwapSimpleMode returns true when
// - tokenIn is not native token
// - the first pool of each path should be able to receive the token before calling swap
func canSwapSimpleMode(tokenIn string, route [][]valueobject.Swap) bool {
	if eth.IsEther(tokenIn) {
		return false
	}

	for _, path := range route {
		if len(path) == 0 {
			return false
		}

		if !business.CanReceiveTokenBeforeSwap(path[0].Exchange) {
			return false
		}
	}

	return true
}
