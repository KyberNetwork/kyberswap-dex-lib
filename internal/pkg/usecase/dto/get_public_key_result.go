package dto

import "github.com/KyberNetwork/router-service/pkg/crypto"

type GetPublicKeyResult struct {
	PEMString string `json:"pemString"`
	KeyID     string `json:"keyId"`
}

func NewPublicKeyResult(key *crypto.KeyPairInfo) (*GetPublicKeyResult, error) {
	publicKeyPEM, err := crypto.ExportRsaPublicKeyAsPEMStr(key.PublicKey)
	if err != nil {
		return nil, err
	}
	return &GetPublicKeyResult{
		PEMString: publicKeyPEM,
		KeyID:     key.ID,
	}, nil
}
