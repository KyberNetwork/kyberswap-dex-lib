package validator

import "context"

//go:generate mockgen -destination ../mocks/validator/blackjack.go -package validator github.com/KyberNetwork/router-service/internal/pkg/validator IBlackjackRepository
type IBlackjackRepository interface {
	Check(ctx context.Context, wallets []string) (map[string]bool, error)
}
