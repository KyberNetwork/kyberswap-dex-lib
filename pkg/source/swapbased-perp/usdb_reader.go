package swapbasedperp

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

type USDBReader struct {
	abi          abi.ABI
	ethrpcClient *ethrpc.Client
	log          logger.Logger
}

func NewUSDBReader(ethrpcClient *ethrpc.Client) *USDBReader {
	return &USDBReader{
		abi:          erc20ABI,
		ethrpcClient: ethrpcClient,
		log: logger.WithFields(logger.Fields{
			"liquiditySource": DexTypeSwapBasedPerp,
			"reader":          "USDBReader",
		}),
	}
}

func (r *USDBReader) Read(ctx context.Context, address string) (*USDB, error) {
	var totalSupply *big.Int
	rpcRequest := r.ethrpcClient.NewRequest().SetContext(ctx)

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    r.abi,
		Target: address,
		Method: usdbMethodTotalSupply,
		Params: nil,
	}, []interface{}{&totalSupply})

	if _, err := rpcRequest.Call(); err != nil {
		r.log.Errorf("error when call rpc request %v", err)
		return nil, err
	}

	return &USDB{
		Address:     address,
		TotalSupply: totalSupply,
	}, nil
}
