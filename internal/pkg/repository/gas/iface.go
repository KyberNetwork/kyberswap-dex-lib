package gas

import (
	"context"
	"math/big"
)

type IFallbackRepository interface {
	GetSuggestedGasPrice(ctx context.Context) (*big.Int, error)
}
