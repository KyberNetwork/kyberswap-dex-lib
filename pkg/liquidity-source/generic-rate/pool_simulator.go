package generic_rate

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	skypsm "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/maker/sky-psm"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	pool.Pool
	paused       bool
	swapFuncArgs []*uint256.Int
	swapFuncData SwapFuncData
	gas          int64
}

var (
	ErrPoolPaused      = errors.New("pool is paused")
	ErrOverflow        = errors.New("overflow")
	ErrInvalidFunction = errors.New("invalid function")
	ErrInvalidRateArgs = errors.New("invalid rate args")
)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	numTokens := len(entityPool.Tokens)
	if numTokens != 2 {
		return nil, fmt.Errorf("invalid pool tokens %v, %v", entityPool, numTokens)
	}
	if numTokens != len(entityPool.Reserves) {
		return nil, fmt.Errorf("invalid pool reserves %v, %v", entityPool, numTokens)
	}

	var tokens = make([]string, numTokens)
	var reserves = make([]*big.Int, numTokens)
	for i := 0; i < numTokens; i += 1 {
		tokens[i] = entityPool.Tokens[i].Address
		reserves[i] = bignumber.NewBig10(entityPool.Reserves[i])
	}

	var poolExtra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &poolExtra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:  strings.ToLower(entityPool.Address),
				Exchange: entityPool.Exchange,
				Type:     entityPool.Type,
				Tokens:   tokens,
				Reserves: reserves,
			},
		},
		paused:       poolExtra.Paused,
		swapFuncArgs: poolExtra.SwapFuncArgs,
		swapFuncData: poolExtra.SwapFuncByPair,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if p.paused {
		return nil, ErrPoolPaused
	}

	var inIdx, outIdx = p.GetTokenIndex(params.TokenAmountIn.Token), p.GetTokenIndex(params.TokenOut)
	if inIdx < 0 || outIdx < 0 {
		return &pool.CalcAmountOutResult{}, fmt.Errorf("inIdx: %v or outIdx: %v is not correct", inIdx, outIdx)
	}

	amountIn := new(uint256.Int)
	if amountIn.SetFromBig(params.TokenAmountIn.Amount) {
		return nil, ErrOverflow
	}

	amountOut, err := p.swap(inIdx, outIdx, amountIn)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  params.TokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  params.TokenAmountIn.Token,
			Amount: bignumber.ZeroBI,
		},
		Gas: p.gas,
	}, nil
}

func (p *PoolSimulator) UpdateBalance(_ pool.UpdateBalanceParams) {
}

func (p *PoolSimulator) CanSwapTo(address string) []string {
	toIdx := p.GetTokenIndex(address)
	if toIdx < 0 {
		return []string{}
	}
	addresses := make([]string, 0)
	for i := 0; i < len(p.Info.Tokens); i += 1 {
		if _, ok := p.swapFuncData[i][toIdx]; ok {
			addresses = append(addresses, p.Info.Tokens[i])
		}
	}
	return addresses
}

func (p *PoolSimulator) CanSwapFrom(address string) []string {
	fromIdx := p.GetTokenIndex(address)
	if fromIdx < 0 {
		return []string{}
	}
	addresses := make([]string, 0)
	for i := 0; i < len(p.Info.Tokens); i += 1 {
		if _, ok := p.swapFuncData[fromIdx][i]; ok {
			addresses = append(addresses, p.Info.Tokens[i])
		}
	}
	return addresses
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return nil
}

func (p *PoolSimulator) swap(inIdx int, outIdx int, amountIn *uint256.Int) (*uint256.Int, error) {
	rd, ok := p.swapFuncData[inIdx][outIdx]
	if !ok || len(rd.Func) == 0 {
		return nil, ErrInvalidFunction
	}

	var fn func(amountIn *uint256.Int, args ...*uint256.Int) (*uint256.Int, error)

	switch valueobject.Exchange(p.Info.Exchange) {
	case valueobject.ExchangeSkyPSM:
		fn, ok = skypsm.FuncMap[rd.Func]
		if !ok {
			return nil, ErrInvalidFunction
		}
	}

	if fn == nil {
		return nil, ErrInvalidFunction
	}

	funcRateArgs, err := p.getRateArgsForFunc(rd.ArgIdxes)
	if err != nil {
		return nil, err
	}

	return fn(amountIn, funcRateArgs...)
}

func (p *PoolSimulator) getRateArgsForFunc(argIdxes []int) ([]*uint256.Int, error) {
	args := make([]*uint256.Int, len(argIdxes))
	for i, idx := range argIdxes {
		if idx < 0 || idx >= len(p.swapFuncArgs) {
			return nil, ErrInvalidRateArgs
		}
		args[i] = p.swapFuncArgs[idx]
	}
	return args, nil
}
