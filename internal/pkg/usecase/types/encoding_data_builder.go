package types

import (
	"encoding/json"
	"math/big"
	"strings"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/business"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/eth"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type IExecutorBalanceRepository interface {
	HasToken(executorAddress string, queries []string) ([]bool, error)
	HasPoolApproval(executorAddress string, queries []dto.PoolApprovalQuery) ([]bool, error)
}

type EncodingDataBuilder struct {
	data                           EncodingData
	executorBalanceRepository      IExecutorBalanceRepository
	isOptimizeExecutorFlagsEnabled bool
}

func NewEncodingDataBuilder(
	executorBalanceRepository IExecutorBalanceRepository,
	isOptimizeExecutorFlagsEnabled bool,
) *EncodingDataBuilder {
	return &EncodingDataBuilder{
		data:                           EncodingData{},
		executorBalanceRepository:      executorBalanceRepository,
		isOptimizeExecutorFlagsEnabled: isOptimizeExecutorFlagsEnabled,
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
	recipient string,
) *EncodingDataBuilder {
	encodingMode := getEncodingMode(routeSummary.TokenIn, routeSummary.Route)
	encodingRoute := transformRoute(routeSummary.Route)
	encodingRoute = b.updateSwapRecipientAndCollectAmount(
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

func (b *EncodingDataBuilder) updateSwapRecipientAndCollectAmount(
	route [][]EncodingSwap,
	encodingMode EncodingMode,
	executorAddress string,
) [][]EncodingSwap {
	flags := b.getRouteEncodingSwapFlags(route, executorAddress)

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

			route[pathIdx][swapIdx].Flags = flags[pathIdx][swapIdx]
			route[pathIdx][swapIdx].Recipient = getRecipient(swap, nextSwap, executorAddress)
			route[pathIdx][swapIdx].CollectAmount = getCollectAmount(swap, prevSwap, encodingMode)
		}
	}

	return route
}

func (b *EncodingDataBuilder) getRouteEncodingSwapFlags(route [][]EncodingSwap, executorAddress string) [][][]EncodingSwapFlag {
	flags := make([][][]EncodingSwapFlag, len(route))
	for pathIdx, path := range route {
		flags[pathIdx] = make([][]EncodingSwapFlag, len(path))
	}

	// If not use optimization, unset ShouldNotKeepDustTokenOut & set ShouldApproveMax
	if !b.isOptimizeExecutorFlagsEnabled {
		for pathIdx, path := range route {
			for swapIdx := range path {
				flags[pathIdx][swapIdx] = append(flags[pathIdx][swapIdx], EncodingSwapFlagShouldApproveMax)
			}
		}
		return flags
	}

	executorAddress = strings.ToLower(executorAddress)

	// Set ShouldNotKeepDustTokenOut flag
	var hasTokenQueries []string
	for _, path := range route {
		for _, swap := range path {
			hasTokenQueries = append(hasTokenQueries, swap.TokenOut)
		}
	}
	hasTokens, err := b.executorBalanceRepository.HasToken(executorAddress, hasTokenQueries)
	if err == nil {
		idx := 0
		for pathIdx, path := range route {
			for swapIdx := range path {
				if hasTokens[idx] {
					flags[pathIdx][swapIdx] = append(flags[pathIdx][swapIdx], EncodingSwapFlagShouldNotKeepDustTokenOut)
				}
				idx++
			}
		}
	}

	// Set ShouldApproveMax flag
	var hasPoolApprovalQueries []dto.PoolApprovalQuery
	mapQueryIndex := make(map[int][2]int)

	for pathIdx, path := range route {
		for swapIdx, swap := range path {
			if !valueobject.IsApproveMaxExchange(swap.Exchange) {
				continue
			}
			approveAddress, err := getAddressToApproveMax(swap)
			if err != nil {
				continue
			}
			query := dto.PoolApprovalQuery{
				TokenIn:     swap.TokenIn,
				PoolAddress: approveAddress,
			}
			hasPoolApprovalQueries = append(hasPoolApprovalQueries, query)
			mapQueryIndex[len(hasPoolApprovalQueries)-1] = [2]int{pathIdx, swapIdx}
		}
	}

	hasPoolApprovals, err := b.executorBalanceRepository.HasPoolApproval(executorAddress, hasPoolApprovalQueries)
	if err == nil {
		for idx, hasPoolApproval := range hasPoolApprovals {
			if hasPoolApproval {
				continue
			}
			routeIdx, exist := mapQueryIndex[idx]
			if !exist {
				// Safe check
				continue
			}
			pathIdx := routeIdx[0]
			swapIdx := routeIdx[1]
			flags[pathIdx][swapIdx] = append(flags[pathIdx][swapIdx], EncodingSwapFlagShouldApproveMax)
		}
	} else {
		// In case of error, set ShouldApproveMax for every swap
		for pathIdx, path := range route {
			for swapIdx := range path {
				flags[pathIdx][swapIdx] = append(flags[pathIdx][swapIdx], EncodingSwapFlagShouldApproveMax)
			}
		}
	}

	return flags
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

func transformRoute(route [][]valueobject.Swap) [][]EncodingSwap {
	encodingRoute := make([][]EncodingSwap, 0, len(route))

	for _, path := range route {
		encodingPath := make([]EncodingSwap, 0, len(path))

		for _, swap := range path {
			encodingPath = append(encodingPath, EncodingSwap{
				Pool:              swap.Pool,
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

func getAddressToApproveMax(swap EncodingSwap) (string, error) {
	switch swap.Exchange {
	case
		valueobject.ExchangeBalancerV2Weighted,
		valueobject.ExchangeBalancerV2Stable,
		valueobject.ExchangeBalancerV2ComposableStable,
		valueobject.ExchangeBeethovenXWeighted,
		valueobject.ExchangeBeethovenXStable,
		valueobject.ExchangeBeethovenXComposableStable,
		valueobject.ExchangeVelocoreV2CPMM,
		valueobject.ExchangeVelocoreV2WombatStable:
		{
			poolExtraBytes, err := json.Marshal(swap.PoolExtra)
			if err != nil {
				return "", nil
			}

			var poolExtra struct {
				Vault string `json:"vault"`
			}
			if err = json.Unmarshal(poolExtraBytes, &poolExtra); err != nil {
				return "", nil
			}

			return poolExtra.Vault, nil
		}
	default:
		return swap.Pool, nil
	}
}
