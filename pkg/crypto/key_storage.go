package crypto

import (
	"context"
	"crypto/rsa"
)

type KeyPairInfo struct {
	ID         string
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
}

type KeyPairStorage interface {
	// Get return KeyPairInfo by key ID.
	Get(ctx context.Context, keyID string) (*KeyPairInfo, error)
}
