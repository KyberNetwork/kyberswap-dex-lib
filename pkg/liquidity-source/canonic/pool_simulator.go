package canonic

import (
	"math/big"
	"slices"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	baseToken     string
	quoteToken    string
	baseDecimals  uint8
	quoteDecimals uint8
	baseScale     *uint256.Int
	quoteScale    *uint256.Int

	midPrice       *uint256.Int
	midPrecision   *uint256.Int
	oracleUpdAt    uint64
	takerFee       *uint256.Int
	feeDenom       *uint256.Int
	minQuoteTaker  *uint256.Int
	marketState    uint8
	stateExpiresAt uint64
	rungDenom      *uint256.Int
	priceSigfigs   *uint256.Int

	askRungs   []uint16
	askVolumes []*uint256.Int
	bidRungs   []uint16
	bidVolumes []*uint256.Int
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(ep.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     ep.Address,
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      lo.Map(ep.Tokens, func(t *entity.PoolToken, _ int) string { return t.Address }),
			Reserves:    lo.Map(ep.Reserves, func(r string, _ int) *big.Int { return bignumber.NewBig(r) }),
			BlockNumber: ep.BlockNumber,
		}},
		baseToken:     staticExtra.BaseToken,
		quoteToken:    staticExtra.QuoteToken,
		baseDecimals:  staticExtra.BaseDecimals,
		quoteDecimals: staticExtra.QuoteDecimals,
		baseScale:     uint256.MustFromDecimal(staticExtra.BaseScale),
		quoteScale:    uint256.MustFromDecimal(staticExtra.QuoteScale),

		midPrice:       uint256.MustFromDecimal(extra.MidPrice),
		midPrecision:   uint256.MustFromDecimal(extra.MidPrecision),
		oracleUpdAt:    extra.OracleUpdAt,
		takerFee:       uint256.NewInt(uint64(extra.TakerFee)),
		feeDenom:       uint256.MustFromDecimal(extra.FeeDenom),
		minQuoteTaker:  uint256.MustFromDecimal(extra.MinQuoteTaker),
		marketState:    extra.MarketState,
		stateExpiresAt: extra.StateExpiresAt,
		rungDenom:      uint256.MustFromDecimal(extra.RungDenom),
		priceSigfigs:   uint256.MustFromDecimal(extra.PriceSigfigs),

		askRungs:   extra.AskRungs,
		askVolumes: parseVolumes(extra.AskVolumes),
		bidRungs:   extra.BidRungs,
		bidVolumes: parseVolumes(extra.BidVolumes),
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn := param.TokenAmountIn.Token
	tokenOut := param.TokenOut
	amountIn := param.TokenAmountIn.Amount

	if amountIn == nil || amountIn.Sign() <= 0 {
		return nil, ErrInvalidAmountIn
	}

	idxIn := s.GetTokenIndex(tokenIn)
	idxOut := s.GetTokenIndex(tokenOut)
	if idxIn < 0 || idxOut < 0 || idxIn == idxOut {
		return nil, ErrInvalidToken
	}

	if err := s.checkMarketState(); err != nil {
		return nil, err
	}

	if s.oracleUpdAt == 0 {
		return nil, ErrOracleStale
	}

	amtIn, overflow := uint256.FromBig(amountIn)
	if overflow {
		return nil, ErrInvalidAmountIn
	}

	var (
		amountOut *uint256.Int
		fee       *uint256.Int
		err       error
	)

	isBuyBase := tokenIn == s.quoteToken

	if isBuyBase {
		if err = s.checkMinQuoteTakerDirect(amtIn); err != nil {
			return nil, err
		}

		amountOut, fee, err = calcBuyBaseTargetIn(
			amtIn,
			s.midPrice, s.midPrecision,
			s.askRungs, s.askVolumes,
			s.takerFee, s.feeDenom,
			s.rungDenom,
			s.baseScale, s.quoteScale,
			s.priceSigfigs.Uint64(),
		)
	} else {
		if err = s.checkMinQuoteTakerForBase(amtIn); err != nil {
			return nil, err
		}

		amountOut, fee, err = calcSellBaseTargetIn(
			amtIn,
			s.midPrice, s.midPrecision,
			s.bidRungs, s.bidVolumes,
			s.takerFee, s.feeDenom,
			s.rungDenom,
			s.baseScale, s.quoteScale,
			s.priceSigfigs.Uint64(),
		)
	}
	if err != nil {
		return nil, err
	}

	if amountOut.IsZero() {
		return nil, ErrInvalidAmountOut
	}

	feeToken := tokenOut
	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: feeToken, Amount: fee.ToBig()},
		Gas:            defaultGas,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	tokenIn := params.TokenAmountIn.Token
	tokenOut := params.TokenAmountOut.Token
	amountIn := params.TokenAmountIn.Amount
	amountOut := params.TokenAmountOut.Amount

	idxIn := s.GetTokenIndex(tokenIn)
	idxOut := s.GetTokenIndex(tokenOut)

	if idxIn >= 0 {
		s.Info.Reserves[idxIn] = new(big.Int).Add(s.Info.Reserves[idxIn], amountIn)
	}
	if idxOut >= 0 {
		s.Info.Reserves[idxOut] = new(big.Int).Sub(s.Info.Reserves[idxOut], amountOut)
	}

	amtOut := uint256.MustFromBig(amountOut)
	fee := uint256.MustFromBig(params.Fee.Amount)
	grossOut := new(uint256.Int).Add(amtOut, fee)

	isBuyBase := tokenIn == s.quoteToken && tokenOut == s.baseToken
	if isBuyBase {
		s.reduceAskVolumes(grossOut)
	} else {
		s.reduceBidVolumes(amtOut, fee)
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.Info.Reserves = slices.Clone(s.Info.Reserves)
	cloned.askVolumes = slices.Clone(s.askVolumes)
	cloned.bidVolumes = slices.Clone(s.bidVolumes)

	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(tokenIn, _ string) any {
	return PoolMeta{
		Pool:        s.Info.Address,
		IsBuyBase:   tokenIn == s.quoteToken,
		BlockNumber: s.Info.BlockNumber,
	}
}

func (s *PoolSimulator) checkMarketState() error {
	switch s.marketState {
	case marketStatePaused:
		return ErrMarketPaused
	case marketStateUnwindOnly:
		return ErrMarketUnwindOnly
	}
	return nil
}

func (s *PoolSimulator) checkMinQuoteTakerDirect(quoteAmount *uint256.Int) error {
	if s.minQuoteTaker.IsZero() {
		return nil
	}
	if quoteAmount.Cmp(s.minQuoteTaker) < 0 {
		return ErrQuoteAmountTooLow
	}

	return nil
}

func (s *PoolSimulator) checkMinQuoteTakerForBase(baseAmount *uint256.Int) error {
	if s.minQuoteTaker.IsZero() {
		return nil
	}
	quoteValue := estimateQuoteValue(baseAmount, s.midPrice, s.midPrecision, s.baseScale, s.quoteScale)
	if quoteValue.Cmp(s.minQuoteTaker) < 0 {
		return ErrQuoteAmountTooLow
	}

	return nil
}

func (s *PoolSimulator) reduceAskVolumes(grossBase *uint256.Int) {
	var remaining uint256.Int
	remaining.Set(grossBase)

	for i := range s.askVolumes {
		if remaining.IsZero() {
			break
		}
		if s.askVolumes[i].Cmp(&remaining) <= 0 {
			remaining.Sub(&remaining, s.askVolumes[i])
			s.askVolumes[i] = new(uint256.Int)
		} else {
			s.askVolumes[i] = new(uint256.Int).Sub(s.askVolumes[i], &remaining)
			remaining.Clear()
		}
	}
}

func (s *PoolSimulator) reduceBidVolumes(netQuote, feeQuote *uint256.Int) {
	var grossQuote, remaining uint256.Int
	grossQuote.Add(netQuote, feeQuote)
	remaining.Set(&grossQuote)

	for i := range s.bidVolumes {
		if remaining.IsZero() {
			break
		}
		if s.bidVolumes[i].Cmp(&remaining) <= 0 {
			remaining.Sub(&remaining, s.bidVolumes[i])
			s.bidVolumes[i] = new(uint256.Int)
		} else {
			s.bidVolumes[i] = new(uint256.Int).Sub(s.bidVolumes[i], &remaining)
			remaining.Clear()
		}
	}
}
