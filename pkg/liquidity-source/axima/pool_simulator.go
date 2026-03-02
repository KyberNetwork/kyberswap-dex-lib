package axima

import (
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

	zeroToOne := params.TokenAmountIn.Token == s.Info.Tokens[0]
	decimalsDiff := lo.Ternary(zeroToOne, s.decimalsDiff, -s.decimalsDiff)

	bins := lo.Ternary(zeroToOne, s.extra.Bids, s.extra.Asks)
	amountOut, err := GetRate(params.TokenAmountIn.Amount, bins, decimalsDiff)
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
	// Find the last bin with amountIn >= bin.cummulativeAmountIn
	// (can be derived from bin.cummulativeVolume and bin.rate)
	binIdx := -1
	for i, bin := range bins {
		amountOutF, _ := bin.CumulativeVolume.Float64()
		amountInF := amountOutF * math.Pow10(int(decimalsDiff)) / bin.Rate
		amountInF = math.Ceil(amountInF)
		amountInRounded := new(big.Int).SetUint64(uint64(amountInF))

		if amountIn.Cmp(amountInRounded) >= 0 {
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

	curBinAmountOutF, _ := bins[binIdx].CumulativeVolume.Float64()
	curBinAmountInF := curBinAmountOutF * math.Pow10(int(decimalsDiff)) / bins[binIdx].Rate

	nextBinAmountOutF, _ := bins[binIdx+1].CumulativeVolume.Float64()
	nextBinAmountInF := nextBinAmountOutF * math.Pow10(int(decimalsDiff)) / bins[binIdx+1].Rate

	// Linear interpolation
	amountInF, _ := amountIn.Float64()
	amountOutF := curBinAmountOutF + (amountInF-curBinAmountInF)*(nextBinAmountOutF-curBinAmountOutF)/(nextBinAmountInF-curBinAmountInF)
	amountOut, _ := big.NewFloat(amountOutF).Int(nil)

	return amountOut, nil
}
