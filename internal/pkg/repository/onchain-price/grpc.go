package onchainprice

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/KyberNetwork/blockchain-toolkit/float"
	onchainpricev1 "github.com/KyberNetwork/grpc-service/go/onchainprice/v1"
	dexlibEntity "github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/service-framework/pkg/client/grpcclient"
	"github.com/samber/lo"
	"github.com/sourcegraph/conc/iter"
	"google.golang.org/grpc/metadata"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type grpcRepository struct {
	chainId            valueobject.ChainID
	grpcClient         onchainpricev1.OnchainPriceServiceClient
	tokenRepository    ITokenRepository
	nativeTokenAddress string
}

const (
	MaxTokensPerCall = 100
)

type ITokenRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]*dexlibEntity.Token, error)
}

var (
	ErrInvalidPrice = errors.New("invalid price")
)

func NewGRPCRepository(config GrpcConfig, chainId valueobject.ChainID, tokenRepository ITokenRepository, nativeTokenAddress string) (*grpcRepository, error) {
	grpcConfig := grpcclient.Config{
		BaseURL:  config.BaseURL,
		Timeout:  config.Timeout,
		Insecure: config.Insecure,
		ClientID: config.ClientID,
	}

	grpcClient, err := grpcclient.New(onchainpricev1.NewOnchainPriceServiceClient, grpcclient.WithConfig(&grpcConfig))
	if err != nil {
		return nil, err
	}

	return &grpcRepository{
		chainId:            chainId,
		grpcClient:         grpcClient.C,
		tokenRepository:    tokenRepository,
		nativeTokenAddress: strings.ToLower(nativeTokenAddress),
	}, nil
}

type priceAndError struct {
	prices map[string]*entity.OnchainPrice
	err    error
}

func (r *grpcRepository) FindByAddresses(ctx context.Context, addresses []string) (map[string]*entity.OnchainPrice, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[onchainprice] grpcRepository.FindByAddresses")
	defer span.End()

	if len(addresses) <= MaxTokensPerCall {
		return r.findByAddressesSingleChunk(ctx, addresses)
	}

	// if there are too many tokens then break to several chunks
	prices := make(map[string]*entity.OnchainPrice, len(addresses))
	chunks := lo.Chunk(addresses, MaxTokensPerCall)

	chunkResults := iter.Map(chunks, func(chunk *[]string) priceAndError {
		chunkPrices, err := r.findByAddressesSingleChunk(ctx, *chunk)
		if err != nil {
			return priceAndError{nil, err}
		}
		return priceAndError{chunkPrices, nil}
	})

	for _, res := range chunkResults {
		if res.err != nil {
			// continue with what we have instead of erroring out
			logger.Errorf(ctx, "error getting onchain-price for chunk %v", res.err)
			continue
		}
		for token, price := range res.prices {
			prices[token] = price
		}
	}

	return prices, nil
}

func (r *grpcRepository) findByAddressesSingleChunk(ctx context.Context, addresses []string) (map[string]*entity.OnchainPrice, error) {
	if len(addresses) == 0 {
		return nil, nil
	}

	// get token info (decimal)
	tokens, err := r.tokenRepository.FindByAddresses(ctx, addresses)
	if err != nil {
		return nil, fmt.Errorf("[findByAddressesSingleChunk] failed to get token info %s %v", addresses, err)
	}
	decimalsByToken := make(map[string]uint8, len(tokens))
	for _, t := range tokens {
		decimalsByToken[t.Address] = t.Decimals
	}
	nativeDecimals := float.TenPow(18)

	// fetch price
	ctxHeader := metadata.AppendToOutgoingContext(ctx, "X-Chain-Id", strconv.Itoa(int(r.chainId)))
	res, err := r.grpcClient.ListPrices(ctxHeader, &onchainpricev1.ListPricesRequest{
		Tokens: addresses,
		Quotes: []string{r.nativeTokenAddress},
		Debug:  false,
	})
	if err != nil {
		return nil, err
	}

	prices := make(map[string]*entity.OnchainPrice, len(addresses))
	for _, p := range res.Result.Prices {
		decimals, ok := decimalsByToken[p.Address]
		if !ok {
			logger.Debugf(ctx, "unknown token info %v", p.Address)
			continue
		}

		tenPowDecimals := utils.TenPowDecimalsFloat(int(decimals))
		if tenPowDecimals == nil {
			logger.Debugf(ctx, "invalid token decimals %v %v", p.Address, decimals)
			continue
		}

		if _, ok := prices[p.Address]; !ok {
			prices[p.Address] = &entity.OnchainPrice{Decimals: decimals}
		}

		for _, detail := range p.Buy {
			if detail.Quote == r.nativeTokenAddress {
				price, ok := new(big.Float).SetString(detail.PriceByQuote)
				if !ok || price.Sign() < 0 {
					logger.Debugf(ctx, "invalid price %v (%v)", p.Address, detail.PriceByQuote)
					continue
				}

				prices[p.Address].NativePrice.Buy = price
				prices[p.Address].NativePriceRaw.Buy = new(big.Float).Quo(
					new(big.Float).Mul(price, nativeDecimals),
					tenPowDecimals)
			}
		}

		for _, detail := range p.Sell {
			if detail.Quote == r.nativeTokenAddress {
				price, ok := new(big.Float).SetString(detail.PriceByQuote)
				if !ok || price.Sign() < 0 {
					logger.Debugf(ctx, "invalid price %v (%v)", p.Address, detail.PriceByQuote)
					continue
				}

				prices[p.Address].NativePrice.Sell = price
				prices[p.Address].NativePriceRaw.Sell = new(big.Float).Quo(
					new(big.Float).Mul(price, nativeDecimals),
					tenPowDecimals)
			}
		}
	}

	for _, addr := range addresses {
		if _, ok := prices[addr]; !ok {
			decimals, ok := decimalsByToken[addr]
			if !ok {
				logger.Debugf(ctx, "unknown token info %v", addr)
				continue
			}

			prices[addr] = &entity.OnchainPrice{
				Decimals: decimals,
				NativePrice: entity.Price{
					Buy:  big.NewFloat(0),
					Sell: big.NewFloat(0),
				},
				NativePriceRaw: entity.Price{
					Buy:  big.NewFloat(0),
					Sell: big.NewFloat(0),
				},
			}
		}
	}

	logger.Debugf(ctx, "fetched prices %v", prices)

	return prices, nil
}

func (r *grpcRepository) GetNativePriceInUsd(ctx context.Context) (*big.Float, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[onchainprice] grpcRepository.GetNativePriceInUSD")
	defer span.End()

	// fetch price
	ctxHeader := metadata.AppendToOutgoingContext(ctx, "X-Chain-Id", strconv.Itoa(int(r.chainId)))
	res, err := r.grpcClient.GetPriceUSD(ctxHeader, &onchainpricev1.GetPriceUSDRequest{
		Address: r.nativeTokenAddress,
	})
	if err != nil {
		return nil, fmt.Errorf("[GetNativePriceInUsd] error getting onchain-price usd for native %v", err)
	}

	logger.Debugf(ctx, "fetched prices %v", res.Price)

	price, ok := new(big.Float).SetString(res.Price)
	if !ok {
		logger.Errorf(ctx, "invalid native price in usd %v", res.Price)
		return nil, ErrInvalidPrice
	}

	return price, nil
}
