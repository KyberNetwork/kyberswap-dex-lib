package tessera

import (
	"encoding/json"
	"math/big"
	"strings"

	"github.com/holiman/uint256"
	"github.com/rs/zerolog/log"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	pool.Pool
	extra Extra

	tesseraSwap string
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)
var _ = pool.RegisterUseSwapLimit(valueobject.ExchangeTessera)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	tokens := make([]string, len(entityPool.Tokens))
	for i, t := range entityPool.Tokens {
		tokens[i] = strings.ToLower(t.Address)
	}

	reserves := make([]*big.Int, len(entityPool.Reserves))
	for i, r := range entityPool.Reserves {
		reserves[i], _ = new(big.Int).SetString(r, 10)
	}

	if extra.MaxBaseToQuoteAmount != nil {
		extra.MaxBaseToQuoteAmount.MulDivOverflow(extra.MaxBaseToQuoteAmount,
			maxOrderbookFillFactorBPS, big256.UBasisPoint)
	}
	if extra.MaxQuoteToBaseAmount != nil {
		extra.MaxQuoteToBaseAmount.MulDivOverflow(extra.MaxQuoteToBaseAmount,
			maxOrderbookFillFactorBPS, big256.UBasisPoint)
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:     entityPool.Address,
				Exchange:    entityPool.Exchange,
				Type:        entityPool.Type,
				Tokens:      tokens,
				Reserves:    reserves,
				BlockNumber: entityPool.BlockNumber,
			},
		},
		extra:       extra,
		tesseraSwap: staticExtra.TesseraSwap,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := params.TokenAmountIn
	tokenOut := strings.ToLower(params.TokenOut)
	tokenIn := strings.ToLower(tokenAmountIn.Token)

	if s.GetTokenIndex(tokenIn) < 0 || s.GetTokenIndex(tokenOut) < 0 {
		return nil, ErrInvalidToken
	}

	if !s.extra.TradingEnabled {
		return nil, ErrTradingDisabled
	}

	if !s.extra.IsInitialised {
		return nil, ErrNotInitialised
	}

	var isBaseToQuote bool
	if strings.EqualFold(tokenIn, s.Info.Tokens[0]) {
		isBaseToQuote = true
	}

	amountInRaw := uint256.MustFromBig(tokenAmountIn.Amount)

	// Now only support swaps up to max prefetch points
	// Quoter may accept larger amounts but interpolation has no data points beyond this range
	// This prevents price deviation when swapping beyond the highest price level
	var maxPrefetchAmount *uint256.Int
	var prefetches []PrefetchRate
	if isBaseToQuote {
		maxPrefetchAmount = s.extra.MaxBaseToQuoteAmount
		prefetches = s.extra.BaseToQuotePrefetches
	} else {
		maxPrefetchAmount = s.extra.MaxQuoteToBaseAmount
		prefetches = s.extra.QuoteToBasePrefetches
	}

	if maxPrefetchAmount != nil && amountInRaw.Cmp(maxPrefetchAmount) > 0 {
		return nil, ErrSwapReverted
	}

	amountOut, err := GetClosestRate(amountInRaw, prefetches)
	if err != nil {
		return nil, err
	}

	if limit := params.Limit; limit != nil {
		inventoryLimit := limit.GetLimit(tokenOut)
		if amountOut.CmpBig(inventoryLimit) > 0 {
			return nil, ErrSwapReverted
		}
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: bignumber.ZeroBI,
		},
		Gas: defaultGas,
	}, nil
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s

	cloned.Info.Reserves = make([]*big.Int, len(s.Info.Reserves))
	for i, r := range s.Info.Reserves {
		cloned.Info.Reserves[i] = new(big.Int).Set(r)
	}

	cloned.extra.BaseToQuotePrefetches = make([]PrefetchRate, len(s.extra.BaseToQuotePrefetches))
	for i, p := range s.extra.BaseToQuotePrefetches {
		cloned.extra.BaseToQuotePrefetches[i] = PrefetchRate{
			AmountIn: p.AmountIn.Clone(),
			Rate:     p.Rate.Clone(),
		}
	}

	cloned.extra.QuoteToBasePrefetches = make([]PrefetchRate, len(s.extra.QuoteToBasePrefetches))
	for i, p := range s.extra.QuoteToBasePrefetches {
		cloned.extra.QuoteToBasePrefetches[i] = PrefetchRate{
			AmountIn: p.AmountIn.Clone(),
			Rate:     p.Rate.Clone(),
		}
	}

	if s.extra.MaxBaseToQuoteAmount != nil {
		cloned.extra.MaxBaseToQuoteAmount = s.extra.MaxBaseToQuoteAmount.Clone()
	}
	if s.extra.MaxQuoteToBaseAmount != nil {
		cloned.extra.MaxQuoteToBaseAmount = s.extra.MaxQuoteToBaseAmount.Clone()
	}

	return &cloned
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	tokenIn := params.TokenAmountIn.Token
	tokenOut := params.TokenAmountOut.Token

	amtIn := params.TokenAmountIn.Amount
	amtOut := params.TokenAmountOut.Amount

	if strings.EqualFold(tokenIn, s.Info.Tokens[0]) {
		s.Info.Reserves[0] = new(big.Int).Add(
			s.Info.Reserves[0],
			amtIn,
		)
		s.Info.Reserves[1] = new(big.Int).Sub(
			s.Info.Reserves[1],
			amtOut,
		)

		s.extra.BaseToQuotePrefetches = s.updatePrefetches(s.extra.BaseToQuotePrefetches, uint256.MustFromBig(amtIn), uint256.MustFromBig(amtOut))
		if s.extra.MaxBaseToQuoteAmount != nil {
			s.extra.MaxBaseToQuoteAmount = subTill0U(s.extra.MaxBaseToQuoteAmount, uint256.MustFromBig(amtIn))
		}
	} else {
		s.Info.Reserves[1] = new(big.Int).Add(
			s.Info.Reserves[1],
			amtIn,
		)
		s.Info.Reserves[0] = new(big.Int).Sub(
			s.Info.Reserves[0],
			amtOut,
		)

		s.extra.QuoteToBasePrefetches = s.updatePrefetches(s.extra.QuoteToBasePrefetches, uint256.MustFromBig(amtIn), uint256.MustFromBig(amtOut))
		if s.extra.MaxQuoteToBaseAmount != nil {
			s.extra.MaxQuoteToBaseAmount = subTill0U(s.extra.MaxQuoteToBaseAmount, uint256.MustFromBig(amtIn))
		}
	}

	if limit := params.SwapLimit; limit != nil {
		_, _, err := limit.UpdateLimit(tokenOut, tokenIn, amtOut, amtIn)
		if err != nil {
			log.Err(err).Msg("tessera.UpdateBalance failed")
		}
	}
}

func (s *PoolSimulator) CalculateLimit() map[string]*big.Int {
	tokens, reserves := s.GetTokens(), s.GetReserves()
	inventory := make(map[string]*big.Int, len(tokens))
	for i, token := range tokens {
		inventory[token] = reserves[i]
	}
	return inventory
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return struct {
		BlockNumber uint64 `json:"blockNumber"`
		TesseraSwap string `json:"tesseraSwap"`
	}{
		BlockNumber: s.Info.BlockNumber,
		TesseraSwap: s.tesseraSwap,
	}
}

func (s *PoolSimulator) updatePrefetches(prefetches []PrefetchRate, amtIn, amtOut *uint256.Int) []PrefetchRate {
	for i := range prefetches {
		prefetches[i].AmountIn = subTill0U(prefetches[i].AmountIn, amtIn)
		if prefetches[i].Rate != nil {
			prefetches[i].Rate = subTill0U(prefetches[i].Rate, amtOut)
		}
	}

	for len(prefetches) > 0 && prefetches[0].AmountIn.IsZero() {
		prefetches = prefetches[1:]
	}

	return prefetches
}

func subTill0U(a, b *uint256.Int) *uint256.Int {
	if a.Cmp(b) <= 0 {
		return uint256.NewInt(0)
	}
	return new(uint256.Int).Sub(a, b)
}
