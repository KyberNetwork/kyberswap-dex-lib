package gmx

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

type USDRReader struct {
	abi          abi.ABI
	ethrpcClient *ethrpc.Client
	log          logger.Logger
}

func NewUSDRReader(ethrpcClient *ethrpc.Client) *USDRReader {
	return &USDRReader{
		abi:          erc20ABI,
		ethrpcClient: ethrpcClient,
		log: logger.WithFields(logger.Fields{
			"liquiditySource": DexTypeGmx,
			"reader":          "USDRReader",
		}),
	}
}

func (r *USDRReader) Read(ctx context.Context, address string) (*USDR, error) {
	var totalSupply *big.Int
	rpcRequest := r.ethrpcClient.NewRequest().SetContext(ctx)

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    r.abi,
		Target: address,
		Method: usdrMethodTotalSupply,
		Params: nil,
	}, []interface{}{&totalSupply})

	if _, err := rpcRequest.Call(); err != nil {
		r.log.Errorf("error when call rpc request %v", err)
		return nil, err
	}

	return &USDR{
		Address:     address,
		TotalSupply: totalSupply,
	}, nil
}
