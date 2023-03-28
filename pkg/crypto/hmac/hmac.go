package hmac

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	cryptopkg "github.com/KyberNetwork/router-service/pkg/crypto"
)

var ErrSignatureMisMatch = errors.New("signature is mismatch")

type hmacSealer struct {
	key []byte
}

func NewHMACSealer(key []byte) cryptopkg.SymmetricSealer {
	return &hmacSealer{key}
}

func (s *hmacSealer) Sign(message []byte) ([]byte, error) {
	mac := hmac.New(sha256.New, s.key)
	mac.Write(message)
	signature := hex.EncodeToString(mac.Sum(nil))
	return []byte(signature), nil
}

func (s *hmacSealer) Verify(message []byte, signature []byte) error {
	sig, err := hex.DecodeString(string(signature))
	if err != nil {
		return err
	}

	mac := hmac.New(sha256.New, s.key)
	mac.Write(message)

	if ok := hmac.Equal(sig, mac.Sum(nil)); !ok {
		return ErrSignatureMisMatch
	}
	return nil
}
