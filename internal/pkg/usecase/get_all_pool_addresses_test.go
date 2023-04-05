package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/router-service/internal/pkg/mocks/usecase"
)

func TestGetAllPoolAddresses_Handle(t *testing.T) {
	t.Parallel()

	type TestCase struct {
		name    string
		prepare func(ctrl *gomock.Controller) *getAllPoolAddressesUseCase
		result  []string
		err     error
	}

	theError := errors.New("some error")

	testCases := []TestCase{
		{
			name: "it should return correct result when repository returns no error",
			prepare: func(ctrl *gomock.Controller) *getAllPoolAddressesUseCase {
				mockPoolRepo := usecase.NewMockIPoolRepository(ctrl)
				mockPoolRepo.EXPECT().
					FindAllAddresses(gomock.Any()).
					Return([]string{"pooladdress1", "pooladdress2"}, nil)

				return NewGetAllPoolAddressesUseCase(mockPoolRepo)
			},
			result: []string{"pooladdress1", "pooladdress2"},
			err:    nil,
		},
		{
			name: "it should return correct error when repository returns error",
			prepare: func(ctrl *gomock.Controller) *getAllPoolAddressesUseCase {

				mockPoolRepo := usecase.NewMockIPoolRepository(ctrl)
				mockPoolRepo.EXPECT().
					FindAllAddresses(gomock.Any()).
					Return(nil, theError)

				return NewGetAllPoolAddressesUseCase(mockPoolRepo)
			},
			result: nil,
			err:    theError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			getPools := tc.prepare(ctrl)

			result, err := getPools.Handle(context.Background())

			assert.Equal(t, tc.result, result)
			assert.ErrorIs(t, tc.err, err)
		})
	}
}
