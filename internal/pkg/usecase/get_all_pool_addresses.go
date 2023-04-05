package usecase

import (
	"context"
)

type getAllPoolAddressesUseCase struct {
	poolRepo IPoolRepository
}

func NewGetAllPoolAddressesUseCase(
	poolRepo IPoolRepository,
) *getAllPoolAddressesUseCase {
	return &getAllPoolAddressesUseCase{
		poolRepo: poolRepo,
	}
}

func (u *getAllPoolAddressesUseCase) Handle(ctx context.Context) ([]string, error) {
	poolAddresses, err := u.poolRepo.FindAllAddresses(ctx)
	if err != nil {
		return nil, err
	}
	return poolAddresses, nil
}
