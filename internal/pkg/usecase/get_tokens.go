package usecase

import (
	"context"
	"strconv"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"

	tokenUtil "github.com/KyberNetwork/router-service/internal/pkg/utils/token"
)

type getTokensUseCase struct {
	tokenRepo              ITokenFullInfoRepository[entity.Token]
	onchainpriceRepository IOnchainPriceRepository
}

func NewGetTokens(
	tokenRepo ITokenFullInfoRepository[entity.Token],
	onchainpriceRepository IOnchainPriceRepository,
) *getTokensUseCase {
	return &getTokensUseCase{
		tokenRepo:              tokenRepo,
		onchainpriceRepository: onchainpriceRepository,
	}
}

func (u *getTokensUseCase) Handle(ctx context.Context, query dto.GetTokensQuery) (*dto.GetTokensResult, error) {
	tokenByAddress, err := u.getTokens(ctx, query.IDs)
	if err != nil {
		return nil, err
	}

	var priceByAddress map[string]*entity.Price
	var onchainPriceByAddress map[string]*routerEntity.OnchainPrice
	onchainPriceByAddress, err = u.getOnchainPrices(ctx, query.IDs)
	if err != nil {
		return nil, err
	}

	return &dto.GetTokensResult{
		Tokens: u.buildResultTokens(tokenByAddress, priceByAddress, onchainPriceByAddress),
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

func (u *getTokensUseCase) getOnchainPrices(
	ctx context.Context,
	addresses []string,
) (map[string]*routerEntity.OnchainPrice, error) {
	prices, err := u.onchainpriceRepository.FindByAddresses(ctx, addresses)
	if err != nil {
		return nil, err
	}

	return prices, nil
}

func (u *getTokensUseCase) buildResultTokens(
	tokenByAddress map[string]*entity.Token,
	priceByAddress map[string]*entity.Price,
	onchainPriceByAddress map[string]*routerEntity.OnchainPrice,
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
		} else if price, ok := onchainPriceByAddress[address]; ok {
			resultPrice = &dto.GetTokensResultPrice{
				Price: price.GetMidPriceUSD(),
			}
		}

		var hash string
		if hashInt, err := tokenUtil.HashToken(token.Address); err == nil {
			hash = strconv.Itoa(int(hashInt))
		}

		resultTokens = append(resultTokens, &dto.GetTokensResultToken{
			Address:  address,
			Name:     token.Name,
			Decimals: token.Decimals,
			Symbol:   token.Symbol,
			Price:    resultPrice,
			Hash:     hash,
		})
	}
	return resultTokens
}
