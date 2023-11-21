package blackjack

import (
	"context"
	"errors"

	blackjackv1 "github.com/KyberNetwork/blackjack/proto/gen/blackjack/v1"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
	"github.com/samber/lo"
)

type blackjackRepository struct {
	blackjackClient blackjackv1.ServiceClient
}

func NewBlackjackRepository(client blackjackv1.ServiceClient) *blackjackRepository {
	return &blackjackRepository{
		blackjackClient: client,
	}
}

func (b *blackjackRepository) GetAddressBlacklisted(ctx context.Context, wallets []string) (map[string]bool, error) {
	operationName := "[blackjackRepository] GetAddressBlacklisted"
	span, _ := tracer.StartSpanFromContext(ctx, operationName)
	defer span.End()
	resp, err := b.blackjackClient.Check(ctx, &blackjackv1.CheckRequest{
		Wallets: wallets,
	})

	if err != nil {
		return nil, err
	}

	if len(resp.Data.Wallets) == 0 {
		return nil, errors.New("Blackjack's response is empty")
	}

	result := lo.SliceToMap(resp.Data.Wallets, func(data *blackjackv1.BlacklistData) (string, bool) {
		return data.GetWallet(), data.GetBlacklisted()
	})

	return result, nil
}
