package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/router-service/internal/pkg/api/params"
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
						"0x71C7656EC7ab88b098defB751B7401B5f6d8976F": true,
					},
				},
			}

			err := validator.Validate(tc.params)

			assert.Equal(t, tc.err, err)
		})
	}
}
