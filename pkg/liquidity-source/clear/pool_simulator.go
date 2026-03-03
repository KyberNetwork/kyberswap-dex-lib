package clear

import (
	"math/big"
	"slices"
	"strings"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	extra Extra
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if len(entityPool.Extra) > 0 {
		if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
			return nil, err
		}
	}

	// Parse reserves from entity
	reserves := make([]*big.Int, len(entityPool.Tokens))
	for i, r := range entityPool.Reserves {
		reserve, ok := new(big.Int).SetString(r, 10)
		if !ok {
			reserve = big.NewInt(0)
		}
		reserves[i] = reserve
	}

	info := pool.PoolInfo{
		Address:  strings.ToLower(entityPool.Address),
		Exchange: entityPool.Exchange,
		Type:     entityPool.Type,
		Tokens:   lo.Map(entityPool.Tokens, func(e *entity.PoolToken, _ int) string { return e.Address }),
		Reserves: reserves,
	}

	return &PoolSimulator{
		Pool:  pool.Pool{Info: info},
		extra: extra,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := param.TokenAmountIn, param.TokenOut
	// Validate tokens
	tokenInIndex, tokenOutIndex := p.GetTokenIndex(tokenAmountIn.Token), p.GetTokenIndex(tokenOut)
	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return nil, ErrInvalidToken
	} else if len(p.Info.Tokens) != len(p.extra.IOUs) {
		return nil, ErrInvalidIOUToken
	} else if tokenAmountIn.Amount == nil || tokenAmountIn.Amount.Sign() <= 0 {
		return nil, ErrInvalidAmountIn
	}

	// For Clear, we need to call previewSwap on-chain to get the exact output
	// Since we can't make RPC calls during simulation, we use the cached rate
	// The actual rate will be verified during execution

	// Estimate output based on cached reserves ratio
	// This is an approximation - actual output comes from previewSwap
	amountOut := p.estimateAmountOut(tokenInIndex, tokenOutIndex, tokenAmountIn.Amount)
	if amountOut.Cmp(p.GetReserves()[tokenOutIndex]) > 0 {
		return nil, ErrInvalidAmountOut
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut},
		Fee:            &pool.TokenAmount{Token: tokenOut, Amount: bignumber.ZeroBI}, // Clear handles fees internally
		SwapInfo: SwapInfo{
			SwapAddress: p.extra.SwapAddress,
			IOU:         p.extra.IOUs[tokenOutIndex],
		},
		Gas: defaultGas,
	}, nil
}

// estimateAmountOut estimates the output amount based on cached data
// For Clear protocol, this is an approximation since actual pricing requires RPC
func (p *PoolSimulator) estimateAmountOut(tokenInIndex, tokenOutIndex int, amountIn *big.Int) *big.Int {
	if p.extra.Rates == nil {
		return new(big.Int)
	}
	rate := p.extra.Rates[tokenInIndex][tokenOutIndex]
	if rate[1] == nil || rate[1].Sign() == 0 {
		return new(big.Int)
	}

	// Simple ratio calculation: amountOut = amountIn * reserveOut / reserveIn
	amtIn := uint256.MustFromBig(amountIn)
	amtOut, _ := amtIn.MulDivOverflow(amtIn, rate[1], rate[0])
	return amtOut.ToBig()
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	return &cloned
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	tokenAmtIn, tokenAmtOut := params.TokenAmountIn, params.TokenAmountOut
	inIndex, outIndex := p.GetTokenIndex(tokenAmtIn.Token), p.GetTokenIndex(tokenAmtOut.Token)
	p.Info.Reserves = slices.Clone(p.Info.Reserves)
	p.Info.Reserves[inIndex] = new(big.Int).Add(p.Info.Reserves[inIndex], tokenAmtIn.Amount)
	p.Info.Reserves[outIndex] = new(big.Int).Sub(p.Info.Reserves[outIndex], tokenAmtOut.Amount)
}

func (p *PoolSimulator) GetMetaInfo(_, _ string) any {
	return nil
}
