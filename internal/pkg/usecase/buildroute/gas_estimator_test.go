package buildroute_test

import (
	"context"
	"math/big"
	"testing"

	routerEntities "github.com/KyberNetwork/router-service/internal/pkg/entity"
	mocks "github.com/KyberNetwork/router-service/internal/pkg/mocks/usecase/buildroute"
	. "github.com/KyberNetwork/router-service/internal/pkg/usecase/buildroute"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestGasEstimator(t *testing.T) {
	mockError := errors.New("mock error")
	testCases := []struct {
		name         string
		input        UnsignedTransaction
		prepare      func(ctrl *gomock.Controller, tx UnsignedTransaction) *GasEstimator
		wantedError  error
		wantedGas    uint64
		wantedGasUSD float64
	}{
		{
			name: "it should return correct gas",
			input: NewUnsignedTransaction(
				"0xc7198437980c041c805a1edcba50c1ce5db95118",
				"0x6131B5fae19EA4f9D964eAc0408E4408b66337b5",
				"0xc7198437980c041c805a1edcba50c1ce5db95118",
				big.NewInt(123),
				big.NewInt(123),
			),
			prepare: func(ctrl *gomock.Controller, tx UnsignedTransaction) *GasEstimator {
				ethEstimator := mocks.NewMockIEthereumGasEstimator(ctrl)
				routerAddress := "0x6131B5fae19EA4f9D964eAc0408E4408b66337b5"
				ethEstimator.EXPECT().EstimateGas(gomock.Any(), gomock.Eq(ConvertTransactionToMsg(tx, routerAddress))).
					Return(uint64(123), nil).Times(1)
				gasRep := mocks.NewMockIGasRepository(ctrl)
				gasRep.EXPECT().GetSuggestedGasPrice(gomock.Any()).Return(big.NewInt(2), nil)
				priceTokenAddress := "0xc7198437980c041c805a1edcba50c1ce5db95118"
				prices := map[string]*routerEntities.OnchainPrice{
					priceTokenAddress: {
						USDPrice: routerEntities.Price{Sell: big.NewFloat(0.5), Buy: big.NewFloat(0.5)},
					},
					"0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": {
						USDPrice: routerEntities.Price{Sell: big.NewFloat(1), Buy: big.NewFloat(1)},
					}}
				onchainPriceRepo := mocks.NewMockIOnchainPriceRepository(ctrl)
				onchainPriceRepo.EXPECT().FindByAddresses(gomock.Any(), []string{priceTokenAddress}).Return(prices, nil)
				return NewGasEstimator(ethEstimator, gasRep, onchainPriceRepo, priceTokenAddress, routerAddress)
			},
			wantedGas:    uint64(123),
			wantedGasUSD: utils.CalcGasUsd(big.NewFloat(2), int64(123), 0.5),
			wantedError:  nil,
		},
		{
			name: "it should return valid gas when sender address is empty",
			input: NewUnsignedTransaction(
				"",
				"0x6131B5fae19EA4f9D964eAc0408E4408b66337b5",
				"0xc7198437980c041c805a1edcba50c1ce5db95118",
				big.NewInt(123),
				big.NewInt(123),
			),
			prepare: func(ctrl *gomock.Controller, tx UnsignedTransaction) *GasEstimator {
				ethEstimator := mocks.NewMockIEthereumGasEstimator(ctrl)
				routerAddress := "0x6131B5fae19EA4f9D964eAc0408E4408b66337b5"
				ethEstimator.EXPECT().EstimateGas(
					gomock.Any(), gomock.Eq(ConvertTransactionToMsg(tx, routerAddress))).
					Return(uint64(123), nil).Times(1)
				gasRep := mocks.NewMockIGasRepository(ctrl)
				gasRep.EXPECT().GetSuggestedGasPrice(gomock.Any()).Return(big.NewInt(2), nil)
				priceTokenAddress := "0xc7198437980c041c805a1edcba50c1ce5db95118"
				prices := map[string]*routerEntities.OnchainPrice{
					priceTokenAddress: {
						USDPrice: routerEntities.Price{Sell: big.NewFloat(0.5), Buy: big.NewFloat(0.5)},
					},
					"0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": {
						USDPrice: routerEntities.Price{Sell: big.NewFloat(1), Buy: big.NewFloat(1)},
					}}
				onchainPriceRepo := mocks.NewMockIOnchainPriceRepository(ctrl)
				onchainPriceRepo.EXPECT().FindByAddresses(gomock.Any(), []string{priceTokenAddress}).Return(prices, nil)
				return NewGasEstimator(ethEstimator, gasRep, onchainPriceRepo, priceTokenAddress, routerAddress)
			},
			wantedGas:    uint64(123),
			wantedGasUSD: utils.CalcGasUsd(big.NewFloat(2), int64(123), 0.5),
			wantedError:  nil,
		},
		{
			name: "it should return error when repository return error",
			input: NewUnsignedTransaction(
				"0xc7198437980c041c805a1edcba50c1ce5db95118",
				"0x6131B5fae19EA4f9D964eAc0408E4408b66337b5",
				"0xc7198437980c041c805a1edcba50c1ce5db95118",
				big.NewInt(123),
				big.NewInt(123),
			),
			prepare: func(ctrl *gomock.Controller, tx UnsignedTransaction) *GasEstimator {
				ethEstimator := mocks.NewMockIEthereumGasEstimator(ctrl)
				routerAddress := "0x6131B5fae19EA4f9D964eAc0408E4408b66337b5"
				ethEstimator.EXPECT().EstimateGas(
					gomock.Any(), gomock.Eq(ConvertTransactionToMsg(tx, routerAddress))).
					Return(uint64(0), mockError).Times(1)
				gasRep := mocks.NewMockIGasRepository(ctrl)
				gasRep.EXPECT().GetSuggestedGasPrice(gomock.Any()).Times(0)
				onchainPriceRepo := mocks.NewMockIOnchainPriceRepository(ctrl)
				onchainPriceRepo.EXPECT().FindByAddresses(gomock.Any(), gomock.Any()).Times(0)
				return NewGasEstimator(ethEstimator, gasRep, onchainPriceRepo, "0xc7198437980c041c805a1edcba50c1ce5db95118", routerAddress)
			},
			wantedGas:    0,
			wantedGasUSD: 0.0,
			wantedError:  mockError,
		},
		{
			name: "it should return error when data is empty",
			input: NewUnsignedTransaction(
				"0xc7198437980c041c805a1edcba50c1ce5db95118",
				"0x6131B5fae19EA4f9D964eAc0408E4408b66337b5",
				"",
				big.NewInt(123),
				big.NewInt(123),
			),
			prepare: func(ctrl *gomock.Controller, tx UnsignedTransaction) *GasEstimator {
				routerAddress := "0x6131B5fae19EA4f9D964eAc0408E4408b66337b5"
				ethEstimator := mocks.NewMockIEthereumGasEstimator(ctrl)
				ethEstimator.EXPECT().EstimateGas(
					gomock.Any(), gomock.Eq(ConvertTransactionToMsg(tx, routerAddress))).Times(0)
				gasRep := mocks.NewMockIGasRepository(ctrl)
				gasRep.EXPECT().GetSuggestedGasPrice(gomock.Any()).Times(0)
				return NewGasEstimator(ethEstimator, gasRep, nil, "0xc7198437980c041c805a1edcba50c1ce5db95118", routerAddress)
			},
			wantedGas:   0,
			wantedError: errors.New("empty hex string"),
		},
		{
			name: "it should return error when get gas price failed",
			input: NewUnsignedTransaction(
				"0xc7198437980c041c805a1edcba50c1ce5db95118",
				"0x6131B5fae19EA4f9D964eAc0408E4408b66337b5",
				"0xc7198437980c041c805a1edcba50c1ce5db95118",
				big.NewInt(123),
				big.NewInt(123),
			),
			prepare: func(ctrl *gomock.Controller, tx UnsignedTransaction) *GasEstimator {
				ethEstimator := mocks.NewMockIEthereumGasEstimator(ctrl)
				routerAddress := "0x6131B5fae19EA4f9D964eAc0408E4408b66337b5"
				ethEstimator.EXPECT().EstimateGas(gomock.Any(), gomock.Eq(ConvertTransactionToMsg(tx, routerAddress))).
					Return(uint64(123), nil).Times(1)
				gasRep := mocks.NewMockIGasRepository(ctrl)
				gasRep.EXPECT().GetSuggestedGasPrice(gomock.Any()).Return(nil, mockError)
				return NewGasEstimator(ethEstimator, gasRep, nil, "0xc7198437980c041c805a1edcba50c1ce5db95118", routerAddress)
			},
			wantedGas:    0,
			wantedGasUSD: 0.0,
			wantedError:  mockError,
		},
		{
			name: "it should return error when get gas token price failed",
			input: NewUnsignedTransaction(
				"0xc7198437980c041c805a1edcba50c1ce5db95118",
				"0x6131B5fae19EA4f9D964eAc0408E4408b66337b5",
				"0xc7198437980c041c805a1edcba50c1ce5db95118",
				big.NewInt(123),
				big.NewInt(123),
			),
			prepare: func(ctrl *gomock.Controller, tx UnsignedTransaction) *GasEstimator {
				ethEstimator := mocks.NewMockIEthereumGasEstimator(ctrl)
				routerAddress := "0x6131B5fae19EA4f9D964eAc0408E4408b66337b5"
				ethEstimator.EXPECT().EstimateGas(gomock.Any(), gomock.Eq(ConvertTransactionToMsg(tx, routerAddress))).
					Return(uint64(123), nil).Times(1)
				gasRep := mocks.NewMockIGasRepository(ctrl)
				gasRep.EXPECT().GetSuggestedGasPrice(gomock.Any()).Return(big.NewInt(2), nil)
				priceTokenAddress := "0xc7198437980c041c805a1edcba50c1ce5db95118"
				onchainPriceRepo := mocks.NewMockIOnchainPriceRepository(ctrl)
				onchainPriceRepo.EXPECT().FindByAddresses(gomock.Any(), []string{priceTokenAddress}).Return(nil, mockError)
				return NewGasEstimator(ethEstimator, gasRep, onchainPriceRepo, priceTokenAddress, routerAddress)
			},
			wantedGas:    0,
			wantedGasUSD: 0.0,
			wantedError:  mockError,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			gasEstimator := tc.prepare(ctrl, tc.input)
			gas, gasUSD, err := gasEstimator.Execute(context.Background(), tc.input)

			assert.Equal(t, tc.wantedGas, gas)
			assert.Equal(t, tc.wantedGasUSD, gasUSD)
			if err != nil {
				assert.EqualErrorf(t, err, tc.wantedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

		})
	}
}
