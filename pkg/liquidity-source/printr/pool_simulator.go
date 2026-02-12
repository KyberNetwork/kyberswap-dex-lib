package printr

import (
	"math/big"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	// Immutable curve parameters (from StaticExtra)
	printrAddr     string
	basePair       string
	totalCurves    uint16
	maxTokenSupply *uint256.Int
	virtualReserve *uint256.Int

	// Mutable state (from Extra)
	reserve             *uint256.Int
	completionThreshold *uint256.Int
	tradingFee          uint16
	paused              bool
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

	maxTokenSupply, err := uint256.FromDecimal(staticExtra.MaxTokenSupply)
	if err != nil {
		return nil, err
	}

	virtualReserve, err := uint256.FromDecimal(staticExtra.VirtualReserve)
	if err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     ep.Address,
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      lo.Map(ep.Tokens, func(item *entity.PoolToken, _ int) string { return item.Address }),
			Reserves:    lo.Map(ep.Reserves, func(item string, _ int) *big.Int { return bignum.NewBig(item) }),
			BlockNumber: ep.BlockNumber,
		}},
		printrAddr:          staticExtra.PrintrAddr,
		basePair:            staticExtra.BasePair,
		totalCurves:         staticExtra.TotalCurves,
		maxTokenSupply:      maxTokenSupply,
		virtualReserve:      virtualReserve,
		reserve:             extra.Reserve,
		completionThreshold: extra.CompletionThreshold,
		tradingFee:          extra.TradingFee,
		paused:              extra.Paused,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := params.TokenAmountIn, params.TokenOut

	indexIn, indexOut := s.GetTokenIndex(tokenAmountIn.Token), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	if s.paused {
		return nil, ErrContractPaused
	}

	if s.completionThreshold.IsZero() {
		return nil, ErrTokenGraduated
	}

	amountIn := uint256.MustFromBig(tokenAmountIn.Amount)
	if amountIn.IsZero() {
		return nil, ErrZeroAmount
	}

	isBuy := indexIn == 0

	var (
		amountOut    *uint256.Int
		fee          *uint256.Int
		reserveDelta *uint256.Int
		remainingIn  *uint256.Int
		gas          int64
	)

	if isBuy {
		// Buy: user spends basePair, receives tokens
		// First calculate how many tokens the baseSpend yields
		tokenAmount := CalcBuyTokenAmount(
			s.maxTokenSupply, s.totalCurves,
			s.virtualReserve, s.reserve,
			s.tradingFee, amountIn,
		)

		if tokenAmount.IsZero() {
			return nil, ErrZeroAmount
		}

		// Then get the exact cost for that token amount (for fee accounting)
		result := CalcBuyCost(
			s.maxTokenSupply, s.totalCurves,
			s.virtualReserve, s.reserve,
			s.completionThreshold, s.tradingFee,
			tokenAmount,
		)

		amountOut = result.AvailableAmount
		fee = result.Fee
		// Solidity: reserve += cost - fee (curveCost only, not fee portion)
		reserveDelta = new(uint256.Int).Sub(result.Cost, result.Fee)
		// Remaining = baseSpend - actualCost
		if amountIn.Gt(result.Cost) {
			remainingIn = new(uint256.Int).Sub(amountIn, result.Cost)
		}
		gas = buyGas
	} else {
		// Sell: user spends tokens, receives basePair
		result := CalcSellRefund(
			s.maxTokenSupply, s.totalCurves,
			s.virtualReserve, s.reserve,
			s.tradingFee, amountIn,
		)

		if result.Refund.IsZero() {
			return nil, ErrZeroAmount
		}

		amountOut = result.Refund
		fee = result.Fee
		// Solidity: reserve -= (refund + fee) = curveRefund
		reserveDelta = new(uint256.Int).Add(result.Refund, result.Fee)
		// Remaining = input - actualTokensSold (if capped by issuedSupply)
		if amountIn.Gt(result.TokenAmountIn) {
			remainingIn = new(uint256.Int).Sub(amountIn, result.TokenAmountIn)
		}
		gas = sellGas
	}

	var remainingTokenAmountIn *pool.TokenAmount
	if remainingIn != nil && !remainingIn.IsZero() {
		remainingTokenAmountIn = &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: remainingIn.ToBig(),
		}
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut:         &pool.TokenAmount{Token: tokenOut, Amount: amountOut.ToBig()},
		RemainingTokenAmountIn: remainingTokenAmountIn,
		Fee:                    &pool.TokenAmount{Token: s.Info.Tokens[0], Amount: fee.ToBig()},
		Gas:                    gas,
		SwapInfo:               &SwapInfo{IsBuy: isBuy, reserveDelta: reserveDelta},
	}, nil
}

func (s *PoolSimulator) CalcAmountIn(params pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	tokenAmountOut, tokenIn := params.TokenAmountOut, params.TokenIn

	indexIn, indexOut := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	if s.paused {
		return nil, ErrContractPaused
	}

	if s.completionThreshold.IsZero() {
		return nil, ErrTokenGraduated
	}

	amountOut := uint256.MustFromBig(tokenAmountOut.Amount)
	if amountOut.IsZero() {
		return nil, ErrZeroAmount
	}

	isBuy := indexIn == 0

	var (
		amountIn     *uint256.Int
		fee          *uint256.Int
		reserveDelta *uint256.Int
		gas          int64
	)

	if isBuy {
		// Buy exact tokens: calculate cost directly
		result := CalcBuyCost(
			s.maxTokenSupply, s.totalCurves,
			s.virtualReserve, s.reserve,
			s.completionThreshold, s.tradingFee,
			amountOut,
		)

		amountIn = result.Cost
		fee = result.Fee
		reserveDelta = new(uint256.Int).Sub(result.Cost, result.Fee)
		gas = buyGas
	} else {
		// Sell for exact base output: binary search for the token amount.
		amountIn, fee, reserveDelta = calcSellAmountIn(
			s.maxTokenSupply, s.totalCurves,
			s.virtualReserve, s.reserve,
			s.tradingFee, amountOut,
		)

		if amountIn == nil {
			return nil, ErrInsufficientReserves
		}

		gas = sellGas
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: tokenIn, Amount: amountIn.ToBig()},
		Fee:           &pool.TokenAmount{Token: s.Info.Tokens[0], Amount: fee.ToBig()},
		Gas:           gas,
		SwapInfo:      &SwapInfo{IsBuy: isBuy, reserveDelta: reserveDelta},
	}, nil
}

// calcSellAmountIn binary searches for the token amount that yields at least targetRefund.
func calcSellAmountIn(
	maxTokenSupply *uint256.Int,
	totalCurves uint16,
	virtualReserve *uint256.Int,
	reserve *uint256.Int,
	tradingFee uint16,
	targetRefund *uint256.Int,
) (*uint256.Int, *uint256.Int, *uint256.Int) {
	if totalCurves == 0 {
		return nil, nil, nil
	}

	// Upper bound: all issued supply
	initialTokenReserve := new(uint256.Int).Div(maxTokenSupply, uint256.NewInt(uint64(totalCurves)))
	curveConstant := new(uint256.Int).Mul(virtualReserve, initialTokenReserve)
	vPlusR := new(uint256.Int).Add(virtualReserve, reserve)
	if vPlusR.IsZero() {
		return nil, nil, nil
	}
	tokenReserve := new(uint256.Int).Div(curveConstant, vPlusR)
	issuedSupply := new(uint256.Int).Sub(initialTokenReserve, tokenReserve)

	if issuedSupply.IsZero() {
		return nil, nil, nil
	}

	var low, high, mid, bestAmount, bestFee, bestDelta uint256.Int
	high.Set(issuedSupply)

	for i := 0; i < 100 && !low.Gt(&high); i++ {
		mid.Add(&low, &high).Div(&mid, uint256.NewInt(2))

		result := CalcSellRefund(
			maxTokenSupply, totalCurves,
			virtualReserve, reserve,
			tradingFee, &mid,
		)

		if result.Refund.Lt(targetRefund) {
			low.AddUint64(&mid, 1)
		} else {
			bestAmount.Set(result.TokenAmountIn)
			bestFee.Set(result.Fee)
			bestDelta.Add(result.Refund, result.Fee)
			high.SubUint64(&mid, 1)
		}
	}

	if bestAmount.IsZero() {
		return nil, nil, nil
	}

	return new(uint256.Int).Set(&bestAmount), new(uint256.Int).Set(&bestFee), new(uint256.Int).Set(&bestDelta)
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swapInfo := params.SwapInfo.(*SwapInfo)

	if swapInfo.IsBuy {
		// Buy: reserve += curveCost (cost minus fee)
		s.reserve.Add(s.reserve, swapInfo.reserveDelta)
	} else {
		// Sell: reserve -= (refund + fee)
		s.reserve.Sub(s.reserve, swapInfo.reserveDelta)
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.reserve = new(uint256.Int).Set(s.reserve)
	cloned.completionThreshold = new(uint256.Int).Set(s.completionThreshold)
	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) any {
	return MetaInfo{
		BlockNumber:     s.Info.BlockNumber,
		ApprovalAddress: s.GetApprovalAddress(tokenIn, tokenOut),
	}
}

// GetApprovalAddress returns the Printr contract address when selling tokens,
// so the executor knows which contract to approve for ERC20 token transfers.
// When buying (basePair → token), no approval of Printr is needed (user sends ETH/basePair directly).
func (s *PoolSimulator) GetApprovalAddress(tokenIn, _ string) string {
	if s.GetTokenIndex(tokenIn) == 0 {
		// Buying: basePair is input, no approval needed on Printr
		return ""
	}
	// Selling: token is input, must approve Printr to transfer tokens
	return s.printrAddr
}
