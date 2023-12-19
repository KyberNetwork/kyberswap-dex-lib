package buildroute

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/pkg/logger"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type GasEstimator struct {
	gasEstimator    IEthereumGasEstimator
	gasRepository   IGasRepository
	priceRepository IPriceRepository
	GasTokenAddress string
}

type UnsignedTransaction struct {
	sender    string
	recipient string
	data      string
	value     *big.Int
	gasPrice  *big.Int
}

type IEthereumGasEstimator interface {
	EstimateGas(ctx context.Context, call ethereum.CallMsg) (uint64, error)
}

func NewGasEstimator(gasEstimator IEthereumGasEstimator, gasRepo IGasRepository,
	priceRepo IPriceRepository, gasToken string) *GasEstimator {
	return &GasEstimator{
		gasEstimator:    gasEstimator,
		gasRepository:   gasRepo,
		priceRepository: priceRepo,
		GasTokenAddress: gasToken,
	}
}

func (e *GasEstimator) EstimateGas(ctx context.Context, tx UnsignedTransaction) (uint64, error) {
	var (
		from             = common.HexToAddress(tx.sender)
		to               = common.HexToAddress(tx.recipient)
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
	if err != nil {
		logger.Infof("EstimateGas failed error %s", err)
	}
	metrics.IncrEstimateGas(err == nil, "allDexes")

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
	gasTokenPriceUSD, err := e.getPriceUSDByAddress(ctx, e.GasTokenAddress)
	if err != nil {
		return 0, 0.0, err
	}
	priceInUSD := utils.CalcGasUsd(gasPrice, int64(estimatedGas), gasTokenPriceUSD[e.GasTokenAddress])

	return estimatedGas, priceInUSD, nil
}

func (e *GasEstimator) getGasPrice(ctx context.Context) (*big.Float, error) {

	suggestedGasPrice, err := e.gasRepository.GetSuggestedGasPrice(ctx)
	if err != nil {
		return nil, err
	}

	return new(big.Float).SetInt(suggestedGasPrice), nil
}

func (u *GasEstimator) getPriceUSDByAddress(ctx context.Context, addresses ...string) (map[string]float64, error) {
	prices, err := u.priceRepository.FindByAddresses(ctx, addresses)
	if err != nil {
		return nil, err
	}

	priceUSDByAddress := make(map[string]float64, len(prices))
	for _, price := range prices {
		priceUSD, _ := price.GetPreferredPrice()

		priceUSDByAddress[price.Address] = priceUSD
	}

	return priceUSDByAddress, nil
}
