package validator

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/KyberNetwork/router-service/internal/pkg/api/params"
	"github.com/KyberNetwork/router-service/internal/pkg/mocks/validator"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/clientid"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func TestBuildRouteParamsValidator_Validate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		params params.BuildRouteParams
		err    error
	}{
		{
			name:   "it should return correct error when validateRoute failed",
			params: params.BuildRouteParams{},
			err:    NewValidationError("route.route", "empty route"),
		},
		{
			name: "it should return correct error when validateTokenIn failed",
			params: params.BuildRouteParams{
				RouteSummary: params.RouteSummary{
					Route: [][]params.Swap{
						{
							{
								Pool: "pool1",
							},
						},
					},
					AmountInUSD:  "2",
					AmountOutUSD: "1.9",
				},
			},
			err: NewValidationError("tokenIn", "required"),
		},
		{
			name: "it should return correct error when validateTokenOut failed",
			params: params.BuildRouteParams{
				RouteSummary: params.RouteSummary{
					Route: [][]params.Swap{
						{
							{
								Pool: "pool1",
							},
						},
					},
					AmountInUSD:  "2",
					AmountOutUSD: "1.9",
					TokenIn:      "0xc7198437980c041c805a1edcba50c1ce5db95118",
					TokenOut:     "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c66",
				},
			},
			err: NewValidationError("tokenOut", "invalid"),
		},
		{
			name: "it should return correct error when validateSlippageTolerance failed because IgnoreCappedSlippage is false",
			params: params.BuildRouteParams{
				RouteSummary: params.RouteSummary{
					Route: [][]params.Swap{
						{
							{
								Pool: "pool1",
							},
						},
					},
					AmountInUSD:  "2",
					AmountOutUSD: "1.9",
					TokenIn:      "0xc7198437980c041c805a1edcba50c1ce5db95118",
					TokenOut:     "0xa7d7079b0fead91f3e65f86e8915cb59c1a34c66",
				},
				SlippageTolerance: 5001,
				Recipient:         "0xa7d7079b0fead91f3e65f86e8915cb59c1a34c66",
			},
			err: NewValidationError("slippageTolerance", "invalid"),
		},
		{
			name: "it should ignore validate Slippage Tolerance checking if IgnoreCappedSlippage is true",
			params: params.BuildRouteParams{
				RouteSummary: params.RouteSummary{
					Route: [][]params.Swap{
						{
							{
								Pool: "pool1",
							},
						},
					},
					AmountInUSD:  "2",
					AmountOutUSD: "1.9",
					TokenIn:      "0xc7198437980c041c805a1edcba50c1ce5db95118",
					TokenOut:     "0xa7d7079b0fead91f3e65f86e8915cb59c1a34c66",
				},
				SlippageTolerance:    10000,
				IgnoreCappedSlippage: true,
				Recipient:            "0xa7d7079b0fead91f3e65f86e8915cb59c1a34c66",
			},
			err: nil,
		},
		{
			name: "it should return correct error when validateChargeFeeBy failed",
			params: params.BuildRouteParams{
				RouteSummary: params.RouteSummary{
					Route: [][]params.Swap{
						{
							{
								Pool: "pool1",
							},
						},
					},
					AmountInUSD:  "2",
					AmountOutUSD: "1.9",
					TokenIn:      "0xc7198437980c041c805a1edcba50c1ce5db95118",
					TokenOut:     "0xa7d7079b0fead91f3e65f86e8915cb59c1a34c66",
					ExtraFee: params.ExtraFee{
						ChargeFeeBy: "ds",
						FeeAmount:   "1000",
						FeeReceiver: "0xa7d7079b0fead91f3e65f86e8915cb59c1a34c66",
					},
				},

				SlippageTolerance: 1500,
			},
			err: NewValidationError("chargeFeeBy", "invalid"),
		},
		{
			name: "it should return correct error when validateFeeReceiver failed",
			params: params.BuildRouteParams{
				RouteSummary: params.RouteSummary{
					Route: [][]params.Swap{
						{
							{
								Pool: "pool1",
							},
						},
					},
					AmountInUSD:  "2",
					AmountOutUSD: "1.9",
					TokenIn:      "0xc7198437980c041c805a1edcba50c1ce5db95118",
					TokenOut:     "0xa7d7079b0fead91f3e65f86e8915cb59c1a34c66",
					ExtraFee: params.ExtraFee{
						ChargeFeeBy: "currency_out",
						FeeAmount:   "1000",
						FeeReceiver: "a",
					},
				},
				SlippageTolerance: 0,
			},
			err: NewValidationError("feeReceiver", "invalid"),
		},
		{
			name: "it should return correct error when validateFeeAmount failed",
			params: params.BuildRouteParams{
				RouteSummary: params.RouteSummary{
					Route: [][]params.Swap{
						{
							{
								Pool: "pool1",
							},
						},
					},
					AmountInUSD:  "2",
					AmountOutUSD: "1.9",
					TokenIn:      "0xc7198437980c041c805a1edcba50c1ce5db95118",
					TokenOut:     "0xa7d7079b0fead91f3e65f86e8915cb59c1a34c66",
					ExtraFee: params.ExtraFee{
						ChargeFeeBy: "currency_out",
						FeeAmount:   "a",
						FeeReceiver: "0xa7d7079b0fead91f3e65f86e8915cb59c1a34c66",
					},
				},
				SlippageTolerance: 1500,
			},
			err: NewValidationError("feeAmount", "invalid"),
		},
		{
			name: "it should return correct error when validateDeadline failed",
			params: params.BuildRouteParams{
				RouteSummary: params.RouteSummary{
					Route: [][]params.Swap{
						{
							{
								Pool: "pool1",
							},
						},
					},
					AmountInUSD:  "2",
					AmountOutUSD: "1.9",
					TokenIn:      "0xc7198437980c041c805a1edcba50c1ce5db95118",
					TokenOut:     "0xa7d7079b0fead91f3e65f86e8915cb59c1a34c66",
				},
				SlippageTolerance: 1500,
				Deadline:          time.Now().Add(-3 * time.Hour).Unix(),
			},
			err: NewValidationError("deadline", "in the past"),
		},
		{
			name: "it should return correct error when validateRecipient failed",
			params: params.BuildRouteParams{
				RouteSummary: params.RouteSummary{
					Route: [][]params.Swap{
						{
							{
								Pool: "pool1",
							},
						},
					},
					AmountInUSD:  "2",
					AmountOutUSD: "1.9",
					TokenIn:      "0xc7198437980c041c805a1edcba50c1ce5db95118",
					TokenOut:     "0xa7d7079b0fead91f3e65f86e8915cb59c1a34c66",
				},
				SlippageTolerance: 1500,
				Recipient:         "a",
			},
			err: NewValidationError("recipient", "invalid"),
		},
		{
			name: "it should return correct error when validatePermit failed",
			params: params.BuildRouteParams{
				RouteSummary: params.RouteSummary{
					Route: [][]params.Swap{
						{
							{
								Pool: "pool1",
							},
						},
					},
					AmountInUSD:  "2",
					AmountOutUSD: "1.9",
					TokenIn:      "0xc7198437980c041c805a1edcba50c1ce5db95118",
					TokenOut:     "0xa7d7079b0fead91f3e65f86e8915cb59c1a34c66",
				},
				SlippageTolerance: 1500,
				Recipient:         "0xa7d7079b0fead91f3e65f86e8915cb59c1a34c66",
				Sender:            "0xa7d7079b0fead91f3e65f86e8915cb59c1a34c66",
				Permit:            "0x1111",
			},
			err: NewValidationError("permit", "invalid"),
		},
		{
			name: "it should return nil when permit is empty",
			params: params.BuildRouteParams{
				RouteSummary: params.RouteSummary{
					Route: [][]params.Swap{
						{
							{
								Pool: "pool1",
							},
						},
					},
					AmountInUSD:  "2",
					AmountOutUSD: "1.9",
					TokenIn:      "0xc7198437980c041c805a1edcba50c1ce5db95118",
					TokenOut:     "0xa7d7079b0fead91f3e65f86e8915cb59c1a34c66",
				},
				SlippageTolerance: 1500,
				Recipient:         "0xa7d7079b0fead91f3e65f86e8915cb59c1a34c66",
				Sender:            "0xa7d7079b0fead91f3e65f86e8915cb59c1a34c66",
				Permit:            "",
			},
			err: nil,
		},
		{
			name: "it should return nil when permit is a valid value",
			params: params.BuildRouteParams{
				RouteSummary: params.RouteSummary{
					Route: [][]params.Swap{
						{
							{
								Pool: "pool1",
							},
						},
					},
					AmountInUSD:  "2",
					AmountOutUSD: "1.9",
					TokenIn:      "0xc7198437980c041c805a1edcba50c1ce5db95118",
					TokenOut:     "0xa7d7079b0fead91f3e65f86e8915cb59c1a34c66",
				},
				SlippageTolerance: 1500,
				Recipient:         "0xa7d7079b0fead91f3e65f86e8915cb59c1a34c66",
				Sender:            "0xa7d7079b0fead91f3e65f86e8915cb59c1a34c66",
				Permit:            "0x1111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111",
			},
			err: nil,
		},
		{
			name: "it should return nil when all validation passed",
			params: params.BuildRouteParams{
				RouteSummary: params.RouteSummary{
					Route: [][]params.Swap{
						{
							{
								Pool: "pool1",
							},
						},
					},
					AmountInUSD:  "2",
					AmountOutUSD: "1.9",
					TokenIn:      "0xc7198437980c041c805a1edcba50c1ce5db95118",
					TokenOut:     "0xa7d7079b0fead91f3e65f86e8915cb59c1a34c66",
				},
				SlippageTolerance: 1500,
				Recipient:         "0xa7d7079b0fead91f3e65f86e8915cb59c1a34c66",
				Sender:            "0xa7d7079b0fead91f3e65f86e8915cb59c1a34c66",
			},
			err: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			paramsValidator := buildRouteParamsValidator{
				nowFunc: func() time.Time {
					return time.Now().Add(20 * time.Minute)
				},
				slippageValidator: NewSlippageValidator(
					SlippageValidatorConfig{
						SlippageToleranceGTE: 0,
						SlippageToleranceLTE: 5000,
					},
				),
			}

			err := paramsValidator.Validate(context.Background(), tc.params)

			assert.Equal(t, tc.err, err)
		})
	}
}

func TestBuildRouteParamsValidator_validateRoute(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		route params.RouteSummary
		err   error
	}{
		{
			name:  "it should return [route.route|empty route]",
			route: params.RouteSummary{},
			err:   NewValidationError("route.route", "empty route"),
		},
		{
			name: "it should return nil",
			route: params.RouteSummary{
				Route: [][]params.Swap{
					{
						{
							Pool: "pool1",
						},
					},
				},
				AmountInUSD:  "2",
				AmountOutUSD: "1.9",
			},
			err: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			paramsValidator := buildRouteParamsValidator{}

			err := paramsValidator.validateRoute(tc.route)

			assert.Equal(t, tc.err, err)
		})
	}
}

func TestBuildRouteParamsValidator_validateTokenIn(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		tokenIn  string
		tokenOut string
		err      error
	}{
		{
			name:     "it should return [tokenIn|required]",
			tokenIn:  "",
			tokenOut: "0x0000000000000000000000000000000000000000",
			err:      NewValidationError("tokenIn", "required"),
		},
		{
			name:     "it should return [tokenIn|invalid]",
			tokenIn:  "abc",
			tokenOut: "0x0000000000000000000000000000000000000000",
			err:      NewValidationError("tokenIn", "invalid"),
		},
		{
			name:     "it should return [tokenIn|invalid]",
			tokenIn:  "0xc7198437980c041c805a1edcba50c1ce5db95118",
			tokenOut: "0xc7198437980c041c805a1edcba50c1ce5db95118",
			err:      NewValidationError("tokenIn", "identical with tokenOut"),
		},
		{
			name:     "it should return nil",
			tokenIn:  "0xc7198437980c041c805a1edcba50c1ce5db95118",
			tokenOut: "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664",
			err:      nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			paramsValidator := buildRouteParamsValidator{}

			err := paramsValidator.validateTokenIn(tc.tokenIn, tc.tokenOut)

			assert.Equal(t, tc.err, err)
		})
	}
}

func TestBuildRouteParamsValidator_validateTokenOut(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		tokenOut string
		err      error
	}{
		{
			name:     "it should return [tokenOut|required]",
			tokenOut: "",
			err:      NewValidationError("tokenOut", "required"),
		},
		{
			name:     "it should return [tokenOut|invalid]",
			tokenOut: "abc",
			err:      NewValidationError("tokenOut", "invalid"),
		},
		{
			name:     "it should return nil",
			tokenOut: "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664",
			err:      nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			paramsValidator := buildRouteParamsValidator{}

			err := paramsValidator.validateTokenOut(tc.tokenOut)

			assert.Equal(t, tc.err, err)
		})
	}
}

func TestBuildRouteParamsValidator_validateChargeFeeBy(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		chargeFeeBy valueobject.ChargeFeeBy
		feeAmount   string
		err         error
	}{
		{
			name:        "it should return nil when feeAmount is empty",
			chargeFeeBy: "",
			feeAmount:   "",
			err:         nil,
		},
		{
			name:        "it should return [chargeFeeBy|invalid]",
			chargeFeeBy: "abc",
			feeAmount:   "1",
			err:         NewValidationError("chargeFeeBy", "invalid"),
		},
		{
			name:        "it should return nil",
			chargeFeeBy: "currency_in",
			feeAmount:   "1",
			err:         nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			paramsValidator := buildRouteParamsValidator{}

			err := paramsValidator.validateChargeFeeBy(tc.chargeFeeBy, tc.feeAmount)

			assert.Equal(t, tc.err, err)
		})
	}
}

func TestBuildRouteParamsValidator_validateFeeReceiver(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		feeReceiver string
		err         error
	}{
		{
			name:        "it should return nil when feeReceiver is emtpy",
			feeReceiver: "",
			err:         nil,
		},
		{
			name:        "it should return [feeReceiver|invalid]",
			feeReceiver: "abc",
			err:         NewValidationError("feeReceiver", "invalid"),
		},
		{
			name:        "it should return [feeReceiver|invalid]",
			feeReceiver: "0x0000000000000000000000000000000000000000",
			err:         NewValidationError("feeReceiver", "invalid"),
		},
		{
			name:        "it should return nil",
			feeReceiver: "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664",
			err:         nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			paramsValidator := buildRouteParamsValidator{}

			err := paramsValidator.validateFeeReceiver(tc.feeReceiver)

			assert.Equal(t, tc.err, err)
		})
	}
}

func TestBuildRouteParamsValidator_validateSender(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                 string
		shouldValidateSender bool
		sender               string
		err                  error
	}{
		{
			name:   "it should return nil",
			sender: "",
			err:    nil,
		},
		{
			name:                 "it should return [sender|required]",
			shouldValidateSender: true,
			sender:               "",
			err:                  NewValidationError("sender", "required"),
		},
		{
			name:                 "it should return [sender|invalid]",
			shouldValidateSender: true,
			sender:               "abc",
			err:                  NewValidationError("sender", "invalid"),
		},
		{
			name:                 "it should return [sender|invalid]",
			shouldValidateSender: true,
			sender:               "0x0000000000000000000000000000000000000000",
			err:                  NewValidationError("sender", "invalid"),
		},
		{
			name:                 "it should return nil",
			shouldValidateSender: true,
			sender:               "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664",
			err:                  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			paramsValidator := buildRouteParamsValidator{
				config: BuildRouteParamsConfig{
					FeatureFlags: valueobject.FeatureFlags{
						ShouldValidateSender: tc.shouldValidateSender,
					},
				},
			}

			err := paramsValidator.validateSender(tc.sender, &[]string{})

			assert.Equal(t, tc.err, err)
		})
	}
}

func TestBuildRouteParamsValidator_validateRecipient(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		recipient string
		err       error
	}{
		{
			name:      "it should return [recipient|required]",
			recipient: "",
			err:       NewValidationError("recipient", "required"),
		},
		{
			name:      "it should return [recipient|invalid]",
			recipient: "abc",
			err:       NewValidationError("recipient", "invalid"),
		},
		{
			name:      "it should return [recipient|invalid]",
			recipient: "0x0000000000000000000000000000000000000000",
			err:       NewValidationError("recipient", "invalid"),
		},
		{
			name:      "it should return nil",
			recipient: "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664",
			err:       nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			paramsValidator := buildRouteParamsValidator{}

			err := paramsValidator.validateRecipient(tc.recipient)

			assert.Equal(t, tc.err, err)
		})
	}
}

func TestBuildRouteParamsValidator_validateFeeAmount(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		feeAmount string
		err       error
	}{
		{
			name:      "it should return nil when feeAmount is empty",
			feeAmount: "",
			err:       nil,
		},
		{
			name:      "it should return [chargeFeeBy|invalid]",
			feeAmount: "a",
			err:       NewValidationError("feeAmount", "invalid"),
		},
		{
			name:      "it should return nil",
			feeAmount: "1",
			err:       nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			paramsValidator := buildRouteParamsValidator{}

			err := paramsValidator.validateFeeAmount(tc.feeAmount)

			assert.Equal(t, tc.err, err)
		})
	}
}

func TestBuildRouteParamsValidator_validateDeadline(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		deadline int64
		err      error
	}{
		{
			name:     "it should return [deadline|in the past]",
			deadline: time.Unix(1665560166, 0).Unix(),
			err:      NewValidationError("deadline", "in the past"),
		},
		{
			name:     "it should return nil when deadline in future",
			deadline: time.Unix(1665560168, 0).Unix(),
			err:      nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			paramsValidator := buildRouteParamsValidator{
				nowFunc: func() time.Time {
					return time.Unix(1665560167, 0)
				},
			}

			err := paramsValidator.validateDeadline(tc.deadline)

			assert.Equal(t, tc.err, err)
		})
	}
}

func TestBuildRouteParamsValidator_validateWallets(t *testing.T) {
	t.Parallel()

	kyberswapCtx := clientid.SetClientIDToContext(context.Background(), "kyberswap")
	notKyberswapCtx := clientid.SetClientIDToContext(context.Background(), "abc")

	testCases := []struct {
		context              context.Context
		name                 string
		isBlackjackEnabled   bool
		wallets              []string
		prepareBlackjackRepo func(ctrl *gomock.Controller) IBlackjackRepository
		err                  error
	}{
		{
			context:            kyberswapCtx,
			name:               "it should return nil",
			isBlackjackEnabled: false,
			wallets:            []string{"0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664", "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c665"},
			prepareBlackjackRepo: func(ctrl *gomock.Controller) IBlackjackRepository {
				return nil
			},
			err: nil,
		},
		{
			context:            kyberswapCtx,
			name:               "it should return nil",
			isBlackjackEnabled: true,
			wallets:            []string{"0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664", "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c665"},
			prepareBlackjackRepo: func(ctrl *gomock.Controller) IBlackjackRepository {
				mockBlackjacklRepo := validator.NewMockIBlackjackRepository(ctrl)
				mockBlackjacklRepo.EXPECT().
					Check(gomock.Any(), []string{"0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664", "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c665"}).
					Return(nil, fmt.Errorf("some error"))

				return mockBlackjacklRepo
			},
			err: nil,
		},
		{
			context:            kyberswapCtx,
			name:               "it should return [wallets|invalid]",
			isBlackjackEnabled: true,
			wallets:            []string{"0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664", "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c665"},
			prepareBlackjackRepo: func(ctrl *gomock.Controller) IBlackjackRepository {
				mockBlackjacklRepo := validator.NewMockIBlackjackRepository(ctrl)
				mockBlackjacklRepo.EXPECT().
					Check(gomock.Any(), []string{"0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664", "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c665"}).
					Return(map[string]bool{"0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664": true}, nil)

				return mockBlackjacklRepo
			},
			err: NewValidationError("wallets", "invalid"),
		},
		{
			context:            kyberswapCtx,
			name:               "it should return nil",
			isBlackjackEnabled: true,
			wallets:            []string{"0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664", "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c665"},
			prepareBlackjackRepo: func(ctrl *gomock.Controller) IBlackjackRepository {
				mockBlackjacklRepo := validator.NewMockIBlackjackRepository(ctrl)
				mockBlackjacklRepo.EXPECT().
					Check(gomock.Any(), []string{"0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664", "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c665"}).
					Return(map[string]bool{}, nil)

				return mockBlackjacklRepo
			},
			err: nil,
		},
		{
			context:            notKyberswapCtx,
			name:               "it should return nil, request is not from kyberswap UI",
			isBlackjackEnabled: true,
			wallets:            []string{"0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664", "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c665"},
			prepareBlackjackRepo: func(ctrl *gomock.Controller) IBlackjackRepository {
				return nil
			},
			err: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			v := buildRouteParamsValidator{
				config: BuildRouteParamsConfig{
					FeatureFlags: valueobject.FeatureFlags{
						IsBlackjackEnabled: tc.isBlackjackEnabled,
					},
				},
				blackjackRepo: tc.prepareBlackjackRepo(ctrl),
			}

			err := v.validateWallets(tc.context, tc.wallets)

			assert.Equal(t, tc.err, err)
		})
	}
}

func TestBuildRouteParamsValidator_validatePermit(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		permit string
		err    error
	}{
		{
			name:   "it should return [permit|invalid]",
			permit: "0x1111",
			err:    NewValidationError("permit", "invalid"),
		},
		{
			name:   "it should return nil when permit is empty",
			permit: "",
			err:    nil,
		},
		{
			name:   "it should return nil when permit is a valid value",
			permit: "0x1111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111",
			err:    nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			paramsValidator := buildRouteParamsValidator{}

			err := paramsValidator.validatePermit(tc.permit)

			assert.Equal(t, tc.err, err)
		})
	}
}
