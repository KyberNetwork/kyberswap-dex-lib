package kuruobv2

import (
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
	if staticExtra.SizePrecision == nil || staticExtra.BaseSizeMultiplier == nil || staticExtra.PricePrecision == 0 {
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
	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow || amountIn.BitLen() > 128 { // OrderBook.swap takes uint128 amountIn
		return nil, ErrOverflow
	} else if amountIn.IsZero() {
		return nil, ErrZeroAmountOut
	}
	var used, out *uint256.Int
	var swapInfo SwapInfo
	var err error
	if idxIn == 0 {
		used, out, swapInfo, err = p.sell(amountIn)
	} else {
		used, out, swapInfo, err = p.buy(amountIn)
	}
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: out.ToBig()},
		Fee:            &pool.TokenAmount{Token: params.TokenAmountIn.Token, Amount: big.NewInt(0)},
		RemainingTokenAmountIn: &pool.TokenAmount{Token: params.TokenAmountIn.Token,
			Amount: amountIn.Sub(amountIn, used).ToBig()},
		Gas:      gasBase + gasLevel*int64(swapInfo.FullLevels+lo.Ternary(swapInfo.PartialFill.IsZero(), 0, 1)),
		SwapInfo: swapInfo,
	}, nil
}

// sell (base in) walks bids. Each fill floors twice: price*size/sizePrecision, then
// *quoteScale/pricePrecision; the taker fee is ceiled once on the aggregate quote.
// ponytail: floors per aggregated L2 level, on-chain floors per maker order (<=1 wei/order lower).
func (p *PoolSimulator) sell(amountIn *uint256.Int) (used, out *uint256.Int, si SwapInfo, err error) {
	if len(p.Bids) == 0 {
		return nil, nil, si, ErrInsufficientLiquidity
	}
	var remaining, fill, quote, tmp uint256.Int
	quoteScale := big256.TenPow(p.QuoteDecimals)
	remaining.Div(amountIn, p.BaseSizeMultiplier)
	used, out, si.PartialFill = remaining.Clone(), new(uint256.Int), new(uint256.Int)
	for _, level := range p.Bids {
		if remaining.IsZero() {
			break
		} else if remaining.Lt(level.Size) {
			fill.Set(&remaining)
			si.PartialFill.Set(&remaining)
		} else {
			fill.Set(level.Size)
			si.FullLevels++
		}
		big256.MulDivDown(&quote, tmp.SetUint64(level.Price), &fill, p.SizePrecision)
		out.Add(out, big256.MulDivDown(&quote, &quote, quoteScale, tmp.SetUint64(p.PricePrecision)))
		remaining.Sub(&remaining, &fill)
	}
	out.Sub(out, big256.MulDivUp(&tmp, out, tmp.SetUint64(p.TakerFeePps), feeDenominator))
	if out.IsZero() {
		return nil, nil, si, ErrZeroAmountOut
	}
	return used.Sub(used, &remaining).Mul(used, p.BaseSizeMultiplier), out, si, nil
}

// buy (quote in) treats amountIn as fee-inclusive: principal = floor(in*D/(D+fee)). Ask fills
// cost ceil(sum(price*size*quoteScale) / (sizePrecision*pricePrecision)) for the whole match,
// and the taker fee is ceiled once on that quote. Verified exact against testnet estimateSwap.
func (p *PoolSimulator) buy(amountIn *uint256.Int) (used, out *uint256.Int, si SwapInfo, err error) {
	if len(p.Asks) == 0 {
		return nil, nil, si, ErrInsufficientLiquidity
	}
	var budget, numerator, perUnit, fill, priceSizePP, tmp uint256.Int
	quoteScale := big256.TenPow(p.QuoteDecimals)
	priceSizePP.Mul(p.SizePrecision, tmp.SetUint64(p.PricePrecision))
	big256.MulDivDown(&budget, amountIn, feeDenominator, tmp.AddUint64(feeDenominator, p.TakerFeePps))
	if _, overflow := budget.MulOverflow(&budget, &priceSizePP); overflow {
		return nil, nil, si, ErrOverflow
	}
	out, si.PartialFill = new(uint256.Int), new(uint256.Int)
	for _, level := range p.Asks {
		perUnit.Mul(tmp.SetUint64(level.Price), quoteScale)
		if fill.Sub(&budget, &numerator).Div(&fill, &perUnit).IsZero() {
			break
		} else if fill.Lt(level.Size) {
			si.PartialFill.Set(&fill)
		} else {
			fill.Set(level.Size)
			si.FullLevels++
		}
		out.Add(out, &fill)
		numerator.Add(&numerator, perUnit.Mul(&perUnit, &fill))
		if !si.PartialFill.IsZero() {
			break
		}
	}
	if out.IsZero() {
		return nil, nil, si, ErrZeroAmountOut
	}
	used = big256.MulDivUp(new(uint256.Int), &numerator, big256.U1, &priceSizePP)
	return used.Add(used, big256.MulDivUp(&tmp, used, tmp.SetUint64(p.TakerFeePps), feeDenominator)),
		out.Mul(out, p.BaseSizeMultiplier), si, nil
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
	if len(*book) > 0 && !si.PartialFill.IsZero() { // copy-on-write: levels are shared with clones
		(*book)[0].Size = new(uint256.Int).Sub((*book)[0].Size, si.PartialFill)
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
