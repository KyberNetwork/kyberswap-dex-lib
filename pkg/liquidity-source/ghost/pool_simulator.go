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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// directionState holds one swap direction's fee curve, reserve limit, and router/scale info.
type directionState struct {
	static           DirectionStatic
	maxFee           *uint256.Int
	halfAmount       *uint256.Int
	reserve          *uint256.Int // nil = no limit
	scaleNumerator   *uint256.Int
	scaleDenominator *uint256.Int
}

type PoolSimulator struct {
	pool.Pool

	// zeroToOne is tokens[0] -> tokens[1]; oneToZero is tokens[1] -> tokens[0]. The two are
	// independent on-chain calls (own sourceRouter/fee-curve/targetRouter reserve) that happen
	// to connect the same token pair — not a shared AMM reserve.
	zeroToOne directionState
	oneToZero directionState

	gas int64
}

var (
	_ = pool.RegisterFactory0(DexType, NewPoolSimulator)
	_ = pool.RegisterUseSwapLimit(valueobject.ExchangeGhost)
)

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
		zeroToOne: buildDirectionState(se.ZeroToOne, ex.ZeroToOne),
		oneToZero: buildDirectionState(se.OneToZero, ex.OneToZero),
		gas:       DefaultGas,
	}, nil
}

func buildDirectionState(se DirectionStatic, ex DirectionExtra) directionState {
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

	var reserve *uint256.Int
	if ex.Reserve != nil {
		if r, overflow := uint256.FromBig(ex.Reserve); !overflow {
			reserve = r
		}
	}

	return directionState{
		static:           se,
		maxFee:           bigToUint256OrZero(ex.MaxFee),
		halfAmount:       bigToUint256OrZero(ex.HalfAmount),
		reserve:          reserve,
		scaleNumerator:   scaleNum,
		scaleDenominator: scaleDen,
	}
}

func bigToUint256OrZero(b *big.Int) *uint256.Int {
	if b == nil {
		return new(uint256.Int)
	}
	u, _ := uint256.FromBig(b)
	return u
}

// directionFor resolves which direction a swap uses from tokenIn, returning the direction
// state along with the tokenIn/tokenOut indices into p.Info.Tokens.
func (p *PoolSimulator) directionFor(tokenIn string) (dir *directionState, idxIn, idxOut int, ok bool) {
	switch {
	case strings.EqualFold(p.Info.Tokens[0], tokenIn):
		return &p.zeroToOne, 0, 1, true
	case strings.EqualFold(p.Info.Tokens[1], tokenIn):
		return &p.oneToZero, 1, 0, true
	default:
		return nil, 0, 0, false
	}
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn := param.TokenAmountIn.Token
	tokenOut := param.TokenOut
	amountInBig := param.TokenAmountIn.Amount

	dir, _, idxOut, ok := p.directionFor(tokenIn)
	if !ok || !strings.EqualFold(p.Info.Tokens[idxOut], tokenOut) {
		return nil, ErrInvalidToken
	}

	if amountInBig == nil || amountInBig.Sign() <= 0 {
		return nil, ErrInvalidToken
	}

	amountIn, overflow := uint256.FromBig(amountInBig)
	if overflow {
		return nil, ErrOverflow
	}

	principal, fee := inverseFee(amountIn, dir.maxFee, dir.halfAmount)
	if principal.IsZero() {
		return nil, ErrInsufficientLiquidity
	}

	var amountOut uint256.Int
	if _, overflow := amountOut.MulDivOverflow(principal, dir.scaleNumerator, dir.scaleDenominator); overflow {
		return nil, ErrOverflow
	}

	if dir.reserve != nil && amountOut.Gt(dir.reserve) {
		return nil, ErrInsufficientLiquidity
	}

	// Multiple ghost pools can share the same underlying targetRouter/token vault balance, so
	// param.Limit (aggregated across all ghost pools by the pathfinder) guards against
	// overselling that shared inventory within a single route build.
	if limit := param.Limit; limit != nil {
		if inventoryLimit := limit.GetLimit(tokenOut); inventoryLimit != nil && amountOut.ToBig().Cmp(inventoryLimit) > 0 {
			return nil, ErrInsufficientLiquidity
		}
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
		Gas:      p.gas,
		SwapInfo: SwapInfo{TotalFeeBps: totalFeeBps(principal, fee)},
	}, nil
}

// totalFeeBps is the feeBps executeGhost needs to recover `principal` from `amountIn` on-chain
// (amount = mulDivRoundingUp(amountIn+1, D, D+totalFeeBps) - 1). Rounded UP (never down): a
// floor-rounded value would make the on-chain amount come out larger than principal, so
// transferRemoteTo would try to pull more than the executor's amountIn balance and revert.
// Rounding up guarantees on-chain amount <= principal, at the cost of ghost keeping at most a
// few wei of extra dust as fee.
func totalFeeBps(principal, fee *uint256.Int) int64 {
	if principal.IsZero() {
		return 0
	}

	num := new(uint256.Int).Mul(fee, uint256.NewInt(uint64(GhostFeeDenominator)))
	var totalFeeBps, rem uint256.Int
	totalFeeBps.DivMod(num, principal, &rem)
	if !rem.IsZero() {
		totalFeeBps.AddUint64(&totalFeeBps, 1)
	}
	return int64(totalFeeBps.Uint64())
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

	dir, _, idxOut, ok := p.directionFor(params.TokenAmountIn.Token)
	if !ok {
		return
	}

	if dir.reserve != nil {
		if amountOut, overflow := uint256.FromBig(amountOutBig); !overflow {
			dir.reserve = new(uint256.Int).Sub(dir.reserve, amountOut)
		}
	}
	p.Info.Reserves[idxOut] = new(big.Int).Sub(p.Info.Reserves[idxOut], amountOutBig)

	if limit := params.SwapLimit; limit != nil {
		_, _, _ = limit.UpdateLimit(
			params.TokenAmountOut.Token, params.TokenAmountIn.Token,
			amountOutBig, params.TokenAmountIn.Amount,
		)
	}
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.Info.Reserves = slices.Clone(p.Info.Reserves)
	if p.zeroToOne.reserve != nil {
		cloned.zeroToOne.reserve = p.zeroToOne.reserve.Clone()
	}
	if p.oneToZero.reserve != nil {
		cloned.oneToZero.reserve = p.oneToZero.reserve.Clone()
	}
	return &cloned
}

func (p *PoolSimulator) CalculateLimit() map[string]*big.Int {
	limits := make(map[string]*big.Int, 2)
	if p.zeroToOne.reserve != nil {
		limits[p.Info.Tokens[1]] = p.zeroToOne.reserve.ToBig()
	}
	if p.oneToZero.reserve != nil {
		limits[p.Info.Tokens[0]] = p.oneToZero.reserve.ToBig()
	}
	if len(limits) == 0 {
		return nil
	}
	return limits
}

func (p *PoolSimulator) GetMetaInfo(tokenIn, _ string) any {
	dir, _, _, ok := p.directionFor(tokenIn)
	if !ok {
		dir = &p.zeroToOne
	}
	return PoolMeta{
		SourceRouter: dir.static.SourceRouter,
		TargetRouter: dir.static.TargetRouter,
	}
}
