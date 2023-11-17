package uniswapv2

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
)

type (
	IFeeTracker interface {
		GetFee(
			ctx context.Context,
			poolAddress string,
			factoryAddress string,
			blockNumber *big.Int,
		) (uint64, error)
	}

	// MDexFeeTracker gets fee from factory contract `getPairFees`
	MDexFeeTracker struct {
		ethrpcClient *ethrpc.Client
	}

	// MMFFeeTracker gets fee from pair contract `swapFee`
	MMFFeeTracker struct {
		ethrpcClient *ethrpc.Client
	}

	// ShibaswapFeeTracker gets fee from pair contract `totalFee`
	ShibaswapFeeTracker struct {
		ethrpcClient *ethrpc.Client
	}

	// DefiSwapFeeTracker gets fee from factory contract `totalFeeBasisPoint`
	DefiSwapFeeTracker struct {
		ethrpcClient *ethrpc.Client
	}
)

func (t *MDexFeeTracker) GetFee(
	ctx context.Context,
	poolAddress string,
	factoryAddress string,
	blockNumber *big.Int,
) (uint64, error) {
	var fee *big.Int

	getFeeRequest := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)

	getFeeRequest.AddCall(&ethrpc.Call{
		ABI:    mdexFactoryABI,
		Target: factoryAddress,
		Method: mdexFactoryMethodGetPairFees,
		Params: []interface{}{common.HexToAddress(poolAddress)},
	}, []interface{}{&fee})

	_, err := getFeeRequest.Call()
	if err != nil {
		return 0, err
	}

	return fee.Uint64(), nil
}

func (t *MMFFeeTracker) GetFee(
	ctx context.Context,
	poolAddress string,
	_ string,
	blockNumber *big.Int,
) (uint64, error) {
	var fee uint32

	getFeeRequest := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)

	getFeeRequest.AddCall(&ethrpc.Call{
		ABI:    meerkatPairABI,
		Target: poolAddress,
		Method: meerkatPairMethodSwapFee,
		Params: nil,
	}, []interface{}{&fee})

	_, err := getFeeRequest.Call()
	if err != nil {
		return 0, err
	}

	return uint64(fee), nil
}

func (t *ShibaswapFeeTracker) GetFee(
	ctx context.Context,
	poolAddress string,
	_ string,
	blockNumber *big.Int,
) (uint64, error) {
	var fee *big.Int

	getFeeRequest := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)

	getFeeRequest.AddCall(&ethrpc.Call{
		ABI:    shibaswapPairABI,
		Target: poolAddress,
		Method: shibaswapPairMethodTotalFee,
		Params: nil,
	}, []interface{}{&fee})

	_, err := getFeeRequest.Call()
	if err != nil {
		return 0, err
	}

	return fee.Uint64(), nil
}

func (t *DefiSwapFeeTracker) GetFee(
	ctx context.Context,
	_ string,
	factoryAddress string,
	blockNumber *big.Int,
) (uint64, error) {
	var fee *big.Int

	getFeeRequest := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)

	getFeeRequest.AddCall(&ethrpc.Call{
		ABI:    croDefiSwapFactoryABI,
		Target: factoryAddress,
		Method: croDefiSwapFactoryMethodTotalFeeBasisPoint,
		Params: nil,
	}, []interface{}{&fee})

	_, err := getFeeRequest.Call()
	if err != nil {
		return 0, err
	}

	return fee.Uint64(), nil
}
