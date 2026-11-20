package generic_simple_rate

import (
	"fmt"
	"math/big"
	"slices"
	"strings"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	pool.Pool
	paused          bool
	gas             int64
	rate            *uint256.Int
	rateUnit        *uint256.Int
	isRateInversed  bool
	isBidirectional bool
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

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
	for i := range numTokens {
		tokens[i] = entityPool.Tokens[i].Address
		reserves[i] = bignumber.NewBig10(entityPool.Reserves[i])
	}

	var poolExtra PoolExtra
	if err := json.Unmarshal([]byte(entityPool.Extra), &poolExtra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Info: pool.PoolInfo{
			Address:     strings.ToLower(entityPool.Address),
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      tokens,
			Reserves:    reserves,
			BlockNumber: entityPool.BlockNumber,
		},
		paused:          poolExtra.Paused,
		rate:            poolExtra.Rate,
		rateUnit:        poolExtra.RateUnit,
		isRateInversed:  poolExtra.IsRateInversed,
		isBidirectional: poolExtra.IsBidirectional,
		gas:             poolExtra.DefaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmtOut(param pool.CalcAmtOutParams) (*pool.CalcAmtOutResult, error) {
	if p.paused {
		return nil, ErrPoolPaused
	}

	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut

	var tokenInIndex = p.GetTokenIndex(tokenAmountIn.Token)
	var tokenOutIndex = p.GetTokenIndex(tokenOut)

	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return &pool.CalcAmtOutResult{}, fmt.Errorf("tokenInIndex: %v or tokenOutIndex: %v is not correct", tokenInIndex, tokenOutIndex)
	}

	amountOut := p.calcAmountOut(tokenInIndex, &tokenAmountIn.Amount)

	return &pool.CalcAmtOutResult{
		TokenAmountOut: &pool.TokenAmt{
			Token:  tokenOut,
			Amount: *amountOut,
		},
		Fee: &pool.TokenAmt{
			Token: tokenAmountIn.Token,
		},
		Gas: p.gas,
	}, nil
}

var _ pool.IPoolExactOutSimulator = (*PoolSimulator)(nil)

// CalcAmountIn returns the minimal amountIn for which CalcAmountOut gives at least TokenAmountOut.
// Reserves are not checked, as CalcAmountOut ignores them.
func (p *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	if p.paused {
		return nil, ErrPoolPaused
	}

	tokenAmountOut := param.TokenAmountOut
	tokenInIndex := p.GetTokenIndex(param.TokenIn)
	tokenOutIndex := p.GetTokenIndex(tokenAmountOut.Token)
	if tokenInIndex < 0 || tokenOutIndex < 0 || tokenInIndex == tokenOutIndex {
		return nil, fmt.Errorf("invalid token pair %v (index %d) -> %v (index %d)", param.TokenIn, tokenInIndex, tokenAmountOut.Token, tokenOutIndex)
	}
	if tokenInIndex == 1 && !p.isBidirectional {
		return nil, ErrUnidirectional
	}

	amountIn, err := p.calcAmountIn(tokenInIndex, tokenAmountOut.Amount)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{
			Token:  param.TokenIn,
			Amount: amountIn.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  param.TokenIn,
			Amount: bignumber.ZeroBI,
		},
		Gas: p.gas,
	}, nil
}

func (p *PoolSimulator) UpdateState(_ pool.UpdateStateParams) {
}

func (p *PoolSimulator) CloneState() pool.IPoolSim {
	cloned := *p
	cloned.Info.Reserves = slices.Clone(p.Info.Reserves)
	return &cloned
}

func (p *PoolSimulator) GetReserves() []uint256.Int {
	out := make([]uint256.Int, len(p.Info.Reserves))
	for i, r := range p.Info.Reserves {
		out[i] = *reserveToU256(r)
	}
	return out
}

// CalcLimit returns nil: generic-simple-rate has no shared inventory limit.
func (p *PoolSimulator) CalcLimit() map[string]*uint256.Int {
	return nil
}

// reserveToU256 converts a reserve. A nil, negative, or oversized reserve becomes zero.
func reserveToU256(b *big.Int) *uint256.Int {
	if b == nil || b.Sign() < 0 {
		return new(uint256.Int)
	}
	u, overflow := uint256.FromBig(b)
	if overflow {
		return new(uint256.Int)
	}
	return u
}

func (p *PoolSimulator) CanSwapTo(address string) []string {
	if strings.EqualFold(p.Info.Tokens[1], address) {
		return []string{p.Info.Tokens[0]}
	}
	if p.isBidirectional && strings.EqualFold(p.Info.Tokens[0], address) {
		return []string{p.Info.Tokens[1]}
	}
	return []string{}
}

func (p *PoolSimulator) CanSwapFrom(address string) []string {
	if strings.EqualFold(p.Info.Tokens[0], address) {
		return []string{p.Info.Tokens[1]}
	}
	if p.isBidirectional && strings.EqualFold(p.Info.Tokens[1], address) {
		return []string{p.Info.Tokens[0]}
	}
	return []string{}
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) any {
	return pool.MetaInfo{BlockNumber: p.Info.BlockNumber}
}

func (p *PoolSimulator) SwapReceiveNativeIn(tokenIn, _ string, chainId valueobject.ChainID) bool {
	_, ok := supportNativeSwapExchanges[p.Info.Exchange]
	return ok && valueobject.IsWrappedNative(tokenIn, chainId)
}

func (p *PoolSimulator) SwapReturnNativeOut(_, tokenOut string, chainId valueobject.ChainID) bool {
	_, ok := supportNativeSwapExchanges[p.Info.Exchange]
	return ok && valueobject.IsWrappedNative(tokenOut, chainId)
}

func (p *PoolSimulator) calcAmountOut(tokenInIndex int, amountIn *uint256.Int) *uint256.Int {
	amountOut := new(uint256.Int).Set(amountIn)
	if p.isRateInversed == (tokenInIndex == 0) {
		amountOut = amountOut.Mul(amountOut, p.rateUnit).Div(amountOut, p.rate)
	} else {
		amountOut = amountOut.Div(amountOut, p.rateUnit).Mul(amountOut, p.rate)
	}
	return amountOut
}

// calcAmountIn inverts calcAmountOut with ceiling rounding.
func (p *PoolSimulator) calcAmountIn(tokenInIndex int, amountOut *big.Int) (*uint256.Int, error) {
	var out uint256.Int
	if out.SetFromBig(amountOut) {
		return nil, ErrOverflow
	}
	if out.IsZero() || p.rate.IsZero() || p.rateUnit.IsZero() {
		return nil, ErrZeroAmount
	}

	amountIn := new(uint256.Int)
	if p.isRateInversed == (tokenInIndex == 0) {
		// out = in*unit/rate  =>  in = ceil(out*rate/unit)
		_, overflow := amountIn.MulDivOverflow(&out, p.rate, p.rateUnit)
		if overflow {
			return nil, ErrOverflow
		}
		var rem uint256.Int
		if !rem.MulMod(&out, p.rate, p.rateUnit).IsZero() {
			if _, carry := amountIn.AddOverflow(amountIn, uint256.NewInt(1)); carry {
				return nil, ErrOverflow
			}
		}
	} else {
		// out = (in/unit)*rate  =>  in = ceil(out/rate)*unit
		var rem uint256.Int
		amountIn.DivMod(&out, p.rate, &rem)
		if !rem.IsZero() {
			amountIn.AddUint64(amountIn, 1)
		}
		if _, overflow := amountIn.MulOverflow(amountIn, p.rateUnit); overflow {
			return nil, ErrOverflow
		}
	}
	return amountIn, nil
}
