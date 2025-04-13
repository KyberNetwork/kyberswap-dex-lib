package ekubo

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	quoting "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting"
	pool2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type EkuboPool = quoting.Pool

type PoolSimulator struct {
	pool.Pool
	EkuboPool
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, fmt.Errorf("unmarshalling extra: %w", err)
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, fmt.Errorf("unmarshalling staticExtra: %w", err)
	}

	extension := staticExtra.Extension
	var ekuboPool EkuboPool

	if extension == pool2.Base {
		p := pool2.NewBasePool(
			staticExtra.PoolKey,
			extra.State,
		)
		ekuboPool = &p
	} else if extension == pool2.Oracle {
		p := pool2.NewOraclePool(
			staticExtra.PoolKey,
			extra.State,
		)
		ekuboPool = &p
	} else {
		return nil, fmt.Errorf("unknown pool extension %v", extension)
	}

	return &PoolSimulator{
		Pool:      pool.Pool{},
		EkuboPool: ekuboPool,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn := common.HexToAddress(params.TokenAmountIn.Token)

	quote, err := p.Quote(
		params.TokenAmountIn.Amount,
		p.GetKey().Token1.Cmp(tokenIn) == 0,
	)
	if err != nil {
		return nil, fmt.Errorf("quoting: %w", err)
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  params.TokenOut,
			Amount: quote.CalculatedAmount,
		},
		Fee: &pool.TokenAmount{
			Token:  params.TokenAmountIn.Token,
			Amount: quote.FeesPaid,
		},
		RemainingTokenAmountIn: &pool.TokenAmount{
			Token:  params.TokenAmountIn.Token,
			Amount: new(big.Int).Sub(params.TokenAmountIn.Amount, quote.ConsumedAmount),
		},
		Gas:      quote.Gas,
		SwapInfo: quote.SwapInfo,
	}, nil
}

func (p *PoolSimulator) CalcAmountIn(params pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	tokenOut := common.HexToAddress(params.TokenAmountOut.Token)
	input := new(big.Int).Neg(params.TokenAmountOut.Amount)

	quote, err := p.Quote(
		input,
		p.GetKey().Token1.Cmp(tokenOut) == 0,
	)
	if err != nil {
		return nil, fmt.Errorf("quoting: %w", err)
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{
			Token:  params.TokenIn,
			Amount: quote.CalculatedAmount,
		},
		Fee: &pool.TokenAmount{
			Token:  params.TokenAmountOut.Token,
			Amount: quote.FeesPaid,
		},
		RemainingTokenAmountOut: &pool.TokenAmount{
			Token:  params.TokenAmountOut.Token,
			Amount: new(big.Int).Add(params.TokenAmountOut.Amount, quote.ConsumedAmount),
		},
		Gas:      quote.Gas,
		SwapInfo: quote.SwapInfo,
	}, nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	p.SetState(params.SwapInfo.(quoting.SwapInfo).StateAfter)
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) any {
	return nil
}
