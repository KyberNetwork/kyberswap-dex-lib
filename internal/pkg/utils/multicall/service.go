package multicall

import (
	"math/big"

	"context"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/abis"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/utils/eth"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/logger"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type Call struct {
	ABI    abi.ABI
	Target string
	Method string
	Params []interface{}
	Output interface{}
}

type TryCall struct {
	ABI     abi.ABI
	Target  string
	Method  string
	Params  []interface{}
	Output  interface{}
	Success *bool
}

type TryCallUnPack struct {
	ABI       abi.ABI
	UnpackABI []abi.ABI
	Target    string
	Method    string
	Params    []interface{}
	Output    []interface{}
	Success   *bool
}

type multicallParam struct {
	Target   common.Address
	CallData []byte
}

type UnPackMulticallError struct {
	Msg string
}

func NewUnPackMulticallError(msg string) error {
	return &UnPackMulticallError{
		Msg: msg,
	}
}

func (e UnPackMulticallError) Error() string { return e.Msg }

func Process(ctx context.Context, address string, rpcs []string, calls []*Call) error {
	var multicallParams []multicallParam

	for _, c := range calls {
		callData, err := c.ABI.Pack(c.Method, c.Params...)
		if err != nil {
			logger.Errorf("failed to build call data for target=%s method=%s, err: %v", c.Target, c.Method, err)
			return err
		}

		multicallParams = append(
			multicallParams, multicallParam{
				Target:   common.HexToAddress(c.Target),
				CallData: callData,
			},
		)
	}

	multiCallData, err := abis.Multicall.Pack("aggregate", multicallParams)
	if err != nil {
		logger.Errorf("failed to build multi call data, err: %v", err)
		return err
	}

	ethCli, err := eth.GetEthClient(ctx, rpcs)
	if err != nil {
		logger.Errorf("failed to get eth client, err: %v", err)
		return err
	}

	multiCallAddress := common.HexToAddress(address)
	msg := ethereum.CallMsg{To: &multiCallAddress, Data: multiCallData}
	resp, err := ethCli.CallContract(ctx, msg, nil)
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

func TryAggregate(ctx context.Context, address string, rpcs []string, requireSuccess bool, calls []*TryCall) error {
	var multicallParams []multicallParam

	for _, c := range calls {
		callData, err := c.ABI.Pack(c.Method, c.Params...)
		if err != nil {
			logger.Errorf("failed to build call data for target=%s method=%s, err: %v", c.Target, c.Method, err)
			return err
		}

		multicallParams = append(
			multicallParams, multicallParam{
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

	ethCli, err := eth.GetEthClient(ctx, rpcs)
	if err != nil {
		logger.Errorf("failed to get eth client, err: %v", err)
		return err
	}

	multiCallAddress := common.HexToAddress(address)
	msg := ethereum.CallMsg{To: &multiCallAddress, Data: multiCallData}
	resp, err := ethCli.CallContract(ctx, msg, nil)
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
					return NewUnPackMulticallError(err.Error())
				}
			} else {
				err = c.ABI.UnpackIntoInterface(c.Output, c.Method, result[i].ReturnData)
				if err != nil {
					logger.Errorf("failed to unpack target=%s method=%s, err: %v", c.Target, c.Method, err)
					return NewUnPackMulticallError(err.Error())
				}
			}
		}
	}

	return nil
}

func TryAggregateUnpack(
	ctx context.Context, address string, rpcs []string, requireSuccess bool, calls []*TryCallUnPack,
) error {
	var multicallParams []multicallParam

	for _, c := range calls {
		callData, err := c.ABI.Pack(c.Method, c.Params...)
		if err != nil {
			logger.Errorf("failed to build call data for target=%s method=%s, err: %v", c.Target, c.Method, err)
			return err
		}

		multicallParams = append(
			multicallParams, multicallParam{
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

	ethCli, err := eth.GetEthClient(ctx, rpcs)
	if err != nil {
		logger.Errorf("failed to get eth client, err: %v", err)
		return err
	}

	multiCallAddress := common.HexToAddress(address)
	msg := ethereum.CallMsg{To: &multiCallAddress, Data: multiCallData}
	resp, err := ethCli.CallContract(ctx, msg, nil)
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
					return NewUnPackMulticallError(err.Error())
				}

			}
		}
	}

	return nil
}

func TryAggregateForce(
	ctx context.Context, address string, rpcs []string, requireSuccess bool, calls []*TryCall,
) error {
	var multicallParams []multicallParam

	for _, c := range calls {
		callData, err := c.ABI.Pack(c.Method, c.Params...)
		if err != nil {
			logger.Errorf("failed to build call data for target=%s method=%s, err: %v", c.Target, c.Method, err)
			return err
		}

		multicallParams = append(
			multicallParams, multicallParam{
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

	ethCli, err := eth.GetEthClient(ctx, rpcs)
	if err != nil {
		logger.Errorf("failed to get eth client, err: %v", err)
		return err
	}

	multiCallAddress := common.HexToAddress(address)
	msg := ethereum.CallMsg{To: &multiCallAddress, Data: multiCallData}
	resp, err := ethCli.CallContract(ctx, msg, nil)
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
					// return NewUnPackMulticallError(err.Error())
				}
			}
		}
	}

	return nil
}
