package canonic

import (
	"math/big"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	midPrice   *uint256.Int
	midPrec    *uint256.Int
	takerFee   *uint256.Int
	baseScale  *uint256.Int
	quoteScale *uint256.Int
	askBps     []uint16
	askVols    []*uint256.Int
	bidBps     []uint16
	bidVols    []*uint256.Int
	active     bool

	baseToken  string
	quoteToken string
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if len(ep.StaticExtra) > 0 {
		if err := json.Unmarshal([]byte(ep.StaticExtra), &staticExtra); err != nil {
			return nil, err
		}
	}

	if len(extra.AskBps) != len(extra.AskVols) || len(extra.BidBps) != len(extra.BidVols) {
		return nil, ErrInvalidState
	}

	baseToken := lo.Ternary(staticExtra.BaseToken != "", staticExtra.BaseToken, ep.Tokens[0].Address)
	quoteToken := lo.Ternary(staticExtra.QuoteToken != "", staticExtra.QuoteToken, ep.Tokens[1].Address)

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:     ep.Address,
				Exchange:    ep.Exchange,
				Type:        ep.Type,
				Tokens:      lo.Map(ep.Tokens, func(t *entity.PoolToken, _ int) string { return t.Address }),
				Reserves:    lo.Map(ep.Reserves, func(r string, _ int) *big.Int { return bignumber.NewBig(r) }),
				BlockNumber: ep.BlockNumber,
			},
		},
		midPrice:   extra.MidPrice,
		midPrec:    extra.MidPrec,
		takerFee:   extra.TakerFee,
		baseScale:  extra.BaseScale,
		quoteScale: extra.QuoteScale,
		askBps:     extra.AskBps,
		askVols:    extra.AskVols,
		bidBps:     extra.BidBps,
		bidVols:    extra.BidVols,
		active:     extra.Active,
		baseToken:  baseToken,
		quoteToken: quoteToken,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn := param.TokenAmountIn.Token
	tokenOut := param.TokenOut

	indexIn, indexOut := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	amountIn := uint256.MustFromBig(param.TokenAmountIn.Amount)
	if amountIn.Sign() <= 0 {
		return nil, ErrInvalidAmountIn
	}

	if err := s.validate(); err != nil {
		return nil, err
	}

	isSellBase := tokenIn == s.baseToken

	var amountOut, fee *uint256.Int
	if isSellBase {
		netQuote, quoteFee, _ := calcSellBaseTargetIn(
			amountIn, s.midPrice, s.midPrec, s.takerFee, s.baseScale, s.quoteScale,
			s.bidBps, s.bidVols,
		)
		amountOut = netQuote
		fee = quoteFee
	} else {
		netBase, baseFee, _ := calcBuyBaseTargetIn(
			amountIn, s.midPrice, s.midPrec, s.takerFee, s.baseScale, s.quoteScale,
			s.askBps, s.askVols,
		)
		amountOut = netBase
		fee = baseFee
	}

	if amountOut.IsZero() {
		return nil, ErrInsufficientLiquidity
	}

	feeToken := tokenOut

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: feeToken, Amount: fee.ToBig()},
		Gas:            defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	tokenIn := param.TokenIn
	tokenOut := param.TokenAmountOut.Token

	indexIn, indexOut := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	amountOut := uint256.MustFromBig(param.TokenAmountOut.Amount)
	if amountOut.Sign() <= 0 {
		return nil, ErrInvalidAmountOut
	}

	if err := s.validate(); err != nil {
		return nil, err
	}

	isSellBase := tokenIn == s.baseToken

	var amountIn, fee *uint256.Int
	if isSellBase {
		baseNeeded, quoteFee := calcSellBaseAmountIn(
			amountOut, s.midPrice, s.midPrec, s.takerFee, s.baseScale, s.quoteScale,
			s.bidBps, s.bidVols,
		)
		if baseNeeded == nil {
			return nil, ErrInsufficientLiquidity
		}
		amountIn = baseNeeded
		fee = quoteFee
	} else {
		quoteNeeded, baseFee := calcBuyBaseAmountIn(
			amountOut, s.midPrice, s.midPrec, s.takerFee, s.baseScale, s.quoteScale,
			s.askBps, s.askVols,
		)
		if quoteNeeded == nil {
			return nil, ErrInsufficientLiquidity
		}
		amountIn = quoteNeeded
		fee = baseFee
	}

	feeToken := tokenOut

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: tokenIn, Amount: amountIn.ToBig()},
		Fee:           &pool.TokenAmount{Token: feeToken, Amount: fee.ToBig()},
		Gas:           defaultGas,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return
	}

	s.Info.Reserves[indexIn] = new(big.Int).Add(s.Info.Reserves[indexIn], params.TokenAmountIn.Amount)
	s.Info.Reserves[indexOut] = new(big.Int).Sub(s.Info.Reserves[indexOut], params.TokenAmountOut.Amount)

	isSellBase := params.TokenAmountIn.Token == s.baseToken
	amountOut := uint256.MustFromBig(params.TokenAmountOut.Amount)
	feeAmount := uint256.MustFromBig(params.Fee.Amount)
	grossOut := new(uint256.Int).Add(amountOut, feeAmount)

	if isSellBase {
		s.consumeBidVolumes(grossOut)
	} else {
		s.consumeAskVolumes(grossOut)
	}
}

func (s *PoolSimulator) consumeAskVolumes(baseAmount *uint256.Int) {
	remaining := new(uint256.Int).Set(baseAmount)
	for i := range s.askVols {
		if remaining.IsZero() {
			break
		}
		if s.askVols[i].Cmp(remaining) <= 0 {
			remaining.Sub(remaining, s.askVols[i])
			s.askVols[i] = new(uint256.Int)
		} else {
			s.askVols[i] = new(uint256.Int).Sub(s.askVols[i], remaining)
			remaining = new(uint256.Int)
		}
	}
}

func (s *PoolSimulator) consumeBidVolumes(quoteAmount *uint256.Int) {
	remaining := new(uint256.Int).Set(quoteAmount)
	for i := range s.bidVols {
		if remaining.IsZero() {
			break
		}
		quoteAtRung := s.bidVols[i]
		if quoteAtRung.Cmp(remaining) <= 0 {
			remaining.Sub(remaining, quoteAtRung)
			s.bidVols[i] = new(uint256.Int)
		} else {
			s.bidVols[i] = new(uint256.Int).Sub(quoteAtRung, remaining)
			remaining = new(uint256.Int)
		}
	}
}

func (s *PoolSimulator) GetMetaInfo(tokenIn, _ string) any {
	return PoolMeta{
		BlockNumber: s.Info.BlockNumber,
		MAOB:        s.Info.Address,
		IsSellBase:  tokenIn == s.baseToken,
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.midPrice = new(uint256.Int).Set(s.midPrice)
	cloned.midPrec = new(uint256.Int).Set(s.midPrec)
	cloned.takerFee = new(uint256.Int).Set(s.takerFee)
	cloned.baseScale = new(uint256.Int).Set(s.baseScale)
	cloned.quoteScale = new(uint256.Int).Set(s.quoteScale)
	cloned.askVols = make([]*uint256.Int, len(s.askVols))
	for i, v := range s.askVols {
		cloned.askVols[i] = new(uint256.Int).Set(v)
	}
	cloned.bidVols = make([]*uint256.Int, len(s.bidVols))
	for i, v := range s.bidVols {
		cloned.bidVols[i] = new(uint256.Int).Set(v)
	}
	return &cloned
}

func (s *PoolSimulator) validate() error {
	if !s.active {
		return ErrMarketNotActive
	}
	if s.midPrice == nil || s.midPrice.IsZero() {
		return ErrZeroMidPrice
	}
	if s.midPrec == nil || s.midPrec.IsZero() {
		return ErrInvalidState
	}
	if s.baseScale == nil || s.baseScale.IsZero() {
		return ErrInvalidState
	}
	if s.takerFee != nil && s.takerFee.Cmp(feeDenom) >= 0 {
		return ErrInvalidState
	}
	if len(s.askBps) == 0 && len(s.bidBps) == 0 {
		return ErrNoRungs
	}
	return nil
}
