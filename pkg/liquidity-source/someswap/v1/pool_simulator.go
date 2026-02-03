package someswapv1

import (
	"math/big"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	reserves [2]*uint256.Int
	feeBps   *uint256.Int
	StaticExtra
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     entityPool.Address,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      []string{entityPool.Tokens[0].Address, entityPool.Tokens[1].Address},
			Reserves:    []*big.Int{bignumber.NewBig(entityPool.Reserves[0]), bignumber.NewBig(entityPool.Reserves[1])},
			BlockNumber: entityPool.BlockNumber,
		}},
		reserves:    [2]*uint256.Int{big256.New(entityPool.Reserves[0]), big256.New(entityPool.Reserves[1])},
		feeBps:      uint256.NewInt(uint64(entityPool.SwapFee * bps)),
		StaticExtra: staticExtra,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := param.TokenAmountIn, param.TokenOut
	tokenInIndex, tokenOutIndex := p.GetTokenIndex(tokenAmountIn.Token), p.GetTokenIndex(tokenOut)
	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return nil, ErrInvalidTokenIndex
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow || amountIn.Sign() <= 0 {
		return nil, ErrInvalidAmountIn
	}

	netOut, outTotal, inFee, err := p.calcAmountOut(amountIn, tokenInIndex, tokenOutIndex)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: netOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: tokenAmountIn.Token, Amount: inFee.ToBig()},
		Gas:            defaultGas,
		SwapInfo:       lo.T2(outTotal, inFee),
	}, nil
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	return &cloned
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	tokenInIndex, tokenOutIndex := p.GetTokenIndex(params.TokenAmountIn.Token), p.GetTokenIndex(params.TokenAmountOut.Token)
	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return
	}

	amountIn := uint256.MustFromBig(params.TokenAmountIn.Amount)
	outTotal, inFee := params.SwapInfo.(lo.Tuple2[*uint256.Int, *uint256.Int]).Unpack()

	inDelta := amountIn.Sub(amountIn, inFee)
	p.reserves[tokenInIndex] = inDelta.Add(p.reserves[tokenInIndex], inDelta)
	p.reserves[tokenOutIndex] = outTotal.Sub(p.reserves[tokenOutIndex], outTotal)
}

func (p *PoolSimulator) GetMetaInfo(_, _ string) any {
	return nil
}

func (p *PoolSimulator) calcAmountOut(amountIn *uint256.Int, tokenInIndex, tokenOutIndex int) (*uint256.Int,
	*uint256.Int, *uint256.Int, error) {
	if amountIn == nil || amountIn.Sign() <= 0 {
		return nil, nil, nil, ErrInvalidAmountIn
	}

	reserveIn := p.reserves[tokenInIndex]
	reserveOut := p.reserves[tokenOutIndex]
	if reserveIn == nil || reserveOut == nil || reserveIn.Sign() <= 0 || reserveOut.Sign() <= 0 {
		return nil, nil, nil, ErrInvalidReserves
	}

	weight := p.WTokens[tokenInIndex]

	var inBps, outBps uint256.Int
	inBps.MulDivOverflow(p.feeBps, weight, weightDen)
	if inBps.Gt(p.feeBps) {
		inBps.Set(p.feeBps)
	}
	outBps.Sub(p.feeBps, &inBps)

	inFee, _ := inBps.MulDivOverflow(amountIn, &inBps, bpsDen)
	effIn := amountIn.Sub(amountIn, inFee)
	if effIn.Sign() <= 0 {
		return nil, nil, nil, ErrAmountInTooSmall
	}

	var outTotal uint256.Int
	denom := outTotal.Add(reserveIn, effIn)
	outTotal.MulDivOverflow(effIn, reserveOut, denom)
	if outTotal.Sign() <= 0 {
		return nil, nil, nil, ErrInvalidTotalOut
	}

	outFee, _ := outBps.MulDivOverflow(&outTotal, &outBps, bpsDen)
	netOut := outFee.Sub(&outTotal, outFee)
	if netOut.Sign() <= 0 {
		return nil, nil, nil, ErrInvalidNetOut
	}

	return netOut, &outTotal, inFee, nil
}
