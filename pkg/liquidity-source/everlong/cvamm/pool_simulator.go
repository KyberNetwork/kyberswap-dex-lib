package everlongcvamm

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

// PoolSimulator prices a CvammALM venue: a single-LP AMM that holds both tokens itself
// and evaluates a closed-form reservation curve — no pool contract, no ticks, no rungs.
// token0 is the 18-decimal stable, token1 the volatile leg (pinned by the contract, not
// sorted by address). Exact-input only; the fee is a directional OUTPUT haircut sampled
// pre-trade; partial fills are normal (the band's finite support truncates rather than
// reverts) and the unspent input is returned as RemainingTokenAmountIn.
type PoolSimulator struct {
	pool.Pool
	StaticExtra StaticExtra
	Extra       Extra

	// accounted tradeable reserves (idle excluded) — the on-chain solvency clamp caps
	// the gross payout at these, and so does the quote.
	reserveStable   *uint256.Int
	reserveVolatile *uint256.Int
	gasStableIn     int64
	gasVolatileIn   int64
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(p entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return nil, err
	}
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	info := pool.PoolInfo{
		Address:     p.Address,
		Exchange:    p.Exchange,
		Type:        p.Type,
		Tokens:      lo.Map(p.Tokens, func(e *entity.PoolToken, _ int) string { return e.Address }),
		Reserves:    lo.Map(p.Reserves, func(e string, _ int) *big.Int { return bignumber.NewBig(e) }),
		BlockNumber: p.BlockNumber,
	}
	if len(info.Reserves) != 2 {
		return nil, ErrInvalidToken
	}
	reserveStable, overflow := uint256.FromBig(info.Reserves[0])
	if overflow {
		return nil, ErrOverflow
	}
	reserveVolatile, overflow := uint256.FromBig(info.Reserves[1])
	if overflow {
		return nil, ErrOverflow
	}

	gasStableIn, gasVolatileIn := staticExtra.GasStableIn, staticExtra.GasVolatileIn
	if gasStableIn == 0 {
		gasStableIn = defaultGasStableIn
	}
	if gasVolatileIn == 0 {
		gasVolatileIn = defaultGasVolatileIn
	}

	return &PoolSimulator{
		Pool:            pool.Pool{Info: info},
		StaticExtra:     staticExtra,
		Extra:           extra,
		reserveStable:   reserveStable,
		reserveVolatile: reserveVolatile,
		gasStableIn:     gasStableIn,
		gasVolatileIn:   gasVolatileIn,
	}, nil
}

// CalcAmountOut mirrors CvammSwapLib.execute step for step (no price bound): coordinate
// fill -> solvency clamp -> pre-trade output-side fee. Pure — no state is written.
func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	indexIn, indexOut := p.GetTokenIndex(params.TokenAmountIn.Token), p.GetTokenIndex(params.TokenOut)
	if indexIn < 0 || indexOut < 0 || indexIn == indexOut {
		return nil, ErrInvalidToken
	}
	stableIn := indexIn == 0

	e := &p.Extra
	if e.Paused {
		return nil, ErrPaused
	}
	if e.XWad == nil || e.AnchorSqrtX96 == nil || e.Kappa == nil ||
		e.Support.AWad == nil || e.Support.XLo == nil || e.Support.XHi == nil || e.Support.YHi == nil {
		return nil, ErrRetractedBook
	}

	var amountIn uint256.Int
	if overflow := amountIn.SetFromBig(params.TokenAmountIn.Amount); overflow || amountIn.IsZero() {
		return nil, ErrOverflow
	}

	var gross, xAfter, unspent uint256.Int
	if err := swapExactInX96(&gross, &xAfter, &unspent, &e.Support, e.AnchorSqrtX96, e.Kappa,
		e.XWad, stableIn, &amountIn); err != nil {
		return nil, err
	}
	var used uint256.Int
	used.Sub(&amountIn, &unspent)
	if used.IsZero() || gross.IsZero() {
		return nil, ErrSwapExhausted
	}

	// The curve PRICES; the accounted reserves are authoritative for SOLVENCY. A fill
	// walking to the band edge can quote a hair above the cached balance — clamp DOWN,
	// as the venue does, so the quote never exceeds what the book holds.
	available := p.reserveStable
	if stableIn {
		available = p.reserveVolatile
	}
	if gross.Gt(available) {
		gross.Set(available)
	}
	if gross.IsZero() {
		return nil, ErrSwapExhausted
	}

	// Fee on the output leg, sampled pre-trade, floored — so the net rounds up by <= 1
	// base unit in the taker's favour, matching the chain.
	feeWad := e.FeeVolatileInWad
	if stableIn {
		feeWad = e.FeeStableInWad
	}
	var fee, netOut uint256.Int
	if feeWad != nil {
		big256.MulDivDown(&fee, &gross, feeWad, uWad)
	}
	netOut.Sub(&gross, &fee)
	if netOut.IsZero() {
		return nil, ErrZeroAmountOut
	}

	remainingTokenAmountIn := &pool.TokenAmount{Token: params.TokenAmountIn.Token, Amount: bignumber.ZeroBI}
	if !unspent.IsZero() {
		remainingTokenAmountIn.Amount = unspent.ToBig()
	}

	gas := p.gasVolatileIn
	if stableIn {
		gas = p.gasStableIn
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut:         &pool.TokenAmount{Token: params.TokenOut, Amount: netOut.ToBig()},
		Fee:                    &pool.TokenAmount{Token: params.TokenOut, Amount: fee.ToBig()},
		RemainingTokenAmountIn: remainingTokenAmountIn,
		Gas:                    gas,
		SwapInfo: SwapInfo{
			XAfter:       &xAfter,
			AmountInUsed: &used,
			GrossOut:     &gross,
			StableIn:     stableIn,
		},
	}, nil
}

// CalcAmountIn is intentionally rejected: the venue is exact-input only (the fee is an
// output-side haircut and the curve solves the forward direction only).
func (p *PoolSimulator) CalcAmountIn(pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	return nil, ErrExactOutNotSupported
}

// UpdateBalance replays the fill from SwapInfo: the input leg grows by the amount
// actually used, the output leg shrinks by the GROSS output (net + fee — the fee leaves
// the priced book into idle), and the coordinate moves to xAfter. kappa, the anchor and
// the band never move on a swap. All pointers are reassigned wholesale (copy-on-write).
func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	si, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		return
	}
	p.Extra.XWad = new(uint256.Int).Set(si.XAfter)
	if si.StableIn {
		p.reserveStable = new(uint256.Int).Add(p.reserveStable, si.AmountInUsed)
		p.reserveVolatile = new(uint256.Int).Sub(p.reserveVolatile, si.GrossOut)
	} else {
		p.reserveVolatile = new(uint256.Int).Add(p.reserveVolatile, si.AmountInUsed)
		p.reserveStable = new(uint256.Int).Sub(p.reserveStable, si.GrossOut)
	}
	p.Info.Reserves = []*big.Int{p.reserveStable.ToBig(), p.reserveVolatile.ToBig()}
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	// UpdateBalance reassigns every mutable pointer wholesale, so the struct value copy
	// is a sufficient snapshot; Info.Reserves is re-sliced for index-safety.
	cloned.Info.Reserves = lo.Map(p.Info.Reserves, func(r *big.Int, _ int) *big.Int {
		return new(big.Int).Set(r)
	})
	return &cloned
}

func (p *PoolSimulator) GetMetaInfo(_, _ string) any {
	return PoolMeta{
		ALM:     p.Info.Address,
		Adapter: p.StaticExtra.Adapter,
	}
}

// GetApprovalAddress: the ALM pulls the input token from the caller with transferFrom.
func (p *PoolSimulator) GetApprovalAddress(_, _ string) string {
	return p.Info.Address
}
