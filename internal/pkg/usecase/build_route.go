package usecase

import (
	"context"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/pkg/errors"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/business"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/clientdata"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/eth"
	timeutil "github.com/KyberNetwork/router-service/internal/pkg/utils/time"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

var OutputChangeNoChange = dto.OutputChange{
	Amount:  "0",
	Percent: 0,
	Level:   dto.OutputChangeLevelNormal,
}

type buildRouteUseCase struct {
	tokenRepository ITokenRepository
	priceRepository IPriceRepository

	rfqHandlerByPoolType map[string]pool.IPoolRFQ
	clientDataEncoder    IClientDataEncoder
	encoder              IEncoder
	nowFunc              func() time.Time

	config BuildRouteConfig
}

func NewBuildRouteUseCase(
	tokenRepository ITokenRepository,
	priceRepository IPriceRepository,
	rfqHandlerByPoolType map[string]pool.IPoolRFQ,
	clientDataEncoder IClientDataEncoder,
	encoder IEncoder,
	nowFunc func() time.Time,
	config BuildRouteConfig,
) *buildRouteUseCase {
	if nowFunc == nil {
		nowFunc = timeutil.NowFunc
	}

	return &buildRouteUseCase{
		tokenRepository:      tokenRepository,
		priceRepository:      priceRepository,
		rfqHandlerByPoolType: rfqHandlerByPoolType,
		clientDataEncoder:    clientDataEncoder,
		encoder:              encoder,
		nowFunc:              nowFunc,
		config:               config,
	}
}

func (uc *buildRouteUseCase) Handle(ctx context.Context, command dto.BuildRouteCommand) (*dto.BuildRouteResult, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "BuildRouteUseCase.Handle")
	defer span.Finish()

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

	// NOTE: currently we don't check the route (check if there is a better route or the route returns different amounts)
	// we return what client submitted
	return &dto.BuildRouteResult{
		AmountIn:    routeSummary.AmountIn.String(),
		AmountInUSD: strconv.FormatFloat(routeSummary.AmountInUSD, 'f', -1, 64),

		AmountOut:    routeSummary.AmountOut.String(),
		AmountOutUSD: strconv.FormatFloat(routeSummary.AmountOutUSD, 'f', -1, 64),

		Gas:    strconv.FormatInt(routeSummary.Gas, 10),
		GasUSD: strconv.FormatFloat(routeSummary.GasUSD, 'f', -1, 64),

		OutputChange: OutputChangeNoChange,

		Data:          encodedData,
		RouterAddress: uc.encoder.GetRouterAddress(),
	}, nil
}

func (uc *buildRouteUseCase) rfq(
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
				return routeSummary, errors.Wrapf(ErrRFQFailed, "rfq failed, swap data: %v, err: [%s]", swap, err.Error())
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
func (uc *buildRouteUseCase) updateRouteSummary(ctx context.Context, routeSummary valueobject.RouteSummary) (valueobject.RouteSummary, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "BuildRouteUseCase.updateRouteSummary")
	defer span.Finish()

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

func (uc *buildRouteUseCase) encode(ctx context.Context, command dto.BuildRouteCommand, routeSummary valueobject.RouteSummary) (string, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "BuildRouteUseCase.encode")
	defer span.Finish()

	clientData, err := uc.encodeClientData(ctx, command, routeSummary)
	if err != nil {
		return "", err
	}

	encodingData := types.NewEncodingDataBuilder().
		SetRoute(&routeSummary, uc.encoder.GetExecutorAddress(), uc.encoder.GetKyberLOAddress(), command.Recipient).
		SetDeadline(big.NewInt(command.Deadline)).
		SetSlippageTolerance(big.NewInt(command.SlippageTolerance)).
		SetClientData(clientData).
		SetPermit(command.Permit).
		GetData()

	return uc.encoder.Encode(encodingData)
}

// encodeClientData recalculates amountInUSD and amountOutUSD then perform encoding
func (uc *buildRouteUseCase) encodeClientData(ctx context.Context, command dto.BuildRouteCommand, routeSummary valueobject.RouteSummary) ([]byte, error) {
	flags, err := clientdata.ConvertFlagsToBitInteger(valueobject.Flags{
		TokenInMarketPriceAvailable:  routeSummary.TokenInMarketPriceAvailable,
		TokenOutMarketPriceAvailable: routeSummary.TokenOutMarketPriceAvailable,
	})
	if err != nil {
		return nil, err
	}

	return uc.clientDataEncoder.Encode(ctx, types.ClientData{
		Source:       command.Source,
		AmountInUSD:  strconv.FormatFloat(routeSummary.AmountInUSD, 'f', -1, 64),
		AmountOutUSD: strconv.FormatFloat(routeSummary.AmountOutUSD, 'f', -1, 64),
		Referral:     command.Referral,
		Flags:        flags,
	})
}

// getTokens returns tokenIn and tokenOut data
func (uc *buildRouteUseCase) getTokens(
	ctx context.Context,
	tokenInAddress string,
	tokenOutAddress string,
) (*entity.Token, *entity.Token, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "BuildRouteUseCase.getTokens")
	defer span.Finish()

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
func (uc *buildRouteUseCase) getPrices(
	ctx context.Context,
	tokenInAddress string,
	tokenOutAddress string,
) (*entity.Price, *entity.Price, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "BuildRouteUseCase.getPrices")
	defer span.Finish()

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
