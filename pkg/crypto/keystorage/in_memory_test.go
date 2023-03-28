package keystorage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/router-service/pkg/crypto"
)

func Test_Get(t *testing.T) {
	keyFilepath := "testdata/rsa_key.json"
	type args struct {
		keyID string
	}
	type expectedKeyInfo struct {
		privateKeyPEM string
		publicKeyPEM  string
	}

	tests := []struct {
		name            string
		args            args
		expectedKeyInfo expectedKeyInfo
		err             error
	}{
		{
			name: "should get successfully when it is exist in the file",
			args: args{
				keyID: "client_data_1",
			},
			expectedKeyInfo: expectedKeyInfo{
				publicKeyPEM:  "-----BEGIN PUBLIC KEY-----\nMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQCI8ylfXTmRqvmD11+T+Cj4qZcZ\nAs2iTn2rFiZPZw8XEQp8Hw/dGFtqdho1KTf0XlezIveHKCguiKbq/w+FlQ/OFNvx\nlkK/3Ih0soC/0AgOpCo27ZQne7aTy3i8FQrugF1h5MkAotIql3JUgWlh6zTx/s8i\n4zeTk/EwKfDmWekEYQIDAQAB\n-----END PUBLIC KEY-----\n%",
				privateKeyPEM: "-----BEGIN RSA PRIVATE KEY-----\nMIICXAIBAAKBgQCI8ylfXTmRqvmD11+T+Cj4qZcZAs2iTn2rFiZPZw8XEQp8Hw/d\nGFtqdho1KTf0XlezIveHKCguiKbq/w+FlQ/OFNvxlkK/3Ih0soC/0AgOpCo27ZQn\ne7aTy3i8FQrugF1h5MkAotIql3JUgWlh6zTx/s8i4zeTk/EwKfDmWekEYQIDAQAB\nAoGAWNTRS0hfJTuv6XL0Tjiz6semeNS2qcccALO3Wd3RjfbBxE0prxIzidTdnwoD\nf4EKhenygTrtBXIiQ1/6o31S6DhecvMEqoI4LZmQMsbCNiP3vapaLwcV2DIDbsfd\nX21jZaWeCtP81M2CmMrLp/2k1tLfrAfZ7i7aB8fLn+M1VaECQQDJAqfAuZHe5ldD\nn7sgjl6Fj+WhqwnR78dk7VfRO93tFSSLOnmp4OvDmIUEHJ4F5tiif7aLAB2yenDN\nBh5Z9QCFAkEArmopE7j2Qb09WhRu3JAe9iKKOqDIMJH9JPdHb2uJGOhr7kfccCVn\neYjoWjZbtJGzUtpAZelB0iRu+AOhYYpJLQJBAJmrU/+8Xk4fnhrupCoxbQWCirTb\ngzhhrPf1kqs8r16uSS+/Vn+Ome8ATMBl+FDeuEMSi8UcI5fsjwvOX6m56dkCQEgL\nZAJYkaggAjq2XADRq2hiZhTHm0ms1BMz7ZcRpWTbhNG9b0oHuVFTgx7Ye1MAKEGe\nE6HFE0I5eHkMDtpao9UCQGUqT1pjATvl8ExTcziuX/wzjwuYoKAiFNkrj+MDA1c/\nbhZgLJOFNU4Em/fgfiM+fYwx9G4DFYosPKuGraH9GII=\n-----END RSA PRIVATE KEY-----\n%",
			},
			err: nil,
		},
		{
			name: "should return KeyPairNotFoundError when keyID is not existed in the file",
			args: args{
				keyID: "client_data_2",
			},
			err: crypto.NewKeyPairNotFoundError("client_data_2"),
		},
	}
	keyStorage, err := NewInMemoryStorageFromFile(keyFilepath)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyPairInfo, err := keyStorage.Get(ctx, tt.args.keyID)
			if tt.err != nil {
				assert.Equal(t, tt.err, err)
			} else {
				privateKey, err := crypto.ParseRsaPrivateKeyFromPEMStr(tt.expectedKeyInfo.privateKeyPEM)
				if err != nil {
					t.Fatal(err)
				}
				publicKey, err := crypto.ParseRsaPublicKeyFromPEMStr(tt.expectedKeyInfo.publicKeyPEM)
				if err != nil {
					t.Fatal(err)
				}
				assert.Equal(t, privateKey, keyPairInfo.PrivateKey)
				assert.Equal(t, publicKey, keyPairInfo.PublicKey)
			}
		})
	}

}

// TODO: write test for ListAll method
