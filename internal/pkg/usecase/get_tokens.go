package usecase

import (
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"

	"context"
)

type getTokensUseCase struct {
	tokenRepo ITokenRepository
	priceRepo IPriceRepository
}

func NewGetTokens(
	tokenRepo ITokenRepository,
	priceRepo IPriceRepository,
) *getTokensUseCase {
	return &getTokensUseCase{
		tokenRepo: tokenRepo,
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

	return &dto.GetTokensResult{
		Tokens: u.buildResultTokens(tokenByAddress, priceByAddress),
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

func (u *getTokensUseCase) buildResultTokens(
	tokenByAddress map[string]*entity.Token,
	priceByAddress map[string]*entity.Price,
) []*dto.GetTokensResultToken {
	resultTokens := make([]*dto.GetTokensResultToken, 0, len(tokenByAddress))

	for address, token := range tokenByAddress {
		var resultPrice *dto.GetTokensResultPrice
		if price, ok := priceByAddress[address]; ok {
			resultPrice = &dto.GetTokensResultPrice{
				Price:             price.Price,
				MarketPrice:       price.MarketPrice,
				PreferPriceSource: string(price.PreferPriceSource),
				Liquidity:         price.Liquidity,
				LpAddress:         price.LpAddress,
			}
		}

		resultTokens = append(resultTokens, &dto.GetTokensResultToken{
			Address:  address,
			Name:     token.Name,
			Decimals: token.Decimals,
			Symbol:   token.Symbol,
			Price:    resultPrice,
		})
	}
	return resultTokens
}
