package shared

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
)

const (
	quoteExactInputSingleMethod = "quoteExactInputSingle"
)

type IQuoter interface {
	QuoteExactInputSingle(ctx context.Context, params QuoteExactSingleParams, blockNumber uint64) (QuoteResult, error)
}

type quoter struct {
	config       QuoterConfig
	ethrpcClient *ethrpc.Client
}

func NewQuoter(config QuoterConfig, ethrpcClient *ethrpc.Client) *quoter {
	return &quoter{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (q *quoter) QuoteExactInputSingle(ctx context.Context, params QuoteExactSingleParams, blockNumber uint64) (QuoteResult, error) {
	req := q.ethrpcClient.NewRequest().SetContext(ctx)
	if blockNumber > 0 {
		req.SetBlockNumber(big.NewInt(int64(blockNumber)))
	}

	var result QuoteResult
	req.AddCall(&ethrpc.Call{
		ABI:    quoterABI,
		Target: q.config.QuoterAddress,
		Method: quoteExactInputSingleMethod,
		Params: []any{params},
	}, []any{&result})

	_, err := req.Call()
	if err != nil {
		return QuoteResult{}, err
	}

	return result, nil
}
