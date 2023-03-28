package eth

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"context"

	"github.com/KyberNetwork/router-service/pkg/logger"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type ContractMethodCallInput struct {
	RPCs    []string
	ABI     abi.ABI
	Address string
	Method  string
	Params  []interface{}
	Result  interface{}
}

func GetEthClient(ctx context.Context, rpcs []string) (*ethclient.Client, error) {
	rpcsLen := len(rpcs)
	if rpcsLen == 0 {
		return nil, fmt.Errorf("failed to connect any RPCs")
	}
	indexRand, err := rand.Int(rand.Reader, big.NewInt(int64(rpcsLen)))
	if err != nil {
		return nil, err
	}
	index := int(indexRand.Int64())

	for i := 0; i < rpcsLen; i++ {
		rpc := rpcs[(index+i)%rpcsLen]
		client, err := ethclient.Dial(rpc)
		if err == nil {
			return client, nil
		}
		logger.Errorf("failed to connect %s, err: %v", rpc, err)
	}

	return nil, fmt.Errorf("failed to connect any RPCs")
}

func ContractMethodCall(ctx context.Context, in *ContractMethodCallInput) error {
	swapCallData, err := in.ABI.Pack(in.Method, in.Params...)
	if err != nil {
		logger.Errorf("failed to pack api, err: %v", err)
		return err
	}

	ethCli, err := GetEthClient(ctx, in.RPCs)
	if err != nil {
		logger.Errorf("failed to get eth client, err: %v", err)
		return err
	}

	stakeTokenAddr := common.HexToAddress(in.Address)
	msg := ethereum.CallMsg{To: &stakeTokenAddr, Data: swapCallData}
	resp, err := ethCli.CallContract(ctx, msg, nil)
	if err != nil {
		logger.Errorf("failed to call multicall, err: %v", err)
		return err
	}

	err = in.ABI.UnpackIntoInterface(in.Result, in.Method, resp)
	if err != nil {
		logger.Errorf("failed to unpack call %s, err: %v", in.Method, err)
		return err
	}

	return nil
}
