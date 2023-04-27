package usecase

import (
	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
)

type UpdateSuggestedGasPrice struct {
	gasRepository IGasRepository
}

func NewUpdateSuggestedGasPrice(
	gasRepository IGasRepository,
) *UpdateSuggestedGasPrice {
	return &UpdateSuggestedGasPrice{
		gasRepository: gasRepository,
	}
}

func (u *UpdateSuggestedGasPrice) Handle(ctx context.Context) (*dto.UpdateGasPriceResult, error) {
	suggestedGasPrice, err := u.gasRepository.UpdateSuggestedGasPrice(ctx)
	if err != nil {
		return nil, err
	}

	return &dto.UpdateGasPriceResult{
		SuggestedGasPrice: suggestedGasPrice,
	}, nil
}
