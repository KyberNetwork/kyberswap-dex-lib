package buildroute

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/KyberNetwork/aggregator-encoding/pkg/encode"
	"github.com/KyberNetwork/aggregator-encoding/pkg/encode/clientdata"
	encodeTypes "github.com/KyberNetwork/aggregator-encoding/pkg/types"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	dexValueObject "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/business"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/clientid"
	ctxUtils "github.com/KyberNetwork/router-service/internal/pkg/utils/context"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/eth"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/requestid"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

var OutputChangeNoChange = dto.OutputChange{
	Amount:  "0",
	Percent: 0,
	Level:   dto.OutputChangeLevelNormal,
}

type BuildRouteUseCase struct {
	tokenRepository           ITokenRepository
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

	// Notice: must check route summary to track faulty pools at the beginning of the handle func to avoid route modification during execution
	isFaultyPoolTrackEnable := uc.IsValidToTrackFaultyPools(command.RouteSummary, command.Checksum)

	encoder := uc.encodeBuilder.GetEncoder(uc.config.ChainID)
	executorAddress := strings.ToLower(encoder.GetExecutorAddress(command.Source))

	routeSummary, err := uc.checkToKeepDustTokenOut(ctx, executorAddress, command.RouteSummary)
	if err != nil {
		return nil, err
	}

	command.RouteSummary = routeSummary

	routeSummary, err = uc.rfq(ctx, command.Sender, command.Recipient, command.Source, command.RouteSummary, isFaultyPoolTrackEnable, command.SlippageTolerance)
	if err != nil {
		if strings.Contains(err.Error(), context.DeadlineExceeded.Error()) {
			return nil, ErrRFQTimeout
		}

		return nil, err
	}

	routeSummary, err = uc.updateRouteSummary(ctx, routeSummary)
	if err != nil {
		return nil, err
	}

	encodedData, err := uc.encode(ctx, command, routeSummary, encoder, executorAddress)
	if err != nil {
		return nil, err
	}

	// estimate gas price for a transaction
	estimatedGas, gasInUSD, l1FeeUSD, err := uc.estimateGas(ctx, routeSummary, command, encodedData, isFaultyPoolTrackEnable)
	if err != nil {
		return nil, err
	}

	// the only additional cost for now is L1 fee
	additionalCostMessage := ""
	if l1FeeUSD > 0 {
		additionalCostMessage = constant.AdditionalCostMessageL1Fee
	}

	transactionValue := constant.Zero
	if eth.IsEther(routeSummary.TokenIn) {
		transactionValue = routeSummary.AmountIn
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

		Data:             encodedData,
		RouterAddress:    uc.encodeBuilder.GetEncoder(dexValueObject.ChainID(uc.config.ChainID)).GetRouterAddress(),
		TransactionValue: transactionValue.String(),
	}, nil
}

func (uc *BuildRouteUseCase) ApplyConfig(config Config) {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	uc.config.FeatureFlags = config.FeatureFlags
	uc.config.RFQAcceptableSlippageFraction = config.RFQAcceptableSlippageFraction
}

func (uc *BuildRouteUseCase) rfq(
	ctx context.Context,
	sender string,
	recipient string,
	source string,
	routeSummary valueobject.RouteSummary,
	isFaultyPoolTrackEnable bool,
	slippageTolerance int64,
) (valueobject.RouteSummary, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "BuildRouteUseCase.rfq")
	defer span.End()

	executorAddress := uc.encodeBuilder.
		GetEncoder(uc.config.ChainID).
		GetExecutorAddress(source)

	rfqParamsByPoolType := make(map[string][]valueobject.IndexedRFQParams)
	for pathIdx, path := range routeSummary.Route {
		for swapIdx, swap := range path {
			_, found := uc.rfqHandlerByPoolType[swap.PoolType]
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

			rfqParamsByPoolType[swap.PoolType] = append(rfqParamsByPoolType[swap.PoolType], valueobject.IndexedRFQParams{
				RFQParams: pool.RFQParams{
					NetworkID:    uint(uc.config.ChainID),
					Sender:       sender,
					Recipient:    recipient,
					RFQSender:    executorAddress,
					RFQRecipient: rfqRecipient,
					Slippage:     slippageTolerance,
					SwapInfo:     swap.Extra,
				},
				PathIdx: pathIdx,
				SwapIdx: swapIdx,
			})
		}
	}

	if len(rfqParamsByPoolType) == 0 {
		return routeSummary, nil
	}

	// NOTE: Each swap in the path must process RFQ pools sequentially,
	// as the `newAmountOut` from one swap becomes the `newAmountIn` for the subsequent swap.
	// We can only parallelize processing for different paths within the route.
	//
	// Currently, this version processes RFQs in parallel for all RFQ liquidity sources in the route,
	// does not fulfill the sequential processing requirement for each path, and `newAmountOut`
	// does not impact the subsequent swap.
	g, ctx := errgroup.WithContext(ctx)
	for poolType, paramsSlice := range rfqParamsByPoolType {
		rfqHandler := uc.rfqHandlerByPoolType[poolType]
		if rfqHandler.SupportBatch() {
			g.Go(func() error {
				return uc.processRFQs(ctx, poolType, routeSummary, isFaultyPoolTrackEnable, paramsSlice...)
			})
		} else {
			for _, params := range paramsSlice {
				g.Go(func() error {
					return uc.processRFQs(ctx, poolType, routeSummary, isFaultyPoolTrackEnable, params)
				})
			}
		}
	}

	if err := g.Wait(); err != nil {
		return routeSummary, errors.WithMessage(err, "rfq failed")
	}

	return uc.estimateRFQSlippage(ctx, routeSummary, slippageTolerance)
}

// Currently, we do not recalculate amountIn and amountOut for each swap after RFQ.
// estimateRFQSlippage estimates routeSummary.amountOut after RFQ and compares with acceptableRFQAmountOut.
func (uc *BuildRouteUseCase) estimateRFQSlippage(
	ctx context.Context,
	routeSummary valueobject.RouteSummary,
	slippageTolerance int64,
) (valueobject.RouteSummary, error) {
	// Estimate new amount out after RFQ
	var estimatedAfterRFQAmountOutBF big.Float
	for _, path := range routeSummary.Route {
		var (
			estimatedAmountOutBF big.Float
			previousAmountOutBF  big.Float
		)
		for _, swap := range path {
			amountInBF := new(big.Float).SetInt(swap.SwapAmount)
			estimatedAmountInBF := new(big.Float)
			if previousAmountOutBF.Sign() == 0 || previousAmountOutBF.Cmp(amountInBF) > 0 {
				estimatedAmountInBF.Set(amountInBF)
			} else {
				estimatedAmountInBF.Set(&previousAmountOutBF)
			}

			// estimatedAmountOutBF = amountOut * estimatedAmountInBF / amountInBF
			estimatedAmountOutBF.SetInt(swap.AmountOut).
				Mul(&estimatedAmountOutBF, estimatedAmountInBF).
				Quo(&estimatedAmountOutBF, amountInBF)

			previousAmountOutBF.Set(&estimatedAmountOutBF)
		}

		estimatedAfterRFQAmountOutBF.Add(&estimatedAfterRFQAmountOutBF, &estimatedAmountOutBF)
	}

	estimatedAfterRFQAmountOut := utils.CeilBigFloat(&estimatedAfterRFQAmountOutBF)

	estimatedAfterRFQAmountOutFloat64, _ := estimatedAfterRFQAmountOut.Float64()
	amountOutFloat64, _ := routeSummary.AmountOut.Float64()
	logger.Debugf(ctx, "afterRFQAmountOut: %v, oldAmountOut %v, estimatedSlippage %.2fbps",
		estimatedAfterRFQAmountOut.String(),
		routeSummary.AmountOut,
		(1-estimatedAfterRFQAmountOutFloat64/amountOutFloat64)*float64(valueobject.BasisPoint.Int64()),
	)

	// acceptableRFQAmountOut = routeSummary.AmountOut * (1 - rfqAcceptableSlippageFraction * slippageTolerance)
	// = routeSummary.AmountOut - routeSummary.AmountOut * rfqAcceptableSlippageFraction * slippageTolerance
	reducedAmountOutWithSlippageTolerance := new(big.Int).Div(
		new(big.Int).Mul(
			routeSummary.AmountOut,
			big.NewInt(slippageTolerance),
		),
		valueobject.BasisPoint,
	)

	// If not configured, the RFQ acceptable slippage is 0,
	// leading the acceptableRFQAmountOut = routeSummary.AmountOut,
	// which is the old behavior.
	reducedAmountOutWithRFQSlippageFraction := new(big.Int).Div(
		new(big.Int).Mul(
			reducedAmountOutWithSlippageTolerance,
			big.NewInt(uc.config.RFQAcceptableSlippageFraction),
		),
		valueobject.BasisPoint,
	)

	acceptableRFQAmountOut := new(big.Int).Sub(
		routeSummary.AmountOut,
		reducedAmountOutWithRFQSlippageFraction,
	)

	// NOTE: Previously, if the afterRFQAmountOut < oldAmountOut due to any RFQ hop, an error would be returned.
	// Reference: https://www.notion.so/kybernetwork/Build-route-behavior-discussion-5a0765555e1e47c1866db5df3d01a0b5
	// However, some RFQs may only result in a slightly lower amount out.
	// To handle this, if afterRFQAmountOut is within an acceptable range (determined by uc.config.rfqAcceptableSlippageFraction),
	// we now allow the RFQ to proceed with the swap.
	if estimatedAfterRFQAmountOut.Cmp(acceptableRFQAmountOut) < 0 {
		acceptableRFQAmountOutFloat64, _ := acceptableRFQAmountOut.Float64()

		logger.Errorf(ctx, "afterRFQAmountOut: %v < acceptableRFQAmountOut: %v < oldAmountOut: %v, diff = %.2f%%",
			estimatedAfterRFQAmountOut.String(),
			acceptableRFQAmountOut,
			routeSummary.AmountOut,
			100*estimatedAfterRFQAmountOutFloat64/acceptableRFQAmountOutFloat64)

		return routeSummary, ErrQuotedAmountSmallerThanEstimated
	}

	return routeSummary, nil
}

func (uc *BuildRouteUseCase) processRFQs(
	ctx context.Context,
	poolType string,
	routeSummary valueobject.RouteSummary,
	isFaultyPoolTrackEnable bool,
	indexedRFQParamsSlice ...valueobject.IndexedRFQParams,
) (err error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "BuildRouteUseCase.processRFQs")
	defer span.End()

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic recovered in processRFQs: %v", r)
		}
	}()

	span.SetTag("poolType", poolType)

	rfqHandler := uc.rfqHandlerByPoolType[poolType]

	var results []*pool.RFQResult

	// If len(indexedRFQParamsSlice)=1, prioritize RFQ() over BatchRFQ()
	if rfqHandler.SupportBatch() && len(indexedRFQParamsSlice) > 1 {
		paramsSlice := lo.Map(indexedRFQParamsSlice, func(param valueobject.IndexedRFQParams, _ int) pool.RFQParams {
			return param.RFQParams
		})
		results, err = rfqHandler.BatchRFQ(ctx, paramsSlice)
	} else {
		var result *pool.RFQResult
		result, err = rfqHandler.RFQ(ctx, indexedRFQParamsSlice[0].RFQParams)
		results = []*pool.RFQResult{result}
	}

	swaps := lo.Map(indexedRFQParamsSlice, func(param valueobject.IndexedRFQParams, _ int) valueobject.Swap {
		return routeSummary.Route[param.PathIdx][param.SwapIdx]
	})
	// Track faulty pools if we got RFQ errors due to market too volatile
	go uc.trackFaultyPools(ctxUtils.NewBackgroundCtxWithReqId(ctx),
		uc.convertPMMSwapsToPoolTrackers(swaps, err),
		isFaultyPoolTrackEnable,
	)

	if err != nil {
		return errors.WithMessagef(err, "swaps data: %v", swaps)
	}

	for i, params := range indexedRFQParamsSlice {
		pathIdx, swapIdx := params.PathIdx, params.SwapIdx

		// Enrich the swap extra with the RFQ extra
		routeSummary.Route[pathIdx][swapIdx].Extra = results[i].Extra

		// We might have to apply the new amount out from RFQ (MM can quote with a different amount out)
		if results[i].NewAmountOut != nil {
			routeSummary.Route[pathIdx][swapIdx].AmountOut = results[i].NewAmountOut
		}
	}

	return err
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
	// TODO: check and deprecate these 2 fields since we no longer has `market` price
	tokenInMarketPriceAvailable = false
	tokenOutMarketPriceAvailable = false

	tokenInPriceUSD, tokenOutPriceUSD, err = uc.getPrices(ctx, tokenInAddress, tokenOutAddress)
	if err != nil {
		return routeSummary, err
	}

	amountInUSD := business.CalcAmountUSD(routeSummary.AmountIn, tokenIn.Decimals, tokenInPriceUSD)
	amountOutUSD := business.CalcAmountUSD(routeSummary.AmountOut, tokenOut.Decimals, tokenOutPriceUSD)

	routeSummary.AmountInUSD, _ = amountInUSD.Float64()
	routeSummary.TokenInMarketPriceAvailable = tokenInMarketPriceAvailable

	routeSummary.AmountOutUSD, _ = amountOutUSD.Float64()
	routeSummary.TokenOutMarketPriceAvailable = tokenOutMarketPriceAvailable

	return routeSummary, nil
}

func (uc *BuildRouteUseCase) encode(
	ctx context.Context,
	command dto.BuildRouteCommand,
	routeSummary valueobject.RouteSummary,
	encoder encode.IEncoder,
	executorAddress string,
) (string, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "BuildRouteUseCase.encode")
	defer span.End()

	clientData, err := uc.encodeClientData(ctx, command, routeSummary)
	if err != nil {
		return "", err
	}

	encodingData := types.NewEncodingDataBuilder(
		ctx,
		uc.executorBalanceRepository,
		uc.config.FeatureFlags).
		SetRoute(&routeSummary, executorAddress, command.Recipient).
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
		TokenOut:     routeSummary.TokenOut,
		AmountOut:    routeSummary.AmountOut.String(),
		Timestamp:    time.Now().Unix(),
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

func (uc *BuildRouteUseCase) estimateGas(ctx context.Context, routeSummary valueobject.RouteSummary, command dto.BuildRouteCommand, encodedData string, isFaultyPoolTrackEnable bool) (uint64, float64, float64, error) {
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
		go uc.trackFaultyPools(
			ctxUtils.NewBackgroundCtxWithReqId(ctx),
			uc.convertAMMSwapsToPoolTrackers(routeSummary, err, command),
			isFaultyPoolTrackEnable,
		)
		if err != nil {
			return 0, 0.0, 0, ErrEstimateGasFailed(err)
		}
	} else if uc.config.FeatureFlags.IsFaultyPoolDetectorEnable && !uc.config.FaultyPoolDetectorDisabled && !utils.IsEmptyString(command.Sender) {
		go func(ctx context.Context) {
			_, err := uc.gasEstimator.EstimateGas(ctx, tx)
			uc.sendEstimateGasLogsAndMetrics(ctx, routeSummary, err, command.SlippageTolerance)
			uc.trackFaultyPools(
				ctx,
				uc.convertAMMSwapsToPoolTrackers(routeSummary, err, command),
				isFaultyPoolTrackEnable,
			)
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
			metrics.CountEstimateGas(ctx, err == nil, string(swap.Exchange), clientId)
			poolTags = append(poolTags, fmt.Sprintf("%s:%s", swap.Exchange, swap.Pool))
		}
	}

	if err != nil {
		logger.WithFields(ctx, logger.Fields{
			"requestId": requestid.GetRequestIDFromCtx(ctx),
			"clientId":  clientId,
			"pool":      strings.Join(poolTags, ","),
		}).Infof("EstimateGas failed error %s", err)

		// send failed metrics with slippage when error is Return amount is not enough
		metrics.RecordEstimateGasWithSlippage(ctx, float64(slippage), !isErrReturnAmountIsNotEnough(err))
	}
}

// This function checks the amount of tokenOut that needs to be retained because the executor contract
// keeps 1 wei of token/native token for gas optimization.
// If the executor has a balance of tokenOut, do nothing. Otherwise, decrease amountOut by 1,
// which reduces minReturnAmount by 1 to ensure tx succeeds when slippageTolerance = 0.
//
// In multi-hop paths, multiple pools could reduce amountOut by 1 each time, but since we only allow whitelisted tokens
// or tokenOut in the middle of the path, we only need to check tokenOut balance.
//
// In dev environment, ensure the feature flag isOptimizeExecutorFlagsEnabled is enabled,
// and EncodingSwapFlagShouldNotKeepDustTokenOut is added to every swap in the paths.
func (uc *BuildRouteUseCase) checkToKeepDustTokenOut(
	ctx context.Context,
	executorAddress string,
	routeSummary valueobject.RouteSummary,
) (valueobject.RouteSummary, error) {
	span, _ := tracer.StartSpanFromContext(ctx, "BuildRouteUseCase.checkToKeepDustTokenOut")
	defer span.End()

	hasTokens, err := uc.executorBalanceRepository.HasToken(ctx, executorAddress, []string{routeSummary.TokenOut})
	if err != nil {
		return routeSummary, err
	}

	if !hasTokens[0] {
		routeSummary.AmountOut.Sub(routeSummary.AmountOut, big.NewInt(1))
		if routeSummary.AmountOut.Cmp(big.NewInt(0)) <= 0 {
			return routeSummary, ErrCannotKeepDustTokenOut
		}
	}
	return routeSummary, nil
}
