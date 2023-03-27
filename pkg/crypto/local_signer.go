package crypto

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
)

type localSigner struct {
	keyPairStorage KeyPairStorage
}

func NewLocalSigner(keyPairStorage KeyPairStorage) *localSigner {
	return &localSigner{
		keyPairStorage: keyPairStorage,
	}
}

func (s *localSigner) Sign(ctx context.Context, keyID, message string) ([]byte, error) {
	keyPairInfo, err := s.keyPairStorage.Get(ctx, keyID)
	if err != nil {
		return nil, err
	}
	rng := rand.Reader
	hashMsg := generateHashFromMsg(message)
	return rsa.SignPSS(rng, keyPairInfo.PrivateKey, crypto.SHA256, hashMsg, nil)
}
