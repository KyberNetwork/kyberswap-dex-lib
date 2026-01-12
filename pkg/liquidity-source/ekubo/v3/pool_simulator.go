package ekubov3

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/quoting"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

type (
	EkuboPool = Pool

	PoolSimulator struct {
		pool.Pool
		EkuboPool
		Core common.Address
	}
)

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn := params.TokenAmountIn.Token

	quote, err := p.quoteWithZeroChecksAndBaseGasCost(
		params.TokenAmountIn.Amount,
		strings.EqualFold(tokenIn, p.GetTokens()[1]),
	)
	if err != nil {
		return nil, fmt.Errorf("ekubo quoting: %w", err)
	}

	consumedAmount := quote.ConsumedAmount.ToBig()
	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  params.TokenOut,
			Amount: quote.CalculatedAmount.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  params.TokenAmountIn.Token,
			Amount: quote.FeesPaid.ToBig(),
		},
		RemainingTokenAmountIn: &pool.TokenAmount{
			Token:  params.TokenAmountIn.Token,
			Amount: consumedAmount.Sub(params.TokenAmountIn.Amount, consumedAmount),
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

	consumedAmount := big256.ToBig(quote.ConsumedAmount)
	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{
			Token:  params.TokenIn,
			Amount: quote.CalculatedAmount.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  params.TokenIn,
			Amount: quote.FeesPaid.ToBig(),
		},
		RemainingTokenAmountOut: &pool.TokenAmount{
			Token:  params.TokenAmountOut.Token,
			Amount: consumedAmount.Add(params.TokenAmountOut.Amount, consumedAmount),
		},
		Gas:      quote.Gas,
		SwapInfo: quote.SwapInfo,
	}, nil
}

func (p *PoolSimulator) quoteWithZeroChecksAndBaseGasCost(amountBig *big.Int, isToken1 bool) (*quoting.Quote, error) {
	amount, overflow := uint256.FromBig(amountBig)
	if overflow {
		return nil, math.ErrOverflow
	} else if amount.IsZero() {
		return nil, ErrZeroAmount
	}

	quote, err := p.Quote(amount, isToken1)
	if err != nil {
		return nil, err
	} else if quote.CalculatedAmount.IsZero() {
		return nil, ErrZeroAmount
	}

	quote.Gas += quoting.BaseGasCost
	return quote, nil
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.EkuboPool = p.EkuboPool.CloneState().(EkuboPool)
	return &cloned
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	p.SetSwapState(params.SwapInfo.(quoting.SwapInfo).SwapStateAfter)
}

func (p *PoolSimulator) GetMetaInfo(_, _ string) any {
	return Meta{
		Core:    p.Core,
		PoolKey: p.EkuboPool.GetKey().ToAbi(),
	}
}

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
