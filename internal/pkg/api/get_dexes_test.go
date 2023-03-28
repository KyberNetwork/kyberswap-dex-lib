package api

import (
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/KyberNetwork/router-service/internal/pkg/utils/test"
)

func TestGetDexes(t *testing.T) {
	testCases := []struct {
		name    string
		prepare func(ctrl *gomock.Controller) test.HTTPTestCase
	}{
		{
			name: "it should return OK",
			prepare: func(ctrl *gomock.Controller) test.HTTPTestCase {
				getDexesResult := &GetDexesResponse{
					Dexes: []string{"a", "b", "c"},
				}
				resp := SuccessResponse{
					Code:    0,
					Message: "successfully",
					Data:    getDexesResult,
				}
				return test.HTTPTestCase{
					ReqMethod:      http.MethodGet,
					ReqURL:         "/api/v1/dexes",
					ReqParams:      nil,
					ReqBody:        nil,
					ReqHandler:     GetDexes([]string{"a", "b", "c"}),
					RespHTTPStatus: http.StatusOK,
					RespBody:       resp,
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
