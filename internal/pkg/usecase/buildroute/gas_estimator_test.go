package buildroute_test

import (
	"context"
	"math/big"
	"testing"

	mocks "github.com/KyberNetwork/router-service/internal/pkg/mocks/usecase/buildroute"
	. "github.com/KyberNetwork/router-service/internal/pkg/usecase/buildroute"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestGasEstimator(t *testing.T) {
	mockError := errors.New("mock error")
	testCases := []struct {
		name        string
		input       UnsignedTransaction
		prepare     func(ctrl *gomock.Controller, tx UnsignedTransaction) *GasEstimator
		wantedError error
		wantedGas   uint64
	}{
		{
			name: "it should return correct gas",
			input: NewUnsignedTransaction(
				"0xc7198437980c041c805a1edcba50c1ce5db95118",
				"0xc7198437980c041c805a1edcba50c1ce5db95118",
				"0xc7198437980c041c805a1edcba50c1ce5db95118",
				big.NewInt(123),
				big.NewInt(123),
			),
			prepare: func(ctrl *gomock.Controller, tx UnsignedTransaction) *GasEstimator {
				ethEstimator := mocks.NewMockIEthereumGasEstimator(ctrl)
				ethEstimator.EXPECT().EstimateGas(gomock.Any(), gomock.Eq(ConvertTransactionToMsg(tx))).
					Return(uint64(123), nil).Times(1)
				return NewGasEstimator(ethEstimator)
			},
			wantedGas:   uint64(123),
			wantedError: nil,
		},
		{
			name: "it should return valid gas when sender address is empty",
			input: NewUnsignedTransaction(
				"",
				"0xc7198437980c041c805a1edcba50c1ce5db95118",
				"0xc7198437980c041c805a1edcba50c1ce5db95118",
				big.NewInt(123),
				big.NewInt(123),
			),
			prepare: func(ctrl *gomock.Controller, tx UnsignedTransaction) *GasEstimator {
				ethEstimator := mocks.NewMockIEthereumGasEstimator(ctrl)
				ethEstimator.EXPECT().EstimateGas(
					gomock.Any(), gomock.Eq(ConvertTransactionToMsg(tx))).
					Return(uint64(123), nil).Times(1)
				return NewGasEstimator(ethEstimator)
			},
			wantedGas:   uint64(123),
			wantedError: nil,
		},
		{
			name: "it should return error when repository return error",
			input: NewUnsignedTransaction(
				"0xc7198437980c041c805a1edcba50c1ce5db95118",
				"0xc7198437980c041c805a1edcba50c1ce5db95118",
				"0xc7198437980c041c805a1edcba50c1ce5db95118",
				big.NewInt(123),
				big.NewInt(123),
			),
			prepare: func(ctrl *gomock.Controller, tx UnsignedTransaction) *GasEstimator {
				ethEstimator := mocks.NewMockIEthereumGasEstimator(ctrl)
				ethEstimator.EXPECT().EstimateGas(
					gomock.Any(), gomock.Eq(ConvertTransactionToMsg(tx))).
					Return(uint64(0), mockError).Times(1)
				return NewGasEstimator(ethEstimator)
			},
			wantedGas:   0,
			wantedError: mockError,
		},
		{
			name: "it should return error when data is empty",
			input: NewUnsignedTransaction(
				"0xc7198437980c041c805a1edcba50c1ce5db95118",
				"0xc7198437980c041c805a1edcba50c1ce5db95118",
				"",
				big.NewInt(123),
				big.NewInt(123),
			),
			prepare: func(ctrl *gomock.Controller, tx UnsignedTransaction) *GasEstimator {
				ethEstimator := mocks.NewMockIEthereumGasEstimator(ctrl)
				ethEstimator.EXPECT().EstimateGas(
					gomock.Any(), gomock.Eq(ConvertTransactionToMsg(tx))).Times(0)
				return NewGasEstimator(ethEstimator)
			},
			wantedGas:   0,
			wantedError: errors.New("empty hex string"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			gasEstimator := tc.prepare(ctrl, tc.input)
			result, err := gasEstimator.Execute(context.Background(), tc.input)

			assert.Equal(t, tc.wantedGas, result)
			if err != nil {
				assert.EqualErrorf(t, err, tc.wantedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

		})
	}
}
