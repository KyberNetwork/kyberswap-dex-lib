package swapbasedperp

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

type PancakePairReader struct {
	abi          abi.ABI
	ethrpcClient *ethrpc.Client
	log          logger.Logger
}

func NewPancakePairReader(ethrpcClient *ethrpc.Client) *PancakePairReader {
	return &PancakePairReader{
		abi:          pancakePairABI,
		ethrpcClient: ethrpcClient,
		log: logger.WithFields(logger.Fields{
			"liquiditySource": DexTypeSwapBasedPerp,
			"reader":          "PancakePairReader",
		}),
	}
}

func (r *PancakePairReader) Read(ctx context.Context, address string) (*PancakePair, error) {
	var reserves struct {
		Reserve0           *big.Int
		Reserve1           *big.Int
		BlockTimestampLast uint32
	}

	rpcRequest := r.ethrpcClient.NewRequest().SetContext(ctx)

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    r.abi,
		Target: address,
		Method: pancakePairMethodGetReserves,
		Params: nil,
	}, []interface{}{&reserves})

	if _, err := rpcRequest.Call(); err != nil {
		r.log.Errorf("error when call rpc request %v", err)
		return nil, err
	}

	return &PancakePair{
		Reserves: []*big.Int{
			reserves.Reserve0,
			reserves.Reserve1,
		},
		TimestampLast: reserves.BlockTimestampLast,
	}, nil
}
