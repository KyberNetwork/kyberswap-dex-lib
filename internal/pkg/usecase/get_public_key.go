package usecase

import (
	"context"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/crypto"
)

type getPublicKeyUseCase struct {
	keyPairStorage crypto.KeyPairStorage
}

func NewGetPublicKeyUseCase(keyPairStorage crypto.KeyPairStorage) *getPublicKeyUseCase {
	return &getPublicKeyUseCase{
		keyPairStorage,
	}
}

func (useCase *getPublicKeyUseCase) Handle(ctx context.Context, keyID string) (*dto.GetPublicKeyResult, error) {
	key, err := useCase.keyPairStorage.Get(ctx, keyID)
	if _, ok := err.(*crypto.KeyPairNotFoundError); ok {
		return nil, ErrPublicKeyNotFound
	}
	if err != nil {
		return nil, err
	}
	return dto.NewPublicKeyResult(key)
}
