package usecase

import (
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"

	"context"

	"k8s.io/apimachinery/pkg/util/sets"
)

type getTokensUseCase struct {
	tokenRepo ITokenRepository
	poolRepo  IPoolRepository
	priceRepo IPriceRepository
}

func NewGetTokens(
	tokenRepo ITokenRepository,
	poolRepo IPoolRepository,
	priceRepo IPriceRepository,
) *getTokensUseCase {
	return &getTokensUseCase{
		tokenRepo: tokenRepo,
		poolRepo:  poolRepo,
		priceRepo: priceRepo,
	}
}

func (u *getTokensUseCase) Handle(ctx context.Context, query dto.GetTokensQuery) (*dto.GetTokensResult, error) {
	tokenByAddress, err := u.getTokens(ctx, query.IDs)
	if err != nil {
		return nil, err
	}

	priceByAddress, err := u.getPrices(ctx, query.IDs)
	if err != nil {
		return nil, err
	}

	poolByAddress, err := u.getPools(ctx, query.PoolTokens, tokenByAddress)
	if err != nil {
		return nil, err
	}

	poolTokenByAddress, err := u.getPoolTokens(ctx, tokenByAddress, poolByAddress)
	if err != nil {
		return nil, err
	}

	resultTokens := aggregateTokens(
		query,
		tokenByAddress,
		priceByAddress,
		poolByAddress,
		poolTokenByAddress,
	)

	return &dto.GetTokensResult{
		Tokens: resultTokens,
	}, nil
}

func (u *getTokensUseCase) getTokens(
	ctx context.Context,
	addresses []string,
) (map[string]*entity.Token, error) {
	tokens, err := u.tokenRepo.FindByAddresses(ctx, addresses)
	if err != nil {
		return nil, err
	}

	tokenByAddress := make(map[string]*entity.Token, len(tokens))
	for _, token := range tokens {
		tokenByAddress[token.Address] = token
	}

	return tokenByAddress, nil
}

func (u *getTokensUseCase) getPrices(
	ctx context.Context,
	addresses []string,
) (map[string]*entity.Price, error) {
	prices, err := u.priceRepo.FindByAddresses(ctx, addresses)
	if err != nil {
		return nil, err
	}

	priceByAddress := make(map[string]*entity.Price, len(prices))
	for _, price := range prices {
		priceByAddress[price.Address] = price
	}

	return priceByAddress, nil
}

func (u *getTokensUseCase) getPools(
	ctx context.Context,
	showPoolTokens bool,
	tokenByAddress map[string]*entity.Token,
) (map[string]*entity.Pool, error) {
	if !showPoolTokens {
		return nil, nil
	}

	poolAddressesSet := sets.NewString()
	for _, token := range tokenByAddress {
		if len(token.PoolAddress) == 0 {
			continue
		}

		poolAddressesSet.Insert(token.PoolAddress)
	}

	if poolAddressesSet.Len() == 0 {
		return nil, nil
	}

	pools, err := u.poolRepo.FindByAddresses(ctx, poolAddressesSet.List())
	if err != nil {
		return nil, err
	}

	poolByAddress := make(map[string]*entity.Pool, len(pools))
	for _, pool := range pools {
		poolByAddress[pool.Address] = pool
	}

	return poolByAddress, nil
}

func (u *getTokensUseCase) getPoolTokens(
	ctx context.Context,
	tokenByAddress map[string]*entity.Token,
	poolByTokenAddress map[string]*entity.Pool,
) (map[string]*entity.Token, error) {
	if len(poolByTokenAddress) == 0 {
		return nil, nil
	}

	poolTokenByAddress := make(map[string]*entity.Token)
	poolTokenSet := sets.NewString()
	for _, pool := range poolByTokenAddress {
		for _, poolToken := range pool.Tokens {
			token, ok := tokenByAddress[poolToken.Address]
			if !ok {
				poolTokenSet.Insert(poolToken.Address)
				continue
			}

			poolTokenByAddress[poolToken.Address] = token
		}
	}

	if poolTokenSet.Len() == 0 {
		return poolTokenByAddress, nil
	}

	poolTokens, err := u.tokenRepo.FindByAddresses(ctx, poolTokenSet.List())
	if err != nil {
		return nil, err
	}

	for _, poolToken := range poolTokens {
		poolTokenByAddress[poolToken.Address] = poolToken
	}

	return poolTokenByAddress, nil
}

func aggregateTokens(
	query dto.GetTokensQuery,
	tokenByAddress map[string]*entity.Token,
	priceByAddress map[string]*entity.Price,
	poolByAddress map[string]*entity.Pool,
	poolTokenByAddress map[string]*entity.Token,
) []*dto.GetTokensResultToken {
	tokens := make([]*dto.GetTokensResultToken, 0, len(tokenByAddress))
	for address, entityToken := range tokenByAddress {
		token := dto.NewGetTokensResultTokenBuilder(query.Extra, query.PoolTokens).
			WithToken(entityToken).
			WithPrice(priceByAddress[address]).
			WithPool(poolByAddress[entityToken.PoolAddress], poolTokenByAddress).
			GetToken()

		tokens = append(tokens, token)
	}

	return tokens
}
