package ringswapbacking

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolSimulator struct {
	pool.Pool
	Extra
	StaticExtra
	consumed bool
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)
var _ = pool.RegisterUseSwapLimit(DexType)

func NewPoolSimulator(p entity.Pool) (*PoolSimulator, error) {
	if len(p.Tokens) != 2 || len(p.Reserves) != 2 {
		return nil, ErrInvalidState
	}
	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return nil, err
	}
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}
	if !validExtra(extra) || !validStaticExtra(p, staticExtra) {
		return nil, ErrInvalidState
	}
	return &PoolSimulator{
		Pool:        pool.FromEntity(p),
		Extra:       extra,
		StaticExtra: staticExtra,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(
	params pool.CalcAmountOutParams,
) (*pool.CalcAmountOutResult, error) {
	if s.consumed {
		return nil, ErrSourceAlreadyUsed
	}
	indexIn := s.GetTokenIndex(strings.ToLower(params.TokenAmountIn.Token))
	indexOut := s.GetTokenIndex(strings.ToLower(params.TokenOut))
	if indexIn < 0 || indexOut < 0 || indexIn == indexOut {
		return nil, ErrInvalidToken
	}
	if params.TokenAmountIn.Amount == nil {
		return nil, uniswapv2.ErrInvalidAmountIn
	}
	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow || amountIn.Sign() <= 0 {
		return nil, uniswapv2.ErrInvalidAmountIn
	}

	reserveIn, reserveOut, err := s.directionalReserves(indexIn)
	if err != nil {
		return nil, err
	}
	amountOut, err := getAmountOut(amountIn, reserveIn, reserveOut)
	if err != nil {
		return nil, err
	}
	if amountOut.Sign() <= 0 || amountOut.Cmp(reserveOut) >= 0 {
		return nil, ErrInsufficientOutput
	}

	bufferOut, capacityOut, wrapperIn, wrapperOut := s.directionalBacking(indexIn)
	amountOutBig := amountOut.ToBig()
	if params.Limit == nil {
		return nil, ErrNoSwapLimit
	}
	deliverable := params.Limit.GetLimit(wrapperOut)
	if deliverable == nil || amountOutBig.Cmp(deliverable) > 0 {
		return nil, ErrInsufficientBacking
	}

	useRecall := amountOutBig.Cmp(bufferOut) > 0
	if useRecall {
		shortfall := new(big.Int).Sub(amountOutBig, bufferOut)
		if shortfall.Cmp(capacityOut) > 0 {
			return nil, ErrInsufficientBacking
		}
	}

	gas := s.NoRecallGasToken1
	if indexOut == 0 {
		gas = s.NoRecallGasToken0
	}
	if useRecall && indexOut == 0 {
		gas = s.RecallGasToken0
	} else if useRecall {
		gas = s.RecallGasToken1
	}
	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: amountOutBig},
		Fee: &pool.TokenAmount{
			Token:  params.TokenAmountIn.Token,
			Amount: big.NewInt(0),
		},
		Gas: gas,
		SwapInfo: SwapInfo{
			RouterAddress:  s.RouterAddress,
			UnderlyingPair: s.PairAddress,
			WrapperIn:      wrapperIn,
			WrapperOut:     wrapperOut,
			UseRecall:      useRecall,
		},
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	info, ok := params.SwapInfo.(SwapInfo)
	if !ok || s.consumed {
		return
	}
	if params.SwapLimit != nil {
		_, _, _ = params.SwapLimit.UpdateLimit(
			info.WrapperOut,
			info.WrapperIn,
			params.TokenAmountOut.Amount,
			params.TokenAmountIn.Amount,
		)
	}
	s.consumed = true
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return PoolMeta{
		ApprovalAddress:      s.RouterAddress,
		RouterAddress:        s.RouterAddress,
		UnderlyingPair:       s.PairAddress,
		SingleUse:            true,
		ReplacesOrdinaryPair: true,
	}
}

func (s *PoolSimulator) GetApprovalAddress(_, _ string) string {
	return s.RouterAddress
}

func (s *PoolSimulator) CalculateLimit() map[string]*big.Int {
	return map[string]*big.Int{
		s.Wrapper0: new(big.Int).Add(s.WrapperBuffer0, s.RecallCapacity0),
		s.Wrapper1: new(big.Int).Add(s.WrapperBuffer1, s.RecallCapacity1),
	}
}

func (s *PoolSimulator) directionalReserves(indexIn int) (*uint256.Int, *uint256.Int, error) {
	reserveIn, overflow := uint256.FromBig(s.Info.Reserves[indexIn])
	if overflow || reserveIn.Sign() <= 0 {
		return nil, nil, ErrInvalidState
	}
	reserveOut, overflow := uint256.FromBig(s.Info.Reserves[1-indexIn])
	if overflow || reserveOut.Sign() <= 0 {
		return nil, nil, ErrInvalidState
	}
	return reserveIn, reserveOut, nil
}

func (s *PoolSimulator) directionalBacking(
	indexIn int,
) (bufferOut, capacityOut *big.Int, wrapperIn, wrapperOut string) {
	if indexIn == 0 {
		return s.WrapperBuffer1, s.RecallCapacity1, s.Wrapper0, s.Wrapper1
	}
	return s.WrapperBuffer0, s.RecallCapacity0, s.Wrapper1, s.Wrapper0
}

func validExtra(extra Extra) bool {
	values := []*big.Int{
		extra.WrapperBuffer0,
		extra.WrapperBuffer1,
		extra.RecallCapacity0,
		extra.RecallCapacity1,
	}
	for _, value := range values {
		if value == nil || value.Sign() < 0 {
			return false
		}
	}
	return true
}

func getAmountOut(
	amountIn *uint256.Int,
	reserveIn *uint256.Int,
	reserveOut *uint256.Int,
) (amountOut *uint256.Int, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			if recoveredError, ok := recovered.(error); ok {
				err = recoveredError
			} else {
				err = fmt.Errorf("ringswap-backing math panic: %v", recovered)
			}
		}
	}()
	amountInWithFee := uniswapv2.SafeMul(amountIn, uint256.NewInt(feeNumerator))
	numerator := uniswapv2.SafeMul(amountInWithFee, reserveOut)
	denominator := uniswapv2.SafeAdd(
		uniswapv2.SafeMul(reserveIn, uint256.NewInt(feeDenominator)),
		amountInWithFee,
	)
	return new(uint256.Int).Div(numerator, denominator), nil
}
