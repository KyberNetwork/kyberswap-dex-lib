package quickperps

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

type USDQReader struct {
	abi          abi.ABI
	ethrpcClient *ethrpc.Client
	log          logger.Logger
}

func NewUSDQReader(ethrpcClient *ethrpc.Client) *USDQReader {
	return &USDQReader{
		abi:          erc20ABI,
		ethrpcClient: ethrpcClient,
		log: logger.WithFields(logger.Fields{
			"liquiditySource": DexTypeQuickperps,
			"reader":          "USDQReader",
		}),
	}
}

func (r *USDQReader) Read(ctx context.Context, address string) (*USDQ, error) {
	var totalSupply *big.Int
	rpcRequest := r.ethrpcClient.NewRequest().SetContext(ctx)

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    r.abi,
		Target: address,
		Method: usdqMethodTotalSupply,
		Params: nil,
	}, []interface{}{&totalSupply})

	if _, err := rpcRequest.Call(); err != nil {
		r.log.Errorf("error when call rpc request %v", err)
		return nil, err
	}

	return &USDQ{
		Address:     address,
		TotalSupply: totalSupply,
	}, nil
}
