package madmex

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/holiman/uint256"
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
			"liquiditySource": DexTypeMadmex,
			"reader":          "PancakePairReader",
		}),
	}
}

func (r *PancakePairReader) Read(ctx context.Context, address string) (*PancakePair, error) {
	var reserves struct {
		Reserve0           *uint256.Int
		Reserve1           *uint256.Int
		BlockTimestampLast uint32
	}

	rpcRequest := r.ethrpcClient.NewRequest().SetContext(ctx)

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    r.abi,
		Target: address,
		Method: pancakePairMethodGetReserves,
		Params: nil,
	}, []any{&reserves})

	if _, err := rpcRequest.Call(); err != nil {
		r.log.Errorf("error when call rpc request %v", err)
		return nil, err
	}

	return &PancakePair{
		Reserves: []*big.Int{
			u256ToBig(reserves.Reserve0),
			u256ToBig(reserves.Reserve1),
		},
		TimestampLast: reserves.BlockTimestampLast,
	}, nil
}
