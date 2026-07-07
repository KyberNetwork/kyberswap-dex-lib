package crypto

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/pkg/errors"
)

var Eip712Prefix = []byte{0x19, 0x01}

type Eip712Signer struct {
	key *ecdsa.PrivateKey
}

func NewEip712Signer(keyBytes []byte) *Eip712Signer {
	key, err := crypto.ToECDSA(keyBytes)
	if err != nil {
		return nil
	}
	return &Eip712Signer{key}
}

func (s *Eip712Signer) Sign(typedData apitypes.TypedData) ([]byte, error) {
	if s == nil {
		return nil, errors.New("signer is nil")
	}

	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return nil, errors.WithMessage(err, "typedData.HashStruct(EIP712Domain)")
	}

	typedDataHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return nil, errors.WithMessage(err, "typedData.HashStruct(primaryType)")
	}

	signatureHash := crypto.Keccak256(Eip712Prefix, domainSeparator, typedDataHash)

	signatureBytes, err := crypto.Sign(signatureHash, s.key)
	if err != nil {
		return nil, errors.WithMessage(err, "crypto.Sign")
	}
	signatureBytes[64] += 27

	return signatureBytes, nil
}
