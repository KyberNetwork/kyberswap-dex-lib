package crypto

import (
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

func ParseRsaPrivateKeyFromPEMStr(privateKeyPEM string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the key")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}

func ParseRsaPublicKeyFromPEMStr(pubPEM string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pubPEM))
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	switch pub := pub.(type) {
	case *rsa.PublicKey:
		return pub, nil
	default:
		break // fall through
	}
	return nil, errors.New("key type is not RSA")
}

func ExportRsaPrivateKeyAsPEMStr(privateKey *rsa.PrivateKey) string {
	privateKeyPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
		},
	)
	return string(privateKeyPEM)
}

func ExportRsaPublicKeyAsPEMStr(publicKey *rsa.PublicKey) (string, error) {
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", err
	}
	publicKeyPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: publicKeyBytes,
		},
	)

	return string(publicKeyPEM), nil
}

func generateHashFromMsg(message string) []byte {
	PSSmessage := []byte(message)
	sha256 := crypto.SHA256
	hash := sha256.New()
	hash.Write(PSSmessage)
	hashed := hash.Sum(nil)
	return hashed[:]
}
