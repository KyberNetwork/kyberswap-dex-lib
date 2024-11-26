package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/KyberNetwork/router-service/internal/pkg/mocks/usecase"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

var (
	ErrFailedToFetchConfig = errors.New("failed to fetch config")
)

func TestReloadConfigUseCase_ShouldReload(t *testing.T) {
	type TestCase struct {
		name    string
		prepare func(ctrl *gomock.Controller) *ReloadConfigUseCase
		result  bool
		err     error
	}

	testCases := []TestCase{
		{
			name: "it should return true when currentConfigHash is empty",
			prepare: func(ctrl *gomock.Controller) *ReloadConfigUseCase {
				mockConfigFetcherRepo := usecase.NewMockIConfigFetcherRepository(ctrl)

				return NewReloadConfigUseCase(mockConfigFetcherRepo)
			},
			result: true,
		},
		{
			name: "it should return false when currentConfigHash is not empty and no config changed",
			prepare: func(ctrl *gomock.Controller) *ReloadConfigUseCase {
				mockConfigFetcherRepo := usecase.NewMockIConfigFetcherRepository(ctrl)
				mockConfigFetcherRepo.
					EXPECT().
					GetConfigs(gomock.Any(), "aggregator", "xyz").
					Return(valueobject.RemoteConfig{
						Hash: "xyz",
					}, nil)

				uc := NewReloadConfigUseCase(mockConfigFetcherRepo)
				uc.currentConfigHash = "xyz"

				return uc
			},
			result: false,
			err:    nil,
		},
		{
			name: "it should return true when currentConfigHash is not empty and config is changed",
			prepare: func(ctrl *gomock.Controller) *ReloadConfigUseCase {
				mockConfigFetcherRepo := usecase.NewMockIConfigFetcherRepository(ctrl)
				mockConfigFetcherRepo.
					EXPECT().
					GetConfigs(gomock.Any(), "aggregator", "xyz").
					Return(valueobject.RemoteConfig{
						Hash: "abc",
					}, nil)

				uc := NewReloadConfigUseCase(mockConfigFetcherRepo)
				uc.currentConfigHash = "xyz"

				return uc
			},
			result: true,
			err:    nil,
		},
		{
			name: "it should return false when there is an error when fetching config",
			prepare: func(ctrl *gomock.Controller) *ReloadConfigUseCase {
				mockConfigFetcherRepo := usecase.NewMockIConfigFetcherRepository(ctrl)
				mockConfigFetcherRepo.
					EXPECT().
					GetConfigs(gomock.Any(), "aggregator", "xyz").
					Return(valueobject.RemoteConfig{}, ErrFailedToFetchConfig)

				uc := NewReloadConfigUseCase(mockConfigFetcherRepo)
				uc.currentConfigHash = "xyz"

				return uc
			},
			result: false,
			err:    ErrFailedToFetchConfig,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			reloadConfigUseCase := tc.prepare(ctrl)

			result, err := reloadConfigUseCase.ShouldReload(context.Background(), "aggregator")
			assert.Equal(t, tc.result, result)
			assert.Equal(t, tc.err, err)
		})
	}
}
