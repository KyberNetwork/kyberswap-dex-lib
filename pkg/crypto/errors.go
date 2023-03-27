package crypto

import "fmt"

type KeyPairNotFoundError struct {
	keyID string
}

func NewKeyPairNotFoundError(keyID string) *KeyPairNotFoundError {
	return &KeyPairNotFoundError{
		keyID: keyID,
	}
}

func (e *KeyPairNotFoundError) Error() string {
	return fmt.Sprintf("KeyPair %s is not found", e.keyID)
}
