package types

import (
	"context"
	"math/big"
	"strings"

	encodeValueObject "github.com/KyberNetwork/aggregator-encoding/pkg/constant/valueobject"
	"github.com/KyberNetwork/aggregator-encoding/pkg/types"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	dexValueObject "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/eth"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type IExecutorBalanceRepository interface {
	HasToken(ctx context.Context, executorAddress string, queries []string) ([]bool, error)
	HasPoolApproval(ctx context.Context, executorAddress string, queries []dto.PoolApprovalQuery) ([]bool, error)
}

type EncodingDataBuilder struct {
	ctx                       context.Context
	chainID                   valueobject.ChainID
	data                      types.EncodingData
	executorBalanceRepository IExecutorBalanceRepository
	featureFlags              valueobject.FeatureFlags
}

func NewEncodingDataBuilder(
	ctx context.Context,
	chainID valueobject.ChainID,
	executorBalanceRepository IExecutorBalanceRepository,
	featureFlags valueobject.FeatureFlags,
) *EncodingDataBuilder {
	return &EncodingDataBuilder{
		ctx:                       ctx,
		chainID:                   chainID,
		data:                      types.EncodingData{},
		executorBalanceRepository: executorBalanceRepository,
		featureFlags:              featureFlags,
	}
}

func (b *EncodingDataBuilder) SetSlippageTolerance(slippageTolerance float64) *EncodingDataBuilder {
	b.data.SlippageTolerance = slippageTolerance

	return b
}

func (b *EncodingDataBuilder) SetDeadline(deadline *big.Int) *EncodingDataBuilder {
	b.data.Deadline = deadline

	return b
}

func (b *EncodingDataBuilder) SetClientID(clientID string) *EncodingDataBuilder {
	b.data.ClientID = strings.ReplaceAll(strings.ToLower(clientID), ".", "_")

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

func (b *EncodingDataBuilder) SetReferral(ref string) *EncodingDataBuilder {
	b.data.Referral = ref

	return b
}

func (b *EncodingDataBuilder) SetRoute(
	routeSummary *valueobject.RouteSummary,
	executorAddress string,
	recipient string,
) *EncodingDataBuilder {
	encodingMode := getEncodingMode(routeSummary.TokenIn, routeSummary.Route)
	encodingRoute := transformRoute(routeSummary.Route)

	b.data.TokenIn = routeSummary.TokenIn
	b.data.TokenOut = routeSummary.TokenOut

	encodingRoute = b.updateSwapRecipientAndCollectAmount(
		encodingRoute,
		encodingMode,
		executorAddress,
	)

	if b.featureFlags.IsMergeDuplicateSwapEnabled {
		encodingRoute = b.updateMergeDuplicateSwap(encodingRoute)
	}

	b.data.InputAmount = routeSummary.AmountIn
	b.data.OutputAmount = routeSummary.AmountOut
	b.data.ExtraFee = encodeValueObject.ExtraFee{
		FeeAmount:   routeSummary.ExtraFee.FeeAmount,
		ChargeFeeBy: encodeValueObject.ChargeFeeBy(routeSummary.ExtraFee.ChargeFeeBy),
		IsInBps:     routeSummary.ExtraFee.IsInBps,
		FeeReceiver: routeSummary.ExtraFee.FeeReceiver,
	}
	b.data.Recipient = recipient
	b.data.Route = encodingRoute
	b.data.EncodingMode = encodingMode
	b.data.Flags = getEncodingFlags(encodingMode, routeSummary.ExtraFee)

	return b
}

func (b *EncodingDataBuilder) GetData() types.EncodingData {
	return b.data
}

func (b *EncodingDataBuilder) updateSwapRecipientAndCollectAmount(
	route [][]types.EncodingSwap,
	encodingMode types.EncodingMode,
	executorAddress string,
) [][]types.EncodingSwap {
	flags := b.getRouteEncodingSwapFlags(route, executorAddress)

	// Assuming the first swap in the first path is from tokenIn,
	// or the wrap token of native token.
	routeTokenIn := route[0][0].TokenIn

	for pathIdx, path := range route {
		for swapIdx, swap := range path {
			var (
				nextSwap types.EncodingSwap
				prevSwap types.EncodingSwap
			)

			if swapIdx == len(path)-1 {
				nextSwap = types.ZeroEncodingSwap
			} else {
				nextSwap = path[swapIdx+1]
			}

			if swapIdx == 0 {
				prevSwap = types.ZeroEncodingSwap
			} else {
				prevSwap = path[swapIdx-1]
			}

			route[pathIdx][swapIdx].Flags = flags[pathIdx][swapIdx]
			route[pathIdx][swapIdx].Recipient = getRecipient(swap, nextSwap, executorAddress)
			route[pathIdx][swapIdx].CollectAmount = getCollectAmount(swap, prevSwap, encodingMode)

			// After EX-2542: Merge duplicate swap in route sequence,
			// if the first swap in a path doesn't start from tokenIn,
			// and it's also the last path that start from that token,
			// we need to set the SwapAmount to max uint256 value,
			// indicating that executor will use all the balance of that token
			// for this swap to avoid dust token left in the executor / insufficient
			// amount for the swap.
			if b.featureFlags.IsMergeDuplicateSwapEnabled &&
				swapIdx == 0 &&
				swap.TokenIn != routeTokenIn &&
				(pathIdx == len(route)-1 ||
					swap.TokenIn != route[pathIdx+1][0].TokenIn) {
				route[pathIdx][swapIdx].CollectAmount = bignumber.MAX_UINT_128
			}
		}
	}

	return route
}

func (b *EncodingDataBuilder) updateMergeDuplicateSwap(
	route [][]types.EncodingSwap,
) [][]types.EncodingSwap {
	routeTokenIn := route[0][0].TokenIn

	for pathIdx, path := range route {
		for swapIdx, swap := range path {
			// After EX-2542: Merge duplicate swap in route sequence,
			// if the first swap in a path doesn't start from tokenIn,
			// and it's also the last path that start from that token,
			// we need to add a "fake swap", indicating that executor
			// will use all the balance of that token for this swap to
			// avoid dust token left in the executor / insufficient
			// amount for the swap.
			if swapIdx == 0 &&
				swap.TokenIn != routeTokenIn &&
				(pathIdx == len(route)-1 || swap.TokenIn != route[pathIdx+1][0].TokenIn) {
				route[pathIdx] = append([]types.EncodingSwap{
					{
						TokenIn:    swap.TokenIn,
						SwapAmount: swap.SwapAmount,
						Exchange:   dexValueObject.ExchangeFakePool,
					},
				}, route[pathIdx]...)

				break
			}
		}
	}

	return route
}

func (b *EncodingDataBuilder) getRouteEncodingSwapFlags(route [][]types.EncodingSwap, executorAddress string) [][][]types.EncodingSwapFlag {
	flags := make([][][]types.EncodingSwapFlag, len(route))
	for pathIdx, path := range route {
		flags[pathIdx] = make([][]types.EncodingSwapFlag, len(path))
	}

	if valueobject.IsL2EncoderSupportedChains(b.chainID) {
		return flags
	}

	// If not use optimization, unset ShouldNotKeepDustTokenOut & set ShouldApproveMax
	if !b.featureFlags.IsOptimizeExecutorFlagsEnabled {
		for pathIdx, path := range route {
			for swapIdx := range path {
				flags[pathIdx][swapIdx] = append(flags[pathIdx][swapIdx], types.EncodingSwapFlagShouldApproveMax)
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
	hasTokens, err := b.executorBalanceRepository.HasToken(b.ctx, executorAddress, hasTokenQueries)
	if err == nil {
		idx := 0
		for pathIdx, path := range route {
			for swapIdx := range path {
				if hasTokens[idx] {
					flags[pathIdx][swapIdx] = append(flags[pathIdx][swapIdx], types.EncodingSwapFlagShouldNotKeepDustTokenOut)
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
			approveAddress, err := getAddressToApproveMax(b.chainID, swap)
			if err != nil || len(approveAddress) == 0 {
				continue
			}

			if !common.IsHexAddress(approveAddress) {
				continue
			}

			query := dto.PoolApprovalQuery{
				TokenIn:     swap.TokenIn,
				PoolAddress: strings.ToLower(approveAddress),
			}
			hasPoolApprovalQueries = append(hasPoolApprovalQueries, query)
			mapQueryIndex[len(hasPoolApprovalQueries)-1] = [2]int{pathIdx, swapIdx}
		}
	}

	hasPoolApprovals, err := b.executorBalanceRepository.HasPoolApproval(b.ctx, executorAddress, hasPoolApprovalQueries)
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
			flags[pathIdx][swapIdx] = append(flags[pathIdx][swapIdx], types.EncodingSwapFlagShouldApproveMax)
		}
	} else {
		// In case of error, set ShouldApproveMax for every swap
		for pathIdx, path := range route {
			for swapIdx := range path {
				flags[pathIdx][swapIdx] = append(flags[pathIdx][swapIdx], types.EncodingSwapFlagShouldApproveMax)
			}
		}
	}

	return flags
}

func getEncodingMode(tokenIn string, route [][]valueobject.Swap) types.EncodingMode {
	if canSwapSimpleMode(tokenIn, route) {
		return types.EncodingModeSimple
	}

	return types.EncodingModeNormal
}

func getEncodingFlags(mode types.EncodingMode, extraFee valueobject.ExtraFee) []types.EncodingFlag {
	var flags []types.EncodingFlag

	if mode.IsSimple() {
		flags = append(flags, types.EncodingFlagSimpleSwap)
	}

	if len(extraFee.FeeReceiver) > 0 && len(extraFee.FeeAmount) > 0 {
		if extraFee.IsInBps {
			flags = append(flags, types.EncodingFlagFeeInBps)
		}

		if extraFee.IsChargeFeeByCurrencyOut() {
			flags = append(flags, types.EncodingFlagFeeOnDst)
		}
	}

	return flags
}

func transformRoute(route [][]valueobject.Swap) [][]types.EncodingSwap {
	encodingRoute := make([][]types.EncodingSwap, 0, len(route))

	for _, path := range route {
		encodingPath := make([]types.EncodingSwap, 0, len(path))

		for _, swap := range path {
			encodingPath = append(encodingPath, types.EncodingSwap{
				Pool:       swap.Pool,
				TokenIn:    swap.TokenIn,
				TokenOut:   swap.TokenOut,
				SwapAmount: swap.SwapAmount,
				AmountOut:  swap.AmountOut,
				Exchange:   swap.Exchange,
				PoolType:   swap.PoolType,
				PoolExtra:  swap.PoolExtra,
				Extra:      swap.Extra,
			})
		}

		encodingRoute = append(encodingRoute, encodingPath)
	}

	return encodingRoute
}

func getRecipient(
	curSwap types.EncodingSwap,
	nextSwap types.EncodingSwap,
	executorAddress string,
) string {
	// curSwap is the last swap
	if nextSwap.IsZero() {
		return executorAddress
	}

	if valueobject.CanReceiveTokenBeforeSwap(curSwap.Exchange) && valueobject.CanReceiveTokenBeforeSwap(nextSwap.Exchange) {
		return nextSwap.Pool
	}

	return executorAddress
}

func getCollectAmount(
	curSwap types.EncodingSwap,
	prevSwap types.EncodingSwap,
	encodingMode types.EncodingMode,
) *big.Int {
	if prevSwap.IsZero() && encodingMode.IsSimple() {
		return types.ZeroCollectAmount
	}

	if valueobject.CanReceiveTokenBeforeSwap(prevSwap.Exchange) && valueobject.CanReceiveTokenBeforeSwap(curSwap.Exchange) {
		return types.ZeroCollectAmount
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

		if !valueobject.CanReceiveTokenBeforeSwap(path[0].Exchange) {
			return false
		}
	}

	return true
}

func getAddressToApproveMax(chainID valueobject.ChainID, swap types.EncodingSwap) (string, error) {
	// If the swap is on L2 chains, or if the dex doesn't check the "SHOULD_APPROVE_MAX" flag
	isApproveMaxExchange, usePoolAsApprovalAddress := valueobject.IsApproveMaxExchange(swap.Exchange)
	if !isApproveMaxExchange {
		return "", nil
	}

	if usePoolAsApprovalAddress {
		return swap.Pool, nil
	}

	var approvalInfo *pool.ApprovalInfo
	var err error

	if dexValueObject.IsRFQSource(swap.Exchange) &&
		swap.Exchange != dexValueObject.ExchangeLO1inch {
		// For RFQ dexes, the approval address should be unmarshalled from extra
		// so it can be updated from quote result.
		// LO1inch is an exception, it is RFQ dex without RFQ handler.
		approvalInfo, err = util.AnyToStruct[pool.ApprovalInfo](swap.Extra)
		if err != nil {
			return "", err
		}
	} else {
		// If the pool extra has a custom approval address
		approvalInfo, err = util.AnyToStruct[pool.ApprovalInfo](swap.PoolExtra)
		if err != nil {
			return "", err
		}
	}

	return approvalInfo.ApprovalAddress, nil
}
