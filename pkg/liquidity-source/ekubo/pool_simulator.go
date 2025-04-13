package ekubo

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting"
	ekubopool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type EkuboPool = quoting.Pool

type PoolSimulator struct {
	pool.Pool
	EkuboPool
	Core string
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var ekuboPool EkuboPool
	switch staticExtra.ExtensionType {
	case ekubopool.Base:
		p := ekubopool.NewBasePool(staticExtra.PoolKey, extra.PoolState)
		ekuboPool = &p
	case ekubopool.Oracle:
		p := ekubopool.NewOraclePool(staticExtra.PoolKey, extra.PoolState)
		ekuboPool = &p
	default:
		return nil, fmt.Errorf("unknown pool extension %v, %v",
			staticExtra.ExtensionType, staticExtra.PoolKey.Config.Extension)
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

	quote, err := p.Quote(
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

	quote, err := p.Quote(
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
	return Meta{
		Core:    p.Core,
		PoolKey: p.EkuboPool.GetKey().ToAbi(),
	}
}
