package zkerafinance

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

type blockNumberContextKey struct{}

func newRequest(client *ethrpc.Client, ctx context.Context) *ethrpc.Request {
	request := client.NewRequest().SetContext(ctx)
	if blockNumber, ok := ctx.Value(blockNumberContextKey{}).(*big.Int); ok {
		request.SetBlockNumber(blockNumber)
	}
	return request
}

func withBlockNumber(ctx context.Context, blockNumber *big.Int) context.Context {
	return context.WithValue(ctx, blockNumberContextKey{}, blockNumber)
}

func validBlockNumber(blockNumber *big.Int) bool {
	return blockNumber != nil && blockNumber.Sign() >= 0 && blockNumber.IsUint64()
}

func CallParamsFactory(abi abi.ABI, address string) func(callMethod string, params []any) *ethrpc.Call {
	return func(callMethod string, params []any) *ethrpc.Call {
		return &ethrpc.Call{
			ABI:    abi,
			Target: address,
			Method: callMethod,
			Params: params,
		}
	}
}
