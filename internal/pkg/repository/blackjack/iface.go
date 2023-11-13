package blackjack

import (
	"context"
)

type IBlackjackRepository interface {
	GetAddressBlacklisted(ctx context.Context, wallets []string) (map[string]bool, error)
}
