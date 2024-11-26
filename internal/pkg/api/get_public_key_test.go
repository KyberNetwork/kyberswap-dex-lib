package api

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/KyberNetwork/router-service/internal/pkg/mocks/api"
	usecasepkg "github.com/KyberNetwork/router-service/internal/pkg/usecase"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/dto"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/test"
)

func Test_GetPublicKey(t *testing.T) {
	testCases := []struct {
		name    string
		prepare func(ctrl *gomock.Controller) test.HTTPTestCase
	}{
		{

			name: "it should return public key when keyId is correct",
			prepare: func(ctrl *gomock.Controller) test.HTTPTestCase {
				keyID := "1"
				getPublicKeyResult := &dto.GetPublicKeyResult{
					PEMString: "fake",
					KeyID:     keyID,
				}

				mockGetPublicKey := api.NewMockIGetPublicKeyUseCase(ctrl)
				mockGetPublicKey.EXPECT().
					Handle(gomock.Any(), keyID).
					Return(getPublicKeyResult, nil)

				resp := SuccessResponse{
					Code:    0,
					Message: "successfully",
					Data:    getPublicKeyResult,
				}

				return test.HTTPTestCase{
					ReqMethod:         http.MethodGet,
					PathIncludeParams: "/api/v1/keys/publics/:keyId",
					ReqURL:            fmt.Sprintf("/api/v1/keys/publics/%s", keyID),
					ReqHandler:        GetPublicKey(mockGetPublicKey),
					RespHTTPStatus:    http.StatusOK,
					RespBody:          resp,
				}
			},
		},
		{

			name: "it should return error public key is not found when keyID is incorrect",
			prepare: func(ctrl *gomock.Controller) test.HTTPTestCase {
				keyID := "1"

				err := usecasepkg.ErrPublicKeyNotFound
				mockGetPublicKey := api.NewMockIGetPublicKeyUseCase(ctrl)
				mockGetPublicKey.EXPECT().
					Handle(gomock.Any(), keyID).
					Return(nil, err)

				resp := ErrorResponse{
					Code:       4040,
					Message:    "public key can not be found",
					HTTPStatus: http.StatusNotFound,
				}

				return test.HTTPTestCase{
					ReqMethod:         http.MethodGet,
					PathIncludeParams: "/api/v1/keys/publics/:keyId",
					ReqURL:            fmt.Sprintf("/api/v1/keys/publics/%s", keyID),
					ReqHandler:        GetPublicKey(mockGetPublicKey),
					RespHTTPStatus:    http.StatusNotFound,
					RespBody:          resp,
				}
			},
		},

		{

			name: "it should return server internal error when usecase return an internal error",
			prepare: func(ctrl *gomock.Controller) test.HTTPTestCase {
				keyID := "1"

				err := errors.New("Something went wrong")
				mockGetPublicKey := api.NewMockIGetPublicKeyUseCase(ctrl)
				mockGetPublicKey.EXPECT().
					Handle(gomock.Any(), keyID).
					Return(nil, err)

				resp := ErrorResponse{
					Code:       500,
					Message:    "internal server error",
					HTTPStatus: http.StatusInternalServerError,
				}

				return test.HTTPTestCase{
					ReqMethod:         http.MethodGet,
					PathIncludeParams: "/api/v1/keys/publics/:keyId",
					ReqURL:            fmt.Sprintf("/api/v1/keys/publics/%s", keyID),
					ReqHandler:        GetPublicKey(mockGetPublicKey),
					RespHTTPStatus:    http.StatusInternalServerError,
					RespBody:          resp,
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			check := tc.prepare(ctrl)

			check.Run(t)
		})
	}
}
