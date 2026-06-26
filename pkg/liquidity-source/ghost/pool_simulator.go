package ghost

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
)

type PoolSimulator struct {
	pool.Pool

	staticExtra StaticExtra

	maxFee           *uint256.Int
	halfAmount       *uint256.Int
	reserve          *uint256.Int // nil = no limit
	scaleNumerator   *uint256.Int
	scaleDenominator *uint256.Int

	gas int64
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	if len(entityPool.Tokens) != 2 || len(entityPool.Reserves) != 2 {
		return nil, fmt.Errorf("ghost: pool %s requires exactly 2 tokens", entityPool.Address)
	}

	var se StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &se); err != nil {
		return nil, fmt.Errorf("ghost: unmarshal StaticExtra: %w", err)
	}

	var ex Extra
	if entityPool.Extra != "" && entityPool.Extra != "{}" {
		if err := json.Unmarshal([]byte(entityPool.Extra), &ex); err != nil {
			return nil, fmt.Errorf("ghost: unmarshal Extra: %w", err)
		}
	}

	tokens := make([]string, 2)
	reserves := make([]*big.Int, 2)
	for i := 0; i < 2; i++ {
		tokens[i] = entityPool.Tokens[i].Address
		reserves[i] = bignumber.NewBig10(entityPool.Reserves[i])
	}

	maxFee := bigToUint256OrZero(ex.MaxFee)
	halfAmount := bigToUint256OrZero(ex.HalfAmount)

	var reserve *uint256.Int
	if ex.Reserve != nil {
		if r, overflow := uint256.FromBig(ex.Reserve); !overflow {
			reserve = r
		}
	}

	scaleNum := uint256.NewInt(1)
	if se.ScaleNumerator != "" {
		if n, err := uint256.FromDecimal(se.ScaleNumerator); err == nil {
			scaleNum = n
		}
	}
	scaleDen := uint256.NewInt(1)
	if se.ScaleDenominator != "" {
		if d, err := uint256.FromDecimal(se.ScaleDenominator); err == nil && !d.IsZero() {
			scaleDen = d
		}
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
		staticExtra:      se,
		maxFee:           maxFee,
		halfAmount:       halfAmount,
		reserve:          reserve,
		scaleNumerator:   scaleNum,
		scaleDenominator: scaleDen,
		gas:              DefaultGas,
	}, nil
}

func bigToUint256OrZero(b *big.Int) *uint256.Int {
	if b == nil {
		return new(uint256.Int)
	}
	u, _ := uint256.FromBig(b)
	return u
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn := param.TokenAmountIn.Token
	tokenOut := param.TokenOut
	amountInBig := param.TokenAmountIn.Amount

	if p.GetTokenIndex(tokenIn) != 0 || p.GetTokenIndex(tokenOut) != 1 {
		return nil, ErrInvalidToken
	}

	if amountInBig == nil || amountInBig.Sign() <= 0 {
		return nil, ErrInvalidToken
	}

	amountIn, overflow := uint256.FromBig(amountInBig)
	if overflow {
		return nil, ErrOverflow
	}

	principal, fee := inverseFee(amountIn, p.maxFee, p.halfAmount)
	if principal.IsZero() {
		return nil, ErrInsufficientLiquidity
	}

	var amountOut uint256.Int
	if _, overflow := amountOut.MulDivOverflow(principal, p.scaleNumerator, p.scaleDenominator); overflow {
		return nil, ErrOverflow
	}

	if p.reserve != nil && amountOut.Gt(p.reserve) {
		return nil, ErrInsufficientLiquidity
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: fee.ToBig(),
		},
		Gas: p.gas,
	}, nil
}

// inverseFee solves for the largest principal such that principal + fee(principal)
// <= amountIn, recovering the integer-division dust so the quoted principal equals
// what the swap transfers on-chain.
//
//	Linear region:  principal = amountIn * 2 * halfAmount / (2 * halfAmount + maxFee), then +1 dust recovery
//	Capped region:  principal = amountIn - maxFee
func inverseFee(amountIn, maxFee, halfAmount *uint256.Int) (principal, fee *uint256.Int) {
	if maxFee.IsZero() || halfAmount.IsZero() {
		return amountIn.Clone(), new(uint256.Int)
	}

	var twoHalf uint256.Int
	twoHalf.Mul(halfAmount, uint256.NewInt(2))

	if !amountIn.Lt(maxFee) {
		var cappedPrincipal uint256.Int
		cappedPrincipal.Sub(amountIn, maxFee)
		if !cappedPrincipal.Lt(&twoHalf) {
			return cappedPrincipal.Clone(), maxFee.Clone()
		}
	}

	var denom uint256.Int
	denom.Add(&twoHalf, maxFee)

	principal = new(uint256.Int)
	if _, overflow := principal.MulDivOverflow(amountIn, &twoHalf, &denom); overflow {
		return new(uint256.Int), new(uint256.Int)
	}

	// Recover up to 1 unit of integer-division dust when principal+1 still fits
	// within amountIn, so the quoted principal matches what the swap transfers
	// on-chain. We are provably in the linear region here (principal < twoHalf),
	// so fee(principal+1) scales linearly and cannot exceed maxFee — no cap check
	// needed.
	var candidate uint256.Int
	candidate.AddUint64(principal, 1)

	var feeUp uint256.Int
	if _, overflow := feeUp.MulDivOverflow(&candidate, maxFee, &twoHalf); !overflow {
		var total uint256.Int
		total.Add(&candidate, &feeUp)
		if !total.Gt(amountIn) {
			principal.Set(&candidate)
		}
	}

	fee = calcFee(principal, maxFee, halfAmount)
	return principal, fee
}

// calcFee implements: fee = min(maxFee, amount * maxFee / (2 * halfAmount))
func calcFee(amount, maxFee, halfAmount *uint256.Int) *uint256.Int {
	if maxFee.IsZero() || halfAmount.IsZero() {
		return new(uint256.Int)
	}

	var twoHalf uint256.Int
	twoHalf.Mul(halfAmount, uint256.NewInt(2))

	var linearFee uint256.Int
	if _, overflow := linearFee.MulDivOverflow(amount, maxFee, &twoHalf); overflow {
		return maxFee.Clone()
	}

	if linearFee.Gt(maxFee) {
		return maxFee.Clone()
	}
	return linearFee.Clone()
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	amountOutBig := params.TokenAmountOut.Amount
	if amountOutBig == nil {
		return
	}
	if p.reserve != nil {
		if amountOut, overflow := uint256.FromBig(amountOutBig); !overflow {
			p.reserve = new(uint256.Int).Sub(p.reserve, amountOut)
		}
	}
	if len(p.Info.Reserves) > 1 {
		p.Info.Reserves[1] = new(big.Int).Sub(p.Info.Reserves[1], amountOutBig)
	}
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.Info.Reserves = slices.Clone(p.Info.Reserves)
	if p.reserve != nil {
		cloned.reserve = p.reserve.Clone()
	}
	return &cloned
}

func (p *PoolSimulator) CanSwapTo(address string) []string {
	if strings.EqualFold(p.Info.Tokens[1], address) {
		return []string{p.Info.Tokens[0]}
	}
	return nil
}

func (p *PoolSimulator) CanSwapFrom(address string) []string {
	if strings.EqualFold(p.Info.Tokens[0], address) {
		return []string{p.Info.Tokens[1]}
	}
	return nil
}

func (p *PoolSimulator) CalculateLimit() map[string]*big.Int {
	if p.reserve == nil {
		return nil
	}
	return map[string]*big.Int{
		p.Info.Tokens[1]: p.reserve.ToBig(),
	}
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) any {
	return PoolMeta{
		SourceRouter: p.staticExtra.SourceRouter,
		TargetRouter: p.staticExtra.TargetRouter,
	}
}
