package kuruobv2

import (
	"math"
	"math/big"
	"slices"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	pool.Pool
	Extra
	StaticExtra
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}
	// The walk relies on these bounds: sizes and prices keep the Mul+Div steps inside 256 bits,
	// and zero divisors would silently give zero (uint256 Div by zero does not panic).
	if len(entityPool.Tokens) != 2 || staticExtra.SizePrecision == nil || staticExtra.BaseSizeMultiplier == nil ||
		staticExtra.PricePrecision == 0 || staticExtra.PricePrecision > math.MaxUint32 ||
		staticExtra.SizePrecision.IsZero() || staticExtra.SizePrecision.BitLen() > 128 ||
		staticExtra.BaseSizeMultiplier.IsZero() || staticExtra.BaseSizeMultiplier.BitLen() > 128 ||
		staticExtra.QuoteDecimals > maxQuoteDecimals || extra.TakerFeePps >= feeDenominator.Uint64() {
		return nil, ErrInvalidStaticExtra
	}
	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     entityPool.Address,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      lo.Map(entityPool.Tokens, func(t *entity.PoolToken, _ int) string { return t.Address }),
			Reserves:    lo.Map(entityPool.Reserves, func(r string, _ int) *big.Int { return bignumber.NewBig(r) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		Extra:       extra,
		StaticExtra: staticExtra,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	idxIn, idxOut := p.GetTokenIndex(params.TokenAmountIn.Token), p.GetTokenIndex(params.TokenOut)
	if idxIn < 0 || idxOut < 0 || idxIn == idxOut {
		return nil, ErrInvalidToken
	}
	var amountIn, used, out uint256.Int
	if amountIn.SetFromBig(params.TokenAmountIn.Amount) || amountIn.BitLen() > 128 { // swap takes uint128
		return nil, ErrOverflow
	} else if amountIn.IsZero() {
		return nil, ErrZeroAmountOut
	}
	var si SwapInfo
	var err error
	if idxIn == 0 {
		err = p.sell(&amountIn, &used, &out, &si)
	} else {
		err = p.buy(&amountIn, &used, &out, &si)
	}
	if err != nil {
		return nil, err
	}

	levels := int64(si.FullLevels)
	if !si.PartialFill.IsZero() {
		levels++
	}
	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: out.ToBig()},
		Fee:            &pool.TokenAmount{Token: params.TokenAmountIn.Token, Amount: bignumber.ZeroBI},
		RemainingTokenAmountIn: &pool.TokenAmount{Token: params.TokenAmountIn.Token,
			Amount: amountIn.Sub(&amountIn, &used).ToBig()},
		Gas:      gasBase + gasLevel*levels,
		SwapInfo: si,
	}, nil
}

// divUp sets z = ceil(x / d) for d != 0.
func divUp(z, x, d *uint256.Int) *uint256.Int {
	var rem uint256.Int
	if z.DivMod(x, d, &rem); !rem.IsZero() {
		z.AddUint64(z, 1)
	}
	return z
}

// sell (base in) walks bids. Each fill floors twice: price*size/sizePrecision, then
// *quoteScale/pricePrecision; the taker fee is ceiled once on the aggregate quote.
// ponytail: floors per aggregated L2 level, on-chain floors per maker order (<=1 wei/order lower).
func (p *PoolSimulator) sell(amountIn, used, out *uint256.Int, si *SwapInfo) error {
	if len(p.Bids) == 0 {
		return ErrInsufficientLiquidity
	}
	var remaining, fill, quote, tmp uint256.Int
	quoteScale := big256.TenPow(p.QuoteDecimals)
	pricePrecision := tmp.SetUint64(p.PricePrecision)
	remaining.Div(amountIn, p.BaseSizeMultiplier)
	used.Set(&remaining)
	for i := range p.Bids {
		level := &p.Bids[i]
		if remaining.IsZero() {
			break
		} else if remaining.Lt(&level.Size) {
			fill.Set(&remaining)
			si.PartialFill.Set(&remaining)
		} else {
			fill.Set(&level.Size)
			si.FullLevels++
		}
		// price < 2^32 and fill <= 2^128, so the product fits; quote*quoteScale may not.
		quote.Mul(quote.SetUint64(level.Price), &fill).Div(&quote, p.SizePrecision)
		out.Add(out, big256.MulDivDown(&quote, &quote, quoteScale, pricePrecision))
		remaining.Sub(&remaining, &fill)
	}
	// out < 2^192 and fee < 2^24, so the product fits.
	out.Sub(out, divUp(&tmp, tmp.Mul(out, quote.SetUint64(p.TakerFeePps)), feeDenominator))
	if out.IsZero() {
		return ErrZeroAmountOut
	}
	used.Sub(used, &remaining).Mul(used, p.BaseSizeMultiplier)
	return nil
}

// buy (quote in) treats amountIn as fee-inclusive: principal = floor(in*D/(D+fee)). Ask fills
// cost ceil(sum(price*size*quoteScale) / (sizePrecision*pricePrecision)) for the whole match,
// and the taker fee is ceiled once on that quote. Verified exact against testnet estimateSwap.
func (p *PoolSimulator) buy(amountIn, used, out *uint256.Int, si *SwapInfo) error {
	if len(p.Asks) == 0 {
		return ErrInsufficientLiquidity
	}
	var budget, numerator, perUnit, fill, priceSizePP, tmp uint256.Int
	quoteScale := big256.TenPow(p.QuoteDecimals)
	priceSizePP.Mul(p.SizePrecision, tmp.SetUint64(p.PricePrecision)) // both bounded: fits
	// amountIn < 2^128 and fee denominator < 2^24, so the product fits.
	budget.Mul(amountIn, feeDenominator).Div(&budget, tmp.AddUint64(feeDenominator, p.TakerFeePps))
	if _, overflow := budget.MulOverflow(&budget, &priceSizePP); overflow {
		return ErrOverflow
	}
	var filled uint256.Int
	for i := range p.Asks {
		level := &p.Asks[i]
		perUnit.Mul(tmp.SetUint64(level.Price), quoteScale)
		if fill.Sub(&budget, &numerator).Div(&fill, &perUnit).IsZero() {
			break
		} else if fill.Lt(&level.Size) {
			si.PartialFill.Set(&fill)
		} else {
			fill.Set(&level.Size)
			si.FullLevels++
		}
		filled.Add(&filled, &fill)
		numerator.Add(&numerator, perUnit.Mul(&perUnit, &fill)) // <= budget
		if !si.PartialFill.IsZero() {
			break
		}
	}
	if filled.IsZero() {
		return ErrZeroAmountOut
	}
	// used < amountIn < 2^128, fee < 2^24: the product fits.
	divUp(used, &numerator, &priceSizePP)
	used.Add(used, divUp(&tmp, tmp.Mul(used, perUnit.SetUint64(p.TakerFeePps)), feeDenominator))
	out.Mul(&filled, p.BaseSizeMultiplier)
	return nil
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.Bids, cloned.Asks = slices.Clone(p.Bids), slices.Clone(p.Asks)
	return &cloned
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	si, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		return
	}
	book := &p.Bids
	if p.GetTokenIndex(params.TokenAmountIn.Token) == 1 {
		book = &p.Asks
	}
	*book = (*book)[min(si.FullLevels, len(*book)):]
	if len(*book) > 0 && !si.PartialFill.IsZero() {
		(*book)[0].Size.Sub(&(*book)[0].Size, &si.PartialFill)
	}
}

func (p *PoolSimulator) GetMetaInfo(tokenIn, _ string) any {
	return MetaInfo{IdxIn: p.GetTokenIndex(tokenIn), HasNative: p.HasNative, BlockNumber: p.Info.BlockNumber}
}

func (p *PoolSimulator) SwapReceiveNativeIn(tokenIn, _ string, chainId valueobject.ChainID) bool {
	return p.HasNative && valueobject.IsWrappedNative(tokenIn, chainId)
}

func (p *PoolSimulator) SwapReturnNativeOut(_, tokenOut string, chainId valueobject.ChainID) bool {
	return p.HasNative && valueobject.IsWrappedNative(tokenOut, chainId)
}
