package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/api/params"
)

func TestGetRoutesParamsValidator_Validate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		params params.GetRoutesParams
		err    error
	}{
		{
			name: "it should return correct error when validateTokenIn failed",
			params: params.GetRoutesParams{
				TokenIn: "",
			},
			err: NewValidationError("tokenIn", "required"),
		},
		{
			name: "it should return correct error when validateTokenOut failed",
			params: params.GetRoutesParams{
				TokenIn:  "0xc7198437980c041c805a1edcba50c1ce5db95118",
				TokenOut: "",
			},
			err: NewValidationError("tokenOut", "required"),
		},
		{
			name: "it should return correct error when validateAmountIn failed",
			params: params.GetRoutesParams{
				TokenIn:  "0xc7198437980c041c805a1edcba50c1ce5db95118",
				TokenOut: "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664",
				AmountIn: "0",
			},
			err: NewValidationError("amountIn", "invalid"),
		},
		{
			name: "it should return correct error when validateFeeReceiver failed",
			params: params.GetRoutesParams{
				TokenIn:     "0xc7198437980c041c805a1edcba50c1ce5db95118",
				TokenOut:    "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664",
				AmountIn:    "1000000",
				FeeReceiver: "abc",
			},
			err: NewValidationError("feeReceiver", "invalid"),
		},
		{
			name: "it should return correct error when validateChargeFeeBy failed",
			params: params.GetRoutesParams{
				TokenIn:   "0xc7198437980c041c805a1edcba50c1ce5db95118",
				TokenOut:  "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664",
				AmountIn:  "1000000",
				FeeAmount: "a",
			},
			err: NewValidationError("feeAmount", "invalid"),
		},
		{
			name: "it should return correct error when validateChargeFeeBy failed",
			params: params.GetRoutesParams{
				TokenIn:     "0xc7198437980c041c805a1edcba50c1ce5db95118",
				TokenOut:    "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664",
				AmountIn:    "1000000",
				FeeAmount:   "1000",
				ChargeFeeBy: "abc",
			},
			err: NewValidationError("chargeFeeBy", "invalid"),
		},
		{
			name: "it should return correct error when validateGasPrice failed",
			params: params.GetRoutesParams{
				TokenIn:  "0xc7198437980c041c805a1edcba50c1ce5db95118",
				TokenOut: "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664",
				AmountIn: "1000000",
				GasPrice: "abc",
			},
			err: NewValidationError("gasPrice", "invalid"),
		},
		{
			name: "it should return nil when there is no error",
			params: params.GetRoutesParams{
				TokenIn:  "0xc7198437980c041c805a1edcba50c1ce5db95118",
				TokenOut: "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664",
				AmountIn: "1000000",
			},
			err: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validator := getRoutesParamsValidator{}

			err := validator.Validate(tc.params)

			assert.Equal(t, tc.err, err)
		})
	}
}

func TestGetRoutesParamsValidator_validateTokenIn(t *testing.T) {
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
			validator := getRoutesParamsValidator{}

			err := validator.validateTokenIn(tc.tokenIn, tc.tokenOut)

			assert.Equal(t, tc.err, err)
		})
	}
}

func TestGetRoutesParamsValidator_validateTokenOut(t *testing.T) {
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
			validator := getRoutesParamsValidator{}

			err := validator.validateTokenOut(tc.tokenOut)

			assert.Equal(t, tc.err, err)
		})
	}
}

func TestGetRoutesParamsValidator_validateAmountIn(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		amountIn string
		err      error
	}{
		{
			name:     "it should return [amountIn|invalid]",
			amountIn: "a",
			err:      NewValidationError("amountIn", "invalid"),
		},
		{
			name:     "it should return [amountIn|invalid] when amountIn is less than or equal 0",
			amountIn: "0",
			err:      NewValidationError("amountIn", "invalid"),
		},
		{
			name:     "it should return nil",
			amountIn: "1000000000000000000",
			err:      nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validator := getRoutesParamsValidator{}

			err := validator.validateAmountIn(tc.amountIn)

			assert.Equal(t, tc.err, err)
		})
	}
}

func TestGetRoutesParamsValidator_validateFeeReceiver(t *testing.T) {
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
			validator := getRoutesParamsValidator{}

			err := validator.validateFeeReceiver(tc.feeReceiver)

			assert.Equal(t, tc.err, err)
		})
	}
}

func TestGetRoutesParamsValidator_validateFeeAmount(t *testing.T) {
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
			validator := getRoutesParamsValidator{}

			err := validator.validateFeeAmount(tc.feeAmount)

			assert.Equal(t, tc.err, err)
		})
	}
}

func TestGetRoutesParamsValidator_validateChargeFeeBy(t *testing.T) {
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
			validator := getRoutesParamsValidator{}

			err := validator.validateChargeFeeBy(tc.chargeFeeBy, tc.feeAmount)

			assert.Equal(t, tc.err, err)
		})
	}
}

func TestGetRoutesParamsValidator_validateGasPrice(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		gasPrice string
		err      error
	}{
		{
			name:     "it should return nil when gasPrice is emtpy",
			gasPrice: "",
			err:      nil,
		},
		{
			name:     "it should return [gasPrice|invalid]",
			gasPrice: "abc",
			err:      NewValidationError("gasPrice", "invalid"),
		},
		{
			name:     "it should return nil when gasPrice is valid",
			gasPrice: "132423.3423423",
			err:      nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validator := getRoutesParamsValidator{}

			err := validator.validateGasPrice(tc.gasPrice)

			assert.Equal(t, tc.err, err)
		})
	}
}
