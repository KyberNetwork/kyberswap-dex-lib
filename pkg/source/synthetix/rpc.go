package synthetix

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
)

func newRequest(client *ethrpc.Client, ctx context.Context, blockNumber uint64) *ethrpc.Request {
	req := client.NewRequest().SetContext(ctx)
	if blockNumber != 0 {
		req.SetBlockNumber(new(big.Int).SetUint64(blockNumber))
	}
	return req
}
