package validator

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/router-service/internal/pkg/api/params"
	"github.com/KyberNetwork/router-service/internal/pkg/mocks/validator"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/clientid"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func TestGetRouteEncodeParamsValidator_Validate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		params params.GetRouteEncodeParams
		err    error
	}{
		{
			name: "it should return correct error when tokenIn and tokenOut are identical failed",
			params: params.GetRouteEncodeParams{
				TokenIn:  "",
				TokenOut: "",
			},
			err: NewValidationError("tokenIn-out", "identical"),
		},
		{
			name: "it should return correct error when tokenIn is missing",
			params: params.GetRouteEncodeParams{
				TokenIn:  "",
				TokenOut: "0x00",
			},
			err: NewValidationError("tokenIn", "required"),
		},
		{
			name: "it should return correct error when tokenOut is missing",
			params: params.GetRouteEncodeParams{
				TokenIn:  "0xd1B47490209CcB7A806E8a45d9479490C040ABf4",
				TokenOut: "",
			},
			err: NewValidationError("tokenOut", "required"),
		},

		{
			name: "it should return correct error when amountIn is invalid",
			params: params.GetRouteEncodeParams{
				TokenIn:  "0xd1B47490209CcB7A806E8a45d9479490C040ABf4",
				TokenOut: "0xd1B47490209CcB7A806E8a45d9479490C040ABf2",
				AmountIn: "0",
			},
			err: NewValidationError("amountIn", "invalid"),
		},

		{
			name: "it should return correct error when feeReceiver is invalid",
			params: params.GetRouteEncodeParams{
				TokenIn:     "0xd1B47490209CcB7A806E8a45d9479490C040ABf4",
				TokenOut:    "0xd1B47490209CcB7A806E8a45d9479490C040ABf2",
				AmountIn:    "10",
				FeeReceiver: "abc",
			},
			err: NewValidationError("feeReceiver", "invalid"),
		},

		{
			name: "it should return correct error when feeAmount is invalid",
			params: params.GetRouteEncodeParams{
				TokenIn:     "0xd1B47490209CcB7A806E8a45d9479490C040ABf4",
				TokenOut:    "0xd1B47490209CcB7A806E8a45d9479490C040ABf2",
				AmountIn:    "10",
				FeeReceiver: "0xd1B47490209CcB7A806E8a45d9479490C040ABf4",
				FeeAmount:   "a",
			},
			err: NewValidationError("feeAmount", "invalid"),
		},

		{
			name: "it should return correct error when ChargeFeeBy is invalid",
			params: params.GetRouteEncodeParams{
				TokenIn:     "0xd1B47490209CcB7A806E8a45d9479490C040ABf4",
				TokenOut:    "0xd1B47490209CcB7A806E8a45d9479490C040ABf2",
				AmountIn:    "10",
				FeeReceiver: "0xd1B47490209CcB7A806E8a45d9479490C040ABf4",
				FeeAmount:   "2",
				ChargeFeeBy: "0",
			},
			err: NewValidationError("chargeFeeBy", "invalid"),
		},

		{
			name: "it should return correct error when gasPrice is invalid",
			params: params.GetRouteEncodeParams{
				TokenIn:     "0xd1B47490209CcB7A806E8a45d9479490C040ABf4",
				TokenOut:    "0xd1B47490209CcB7A806E8a45d9479490C040ABf2",
				AmountIn:    "10",
				GasPrice:    "abc",
				FeeReceiver: "0xd1B47490209CcB7A806E8a45d9479490C040ABf4",
				FeeAmount:   "2",
				ChargeFeeBy: "currency_in",
				To:          "0xd1B47490209CcB7A806E8a45d9479490C040ABf4",
			},
			err: NewValidationError("gasPrice", "invalid"),
		},
		{
			name: "it should return correct error when to is blacklisted",
			params: params.GetRouteEncodeParams{
				TokenIn:     "0xd1B47490209CcB7A806E8a45d9479490C040ABf4",
				TokenOut:    "0xd1B47490209CcB7A806E8a45d9479490C040ABf2",
				AmountIn:    "10",
				GasPrice:    "22",
				FeeReceiver: "0xd1B47490209CcB7A806E8a45d9479490C040ABf4",
				FeeAmount:   "2",
				ChargeFeeBy: "currency_in",
				To:          "0x71C7656EC7ab88b098defB751B7401B5f6d8976F",
			},
			err: NewValidationError("to", "invalid"),
		},
		{
			name: "it should return nil",
			params: params.GetRouteEncodeParams{
				TokenIn:     "0xd1B47490209CcB7A806E8a45d9479490C040ABf4",
				TokenOut:    "0xd1B47490209CcB7A806E8a45d9479490C040ABf2",
				AmountIn:    "10",
				GasPrice:    "22",
				FeeReceiver: "0xd1B47490209CcB7A806E8a45d9479490C040ABf4",
				FeeAmount:   "2",
				ChargeFeeBy: "currency_in",
				To:          "0xd1B47490209CcB7A806E8a45d9479490C040ABf4",
			},
			err: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validator := getRouteEncodeParamsValidator{
				config: GetRouteEncodeParamsConfig{
					BlacklistedRecipientSet: map[string]bool{
						"0x71c7656ec7ab88b098defb751b7401b5f6d8976f": true,
					},
				},
			}

			err := validator.Validate(context.Background(), tc.params)

			assert.Equal(t, tc.err, err)
		})
	}
}

func TestGetRouteEncodeDexesValidator_validateSources(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		sources string
		err     error
	}{
		{
			name:    "it should return nil when sources is oke",
			sources: "velodrome-v2, kyber-pmm",
			err:     nil,
		},
		{
			name:    "it should return nil when sources is empty",
			sources: "",
			err:     nil,
		},
		{
			name:    "it should return err when sources is invalid",
			sources: "velodrome2-v2, kyber-pmm",
			err:     NewValidationError("AvailableSources", "invalid"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validator := getRouteEncodeParamsValidator{}

			err := validator.validateSources(tc.sources)

			assert.Equal(t, tc.err, err)
		})
	}
}

func TestGetRouteEncodeDexesValidator_validateTo(t *testing.T) {
	t.Parallel()

	kyberswapCtx := clientid.SetClientIDToContext(context.Background(), "kyberswap")
	notKyberswapCtx := clientid.SetClientIDToContext(context.Background(), "abc")

	testCases := []struct {
		context              context.Context
		name                 string
		to                   string
		config               GetRouteEncodeParamsConfig
		prepareBlackjackRepo func(ctrl *gomock.Controller) IBlackjackRepository
		err                  error
	}{
		{
			context: kyberswapCtx,
			name:    "it should return [to|required]",
			to:      "",
			config:  GetRouteEncodeParamsConfig{},
			err:     NewValidationError("to", "required"),
			prepareBlackjackRepo: func(ctrl *gomock.Controller) IBlackjackRepository {
				return nil
			},
		},
		{
			context: kyberswapCtx,
			name:    "it should return [to|invalid]",
			to:      "abc",
			config:  GetRouteEncodeParamsConfig{},
			err:     NewValidationError("to", "invalid"),
			prepareBlackjackRepo: func(ctrl *gomock.Controller) IBlackjackRepository {
				return nil
			},
		},
		{
			context: kyberswapCtx,
			name:    "it should return [to][invalid], isBlackjackEnabled is false",
			to:      "0x71C7656EC7ab88b098defB751B7401B5f6d8976F",
			config: GetRouteEncodeParamsConfig{
				BlacklistedRecipientSet: map[string]bool{
					"0x71c7656ec7ab88b098defb751b7401b5f6d8976f": true,
				},
			},
			err: NewValidationError("to", "invalid"),
			prepareBlackjackRepo: func(ctrl *gomock.Controller) IBlackjackRepository {
				return nil
			},
		},
		{
			context: kyberswapCtx,
			name:    "it should return [to][blacklisted wallet], isBlackjackEnabled is true",
			to:      "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c764",
			err:     NewValidationError("to", "blacklisted wallet"),
			config: GetRouteEncodeParamsConfig{
				FeatureFlags: valueobject.FeatureFlags{
					IsBlackjackEnabled: true,
				},
			},
			prepareBlackjackRepo: func(ctrl *gomock.Controller) IBlackjackRepository {
				mockBlackjackRepo := validator.NewMockIBlackjackRepository(ctrl)
				to := "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c764"
				mockBlackjackRepo.EXPECT().Check(gomock.Any(), gomock.Any()).
					Return(map[string]bool{to: true}, nil)

				return mockBlackjackRepo
			},
		},
		{
			context: kyberswapCtx,
			name:    "it should return nil, isBlackjackEnabled is true",
			to:      "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664",
			err:     nil,
			config: GetRouteEncodeParamsConfig{
				FeatureFlags: valueobject.FeatureFlags{
					IsBlackjackEnabled: true,
				},
			},
			prepareBlackjackRepo: func(ctrl *gomock.Controller) IBlackjackRepository {
				mockBlackjackRepo := validator.NewMockIBlackjackRepository(ctrl)
				to := "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664"
				mockBlackjackRepo.EXPECT().Check(gomock.Any(), gomock.Any()).
					Return(map[string]bool{to: false}, nil)

				return mockBlackjackRepo
			},
		},
		{
			context: notKyberswapCtx,
			name:    "it should return nil, request is not from kyberswap UI",
			to:      "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664",
			err:     nil,
			config: GetRouteEncodeParamsConfig{
				FeatureFlags: valueobject.FeatureFlags{
					IsBlackjackEnabled: true,
				},
			},
			prepareBlackjackRepo: func(ctrl *gomock.Controller) IBlackjackRepository {
				return nil
			},
		},
		{
			context: kyberswapCtx,
			name:    "it should return nil, isBlackjackEnabled is true, Blackjack returns an error",
			to:      "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664",
			err:     nil,
			config: GetRouteEncodeParamsConfig{
				FeatureFlags: valueobject.FeatureFlags{
					IsBlackjackEnabled: true,
				},
			},
			prepareBlackjackRepo: func(ctrl *gomock.Controller) IBlackjackRepository {
				mockBlackjackRepo := validator.NewMockIBlackjackRepository(ctrl)
				mockBlackjackRepo.EXPECT().Check(gomock.Any(), gomock.Any()).Return(nil, errors.New("test"))
				return mockBlackjackRepo
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockBlackjackRepo := tc.prepareBlackjackRepo(ctrl)
			validator := getRouteEncodeParamsValidator{
				config:        tc.config,
				blackjackRepo: mockBlackjackRepo,
			}
			err := validator.validateTo(tc.context, tc.to)

			assert.Equal(t, tc.err, err)
		})
	}
}
