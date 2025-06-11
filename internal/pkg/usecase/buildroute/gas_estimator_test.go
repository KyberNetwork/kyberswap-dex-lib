package buildroute_test

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	routerEntities "github.com/KyberNetwork/router-service/internal/pkg/entity"
	mocks "github.com/KyberNetwork/router-service/internal/pkg/mocks/usecase/buildroute"
	. "github.com/KyberNetwork/router-service/internal/pkg/usecase/buildroute"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

const (
	routerABI = `[{
		"inputs": [{
			"components": [
				{"internalType": "address", "name": "callTarget", "type": "address"},
				{"internalType": "address", "name": "approveTarget", "type": "address"},
				{"internalType": "bytes", "name": "targetData", "type": "bytes"},
				{"components": [
					{"internalType": "contract IERC20", "name": "srcToken", "type": "address"},
					{"internalType": "contract IERC20", "name": "dstToken", "type": "address"},
					{"internalType": "address", "name": "dstReceiver", "type": "address"},
					{"internalType": "uint256", "name": "amount", "type": "uint256"},
					{"internalType": "uint256", "name": "minReturnAmount", "type": "uint256"},
					{"internalType": "uint256", "name": "flags", "type": "uint256"}
				], "internalType": "struct MetaAggregationRouterV2.SwapDescriptionV2", "name": "desc", "type": "tuple"},
				{"internalType": "bytes", "name": "clientData", "type": "bytes"}
			], "internalType": "struct MetaAggregationRouterV2.SwapExecutionParams", "name": "execution", "type": "tuple"}
		],
		"name": "swap",
		"outputs": [
			{"internalType": "uint256", "name": "returnAmount", "type": "uint256"},
			{"internalType": "uint256", "name": "gasUsed", "type": "uint256"}
		],
		"stateMutability": "payable",
		"type": "function"
	}]`
)

func TestGasEstimator(t *testing.T) {
	mockError := errors.New("mock error")
	parsedABI, err := abi.JSON(strings.NewReader(routerABI))
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name               string
		input              UnsignedTransaction
		prepare            func(ctrl *gomock.Controller, tx UnsignedTransaction) *GasEstimator
		wantedError        error
		wantedGas          uint64
		wantedReturnAmount *big.Int
		wantedGasUSD       float64
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
				ethClient := mocks.NewMockIETHClient(ctrl)
				routerAddress := "0x6131B5fae19EA4f9D964eAc0408E4408b66337b5"

				returnAmount := big.NewInt(123)
				gasUsed := big.NewInt(123)

				packedData, err := parsedABI.Methods["swap"].Outputs.Pack(returnAmount, gasUsed)
				if err != nil {
					t.Fatal(err)
				}

				ethClient.EXPECT().CallContract(gomock.Any(), gomock.Any(), gomock.Any()).Return(packedData, nil).Times(1)
				gasRep := mocks.NewMockIGasRepository(ctrl)
				gasRep.EXPECT().GetSuggestedGasPrice(gomock.Any()).Return(big.NewInt(2), nil).Times(1)
				chainId := valueobject.ChainIDEthereum
				prices := map[string]*routerEntities.OnchainPrice{
					"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": {
						USDPrice: routerEntities.Price{Sell: big.NewFloat(0.5), Buy: big.NewFloat(0.5)},
					},
					"0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": {
						USDPrice: routerEntities.Price{Sell: big.NewFloat(1), Buy: big.NewFloat(1)},
					}}
				onchainPriceRepo := mocks.NewMockIOnchainPriceRepository(ctrl)
				onchainPriceRepo.EXPECT().FindByAddresses(gomock.Any(), []string{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"}).Return(prices, nil).Times(1)
				return NewGasEstimator(ethClient, gasRep, onchainPriceRepo, chainId, routerAddress)
			},
			wantedGas:          EstimateTotalGas(123),
			wantedReturnAmount: big.NewInt(123),
			wantedGasUSD:       utils.CalcGasUsd(big.NewFloat(2), int64(EstimateTotalGas(123)), 0.5),
			wantedError:        nil,
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
				ethClient := mocks.NewMockIETHClient(ctrl)
				routerAddress := "0x6131B5fae19EA4f9D964eAc0408E4408b66337b5"

				returnAmount := big.NewInt(123)
				gasUsed := big.NewInt(123)

				packedData, err := parsedABI.Methods["swap"].Outputs.Pack(returnAmount, gasUsed)
				if err != nil {
					t.Fatal(err)
				}

				ethClient.EXPECT().CallContract(gomock.Any(), gomock.Any(), gomock.Any()).Return(packedData, nil).Times(1)
				gasRep := mocks.NewMockIGasRepository(ctrl)
				gasRep.EXPECT().GetSuggestedGasPrice(gomock.Any()).Return(big.NewInt(2), nil).Times(1)
				chainId := valueobject.ChainIDEthereum
				prices := map[string]*routerEntities.OnchainPrice{
					"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": {
						USDPrice: routerEntities.Price{Sell: big.NewFloat(0.5), Buy: big.NewFloat(0.5)},
					},
					"0xc3d088842dcf02c13699f936bb83dfbbc6f721ab": {
						USDPrice: routerEntities.Price{Sell: big.NewFloat(1), Buy: big.NewFloat(1)},
					}}
				onchainPriceRepo := mocks.NewMockIOnchainPriceRepository(ctrl)
				onchainPriceRepo.EXPECT().FindByAddresses(gomock.Any(), []string{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"}).Return(prices, nil).Times(1)
				return NewGasEstimator(ethClient, gasRep, onchainPriceRepo, chainId, routerAddress)
			},
			wantedGas:          EstimateTotalGas(123),
			wantedReturnAmount: big.NewInt(123),
			wantedGasUSD:       utils.CalcGasUsd(big.NewFloat(2), int64(EstimateTotalGas(123)), 0.5),
			wantedError:        nil,
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
				ethClient := mocks.NewMockIETHClient(ctrl)
				routerAddress := "0x6131B5fae19EA4f9D964eAc0408E4408b66337b5"

				returnAmount := big.NewInt(123)
				gasUsed := big.NewInt(123)

				packedData, err := parsedABI.Methods["swap"].Outputs.Pack(returnAmount, gasUsed)
				if err != nil {
					t.Fatal(err)
				}

				ethClient.EXPECT().CallContract(gomock.Any(), gomock.Any(), gomock.Any()).Return(packedData, nil).Times(1)
				gasRep := mocks.NewMockIGasRepository(ctrl)
				gasRep.EXPECT().GetSuggestedGasPrice(gomock.Any()).Return(nil, mockError).Times(1)
				onchainPriceRepo := mocks.NewMockIOnchainPriceRepository(ctrl)
				onchainPriceRepo.EXPECT().FindByAddresses(gomock.Any(), gomock.Any()).Return(nil, mockError).Times(0)
				return NewGasEstimator(ethClient, gasRep, onchainPriceRepo, valueobject.ChainIDEthereum, routerAddress)
			},
			wantedGas:          0,
			wantedReturnAmount: nil,
			wantedGasUSD:       0.0,
			wantedError:        mockError,
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
				ethClient := mocks.NewMockIETHClient(ctrl)
				ethClient.EXPECT().CallContract(gomock.Any(), gomock.Any(), gomock.Any()).Return([]byte{}, nil).Times(0)
				gasRep := mocks.NewMockIGasRepository(ctrl)
				gasRep.EXPECT().GetSuggestedGasPrice(gomock.Any()).Times(0)
				onchainPriceRepo := mocks.NewMockIOnchainPriceRepository(ctrl)
				return NewGasEstimator(ethClient, gasRep, onchainPriceRepo, valueobject.ChainIDEthereum, routerAddress)
			},
			wantedGas:          0,
			wantedReturnAmount: nil,
			wantedGasUSD:       0.0,
			wantedError:        errors.New("empty hex string"),
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
				ethClient := mocks.NewMockIETHClient(ctrl)
				routerAddress := "0x6131B5fae19EA4f9D964eAc0408E4408b66337b5"

				returnAmount := big.NewInt(123)
				gasUsed := big.NewInt(123)

				packedData, err := parsedABI.Methods["swap"].Outputs.Pack(returnAmount, gasUsed)
				if err != nil {
					t.Fatal(err)
				}

				ethClient.EXPECT().CallContract(gomock.Any(), gomock.Any(), gomock.Any()).Return(packedData, nil).Times(1)
				gasRep := mocks.NewMockIGasRepository(ctrl)
				gasRep.EXPECT().GetSuggestedGasPrice(gomock.Any()).Return(nil, mockError)
				onchainPriceRepo := mocks.NewMockIOnchainPriceRepository(ctrl)
				return NewGasEstimator(ethClient, gasRep, onchainPriceRepo, valueobject.ChainIDEthereum, routerAddress)
			},
			wantedGas:          0,
			wantedReturnAmount: nil,
			wantedGasUSD:       0.0,
			wantedError:        mockError,
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
				ethClient := mocks.NewMockIETHClient(ctrl)
				routerAddress := "0x6131B5fae19EA4f9D964eAc0408E4408b66337b5"

				returnAmount := big.NewInt(123)
				gasUsed := big.NewInt(123)

				packedData, err := parsedABI.Methods["swap"].Outputs.Pack(returnAmount, gasUsed)
				if err != nil {
					t.Fatal(err)
				}

				ethClient.EXPECT().CallContract(gomock.Any(), gomock.Any(), gomock.Any()).Return(packedData, nil).Times(1)
				gasRep := mocks.NewMockIGasRepository(ctrl)
				gasRep.EXPECT().GetSuggestedGasPrice(gomock.Any()).Return(big.NewInt(2), nil)
				onchainPriceRepo := mocks.NewMockIOnchainPriceRepository(ctrl)
				onchainPriceRepo.EXPECT().FindByAddresses(gomock.Any(), []string{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"}).Return(nil, mockError)
				return NewGasEstimator(ethClient, gasRep, onchainPriceRepo, valueobject.ChainIDEthereum, routerAddress)
			},
			wantedGas:          0,
			wantedReturnAmount: nil,
			wantedGasUSD:       0.0,
			wantedError:        mockError,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			gasEstimator := tc.prepare(ctrl, tc.input)
			gas, gasUSD, returnAmount, err := gasEstimator.EstimateGasAndPriceUSD(context.Background(), tc.input)

			assert.Equal(t, tc.wantedGas, gas)
			assert.Equal(t, tc.wantedReturnAmount, returnAmount)
			assert.Equal(t, tc.wantedGasUSD, gasUSD)
			if err != nil {
				assert.EqualErrorf(t, err, tc.wantedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
