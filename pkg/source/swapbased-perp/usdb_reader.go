package swapbasedperp

import (
	"context"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/holiman/uint256"
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
	var totalSupply *uint256.Int
	rpcRequest := r.ethrpcClient.NewRequest().SetContext(ctx)

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    r.abi,
		Target: address,
		Method: usdbMethodTotalSupply,
		Params: nil,
	}, []any{&totalSupply})

	if _, err := rpcRequest.Call(); err != nil {
		r.log.Errorf("error when call rpc request %v", err)
		return nil, err
	}

	return &USDB{
		Address:     address,
		TotalSupply: u256ToBig(totalSupply),
	}, nil
}
