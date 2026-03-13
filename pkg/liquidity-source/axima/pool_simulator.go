package axima

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
	extra         Extra
	decimalsDiff  int
}

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
	ErrUnavailableQuote      = errors.New("quote not available")
	ErrStalePoolData         = errors.New("stale pool data")
)

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:  entityPool.Address,
			Exchange: entityPool.Exchange,
			Type:     entityPool.Type,
			Tokens: lo.Map(entityPool.Tokens,
				func(item *entity.PoolToken, _ int) string { return item.Address }),
			Reserves: lo.Map(entityPool.Reserves,
				func(item string, _ int) *big.Int { return bignumber.NewBig(item) }),
		}},
		poolTimestamp: entityPool.Timestamp,
		extra:         extra,
		decimalsDiff:  int(entityPool.Tokens[0].Decimals) - int(entityPool.Tokens[1].Decimals),
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if !s.extra.QuoteAvailable {
		return nil, ErrUnavailableQuote
	}

	if s.poolTimestamp+s.extra.MaxAge < time.Now().Unix() {
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
		Gas: defaultGas,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn != -1 && indexOut != -1 {
		s.Info.Reserves[indexIn] = new(big.Int).Add(s.Info.Reserves[indexIn], params.TokenAmountIn.Amount)
		s.Info.Reserves[indexOut] = new(big.Int).Sub(s.Info.Reserves[indexOut], params.TokenAmountOut.Amount)

		zeroToOne := indexIn == 0
		if zeroToOne {
			s.extra.Bids = lo.Filter(s.extra.Bids, func(bin Bin, _ int) bool {
				return bin.CumulativeVolume.Cmp(params.TokenAmountOut.Amount) <= 0
			})

			s.extra.Bids = lo.Map(s.extra.Bids, func(bin Bin, _ int) Bin {
				bin.CumulativeVolume.Sub(bin.CumulativeVolume, params.TokenAmountOut.Amount)
				return bin
			})
		} else {
			s.extra.Asks = lo.Filter(s.extra.Asks, func(bin Bin, _ int) bool {
				return bin.CumulativeVolume.Cmp(params.TokenAmountOut.Amount) <= 0
			})

			s.extra.Asks = lo.Map(s.extra.Asks, func(bin Bin, _ int) Bin {
				bin.CumulativeVolume.Sub(bin.CumulativeVolume, params.TokenAmountOut.Amount)
				return bin
			})
		}
	}
}

func (s *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) any {
	return PoolMeta{
		SwapDirection: tokenIn == s.Info.Tokens[0],
	}
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
		}
	}

	cloned.extra.Bids = make([]Bin, len(s.extra.Bids))
	for i, bid := range s.extra.Bids {
		cloned.extra.Bids[i] = Bin{
			BinIdx:           bid.BinIdx,
			Price:            bid.Price,
			CumulativeVolume: new(big.Int).Set(bid.CumulativeVolume),
		}
	}
	return &cloned
}

func (s *PoolSimulator) getRate(zeroToOne bool, amountIn *big.Int) (*big.Int, error) {
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
