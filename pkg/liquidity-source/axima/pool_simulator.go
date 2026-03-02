package axima

import (
	"fmt"
	"math"
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

	amountInF, _ := params.TokenAmountIn.Amount.Float64()

	zeroToOne := params.TokenAmountIn.Token == s.Info.Tokens[0]
	rate := lo.Ternary(zeroToOne, s.extra.ZeroToOneRate, s.extra.OneToZeroRate)
	decimalsDiff := lo.Ternary(zeroToOne, s.decimalsDiff, -s.decimalsDiff)

	oldAmountOutF := amountInF * rate / math.Pow10(int(decimalsDiff))
	oldAmountOut, _ := big.NewFloat(oldAmountOutF).Int(nil)

	bins := lo.Ternary(zeroToOne, s.extra.Bids, s.extra.Asks)
	amountOut, err := GetRate(params.TokenAmountIn.Amount, bins, decimalsDiff)
	if err != nil {
		return nil, err
	}

	fmt.Println("Old amount out:", oldAmountOut.String())
	fmt.Println("New amount out:", amountOut.String())
	fmt.Println("Diff:", new(big.Int).Sub(amountOut, oldAmountOut).String())

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
				return bin.CumulativeVolume.Cmp(params.TokenAmountIn.Amount) > 0
			})

			s.extra.Bids = lo.Map(s.extra.Bids, func(bin Bin, _ int) Bin {
				bin.CumulativeVolume.Sub(bin.CumulativeVolume, params.TokenAmountIn.Amount)
				return bin
			})
		} else {
			s.extra.Asks = lo.Filter(s.extra.Asks, func(bin Bin, _ int) bool {
				return bin.CumulativeVolume.Cmp(params.TokenAmountIn.Amount) > 0
			})

			s.extra.Asks = lo.Map(s.extra.Asks, func(bin Bin, _ int) Bin {
				bin.CumulativeVolume.Sub(bin.CumulativeVolume, params.TokenAmountIn.Amount)
				return bin
			})
		}
	}
}

func (s *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) any {
	// Return swapDirection
	return tokenIn == s.Info.Tokens[0]
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.Info.Reserves = slices.Clone(s.Info.Reserves)
	cloned.extra.Asks = make([]Bin, len(s.extra.Asks))
	for i, ask := range s.extra.Asks {
		cloned.extra.Asks[i] = Bin{
			BinIdx:           ask.BinIdx,
			Rate:             ask.Rate,
			CumulativeVolume: new(big.Int).Set(ask.CumulativeVolume),
			PriceImpactE6:    ask.PriceImpactE6,
		}
	}

	cloned.extra.Bids = make([]Bin, len(s.extra.Bids))
	for i, bid := range s.extra.Bids {
		cloned.extra.Bids[i] = Bin{
			BinIdx:           bid.BinIdx,
			Rate:             bid.Rate,
			CumulativeVolume: new(big.Int).Set(bid.CumulativeVolume),
			PriceImpactE6:    bid.PriceImpactE6,
		}
	}
	return &cloned
}

func GetRate(amountIn *big.Int, bins []Bin, decimalsDiff int) (*big.Int, error) {
	// Find the last bin with amountIn <= bin.CummlativeVolume
	binIdx := -1
	for i, bin := range bins {
		if amountIn.Cmp(bin.CumulativeVolume) <= 0 {
			binIdx = i
		}
	}

	if binIdx == -1 {
		return nil, ErrInsufficientLiquidity
	}

	if binIdx == len(bins)-1 {
		// Last bin, can't interpolate
		amountInF, _ := amountIn.Float64()
		amountOutF := amountInF * bins[binIdx].Rate / math.Pow10(int(decimalsDiff))
		amountOut, _ := big.NewFloat(amountOutF).Int(nil)
		return amountOut, nil
	}

	curBinAmountInF, _ := bins[binIdx].CumulativeVolume.Float64()
	curBinAmountOutF := curBinAmountInF * bins[binIdx].Rate / math.Pow10(int(decimalsDiff))

	nextBinAmountInF, _ := bins[binIdx+1].CumulativeVolume.Float64()
	nextBinAmountOutF := nextBinAmountInF * bins[binIdx+1].Rate / math.Pow10(int(decimalsDiff))

	// Linear interpolation
	amountInF, _ := amountIn.Float64()
	amountOutF := curBinAmountOutF + (amountInF-curBinAmountInF)*(nextBinAmountOutF-curBinAmountOutF)/(nextBinAmountInF-curBinAmountInF)
	amountOut, _ := big.NewFloat(amountOutF).Int(nil)

	return amountOut, nil
}
