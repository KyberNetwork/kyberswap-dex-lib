package buildroute

import (
	"context"
	"fmt"
	"math/big"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	v1 "github.com/KyberNetwork/aggregation-stats/messages/v1"
	"github.com/KyberNetwork/aggregator-encoding/pkg/encode"
	"github.com/KyberNetwork/aggregator-encoding/pkg/encode/clientdata"
	encodeTypes "github.com/KyberNetwork/aggregator-encoding/pkg/types"
	"github.com/KyberNetwork/kutils/klog"
	kyberpmm "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/liquidity-source/kyber-pmm"
	"github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/liquidity-source/onebit"
	privo "github.com/KyberNetwork/kyberswap-dex-lib-private/pkg/valueobject"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	dexValueObject "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	routerpoolpkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/alphafee"
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

type RouteType string

const (
	AMM RouteType = "AMM"
	RFQ RouteType = "RFQ"
)

type BuildRouteUseCase struct {
	tokenRepository           ITokenRepository
	poolRepository            IPoolRepository
	executorBalanceRepository IExecutorBalanceRepository
	onchainpriceRepository    IOnchainPriceRepository
	alphaFeeRepository        IAlphaFeeRepository
	publisherRepository       IPublisherRepository
	gasEstimator              IGasEstimator
	l1FeeCalculator           IL1FeeCalculator

	rfqHandlerByExchange map[valueobject.Exchange]pool.IPoolRFQ
	clientDataEncoder    IClientDataEncoder
	encoder              encode.IEncoder

	alphaFeeCalculation *alphafee.AlphaFeeCalculation

	config Config

	mu sync.RWMutex
}

func NewBuildRouteUseCase(
	tokenRepository ITokenRepository,
	poolRepository IPoolRepository,
	executorBalanceRepository IExecutorBalanceRepository,
	onchainPriceRepository IOnchainPriceRepository,
	alphaFeeRepository IAlphaFeeRepository,
	publisherRepository IPublisherRepository,
	gasEstimator IGasEstimator,
	l1FeeCalculator IL1FeeCalculator,
	rfqHandlerByExchange map[valueobject.Exchange]pool.IPoolRFQ,
	clientDataEncoder IClientDataEncoder,
	encoder encode.IEncoder,
	config Config,
) *BuildRouteUseCase {
	return &BuildRouteUseCase{
		tokenRepository:           tokenRepository,
		poolRepository:            poolRepository,
		executorBalanceRepository: executorBalanceRepository,
		onchainpriceRepository:    onchainPriceRepository,
		alphaFeeRepository:        alphaFeeRepository,
		publisherRepository:       publisherRepository,
		gasEstimator:              gasEstimator,
		l1FeeCalculator:           l1FeeCalculator,
		rfqHandlerByExchange:      rfqHandlerByExchange,
		clientDataEncoder:         clientDataEncoder,
		encoder:                   encoder,
		config:                    config,

		alphaFeeCalculation: alphafee.NewAlphaFeeCalculation(config.AlphaFeeConfig,
			routerpoolpkg.NewCustomFuncs(nil)),
	}
}

func (uc *BuildRouteUseCase) Handle(ctx context.Context, command dto.BuildRouteCommand) (*dto.BuildRouteResult, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "BuildRouteUseCase.Handle")
	defer span.End()

	isValidChecksum := uc.IsValidChecksum(command.RouteSummary, command.Checksum)
	if uc.config.FeatureFlags.IsAlphaFeeReductionEnable {
		if !isValidChecksum { // the route might have an alphaFee
			command.RouteSummary.AlphaFee, _ = uc.alphaFeeRepository.GetByRouteId(ctx, command.RouteSummary.RouteID)
			isValidChecksum = uc.IsValidChecksum(command.RouteSummary, command.Checksum)
		}
		if command.RouteSummary.AlphaFee == nil { // charge default fee if no best amm
			command.RouteSummary.AlphaFee, _ = uc.alphaFeeCalculation.CalculateDefaultAlphaFee(
				ctx, alphafee.DefaultAlphaFeeParams{
					RouteSummary: command.RouteSummary,
				})
		}
	}

	if !isValidChecksum && uc.config.ValidateChecksumBySource[command.Source] {
		return nil, ErrInvalidRouteChecksum
	}

	// Notice: must check route summary to track faulty pools at the beginning of the handle func to avoid route modification during execution
	var isFaultyPoolTrackEnable bool
	if isValidChecksum && uc.config.FeatureFlags.IsFaultyPoolDetectorEnable {
		isFaultyPoolTrackEnable = uc.IsValidToTrackFaultyPools(command.RouteSummary.Timestamp)
	}

	executorAddress := strings.ToLower(uc.encoder.GetExecutorAddress(command.Source))

	routeSummary, err := uc.checkToKeepDustTokenOut(ctx, executorAddress, command.RouteSummary)
	if err != nil {
		return nil, err
	}

	command.RouteSummary = routeSummary

	// Prepare tokens and prices data
	tokenInAddress, err := eth.ConvertEtherToWETH(routeSummary.TokenIn, uc.config.ChainID)
	if err != nil {
		return nil, err
	}
	tokenOutAddress, err := eth.ConvertEtherToWETH(routeSummary.TokenOut, uc.config.ChainID)
	if err != nil {
		return nil, err
	}

	addresses := uc.getRouteAndAlphaTokens(ctx, tokenInAddress, tokenOutAddress, routeSummary)
	tokens, err := uc.getTokens(ctx, addresses)
	if err != nil {
		return nil, err
	}

	if tokens[tokenInAddress] == nil {
		return nil, errors.WithMessagef(ErrTokenNotFound, "tokenIn: [%s]", tokenInAddress)
	}
	if tokens[tokenOutAddress] == nil {
		return nil, errors.WithMessagef(ErrTokenNotFound, "tokenOut: [%s]", tokenOutAddress)
	}

	prices, err := uc.getPrices(ctx, tokenInAddress, tokenOutAddress, addresses)
	if err != nil {
		return nil, err
	}

	// Initialize a message to track route that contain an `alpha fee`
	var rfqRouteMsg = new(v1.RouteSummary)
	routeSummary, err = uc.rfq(ctx, command.Sender, command.Recipient, command.Source, command.RouteSummary,
		rfqRouteMsg, isFaultyPoolTrackEnable, command.SlippageTolerance, tokens, prices)

	if err != nil {
		if strings.Contains(err.Error(), context.DeadlineExceeded.Error()) {
			return nil, ErrRFQTimeout
		}

		return nil, err
	}

	routeSummary, err = uc.updateRouteSummary(routeSummary, tokenInAddress, tokenOutAddress, tokens, prices)
	if err != nil {
		return nil, err
	}

	encodedData, err := uc.encode(ctx, command, routeSummary, uc.encoder, executorAddress)
	if err != nil {
		return nil, err
	}

	// estimate gas price for a transaction
	estimatedGas, gasInUSD, l1FeeUSD, err := uc.estimateGas(ctx, routeSummary, command, encodedData,
		isFaultyPoolTrackEnable)
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

	// parse message, we always send events to kafka although the route might not contain alpha fee
	// TODO: refactor convert routeSummary to rfqRouteMsg later for more understanding
	// just a temporarily check if rfqRouteMsg hasn't init yet, we will not send it to kafka
	if rfqRouteMsg.RouteId != "" {
		data, err := proto.Marshal(rfqRouteMsg)
		if err != nil {
			logger.Errorf(ctx, "ConsumerGroupHandler.ConsumeClaim unable to marshal protobuf message %v", err)
		} else {
			err = uc.publisherRepository.Publish(ctx, uc.config.PublisherConfig.AggregatorTransactionTopic, data)
			if err != nil {
				// ignore error
				logger.Errorf(ctx, "ConsumerGroupHandler.ConsumeClaim unable to push message to kafka %v", err)
			}
		}
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
		RouterAddress:    uc.encoder.GetRouterAddress(),
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
	rfqRouteMsg *v1.RouteSummary,
	isFaultyPoolTrackEnable bool,
	slippageTolerance float64,
	tokens map[string]*entity.Token,
	prices map[string]float64,
) (valueobject.RouteSummary, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "BuildRouteUseCase.rfq")
	defer span.End()

	executorAddress := uc.encoder.GetExecutorAddress(source)

	rfqParamsByExchange := make(map[valueobject.Exchange][]valueobject.IndexedRFQParams)
	totalSwap := 0
	for pathIdx, path := range routeSummary.Route {
		for swapIdx, swap := range path {
			_, found := uc.rfqHandlerByExchange[swap.Exchange]
			if !found {
				// This exchange does not have RFQ handler
				// It means that this swap does not need to be processed via RFQ
				logger.Debugf(ctx, "no RFQ handler for pool type: %v", swap.Exchange)
				continue
			}

			var rfqRecipient string
			if swapIdx < len(path)-1 &&
				valueobject.CanReceiveTokenBeforeSwap(path[swapIdx+1].Exchange) &&
				valueobject.CanReceiveTokenBeforeSwap(swap.Exchange) {
				// NOTE: We also need to ensure that current swap
				// can receive token before swap due to smart contract logic

				rfqRecipient = path[swapIdx+1].Pool
			} else {
				rfqRecipient = executorAddress
			}

			var alphaFee *big.Int
			if routeSummary.AlphaFee != nil &&
				routeSummary.AlphaFee.ExecutedId == int32(totalSwap+swapIdx) {
				alphaFee = routeSummary.AlphaFee.Amount
			}

			rfqParamsByExchange[swap.Exchange] = append(rfqParamsByExchange[swap.Exchange],
				valueobject.IndexedRFQParams{
					RFQParams: pool.RFQParams{
						NetworkID:    uc.config.ChainID,
						RequestID:    routeSummary.RouteID,
						Sender:       sender,
						Recipient:    recipient,
						RFQSender:    executorAddress,
						RFQRecipient: rfqRecipient,
						Source:       source,
						TokenIn:      swap.TokenIn,
						TokenOut:     swap.TokenOut,
						SwapAmount:   swap.SwapAmount,
						AmountOut:    swap.AmountOut,
						Slippage:     int64(slippageTolerance),
						PoolExtra:    swap.PoolExtra,
						SwapInfo:     swap.Extra,
						FeeInfo:      alphaFee,
					},
					PathIdx:    pathIdx,
					SwapIdx:    swapIdx,
					ExecutedId: int32(totalSwap + swapIdx),
				})
		}
		totalSwap += len(path)
	}

	if len(rfqParamsByExchange) == 0 {
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
	for exchange, paramsSlice := range rfqParamsByExchange {
		rfqHandler := uc.rfqHandlerByExchange[exchange]
		if rfqHandler.SupportBatch() {
			g.Go(func() error {
				return uc.processRFQs(ctx, exchange, routeSummary, rfqRouteMsg, isFaultyPoolTrackEnable, tokens,
					prices, paramsSlice...)
			})
		} else {
			for _, params := range paramsSlice {
				g.Go(func() error {
					return uc.processRFQs(ctx, exchange, routeSummary, rfqRouteMsg, isFaultyPoolTrackEnable, tokens,
						prices, params)
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
	slippageTolerance float64,
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
			big.NewInt(int64(slippageTolerance)),
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
	exchange valueobject.Exchange,
	routeSummary valueobject.RouteSummary,
	rfqRouteMsg *v1.RouteSummary,
	isFaultyPoolTrackEnable bool,
	tokens map[string]*entity.Token,
	prices map[string]float64,
	indexedRFQParamsSlice ...valueobject.IndexedRFQParams,
) (err error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "BuildRouteUseCase.processRFQs")
	defer span.End()

	defer func() {
		if r := recover(); r != nil {
			klog.Errorf(ctx, "panic: %v\n%s", r, string(debug.Stack()))
			err = fmt.Errorf("panic recovered in processRFQs: %v", r)
		}
	}()

	span.SetTag("exchange", string(exchange))

	rfqHandler := uc.rfqHandlerByExchange[exchange]

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

		uc.extractAlphaFee(results[i].Extra, tokens, prices, routeSummary, params.ExecutedId, pathIdx, swapIdx, rfqRouteMsg)
	}

	return err
}

func (uc *BuildRouteUseCase) extractAlphaFee(extra any, tokens map[string]*entity.Token,
	prices map[string]float64, routeSummary valueobject.RouteSummary, executedId int32, pathIdx, swapIdx int,
	rfqRouteMsg *v1.RouteSummary) {
	extraWithAlphaFee, ok := extra.(WithAlphaFee)
	if !ok {
		return
	}

	alphaFeeAmt, alphaFeeAsset := extraWithAlphaFee.AlphaFee()
	if alphaFeeAmt == nil {
		return
	}

	alphaFeeAsset = strings.ToLower(alphaFeeAsset)

	alphaFeeInUSD := business.CalcAmountUSD(alphaFeeAmt, tokens[alphaFeeAsset].Decimals, prices[alphaFeeAsset])

	alphaFeeInUSDFloat, _ := alphaFeeInUSD.Float64()
	rfqRouteMsg.TotalAlphaFeeInUsd += alphaFeeInUSDFloat

	// we must update alpha fee because alpha fee can be changed, and it might be equal to ps
	if routeSummary.AlphaFee != nil &&
		routeSummary.AlphaFee.ExecutedId == executedId {
		routeSummary.AlphaFee = &routerEntity.AlphaFee{
			AlphaFeeToken: alphaFeeAsset,
			Amount:        alphaFeeAmt,
			AmountUsd:     alphaFeeInUSDFloat,
			Pool:          routeSummary.Route[pathIdx][swapIdx].Pool,
			AMMAmount:     routeSummary.AlphaFee.AMMAmount,
			ExecutedId:    executedId,
			TokenIn:       routeSummary.AlphaFee.TokenIn,
		}
	}

	uc.convertToRouterSwappedEvent(routeSummary, routeSummary.Route[pathIdx][swapIdx].Exchange, extra, rfqRouteMsg)
}

// TODO refactor later for other rfqs
func (uc *BuildRouteUseCase) convertToRouterSwappedEvent(routeSummary valueobject.RouteSummary,
	rfq dexValueObject.Exchange, extra any, rfqRouteMsg *v1.RouteSummary) {
	// General information
	rfqRouteMsg.RouteId = routeSummary.RouteID
	rfqRouteMsg.RfqSource = string(rfq)
	rfqRouteMsg.SellToken = routeSummary.TokenIn
	rfqRouteMsg.BuyToken = routeSummary.TokenOut
	rfqRouteMsg.RequestedAmount = routeSummary.AmountIn.Text(10)
	rfqRouteMsg.VolumeInUsd = routeSummary.AmountInUSD
	rfqRouteMsg.AmountOut = routeSummary.AmountOut.Text(10)

	// info related to alpha fee, incase we don't have alpha fee, we don't need to track these fields
	// because we only care about positive slippage
	if routeSummary.AlphaFee != nil {
		rfqRouteMsg.AmmAmount = routeSummary.AlphaFee.AMMAmount.Text(10)
		rfqRouteMsg.AlphaFee = routeSummary.AlphaFee.Amount.Text(10)
		rfqRouteMsg.AlphaFeeToken = routeSummary.AlphaFee.AlphaFeeToken
		rfqRouteMsg.AlphaFeeInUsd = routeSummary.AlphaFee.AmountUsd
		rfqRouteMsg.ExecutedId = routeSummary.AlphaFee.ExecutedId
	}

	switch rfq {
	case dexValueObject.ExchangeKyberPMM:
		kyberpmmExtra, ok := extra.(kyberpmm.RFQExtra)
		if ok {
			rfqRouteMsg.QuoteTimestamp = timestamppb.New(time.Unix(kyberpmmExtra.QuoteTimestamp, 0))
			takerAmount, _ := new(big.Int).SetString(kyberpmmExtra.TakerAmount, 10)
			makerAmount, _ := new(big.Int).SetString(kyberpmmExtra.MakerAmount, 10)

			rfqRouteMsg.TakerAmount = takerAmount.Text(10)
			rfqRouteMsg.MakerAmount = makerAmount.Text(10)
			rfqRouteMsg.TakerAsset = kyberpmmExtra.TakerAsset
			rfqRouteMsg.MakerAsset = kyberpmmExtra.MakerAsset
			rfqRouteMsg.PartnerName = kyberpmmExtra.Partner
		}
		rfqRouteMsg.RouteType = string(RFQ)

	case dexValueObject.ExchangePmm2:
		onebitExtra, ok := extra.(onebit.RFQExtra)
		if ok {
			rfqRouteMsg.QuoteTimestamp = timestamppb.New(time.Unix(onebitExtra.QuoteTimestamp, 0))
			takerAmount, _ := new(big.Int).SetString(onebitExtra.TakerAmount, 10)
			makerAmount, _ := new(big.Int).SetString(onebitExtra.MakerAmount, 10)

			rfqRouteMsg.TakerAmount = takerAmount.Text(10)
			rfqRouteMsg.MakerAmount = makerAmount.Text(10)
			rfqRouteMsg.TakerAsset = onebitExtra.TakerAsset
			rfqRouteMsg.MakerAsset = onebitExtra.MakerAsset
			rfqRouteMsg.PartnerName = onebitExtra.Partner
		}
		rfqRouteMsg.RouteType = string(RFQ)

	case dexValueObject.ExchangeUniswapV4Kem:
		// TODO implement me
		rfqRouteMsg.RouteType = string(RFQ)

	default:
		rfqRouteMsg.RouteType = string(AMM)
	}
}

// updateRouteSummary updates AmountInUSD/AmountOutUSD in command.RouteSummary
// and returns updated command
// We need these values, and they should be calculated in backend side because some services such as campaign or data
// need them for their business.
func (uc *BuildRouteUseCase) updateRouteSummary(
	routeSummary valueobject.RouteSummary,
	tokenIn, tokenOut string,
	tokens map[string]*entity.Token,
	prices map[string]float64) (valueobject.RouteSummary, error) {

	amountInUSD := business.CalcAmountUSD(routeSummary.AmountIn, tokens[tokenIn].Decimals, prices[tokenIn])
	amountOutUSD := business.CalcAmountUSD(routeSummary.AmountOut, tokens[tokenOut].Decimals, prices[tokenOut])

	routeSummary.AmountInUSD, _ = amountInUSD.Float64()
	routeSummary.AmountOutUSD, _ = amountOutUSD.Float64()

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
		SetSlippageTolerance(command.SlippageTolerance).
		SetClientID(command.Source).
		SetClientData(clientData).
		SetPermit(command.Permit).
		SetReferral(lo.CoalesceOrEmpty(command.Referral, uc.config.ClientRefCode[command.Source])).
		GetData()
	return encoder.Encode(encodingData)
}

// encodeClientData recalculates amountInUSD and amountOutUSD then perform encoding
func (uc *BuildRouteUseCase) encodeClientData(ctx context.Context, command dto.BuildRouteCommand,
	routeSummary valueobject.RouteSummary) ([]byte, error) {
	flags, err := clientdata.ConvertFlagsToBitInteger(encodeTypes.Flags{})
	if err != nil {
		return nil, err
	}

	return uc.clientDataEncoder.Encode(ctx, encodeTypes.ClientData{
		Source:       command.Source,
		AmountInUSD:  strconv.FormatFloat(routeSummary.AmountInUSD, 'f', -1, 64),
		AmountOutUSD: strconv.FormatFloat(routeSummary.AmountOutUSD, 'f', -1, 64),
		Referral:     lo.CoalesceOrEmpty(command.Referral, uc.config.ClientRefCode[command.Source]),
		Flags:        flags,
		TokenOut:     routeSummary.TokenOut,
		AmountOut:    routeSummary.AmountOut.String(),
		Timestamp:    time.Now().Unix(),
		RouteID:      command.RouteSummary.RouteID,
	})
}

func (uc *BuildRouteUseCase) getRouteAndAlphaTokens(_ context.Context,
	tokenIn string, tokenOut string,
	routeSummary valueobject.RouteSummary) []string {
	addresses := mapset.NewThreadUnsafeSet(tokenIn, tokenOut)
	for _, path := range routeSummary.Route {
		for _, swap := range path {
			if privo.IsAlphaFeeSource(swap.PoolType) {
				addresses.Add(swap.TokenOut)
			}
		}
	}

	return addresses.ToSlice()
}

// getTokens returns tokenIn and tokenOut data
func (uc *BuildRouteUseCase) getTokens(
	ctx context.Context,
	addresses []string) (map[string]*entity.Token, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "BuildRouteUseCase.getTokens")
	defer span.End()

	tokens, err := uc.tokenRepository.FindByAddresses(ctx, addresses)
	if err != nil {
		return nil, err
	}

	result := map[string]*entity.Token{}
	for _, token := range tokens {
		result[token.Address] = token
	}

	return result, nil
}

// getPrices returns tokenIn and tokenOut price
func (uc *BuildRouteUseCase) getPrices(ctx context.Context,
	tokenIn string,
	tokenOut string,
	addresses []string) (map[string]float64, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "BuildRouteUseCase.getPrices")
	defer span.End()

	priceByAddress, err := uc.onchainpriceRepository.FindByAddresses(ctx, addresses)
	if err != nil {
		return nil, err
	}

	result := map[string]float64{}
	// use sell price for token in
	if price, ok := priceByAddress[tokenIn]; ok && price != nil && price.USDPrice.Sell != nil {
		result[tokenIn], _ = price.USDPrice.Sell.Float64()
	}

	// use buy price for token out and gas
	if price, ok := priceByAddress[tokenOut]; ok && price != nil && price.USDPrice.Buy != nil {
		result[tokenOut], _ = price.USDPrice.Buy.Float64()
	}

	// use buy price for other token out because we only need to find token out price
	for token, price := range priceByAddress {
		if price != nil && price.USDPrice.Buy != nil {
			result[token], _ = price.USDPrice.Buy.Float64()
		}
	}

	return result, nil
}

func (uc *BuildRouteUseCase) estimateGas(ctx context.Context, routeSummary valueobject.RouteSummary,
	command dto.BuildRouteCommand, encodedData string, isFaultyPoolTrackEnable bool) (uint64, float64, float64, error) {
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
	l1FeeUSDFloat, err := uc.calculateL1FeeUSD(ctx, routeSummary, encodedData)
	if err != nil {
		return 0, 0.0, 0, fmt.Errorf("failed to estimate L1 fee %s", err.Error())
	}

	return gas, gasUSD, l1FeeUSDFloat, nil
}

func (uc *BuildRouteUseCase) calculateL1FeeUSD(ctx context.Context, routeSummary valueobject.RouteSummary,
	encodedData string) (float64, error) {
	// Using the estimated L1 fee because we haven’t implemented Brotli compression for Arbitrum yet.
	if uc.config.ChainID == valueobject.ChainIDArbitrumOne {
		return routeSummary.L1FeeUSD, nil
	}

	l1Fee, err := uc.l1FeeCalculator.CalculateL1Fee(ctx, routeSummary, encodedData)
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
	routeSummary valueobject.RouteSummary, err error, slippage float64) {
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
		metrics.RecordEstimateGasWithSlippage(ctx, slippage, !isErrReturnAmountIsNotEnough(err))
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
