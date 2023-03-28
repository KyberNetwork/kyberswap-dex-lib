package getroute

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func Test_calcAmountOutAfterFee(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name              string
		amountOut         *big.Int
		extraFee          valueobject.ExtraFee
		expectedAmountOut *big.Int
		expectedError     error
	}{
		{
			name:      "it should return correct amountOut when chargeFeeBy currency_out and isInBps false",
			amountOut: big.NewInt(100000),
			extraFee: valueobject.ExtraFee{
				ChargeFeeBy: valueobject.ChargeFeeByCurrencyOut,
				IsInBps:     false,
				FeeAmount:   big.NewInt(100),
			},
			expectedAmountOut: big.NewInt(99900),
			expectedError:     nil,
		},
		{
			name:      "it should return correct amountOut when chargeFeeBy currency_out and isInBps true",
			amountOut: big.NewInt(100000),
			extraFee: valueobject.ExtraFee{
				ChargeFeeBy: valueobject.ChargeFeeByCurrencyOut,
				IsInBps:     true,
				FeeAmount:   big.NewInt(10),
			},
			expectedAmountOut: big.NewInt(99900),
			expectedError:     nil,
		},
		{
			name:      "it should return correct amountOut when chargeFeeBy currency_in",
			amountOut: big.NewInt(100000),
			extraFee: valueobject.ExtraFee{
				ChargeFeeBy: valueobject.ChargeFeeByCurrencyIn,
				IsInBps:     true,
				FeeAmount:   big.NewInt(10),
			},
			expectedAmountOut: big.NewInt(100000),
			expectedError:     nil,
		},
		{
			name:      "it should return ErrFeeAmountIsGreaterThanAmountOut",
			amountOut: big.NewInt(100000),
			extraFee: valueobject.ExtraFee{
				ChargeFeeBy: valueobject.ChargeFeeByCurrencyOut,
				IsInBps:     true,
				FeeAmount:   big.NewInt(10001),
			},
			expectedAmountOut: nil,
			expectedError:     usecase.ErrFeeAmountIsGreaterThanAmountOut,
		},
		{
			name:      "it should return ErrFeeAmountIsGreaterThanAmountOut",
			amountOut: big.NewInt(100000),
			extraFee: valueobject.ExtraFee{
				ChargeFeeBy: valueobject.ChargeFeeByCurrencyOut,
				IsInBps:     false,
				FeeAmount:   big.NewInt(100001),
			},
			expectedAmountOut: nil,
			expectedError:     usecase.ErrFeeAmountIsGreaterThanAmountOut,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			amountOut, err := calcAmountOutAfterFee(tc.amountOut, tc.extraFee)

			assert.ErrorIs(t, err, tc.expectedError)
			assert.Equal(t, tc.expectedAmountOut, amountOut)
		})
	}
}
