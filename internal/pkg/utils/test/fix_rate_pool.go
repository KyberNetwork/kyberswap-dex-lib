package test

import (
	"math/big"

	dexlibPool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	dexlibValueObject "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/google/uuid"
)

type FixRatePool struct {
	ID       string
	Token0   string
	Token1   string
	Exchange dexlibValueObject.Exchange
	Rate     float64
}

func NewFixRatePool(token0, token1 string, rate float64, exchange dexlibValueObject.Exchange) *FixRatePool {
	id, _ := uuid.NewRandom()

	return &FixRatePool{
		ID:       id.String(),
		Token0:   token0,
		Token1:   token1,
		Exchange: exchange,
		Rate:     rate,
	}
}

func NewFixRatePoolWithID(id, token0, token1 string, rate float64, exchange dexlibValueObject.Exchange) *FixRatePool {
	pool := NewFixRatePool(token0, token1, rate, exchange)
	pool.ID = id
	return pool
}

func (p *FixRatePool) CalcAmountOut(params dexlibPool.CalcAmountOutParams) (*dexlibPool.CalcAmountOutResult, error) {
	swapInToOut := true
	if params.TokenAmountIn.Token == p.Token1 {
		swapInToOut = false
	}

	rate := p.Rate
	if !swapInToOut {
		rate = 1 / rate
	}

	amountIn := params.TokenAmountIn.Amount
	amountInF, _ := amountIn.Float64()

	amountOutF := amountInF * rate
	amountOut, _ := new(big.Float).SetFloat64(amountOutF).Int(nil)
	tokenOut := p.Token1
	if !swapInToOut {
		tokenOut = p.Token0
	}

	return &dexlibPool.CalcAmountOutResult{
		TokenAmountOut: &dexlibPool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut,
		},
		Fee: &dexlibPool.TokenAmount{
			Token:  tokenOut,
			Amount: new(big.Int).SetInt64(0),
		},
		RemainingTokenAmountIn: nil,
		Gas:                    0,
	}, nil
}

func (p *FixRatePool) UpdateBalance(params dexlibPool.UpdateBalanceParams) {}

func (p *FixRatePool) CalculateLimit() map[string]*big.Int   { return nil }
func (p *FixRatePool) CloneState() dexlibPool.IPoolSimulator { return p }

func (p *FixRatePool) GetAddress() string  { return p.ID }
func (p *FixRatePool) GetExchange() string { return string(p.Exchange) }
func (p *FixRatePool) GetType() string     { return "fix-rate" }

func (p *FixRatePool) GetMetaInfo(tokenIn string, tokenOut string) interface{} { return nil }
func (p *FixRatePool) GetReserves() []*big.Int                                 { return nil }

func (p *FixRatePool) GetTokens() []string { return []string{p.Token0, p.Token1} }

func (p *FixRatePool) GetTokenIndex(token string) int {
	if token == p.Token0 {
		return 0
	}
	if token == p.Token1 {
		return 1
	}
	return -1
}

func (p *FixRatePool) CanSwapFrom(token string) []string {
	if token == p.Token0 {
		return []string{p.Token1}
	}
	if token == p.Token1 {
		return []string{p.Token0}
	}
	return []string{}
}

func (p *FixRatePool) CanSwapTo(token string) []string {
	return p.CanSwapFrom(token)
}
