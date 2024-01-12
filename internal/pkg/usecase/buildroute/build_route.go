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
	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/business"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/clientid"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/eth"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/requestid"
	timeutil "github.com/KyberNetwork/router-service/internal/pkg/utils/time"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
	"github.com/pkg/errors"
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
	gasEstimator              IGasEstimator

	rfqHandlerByPoolType map[string]pool.IPoolRFQ
	clientDataEncoder    IClientDataEncoder
	encodeBuilder        encode.IEncodeBuilder
	nowFunc              func() time.Time

	config Config

	mu sync.RWMutex
}

func NewBuildRouteUseCase(
	tokenRepository ITokenRepository,
	priceRepository IPriceRepository,
	poolRepository IPoolRepository,
	executorBalanceRepository IExecutorBalanceRepository,
	gasEstimator IGasEstimator,
	rfqHandlerByPoolType map[string]pool.IPoolRFQ,
	clientDataEncoder IClientDataEncoder,
	encodeBuilder encode.IEncodeBuilder,
	nowFunc func() time.Time,
	config Config,
) *BuildRouteUseCase {
	if nowFunc == nil {
		nowFunc = timeutil.NowFunc
	}

	return &BuildRouteUseCase{
		tokenRepository:           tokenRepository,
		priceRepository:           priceRepository,
		poolRepository:            poolRepository,
		executorBalanceRepository: executorBalanceRepository,
		gasEstimator:              gasEstimator,
		rfqHandlerByPoolType:      rfqHandlerByPoolType,
		clientDataEncoder:         clientDataEncoder,
		encodeBuilder:             encodeBuilder,
		nowFunc:                   nowFunc,
		config:                    config,
	}
}

func (uc *BuildRouteUseCase) Handle(ctx context.Context, command dto.BuildRouteCommand) (*dto.BuildRouteResult, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "BuildRouteUseCase.Handle")
	defer span.End()

	routeSummary, err := uc.rfq(ctx, command.Recipient, command.RouteSummary)
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

	// track total count of pool for every build route request, using for calculating fault density
	go func() {
		uc.trackFaultyPoolsKeyTotalCount(context.Background(), routeSummary, uc.nowFunc().UTC())
	}()

	// estimate gas price for a transaction
	estimatedGas, gasInUSD, err := uc.estimateGas(ctx, command, encodedData)
	if err != nil {
		return nil, err
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
	recipient string,
	routeSummary valueobject.RouteSummary,
) (valueobject.RouteSummary, error) {
	for pathIdx, path := range routeSummary.Route {
		for swapIdx, swap := range path {
			rfqHandler, found := uc.rfqHandlerByPoolType[swap.PoolType]
			if !found {
				// This pool type does not have RFQ handler
				// It means that this swap does not need to be processed via RFQ
				logger.Debugf("no RFQ handler for pool type: %v", swap.PoolType)
				continue
			}

			result, err := rfqHandler.RFQ(ctx, recipient, swap.Extra)
			if err != nil {
				return routeSummary, errors.Wrapf(err, "rfq failed, swap data: %v", swap)
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
		return routeSummary, errors.Wrapf(ErrTokenNotFound, "tokenIn: [%s]", tokenInAddress)
	}

	if tokenOut == nil {
		return routeSummary, errors.Wrapf(ErrTokenNotFound, "tokenOut: [%s]", tokenOutAddress)
	}

	tokenInPrice, tokenOutPrice, err := uc.getPrices(ctx, tokenInAddress, tokenOutAddress)
	if err != nil {
		return routeSummary, err
	}

	var (
		tokenInPriceUSD             float64
		tokenInMarketPriceAvailable bool
	)
	if tokenInPrice != nil {
		tokenInPriceUSD, tokenInMarketPriceAvailable = tokenInPrice.GetPreferredPrice()
	}

	var (
		tokenOutPriceUSD             float64
		tokenOutMarketPriceAvailable bool
	)
	if tokenOutPrice != nil {
		tokenOutPriceUSD, tokenOutMarketPriceAvailable = tokenOutPrice.GetPreferredPrice()

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
func (uc *BuildRouteUseCase) getPrices(
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

func (uc *BuildRouteUseCase) estimateGas(ctx context.Context, command dto.BuildRouteCommand, encodedData string) (uint64, float64, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "BuildRouteUseCase.estimateGas")
	defer span.End()

	value := constant.Zero
	if eth.IsEther(command.RouteSummary.TokenIn) {
		value = command.RouteSummary.AmountIn
	}
	tx := UnsignedTransaction{
		command.Sender,
		encodedData,
		value,
		nil,
	}

	if uc.config.FeatureFlags.IsGasEstimatorEnabled {
		if command.EnableGasEstimation {
			if utils.IsEmptyString(command.Sender) {
				return 0, 0.0, ErrSenderEmptyWhenEnableEstimateGas
			}

			gas, gasUSD, err := uc.gasEstimator.Execute(ctx, tx)
			uc.sendEstimateGasLogsAndMetrics(ctx, command.RouteSummary, err, command.SlippageTolerance)
			if err != nil {
				return 0, 0.0, errors.Wrapf(ErrEstimateGasFailed, "Estimate gas failed due to %s", err.Error())
			}

			return gas, gasUSD, nil
		} else {
			if !utils.IsEmptyString(command.Sender) {
				go func() {
					_, err := uc.gasEstimator.EstimateGas(context.Background(), tx)
					uc.sendEstimateGasLogsAndMetrics(ctx, command.RouteSummary, err, command.SlippageTolerance)
				}()
			}
		}

	}
	return uint64(command.RouteSummary.Gas), command.RouteSummary.GasUSD, nil

}

func (uc *BuildRouteUseCase) sendEstimateGasLogsAndMetrics(ctx context.Context,
	routeSummary valueobject.RouteSummary, err error, slippage int64) {
	clientId := clientid.GetClientIDFromCtx(ctx)
	poolTags := make([]string, 0)

	for _, path := range routeSummary.Route {
		for _, swap := range path {
			metrics.IncrEstimateGas(err == nil, string(swap.Exchange), clientId)
			poolTags = append(poolTags, fmt.Sprintf("%s:%s", swap.Exchange, swap.Pool))
		}
	}

	if err != nil {
		logger.WithFields(logger.Fields{
			"requestId": requestid.GetRequestIDFromCtx(ctx),
			"clientId":  clientId,
			"pool":      strings.Join(poolTags, ","),
		}).Infof("EstimateGas failed error %s", err)

		if strings.Contains(err.Error(), ErrReturnAmountIsNotEnough.Error()) {
			// send failed metrics with slippage when error is Return amount is not enough
			metrics.HistogramEstimateGasWithSlippage(float64(slippage), false)
		} else {
			// send success metrics with slippage
			metrics.HistogramEstimateGasWithSlippage(float64(slippage), true)
		}
	}
}

func (uc *BuildRouteUseCase) trackFaultyPoolsKeyTotalCount(ctx context.Context, routeSummary valueobject.RouteSummary, currentTime time.Time) {
	windowSize := int(uc.config.FaultyPoolsConfig.WindowSize.Minutes())
	currentWindow := ((currentTime.Minute() / windowSize) + 1) * windowSize
	currentWindowKey := fmt.Sprintf("%02d:%02d:%02d", currentTime.Day(), currentTime.Hour(), currentWindow)

	// tracking total count of pool address related to build route API
	counter := make(map[string]int64)
	for _, path := range routeSummary.Route {
		for _, swap := range path {
			counter[fmt.Sprintf("%s:%s", swap.Pool, currentWindowKey)]++
		}
	}
	_, errors := uc.poolRepository.IncreasePoolsTotalCount(ctx, counter, 2*uc.config.FaultyPoolsConfig.WindowSize)
	for _, err := range errors {
		logger.Errorf("[TrackFaultyPoolsUseCase] HIncreaseByMultiple err: %v", err)
	}
}
