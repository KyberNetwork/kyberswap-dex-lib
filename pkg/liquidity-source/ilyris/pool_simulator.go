package ilyris

import (
	"math/big"
	"slices"
	"sort"
	"strings"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolSimulator struct {
	pool.Pool
	binStepBps     uint32
	decimalFactor  uint256.Int
	activeID       int32
	bins           []Bin // ascending by ID; replaced wholesale by UpdateBalance (copy-on-write)
	fee            FeeParams
	feeRate        uint64 // getTotalFeeRate() derived from fee at blockTimestamp, 1e9 precision
	blockTimestamp uint64
	swapsPaused    bool
	freezeEnd      uint64
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	var se StaticExtra
	if err := json.Unmarshal([]byte(ep.StaticExtra), &se); err != nil {
		return nil, err
	}
	var ex Extra
	if err := json.Unmarshal([]byte(ep.Extra), &ex); err != nil {
		return nil, err
	}
	if len(ep.Tokens) != 2 || ex.Fee == nil || se.BinStepBps == 0 || se.BinStepBps > 1000 ||
		se.DecimalsX > 18 || se.DecimalsY > 18 || ex.ActiveID < minBinID || ex.ActiveID > maxBinID {
		return nil, ErrMalformedExtra
	}

	bins := slices.DeleteFunc(ex.Bins, func(b Bin) bool { return b.ReserveX.IsZero() && b.ReserveY.IsZero() })
	if len(bins) == 0 {
		return nil, ErrEmptyBook
	}
	sort.Slice(bins, func(i, j int) bool { return bins[i].ID < bins[j].ID })
	var sumX, sumY uint256.Int
	for i := range bins {
		sumX.Add(&sumX, &bins[i].ReserveX)
		sumY.Add(&sumY, &bins[i].ReserveY)
	}

	p := &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     strings.ToLower(ep.Address),
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      []string{strings.ToLower(ep.Tokens[0].Address), strings.ToLower(ep.Tokens[1].Address)},
			Reserves:    []*big.Int{sumX.ToBig(), sumY.ToBig()},
			BlockNumber: ep.BlockNumber,
		}},
		binStepBps:     se.BinStepBps,
		activeID:       ex.ActiveID,
		bins:           bins,
		fee:            *ex.Fee,
		blockTimestamp: ex.BlockTimestamp,
		swapsPaused:    ex.GuardSwapsPaused,
		freezeEnd:      ex.GuardFreezeEnd,
	}
	decimalFactor(&p.decimalFactor, se.DecimalsX, se.DecimalsY)
	p.feeRate = p.fee.totalFeeRate(p.binStepBps, p.activeID, p.blockTimestamp)
	return p, nil
}

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	// The guard is skipped by quoteExactIn but enforced by swapExactIn. Any freeze ending after
	// the snapshot blocks, since the snapshot cannot see when a scheduled window starts.
	if p.swapsPaused {
		return nil, ErrSwapsPaused
	} else if p.freezeEnd > p.blockTimestamp {
		return nil, ErrCorporateActionFreeze
	}
	inIdx, outIdx := p.GetTokenIndex(params.TokenAmountIn.Token), p.GetTokenIndex(params.TokenOut)
	if inIdx < 0 || outIdx < 0 || inIdx == outIdx {
		return nil, ErrInvalidToken
	}
	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow || amountIn.IsZero() {
		return nil, ErrInsufficientLiquidity
	}

	xForY := inIdx == 0
	var amountOut, fee uint256.Int
	finalID, fills, err := p.quoteExactIn(xForY, amountIn, &amountOut, &fee)
	if err != nil {
		return nil, err
	} else if amountOut.IsZero() {
		return nil, ErrInsufficientLiquidity
	}

	crossed := int64(finalID - p.activeID)
	if crossed < 0 {
		crossed = -crossed
	}
	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: params.TokenAmountIn.Token, Amount: fee.ToBig()},
		Gas:            baseSwapGas + crossed*perExtraBinGas,
		SwapInfo:       SwapInfo{xForY: xForY, newActiveID: finalID, fills: fills},
	}, nil
}

// quoteExactIn mirrors BinPool._planSwap: the fee is taken once on the input at the rate fixed
// at swap start, then the net input walks bins from activeId towards the output side.
func (p *PoolSimulator) quoteExactIn(xForY bool, amountIn, amountOut, fee *uint256.Int) (int32, []binFill, error) {
	var remaining uint256.Int
	if _, overflow := remaining.MulOverflow(amountIn, fee.SetUint64(feePrecision-p.feeRate)); overflow {
		return 0, nil, ErrOverflow
	}
	remaining.Div(&remaining, feePrecisionU)
	if fee.Sub(amountIn, &remaining).Gt(maxUint128) {
		return 0, nil, ErrOverflow // _accrueFee FeeOverflow
	}

	var price, maxIn, reserveIn uint256.Int
	fills := make([]binFill, 0, 2)
	cursor, finalID := p.activeID, p.activeID
	for !remaining.IsZero() {
		i, ok := p.nextWithOutput(cursor, xForY)
		if !ok {
			return 0, nil, ErrInsufficientLiquidity
		}
		b := &p.bins[i]
		next := b.ID + 1
		availIn, availOut := &b.ReserveY, &b.ReserveX
		if xForY {
			next = b.ID - 1
			availIn, availOut = &b.ReserveX, &b.ReserveY
		}
		if err := priceFromID(&price, p.binStepBps, b.ID); err != nil {
			return 0, nil, err
		}
		var err error
		if xForY {
			err = mulDivUp(&maxIn, availOut, &p.decimalFactor, &price) // xFromQuoteUp
		} else {
			err = mulDivUp(&maxIn, availOut, &price, &p.decimalFactor) // quoteFromXUp
		}
		if err != nil {
			return 0, nil, err
		}
		if maxIn.IsZero() {
			cursor = next
			continue
		}

		fill := binFill{idx: i}
		if remaining.Lt(&maxIn) {
			fill.in.Set(&remaining)
			if xForY {
				err = mulDiv(&fill.out, &remaining, &price, &p.decimalFactor) // quoteFromX
			} else {
				err = mulDiv(&fill.out, &remaining, &p.decimalFactor, &price) // xFromQuote
			}
			if err != nil {
				return 0, nil, err
			}
			cursor = b.ID
		} else {
			fill.in.Set(&maxIn)
			fill.out.Set(availOut)
			cursor = next
		}
		if reserveIn.Add(availIn, &fill.in).Gt(maxUint128) {
			return 0, nil, ErrOverflow // _toUint128 ReserveOverflow
		}
		amountOut.Add(amountOut, &fill.out)
		remaining.Sub(&remaining, &fill.in)
		fills = append(fills, fill)
		finalID = b.ID
	}
	return finalID, fills, nil
}

// nextWithOutput mirrors BinPool._nextWithOutput: the nearest bin from fromID (inclusive)
// holding output-side reserves, walking down for xForY (Y sits at/below active) and up otherwise.
func (p *PoolSimulator) nextWithOutput(fromID int32, xForY bool) (int, bool) {
	if fromID < minBinID || fromID > maxBinID {
		return 0, false
	}
	i := sort.Search(len(p.bins), func(i int) bool { return p.bins[i].ID >= fromID })
	if xForY {
		if i == len(p.bins) || p.bins[i].ID != fromID {
			i--
		}
		for ; i >= 0; i-- {
			if !p.bins[i].ReserveY.IsZero() {
				return i, true
			}
		}
		return 0, false
	}
	for ; i < len(p.bins); i++ {
		if !p.bins[i].ReserveX.IsZero() {
			return i, true
		}
	}
	return 0, false
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	si, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		return
	}
	inIdx, outIdx := 1, 0
	if si.xForY {
		inIdx, outIdx = 0, 1
	}
	// Net of fee: BinPool credits the fee to per-share accumulators, not to bin reserves.
	reserveIn := new(big.Int).Add(p.Info.Reserves[inIdx], params.TokenAmountIn.Amount)
	p.Info.Reserves[inIdx] = reserveIn.Sub(reserveIn, params.Fee.Amount)
	p.Info.Reserves[outIdx] = new(big.Int).Sub(p.Info.Reserves[outIdx], params.TokenAmountOut.Amount)

	p.bins = slices.Clone(p.bins)
	for i := range si.fills {
		f, b := &si.fills[i], &p.bins[si.fills[i].idx]
		if si.xForY {
			b.ReserveX.Add(&b.ReserveX, &f.in)
			b.ReserveY.Sub(&b.ReserveY, &f.out)
		} else {
			b.ReserveY.Add(&b.ReserveY, &f.in)
			b.ReserveX.Sub(&b.ReserveX, &f.out)
		}
	}

	p.fee.commit(p.activeID, si.newActiveID, p.blockTimestamp)
	p.activeID = si.newActiveID
	p.feeRate = p.fee.totalFeeRate(p.binStepBps, p.activeID, p.blockTimestamp)
}

// CloneState copies Info.Reserves, which UpdateBalance writes by index. Bins are replaced
// wholesale and the fee state is a value, so both are safe to share.
func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.Info.Reserves = slices.Clone(p.Info.Reserves)
	return &cloned
}

func (p *PoolSimulator) GetMetaInfo(_, _ string) any {
	return pool.MetaInfo{BlockNumber: p.Info.BlockNumber}
}
