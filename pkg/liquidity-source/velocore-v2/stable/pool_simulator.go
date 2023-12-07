package stable

import (
	"encoding/json"
	"math/big"
	"strings"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/velocore-v2/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	amp             *big.Int
	lpTokenBalances map[string]*big.Int
	tokenInfo       map[string]tokenInfo
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var (
		tokenNbr = len(entityPool.Tokens)
		tokens   = make([]string, tokenNbr)
		reserves = make([]*big.Int, tokenNbr)

		extra Extra
	)

	if len(entityPool.Reserves) == tokenNbr && len(entityPool.Tokens) == tokenNbr {
		for i := 0; i < tokenNbr; i++ {
			tokens[i] = entityPool.Tokens[i].Address
			reserves[i] = bignumber.NewBig10(entityPool.Reserves[i])
		}
	}

	err := json.Unmarshal([]byte(entityPool.Extra), &extra)
	if err != nil {
		return nil, err
	}

	info := pool.PoolInfo{
		Address:    strings.ToLower(entityPool.Address),
		ReserveUsd: entityPool.ReserveUsd,
		SwapFee:    nil,
		Exchange:   entityPool.Exchange,
		Type:       entityPool.Type,
		Tokens:     tokens,
		Reserves:   reserves,
		Checked:    false,
	}

	return &PoolSimulator{
		Pool:            pool.Pool{Info: info},
		amp:             extra.Amp,
		lpTokenBalances: extra.LpTokenBalances,
		tokenInfo:       extra.TokenInfo,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	err := p.validateTokens([]string{tokenAmountIn.Token, tokenOut})
	if err != nil {
		return nil, err
	}

	amountOut, err := p.swap(tokenAmountIn.Token, tokenOut, tokenAmountIn.Amount)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut,
		},
		Fee: &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: integer.Zero(),
		},
		Gas:      defaultGas,
		SwapInfo: nil,
	}, nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	for idx, token := range p.Info.Tokens {
		if token == params.TokenAmountIn.Token {
			p.Info.Reserves[idx] = new(big.Int).Add(p.Info.Reserves[idx], params.TokenAmountIn.Amount)
		}
		if token == params.TokenAmountOut.Token {
			p.Info.Reserves[idx] = new(big.Int).Sub(p.Info.Reserves[idx], params.TokenAmountOut.Amount)
		}
	}
}

func (t *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return nil
}

func (p *PoolSimulator) swap(
	k string,
	u string,
	dAk *big.Int,
) (*big.Int, error) {
	Au, Lu, Du, err := p.tokenStatScaled(u)
	if err != nil {
		return nil, err
	}

	Ak, Lk, Dk, err := p.tokenStatScaled(k)
	if err != nil {
		return nil, err
	}

	newDk, err := p.partialInvariant(
		new(big.Int).Add(Ak, p.upscale(k, dAk)),
		Lk,
	)
	if err != nil {
		return nil, err
	}

	newDu := new(big.Int).Sub(new(big.Int).Add(Dk, Du), newDk)
	_4ac := new(big.Int).Mul(
		new(big.Int).Quo(
			new(big.Int).Mul(new(big.Int).Mul(integer.Four(), p.amp), Lu),
			integer.TenPow(18),
		),
		Lu,
	)

	var newAu *big.Int
	{
		v, overflow := uint256.FromBig(new(big.Int).Add(
			new(big.Int).Mul(newDu, newDu),
			_4ac,
		))
		if overflow {
			return nil, ErrOverflow
		}

		newAu = new(big.Int).Quo(
			new(big.Int).Add(
				newDu,
				new(big.Int).Add(
					math.Common.SqrtRounding(v, true).ToBig(),
					integer.One(),
				),
			),
			integer.Two(),
		)
	}

	amountOut := p.downscale(u, new(big.Int).Sub(newAu, Au))
	amountOut.Neg(amountOut)

	return amountOut, nil
}

func (p *PoolSimulator) validateTokens(tokens []string) error {
	dup := make(map[string]struct{})
	for _, t := range tokens {
		if p.GetTokenIndex(t) < 0 {
			return ErrInvalidToken
		}
		if _, ok := dup[t]; ok {
			return ErrInvalidToken
		}
		dup[t] = struct{}{}
	}
	return nil
}

func (p *PoolSimulator) tokenStatScaled(token string) (*big.Int, *big.Int, *big.Int, error) {
	scale := integer.TenPow(p.tokenInfo[token].Scale)
	l := new(big.Int).Mul(
		new(big.Int).Sub(maxUint128, p.lpTokenBalances[token]),
		scale,
	)
	a := new(big.Int).Mul(p.getReserve(token), scale)
	partialInvariant, err := p.partialInvariant(a, l)
	if err != nil {
		return nil, nil, nil, err
	}
	return a, l, partialInvariant, nil
}

func (p *PoolSimulator) partialInvariant(a *big.Int, l *big.Int) (*big.Int, error) {
	if a.Cmp(integer.Zero()) == 0 {
		if l.Cmp(integer.Zero()) != 0 {
			return nil, ErrInvariant
		}
		return integer.Zero(), nil
	}
	return new(big.Int).Sub(
		a, new(big.Int).Quo(
			new(big.Int).Mul(
				new(big.Int).Quo(
					new(big.Int).Mul(l, p.amp),
					integer.TenPow(18),
				), l,
			), a,
		),
	), nil
}

func (p *PoolSimulator) upscale(t string, x *big.Int) *big.Int {
	return new(big.Int).Mul(x, integer.TenPow(p.tokenInfo[t].Scale))
}

func (p *PoolSimulator) downscale(t string, x *big.Int) *big.Int {
	return new(big.Int).Quo(x, integer.TenPow(p.tokenInfo[t].Scale))
}

func (p *PoolSimulator) getReserve(token string) *big.Int {
	idx := p.GetTokenIndex(token)
	return p.Info.Reserves[idx]
}
