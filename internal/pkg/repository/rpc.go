package repository

import (
	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	"github.com/KyberNetwork/router-service/pkg/logger"

	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	ErrNoConfiguredPRC = errors.New("no configured rpc")
	ErrNoAvailableRPC  = errors.New("no available rpc")
)

type CallParams struct {
	ABI    abi.ABI
	Target string
	Method string
	Params []interface{}
	Output interface{}
}

func CallParamsFactory(abi abi.ABI, address string) func(callMethod string, output interface{}, params []interface{}) *CallParams {
	return func(callMethod string, output interface{}, params []interface{}) *CallParams {
		return &CallParams{
			ABI:    abi,
			Target: address,
			Method: callMethod,
			Output: output,
			Params: params,
		}
	}
}

func TryCallParamsFactory(abi abi.ABI, address string) func(callMethod string, output interface{}, params []interface{}) *TryCallParams {
	return func(callMethod string, output interface{}, params []interface{}) *TryCallParams {
		return &TryCallParams{
			ABI:    abi,
			Target: address,
			Method: callMethod,
			Output: output,
			Params: params,
		}
	}
}

type TryCallParams struct {
	ABI     abi.ABI
	Target  string
	Method  string
	Params  []interface{}
	Output  interface{}
	Success *bool
}

type TryCallUnPackParams struct {
	ABI       abi.ABI
	UnpackABI []abi.ABI
	Target    string
	Method    string
	Params    []interface{}
	Output    []interface{}
	Success   *bool
}

type multiCallParam struct {
	Target   common.Address
	CallData []byte
}

type RPCRepository struct {
	client *ethclient.Client
	config RPCRepositoryConfig
}

func NewRPCRepository(
	client *ethclient.Client,
	config RPCRepositoryConfig,
) *RPCRepository {
	return &RPCRepository{
		client: client,
		config: config,
	}
}

func (r *RPCRepository) GetLatestBlockTimestamp(ctx context.Context) (uint64, error) {
	latestBlockNumber, err := r.client.BlockNumber(ctx)
	if err != nil {
		return 0, err
	}

	block, err := r.client.BlockByNumber(ctx, big.NewInt(int64(latestBlockNumber)))
	if err != nil {
		return 0, err
	}

	return block.Time(), nil
}

func (r *RPCRepository) Call(ctx context.Context, in *CallParams) error {
	swapCallData, err := in.ABI.Pack(in.Method, in.Params...)
	if err != nil {
		logger.Errorf("failed to pack api, err: %v", err)
		return err
	}

	stakeTokenAddr := common.HexToAddress(in.Target)
	msg := ethereum.CallMsg{To: &stakeTokenAddr, Data: swapCallData}
	resp, err := r.client.CallContract(ctx, msg, nil)
	if err != nil {
		logger.Errorf("failed to call multicall, err: %v", err)
		return err
	}

	if err = in.ABI.UnpackIntoInterface(in.Output, in.Method, resp); err != nil {
		logger.Errorf("failed to unpack call %s, err: %v", in.Method, err)
		return err
	}

	return nil
}

func (r *RPCRepository) MultiCall(ctx context.Context, calls []*CallParams) error {
	var multiCallParams []multiCallParam

	for _, c := range calls {
		callData, err := c.ABI.Pack(c.Method, c.Params...)
		if err != nil {
			logger.Errorf("failed to build call data for target=%s method=%s, err: %v", c.Target, c.Method, err)
			return err
		}

		multiCallParams = append(
			multiCallParams, multiCallParam{
				Target:   common.HexToAddress(c.Target),
				CallData: callData,
			},
		)
	}

	multiCallData, err := abis.Multicall.Pack("aggregate", multiCallParams)
	if err != nil {
		logger.Errorf("failed to build multi call data, err: %v", err)
		return err
	}

	multiCallAddress := common.HexToAddress(r.config.MulticallAddress)
	msg := ethereum.CallMsg{To: &multiCallAddress, Data: multiCallData}
	resp, err := r.client.CallContract(ctx, msg, nil)
	if err != nil {
		logger.Errorf("failed to call multicall, err: %v", err)
		return err
	}

	var result struct {
		BlockNumber *big.Int
		ReturnData  [][]byte
	}
	err = abis.Multicall.UnpackIntoInterface(&result, "aggregate", resp)
	if err != nil || len(result.ReturnData) != len(calls) {
		logger.Errorf("failed to unpack multicall response, err: %v", err)
		return err
	}

	for i, c := range calls {
		err = c.ABI.UnpackIntoInterface(c.Output, c.Method, result.ReturnData[i])
		if err != nil {
			logger.Errorf("failed to unpack target=%s method=%s, err: %v", c.Target, c.Method, err)
			return err
		}
	}

	return nil
}

func (r *RPCRepository) TryAggregate(ctx context.Context, requireSuccess bool, calls []*TryCallParams) error {
	var multiCallParams []multiCallParam

	for _, c := range calls {
		callData, err := c.ABI.Pack(c.Method, c.Params...)
		if err != nil {
			logger.Errorf("failed to build call data for target=%s method=%s, err: %v", c.Target, c.Method, err)
			return err
		}

		multiCallParams = append(
			multiCallParams, multiCallParam{
				Target:   common.HexToAddress(c.Target),
				CallData: callData,
			},
		)
	}

	multiCallData, err := abis.Multicall.Pack("tryAggregate", requireSuccess, multiCallParams)
	if err != nil {
		logger.Errorf("failed to build multi call data, err: %v", err)
		return err
	}

	multiCallAddress := common.HexToAddress(r.config.MulticallAddress)
	msg := ethereum.CallMsg{To: &multiCallAddress, Data: multiCallData}
	resp, err := r.client.CallContract(ctx, msg, nil)
	if err != nil {
		logger.Errorf("failed to call multicall, err: %v", err)
		return err
	}

	var result []struct {
		Success    bool
		ReturnData []byte
	}
	err = abis.Multicall.UnpackIntoInterface(&result, "tryAggregate", resp)
	if err != nil || len(result) != len(calls) {
		logger.Errorf("failed to unpack multicall response, err: %v", err)
		return err
	}

	for i, c := range calls {
		if calls[i].Success == nil {
			calls[i].Success = &result[i].Success
		} else {
			*calls[i].Success = result[i].Success
		}
		if result[i].Success {
			if val, ok := c.Output.(map[string]interface{}); ok {
				if err := c.ABI.UnpackIntoMap(val, c.Method, result[i].ReturnData); err != nil {
					logger.Errorf("failed to unpack map target=%s method=%s, err: %v", c.Target, c.Method, err)
					return NewUnPackMulticallError(err)
				}
			} else {
				err = c.ABI.UnpackIntoInterface(c.Output, c.Method, result[i].ReturnData)
				if err != nil {
					logger.Errorf("failed to unpack target=%s method=%s, err: %v", c.Target, c.Method, err)
					return NewUnPackMulticallError(err)
				}
			}
		}
	}

	return nil
}

func (r *RPCRepository) TryAggregateForce(ctx context.Context, requireSuccess bool, calls []*TryCallParams) error {
	var multicallParams []multiCallParam

	for _, c := range calls {
		callData, err := c.ABI.Pack(c.Method, c.Params...)
		if err != nil {
			logger.Errorf("failed to build call data for target=%s method=%s, err: %v", c.Target, c.Method, err)
			return err
		}

		multicallParams = append(
			multicallParams, multiCallParam{
				Target:   common.HexToAddress(c.Target),
				CallData: callData,
			},
		)
	}

	multiCallData, err := abis.Multicall.Pack("tryAggregate", requireSuccess, multicallParams)
	if err != nil {
		logger.Errorf("failed to build multi call data, err: %v", err)
		return err
	}

	multiCallAddress := common.HexToAddress(r.config.MulticallAddress)
	msg := ethereum.CallMsg{To: &multiCallAddress, Data: multiCallData}
	resp, err := r.client.CallContract(ctx, msg, nil)
	if err != nil {
		logger.Errorf("failed to call multicall, err: %v", err)
		return err
	}

	var result []struct {
		Success    bool
		ReturnData []byte
	}
	err = abis.Multicall.UnpackIntoInterface(&result, "tryAggregate", resp)
	if err != nil || len(result) != len(calls) {
		logger.Errorf("failed to unpack multicall response, err: %v", err)
		return err
	}

	for i, c := range calls {
		if calls[i].Success == nil {
			calls[i].Success = &result[i].Success
		} else {
			*calls[i].Success = result[i].Success
		}
		if result[i].Success {
			if val, ok := c.Output.(map[string]interface{}); ok {
				if err := c.ABI.UnpackIntoMap(val, c.Method, result[i].ReturnData); err != nil {
					logger.Errorf("failed to unpack map target=%s method=%s, err: %v", c.Target, c.Method, err)
					// return NewUnPackMulticallError(err.Error())
				}
			} else {
				err = c.ABI.UnpackIntoInterface(c.Output, c.Method, result[i].ReturnData)
				if err != nil {
					logger.Errorf("failed to unpack target=%s method=%s, err: %v", c.Target, c.Method, err)
					//return NewUnPackMulticallError(err.Error())
				}
			}
		}
	}

	return nil
}

func (r *RPCRepository) TryAggregateUnpack(ctx context.Context, requireSuccess bool, calls []*TryCallUnPackParams) error {
	var multiCallParams []multiCallParam

	for _, c := range calls {
		callData, err := c.ABI.Pack(c.Method, c.Params...)
		if err != nil {
			logger.Errorf("failed to build call data for target=%s method=%s, err: %v", c.Target, c.Method, err)
			return err
		}

		multiCallParams = append(
			multiCallParams, multiCallParam{
				Target:   common.HexToAddress(c.Target),
				CallData: callData,
			},
		)
	}

	multiCallData, err := abis.Multicall.Pack("tryAggregate", requireSuccess, multiCallParams)
	if err != nil {
		logger.Errorf("failed to build multi call data, err: %v", err)
		return err
	}

	multiCallAddress := common.HexToAddress(r.config.MulticallAddress)
	msg := ethereum.CallMsg{To: &multiCallAddress, Data: multiCallData}
	resp, err := r.client.CallContract(ctx, msg, nil)
	if err != nil {
		logger.Errorf("failed to call multicall, err: %v", err)
		return err
	}

	var result []struct {
		Success    bool
		ReturnData []byte
	}
	err = abis.Multicall.UnpackIntoInterface(&result, "tryAggregate", resp)
	if err != nil || len(result) != len(calls) {
		logger.Errorf("failed to unpack multicall response, err: %v", err)
		return err
	}

	for i, c := range calls {
		if calls[i].Success == nil {
			calls[i].Success = &result[i].Success
		} else {
			*calls[i].Success = result[i].Success
		}
		if result[i].Success {
			for t, a := range c.UnpackABI {
				if err = a.UnpackIntoInterface(&c.Output[t], c.Method, result[i].ReturnData); err == nil {
					break
				}
				if t == len(c.UnpackABI)-1 {
					logger.Errorf("failed to unpack map target=%s method=%s, err: %v", c.Target, c.Method, err)
					return NewUnPackMulticallError(err)
				}
			}
		}
	}

	return nil
}
