package buildroute

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type GasEstimator struct {
	gasEstimator IEthereumGasEstimator
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

func NewGasEstimator(gasEstimator IEthereumGasEstimator) *GasEstimator {
	return &GasEstimator{
		gasEstimator: gasEstimator,
	}
}

func (e *GasEstimator) Execute(ctx context.Context, tx UnsignedTransaction) (uint64, error) {
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
		return 0, err
	}

	return estimatedGas, nil
}
