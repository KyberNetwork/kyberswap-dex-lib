package keystorage

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/goccy/go-json"

	"github.com/KyberNetwork/router-service/pkg/crypto"
)

type inMemoryStorage struct {
	db map[string]*crypto.KeyPairInfo
}

func NewInMemoryStorageFromFile(filePath string) (*inMemoryStorage, error) {
	db, err := loadDataFromFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load keyPairInfos from %s caused by %v", filePath, err)
	}
	return NewInMemoryStorage(db), nil
}

func NewInMemoryStorage(db map[string]*crypto.KeyPairInfo) *inMemoryStorage {
	return &inMemoryStorage{
		db,
	}
}

func (i *inMemoryStorage) Get(ctx context.Context, id string) (*crypto.KeyPairInfo, error) {
	keyPairInfo, ok := i.db[id]
	if !ok {
		return nil, crypto.NewKeyPairNotFoundError(id)
	}
	return keyPairInfo, nil
}

func loadDataFromFile(path string) (map[string]*crypto.KeyPairInfo, error) {
	type keyPairPEMString struct {
		PrivateKeyPEM string `json:"privateKeyPEM"`
		PublicKeyPEM  string `json:"publicKeyPEM"`
		ID            string `json:"id"`
	}
	jsonFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()
	bytesValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}

	var keyPairPEMStrings []*keyPairPEMString
	err = json.Unmarshal(bytesValue, &keyPairPEMStrings)
	if err != nil {
		return nil, err
	}

	result := make(map[string]*crypto.KeyPairInfo, len(keyPairPEMStrings))
	for _, keyPairPEMString := range keyPairPEMStrings {
		privateKey, err := crypto.ParseRsaPrivateKeyFromPEMStr(keyPairPEMString.PrivateKeyPEM)
		if err != nil {
			return nil, fmt.Errorf("failed when parsing private key cause by %v", err)
		}

		publicKey, err := crypto.ParseRsaPublicKeyFromPEMStr(keyPairPEMString.PublicKeyPEM)
		if err != nil {
			return nil, fmt.Errorf("failed when parsing public key cause by %v", err)
		}
		result[keyPairPEMString.ID] = &crypto.KeyPairInfo{
			PrivateKey: privateKey,
			PublicKey:  publicKey,
			ID:         keyPairPEMString.ID,
		}
	}

	return result, nil
}
