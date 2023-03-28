package api

import (
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/KyberNetwork/router-service/internal/pkg/utils/test"
)

func TestRouteController_GetRoutes(t *testing.T) {
	testCases := []struct {
		name    string
		prepare func(ctrl *gomock.Controller) test.HTTPTestCase
	}{}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			check := tc.prepare(ctrl)

			check.Run(t)
		})
	}
}
