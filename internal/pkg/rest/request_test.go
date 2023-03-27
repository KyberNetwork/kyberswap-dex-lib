package rest

import (
	"testing"

	aggregatorerrors "github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/errors"

	"github.com/KyberNetwork/kyberswap-error/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestFindEncodedRouteRequest_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		request     *FindEncodedRouteRequest
		expectedErr *errors.DomainError
	}{
		{
			name: "it should return correct error when validate amountIn failed",
			request: &FindEncodedRouteRequest{
				FindRouteRequest: FindRouteRequest{
					AmountIn: "abc",
				},
			},
			expectedErr: errors.NewDomainErrorInvalid(nil, "amountIn"),
		},
		{
			name: "it should return correct error when validate tokens failed",
			request: &FindEncodedRouteRequest{
				FindRouteRequest: FindRouteRequest{
					AmountIn: "123",
					TokenIn:  "123",
					TokenOut: "123",
				},
			},
			expectedErr: aggregatorerrors.NewDomainErrTokensAreIdentical(),
		},
		{
			name: "it should return correct error when validate to failed",
			request: &FindEncodedRouteRequest{
				FindRouteRequest: FindRouteRequest{
					AmountIn: "123",
					TokenIn:  "123",
					TokenOut: "234",
				},
				EncodedRequestParams: EncodedRequestParams{
					To: "",
				},
			},
			expectedErr: errors.NewDomainErrorRequired(nil, "to"),
		},
		{
			name: "it should return correct error when validate feeReceiver failed",
			request: &FindEncodedRouteRequest{
				FindRouteRequest: FindRouteRequest{
					AmountIn: "123",
					TokenIn:  "123",
					TokenOut: "234",
				},
				EncodedRequestParams: EncodedRequestParams{
					FeeReceiver: "adas",
					To:          "0x0000000000000000000000000000000000000000",
				},
			},
			expectedErr: errors.NewDomainErrorInvalid(nil, "feeReceiver"),
		},
		{
			name: "it should return correct error when validate feeAmount failed",
			request: &FindEncodedRouteRequest{
				FindRouteRequest: FindRouteRequest{
					AmountIn: "123",
					TokenIn:  "123",
					TokenOut: "234",
				},
				EncodedRequestParams: EncodedRequestParams{
					To:        "0x0000000000000000000000000000000000000000",
					FeeAmount: "abc",
				},
			},
			expectedErr: errors.NewDomainErrorInvalid(nil, "feeAmount"),
		},
		{
			name: "it should return correct error when feeAmount >= amountIn",
			request: &FindEncodedRouteRequest{
				FindRouteRequest: FindRouteRequest{
					AmountIn: "123",
					TokenIn:  "123",
					TokenOut: "234",
				},
				EncodedRequestParams: EncodedRequestParams{
					To:          "0x0000000000000000000000000000000000000000",
					ChargeFeeBy: "currency_in",
					FeeAmount:   "123",
				},
			},
			expectedErr: errors.NewDomainErrorInvalid(nil, "feeAmount"),
		},
		{
			name: "it should return correct error when validate chargeFeeBy failed",
			request: &FindEncodedRouteRequest{
				FindRouteRequest: FindRouteRequest{
					AmountIn: "123",
					TokenIn:  "123",
					TokenOut: "234",
				},
				EncodedRequestParams: EncodedRequestParams{
					FeeAmount:   "123",
					To:          "0x0000000000000000000000000000000000000000",
					ChargeFeeBy: "invalid",
				},
			},
			expectedErr: errors.NewDomainErrorInvalid(nil, "chargeFeeBy"),
		},
		{
			name: "it should return correct error when validate slippageTolerance failed",
			request: &FindEncodedRouteRequest{
				FindRouteRequest: FindRouteRequest{
					AmountIn: "123",
					TokenIn:  "123",
					TokenOut: "234",
				},
				EncodedRequestParams: EncodedRequestParams{
					FeeAmount:         "123",
					To:                "0x0000000000000000000000000000000000000000",
					ChargeFeeBy:       "currency_out",
					SlippageTolerance: "2001",
				},
			},
			expectedErr: errors.NewDomainErrorOutOfRange(nil, "slippageTolerance"),
		},
		{
			name: "it should return correct error when validate permit failed",
			request: &FindEncodedRouteRequest{
				FindRouteRequest: FindRouteRequest{
					AmountIn: "123",
					TokenIn:  "123",
					TokenOut: "234",
				},
				EncodedRequestParams: EncodedRequestParams{
					FeeAmount:   "123",
					To:          "0x0000000000000000000000000000000000000000",
					ChargeFeeBy: "currency_out",
					Permit:      "0x1111",
				},
			},
			expectedErr: errors.NewDomainErrorInvalid(nil, "permit"),
		},
		{
			name: "it should return nil when permit is empty",
			request: &FindEncodedRouteRequest{
				FindRouteRequest: FindRouteRequest{
					AmountIn: "123",
					TokenIn:  "123",
					TokenOut: "234",
				},
				EncodedRequestParams: EncodedRequestParams{
					FeeAmount:   "123",
					To:          "0x0000000000000000000000000000000000000000",
					ChargeFeeBy: "currency_out",
				},
			},
			expectedErr: nil,
		},
		{
			name: "it should return nil when permit is a valid value",
			request: &FindEncodedRouteRequest{
				FindRouteRequest: FindRouteRequest{
					AmountIn: "123",
					TokenIn:  "123",
					TokenOut: "234",
				},
				EncodedRequestParams: EncodedRequestParams{
					FeeAmount:   "123",
					To:          "0x0000000000000000000000000000000000000000",
					ChargeFeeBy: "currency_out",
					Permit:      "0x1111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111",
				},
			},
			expectedErr: nil,
		},
		{
			name: "it should return nil when all validation succeed",
			request: &FindEncodedRouteRequest{
				FindRouteRequest: FindRouteRequest{
					AmountIn: "123",
					TokenIn:  "123",
					TokenOut: "234",
				},
				EncodedRequestParams: EncodedRequestParams{
					FeeAmount:   "123",
					To:          "0x0000000000000000000000000000000000000000",
					ChargeFeeBy: "currency_out",
				},
			},
			expectedErr: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.request.Validate()

			assert.Equal(t, test.expectedErr, err)
		})
	}
}

func TestFindEncodedRouteRequest_ValidateAmountIn(t *testing.T) {
	t.Parallel()

	t.Run("it should return nil when amountIn is valid", func(t *testing.T) {
		r := &FindEncodedRouteRequest{
			FindRouteRequest: FindRouteRequest{
				AmountIn: "1000000",
			},
		}

		err := r.ValidateAmountIn()

		assert.Nil(t, err)
	})

	t.Run("it should return correct error when amountIn is invalid", func(t *testing.T) {
		expectedErr := errors.NewDomainErrorInvalid(nil, "amountIn")
		r := &FindEncodedRouteRequest{
			FindRouteRequest: FindRouteRequest{
				AmountIn: "abc",
			},
		}

		err := r.ValidateAmountIn()

		assert.Equal(t, expectedErr, err)
	})
}

func TestFindEncodedRouteRequest_ValidateTokens(t *testing.T) {
	t.Parallel()

	t.Run("it should return nil when tokenIn and tokenOut are different", func(t *testing.T) {
		r := &FindEncodedRouteRequest{
			FindRouteRequest: FindRouteRequest{
				TokenIn:  "123",
				TokenOut: "234",
			},
		}

		err := r.ValidateTokens()

		assert.Nil(t, err)
	})

	t.Run("it should return correct error when tokenIn and tokenOut are identical", func(t *testing.T) {
		expectedErr := aggregatorerrors.NewDomainErrTokensAreIdentical()

		r := &FindEncodedRouteRequest{
			FindRouteRequest: FindRouteRequest{
				TokenIn:  "123",
				TokenOut: "123",
			},
		}

		err := r.ValidateTokens()

		assert.Equal(t, expectedErr, err)
	})
}

func TestFindEncodedRouteRequest_ValidateTo(t *testing.T) {
	t.Parallel()

	t.Run("it should return nil when to is valid", func(t *testing.T) {
		r := &FindEncodedRouteRequest{
			EncodedRequestParams: EncodedRequestParams{
				To: "0x0000000000000000000000000000000000000000",
			},
		}

		err := r.ValidateTo()

		assert.Nil(t, err)
	})

	t.Run("it should return correct error when to is empty", func(t *testing.T) {
		expectedError := errors.NewDomainErrorRequired(nil, "to")

		r := &FindEncodedRouteRequest{
			EncodedRequestParams: EncodedRequestParams{
				To: "",
			},
		}

		err := r.ValidateTo()

		assert.Equal(t, expectedError, err)
	})

	t.Run("it should return correct error when to is invalid", func(t *testing.T) {
		expectedError := errors.NewDomainErrorInvalid(nil, "to")

		r := &FindEncodedRouteRequest{
			EncodedRequestParams: EncodedRequestParams{
				To: "aabc",
			},
		}

		err := r.ValidateTo()

		assert.Equal(t, expectedError, err)
	})
}

func TestFindEncodedRouteRequest_ValidateFeeReceiver(t *testing.T) {
	t.Parallel()

	t.Run("it should return nil when feeReceiver is empty", func(t *testing.T) {
		r := &FindEncodedRouteRequest{
			EncodedRequestParams: EncodedRequestParams{
				FeeReceiver: "",
			},
		}

		err := r.ValidateFeeReceiver()

		assert.Nil(t, err)
	})

	t.Run("it should return correct error when feeReceiver is invalid", func(t *testing.T) {
		expectedError := errors.NewDomainErrorInvalid(nil, "feeReceiver")

		r := &FindEncodedRouteRequest{
			EncodedRequestParams: EncodedRequestParams{
				FeeReceiver: "adas",
			},
		}

		err := r.ValidateFeeReceiver()

		assert.Equal(t, expectedError, err)
	})

	t.Run("it should return nil when feeReceiver is valid", func(t *testing.T) {
		r := &FindEncodedRouteRequest{
			EncodedRequestParams: EncodedRequestParams{
				FeeReceiver: "0x0000000000000000000000000000000000000000",
			},
		}

		err := r.ValidateFeeReceiver()

		assert.Nil(t, err)
	})
}

func TestFindEncodedRouteRequest_ValidateFeeAmount(t *testing.T) {
	t.Parallel()

	t.Run("it should return nil when feeAmount is empty", func(t *testing.T) {
		r := &FindEncodedRouteRequest{
			EncodedRequestParams: EncodedRequestParams{
				FeeAmount: "",
			},
		}

		err := r.ValidateFeeAmount()

		assert.Nil(t, err)
	})

	t.Run("it should return correct error when feeAmount is invalid", func(t *testing.T) {
		expectedError := errors.NewDomainErrorInvalid(nil, "feeAmount")

		r := &FindEncodedRouteRequest{
			EncodedRequestParams: EncodedRequestParams{
				FeeAmount: "abc",
			},
		}

		err := r.ValidateFeeAmount()

		assert.Equal(t, expectedError, err)
	})

	t.Run("it should return nil when feeAmount is valid", func(t *testing.T) {
		r := &FindEncodedRouteRequest{
			EncodedRequestParams: EncodedRequestParams{
				FeeAmount: "123",
			},
		}

		err := r.ValidateFeeAmount()

		assert.Nil(t, err)
	})
}

func TestFindEncodedRouteRequest_ValidateChargeFeeBy(t *testing.T) {
	t.Parallel()

	t.Run("it should return nil when feeAmount is empty", func(t *testing.T) {
		r := &FindEncodedRouteRequest{
			EncodedRequestParams: EncodedRequestParams{
				FeeAmount:   "",
				ChargeFeeBy: "currency_in",
			},
		}

		err := r.ValidateChargeFeeBy()

		assert.Nil(t, err)
	})

	t.Run("it should return correct error when chargeFeeBy is invalid", func(t *testing.T) {
		expectedError := errors.NewDomainErrorInvalid(nil, "chargeFeeBy")

		r := &FindEncodedRouteRequest{
			EncodedRequestParams: EncodedRequestParams{
				FeeAmount:   "123",
				ChargeFeeBy: "invalid",
			},
		}

		err := r.ValidateChargeFeeBy()

		assert.Equal(t, expectedError, err)
	})

	t.Run("it should return nil when chargeFeeBy is valid", func(t *testing.T) {
		r := &FindEncodedRouteRequest{
			EncodedRequestParams: EncodedRequestParams{
				FeeAmount:   "123",
				ChargeFeeBy: "currency_in",
			},
		}

		err := r.ValidateChargeFeeBy()

		assert.Nil(t, err)
	})
}

func TestFindEncodedRouteRequest_ValidateSlippageTolerance(t *testing.T) {
	t.Parallel()

	t.Run("it should return nil when slippageTolerance is empty", func(t *testing.T) {
		r := &FindEncodedRouteRequest{
			EncodedRequestParams: EncodedRequestParams{
				SlippageTolerance: "",
			},
		}

		err := r.ValidateSlippageTolerance()

		assert.Nil(t, err)
	})

	t.Run("it should return nil when slippageTolerance is invalid", func(t *testing.T) {
		r := &FindEncodedRouteRequest{
			EncodedRequestParams: EncodedRequestParams{
				SlippageTolerance: "abc",
			},
		}

		err := r.ValidateSlippageTolerance()

		assert.Nil(t, err)
	})

	t.Run("it should return correct error when slippageTolerance is higher than max", func(t *testing.T) {
		expectedError := errors.NewDomainErrorOutOfRange(nil, "slippageTolerance")
		r := &FindEncodedRouteRequest{
			EncodedRequestParams: EncodedRequestParams{
				SlippageTolerance: "2001",
			},
		}

		err := r.ValidateSlippageTolerance()

		assert.Equal(t, expectedError, err)
	})

	t.Run("it should return nil when slippageTolerance is valid", func(t *testing.T) {
		r := &FindEncodedRouteRequest{
			EncodedRequestParams: EncodedRequestParams{
				SlippageTolerance: "1000",
			},
		}

		err := r.ValidateSlippageTolerance()

		assert.Nil(t, err)
	})
}
