package equalizer

import (
	"fmt"
	"math/big"
	"slices"
	"strings"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	stable   bool
	swapFee  *uint256.Int
	reserves []*uint256.Int
	decimals []*uint256.Int
	gas      Gas
}

var _ = pool.RegisterFactory0(DexTypeEqualizer, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	swapFeeFloat := new(big.Float).Mul(big.NewFloat(entityPool.SwapFee), bignumber.BoneFloat)
	swapFeeBig, _ := swapFeeFloat.Int(nil)
	swapFee, overflow := uint256.FromBig(swapFeeBig)
	if overflow {
		return nil, fmt.Errorf("swapFee overflow")
	}

	tokens := make([]string, 2)
	tokens[0] = entityPool.Tokens[0].Address
	tokens[1] = entityPool.Tokens[1].Address

	bigReserves := make([]*big.Int, 2)
	reserves := make([]*uint256.Int, 2)
	decimals := make([]*uint256.Int, 2)
	for i := range 2 {
		bigReserves[i] = bignumber.NewBig10(entityPool.Reserves[i])
		var err error
		if reserves[i], err = uint256.FromDecimal(entityPool.Reserves[i]); err != nil {
			return nil, fmt.Errorf("invalid reserve: %w", err)
		}
		decimals[i] = big256.TenPow(entityPool.Tokens[i].Decimals)
	}

	info := pool.PoolInfo{
		Address:  strings.ToLower(entityPool.Address),
		SwapFee:  swapFeeBig,
		Exchange: entityPool.Exchange,
		Type:     entityPool.Type,
		Tokens:   tokens,
		Reserves: bigReserves,
	}

	staticExtra, err := extractStaticExtra(entityPool.StaticExtra)
	if err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool:     pool.Pool{Info: info},
		stable:   staticExtra.Stable,
		swapFee:  swapFee,
		reserves: reserves,
		decimals: decimals,
		gas:      DefaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut
	tokenInIndex := p.GetTokenIndex(tokenAmountIn.Token)
	tokenOutIndex := p.GetTokenIndex(tokenOut)

	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return &pool.CalcAmountOutResult{}, fmt.Errorf("tokenInIndex %v or tokenOutIndex %v is not correct", tokenInIndex, tokenOutIndex)
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return &pool.CalcAmountOutResult{}, fmt.Errorf("amountIn overflow")
	}

	amountOut, err := getAmountOut(
		amountIn,
		p.reserves[tokenInIndex],
		p.reserves[tokenOutIndex],
		p.decimals[tokenInIndex],
		p.decimals[tokenOutIndex],
		p.swapFee,
		p.stable,
	)
	if err != nil {
		return &pool.CalcAmountOutResult{}, err
	}

	if amountOut.IsZero() {
		return &pool.CalcAmountOutResult{}, fmt.Errorf("amountOut is 0")
	}

	if amountOut.Cmp(p.reserves[tokenOutIndex]) > 0 {
		return &pool.CalcAmountOutResult{}, fmt.Errorf("amountOut exceeds reserve")
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: tokenAmountIn.Token, Amount: nil},
		Gas:            p.gas.Swap,
	}, nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn := p.GetTokenIndex(params.TokenAmountIn.Token)
	indexOut := p.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return
	}

	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow {
		return
	}
	amountOut, overflow := uint256.FromBig(params.TokenAmountOut.Amount)
	if overflow {
		return
	}

	amountInAfterFee := calAmountAfterFee(amountIn, p.swapFee)

	p.reserves[indexIn] = new(uint256.Int).Add(p.reserves[indexIn], amountInAfterFee)
	p.reserves[indexOut] = new(uint256.Int).Sub(p.reserves[indexOut], amountOut)
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.reserves = slices.Clone(p.reserves)
	return &cloned
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) any {
	return StaticExtra{Stable: p.stable}
}
