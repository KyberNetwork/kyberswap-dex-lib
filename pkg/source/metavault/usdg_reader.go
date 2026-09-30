package metavault

import (
	"context"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/holiman/uint256"
)

type USDMReader struct {
	abi          abi.ABI
	ethrpcClient *ethrpc.Client
	log          logger.Logger
}

func NewUSDMReader(ethrpcClient *ethrpc.Client) *USDMReader {
	return &USDMReader{
		abi:          erc20ABI,
		ethrpcClient: ethrpcClient,
		log: logger.WithFields(logger.Fields{
			"liquiditySource": DexTypeMetavault,
			"reader":          "USDMReader",
		}),
	}
}

func (r *USDMReader) Read(ctx context.Context, address string) (*USDM, error) {
	var totalSupply *uint256.Int

	rpcRequest := r.ethrpcClient.NewRequest().SetContext(ctx)

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    r.abi,
		Target: address,
		Method: USDMMethodTotalSupply,
		Params: nil,
	}, []any{&totalSupply})

	if _, err := rpcRequest.Call(); err != nil {
		r.log.Errorf("error when call rpc request %v", err)
		return nil, err
	}

	return &USDM{
		Address:     address,
		TotalSupply: u256ToBig(totalSupply),
	}, nil
}
