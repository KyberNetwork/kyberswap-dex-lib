package velodromev1

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
			isStable bool,
			factoryAddress string,
			blockNumber *big.Int,
		) (uint64, error)
	}

	// VelodromeFeeTracker gets fee from factory contract `getFee(bool _stable)`
	VelodromeFeeTracker struct {
		ethrpcClient *ethrpc.Client
	}

	// StratumFeeTracker gets fee from factory contract `getFee(address pool)`
	StratumFeeTracker struct {
		ethrpcClient *ethrpc.Client
	}

	// NuriFeeTracker gets fee from factory contract `pairFee(address pool)`
	NuriFeeTracker struct {
		ethrpcClient *ethrpc.Client
	}

	// LyveFeeTracker gets fee from factory contract `getFee(address pool, bool _stable)`
	LyveFeeTracker struct {
		ethrpcClient *ethrpc.Client
	}
)

func (t *VelodromeFeeTracker) GetFee(
	ctx context.Context,
	_ string,
	isStable bool,
	factoryAddress string,
	blockNumber *big.Int,
) (uint64, error) {
	var fee uint16

	getFeeRequest := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)

	getFeeRequest.AddCall(&ethrpc.Call{
		ABI:    pairFactoryABI,
		Target: factoryAddress,
		Method: pairFactoryMethodGetFee,
		Params: []interface{}{isStable},
	}, []interface{}{&fee})

	_, err := getFeeRequest.Call()
	if err != nil {
		return 0, err
	}

	return uint64(fee), nil
}

func (t *StratumFeeTracker) GetFee(
	ctx context.Context,
	poolAddress string,
	isStable bool,
	factoryAddress string,
	blockNumber *big.Int,
) (uint64, error) {
	var fee *big.Int

	getFeeRequest := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)

	getFeeRequest.AddCall(&ethrpc.Call{
		ABI:    stratumPairFactoryABI,
		Target: factoryAddress,
		Method: stratumPairFactoryMethodGetFee,
		Params: []interface{}{common.HexToAddress(poolAddress)},
	}, []interface{}{&fee})

	_, err := getFeeRequest.Call()
	if err != nil {
		return 0, err
	}

	return fee.Uint64(), nil
}

func (t *NuriFeeTracker) GetFee(
	ctx context.Context,
	poolAddress string,
	isStable bool,
	factoryAddress string,
	blockNumber *big.Int,
) (uint64, error) {
	var fee *big.Int

	getFeeRequest := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)

	getFeeRequest.AddCall(&ethrpc.Call{
		ABI:    nuriPairFactoryABI,
		Target: factoryAddress,
		Method: nuriPairFactoryMethodPairFee,
		Params: []interface{}{common.HexToAddress(poolAddress)},
	}, []interface{}{&fee})

	_, err := getFeeRequest.Call()
	if err != nil {
		return 0, err
	}

	return fee.Uint64(), nil
}

func (t *LyveFeeTracker) GetFee(
	ctx context.Context,
	poolAddress string,
	isStable bool,
	factoryAddress string,
	blockNumber *big.Int,
) (uint64, error) {
	var fee uint16

	getFeeRequest := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)

	getFeeRequest.AddCall(&ethrpc.Call{
		ABI:    lyvePairFactoryABI,
		Target: factoryAddress,
		Method: lyvePairFactoryMethodGetFee,
		Params: []interface{}{common.HexToAddress(poolAddress), isStable},
	}, []interface{}{&fee})

	_, err := getFeeRequest.Call()
	if err != nil {
		return 0, err
	}

	return uint64(fee), nil
}
