package crypto

import (
	"context"
	"crypto"
	"crypto/rsa"
)

type localVerifier struct {
	keyPairStorage KeyPairStorage
}

func NewLocalVerifier(keyPairStorage KeyPairStorage) *localVerifier {
	return &localVerifier{
		keyPairStorage: keyPairStorage,
	}
}

func (s *localVerifier) Verify(ctx context.Context, signature []byte, keyID, message string) error {
	keyPairInfo, err := s.keyPairStorage.Get(ctx, keyID)
	if err != nil {
		return err
	}
	hashMsg := generateHashFromMsg(message)
	return rsa.VerifyPSS(keyPairInfo.PublicKey, crypto.SHA256, hashMsg, signature, nil)
}
