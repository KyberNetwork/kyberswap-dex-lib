package validator

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/router-service/internal/pkg/api/params"
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
			name: "it should return correct error when validateSlippageTolerance failed",
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
				SlippageTolerance: 2001,
			},
			err: NewValidationError("slippageTolerance", "invalid"),
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
				SlippageTolerance: 1500,
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
			},
			err: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validator := buildRouteParamsValidator{
				nowFunc: func() time.Time {
					return time.Now().Add(20 * time.Minute)
				},
			}

			err := validator.Validate(tc.params)

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
			validator := buildRouteParamsValidator{}

			err := validator.validateRoute(tc.route)

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
			tokenIn:  "0x0000000000000000000000000000000000000000",
			tokenOut: "0x0000000000000000000000000000000000000000",
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
			validator := buildRouteParamsValidator{}

			err := validator.validateTokenIn(tc.tokenIn, tc.tokenOut)

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
			validator := buildRouteParamsValidator{}

			err := validator.validateTokenOut(tc.tokenOut)

			assert.Equal(t, tc.err, err)
		})
	}
}

func TestBuildRouteParamsValidator_validateSlippageTolerance(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name              string
		slippageTolerance int64
		err               error
	}{
		{
			name:              "it should return [slippageTolerance|invalid]",
			slippageTolerance: 2001,
			err:               NewValidationError("slippageTolerance", "invalid"),
		},
		{
			name:              "it should return [chargeFeeBy|invalid]",
			slippageTolerance: -1,
			err:               NewValidationError("slippageTolerance", "invalid"),
		},
		{
			name:              "it should return nil",
			slippageTolerance: 0,
			err:               nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validator := buildRouteParamsValidator{}

			err := validator.validateSlippageTolerance(tc.slippageTolerance)

			assert.Equal(t, tc.err, err)
		})
	}
}

func TestBuildRouteParamsValidator_validateChargeFeeBy(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		chargeFeeBy string
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
			validator := buildRouteParamsValidator{}

			err := validator.validateChargeFeeBy(tc.chargeFeeBy, tc.feeAmount)

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
			name:        "it should return nil",
			feeReceiver: "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664",
			err:         nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validator := buildRouteParamsValidator{}

			err := validator.validateFeeReceiver(tc.feeReceiver)

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
			validator := buildRouteParamsValidator{}

			err := validator.validateFeeAmount(tc.feeAmount)

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
			validator := buildRouteParamsValidator{
				nowFunc: func() time.Time {
					return time.Unix(1665560167, 0)
				},
			}

			err := validator.validateDeadline(tc.deadline)

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
			name:      "it should return nil",
			recipient: "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664",
			err:       nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validator := buildRouteParamsValidator{}

			err := validator.validateRecipient(tc.recipient)

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
			validator := buildRouteParamsValidator{}

			err := validator.validatePermit(tc.permit)

			assert.Equal(t, tc.err, err)
		})
	}
}
