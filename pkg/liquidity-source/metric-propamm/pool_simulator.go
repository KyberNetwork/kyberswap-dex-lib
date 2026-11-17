package metricpropamm

import (
	"math/big"
	"slices"
	"time"

	"github.com/goccy/go-json"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	poolTimestamp int64
	noStaleCheck  bool // set by the factory when Opts.StaleCheck is false (indexing)
	extra         Extra
	staticExtra   StaticExtra
	decimalsDiff  int
}

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
	ErrUnavailableQuote      = errors.New("quote not available")
	ErrStalePoolData         = errors.New("stale pool data")
)

// NewPoolSimulatorFromParams is the registered factory. Indexing (Opts.StaleCheck=false) skips the age check.
func NewPoolSimulatorFromParams(params pool.FactoryParams) (*PoolSimulator, error) {
	s, err := NewPoolSimulator(params.EntityPool)
	if err == nil {
		s.noStaleCheck = !params.Opts.StaleCheck
	}
	return s, err
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Info: pool.PoolInfo{
			Address:  entityPool.Address,
			Exchange: entityPool.Exchange,
			Type:     entityPool.Type,
			Tokens: lo.Map(entityPool.Tokens,
				func(item *entity.PoolToken, _ int) string { return item.Address }),
			Reserves: lo.Map(entityPool.Reserves,
				func(item string, _ int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		},
		poolTimestamp: entityPool.Timestamp,
		extra:         extra,
		staticExtra:   staticExtra,
		decimalsDiff:  int(entityPool.Tokens[0].Decimals) - int(entityPool.Tokens[1].Decimals),
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if !s.extra.QuoteAvailable {
		return nil, ErrUnavailableQuote
	}

	if !s.noStaleCheck && s.poolTimestamp+s.extra.MaxAge < time.Now().Unix() {
		return nil, ErrStalePoolData
	}

	zeroToOne := params.TokenAmountIn.Token == s.Info.Tokens[0]

	amountOut, err := s.getRate(zeroToOne, params.TokenAmountIn.Amount)
	if err != nil {
		return nil, err
	}

	indexOut := s.GetTokenIndex(params.TokenOut)
	if indexOut == -1 {
		return nil, ErrInvalidToken
	}

	if amountOut.Cmp(s.Info.Reserves[indexOut]) > 0 {
		return nil, ErrInsufficientLiquidity
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  params.TokenOut,
			Amount: amountOut,
		},
		Fee: &pool.TokenAmount{
			Token:  params.TokenOut,
			Amount: bignumber.ZeroBI,
		},
		Gas: baseGas + int64(s.extra.Extensions)*gasPerExtension,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn == -1 || indexOut == -1 {
		return
	}
	s.Info.Reserves[indexIn] = new(big.Int).Add(s.Info.Reserves[indexIn], params.TokenAmountIn.Amount)
	s.Info.Reserves[indexOut] = new(big.Int).Sub(s.Info.Reserves[indexOut], params.TokenAmountOut.Amount)

	if indexIn == 0 {
		s.extra.Bids, s.extra.InitBid = consumeBins(s.extra.Bids, s.extra.InitBid, params.TokenAmountIn.Amount, params.TokenAmountOut.Amount)
	} else {
		s.extra.Asks, s.extra.InitAsk = consumeBins(s.extra.Asks, s.extra.InitAsk, params.TokenAmountIn.Amount, params.TokenAmountOut.Amount)
	}
}

// consumeBins takes amountOut of maker volume off the front of bins, which start at price start. The bins the
// swap fills are dropped; the one it ends in keeps its unfilled volume, and the start price moves to where the
// fill ended, interpolated within that bin as getRateLegacy interpolates. Every bin left has its cumulative volume
// lowered by amountOut and its cumulative input, when the pool tracks it, by amountIn, which keeps walkDepth's
// per-bin input and output. It returns new bins and a new start price, and does not modify its arguments (a clone
// shares the prices with its original).
//
// It moves the swapped side only. The contract keeps one position for both sides, so a swap also moves the other
// side's start price and adds the crossed bins to it as liquidity. Left as it was, the other side quotes slightly
// less than the chain does after the swap (0.06-2 bps on an Ethereum fork), never more. Consecutive swaps in the
// same direction match the contract exactly.
func consumeBins(bins []Bin, start, amountIn, amountOut *big.Int) ([]Bin, *big.Int) {
	prevVolume, prevPrice := bignumber.ZeroBI, start
	for i, bin := range bins {
		if bin.CumulativeVolume.Cmp(amountOut) <= 0 {
			prevVolume, prevPrice = bin.CumulativeVolume, bin.Price
			continue
		}
		inBin := new(big.Int).Sub(bin.CumulativeVolume, prevVolume)
		filled := new(big.Int).Sub(amountOut, prevVolume)
		price := new(big.Int).Sub(bin.Price, prevPrice)
		price.Mul(price, filled)
		price.Quo(price, inBin)
		price.Add(price, prevPrice)

		left := make([]Bin, len(bins)-i)
		for j, b := range bins[i:] {
			left[j] = Bin{BinIdx: b.BinIdx, Price: b.Price, CumulativeVolume: new(big.Int).Sub(b.CumulativeVolume, amountOut)}
			if b.CumulativeInput != nil {
				left[j].CumulativeInput = new(big.Int).Sub(b.CumulativeInput, amountIn)
			}
		}
		return left, price
	}
	return []Bin{}, prevPrice
}

func (s *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) any {
	return PoolMeta{
		SwapDirection:        tokenIn == s.Info.Tokens[0],
		PriceProviderAddress: s.staticExtra.PriceProviderAddress,
		BlockNumber:          s.Info.BlockNumber,
	}
}

func cloneBig(x *big.Int) *big.Int {
	if x == nil {
		return nil
	}
	return new(big.Int).Set(x)
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.Info.Reserves = slices.Clone(s.Info.Reserves)
	cloned.extra.Asks = make([]Bin, len(s.extra.Asks))
	for i, ask := range s.extra.Asks {
		cloned.extra.Asks[i] = Bin{
			BinIdx:           ask.BinIdx,
			Price:            ask.Price,
			CumulativeVolume: new(big.Int).Set(ask.CumulativeVolume),
			CumulativeInput:  cloneBig(ask.CumulativeInput),
		}
	}

	cloned.extra.Bids = make([]Bin, len(s.extra.Bids))
	for i, bid := range s.extra.Bids {
		cloned.extra.Bids[i] = Bin{
			BinIdx:           bid.BinIdx,
			Price:            bid.Price,
			CumulativeVolume: new(big.Int).Set(bid.CumulativeVolume),
			CumulativeInput:  cloneBig(bid.CumulativeInput),
		}
	}
	return &cloned
}

// getRate walks the depth per Metric's "Estimate a fill from depth". Pools stored without
// cumulative input (before it was tracked) use the legacy price-averaging estimate.
func (s *PoolSimulator) getRate(zeroToOne bool, amountIn *big.Int) (*big.Int, error) {
	if bins := lo.Ternary(zeroToOne, s.extra.Bids, s.extra.Asks); len(bins) > 0 && bins[0].CumulativeInput != nil {
		return s.walkDepth(!zeroToOne, bins, amountIn)
	}
	return s.getRateLegacy(zeroToOne, amountIn)
}

func (s *PoolSimulator) getRateLegacy(zeroToOne bool, amountIn *big.Int) (*big.Int, error) {
	var currentPrice big.Int
	currentPrice.Set(lo.Ternary(zeroToOne, s.extra.InitBid, s.extra.InitAsk))

	var remainingVolume big.Int
	remainingVolume.Set(amountIn)

	var finalFillPriceNumerator, finalFillPriceDenominator, tmp big.Int

	bins := lo.Ternary(zeroToOne, s.extra.Bids, s.extra.Asks)

	var enoughLiquidity bool

	for i, bin := range bins {
		var volumeInThisBin big.Int

		if i == 0 {
			volumeInThisBin.Set(bin.CumulativeVolume)
		} else {
			prevBin := bins[i-1]
			volumeInThisBin.Sub(bin.CumulativeVolume, prevBin.CumulativeVolume)
		}

		var convertedVolume big.Int
		// We convert back maker token amount to taker token amount
		// using the price of the current bin, to compare with remainingVolume.
		// So pass !zeroToOne to calculateAmountFromPrice.
		convertedVolume.Set(s.calculateAmountFromPrice(!zeroToOne, &volumeInThisBin, bin.Price))

		if remainingVolume.Cmp(&convertedVolume) <= 0 {
			var exitPrice, fillPrice big.Int
			exitPrice.Sub(bin.Price, &currentPrice)
			exitPrice.Mul(&exitPrice, &remainingVolume)
			exitPrice.Div(&exitPrice, &convertedVolume)
			exitPrice.Add(&exitPrice, &currentPrice)

			fillPrice.Add(&currentPrice, &exitPrice)
			fillPrice.Div(&fillPrice, bignumber.Two)

			finalFillPriceNumerator.Add(
				&finalFillPriceNumerator,
				tmp.Mul(&fillPrice, &remainingVolume),
			)
			finalFillPriceDenominator.Add(
				&finalFillPriceDenominator,
				&remainingVolume,
			)
			enoughLiquidity = true
			break
		} else {
			var fillPrice big.Int
			fillPrice.Add(&currentPrice, bin.Price)
			fillPrice.Div(&fillPrice, bignumber.Two)

			finalFillPriceNumerator.Add(
				&finalFillPriceNumerator,
				tmp.Mul(&fillPrice, &convertedVolume),
			)
			finalFillPriceDenominator.Add(
				&finalFillPriceDenominator,
				&convertedVolume,
			)

			remainingVolume.Sub(&remainingVolume, &convertedVolume)
			currentPrice.Set(bin.Price)
		}
	}

	if !enoughLiquidity && remainingVolume.Sign() > 0 {
		return nil, ErrInsufficientLiquidity
	}

	var fillPrice big.Int
	if finalFillPriceDenominator.Sign() == 0 {
		// Should not possible, just safety check to avoid division by zero.
		return nil, ErrUnavailableQuote
	}

	fillPrice.Div(&finalFillPriceNumerator, &finalFillPriceDenominator)

	amountOut := s.calculateAmountFromPrice(zeroToOne, amountIn, &fillPrice)

	return amountOut, nil
}

func (s *PoolSimulator) calculateAmountFromPrice(zeroToOne bool, amountIn *big.Int, price *big.Int) *big.Int {
	var amountOut big.Int
	if zeroToOne {
		amountOut.Mul(amountIn, price)
		amountOut.Div(&amountOut, Q64BI)
		if s.decimalsDiff > 0 {
			amountOut.Div(&amountOut, bignumber.TenPowInt(s.decimalsDiff))
		} else {
			amountOut.Mul(&amountOut, bignumber.TenPowInt(-s.decimalsDiff))
		}
	} else {
		amountOut.Mul(amountIn, Q64BI)
		if s.decimalsDiff > 0 {
			amountOut.Mul(&amountOut, bignumber.TenPowInt(s.decimalsDiff))
		} else {
			amountOut.Div(&amountOut, bignumber.TenPowInt(-s.decimalsDiff))
		}
		amountOut.Div(&amountOut, price)
	}

	return &amountOut
}
