package wombatstable

import (
	"errors"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/velocore-v2/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ErrInvalidToken         = errors.New("invalid token")
	ErrInvariant            = errors.New("invariant")
	ErrOverflow             = errors.New("overflow")
	ErrNonPositiveAmountOut = errors.New("non positive amount out")
)

type PoolSimulator struct {
	pool.Pool

	amp             *big.Int
	lpTokenBalances map[string]*big.Int
	tokenInfo       map[string]tokenInfo

	vault    string
	wrappers map[string]string
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var (
		tokenNbr = len(entityPool.Tokens)
		tokens   = make([]string, tokenNbr)
		reserves = make([]*big.Int, tokenNbr)

		extra       Extra
		staticExtra StaticExtra
	)

	for i := 0; i < tokenNbr; i++ {
		tokens[i] = entityPool.Tokens[i].Address
		reserves[i] = bignumber.NewBig10(entityPool.Reserves[i])
	}

	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	info := pool.PoolInfo{
		Address:    entityPool.Address,
		ReserveUsd: entityPool.ReserveUsd,
		SwapFee:    nil,
		Exchange:   entityPool.Exchange,
		Type:       entityPool.Type,
		Tokens:     tokens,
		Reserves:   reserves,
		Checked:    true,
	}

	return &PoolSimulator{
		Pool:            pool.Pool{Info: info},
		amp:             extra.Amp,
		lpTokenBalances: extra.LpTokenBalances,
		tokenInfo:       extra.TokenInfo,
		vault:           staticExtra.Vault,
		wrappers:        staticExtra.Wrappers,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := params.TokenAmountIn, params.TokenOut

	err := p.validateTokens([]string{tokenAmountIn.Token, tokenOut})
	if err != nil {
		return nil, err
	}

	// NOTE: LP swap is not supported
	// because it does not have IERC20 standard.
	amountOut, err := p.swap(tokenAmountIn.Token, tokenOut, tokenAmountIn.Amount)
	if err != nil {
		return nil, err
	}

	amountOut = new(big.Int).Neg(amountOut)
	if amountOut.Cmp(integer.Zero()) <= 0 {
		return nil, ErrNonPositiveAmountOut
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
		Gas:      p.gas(params),
		SwapInfo: nil,
	}, nil
}

func (p *PoolSimulator) gas(params pool.CalcAmountOutParams) int64 {
	// NOTE: with sc execution, tx failed if swap between
	// USDT+ and USD+ through a pool containing them.

	if _, ok := p.wrappers[params.TokenAmountIn.Token]; ok {
		return defaultGas.SwapConvertIn
	}

	if _, ok := p.wrappers[params.TokenOut]; ok {
		return defaultGas.SwapConvertOut
	}

	return defaultGas.SwapNoConvert
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
	return Meta{
		Vault:    t.vault,
		Wrappers: t.wrappers,
	}
}

// https://github.com/velocore/velocore-contracts/blob/c29678e5acbe5e60fc018e08289b49e53e1492f3/src/pools/wombat/WombatPool.sol#L164
func (p *PoolSimulator) swap(k string, u string, dAk *big.Int) (*big.Int, error) {
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
			bignumber.BONE,
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

	return p.downscale(u, new(big.Int).Sub(newAu, Au)), nil
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

// https://github.com/velocore/velocore-contracts/blob/c29678e5acbe5e60fc018e08289b49e53e1492f3/src/pools/wombat/WombatPool.sol#L145
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

// https://github.com/velocore/velocore-contracts/blob/c29678e5acbe5e60fc018e08289b49e53e1492f3/src/pools/wombat/WombatPool.sol#L137
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
					bignumber.BONE,
				), l,
			), a,
		),
	), nil
}

// https://github.com/velocore/velocore-contracts/blob/c29678e5acbe5e60fc018e08289b49e53e1492f3/src/pools/wombat/WombatPool.sol#L152
func (p *PoolSimulator) upscale(t string, x *big.Int) *big.Int {
	return new(big.Int).Mul(x, integer.TenPow(p.tokenInfo[t].Scale))
}

// https://github.com/velocore/velocore-contracts/blob/c29678e5acbe5e60fc018e08289b49e53e1492f3/src/pools/wombat/WombatPool.sol#L158
func (p *PoolSimulator) downscale(t string, x *big.Int) *big.Int {
	return new(big.Int).Quo(x, integer.TenPow(p.tokenInfo[t].Scale))
}

func (p *PoolSimulator) getReserve(token string) *big.Int {
	idx := p.GetTokenIndex(token)
	return p.Info.Reserves[idx]
}
