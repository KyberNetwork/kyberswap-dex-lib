package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/mocks/crypto"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/usecase/dto"
	cryptopkg "github.com/KyberNetwork/kyberswap-aggregator/pkg/crypto"
)

func Test_GetPublicKey(t *testing.T) {
	t.Parallel()

	type TestCase struct {
		name    string
		keyID   string
		prepare func(ctrl *gomock.Controller) *getPublicKeyUseCase
		result  *dto.GetPublicKeyResult
		err     error
	}
	internalErr := errors.New("Something went wrong when getting public key")
	testCases := []TestCase{
		{
			name: "it should return correct result when there is no error",
			prepare: func(ctrl *gomock.Controller) *getPublicKeyUseCase {
				mockKeyStorage := crypto.NewMockKeyPairStorage(ctrl)
				publicKey, _ := cryptopkg.ParseRsaPublicKeyFromPEMStr("-----BEGIN PUBLIC KEY-----\nMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQCI8ylfXTmRqvmD11+T+Cj4qZcZ\nAs2iTn2rFiZPZw8XEQp8Hw/dGFtqdho1KTf0XlezIveHKCguiKbq/w+FlQ/OFNvx\nlkK/3Ih0soC/0AgOpCo27ZQne7aTy3i8FQrugF1h5MkAotIql3JUgWlh6zTx/s8i\n4zeTk/EwKfDmWekEYQIDAQAB\n-----END PUBLIC KEY-----\n%")
				mockKeyStorage.
					EXPECT().
					Get(gomock.Any(), "1").
					Return(&cryptopkg.KeyPairInfo{
						ID:        "1",
						PublicKey: publicKey,
					}, nil)
				return NewGetPublicKeyUseCase(mockKeyStorage)
			},
			keyID: "1",
			result: &dto.GetPublicKeyResult{
				KeyID:     "1",
				PEMString: "-----BEGIN PUBLIC KEY-----\nMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQCI8ylfXTmRqvmD11+T+Cj4qZcZ\nAs2iTn2rFiZPZw8XEQp8Hw/dGFtqdho1KTf0XlezIveHKCguiKbq/w+FlQ/OFNvx\nlkK/3Ih0soC/0AgOpCo27ZQne7aTy3i8FQrugF1h5MkAotIql3JUgWlh6zTx/s8i\n4zeTk/EwKfDmWekEYQIDAQAB\n-----END PUBLIC KEY-----\n",
			},
		},
		{
			name: "it should return ErrPublicKeyNotFound when keyStorage return error KeyPairNotFoundError",
			prepare: func(ctrl *gomock.Controller) *getPublicKeyUseCase {
				mockKeyStorage := crypto.NewMockKeyPairStorage(ctrl)
				mockKeyStorage.
					EXPECT().
					Get(gomock.Any(), "1").
					Return(nil, cryptopkg.NewKeyPairNotFoundError("1"))
				return NewGetPublicKeyUseCase(mockKeyStorage)
			},
			keyID:  "1",
			result: nil,
			err:    ErrPublicKeyNotFound,
		},
		{
			name: "it should return an internal error when keyStorage return an internal error",
			prepare: func(ctrl *gomock.Controller) *getPublicKeyUseCase {
				mockKeyStorage := crypto.NewMockKeyPairStorage(ctrl)
				mockKeyStorage.
					EXPECT().
					Get(gomock.Any(), "1").
					Return(nil, internalErr)
				return NewGetPublicKeyUseCase(mockKeyStorage)
			},
			keyID: "1",
			err:   internalErr,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			getPublicKeyUseCase := tc.prepare(ctrl)
			result, err := getPublicKeyUseCase.Handle(context.Background(), tc.keyID)
			assert.ErrorIs(t, tc.err, err)
			assert.Equal(t, tc.result, result)
		})
	}
}
