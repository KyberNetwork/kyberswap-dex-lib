package buildroute

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/clientid"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type GasEstimator struct {
	gasEstimator    IEthereumGasEstimator
	gasRepository   IGasRepository
	gasTokenAddress string
	routerAddress   string

	onchainpriceRepository IOnchainPriceRepository
}

type UnsignedTransaction struct {
	sender   string
	data     string
	value    *big.Int
	gasPrice *big.Int
}

//go:generate go run go.uber.org/mock/mockgen -destination ../../mocks/usecase/buildroute/ethereum_gas_estimator.go -package buildroute github.com/KyberNetwork/router-service/internal/pkg/usecase/buildroute IEthereumGasEstimator
type IEthereumGasEstimator interface {
	EstimateGas(ctx context.Context, call ethereum.CallMsg) (uint64, error)
}

func NewGasEstimator(
	gasEstimator IEthereumGasEstimator,
	gasRepo IGasRepository,
	onchainpriceRepository IOnchainPriceRepository,
	gasToken string,
	routerAddress string,
) *GasEstimator {
	return &GasEstimator{
		gasEstimator:    gasEstimator,
		gasRepository:   gasRepo,
		gasTokenAddress: gasToken,
		routerAddress:   routerAddress,

		onchainpriceRepository: onchainpriceRepository,
	}
}

func (e *GasEstimator) EstimateGas(ctx context.Context, tx UnsignedTransaction) (uint64, error) {
	var (
		from             = common.HexToAddress(tx.sender)
		to               = common.HexToAddress(e.routerAddress)
		encodedData, err = hexutil.Decode(tx.data)
	)
	// We still return error incase data is empty because in router service, every transaction must contain data payload
	if err != nil {
		return 0, err
	}

	estimatedGas, err := e.gasEstimator.EstimateGas(ctx, ethereum.CallMsg{
		From:       from,
		To:         &to,
		Gas:        0,
		GasPrice:   tx.gasPrice,
		GasFeeCap:  nil,
		GasTipCap:  nil,
		Value:      tx.value,
		Data:       encodedData,
		AccessList: nil,
	})
	clientid := clientid.GetClientIDFromCtx(ctx)
	metrics.CountEstimateGas(ctx, err == nil, "allDexes", clientid)

	return estimatedGas, err
}

func (e *GasEstimator) Execute(ctx context.Context, tx UnsignedTransaction) (uint64, float64, error) {
	estimatedGas, err := e.EstimateGas(ctx, tx)
	if err != nil {
		return 0, 0.0, err
	}
	gasPrice, err := e.getGasPrice(ctx)
	if err != nil {
		return 0, 0.0, err
	}
	gasTokenPriceUSD, err := e.GetGasTokenPriceUSD(ctx)
	if err != nil {
		return 0, 0.0, err
	}
	priceInUSD := utils.CalcGasUsd(gasPrice, int64(estimatedGas), gasTokenPriceUSD)

	return estimatedGas, priceInUSD, nil
}

func (e *GasEstimator) GetGasTokenPriceUSD(ctx context.Context) (float64, error) {
	priceByAddress, err := e.onchainpriceRepository.FindByAddresses(ctx, []string{e.gasTokenAddress})
	if err != nil {
		return 0, err
	}
	// use buy price for gas
	if price, ok := priceByAddress[e.gasTokenAddress]; ok && price != nil && price.USDPrice.Buy != nil {
		gasTokenPriceUSD, _ := price.USDPrice.Buy.Float64()
		return gasTokenPriceUSD, nil
	}
	return 0, nil
}

func (e *GasEstimator) getGasPrice(ctx context.Context) (*big.Float, error) {

	suggestedGasPrice, err := e.gasRepository.GetSuggestedGasPrice(ctx)
	if err != nil {
		return nil, err
	}

	return new(big.Float).SetInt(suggestedGasPrice), nil
}
