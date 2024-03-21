package balancerv1

import (
	"errors"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ErrNotBound        = errors.New("ERR_NOT_BOUND")
	ErrSwapNotPublic   = errors.New("ERR_SWAP_NOT_PUBLIC")
	ErrMathApprox      = errors.New("ERR_MATH_APPROX")
	ErrInvalidAmountIn = errors.New("invalid amount in")
	ErrMaxInRatio      = errors.New("ERR_MAX_IN_RATIO")
	ErrMaxTotalInRatio = errors.New("ERR_MAX_TOTAL_IN_RATIO")
)

type PoolSimulator struct {
	poolpkg.Pool

	Records    map[string]Record
	PublicSwap bool
	SwapFee    *uint256.Int

	TotalAmountsIn    map[string]*uint256.Int
	MaxTotalAmountsIn map[string]*uint256.Int

	Gas Gas
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra PoolExtra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var (
		totalAmountsIn    = make(map[string]*uint256.Int)
		maxTotalAmountsIn = make(map[string]*uint256.Int)
	)
	for _, token := range entityPool.Tokens {
		tokenAddr := token.Address
		balance := extra.Records[tokenAddr].Balance

		maxIn, err := BNum.BMul(balance, BConst.MAX_IN_RATIO)
		if err != nil {
			return nil, err
		}
		maxTotalAmountsIn[tokenAddr] = maxIn

		totalAmountsIn[tokenAddr] = number.Zero
	}

	return &PoolSimulator{
		Pool: poolpkg.Pool{Info: poolpkg.PoolInfo{
			Address:     entityPool.Address,
			ReserveUsd:  entityPool.ReserveUsd,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      lo.Map(entityPool.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves:    lo.Map(entityPool.Reserves, func(item string, index int) *big.Int { return utils.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		Records:           extra.Records,
		PublicSwap:        extra.PublicSwap,
		SwapFee:           extra.SwapFee,
		TotalAmountsIn:    totalAmountsIn,
		MaxTotalAmountsIn: maxTotalAmountsIn,
		Gas:               defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow {
		return nil, ErrInvalidAmountIn
	}

	amountOut, _, err := s.swapExactAmountIn(params.TokenAmountIn.Token, amountIn, params.TokenOut, nil, nil)
	if err != nil {
		return nil, err
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: params.TokenOut, Amount: amountOut.ToBig()},
		Fee:            &poolpkg.TokenAmount{Token: params.TokenAmountIn.Token, Amount: big.NewInt(0)},
		Gas:            s.Gas.SwapExactAmountIn,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params poolpkg.UpdateBalanceParams) {
	inRecord, outRecord := s.Records[params.TokenAmountIn.Token], s.Records[params.TokenAmountOut.Token]
	amountIn, amountOut := uint256.MustFromBig(params.TokenAmountIn.Amount), uint256.MustFromBig(params.TokenAmountOut.Amount)
	newTotalAmountIn := s.TotalAmountsIn[params.TokenAmountIn.Token]

	newBalanceIn, err := BNum.BAdd(inRecord.Balance, amountIn)
	if err != nil {
		logger.
			WithFields(logger.Fields{"poolAddress": s.GetAddress(), "err": err}).
			Warn("failed to update balance")
		return
	}

	newBalanceOut, err := BNum.BSub(outRecord.Balance, amountOut)
	if err != nil {
		logger.
			WithFields(logger.Fields{"poolAddress": s.GetAddress(), "err": err}).
			Warn("failed to update balance")
		return
	}

	newTotalAmountIn, err = BNum.BAdd(newTotalAmountIn, amountIn)
	if err != nil {
		logger.
			WithFields(logger.Fields{"poolAddress": s.GetAddress(), "err": err}).
			Warn("failed to update total amount in")
		return
	}

	inRecord.Balance = newBalanceIn
	outRecord.Balance = newBalanceOut

	s.Records[params.TokenAmountIn.Token] = inRecord
	s.Records[params.TokenAmountOut.Token] = outRecord
	s.TotalAmountsIn[params.TokenAmountIn.Token] = newTotalAmountIn
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		BlockNumber: s.Pool.Info.BlockNumber,
	}
}

// https://github.com/balancer/balancer-core/blob/f4ed5d65362a8d6cec21662fb6eae233b0babc1f/contracts/BPool.sol#L423
// NOTE:
// - ignore minAmountOut and maxPrice because they are not necessary for our simulation
// - this implementation does not cover MAX_IN_RATIO validation when paths are merged
func (s *PoolSimulator) swapExactAmountIn(
	tokenIn string,
	tokenAmountIn *uint256.Int,
	tokenOut string,
	_ *uint256.Int, // minAmountOut
	_ *uint256.Int, // maxPrice
) (*uint256.Int, *uint256.Int, error) {
	if !s.Records[tokenIn].Bound {
		return nil, nil, ErrNotBound
	}

	if !s.Records[tokenOut].Bound {
		return nil, nil, ErrNotBound
	}

	if !s.PublicSwap {
		return nil, nil, ErrSwapNotPublic
	}

	if err := s.validateAmountIn(tokenIn, tokenAmountIn); err != nil {
		return nil, nil, err
	}

	inRecord, outRecord := s.Records[tokenIn], s.Records[tokenOut]
	spotPriceBefore, err := BMath.CalcSpotPrice(
		inRecord.Balance,
		inRecord.Denorm,
		outRecord.Balance,
		outRecord.Denorm,
		s.SwapFee,
	)
	if err != nil {
		return nil, nil, err
	}

	//if spotPriceBefore.Cmp(maxPrice) > 0 {
	//	return nil, nil, ErrBadLimitPrice
	//}

	tokenAmountOut, err := BMath.CalcOutGivenIn(
		inRecord.Balance,
		inRecord.Denorm,
		outRecord.Balance,
		outRecord.Denorm,
		tokenAmountIn,
		s.SwapFee,
	)
	if err != nil {
		return nil, nil, err
	}

	//if tokenAmountOut.Cmp(minAmountOut) < 0 {
	//	return nil, nil, ErrLimitOut
	//}

	inRecord.Balance, err = BNum.BAdd(inRecord.Balance, tokenAmountIn)
	if err != nil {
		return nil, nil, err
	}

	outRecord.Balance, err = BNum.BSub(outRecord.Balance, tokenAmountOut)
	if err != nil {
		return nil, nil, err
	}

	spotPriceAfter, err := BMath.CalcSpotPrice(
		inRecord.Balance,
		inRecord.Denorm,
		outRecord.Balance,
		outRecord.Denorm,
		s.SwapFee,
	)
	if err != nil {
		return nil, nil, err
	}

	if spotPriceAfter.Lt(spotPriceBefore) {
		return nil, nil, ErrMathApprox
	}

	//if spotPriceAfter.Cmp(maxPrice) > 0 {
	//	return nil, nil, ErrLimitPrice
	//}

	bDivTokenAmountInAndOut, err := BNum.BDiv(tokenAmountIn, tokenAmountOut)
	if err != nil {
		return nil, nil, err
	}

	if spotPriceBefore.Gt(bDivTokenAmountInAndOut) {
		return nil, nil, ErrMathApprox
	}

	return tokenAmountOut, spotPriceAfter, nil
}

func (s *PoolSimulator) validateAmountIn(tokenIn string, amountIn *uint256.Int) error {
	bMulBalanceInAndMaxIn, err := BNum.BMul(s.Records[tokenIn].Balance, BConst.MAX_IN_RATIO)
	if err != nil {
		return err
	}

	if amountIn.Gt(bMulBalanceInAndMaxIn) {
		return ErrMaxInRatio
	}

	bAddTotalAmountInAndAmountIn, err := BNum.BAdd(s.TotalAmountsIn[tokenIn], amountIn)
	if err != nil {
		return err
	}

	if bAddTotalAmountInAndAmountIn.Gt(s.MaxTotalAmountsIn[tokenIn]) {
		return ErrMaxTotalInRatio
	}

	return nil
}
