package cloberob

import (
	"math/big"

	"github.com/KyberNetwork/int256"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	cloberlib "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/clober-ob/libraries"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

type PoolSimulator struct {
	pool.Pool
	highest     cloberlib.Tick
	depths      []Liquidity
	takerPolicy cloberlib.FeePolicy
	unitSize    *uint256.Int
}

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
			Tokens:      lo.Map(ep.Tokens, func(e *entity.PoolToken, _ int) string { return e.Address }),
			Reserves:    lo.Map(ep.Reserves, func(e string, _ int) *big.Int { return bignumber.NewBig(e) }),
			BlockNumber: ep.BlockNumber,
		}},
		highest:     extra.Highest,
		depths:      extra.Depths,
		takerPolicy: staticExtra.TakerPolicy,
		unitSize:    uint256.NewInt(staticExtra.UnitSize),
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	var (
		amountIn = uint256.MustFromBig(params.TokenAmountIn.Amount)
		tokenIn  = params.TokenAmountIn.Token
		tokenOut = params.TokenOut
	)
	indexIn, indexOut := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	takenQuoteAmount, _, fee, err := s.getExpectedOutput(u256.U0, amountIn)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  s.Info.Tokens[indexOut],
			Amount: takenQuoteAmount.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  lo.Ternary(s.takerPolicy.UsesQuote(), tokenOut, tokenIn),
			Amount: fee.ToBig(),
		},
	}, nil
}

func (s *PoolSimulator) getExpectedOutput(limitPrice, baseAmount *uint256.Int) (
	*uint256.Int, *uint256.Int, *uint256.Int, error,
) {
	var takenQuoteAmount, spentBaseAmount, feeAmount uint256.Int

	if len(s.depths) == 0 {
		return nil, nil, nil, ErrNoLiquidity
	}

	tempU, tempI := new(uint256.Int), new(int256.Int)
	tick, tickIndex := s.highest, 0

	if tick != s.depths[0].Tick {
		return nil, nil, nil, ErrInvalidState
	}

	for {
		if spentBaseAmount.Gt(baseAmount) || tick < int24Min {
			break
		}

		tickToPrice, err := cloberlib.ToPrice(tick)
		if err != nil {
			return nil, nil, nil, err
		}
		if limitPrice.Gt(tickToPrice) {
			break
		}

		var maxAmount uint256.Int
		if s.takerPolicy.UsesQuote() {
			maxAmount.Sub(baseAmount, &spentBaseAmount)
		} else {
			maxAmount.Set(s.takerPolicy.CalculateOriginalAmount(tempU.Sub(baseAmount, &spentBaseAmount), false))
		}

		tempU, err = cloberlib.BaseToQuote(tick, &maxAmount, false)
		if err != nil {
			return nil, nil, nil, err
		}
		maxAmount.Div(tempU, s.unitSize)

		if maxAmount.IsZero() {
			break
		}

		currentDepth := new(uint256.Int).SetUint64(s.depths[tickIndex].Depth)
		quoteAmount := new(uint256.Int)
		if currentDepth.Gt(&maxAmount) {
			quoteAmount.Mul(&maxAmount, s.unitSize)
		} else {
			quoteAmount.Mul(currentDepth, s.unitSize)
		}
		baseAmount, err = cloberlib.QuoteToBase(tick, quoteAmount, true)
		if err != nil {
			return nil, nil, nil, err
		}

		if s.takerPolicy.UsesQuote() {
			tempI.Set(u256.SNeg(quoteAmount))
			fee := s.takerPolicy.CalculateFee(quoteAmount, false)
			tempI.Sub(tempI, fee)
			quoteAmount.SetFromBig(tempI.ToBig())

			if fee.Sign() > 0 {
				feeAmount.Add(&feeAmount, u256.FromBig(fee.ToBig()))
			} else {
				feeAmount.Add(&feeAmount, u256.FromBig(fee.Neg(fee).ToBig()))
			}
		} else {
			tempI.Set(u256.SInt256(baseAmount))
			fee := s.takerPolicy.CalculateFee(baseAmount, false)
			tempI.Add(tempI, fee)
			baseAmount.SetFromBig(tempI.ToBig())

			if fee.Sign() > 0 {
				feeAmount.Add(&feeAmount, u256.FromBig(fee.ToBig()))
			} else {
				feeAmount.Add(&feeAmount, u256.FromBig(fee.Neg(fee).ToBig()))
			}
		}

		if baseAmount.IsZero() {
			break
		}

		takenQuoteAmount.Add(&takenQuoteAmount, quoteAmount)
		spentBaseAmount.Add(&spentBaseAmount, baseAmount)

		if tickIndex+1 >= len(s.depths) {
			break
		}

		tickIndex++
		tick = s.depths[tickIndex].Tick
	}

	return &takenQuoteAmount, &spentBaseAmount, &feeAmount, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {

}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return Meta{
		BlockNumber: s.Info.BlockNumber,
	}
}
