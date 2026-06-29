package aeonvamm

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
	ErrInvalidAmountIn       = errors.New("invalid amount in")
)

type PoolSimulator struct {
	pool.Pool
	extra Extra
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}
	return &PoolSimulator{
		Pool:  pool.Pool{Info: pool.PoolInfo{Address: entityPool.Address, Exchange: entityPool.Exchange, Type: entityPool.Type, Tokens: entityPool.Tokens, Reserves: entityPool.Reserves}},
		extra: extra,
	}, nil
}

// CalcAmountOut implements constant product formula with fee
// amountOut = amountIn*(10000-fee)*reserveOut / (reserveIn*10000 + amountIn*(10000-fee))
func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if params.TokenAmountIn.Amount == nil || params.TokenAmountIn.Amount.Sign() <= 0 {
		return nil, ErrInvalidAmountIn
	}

	tokenInIdx := p.GetTokenIndex(params.TokenAmountIn.Token)
	tokenOutIdx := p.GetTokenIndex(params.TokenAmountOut)
	if tokenInIdx < 0 || tokenOutIdx < 0 {
		return nil, pool.ErrTokenNotFound
	}

	var reserveIn, reserveOut *big.Int
	if tokenInIdx == 0 {
		reserveIn, reserveOut = p.extra.Reserve0, p.extra.Reserve1
	} else {
		reserveIn, reserveOut = p.extra.Reserve1, p.extra.Reserve0
	}

	if reserveIn == nil || reserveOut == nil || reserveIn.Sign() == 0 || reserveOut.Sign() == 0 {
		return nil, ErrInsufficientLiquidity
	}

	fee := int64(p.extra.Fee) // bps
	feeDenominator := big.NewInt(10000)
	feeNumerator := big.NewInt(10000 - fee)

	amountIn := params.TokenAmountIn.Amount
	// amountInWithFee = amountIn * (10000 - fee)
	amountInWithFee := new(big.Int).Mul(amountIn, feeNumerator)
	// numerator = amountInWithFee * reserveOut
	numerator := new(big.Int).Mul(amountInWithFee, reserveOut)
	// denominator = reserveIn * 10000 + amountInWithFee
	denominator := new(big.Int).Add(
		new(big.Int).Mul(reserveIn, feeDenominator),
		amountInWithFee,
	)
	amountOut := new(big.Int).Div(numerator, denominator)

	if amountOut.Sign() <= 0 || amountOut.Cmp(reserveOut) >= 0 {
		return nil, ErrInsufficientLiquidity
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenAmountOut, Amount: amountOut},
		Fee:            &pool.TokenAmount{Token: params.TokenAmountIn.Token, Amount: new(big.Int).Sub(amountIn, new(big.Int).Div(new(big.Int).Mul(amountIn, feeNumerator), feeDenominator))},
		Gas:            80000,
		SwapInfo:       nil,
	}, nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	tokenInIdx := p.GetTokenIndex(params.TokenAmountIn.Token)
	tokenOutIdx := p.GetTokenIndex(params.TokenAmountOut.Token)
	if tokenInIdx < 0 || tokenOutIdx < 0 {
		return
	}
	if tokenInIdx == 0 {
		p.extra.Reserve0 = new(big.Int).Add(p.extra.Reserve0, params.TokenAmountIn.Amount)
		p.extra.Reserve1 = new(big.Int).Sub(p.extra.Reserve1, params.TokenAmountOut.Amount)
	} else {
		p.extra.Reserve1 = new(big.Int).Add(p.extra.Reserve1, params.TokenAmountIn.Amount)
		p.extra.Reserve0 = new(big.Int).Sub(p.extra.Reserve0, params.TokenAmountOut.Amount)
	}
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{Fee: p.extra.Fee}
}

func (p *PoolSimulator) CanSwapTo(token string) []string {
	tokens := p.Pool.CanSwapTo(token)
	return tokens
}

var _ = bignumber.ZeroBI // ensure import used
