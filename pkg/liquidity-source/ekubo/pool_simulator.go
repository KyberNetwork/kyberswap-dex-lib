package ekubo

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type EkuboPool = Pool

type PoolSimulator struct {
	pool.Pool
	EkuboPool
	Core common.Address
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, fmt.Errorf("unmarshalling static extra: %w", err)
	}

	ekuboPool, err := unmarshalPool([]byte(entityPool.Extra), &staticExtra)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling extra: %w", err)
	}

	tokens := lo.Map(entityPool.Tokens, func(item *entity.PoolToken, _ int) string {
		return item.Address
	})

	reserves := lo.Map(entityPool.Reserves, func(item string, _ int) *big.Int {
		return bignumber.NewBig(item)
	})

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     entityPool.Address,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      tokens,
			Reserves:    reserves,
			BlockNumber: entityPool.BlockNumber,
		}},
		EkuboPool: ekuboPool,
		Core:      staticExtra.Core,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn := params.TokenAmountIn.Token

	quote, err := p.quoteWithZeroChecksAndBaseGasCost(
		params.TokenAmountIn.Amount,
		strings.EqualFold(tokenIn, p.GetTokens()[1]),
	)
	if err != nil {
		return nil, fmt.Errorf("ekubo quoting: %w", err)
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
	tokenOut := params.TokenAmountOut.Token
	input := new(big.Int).Neg(params.TokenAmountOut.Amount)

	quote, err := p.quoteWithZeroChecksAndBaseGasCost(
		input,
		strings.EqualFold(tokenOut, p.GetTokens()[1]),
	)
	if err != nil {
		return nil, fmt.Errorf("ekubo quoting: %w", err)
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{
			Token:  params.TokenIn,
			Amount: quote.CalculatedAmount,
		},
		Fee: &pool.TokenAmount{
			Token:  params.TokenIn,
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

func (p *PoolSimulator) quoteWithZeroChecksAndBaseGasCost(amount *big.Int, isToken1 bool) (*quoting.Quote, error) {
	if amount.Sign() == 0 {
		return nil, ErrZeroAmount
	}

	quote, err := p.Quote(amount, isToken1)
	if err != nil {
		return nil, err
	}

	if quote.CalculatedAmount.Sign() == 0 {
		return nil, ErrZeroAmount
	}

	quote.Gas += quoting.BaseGasCost

	return quote, nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	p.SetSwapState(params.SwapInfo.(quoting.SwapInfo).SwapStateAfter)
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) any {
	return Meta{
		Core:    p.Core,
		PoolKey: p.EkuboPool.GetKey().ToAbi(),
	}
}
