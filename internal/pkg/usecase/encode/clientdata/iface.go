package clientdata

//go:generate mockgen -destination ../../../mocks/usecase/encode/clientdata/signer.go -package clientdata github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/usecase/encode/clientdata ISigner

import (
	"context"
)

// ISigner signs data with an asymmetric algorithm
type ISigner interface {
	Sign(ctx context.Context, keyID, message string) ([]byte, error)
}
