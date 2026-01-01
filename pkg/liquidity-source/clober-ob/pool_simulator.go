package cloberob

import (
	"math/big"
	"slices"
	"strings"

	"github.com/KyberNetwork/int256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	cloberlib "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/clober-ob/libraries"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
)

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

type PoolSimulator struct {
	pool.Pool
	Extra
	StaticExtra
	unitSize *uint256.Int
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
		Extra:       extra,
		StaticExtra: staticExtra,
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

	takenQuoteAmount, spentBaseAmount, fee, tickIdx, err := getExpectedOutput(u256.U0, amountIn, s.Depths, s.TakerPolicy, s.unitSize)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  s.Info.Tokens[indexOut],
			Amount: takenQuoteAmount.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  lo.Ternary(s.TakerPolicy.UsesQuote(), tokenOut, tokenIn),
			Amount: fee.ToBig(),
		},
		Gas: defaultBaseGas + defaultTakeGas*(int64(tickIdx)+1),
		SwapInfo: SwapInfo{
			SpentBaseAmount: spentBaseAmount,
			LimitPrice:      u256.U0,
		},
	}, nil
}

func getExpectedOutput(
	limitPrice, pBaseAmount *uint256.Int,
	depths []Liquidity, takerPolicy cloberlib.FeePolicy, unitSize *uint256.Int) (
	*uint256.Int, *uint256.Int, *uint256.Int, int, error) {
	var takenQuoteAmount, spentBaseAmount, feeAmount uint256.Int

	if len(depths) == 0 {
		return nil, nil, nil, 0, ErrNoLiquidity
	}

	tempU, tempI := new(uint256.Int), new(int256.Int)
	tick, tickIndex := depths[0].Tick, 0

	for !spentBaseAmount.Gt(pBaseAmount) && tick >= int24Min {
		tickToPrice, err := cloberlib.ToPrice(tick)
		if err != nil {
			return nil, nil, nil, 0, err
		}

		if limitPrice.Gt(tickToPrice) {
			break
		}

		var maxAmount uint256.Int
		if takerPolicy.UsesQuote() {
			maxAmount.Sub(pBaseAmount, &spentBaseAmount)
		} else {
			maxAmount.Set(takerPolicy.CalculateOriginalAmount(tempU.Sub(pBaseAmount, &spentBaseAmount), false))
		}

		tempU, err = cloberlib.BaseToQuote(tick, &maxAmount, false)
		if err != nil {
			return nil, nil, nil, 0, err
		}
		maxAmount.Div(tempU, unitSize)

		if maxAmount.IsZero() {
			break
		}

		currentDepth := new(uint256.Int).SetUint64(depths[tickIndex].Depth)
		quoteAmount := new(uint256.Int)
		if currentDepth.Gt(&maxAmount) {
			quoteAmount.Mul(&maxAmount, unitSize)
		} else {
			quoteAmount.Mul(currentDepth, unitSize)
		}

		baseAmount, err := cloberlib.QuoteToBase(tick, quoteAmount, true)
		if err != nil {
			return nil, nil, nil, 0, err
		}

		if takerPolicy.UsesQuote() {
			tempI.SetFromBig(quoteAmount.ToBig())
			fee := takerPolicy.CalculateFee(quoteAmount, false)
			tempI.Sub(tempI, fee)
			quoteAmount.SetFromBig(tempI.ToBig())

			if fee.Sign() > 0 {
				feeAmount.Add(&feeAmount, u256.FromBig(fee.ToBig()))
			} else {
				feeAmount.Add(&feeAmount, u256.FromBig(fee.Neg(fee).ToBig()))
			}
		} else {
			tempI.SetFromBig(baseAmount.ToBig())
			fee := takerPolicy.CalculateFee(baseAmount, false)
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

		if tickIndex+1 >= len(depths) {
			break
		}

		tickIndex++
		tick = depths[tickIndex].Tick
	}

	return &takenQuoteAmount, &spentBaseAmount, &feeAmount, tickIndex, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swapInfo, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		return
	}

	spentBaseAmount := swapInfo.SpentBaseAmount.Clone()
	for i := 0; i < len(s.Depths) && spentBaseAmount.Sign() > 0; i++ {
		depth := &s.Depths[i]
		tick := depth.Tick

		quoteAmount, err := cloberlib.BaseToQuote(tick, spentBaseAmount, false)
		if err != nil {
			return
		}

		units := new(uint256.Int).Div(quoteAmount, s.unitSize)

		currentDepth := uint256.NewInt(depth.Depth)
		if !currentDepth.Gt(units) {
			depth.Depth = 0
			takenAmount := new(uint256.Int).Mul(currentDepth, s.unitSize)
			spentAmount, err := cloberlib.QuoteToBase(tick, takenAmount, true)
			if err != nil {
				return
			}
			spentBaseAmount.Sub(spentBaseAmount, spentAmount)
		} else {
			depth.Depth -= units.Uint64()
			spentBaseAmount.Clear()
		}
	}

	highestIdx := 0
	for highestIdx < len(s.Depths) && s.Depths[highestIdx].Depth == 0 {
		highestIdx++
	}

	s.Depths = s.Depths[highestIdx:]
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return Meta{
		BookManager: s.BookManager,
		Base:        s.Base,
		Quote:       s.Quote,
		UnitSize:    s.unitSize.Uint64(),
		MakerPolicy: s.MakerPolicy,
		TakerPolicy: s.TakerPolicy,
		Hooks:       s.Hooks,
		HookData:    []byte{},
		BlockNumber: s.Info.BlockNumber,
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.Depths = slices.Clone(s.Depths)

	return &cloned
}

func (s *PoolSimulator) CanSwapFrom(token string) []string {
	if strings.EqualFold(s.Info.Tokens[0], token) {
		return []string{s.Info.Tokens[1]}
	}

	return []string{}
}

func (s *PoolSimulator) CanSwapTo(token string) []string {
	if strings.EqualFold(s.Info.Tokens[1], token) {
		return []string{s.Info.Tokens[0]}
	}

	return []string{}
}
