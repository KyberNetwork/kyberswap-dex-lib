package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	t "github.com/KyberNetwork/kyberswap-error/pkg/transformers"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	redisv8 "github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/KyberNetwork/router-service/internal/pkg/config"
	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/core"
	"github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	internalErrors "github.com/KyberNetwork/router-service/internal/pkg/errors"
	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/model"
	"github.com/KyberNetwork/router-service/internal/pkg/rest"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase"
	usecasecore "github.com/KyberNetwork/router-service/internal/pkg/usecase/core"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/clientdata"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/findroute/spfa"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/validateroute"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
	"github.com/KyberNetwork/router-service/pkg/redis"
)

type IPoolFactory interface {
	NewPools(entityPools []entity.Pool) []pool.IPool
	NewPoolByAddress(entityPools []entity.Pool) map[string]pool.IPool
}

type IPoolRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]entity.Pool, error)
}

// ITokenRepository receives token addresses, fetch token data from datastore, decode them and return []entity.Token
type ITokenRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]entity.Token, error)
}

// IPriceRepository receives token addresses, fetch price data from datastore, decode them and return []entity.Price
type IPriceRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]entity.Price, error)
}

type IClientDataEncoder interface {
	Encode(ctx context.Context, clientData types.ClientData) ([]byte, error)
}

type IEncoder interface {
	Encode(data types.EncodingData) (string, error)
	GetExecutorAddress() string
	GetRouterAddress() string
	GetKyberLOAddress() string
}

type findRouteParams struct {
	query dto.GetRoutesQuery

	tokenIn  entity.Token
	tokenOut entity.Token
	gasToken entity.Token

	tokenInPrice  entity.Price
	tokenOutPrice entity.Price
	gasTokenPrice entity.Price

	gasPrice *big.Float

	sources        []string
	pools          []entity.Pool
	tokenByAddress map[string]entity.Token
	priceByAddress map[string]entity.Price
}

type RouteService struct {
	configLoader           *config.ConfigLoader
	router                 *gin.RouterGroup
	enableDexes            []string
	db                     *redis.Redis
	config                 *config.Common
	gasConfig              *config.Gas
	poolRepo               IPoolRepository
	tokenRepo              ITokenRepository
	priceRepo              IPriceRepository
	epsilon                float64
	cachePoints            []*config.CachePoint
	cacheRanges            []*config.CacheRange
	poolFactory            IPoolFactory
	validateRouteUseCase   *validateroute.ValidateRouteUseCase
	l2FeeCalculatorUseCase usecase.IL2FeeCalculatorUseCase
	blacklistedPools       []string
	featureFlags           valueobject.FeatureFlags
	clientDataEncoder      IClientDataEncoder
	encoder                IEncoder
	mu                     sync.RWMutex
}

type EncodingInput struct {
	SlippageTolerance int64
	ChargeFeeBy       string
	FeeReceiver       string
	IsInBps           bool
	FeeAmount         *big.Int
	Deadline          int64
	To                string
	ClientDataRaw     string
	Referral          string
	Permit            []byte
}

func NewRoute(
	configLoader *config.ConfigLoader,
	router *gin.RouterGroup,
	gasConfig *config.Gas,
	config *config.Common,
	poolRepo IPoolRepository,
	tokenRepo ITokenRepository,
	priceRepo IPriceRepository,
	enableDexes []string,
	epsilon float64,
	cachePoints []*config.CachePoint,
	cacheRanges []*config.CacheRange,
	poolFactory IPoolFactory,
	validateRouteUseCase *validateroute.ValidateRouteUseCase,
	l2FeeCalculatorUseCase usecase.IL2FeeCalculatorUseCase,
	clientDataEncoder IClientDataEncoder,
	encoder IEncoder,
	blacklistedPools []string,
	featureFlags valueobject.FeatureFlags,
) *RouteService {
	var ret = RouteService{
		configLoader:           configLoader,
		router:                 router,
		enableDexes:            enableDexes,
		config:                 config,
		gasConfig:              gasConfig,
		poolRepo:               poolRepo,
		tokenRepo:              tokenRepo,
		priceRepo:              priceRepo,
		epsilon:                epsilon,
		cachePoints:            cachePoints,
		cacheRanges:            cacheRanges,
		poolFactory:            poolFactory,
		validateRouteUseCase:   validateRouteUseCase,
		l2FeeCalculatorUseCase: l2FeeCalculatorUseCase,
		clientDataEncoder:      clientDataEncoder,
		encoder:                encoder,
		blacklistedPools:       blacklistedPools,
		featureFlags:           featureFlags,
	}
	ret.setupRoute()
	return &ret
}

func (t *RouteService) ApplyConfig(_ context.Context) error {
	cfg, err := t.configLoader.Get()
	if err != nil {
		return err
	}

	t.mu.Lock()
	t.enableDexes = cfg.EnableDexes
	t.blacklistedPools = cfg.BlacklistedPools
	t.featureFlags = cfg.FeatureFlags
	t.mu.Unlock()

	return nil
}

func (t *RouteService) IsEnableDex(dex string) bool {
	for _, v := range t.enableDexes {
		if v == dex {
			return true
		}
	}
	return false
}

func ValidateEncodingInput() gin.HandlerFunc {
	return func(c *gin.Context) {

		request := rest.FindEncodedRouteRequest{}
		if err := c.ShouldBindQuery(&request); err != nil {
			apiErr := t.RestTransformerInstance().ValidationErrToRestAPIErr(err)

			logger.Errorf("%v", apiErr)
			c.AbortWithStatusJSON(apiErr.HttpStatus, apiErr)
			return
		}

		if err := request.Validate(); err != nil {
			apiErr := t.RestTransformerInstance().DomainErrToRestAPIErr(err)

			logger.Errorf("%v", apiErr)
			c.AbortWithStatusJSON(apiErr.HttpStatus, apiErr)
			return
		}

		c.Next()
	}
}

func (t *RouteService) setupRoute() {
	t.router.GET(
		"route/encode", ValidateEncodingInput(), func(c *gin.Context) {
			span, _ := tracer.StartSpanFromContext(c.Request.Context(), "setupRoute")
			span.SetTag("http.uri", c.Request.URL.RequestURI())
			defer span.Finish()
			params := rest.FindEncodedRouteRequest{}
			if err := c.ShouldBindQuery(&params); err != nil {
				logger.Errorf("failed to bind params, err: %v", err)
				AbortWith400(c, err.Error())
				return
			}
			var dexes []string
			if params.Dexes == "" {
				dexes = t.enableDexes
			} else {
				dexes = make([]string, 0)
				parts := strings.Split(params.Dexes, ",")
				dexes = append(dexes, parts...)
			}

			deadline, err := strconv.ParseInt(params.Deadline, 10, 64)
			if err != nil {
				deadline = time.Now().Add(constant.DefaultDeadlineInMinute).Unix()
			} else {
				if deadline < time.Now().Unix() {
					apiErr := internalErrors.NewRestAPIErrDeadlineIsInThePast(nil, "deadline")

					logger.Errorf("%v", apiErr)
					c.AbortWithStatusJSON(apiErr.HttpStatus, apiErr)
					return
				}
			}

			slippageTolerance, err := strconv.ParseInt(params.SlippageTolerance, 10, 64)
			if err != nil {
				slippageTolerance = constant.DefaultSlippage
			}

			feeAmount, ok := new(big.Int).SetString(params.FeeAmount, 10)
			if !ok {
				feeAmount = constant.Zero
			}

			encodingInput := EncodingInput{
				SlippageTolerance: slippageTolerance,
				ChargeFeeBy:       params.ChargeFeeBy,
				FeeReceiver:       params.FeeReceiver,
				IsInBps:           utils.IsEnable(params.IsInBps),
				FeeAmount:         feeAmount,
				Deadline:          deadline,
				To:                params.To,
				ClientDataRaw:     params.ClientData,
				Referral:          params.Referral,
				Permit:            common.FromHex(params.Permit),
			}

			query, err := transformGetRouteEncodeParams(params)
			if err != nil {
				logger.Errorf("failed to transform params, err: %v", err)
				AbortWith400(c, err.Error())
				return
			}

			response, err := t.findRoute(
				c,
				query,
				strings.ToLower(params.TokenIn),
				strings.ToLower(params.TokenOut),
				params.AmountIn,
				utils.IsEnable(params.SaveGas),
				dexes,
				utils.IsEnable(params.GasInclude),
				params.GasPrice,
				&encodingInput,
				utils.IsEnable(params.Debug),
			)
			if err != nil {
				apiErr := internalErrors.NewRestAPIErrCouldNotFindRoute(err)

				logger.Errorf(
					"could not find route: %v, url: %v", err,
					fmt.Sprintf("%s?%s", c.Request.Host+c.Request.URL.Path, c.Request.URL.RawQuery),
				)
				logger.Errorf("%v", apiErr)
				c.AbortWithStatusJSON(apiErr.HttpStatus, apiErr)
				return
			}
			RespondWith(c, http.StatusOK, "success", response)
		},
	)
}

func transformGetRouteEncodeParams(params rest.FindEncodedRouteRequest) (dto.GetRoutesQuery, error) {
	amountIn, ok := new(big.Int).SetString(params.AmountIn, 10)
	if !ok {
		return dto.GetRoutesQuery{}, errors.Wrapf(
			ErrInvalidValue,
			"amountIn: [%s]",
			params.AmountIn,
		)
	}

	var gasPrice *big.Float
	if params.GasPrice != "" {
		gasPrice, ok = new(big.Float).SetString(params.GasPrice)
		if !ok {
			return dto.GetRoutesQuery{}, errors.Wrapf(
				ErrInvalidValue,
				"gasPrice: [%s]",
				params.GasPrice,
			)
		}
	}

	extraFee := valueobject.ZeroExtraFee
	if params.FeeAmount != "" {
		feeAmount, ok := new(big.Int).SetString(params.FeeAmount, 10)
		if !ok {
			return dto.GetRoutesQuery{}, errors.Wrapf(
				ErrInvalidValue,
				"feeAmount: [%s]",
				params.FeeAmount,
			)
		}

		extraFee = valueobject.ExtraFee{
			FeeAmount:   feeAmount,
			ChargeFeeBy: valueobject.ChargeFeeBy(params.ChargeFeeBy),
			IsInBps:     utils.IsEnable(params.IsInBps),
			FeeReceiver: params.FeeReceiver,
		}
	}

	return dto.GetRoutesQuery{
		TokenIn:    cleanUpParam(params.TokenIn),
		TokenOut:   cleanUpParam(params.TokenOut),
		AmountIn:   amountIn,
		SaveGas:    utils.IsEnable(params.SaveGas),
		GasInclude: utils.IsEnable(params.GasInclude),
		GasPrice:   gasPrice,
		ExtraFee:   extraFee,
	}, nil
}

func cleanUpParam(param string) string {
	return strings.ToLower(strings.TrimSpace(param))
}

func (t *RouteService) findRoute(
	ctx *gin.Context,
	query dto.GetRoutesQuery,
	tokenInAddress string,
	tokenOutAddress string,
	amountInStr string,
	saveGas bool,
	dexes []string,
	gasInclude bool,
	gasPriceStr string,
	encodingInput *EncodingInput,
	debug bool,
) (*rest.RouteResponse, error) {
	span, ctxWithSpan := tracer.StartSpanFromContext(ctx.Request.Context(), "findRoute")
	defer span.Finish()

	tokenInOrEther := tokenInAddress
	tokenOutOrEther := tokenOutAddress

	// Wrap tokenIn to WETH.
	if strings.EqualFold(tokenInAddress, constant.EtherAddress) {
		tokenInAddress = strings.ToLower(constant.WETH9[uint(t.config.ChainID)].Address.String())
	}
	// Wrap tokenOut to WETH.
	if strings.EqualFold(tokenOutAddress, constant.EtherAddress) {
		tokenOutAddress = strings.ToLower(constant.WETH9[uint(t.config.ChainID)].Address.String())
	}

	key := GetPairAddressKey(tokenInAddress, tokenOutAddress)
	cmders, err := t.db.Client.Pipelined(
		ctx, func(tx redisv8.Pipeliner) error {
			tx.HGet(ctx, t.db.FormatKey(ConfigKey), GasPriceKey)
			tx.ZRevRangeByScore(
				ctx, t.db.FormatKey(model.PairKey, key), &redisv8.ZRangeBy{
					Min:   "0",
					Max:   "+inf",
					Count: 100,
				},
			)
			tx.ZRevRangeByScore(
				ctx, t.db.FormatKey(model.PairKey, model.WhiteListKey), &redisv8.ZRangeBy{
					Min:   "0",
					Max:   "+inf",
					Count: 500,
				},
			)
			tx.ZRevRangeByScore(
				ctx, t.db.FormatKey(model.PairKey, model.WhiteListKey, tokenInAddress), &redisv8.ZRangeBy{
					Min:   "0",
					Max:   "+inf",
					Count: 200,
				},
			)
			tx.ZRevRangeByScore(
				ctx, t.db.FormatKey(model.PairKey, model.WhiteListKey, tokenOutAddress), &redisv8.ZRangeBy{
					Min:   "0",
					Max:   "+inf",
					Count: 200,
				},
			)
			tx.ZRevRangeByScore(
				ctx, t.db.FormatKey(model.AmplifiedTvlKey, model.PairKey, key), &redisv8.ZRangeBy{
					Min:   "0",
					Max:   "+inf",
					Count: 50,
				},
			)
			tx.ZRevRangeByScore(
				ctx, t.db.FormatKey(model.AmplifiedTvlKey, model.PairKey, model.WhiteListKey), &redisv8.ZRangeBy{
					Min:   "0",
					Max:   "+inf",
					Count: 200,
				},
			)
			tx.ZRevRangeByScore(
				ctx, t.db.FormatKey(model.AmplifiedTvlKey, model.PairKey, model.WhiteListKey, tokenInAddress),
				&redisv8.ZRangeBy{
					Min:   "0",
					Max:   "+inf",
					Count: 100,
				},
			)
			tx.ZRevRangeByScore(
				ctx, t.db.FormatKey(model.AmplifiedTvlKey, model.PairKey, model.WhiteListKey, tokenOutAddress),
				&redisv8.ZRangeBy{
					Min:   "0",
					Max:   "+inf",
					Count: 100,
				},
			)
			return nil
		},
	)
	if err != nil {
		return nil, err
	}
	if len(gasPriceStr) == 0 {
		gasPriceStr = cmders[0].(*redisv8.StringCmd).Val()
	}
	gasPrice, ok := new(big.Float).SetString(gasPriceStr)
	if !ok {
		return nil, errors.New("invalid gas price")
	}
	directPoolIds := cmders[1].(*redisv8.StringSliceCmd).Val()
	whitelistPoolIds := cmders[2].(*redisv8.StringSliceCmd).Val()
	tokenInPoolIds := cmders[3].(*redisv8.StringSliceCmd).Val()
	tokenOutPoolIds := cmders[4].(*redisv8.StringSliceCmd).Val()

	directPoolIdsByAmplifiedTvl := cmders[5].(*redisv8.StringSliceCmd).Val()
	whitelistPoolIdsByAmplifiedTvl := cmders[6].(*redisv8.StringSliceCmd).Val()
	tokenInPoolIdsByAmplifiedTvl := cmders[7].(*redisv8.StringSliceCmd).Val()
	tokenOutPoolIdsByAmplifiedTvl := cmders[8].(*redisv8.StringSliceCmd).Val()

	poolSet := sets.NewString(directPoolIds...)
	mergeIds := func(ids []string) {
		for _, id := range ids {
			poolSet.Insert(id)
		}
	}
	mergeIds(whitelistPoolIds)
	mergeIds(tokenInPoolIds)
	mergeIds(tokenOutPoolIds)
	mergeIds(directPoolIdsByAmplifiedTvl)
	mergeIds(whitelistPoolIdsByAmplifiedTvl)
	mergeIds(tokenInPoolIdsByAmplifiedTvl)
	mergeIds(tokenOutPoolIdsByAmplifiedTvl)

	for _, pool := range t.blacklistedPools {
		poolSet.Delete(strings.ToLower(pool))
	}

	poolIds := poolSet.List()

	rawPools, err := t.poolRepo.FindByAddresses(ctx, poolIds)
	if err != nil {
		return nil, err
	}
	tokenSet := sets.NewString(t.config.Address.GasToken, tokenOutAddress, tokenInAddress)
	dexSet := sets.NewString(dexes...)

	var pools []entity.Pool
	var curveBasePools = make(map[string]bool)
	for _, pool := range rawPools {
		if pool.Type == constant.PoolTypes.CurveBase {
			curveBasePools[pool.Address] = true
		}
	}

	for _, pool := range rawPools {
		if dexSet.Has(pool.Exchange) && (pool.HasReserves() || pool.HasAmplifiedTvl()) {
			pools = append(pools, pool)
			for _, token := range pool.Tokens {
				tokenSet.Insert(token.Address)
			}
			if pool.Type == constant.PoolTypes.CurveMeta {
				var staticExtra struct {
					BasePool string `json:"basePool"`
				}
				if err := json.Unmarshal([]byte(pool.StaticExtra), &staticExtra); err == nil {
					_, ok := curveBasePools[staticExtra.BasePool]
					if !ok {
						basePool, err := t.poolRepo.FindByAddresses(ctx, []string{staticExtra.BasePool})
						if err != nil {
							continue
						}
						if len(basePool) == 0 || basePool[0].Address == "" {
							continue
						}
						pools = append(pools, basePool[0])
						for _, token := range basePool[0].Tokens {
							tokenSet.Insert(token.Address)
						}
						curveBasePools[staticExtra.BasePool] = true
					}
				}
			}
		}
	}

	logger.Infof("pool length: %v", len(pools))

	tokenAddresses := tokenSet.List()

	tokenByAddress, err := t.getTokenByAddress(ctx, tokenAddresses)
	if err != nil {
		return nil, err
	}

	priceByAddress, err := t.getPriceByAddress(ctx, tokenAddresses)
	if err != nil {
		return nil, err
	}

	prices := ExtractPricesMapping(priceByAddress)

	gasTokenAddress := strings.ToLower(t.config.Address.GasToken)

	tokenInInfo := tokenByAddress[tokenInAddress]
	tokenOutInfo := tokenByAddress[tokenOutAddress]
	gasTokenPrice := prices[strings.ToLower(t.config.Address.GasToken)]

	extraFee := valueobject.ExtraFee{
		ChargeFeeBy: valueobject.ChargeFeeBy(encodingInput.ChargeFeeBy),
		FeeReceiver: encodingInput.FeeReceiver,
		IsInBps:     encodingInput.IsInBps,
		FeeAmount:   encodingInput.FeeAmount,
	}

	// Start from here...
	// Convert tokenIn amount to USD
	amountIn, ok := new(big.Int).SetString(amountInStr, 10)
	if !ok {
		return nil, errors.New("invalid amountIn")
	}

	totalSwapAmount := usecasecore.CalcAmountInAfterFee(amountIn, extraFee)

	totalSwapAmountUSD := utils.CalcTokenAmountUsd(totalSwapAmount, tokenInInfo.Decimals, prices[tokenInAddress])
	if totalSwapAmountUSD > constant.MaxAmountInUSD {
		return nil, errors.New("invalid amountIn")
	}

	totalSwapAmountUSDRound := int(math.Round(totalSwapAmountUSD))

	var cachedRouteKey string
	var bestRoute = &core.Route{}
	var response *rest.RouteResponse

	pairRouteCacheKey := getPairRouteCacheKey(tokenInAddress, tokenOutAddress)

	cachedRouteKey, err = t.compoundCachedRouteKey(
		totalSwapAmount.String(), totalSwapAmountUSDRound, tokenInInfo.Decimals, pairRouteCacheKey, saveGas, dexes, gasInclude,
	)
	if err != nil {
		return nil, fmt.Errorf("compounding cachedRouteKey failed due error: %w", err)
	}

	cachedRouteBytes, ttl, err := t.lookupCachedRoute(ctx.Request.Context(), cachedRouteKey)
	if err != nil {
		// it's okay to continue with a warning
		logger.Warnf(
			"could not find cachedRoute. Error: %v, url: %v, tokenIn: %v, tokenOut: %v, amountIn: %v, amountInUSDRound: %v",
			err, fmt.Sprintf("%s?%s", ctx.Request.Host+ctx.Request.URL.Path, ctx.Request.URL.RawQuery), tokenInAddress,
			tokenOutAddress, amountIn, totalSwapAmountUSDRound,
		)
	}

	// If best route of the data point (amountInUSD) has been cached already, and the cache is still valid,
	// let's consider using it instead of finding best route
	isValidCache := len(cachedRouteBytes) != 0 && ttl > 0
	isUseCache := false
	if isValidCache {
		var cachedRoute core.CachedRoute
		if err := json.Unmarshal(cachedRouteBytes, &cachedRoute); err != nil {
			logger.Warnf("unmarshal cached route failed, err: %v", err)
		}

		if err = cachedRoute.RedistributeInputAmount(totalSwapAmount, tokenInInfo.Decimals, prices[tokenInAddress]); err != nil {
			logger.Warnf("redistribute input amount failed, err: %v", err)
		}

		bestRoute, err = cachedRoute.ToRoute(
			t.poolFactory.NewPools(pools),
			t.poolFactory.NewPools(pools),
		)
		if err != nil {
			logger.Warnf("convert cachedRoute to route failed, err: %v", err)
		}

		response, err = bestRoute.Finalize(span)
		if err != nil {
			logger.Warnf("finalize bestRoute failed, err: %v", err)
		}

		amountOutUSD := utils.CalcTokenAmountUsd(
			utils.NewBig10(response.OutputAmount), tokenOutInfo.Decimals, prices[tokenOutAddress],
		)

		if !utils.AlmostEqual(totalSwapAmountUSD, 0) {
			priceImpact := (totalSwapAmountUSD - amountOutUSD) / totalSwapAmountUSD
			logger.Infof("priceImpact: %f", priceImpact)
			if priceImpact < t.epsilon {
				isUseCache = true
			}
		}
		logger.Infof("isUseCache: %s ", isUseCache)
	}

	// If there's no cache or cache best route giving a result that is not good enough,
	// let's find a new best route
	if !isUseCache {
		if t.featureFlags.UseOptimizedSPFA {
			params := &findRouteParams{
				query: query,

				tokenIn:  tokenByAddress[tokenInAddress],
				tokenOut: tokenByAddress[tokenOutAddress],
				gasToken: tokenByAddress[gasTokenAddress],

				tokenInPrice:  priceByAddress[tokenInAddress],
				tokenOutPrice: priceByAddress[tokenOutAddress],
				gasTokenPrice: priceByAddress[gasTokenAddress],

				gasPrice: gasPrice,

				sources:        dexes,
				pools:          pools,
				tokenByAddress: tokenByAddress,
				priceByAddress: priceByAddress,
			}

			bestRoute, err = t.findRouteWithSPFA(ctxWithSpan, params, amountIn)
			if err != nil {
				return nil, err
			}

			// Set the OriginalPools here because the new SPFA algo does not return OriginalPools in object core.Route
			bestRoute.OriginalPools = t.poolFactory.NewPools(pools)
		} else {
			f := func(isSaveGas bool) (*core.Route, error) {
				whiteListPools := t.poolFactory.NewPools(pools)
				var originalPools = t.poolFactory.NewPools(pools)
				return core.BestRouteExactIn(
					ctxWithSpan,
					whiteListPools,
					originalPools,
					tokenByAddress,
					prices,
					tokenInAddress,
					tokenOutAddress,
					totalSwapAmount,
					core.BestRouteOption{
						SaveGas:    isSaveGas,
						MaxHops:    3,
						MaxPools:   5,
						MaxPaths:   5,
						MinPartUsd: 500,
						Gas: core.GasOption{
							GasFeeInclude: gasInclude,
							Price:         gasPrice,
							TokenPrice:    gasTokenPrice,
						},
					},
				)
			}
			bestRoute, err = f(saveGas)
			if err != nil {
				return nil, err
			}
			if !saveGas {
				singleRoute, err := f(true)
				if err != nil {
					return nil, err
				}
				// If the single route is better than best route or there is no path in the best route
				if singleRoute.CompareTo(bestRoute, gasInclude) > 0 || len(bestRoute.Paths) == 0 {
					bestRoute = singleRoute
				}
			}
		}

		response, err = bestRoute.Finalize(span)
		if err != nil {
			return nil, err
		}
	}

	// Write best route to cache
	if !isValidCache || !isUseCache {
		err := t.writeBestRouteToCache(ctx, cachedRouteKey, bestRoute, tokenInInfo, totalSwapAmount.String(), totalSwapAmountUSDRound)
		if err != nil {
			// just log a warning and move on
			logger.Warnf("Can not write best route to cache: %v", err)
		}
	}

	err = t.validateRouteUseCase.ValidateRouteResult(*bestRoute)
	if err != nil {
		logger.Errorf("failed to validate route result, err: %v", err)
	}

	var tokens = make(map[string]rest.TokenInfo)
	for _, swap := range response.Swaps {
		for _, item := range swap {
			if _, ok := tokens[item.TokenIn]; !ok {
				tokenInfo := tokenByAddress[item.TokenIn]
				tokens[item.TokenIn] = rest.TokenInfo{
					Address:  tokenInfo.Address,
					Symbol:   tokenInfo.Symbol,
					Name:     tokenInfo.Name,
					Decimals: tokenInfo.Decimals,
					Price:    prices[tokenInfo.Address],
				}
			}
			if _, ok := tokens[item.TokenOut]; !ok {
				tokenInfo := tokenByAddress[item.TokenOut]
				tokens[item.TokenOut] = rest.TokenInfo{
					Address:  tokenInfo.Address,
					Symbol:   tokenInfo.Symbol,
					Name:     tokenInfo.Name,
					Decimals: tokenInfo.Decimals,
					Price:    prices[tokenInfo.Address],
				}
			}
		}
	}
	response.Tokens = tokens

	routeSummary, err := t.summarizeRoute(
		bestRoute,
		tokenInOrEther,
		tokenOutOrEther,
		amountIn,
		tokenInInfo,
		tokenOutInfo,
		priceByAddress[tokenInAddress],
		priceByAddress[tokenOutAddress],
		priceByAddress[strings.ToLower(t.config.Address.GasToken)],
		gasPrice,
		extraFee,
		pools,
	)
	if err != nil {
		return nil, err
	}

	if len(response.Swaps) > 0 && encodingInput != nil {
		if extraFee.FeeAmount.Cmp(constant.Zero) > 0 && routeSummary.AmountOut.Cmp(constant.Zero) <= 0 {
			return nil, errors.New("feeAmount should not be larger than amountOut")
		}

		encodedData, err := t.encode(
			ctx,
			routeSummary,
			encodingInput,
		)
		if err != nil {
			return nil, err
		}

		response.EncodedSwapData = encodedData
		response.RouterAddress = t.encoder.GetRouterAddress()

	}

	response.InputAmount = routeSummary.AmountIn.String()
	response.OutputAmount = routeSummary.AmountOut.String()
	response.GasPriceGwei = routeSummary.GasPrice.String()

	// Fee to publish tx to L1, applied only to L2 chains
	l1PublishTxFee, err := t.l2FeeCalculatorUseCase.GetL1Fee(ctx, response.EncodedSwapData)
	if err != nil {
		l1PublishTxFee = constant.Zero
	}
	l1PublishTxFeeUsd, _ := usecasecore.CalcL1FeeUSD(l1PublishTxFee, gasTokenPrice).Float64()

	// Fee to execute tx, applied to all chains
	executeTxFeeUsd, _ := usecasecore.CalcGasUSD(gasPrice, routeSummary.Gas, gasTokenPrice).Float64()

	// We keep totalGas = L2 gas only, because it's very confused when combining L1 publish tx gas here
	// More info: https://medium.com/offchainlabs/understanding-arbitrum-2-dimensional-fees-fd1d582596c9
	response.TotalGas = routeSummary.Gas
	response.GasUsd = executeTxFeeUsd + l1PublishTxFeeUsd
	response.AmountInUsd = routeSummary.AmountInUSD
	response.AmountOutUsd = routeSummary.AmountOutUSD
	response.ReceivedUsd = routeSummary.AmountOutUSD - response.GasUsd

	if debug {
		response.Debug = map[string]interface{}{
			"featureFlags":     t.featureFlags,
			"pools":            pools,
			"whitelistPoolIds": whitelistPoolIds,
			"tokenOutPoolIds":  tokenOutPoolIds,
			"tokenInPoolIds":   tokenInPoolIds,
		}
	}

	return response, nil
}

func (t *RouteService) getTokenByAddress(
	ctx context.Context,
	tokenAddresses []string,
) (map[string]entity.Token, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "RouteService.getTokenByAddress")
	defer span.Finish()

	tokens, err := t.tokenRepo.FindByAddresses(ctx, tokenAddresses)
	if err != nil {
		return nil, err
	}

	tokenByAddress := make(map[string]entity.Token, len(tokens))
	for _, token := range tokens {
		tokenByAddress[token.Address] = token
	}

	return tokenByAddress, nil
}

// getPriceByAddress fetch price data and return a map from token address to price in USD of the token
func (t *RouteService) getPriceByAddress(
	ctx context.Context,
	tokenAddresses []string,
) (map[string]entity.Price, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "RouteService.getPriceByAddress")
	defer span.Finish()

	prices, err := t.priceRepo.FindByAddresses(ctx, tokenAddresses)
	if err != nil {
		return nil, err
	}

	priceByAddress := make(map[string]entity.Price, len(prices))
	for _, price := range prices {
		priceByAddress[price.Address] = price
	}

	return priceByAddress, nil
}

// findRouteWithSPFA performs SPFA to find the best route
// if saveGas, it finds and returns the best single path route
// otherwise, it finds the best single path route and the best multiple path route and returns the better one
func (t *RouteService) findRouteWithSPFA(
	ctx context.Context,
	params *findRouteParams,
	amountIn *big.Int,
) (*core.Route, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "RouteService.findRouteWithSPFA")
	defer span.Finish()

	preferredPriceUSDByAddress := ExtractPricesMapping(params.priceByAddress)

	gasTokenPriceUSD, _ := params.gasTokenPrice.GetPreferredPrice()

	input := findroute.Input{
		TokenInAddress:   params.tokenIn.Address,
		TokenOutAddress:  params.tokenOut.Address,
		AmountIn:         amountIn,
		GasPrice:         params.gasPrice,
		GasTokenPriceUSD: gasTokenPriceUSD,
		SaveGas:          params.query.SaveGas,
		GasInclude:       params.query.GasInclude,
	}

	var finder findroute.IFinder = spfa.NewDefaultSPFAFinder()

	bestRoutes, err := finder.Find(
		ctx,
		input,
		findroute.FinderData{
			PoolByAddress:     t.poolFactory.NewPoolByAddress(params.pools),
			TokenByAddress:    params.tokenByAddress,
			PriceUSDByAddress: preferredPriceUSDByAddress,
		})
	if err != nil {
		return nil, err
	}

	return extractBestRoute(bestRoutes), nil
}

func extractBestRoute(routes []*core.Route) *core.Route {
	if len(routes) == 0 {
		return nil
	}

	return routes[0]
}

// getPairRouteCacheKey returns the pair route cache key of a swap token pair
// The order of token is IMPORTANT
func getPairRouteCacheKey(a, b string) string {
	return a + "-" + b
}

func (t *RouteService) findCacheTTL(amountIn string, decimals uint8, amountInUSD int) (*time.Duration, error) {
	// Default cache TTL is 10s
	var ttl time.Duration = 10 * 1e9

	amount, err := utils.DivDecimals(amountIn, decimals)
	if err != nil {
		return nil, err
	}

	for _, point := range t.cachePoints {
		if amount.Cmp(new(big.Float).SetInt64(int64(point.Amount))) == 0 {
			ttl = time.Duration(point.TTL) * 1e9
			return &ttl, nil
		}
	}

	for _, amountRange := range t.cacheRanges {
		if amountRange.FromUSD <= amountInUSD && amountInUSD <= amountRange.ToUSD {
			ttl = time.Duration(amountRange.TTL) * 1e9
			return &ttl, nil
		}
	}

	return &ttl, nil
}

func (t *RouteService) isCachePoint(amountIn string, decimals uint8) (bool, error) {
	amount, err := utils.DivDecimals(amountIn, decimals)
	if err != nil {
		return false, err
	}
	for _, point := range t.cachePoints {
		if amount.Cmp(new(big.Float).SetInt64(int64(point.Amount))) == 0 {
			return true, nil
		}
	}
	return false, nil
}

// Coumpound cache route key: model.Route:pairKey:saveGas:cacheMode:amountInUSD:dexes
// dexes=dexId1-dexId2-dexId3...
func (t *RouteService) compoundCachedRouteKey(
	amountIn string, amountInUSDRound int, decimals uint8, pairRouteCacheKey string, saveGas bool, dexes []string, gasInclude bool,
) (string, error) {
	var cachedRouteKey string
	var isCachePoint bool

	isCachePoint, err := t.isCachePoint(amountIn, decimals)
	if err != nil {
		return cachedRouteKey, fmt.Errorf("invalid amountIn %s, %w", amountIn, err)
	}

	if isCachePoint {
		cachedRouteKey = t.db.FormatKey(
			model.Route, pairRouteCacheKey, saveGas, model.CachePoint, amountIn, strings.Join(dexes, "-"), gasInclude,
		)
	} else {
		cachedRouteKey = t.db.FormatKey(
			model.Route, pairRouteCacheKey, saveGas, model.CacheRange, fmt.Sprint(amountInUSDRound), strings.Join(dexes, "-"), gasInclude,
		)
	}

	return cachedRouteKey, nil
}

// Get cache best route for this data point
func (t *RouteService) lookupCachedRoute(ctx context.Context, cachedRouteKey string) ([]byte, time.Duration, error) {
	// Second try, lookup cloud redis
	cmders, err := t.db.Client.Pipelined(
		ctx, func(tx redisv8.Pipeliner) error {
			tx.Get(ctx, cachedRouteKey)
			tx.TTL(ctx, cachedRouteKey)
			return nil
		},
	)
	if err == nil {
		cachedRoute, err1 := cmders[0].(*redisv8.StringCmd).Bytes()
		ttl, err2 := cmders[1].(*redisv8.DurationCmd).Result()
		if err1 == nil && err1 != redisv8.Nil && err2 == nil && err2 != redisv8.Nil {
			return cachedRoute, ttl, nil
		}
	}
	return []byte{}, 0, err
}

func (t *RouteService) writeBestRouteToCache(
	ctx context.Context,
	cachedRouteKey string,
	route *core.Route,
	tokenInInfo entity.Token,
	amountIn string,
	amountInUSDRound int,
) error {
	span, ctx := tracer.StartSpanFromContext(ctx, "writeBestRouteToCache")
	defer span.Finish()
	if route == nil || len(route.Input.Amount.Bits()) == 0 {
		return nil
	}

	cachedRoute, err := route.ToCachedRoute()
	if err != nil {
		return err
	}

	routeBytes, err := json.Marshal(cachedRoute)
	if err != nil {
		return err
	}

	ttl, err := t.findCacheTTL(amountIn, tokenInInfo.Decimals, amountInUSDRound)
	if err != nil {
		return err
	}

	t.db.Client.Set(ctx, cachedRouteKey, routeBytes, *ttl)

	return nil
}

func (t *RouteService) encode(
	ctx context.Context,
	routeSummary *valueobject.RouteSummary,
	encodingInput *EncodingInput,
) (string, error) {
	to := encodingInput.To
	slippageTolerance := big.NewInt(encodingInput.SlippageTolerance)
	deadline := big.NewInt(encodingInput.Deadline)
	clientRawData := encodingInput.ClientDataRaw
	referral := encodingInput.Referral
	permit := encodingInput.Permit

	flags, err := clientdata.ConvertFlagsToBitInteger(valueobject.Flags{
		TokenInMarketPriceAvailable:  routeSummary.TokenInMarketPriceAvailable,
		TokenOutMarketPriceAvailable: routeSummary.TokenOutMarketPriceAvailable,
	})
	if err != nil {
		return "", err
	}

	var rawClientData struct {
		Source string `json:"source"`
	}
	_ = json.Unmarshal([]byte(clientRawData), &rawClientData)

	clientData, err := t.clientDataEncoder.Encode(ctx, types.ClientData{
		Source:       rawClientData.Source,
		AmountInUSD:  strconv.FormatFloat(routeSummary.AmountInUSD, 'f', -1, 64),
		AmountOutUSD: strconv.FormatFloat(routeSummary.AmountOutUSD, 'f', -1, 64),
		Referral:     referral,
		Flags:        flags,
	})
	if err != nil {
		return "", err
	}

	encodingData := types.NewEncodingDataBuilder().
		SetRoute(routeSummary, t.encoder.GetExecutorAddress(), t.encoder.GetKyberLOAddress(), to).
		SetDeadline(deadline).
		SetSlippageTolerance(slippageTolerance).
		SetClientData(clientData).
		SetPermit(permit).
		GetData()

	return t.encoder.Encode(encodingData)
}

func (t *RouteService) summarizeRoute(
	route *core.Route,
	tokenInOrEther string,
	tokenOutOrEther string,
	amountIn *big.Int,
	tokenIn entity.Token,
	tokenOut entity.Token,
	tokenInPrice entity.Price,
	tokenOutPrice entity.Price,
	gasTokenPrice entity.Price,
	gasPrice *big.Float,
	extraFee valueobject.ExtraFee,
	pools []entity.Pool,
) (*valueobject.RouteSummary, error) {
	iPools := t.poolFactory.NewPools(pools)

	poolByAddress := make(map[string]pool.IPool, len(pools))
	for _, iPool := range iPools {
		poolByAddress[iPool.GetAddress()] = iPool
	}

	var (
		amountOut = constant.Zero
	)

	var gas int64
	if t.gasConfig != nil {
		gas = t.gasConfig.Default
	} else {
		gas = pool.GasDefault
	}

	summarizedRoute := make([][]valueobject.Swap, 0, len(route.Paths))
	for _, path := range route.Paths {
		gas += path.TotalGas

		summarizedPath := make([]valueobject.Swap, 0, len(path.Pools))
		swapIn := path.Input
		for swapIdx, swapPool := range path.Pools {
			freshPool, ok := poolByAddress[swapPool.GetAddress()]
			if !ok {
				logger.WithFields(logger.Fields{
					"pool.Address": swapPool.GetAddress(),
				}).Error("[getRoutes.summarizeRoute] pool not found")

				return nil, ErrPoolNotFound
			}

			calcAmountOutResult, err := freshPool.CalcAmountOut(swapIn, path.Tokens[swapIdx+1].Address)
			if err != nil {
				logger.WithFields(logger.Fields{
					"tokenIn":      swapIn.Token,
					"amountIn":     swapIn.Amount.String(),
					"pool.Address": swapPool.GetAddress(),
				}).Error("[getRoutes.summarizeRoute] invalid swap")
				return nil, ErrInvalidSwap
			}
			swapOut, swapFee := calcAmountOutResult.TokenAmountOut, calcAmountOutResult.Fee
			if swapOut == nil || swapOut.Amount == nil || swapOut.Amount.Cmp(constant.Zero) <= 0 {
				logger.WithFields(logger.Fields{
					"tokenIn":      swapIn.Token,
					"amountIn":     swapIn.Amount.String(),
					"pool.Address": swapPool.GetAddress(),
				}).Error("[getRoutes.summarizeRoute] invalid swap")

				return nil, ErrInvalidSwap
			}

			swap := valueobject.Swap{
				Pool:              freshPool.GetAddress(),
				TokenIn:           swapIn.Token,
				TokenOut:          swapOut.Token,
				SwapAmount:        swapIn.Amount,
				AmountOut:         swapOut.Amount,
				LimitReturnAmount: constant.Zero,
				Exchange:          valueobject.Exchange(freshPool.GetExchange()),
				PoolLength:        len(freshPool.GetTokens()),
				PoolType:          freshPool.GetType(),
				PoolExtra:         freshPool.GetMetaInfo(swapIn.Token, swapOut.Token),
				Extra:             calcAmountOutResult.SwapInfo,
			}

			summarizedPath = append(summarizedPath, swap)

			updateBalanceParams := pool.UpdateBalanceParams{
				TokenAmountIn:  swapIn,
				TokenAmountOut: *swapOut,
				Fee:            *swapFee,
				SwapInfo:       calcAmountOutResult.SwapInfo,
			}
			freshPool.UpdateBalance(updateBalanceParams)
			swapIn = *swapOut

			metrics.IncrDexHitRate(string(swap.Exchange))
			metrics.IncrPoolTypeHitRate(swap.PoolType)
		}

		amountOut = new(big.Int).Add(amountOut, swapIn.Amount)
		summarizedRoute = append(summarizedRoute, summarizedPath)
	}

	// amountOut is actual amount of token to be received
	// in case charge fee by currencyIn: amountIn = amountIn - extraFeeAmount
	// in case charge fee by currencyOut: amountOut = amountOut - extraFeeAmount will be included in summarizeRoute
	amountOut = usecasecore.CalcAmountOutAfterFee(amountOut, extraFee)

	tokenInPriceVal, tokenInMarketPriceAvailable := tokenInPrice.GetPreferredPrice()
	tokenOutPriceVal, tokenOutMarketPriceAvailable := tokenOutPrice.GetPreferredPrice()
	gasTokenPriceVal, _ := gasTokenPrice.GetPreferredPrice()

	metrics.IncrRequestPairCount(tokenIn.Address, tokenOut.Address, amountIn.String())

	return &valueobject.RouteSummary{
		TokenIn:                      tokenInOrEther,
		AmountIn:                     amountIn,
		AmountInUSD:                  utils.CalcTokenAmountUsd(amountIn, tokenIn.Decimals, tokenInPriceVal),
		TokenInMarketPriceAvailable:  tokenInMarketPriceAvailable,
		TokenOut:                     tokenOutOrEther,
		AmountOut:                    amountOut,
		AmountOutUSD:                 utils.CalcTokenAmountUsd(amountOut, tokenOut.Decimals, tokenOutPriceVal),
		TokenOutMarketPriceAvailable: tokenOutMarketPriceAvailable,
		Gas:                          gas,
		GasPrice:                     new(big.Float).Quo(gasPrice, constant.TenPowDecimals(9)), // convert from wei to gwei
		GasUSD:                       utils.CalcGasUsd(gasPrice, gas, gasTokenPriceVal),
		ExtraFee:                     extraFee,
		Route:                        summarizedRoute,
	}, nil
}
