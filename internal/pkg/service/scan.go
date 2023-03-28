package service

import (
	"bytes"
	"context"
	"encoding/json"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	"github.com/KyberNetwork/router-service/internal/pkg/config"
	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/core/synthetix"
	"github.com/KyberNetwork/router-service/internal/pkg/core/univ3"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/repository"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/envvar"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/eth"
	"github.com/KyberNetwork/router-service/pkg/logger"
	"github.com/KyberNetwork/router-service/pkg/util/env"
)

type TokenSCField string

const (
	TokenSCFieldDecimals = "decimals"
	TokenSCFieldName     = "name"
	TokenSCFieldSymbol   = "symbol"
)

var TokenSCFieldsToRead = []TokenSCField{
	TokenSCFieldDecimals,
	TokenSCFieldName,
	TokenSCFieldSymbol,
}

const (
	TokenSCMethodDecimals = "decimals"
	TokenSCMethodName     = "name"
	TokenSCMethodSymbol   = "symbol"
)

const tokenSourceForAggregator = "dmm-aggregator"

const limitOrderPoolReserveUSD = 1000000000

// TokenState represents data of token smart contract
type TokenState struct {
	Address  string
	Name     string
	Decimals uint8
	Symbol   string
}

type ScanService struct {
	rpcRepo          repository.IRPCRepository
	poolRepo         repository.IPoolRepository
	tokenRepo        repository.ITokenRepository
	priceRepo        repository.IPriceRepository
	routeRepo        repository.IRouteRepository
	scannerStateRepo repository.IScannerStateRepository
	scanConfigSvc    *ScanConfigService
	tokenCatalogRepo ITokenCatalogRepository
}

func NewScanService(
	rpcRepo repository.IRPCRepository,
	poolRepo repository.IPoolRepository,
	tokenRepo repository.ITokenRepository,
	priceRepo repository.IPriceRepository,
	routeRepo repository.IRouteRepository,
	scannerStateRepo repository.IScannerStateRepository,
	scanConfigSvc *ScanConfigService,
	tokenCatalogRepo ITokenCatalogRepository,
) *ScanService {
	var ret = &ScanService{
		rpcRepo:          rpcRepo,
		poolRepo:         poolRepo,
		tokenRepo:        tokenRepo,
		priceRepo:        priceRepo,
		routeRepo:        routeRepo,
		scannerStateRepo: scannerStateRepo,
		scanConfigSvc:    scanConfigSvc,
		tokenCatalogRepo: tokenCatalogRepo,
	}

	err := ret.initDataSource(context.Background())
	if err != nil {
		panic(err)
	}

	return ret
}

func (s *ScanService) Config() *config.Common {
	return s.scanConfigSvc.Config()
}

func (s *ScanService) initDataSource(ctx context.Context) error {
	pools, err := s.poolRepo.FindAll(ctx)
	if err != nil {
		logger.Errorf("failed to get all pools: %v", err)
		return err
	}

	for _, pool := range pools {
		_ = s.poolRepo.Set(ctx, pool.Address, pool)
	}

	tokens, err := s.tokenRepo.FindAll(ctx)
	if err != nil {
		logger.Errorf("failed to get all tokens: %v", err)
		return err
	}

	for _, token := range tokens {
		_ = s.tokenRepo.Set(ctx, token.Address, token)
	}

	prices, err := s.priceRepo.FindAll(ctx)
	if err != nil {
		logger.Errorf("failed to get all prices: %v", err)
		return err
	}

	for _, price := range prices {
		_ = s.priceRepo.Set(ctx, price.Address, price)
	}

	logger.Infof(
		"Load %v pool %v tokens %v prices from db", len(pools), len(tokens), len(prices),
	)

	return nil

}

func (s *ScanService) FetchOrGetTokenType(
	ctx context.Context, tokenAddress string, tokenType string, poolToken string,
) (entity.Token, error) {
	tokenAddress = strings.ToLower(tokenAddress)

	if token, err := s.tokenRepo.Get(ctx, tokenAddress); err == nil {
		return token, nil
	}

	token := entity.Token{}
	tokenInfo, tokenErr := s.readTokenSC(ctx, tokenAddress, TokenSCFieldsToRead...)
	if tokenErr != nil {
		if _, ok := tokenErr.(*repository.UnPackMulticallError); ok {
			return token, nil
		}
		logger.Errorf("Can not get token info given by address=%s from chain err=%v", tokenAddress, tokenErr)
		return token, tokenErr
	}

	token.Address = tokenAddress
	token.Symbol = tokenInfo.Symbol
	token.Name = tokenInfo.Name
	token.Decimals = tokenInfo.Decimals
	token.PoolAddress = poolToken
	token.Type = tokenType

	if err := s.tokenRepo.Save(ctx, token); err != nil {
		logger.Errorf("Can not save token address=%s to database err=%s", tokenAddress, err)
		return token, err
	}
	s.storeTokenTokenCatalog(ctx, token)
	return token, nil
}

func (s *ScanService) storeTokenTokenCatalog(ctx context.Context, token entity.Token) {
	catalogToken := CatalogToken{
		Source:  tokenSourceForAggregator,
		ChainID: strconv.Itoa(s.Config().ChainID),
		Address: token.Address,
	}

	err := s.tokenCatalogRepo.Upsert(ctx, catalogToken)

	if err != nil {
		logger.WithFields(logger.Fields{
			"catalogToken": catalogToken,
		}).Warnf("storeTokenTokenCatalog  failed cause by %v", err)
	}
}

func (s *ScanService) FetchOrGetToken(ctx context.Context, tokenAddress string) (entity.Token, error) {
	return s.FetchOrGetTokenType(ctx, tokenAddress, "erc20", "")
}

func (s *ScanService) ExistPool(ctx context.Context, address string) bool {
	return s.poolRepo.IsPoolExist(ctx, address)
}

func (s *ScanService) UpdatePoolExtra(ctx context.Context, address, extra string) error {
	pool, ok := s.GetPoolByAddress(ctx, address)
	if !ok {
		return errors.New("can not find pool by address : " + address)
	}
	if pool.Extra != extra {
		pool.Extra = extra
		if err := s.poolRepo.Save(ctx, pool); err != nil {
			return err
		}

		if err := s.indexPair(ctx, pool); err != nil {
			return err
		}
	}

	return nil
}

func (s *ScanService) UpdatePoolStaticExtra(ctx context.Context, address, staticExtra string) error {
	pool, ok := s.GetPoolByAddress(ctx, address)
	if !ok {
		return errors.New("can not find pool by address : " + address)
	}
	if pool.StaticExtra != staticExtra {
		pool.StaticExtra = staticExtra
		if err := s.poolRepo.Save(ctx, pool); err != nil {
			return err
		}

		if err := s.indexPair(ctx, pool); err != nil {
			return err
		}
	}

	return nil
}

func (s *ScanService) UpdatePoolSwapFee(ctx context.Context, address string, swapFee float64) error {
	pool, ok := s.GetPoolByAddress(ctx, address)
	if !ok {
		return errors.New("can not find pool by address : " + address)
	}
	if pool.SwapFee != swapFee {
		pool.SwapFee = swapFee
		if err := s.poolRepo.Save(ctx, pool); err != nil {
			return err
		}

		if err := s.indexPair(ctx, pool); err != nil {
			return err
		}
	}

	return nil
}

func (s *ScanService) UpdatePoolSupply(ctx context.Context, address string, totalSupply string) error {
	pool, ok := s.GetPoolByAddress(ctx, address)
	if !ok {
		return errors.New("can not find pool by address : " + address)
	}
	if pool.TotalSupply != totalSupply {
		pool.TotalSupply = totalSupply
		if err := s.poolRepo.Save(ctx, pool); err != nil {
			return err
		}

		if err := s.indexPair(ctx, pool); err != nil {
			return err
		}
	}

	return nil
}

func (s *ScanService) UpdatePoolReserve(ctx context.Context, address string, timestamp int64, reserves entity.PoolReserves) error {
	pool, ok := s.GetPoolByAddress(ctx, address)
	if !ok {
		return errors.New("can not find pool by address: " + address)
	}

	pool.Timestamp = timestamp
	pool.Reserves = reserves
	reserveUsd, err := s.calculateReserveUsd(ctx, pool)
	if err == nil {
		pool.ReserveUsd = reserveUsd
	}

	amplifiedTvl, err := s.calculateAmplifiedTvl(ctx, pool)
	if err == nil {
		pool.AmplifiedTvl = amplifiedTvl
	}

	return s.SavePool(ctx, pool)
}

func (s *ScanService) indexPair(ctx context.Context, pool entity.Pool) error {
	if !pool.HasReserves() && !pool.HasAmplifiedTvl() {
		return nil
	}

	poolTokens := pool.Tokens
	for i := 0; i < len(poolTokens); i++ {
		tokenI := poolTokens[i]
		whiteListI := s.scanConfigSvc.IsWhiteListToken(tokenI.Address)
		if !tokenI.Swappable {
			continue
		}
		for j := i + 1; j < len(poolTokens); j++ {
			tokenJ := poolTokens[j]
			if !tokenJ.Swappable {
				continue
			}
			whiteListJ := s.scanConfigSvc.IsWhiteListToken(tokenJ.Address)
			key := GetPairAddressKey(tokenI.Address, tokenJ.Address)

			if pool.HasReserves() {
				err := s.routeRepo.AddToSortedSetScoreByReserveUsd(ctx, pool, key, tokenI.Address, tokenJ.Address, whiteListI, whiteListJ)

				if err != nil {
					logger.Errorf("failed to AddToSortedSetScoreByReserveUsd, err: %v", err)
				}
			}

			if pool.HasAmplifiedTvl() {
				err := s.routeRepo.AddToSortedSetScoreByAmplifiedTvl(ctx, pool, key, tokenI.Address, tokenJ.Address, whiteListI, whiteListJ)

				if err != nil {
					logger.Errorf("failed to AddToSortedSetScoreByReserveUsd, err: %v", err)
				}
			}
		}
	}
	// curve metapool underlying
	if pool.Type == constant.PoolTypes.CurveMeta || pool.Type == constant.PoolTypes.CurveAave {
		var extra struct {
			UnderlyingTokens []string `json:"underlyingTokens"`
		}
		var err = json.Unmarshal([]byte(pool.StaticExtra), &extra)
		if err == nil {
			for i := 0; i < len(extra.UnderlyingTokens); i++ {
				for j := i + 1; j < len(extra.UnderlyingTokens); j++ {
					tokenI := extra.UnderlyingTokens[i]
					whiteListI := s.scanConfigSvc.IsWhiteListToken(tokenI)
					tokenJ := extra.UnderlyingTokens[j]
					whiteListJ := s.scanConfigSvc.IsWhiteListToken(tokenJ)
					key := GetPairAddressKey(tokenI, tokenJ)

					if pool.HasReserves() {
						err := s.routeRepo.AddToSortedSetScoreByReserveUsd(ctx, pool, key, tokenI, tokenJ, whiteListI, whiteListJ)

						if err != nil {
							logger.Errorf("failed to AddToSortedSetScoreByReserveUsd, err: %v", err)
						}
					}

					if pool.HasAmplifiedTvl() {
						err := s.routeRepo.AddToSortedSetScoreByAmplifiedTvl(ctx, pool, key, tokenI, tokenJ, whiteListI, whiteListJ)

						if err != nil {
							logger.Errorf("failed to AddToSortedSetScoreByAmplifiedTvl, err: %v", err)
						}
					}
				}
			}
		}
	}

	return nil
}

func (s *ScanService) SavePool(ctx context.Context, pool entity.Pool) error {
	if err := s.poolRepo.Save(ctx, pool); err != nil {
		return err
	}

	if err := s.indexPair(ctx, pool); err != nil {
		return err
	}

	return nil
}

func GetPairAddressKey(a, b string) string {
	if a > b {
		return a + "-" + b
	}
	return b + "-" + a
}

func (s *ScanService) GetTokenByAddress(ctx context.Context, address string) (entity.Token, bool) {
	token, err := s.tokenRepo.Get(ctx, address)

	if err != nil {
		return entity.Token{
			Address: address,
		}, false
	}

	return token, true
}

func (s *ScanService) GetPriceByAddress(ctx context.Context, address string) (entity.Price, bool) {
	price, err := s.priceRepo.Get(ctx, address)

	if err != nil {
		return entity.Price{
			Address: address,
		}, false
	}

	return price, true
}

func (s *ScanService) GetPoolByAddress(ctx context.Context, address string) (entity.Pool, bool) {
	pool, err := s.poolRepo.Get(ctx, address)

	if err != nil {
		return entity.Pool{
			Address: address,
		}, false
	}

	return pool, true
}

func (s ScanService) calculateReserveUsd(ctx context.Context, pool entity.Pool) (float64, error) {
	minLiquidityUsd := env.ParseFloatFromEnv(envvar.MinLiquidityUsd, constant.MinLiquidityUsd, 0, 10000)

	poolTokens := pool.Tokens
	switch pool.Type {
	case constant.PoolTypes.Uni, constant.PoolTypes.UniV3, constant.PoolTypes.Firebird, constant.PoolTypes.Dmm, constant.PoolTypes.ProMM:
		{
			var token0 = poolTokens[0]
			var token1 = poolTokens[1]
			token0Info, ok0 := s.GetTokenByAddress(ctx, token0.Address)
			token1Info, ok1 := s.GetTokenByAddress(ctx, token1.Address)
			if !ok0 || !ok1 {
				return 0, errors.New("token not found in tokenCache")
			}
			reserve0BF, err0 := new(big.Float).SetString(pool.Reserves[0])
			reserve1BF, err1 := new(big.Float).SetString(pool.Reserves[1])
			if !err0 || !err1 {
				return 0, errors.New("can not convert pool reserve to big float")
			}
			reserve0BF = new(big.Float).Quo(reserve0BF, constant.TenPowDecimals(token0Info.Decimals))
			reserve1BF = new(big.Float).Quo(reserve1BF, constant.TenPowDecimals(token1Info.Decimals))
			reserve0, _ := reserve0BF.Float64()
			reserve1, _ := reserve1BF.Float64()
			var price0 float64 = 0
			if price0Cache, ok := s.GetPriceByAddress(ctx, token0.Address); ok {
				price0 = price0Cache.MarketPrice
			}
			var price1 float64 = 0
			if price1Cache, ok := s.GetPriceByAddress(ctx, token1.Address); ok {
				price1 = price1Cache.MarketPrice
			}

			return price0*reserve0 + price1*reserve1, nil
		}

	case constant.PoolTypes.BalancerWeighted, constant.PoolTypes.BalancerStable, constant.PoolTypes.BalancerMetaStable:
		{
			var whitelistLiquidity float64
			var whitelistPriceLiquidity float64
			for i := range poolTokens {
				token, ok := s.GetTokenByAddress(ctx, poolTokens[i].Address)
				if !ok {
					return 0, errors.New("token not found in tokenCache")
				}
				price, ok := s.GetPriceByAddress(ctx, poolTokens[i].Address)
				if !ok {
					return 0, errors.New("price not found in priceCache")
				}
				var isWhitelist = s.scanConfigSvc.IsWhiteListToken(poolTokens[i].Address)
				var isStable = s.scanConfigSvc.IsStableToken(poolTokens[i].Address)
				reserveBF, reserveOk := new(big.Float).SetString(pool.Reserves[i])
				if !reserveOk {
					return 0, errors.New("can not convert pool reserve to big float")
				}
				reserveBF = new(big.Float).Quo(reserveBF, constant.TenPowDecimals(token.Decimals))
				reserve, _ := reserveBF.Float64()
				liquidity := price.MarketPrice * (reserve) * (1e18 / float64(poolTokens[i].Weight))
				if isStable {
					return liquidity, nil
				}
				if isWhitelist && price.Liquidity > whitelistPriceLiquidity {
					whitelistLiquidity = liquidity
					whitelistPriceLiquidity = price.Liquidity
				}
			}
			return whitelistLiquidity, nil
		}

	case constant.PoolTypes.LimitOrder:
		// Currently, total of reserve in limit order pool will very small with other pools. So it will filter in choosing pools process
		// We will use big hardcode number to can push it into eligible pools for findRoute algorithm.
		// TODO: when we has correct formula that pool's reserve can be eligible pools.
		return limitOrderPoolReserveUSD, nil
	case constant.PoolTypes.Synthetix:
		{
			var extra synthetix.Extra

			var err = json.Unmarshal([]byte(pool.Extra), &extra)
			if err != nil {
				return 0, err
			}

			poolState := extra.PoolState
			totalIssuedSUSD := poolState.TotalIssuedSUSD
			sUSDAddress := poolState.Synths[poolState.SUSDCurrencyKey]
			sUSDAddressStr := strings.ToLower(sUSDAddress.String())

			if eth.IsZeroAddress(sUSDAddress) {
				return 0, errors.New("sUSD not found")
			}

			sUSDTokenInfo, ok := s.GetTokenByAddress(ctx, sUSDAddressStr)
			if !ok {
				return 0, errors.New("token not found in tokenCache")
			}

			totalIssuedSUsdBF := new(big.Float).SetInt(totalIssuedSUSD)
			totalIssuedSUsdBF = new(big.Float).Quo(totalIssuedSUsdBF, constant.TenPowDecimals(sUSDTokenInfo.Decimals))
			// reserveUsd is the total issued sUSD in Synthetix protocol
			reserveUsd, _ := totalIssuedSUsdBF.Float64()

			return reserveUsd, nil
		}

	default:
		{
			var reserveUsd = float64(0)
			for i := range poolTokens {
				tokenInfo, ok := s.GetTokenByAddress(ctx, poolTokens[i].Address)
				if !ok {
					return 0, errors.New("token not found in tokenCache")
				}
				reserveBF, reserveOk := new(big.Float).SetString(pool.Reserves[i])
				if !reserveOk {
					return 0, errors.New("can not convert pool reserve to big float")
				}
				reserveBF = new(big.Float).Quo(reserveBF, constant.TenPowDecimals(tokenInfo.Decimals))
				reserve, _ := reserveBF.Float64()
				var price float64 = 0
				var liquidity float64 = 0
				if priceCache, ok := s.GetPriceByAddress(ctx, poolTokens[i].Address); ok {
					price = priceCache.MarketPrice
					liquidity = priceCache.Liquidity
				}

				var isWhitelist = s.scanConfigSvc.IsWhiteListToken(poolTokens[i].Address)
				var isStable = s.scanConfigSvc.IsStableToken(poolTokens[i].Address)
				if isStable || isWhitelist || liquidity > minLiquidityUsd {
					reserveUsd += reserve * price
				}
			}
			return reserveUsd, nil
		}
	}
}

func (s ScanService) calculateAmplifiedTvl(ctx context.Context, pool entity.Pool) (float64, error) {
	poolTokens := pool.Tokens
	switch pool.Type {
	case constant.PoolTypes.UniV3, constant.PoolTypes.ProMM:
		extraStr := univ3.Extra{}
		var err = json.Unmarshal([]byte(pool.Extra), &extraStr)
		if err != nil {
			return 0, err
		}

		if extraStr.Liquidity == nil || extraStr.SqrtPriceX96 == nil {
			return 0, nil
		}

		if extraStr.Liquidity.Cmp(constant.Zero) == 0 || extraStr.SqrtPriceX96.Cmp(constant.Zero) == 0 {
			return 0, nil
		}

		liquidityBF := new(big.Float).SetInt(extraStr.Liquidity)
		sqrtPriceBF := new(big.Float).SetInt(extraStr.SqrtPriceX96)

		var token0 = poolTokens[0]
		var token1 = poolTokens[1]

		var price0 float64 = 0
		if price0Cache, ok := s.GetPriceByAddress(ctx, token0.Address); ok {
			price0 = price0Cache.MarketPrice
		}
		price0BF := new(big.Float).SetFloat64(price0)

		var price1 float64 = 0
		if price1Cache, ok := s.GetPriceByAddress(ctx, token1.Address); ok {
			price1 = price1Cache.MarketPrice
		}
		price1BF := new(big.Float).SetFloat64(price1)

		// Formula: amplifiedTvl = priceOfXinUSD*Liquidity/SqrtPrice + Liquidity*SqrtPrice*priceOfYinUSD
		amplifiedTvlBF := new(big.Float).Add(
			new(big.Float).Mul(price0BF, new(big.Float).Quo(liquidityBF, sqrtPriceBF)),
			new(big.Float).Mul(price1BF, new(big.Float).Mul(liquidityBF, sqrtPriceBF)),
		)
		amplifiedTvl, _ := amplifiedTvlBF.Float64()

		return amplifiedTvl, nil
	default:
		if pool.HasReserves() {
			return pool.ReserveUsd, nil
		}
		return 0, nil
	}
}

func (s *ScanService) GetPoolIdsByExchange(ctx context.Context, dexID string) []string {
	poolIDs := s.poolRepo.GetPoolIdsByExchange(ctx, dexID)

	logger.Infof("ScanService.GetPoolIdsByExchange >>> dexID: [%s], poolLen: [%d]", dexID, len(poolIDs))

	return poolIDs
}

func (s *ScanService) GetPoolsByExchange(ctx context.Context, id string) ([]entity.Pool, error) {
	return s.poolRepo.GetPoolsByExchange(ctx, id)
}

func (s *ScanService) GetPoolsByAddresses(ctx context.Context, ids []string) ([]entity.Pool, error) {
	return s.poolRepo.GetByAddresses(ctx, ids)
}

func (s *ScanService) GetLastDexOffset(ctx context.Context, offsetKey string) (int, error) {
	return s.scannerStateRepo.GetDexOffset(ctx, offsetKey)
}

func (s *ScanService) SetLastDexOffset(ctx context.Context, offsetKey string, offset interface{}) error {
	return s.scannerStateRepo.SetDexOffset(ctx, offsetKey, offset)
}

func (s *ScanService) GetCurveAddressProviders(ctx context.Context) (string, error) {
	return s.scannerStateRepo.GetCurveAddressProviders(ctx)
}

func (s *ScanService) SetCurveAddressProviders(ctx context.Context, providers string) error {
	return s.scannerStateRepo.SetCurveAddressProviders(ctx, providers)
}

func (s *ScanService) GetLatestBlockTimestamp(ctx context.Context) (uint64, error) {
	return s.rpcRepo.GetLatestBlockTimestamp(ctx)
}

func (s *ScanService) Call(ctx context.Context, calls *repository.CallParams) (err error) {
	defer func() {
		if err != nil {
			handleRPCError(err, calls.Target, calls.Method, "Call")
		}
	}()

	defer func() {
		if r := recover(); r != nil {
			err = errors.Wrapf(ErrRPCPanic, "[%v]", r)
		}
	}()

	err = s.rpcRepo.Call(ctx, calls)

	return
}

func (s *ScanService) MultiCall(ctx context.Context, calls []*repository.CallParams) (err error) {
	defer func() {
		if err != nil {
			targets := make([]string, 0, len(calls))
			methods := make([]string, 0, len(calls))

			for _, call := range calls {
				targets = append(targets, call.Target)
				methods = append(methods, call.Method)
			}

			handleRPCError(err, strings.Join(targets, "-"), strings.Join(methods, "-"), "MultiCall")
		}
	}()

	defer func() {
		if r := recover(); r != nil {
			err = errors.Wrapf(ErrRPCPanic, "[%v]", r)
		}
	}()

	err = s.rpcRepo.MultiCall(ctx, calls)

	return
}

func (s *ScanService) TryAggregate(ctx context.Context, requireSuccess bool, calls []*repository.TryCallParams) (err error) {
	defer func() {
		if err != nil {
			targets := make([]string, 0, len(calls))
			methods := make([]string, 0, len(calls))

			for _, call := range calls {
				targets = append(targets, call.Target)
				methods = append(methods, call.Method)
			}

			handleRPCError(err, strings.Join(targets, "-"), strings.Join(methods, "-"), "TryAggregate")
		}
	}()

	defer func() {
		if r := recover(); r != nil {
			err = errors.Wrapf(ErrRPCPanic, "[%v]", r)
		}
	}()

	err = s.rpcRepo.TryAggregate(ctx, requireSuccess, calls)

	return
}

func (s *ScanService) TryAggregateForce(ctx context.Context, requireSuccess bool, calls []*repository.TryCallParams) (err error) {
	defer func() {
		if err != nil {
			targets := make([]string, 0, len(calls))
			methods := make([]string, 0, len(calls))

			for _, call := range calls {
				targets = append(targets, call.Target)
				methods = append(methods, call.Method)
			}

			handleRPCError(err, strings.Join(targets, "-"), strings.Join(methods, "-"), "TryAggregateForce")
		}
	}()

	defer func() {
		if r := recover(); r != nil {
			err = errors.Wrapf(ErrRPCPanic, "[%v]", r)
		}
	}()

	err = s.rpcRepo.TryAggregateForce(ctx, requireSuccess, calls)

	return
}

func (s *ScanService) readTokenSC(
	ctx context.Context,
	address string,
	fields ...TokenSCField,
) (TokenState, error) {
	var calls []*repository.TryCallUnPackParams

	var tokenState = TokenState{
		Address:  address,
		Name:     "",
		Symbol:   "",
		Decimals: 0,
	}

	var successName, successDecimals, successSymbol bool
	var mDecimals *big.Int
	var mSymbol, mName [32]byte

	for _, field := range fields {
		switch field {
		case TokenSCFieldDecimals:
			calls = append(calls, &repository.TryCallUnPackParams{
				ABI:       abis.ERC20,
				UnpackABI: []abi.ABI{abis.ERC20, abis.ERC20DS},
				Target:    address,
				Method:    TokenSCMethodDecimals,
				Params:    nil,
				Output:    []interface{}{&tokenState.Decimals, &mDecimals},
				Success:   &successDecimals,
			})
		case TokenSCFieldSymbol:
			calls = append(calls, &repository.TryCallUnPackParams{
				ABI:       abis.ERC20,
				UnpackABI: []abi.ABI{abis.ERC20, abis.ERC20DS},
				Target:    address,
				Method:    TokenSCMethodSymbol,
				Params:    nil,
				Output:    []interface{}{&tokenState.Symbol, &mSymbol},
				Success:   &successSymbol,
			})
		case TokenSCFieldName:
			calls = append(calls, &repository.TryCallUnPackParams{
				ABI:       abis.ERC20,
				UnpackABI: []abi.ABI{abis.ERC20, abis.ERC20DS},
				Target:    address,
				Method:    TokenSCMethodName,
				Params:    nil,
				Output:    []interface{}{&tokenState.Name, &mName},
				Success:   &successName,
			})
		default:
			continue
		}
	}

	if err := s.rpcRepo.TryAggregateUnpack(ctx, false, calls); err != nil {
		return tokenState, err
	}

	if tokenState.Decimals == 0 && mDecimals != nil {
		tokenState.Decimals = uint8(mDecimals.Int64())
	}

	if len(tokenState.Symbol) == 0 {
		tokenState.Symbol = getString(mSymbol)
	}

	if len(tokenState.Name) == 0 {
		tokenState.Name = getString(mName)
	}

	if !successDecimals || !successSymbol {
		logger.Warnf("can not get decimals or symbol of address %s", address)
		return tokenState, nil
	}

	// Fallback to using symbol if unable to fetch token's name
	if !successName {
		tokenState.Name = tokenState.Symbol
	}

	return tokenState, nil
}

// getString returns a string from a [32]byte to prevent out of range error
func getString(r interface{}) string {
	switch r := r.(type) {
	case string:
		return r
	case [32]byte:
		t := bytes.TrimRightFunc(r[:], func(r rune) bool {
			return r == 0
		})
		return string(t)
	}

	return ""
}

func handleRPCError(err error, callTargets string, callMethods string, method string) {
	logger.WithFields(map[string]interface{}{
		"method":      method,
		"error":       err,
		"callTargets": callTargets,
		"callMethods": callMethods,
	}).Error("RPC call failed")
}
