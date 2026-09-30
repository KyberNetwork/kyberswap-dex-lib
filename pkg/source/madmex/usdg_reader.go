package madmex

import (
	"context"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/holiman/uint256"
)

type USDGReader struct {
	abi          abi.ABI
	ethrpcClient *ethrpc.Client
	log          logger.Logger
}

func NewUSDGReader(ethrpcClient *ethrpc.Client) *USDGReader {
	return &USDGReader{
		abi:          erc20ABI,
		ethrpcClient: ethrpcClient,
		log: logger.WithFields(logger.Fields{
			"liquiditySource": DexTypeMadmex,
			"reader":          "USDGReader",
		}),
	}
}

func (r *USDGReader) Read(ctx context.Context, address string) (*USDG, error) {
	var totalSupply *uint256.Int
	rpcRequest := r.ethrpcClient.NewRequest().SetContext(ctx)

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    r.abi,
		Target: address,
		Method: usdgMethodTotalSupply,
		Params: nil,
	}, []any{&totalSupply})

	if _, err := rpcRequest.Call(); err != nil {
		r.log.Errorf("error when call rpc request %v", err)
		return nil, err
	}

	return &USDG{
		Address:     address,
		TotalSupply: u256ToBig(totalSupply),
	}, nil
}
