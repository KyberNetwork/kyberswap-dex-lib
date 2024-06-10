package buildroute

import (
	"context"
	"fmt"
	"math/big"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"

	"github.com/KyberNetwork/aggregator-encoding/pkg/encode"
	"github.com/KyberNetwork/aggregator-encoding/pkg/encode/clientdata"
	encodeTypes "github.com/KyberNetwork/aggregator-encoding/pkg/types"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	dexValueObject "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	ctxUtils "github.com/KyberNetwork/router-service/internal/pkg/utils/context"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	routerEntities "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/business"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/clientid"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/eth"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/requestid"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
	mapset "github.com/deckarep/golang-set/v2"
)

var OutputChangeNoChange = dto.OutputChange{
	Amount:  "0",
	Percent: 0,
	Level:   dto.OutputChangeLevelNormal,
}

type BuildRouteUseCase struct {
	tokenRepository           ITokenRepository
	priceRepository           IPriceRepository
	poolRepository            IPoolRepository
	executorBalanceRepository IExecutorBalanceRepository
	onchainpriceRepository    IOnchainPriceRepository
	gasEstimator              IGasEstimator
	l1FeeCalculator           IL1FeeCalculator

	rfqHandlerByPoolType map[string]pool.IPoolRFQ
	clientDataEncoder    IClientDataEncoder
	encodeBuilder        encode.IEncodeBuilder

	config Config

	mu sync.RWMutex
}

func NewBuildRouteUseCase(
	tokenRepository ITokenRepository,
	priceRepository IPriceRepository,
	poolRepository IPoolRepository,
	executorBalanceRepository IExecutorBalanceRepository,
	onchainpriceRepository IOnchainPriceRepository,
	gasEstimator IGasEstimator,
	l1FeeCalculator IL1FeeCalculator,
	rfqHandlerByPoolType map[string]pool.IPoolRFQ,
	clientDataEncoder IClientDataEncoder,
	encodeBuilder encode.IEncodeBuilder,
	config Config,
) *BuildRouteUseCase {

	return &BuildRouteUseCase{
		tokenRepository:           tokenRepository,
		priceRepository:           priceRepository,
		poolRepository:            poolRepository,
		executorBalanceRepository: executorBalanceRepository,
		onchainpriceRepository:    onchainpriceRepository,
		gasEstimator:              gasEstimator,
		l1FeeCalculator:           l1FeeCalculator,
		rfqHandlerByPoolType:      rfqHandlerByPoolType,
		clientDataEncoder:         clientDataEncoder,
		encodeBuilder:             encodeBuilder,
		config:                    config,
	}
}

func (uc *BuildRouteUseCase) Handle(ctx context.Context, command dto.BuildRouteCommand) (*dto.BuildRouteResult, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "BuildRouteUseCase.Handle")
	defer span.End()

	routeSummary, err := uc.rfq(ctx, command.Sender, command.Recipient, command.Source, command.RouteSummary, command.SlippageTolerance)
	if err != nil {
		return nil, err
	}

	routeSummary, err = uc.updateRouteSummary(ctx, routeSummary)
	if err != nil {
		return nil, err
	}

	encodedData, err := uc.encode(ctx, command, routeSummary)
	if err != nil {
		return nil, err
	}

	// estimate gas price for a transaction
	estimatedGas, gasInUSD, l1FeeUSD, err := uc.estimateGas(ctx, routeSummary, command, encodedData)
	if err != nil {
		return nil, err
	}

	// the only additional cost for now is L1 fee
	additionalCostMessage := ""
	if l1FeeUSD > 0 {
		additionalCostMessage = constant.AdditionalCostMessageL1Fee
	}

	// NOTE: currently we don't check the route (check if there is a better route or the route returns different amounts)
	// we return what client submitted
	return &dto.BuildRouteResult{
		AmountIn:    routeSummary.AmountIn.String(),
		AmountInUSD: strconv.FormatFloat(routeSummary.AmountInUSD, 'f', -1, 64),

		AmountOut:    routeSummary.AmountOut.String(),
		AmountOutUSD: strconv.FormatFloat(routeSummary.AmountOutUSD, 'f', -1, 64),

		Gas:    strconv.FormatUint(estimatedGas, 10),
		GasUSD: strconv.FormatFloat(gasInUSD, 'f', -1, 64),

		AdditionalCostUsd:     strconv.FormatFloat(l1FeeUSD, 'f', -1, 64),
		AdditionalCostMessage: additionalCostMessage,

		OutputChange: OutputChangeNoChange,

		Data:          encodedData,
		RouterAddress: uc.encodeBuilder.GetEncoder(dexValueObject.ChainID(uc.config.ChainID)).GetRouterAddress(),
	}, nil
}

func (uc *BuildRouteUseCase) ApplyConfig(config Config) {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	uc.config.FeatureFlags = config.FeatureFlags
}

func (uc *BuildRouteUseCase) rfq(
	ctx context.Context,
	sender string,
	recipient string,
	source string,
	routeSummary valueobject.RouteSummary,
	slippageTolerance int64,
) (valueobject.RouteSummary, error) {
	executorAddress := uc.encodeBuilder.
		GetEncoder(dexValueObject.ChainID(uc.config.ChainID)).
		GetExecutorAddress(source)

	for pathIdx, path := range routeSummary.Route {
		for swapIdx, swap := range path {
			rfqHandler, found := uc.rfqHandlerByPoolType[swap.PoolType]
			if !found {
				// This pool type does not have RFQ handler
				// It means that this swap does not need to be processed via RFQ
				logger.Debugf(ctx, "no RFQ handler for pool type: %v", swap.PoolType)
				continue
			}

			var rfqRecipient string
			if swapIdx < len(path)-1 &&
				business.CanReceiveTokenBeforeSwap(path[swapIdx+1].Exchange) &&
				business.CanReceiveTokenBeforeSwap(swap.Exchange) {
				// NOTE: We also need to ensure that current swap
				// can receive token before swap due to smart contract logic

				rfqRecipient = path[swapIdx+1].Pool
			} else {
				rfqRecipient = executorAddress
			}

			result, err := rfqHandler.RFQ(ctx, pool.RFQParams{
				NetworkID:    uint(uc.config.ChainID),
				Sender:       sender,
				Recipient:    recipient,
				RFQSender:    executorAddress,
				RFQRecipient: rfqRecipient,
				Slippage:     slippageTolerance,
				SwapInfo:     swap.Extra,
			})
			if err != nil {
				return routeSummary, errors.WithMessagef(err, "rfq failed, swap data: %v", swap)
			}

			// Enrich the swap extra with the RFQ extra
			routeSummary.Route[pathIdx][swapIdx].Extra = result.Extra

			// We might have to apply the new amount out from RFQ (MM can quote with a different amount out)
			if result.NewAmountOut != nil {
				routeSummary.Route[pathIdx][swapIdx].AmountOut = result.NewAmountOut
			}
		}
	}

	// Recalculate the new amount out after RFQ
	afterRFQAmountOut := big.NewInt(0)
	for _, path := range routeSummary.Route {
		afterRFQAmountOut.Add(afterRFQAmountOut, path[len(path)-1].AmountOut)
	}

	// NOTE: if afterRFQAmountOut < oldAmountOut due to any RFQ hop, we will return error.
	// Reference: https://www.notion.so/kybernetwork/Build-route-behavior-discussion-5a0765555e1e47c1866db5df3d01a0b5
	if afterRFQAmountOut.Cmp(routeSummary.AmountOut) < 0 {
		logger.Errorf(ctx, "afterRFQAmountOut: %v < oldAmountOut: %v, diff = %.2f%%",
			afterRFQAmountOut, routeSummary.AmountOut, 100*float64(afterRFQAmountOut.Uint64())/float64(routeSummary.AmountOut.Uint64()))
		return routeSummary, ErrQuotedAmountSmallerThanEstimated
	}

	return routeSummary, nil
}

// updateRouteSummary updates AmountInUSD/AmountOutUSD, TokenInMarketPriceAvailable/TokenOutMarketPriceAvailable in command.RouteSummary
// and returns updated command
// We need these values, and they should be calculated in backend side because some services such as campaign or data
// need them for their business.
func (uc *BuildRouteUseCase) updateRouteSummary(ctx context.Context, routeSummary valueobject.RouteSummary) (valueobject.RouteSummary, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "BuildRouteUseCase.updateRouteSummary")
	defer span.End()

	tokenInAddress, err := eth.ConvertEtherToWETH(routeSummary.TokenIn, uc.config.ChainID)
	if err != nil {
		return routeSummary, err
	}

	tokenOutAddress, err := eth.ConvertEtherToWETH(routeSummary.TokenOut, uc.config.ChainID)
	if err != nil {
		return routeSummary, err
	}

	tokenIn, tokenOut, err := uc.getTokens(ctx, tokenInAddress, tokenOutAddress)
	if err != nil {
		return routeSummary, err
	}

	if tokenIn == nil {
		return routeSummary, errors.WithMessagef(ErrTokenNotFound, "tokenIn: [%s]", tokenInAddress)
	}

	if tokenOut == nil {
		return routeSummary, errors.WithMessagef(ErrTokenNotFound, "tokenOut: [%s]", tokenOutAddress)
	}

	var (
		tokenInPriceUSD              float64
		tokenInMarketPriceAvailable  bool
		tokenOutPriceUSD             float64
		tokenOutMarketPriceAvailable bool
	)
	if uc.onchainpriceRepository != nil {
		// TODO: check and deprecate these 2 fields since we no longer has `market` price
		tokenInMarketPriceAvailable = false
		tokenOutMarketPriceAvailable = false

		tokenInPriceUSD, tokenOutPriceUSD, err = uc.getPrices(ctx, tokenInAddress, tokenOutAddress)
		if err != nil {
			return routeSummary, err
		}
	} else {
		tokenInPrice, tokenOutPrice, err := uc.getPricesLegacy(ctx, tokenInAddress, tokenOutAddress)
		if err != nil {
			return routeSummary, err
		}

		if tokenInPrice != nil {
			tokenInPriceUSD, tokenInMarketPriceAvailable = tokenInPrice.GetPreferredPrice()
		}

		if tokenOutPrice != nil {
			tokenOutPriceUSD, tokenOutMarketPriceAvailable = tokenOutPrice.GetPreferredPrice()
		}
	}

	amountInUSD := business.CalcAmountUSD(routeSummary.AmountIn, tokenIn.Decimals, tokenInPriceUSD)
	amountOutUSD := business.CalcAmountUSD(routeSummary.AmountOut, tokenOut.Decimals, tokenOutPriceUSD)

	routeSummary.AmountInUSD, _ = amountInUSD.Float64()
	routeSummary.TokenInMarketPriceAvailable = tokenInMarketPriceAvailable

	routeSummary.AmountOutUSD, _ = amountOutUSD.Float64()
	routeSummary.TokenOutMarketPriceAvailable = tokenOutMarketPriceAvailable

	return routeSummary, nil
}

func (uc *BuildRouteUseCase) encode(ctx context.Context, command dto.BuildRouteCommand, routeSummary valueobject.RouteSummary) (string, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "BuildRouteUseCase.encode")
	defer span.End()

	clientData, err := uc.encodeClientData(ctx, command, routeSummary)
	if err != nil {
		return "", err
	}

	encoder := uc.encodeBuilder.GetEncoder(dexValueObject.ChainID(uc.config.ChainID))

	encodingData := types.NewEncodingDataBuilder(
		uc.executorBalanceRepository,
		uc.config.FeatureFlags.IsOptimizeExecutorFlagsEnabled).
		SetRoute(&routeSummary, encoder.GetExecutorAddress(command.Source), command.Recipient).
		SetDeadline(big.NewInt(command.Deadline)).
		SetSlippageTolerance(big.NewInt(command.SlippageTolerance)).
		SetClientID(command.Source).
		SetClientData(clientData).
		SetPermit(command.Permit).
		GetData()

	return encoder.Encode(encodingData)
}

// encodeClientData recalculates amountInUSD and amountOutUSD then perform encoding
func (uc *BuildRouteUseCase) encodeClientData(ctx context.Context, command dto.BuildRouteCommand, routeSummary valueobject.RouteSummary) ([]byte, error) {
	flags, err := clientdata.ConvertFlagsToBitInteger(encodeTypes.Flags{
		TokenInMarketPriceAvailable:  routeSummary.TokenInMarketPriceAvailable,
		TokenOutMarketPriceAvailable: routeSummary.TokenOutMarketPriceAvailable,
	})
	if err != nil {
		return nil, err
	}

	return uc.clientDataEncoder.Encode(ctx, encodeTypes.ClientData{
		Source:       command.Source,
		AmountInUSD:  strconv.FormatFloat(routeSummary.AmountInUSD, 'f', -1, 64),
		AmountOutUSD: strconv.FormatFloat(routeSummary.AmountOutUSD, 'f', -1, 64),
		Referral:     command.Referral,
		Flags:        flags,
	})
}

// getTokens returns tokenIn and tokenOut data
func (uc *BuildRouteUseCase) getTokens(
	ctx context.Context,
	tokenInAddress string,
	tokenOutAddress string,
) (*entity.Token, *entity.Token, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "BuildRouteUseCase.getTokens")
	defer span.End()

	tokens, err := uc.tokenRepository.FindByAddresses(ctx, []string{tokenInAddress, tokenOutAddress})
	if err != nil {
		return nil, nil, err
	}

	var (
		tokenIn  *entity.Token
		tokenOut *entity.Token
	)

	for _, token := range tokens {
		if strings.EqualFold(token.Address, tokenInAddress) {
			tokenIn = token
		}

		if strings.EqualFold(token.Address, tokenOutAddress) {
			tokenOut = token
		}
	}

	return tokenIn, tokenOut, nil
}

// getPrices returns tokenIn and tokenOut price
func (uc *BuildRouteUseCase) getPrices(ctx context.Context, tokenIn, tokenOut string) (float64, float64, error) {
	priceByAddress, err := uc.onchainpriceRepository.FindByAddresses(ctx, []string{tokenIn, tokenOut})
	if err != nil {
		return 0, 0, err
	}

	// use sell price for token in
	tokenInPriceUSD := 0.0
	if price, ok := priceByAddress[tokenIn]; ok && price != nil && price.USDPrice.Sell != nil {
		tokenInPriceUSD, _ = price.USDPrice.Sell.Float64()
	}

	// use buy price for token out and gas
	tokenOutPriceUSD := 0.0
	if price, ok := priceByAddress[tokenOut]; ok && price != nil && price.USDPrice.Buy != nil {
		tokenOutPriceUSD, _ = price.USDPrice.Buy.Float64()
	}

	return tokenInPriceUSD, tokenOutPriceUSD, nil
}

func (uc *BuildRouteUseCase) getPricesLegacy(
	ctx context.Context,
	tokenInAddress string,
	tokenOutAddress string,
) (*entity.Price, *entity.Price, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "BuildRouteUseCase.getPrices")
	defer span.End()

	prices, err := uc.priceRepository.FindByAddresses(ctx, []string{tokenInAddress, tokenOutAddress})
	if err != nil {
		return nil, nil, err
	}

	var (
		tokenInPrice  *entity.Price
		tokenOutPrice *entity.Price
	)

	for _, price := range prices {
		if strings.EqualFold(price.Address, tokenInAddress) {
			tokenInPrice = price
		}

		if strings.EqualFold(price.Address, tokenOutAddress) {
			tokenOutPrice = price
		}
	}

	return tokenInPrice, tokenOutPrice, nil
}

func (uc *BuildRouteUseCase) estimateGas(ctx context.Context, routeSummary valueobject.RouteSummary, command dto.BuildRouteCommand, encodedData string) (uint64, float64, float64, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "BuildRouteUseCase.estimateGas")
	defer span.End()

	value := constant.Zero
	if eth.IsEther(routeSummary.TokenIn) {
		value = routeSummary.AmountIn
	}
	tx := UnsignedTransaction{
		command.Sender,
		encodedData,
		value,
		nil,
	}

	gas := uint64(routeSummary.Gas)
	gasUSD := routeSummary.GasUSD
	var err error
	if uc.config.FeatureFlags.IsGasEstimatorEnabled && command.EnableGasEstimation {
		if utils.IsEmptyString(command.Sender) {
			return 0, 0.0, 0, ErrSenderEmptyWhenEnableEstimateGas
		}

		gas, gasUSD, err = uc.gasEstimator.Execute(ctx, tx)
		uc.sendEstimateGasLogsAndMetrics(ctx, routeSummary, err, command.SlippageTolerance)
		go uc.trackFaultyPools(ctxUtils.NewBackgroundCtxWithReqId(ctx), routeSummary, command, err)
		if err != nil {
			return 0, 0.0, 0, ErrEstimateGasFailed(err)
		}
	} else if uc.config.FeatureFlags.IsFaultyPoolDetectorEnable && !utils.IsEmptyString(command.Sender) {
		go func(ctx context.Context) {
			_, err := uc.gasEstimator.EstimateGas(ctx, tx)
			uc.sendEstimateGasLogsAndMetrics(ctx, routeSummary, err, command.SlippageTolerance)
			uc.trackFaultyPools(ctx, routeSummary, command, err)
		}(ctxUtils.NewBackgroundCtxWithReqId(ctx))
	}

	// for some L2 chains we'll need to account for L1 fee as well
	l1FeeUSDFloat, err := uc.calculateL1FeeUSD(ctx, encodedData)
	if err != nil {
		return 0, 0.0, 0, fmt.Errorf("failed to estimate L1 fee %s", err.Error())
	}

	return gas, gasUSD, l1FeeUSDFloat, nil
}

func (uc *BuildRouteUseCase) calculateL1FeeUSD(ctx context.Context, encodedData string) (float64, error) {
	l1Fee, err := uc.l1FeeCalculator.CalculateL1Fee(ctx, uc.config.ChainID, encodedData)
	if err != nil {
		return 0, fmt.Errorf("failed to estimate L1 fee %s", err.Error())
	}
	l1FeeUSDFloat := 0.0
	if l1Fee != nil {
		// the fee calculated is already in GasToken unit, so just multiply with GasTokenPriceUSD only without GasPrice
		gasPriceUsd, err := uc.gasEstimator.GetGasTokenPriceUSD(ctx)
		if err != nil {
			return 0, fmt.Errorf("failed to get gas token price in USD %s", err.Error())
		}
		l1FeeUSD := new(big.Float).Quo(
			new(big.Float).Mul(
				new(big.Float).SetFloat64(gasPriceUsd),
				new(big.Float).SetInt(l1Fee),
			),
			constant.BoneFloat)
		l1FeeUSDFloat, _ = l1FeeUSD.Float64()
	}
	return l1FeeUSDFloat, nil
}

func (uc *BuildRouteUseCase) sendEstimateGasLogsAndMetrics(ctx context.Context,
	routeSummary valueobject.RouteSummary, err error, slippage int64) {
	clientId := clientid.GetClientIDFromCtx(ctx)
	poolTags := make([]string, 0)

	for _, path := range routeSummary.Route {
		for _, swap := range path {
			metrics.IncrEstimateGas(ctx, err == nil, string(swap.Exchange), clientId)
			poolTags = append(poolTags, fmt.Sprintf("%s:%s", swap.Exchange, swap.Pool))
		}
	}

	if err != nil {
		logger.WithFields(ctx, logger.Fields{
			"requestId": requestid.GetRequestIDFromCtx(ctx),
			"clientId":  clientId,
			"pool":      strings.Join(poolTags, ","),
		}).Infof("EstimateGas failed error %s", err)

		if isErrReturnAmountIsNotEnough(err) {
			// send failed metrics with slippage when error is Return amount is not enough
			metrics.HistogramEstimateGasWithSlippage(ctx, float64(slippage), false)
		} else {
			// send success metrics with slippage
			metrics.HistogramEstimateGasWithSlippage(ctx, float64(slippage), true)
		}
	}
}

func (uc *BuildRouteUseCase) trackFaultyPools(ctx context.Context, routeSummary valueobject.RouteSummary, command dto.BuildRouteCommand, err error) {
	if !uc.config.FeatureFlags.IsFaultyPoolDetectorEnable {
		return
	}

	// requests to be tracked should only involve tokens that have been whitelisted or native token
	if !IsTokenValid(command.RouteSummary.TokenIn, uc.config.FaultyPoolsConfig, uc.config.ChainID) ||
		!IsTokenValid(command.RouteSummary.TokenOut, uc.config.FaultyPoolsConfig, uc.config.ChainID) {
		return
	}

	trackers := []routerEntities.FaultyPoolTracker{}
	failedCount := int64(0)
	// if SlippageTolerance >= 5%, we will consider a pool is faulty, otherwise, we do not encounter it
	// because in case that pool contains FOT token, slippage is high but that pool's state is not stale
	if isErrReturnAmountIsNotEnough(err) && slippageIsAboveMinThreshold(command.SlippageTolerance, uc.config.FaultyPoolsConfig) {
		failedCount = 1
	}
	totalSet := mapset.NewSet[string]()
	for _, path := range routeSummary.Route {
		for _, swap := range path {
			trackers = append(trackers, routerEntities.FaultyPoolTracker{
				Address:     swap.Pool,
				TotalCount:  1,
				FailedCount: failedCount,
			})
			totalSet.Add(swap.Pool)
		}
	}
	// pool-service will return InvalidArgument error if trackers list is empty
	if len(trackers) == 0 {
		return
	}
	results, err := uc.poolRepository.TrackFaultyPools(ctx, trackers)

	if err != nil {
		addedSet := mapset.NewSet(results...)

		logger.WithFields(
			ctx,
			logger.Fields{
				"error":            err,
				"stacktrace":       string(debug.Stack()),
				"failedTrackPools": totalSet.Difference(addedSet),
				"requestId":        requestid.GetRequestIDFromCtx(ctx),
			}).Error("fail to add faulty pools")
	}

}
