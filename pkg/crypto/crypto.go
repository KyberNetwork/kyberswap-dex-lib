package crypto

import "context"

type Signer interface {
	// Sign returns to a signature.
	Sign(ctx context.Context, keyID, message string) ([]byte, error)
}

type Verifier interface {
	// Verify used to verify the signature of a given message string.
	Verify(ctx context.Context, signature []byte, keyID, message string) error
}

// SymmetricSealer is an represent for sealer that it will use symmetric algorithm to seal data.
type SymmetricSealer interface {
	Sign(message []byte) ([]byte, error)
	Verify(message []byte, signature []byte) error
}
