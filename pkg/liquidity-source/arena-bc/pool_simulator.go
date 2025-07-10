package arenabc

import (
	"math"
	"math/big"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	pool.Pool

	chainId      valueobject.ChainID
	tokenId      *big.Int
	tokenManager string

	isPaused              bool
	canDeployLp           bool
	allowedTotalSupply    *uint256.Int
	protocolFeeBasisPoint *uint256.Int
	referralFeeBasisPoint *uint256.Int

	a                     *uint256.Int
	b                     *uint256.Int
	lpDeployed            bool
	curveScaler           *uint256.Int
	salePercentage        *uint256.Int
	totalSupply           *uint256.Int
	nativeBalance         *uint256.Int
	creatorFeeBasisPoints *uint256.Int
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
			Tokens:      lo.Map(ep.Tokens, func(item *entity.PoolToken, _ int) string { return item.Address }),
			Reserves:    lo.Map(ep.Reserves, func(item string, _ int) *big.Int { return bignum.NewBig(item) }),
			BlockNumber: ep.BlockNumber,
		}},
		chainId:      staticExtra.ChainId,
		tokenId:      staticExtra.TokenId,
		tokenManager: staticExtra.TokenManager,

		isPaused:              extra.IsPaused,
		canDeployLp:           extra.CanDeployLp,
		allowedTotalSupply:    extra.AllowedTokenSupply,
		protocolFeeBasisPoint: uint256.NewInt(uint64(extra.ProtocolFeeBasisPoint)),
		referralFeeBasisPoint: uint256.NewInt(uint64(extra.ReferralFeeBasisPoint)),

		totalSupply:           extra.TokenSupply,
		curveScaler:           extra.TokenParams.CurveScaler,
		salePercentage:        uint256.NewInt(uint64(extra.TokenParams.SalePercentage)),
		a:                     uint256.NewInt(uint64(extra.TokenParams.A)),
		b:                     uint256.NewInt(uint64(extra.TokenParams.B)),
		lpDeployed:            extra.TokenParams.LpDeployed,
		nativeBalance:         extra.TokenBalance,
		creatorFeeBasisPoints: uint256.NewInt(uint64(extra.TokenParams.CreatorFeeBasisPoints)),
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := params.TokenAmountIn, params.TokenOut

	indexIn, indexOut := s.GetTokenIndex(tokenAmountIn.Token), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	if s.isPaused {
		return nil, ErrPoolPaused
	}

	if s.lpDeployed {
		return nil, ErrLpAlreadyDeployed
	}

	amountIn := uint256.MustFromBig(tokenAmountIn.Amount)
	if amountIn.IsZero() {
		return nil, ErrZeroSwap
	}

	var (
		swapInfo  *SwapInfo
		amountOut *uint256.Int
		gas       int64
		err       error
	)

	isBuy := indexIn == 0
	if isBuy {
		amountOut, swapInfo, gas, err = s.buyAndCreateLpIfPossible(amountIn)
	} else {
		amountOut, swapInfo, gas, err = s.sell(amountIn)
	}
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut:         &pool.TokenAmount{Token: tokenOut, Amount: amountOut.ToBig()},
		RemainingTokenAmountIn: swapInfo.remainingTokenIn,
		Fee:                    &pool.TokenAmount{Token: s.Info.Tokens[0], Amount: swapInfo.fee.ToBig()},
		Gas:                    gas,
		SwapInfo:               swapInfo,
	}, nil
}

func (s *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) any {
	return MetaInfo{
		BlockNumber:     s.Info.BlockNumber,
		ApprovalAddress: s.GetApprovalAddress(tokenIn, tokenOut),
	}
}

func (s *PoolSimulator) GetApprovalAddress(tokenIn, _ string) string {
	return lo.Ternary(s.GetTokenIndex(tokenIn) == 0, "", s.tokenManager)
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.totalSupply = new(uint256.Int).Set(s.totalSupply)
	cloned.nativeBalance = new(uint256.Int).Set(s.nativeBalance)

	return &cloned
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swapInfo := params.SwapInfo.(*SwapInfo)

	s.totalSupply = swapInfo.totalSupply
	s.nativeBalance = swapInfo.nativeBalance

	if swapInfo.IsBuy {
		if s.isLpTokenThresholdReached(s.totalSupply) {
			s.lpDeployed = true
		}
	}
}

func (s *PoolSimulator) sell(tokenAmount *uint256.Int) (*uint256.Int, *SwapInfo, int64, error) {
	scaledTokenAmount, remaining := new(uint256.Int).DivMod(tokenAmount, granularityScaler, new(uint256.Int))
	if scaledTokenAmount.IsZero() {
		return nil, nil, 0, ErrZeroSwap
	}

	reward, err := s.calculateReward(scaledTokenAmount)
	if err != nil {
		return nil, nil, 0, err
	}
	fee := getFee(reward, s.protocolFeeBasisPoint, s.creatorFeeBasisPoints, s.referralFeeBasisPoint)

	var currentNativeBalance, currentTotalSupply uint256.Int
	if _, underflow := currentNativeBalance.SubOverflow(s.nativeBalance, reward); underflow {
		return nil, nil, 0, ErrNativeBalanceOverflowOrUnderflow
	} else if _, underflow = currentTotalSupply.SubOverflow(s.totalSupply, tokenAmount); underflow {
		return nil, nil, 0, ErrTotalSupplyOverflowOrUnderflow
	}

	return new(uint256.Int).Sub(reward, fee), &SwapInfo{
		TokenManager:  s.tokenManager,
		IsBuy:         false,
		TokenId:       s.tokenId,
		SwapAmount:    new(uint256.Int).Sub(tokenAmount, remaining),
		fee:           fee,
		totalSupply:   &currentTotalSupply,
		nativeBalance: &currentNativeBalance,
		remainingTokenIn: &pool.TokenAmount{
			Token:  s.Info.Tokens[1],
			Amount: remaining.ToBig(),
		},
	}, sellGas, nil
}

func (s *PoolSimulator) buyAndCreateLpIfPossible(nativeAmount *uint256.Int) (*uint256.Int, *SwapInfo, int64, error) {
	tokenAmount := s.calculatePurchaseAmount(nativeAmount)

	scaledTokenAmount := new(uint256.Int).Div(tokenAmount, granularityScaler)
	if scaledTokenAmount.IsZero() {
		return nil, nil, 0, ErrZeroSwap
	}

	cost := s.calculateCost(scaledTokenAmount)
	fee := getFee(cost, s.protocolFeeBasisPoint, s.creatorFeeBasisPoints, s.referralFeeBasisPoint)
	cost.Add(cost, fee)

	var currentNativeBalance, currentTotalSupply uint256.Int
	if _, overflow := currentNativeBalance.AddOverflow(s.nativeBalance, cost); overflow {
		return nil, nil, 0, ErrNativeBalanceOverflowOrUnderflow
	} else if _, overflow = currentTotalSupply.AddOverflow(s.totalSupply, tokenAmount); overflow {
		return nil, nil, 0, ErrTotalSupplyOverflowOrUnderflow
	}

	tokenAmountTolerance, _ := new(uint256.Int).MulDivOverflow(tokenAmount, swapAmountTolerancePercentage, U100)
	minTokenAmountOut := new(uint256.Int).Sub(tokenAmount, tokenAmountTolerance)
	maxTokenAmountOut := new(uint256.Int).Add(tokenAmount, tokenAmountTolerance)

	gas := s.estimateCalculateCostWithFeesGas(tokenAmountTolerance.Sub(maxTokenAmountOut, minTokenAmountOut)) + buyGas

	if s.isLpTokenThresholdReached(&currentTotalSupply) {
		if !s.canDeployLp {
			return nil, nil, 0, ErrLpDeployNotAllowedRightNow
		}
		gas += createLpGas
	}

	scaledMaxTokensForSale := getMaxTokensForSale(s.allowedTotalSupply, s.salePercentage)
	scaledMaxTokensForSale.Div(scaledMaxTokensForSale, granularityScaler)

	scaledTotalSupply := new(uint256.Int).Div(s.totalSupply, granularityScaler)

	// sanity check
	if new(uint256.Int).Add(scaledTotalSupply, scaledTokenAmount).Gt(scaledMaxTokensForSale) {
		return nil, nil, 0, ErrSupplyMismatchInBuy
	}

	return tokenAmount, &SwapInfo{
		TokenManager:            s.tokenManager,
		IsBuy:                   true,
		TokenId:                 s.tokenId,
		SwapAmount:              nativeAmount,
		MinScaledTokenAmountOut: minTokenAmountOut.Div(minTokenAmountOut, granularityScaler).Uint64(),
		MaxScaledTokenAmountOut: maxTokenAmountOut.Div(maxTokenAmountOut, granularityScaler).Uint64(),
		fee:                     fee,
		remainingTokenIn: &pool.TokenAmount{
			Token:  s.Info.Tokens[0],
			Amount: new(uint256.Int).Sub(nativeAmount, cost).ToBig(),
		},
		totalSupply:   &currentTotalSupply,
		nativeBalance: &currentNativeBalance,
	}, gas, nil
}

func (s *PoolSimulator) calculateCost(scaledAmount *uint256.Int) *uint256.Int {
	scaledTotalSupply := new(uint256.Int).Div(s.totalSupply, granularityScaler)

	return integralCeil(new(uint256.Int).Add(scaledAmount, scaledTotalSupply), scaledTotalSupply, s.a, s.b, s.curveScaler)
}

func (s *PoolSimulator) isLpTokenThresholdReached(currentTotalSupply *uint256.Int) bool {
	lhs := new(uint256.Int).Mul(s.allowedTotalSupply, s.salePercentage)
	rhs := new(uint256.Int).Mul(currentTotalSupply, U100)

	return lhs.Eq(rhs)
}

func (s *PoolSimulator) calculatePurchaseAmount(nativeAmount *uint256.Int) *uint256.Int {
	buyLimit := getBuyLimit(s.totalSupply, s.allowedTotalSupply, s.salePercentage)

	cost := s.calculateCostWithFees(buyLimit)

	if nativeAmount.Gt(cost) {
		return buyLimit
	}

	var low, high, mid, amountInWei uint256.Int
	high.Div(buyLimit, granularityScaler)

	maxIterations := 100

	for low.Lt(&high) && maxIterations > 0 {
		mid.Add(&low, &high).Add(&mid, u256.U1).Div(&mid, u256.U2)

		cost.Set(s.calculateCostWithFees(amountInWei.Mul(&mid, granularityScaler)))
		if cost.Eq(nativeAmount) {
			return mid.Mul(&mid, u256.BONE)
		} else if cost.Lt(nativeAmount) {
			low.Set(&mid)
		} else {
			high.Sub(&mid, u256.U1)
		}

		maxIterations--
	}

	return low.Mul(&low, u256.BONE)
}

func (s *PoolSimulator) calculateCostWithFees(amount *uint256.Int) *uint256.Int {
	scaledAmount := new(uint256.Int).Div(amount, granularityScaler)
	scaledTotalSupply := new(uint256.Int).Div(s.totalSupply, granularityScaler)

	rawCosts := integralCeil(new(uint256.Int).Add(scaledTotalSupply, scaledAmount), scaledTotalSupply, s.a, s.b, s.curveScaler)
	fee := getFee(rawCosts, s.protocolFeeBasisPoint, s.creatorFeeBasisPoints, s.referralFeeBasisPoint)

	return rawCosts.Add(rawCosts, fee)
}

func (s *PoolSimulator) estimateCalculateCostWithFeesGas(acceptableTokenAmountDiff *uint256.Int) int64 {
	acceptableTokenAmountDiff.Div(acceptableTokenAmountDiff, granularityScaler)

	if acceptableTokenAmountDiff.IsZero() {
		return calculateCostOverheadGas + calculateCostGas
	}

	// Executor should perform binary search on the range
	// [scaledTokenAmount * (1 - swapAmountTolerance/2), scaledTokenAmount * (1 + swapAmountTolerance/2)]
	return calculateCostOverheadGas + int64(math.Ceil(math.Log2(acceptableTokenAmountDiff.Float64())))*calculateCostGas
}

func (s *PoolSimulator) CalcAmountIn(params pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	tokenAmountOut, tokenIn := params.TokenAmountOut, params.TokenIn

	indexIn, indexOut := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	if s.isPaused {
		return nil, ErrPoolPaused
	}

	if s.lpDeployed {
		return nil, ErrLpAlreadyDeployed
	}

	amountOut := uint256.MustFromBig(tokenAmountOut.Amount)
	if amountOut.IsZero() {
		return nil, ErrZeroSwap
	}

	var (
		swapInfo *SwapInfo
		amountIn *uint256.Int
		gas      int64
		err      error
	)

	isBuy := indexIn == 0
	if isBuy {
		amountIn, swapInfo, gas, err = s.buyAndCreateLpIfPossibleWithAmountOut(amountOut)
	} else {
		amountIn, swapInfo, gas, err = s.sellWithAmountOut(amountOut)
	}
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: tokenIn, Amount: amountIn.ToBig()},
		Fee:           &pool.TokenAmount{Token: s.Info.Tokens[0], Amount: swapInfo.fee.ToBig()},
		Gas:           gas,
		SwapInfo:      swapInfo,
	}, nil
}

func (s *PoolSimulator) buyAndCreateLpIfPossibleWithAmountOut(tokenAmount *uint256.Int) (*uint256.Int, *SwapInfo, int64, error) {
	scaledTokenAmount, remaining := new(uint256.Int).DivMod(tokenAmount, granularityScaler, new(uint256.Int))
	if remaining.Sign() > 0 {
		scaledTokenAmount.Add(scaledTokenAmount, u256.U1)
		tokenAmount.Add(tokenAmount, granularityScaler)
	}

	buyLimit := getBuyLimit(s.totalSupply, s.allowedTotalSupply, s.salePercentage)

	scaledBuyLimit := new(uint256.Int).Div(buyLimit, granularityScaler)
	if scaledTokenAmount.Gt(scaledBuyLimit) {
		return nil, nil, 0, ErrBuyLimitExceeded
	}

	scaledMaxTokensForSale := getMaxTokensForSale(s.allowedTotalSupply, s.salePercentage)
	scaledMaxTokensForSale.Div(scaledMaxTokensForSale, granularityScaler)

	cost := s.calculateCost(scaledTokenAmount)
	fee := getFee(cost, s.protocolFeeBasisPoint, s.creatorFeeBasisPoints, s.referralFeeBasisPoint)
	cost.Add(cost, fee)

	var currentNativeBalance, currentTotalSupply uint256.Int
	if _, overflow := currentNativeBalance.AddOverflow(s.nativeBalance, cost); overflow {
		return nil, nil, 0, ErrNativeBalanceOverflowOrUnderflow
	} else if _, overflow = currentTotalSupply.AddOverflow(s.totalSupply, tokenAmount); overflow {
		return nil, nil, 0, ErrTotalSupplyOverflowOrUnderflow
	}

	tokenAmountTolerance, _ := new(uint256.Int).MulDivOverflow(tokenAmount, swapAmountTolerancePercentage, U100)
	minTokenAmountOut := new(uint256.Int).Sub(tokenAmount, tokenAmountTolerance)
	maxTokenAmountOut := new(uint256.Int).Add(tokenAmount, tokenAmountTolerance)

	gas := s.estimateCalculateCostWithFeesGas(tokenAmountTolerance.Sub(maxTokenAmountOut, minTokenAmountOut)) + buyGas

	if s.isLpTokenThresholdReached(&currentTotalSupply) {
		if !s.canDeployLp {
			return nil, nil, 0, ErrLpDeployNotAllowedRightNow
		}
		gas += createLpGas
	}

	scaledTotalSupply := new(uint256.Int).Div(s.totalSupply, granularityScaler)

	// sanity check
	if new(uint256.Int).Add(scaledTotalSupply, scaledTokenAmount).Gt(scaledMaxTokensForSale) {
		return nil, nil, 0, ErrSupplyMismatchInBuy
	}

	return cost, &SwapInfo{
		TokenManager:            s.tokenManager,
		IsBuy:                   true,
		TokenId:                 s.tokenId,
		SwapAmount:              cost,
		MinScaledTokenAmountOut: minTokenAmountOut.Div(minTokenAmountOut, granularityScaler).Uint64(),
		MaxScaledTokenAmountOut: maxTokenAmountOut.Div(maxTokenAmountOut, granularityScaler).Uint64(),
		fee:                     fee,
		totalSupply:             &currentTotalSupply,
		nativeBalance:           &currentNativeBalance,
	}, gas, nil
}

func (s *PoolSimulator) sellWithAmountOut(nativeAmount *uint256.Int) (*uint256.Int, *SwapInfo, int64, error) {
	tokenAmount, err := s.calculateSellAmount(nativeAmount)
	if err != nil {
		return nil, nil, 0, err
	}

	reward, err := s.calculateReward(new(uint256.Int).Div(tokenAmount, granularityScaler))
	if err != nil {
		return nil, nil, 0, err
	}
	fee := getFee(reward, s.protocolFeeBasisPoint, s.creatorFeeBasisPoints, s.referralFeeBasisPoint)

	var currentNativeBalance, currentTotalSupply uint256.Int
	if _, underflow := currentNativeBalance.SubOverflow(s.nativeBalance, reward); underflow {
		return nil, nil, 0, ErrNativeBalanceOverflowOrUnderflow
	} else if _, underflow = currentTotalSupply.SubOverflow(s.totalSupply, tokenAmount); underflow {
		return nil, nil, 0, ErrTotalSupplyOverflowOrUnderflow
	}

	return tokenAmount, &SwapInfo{
		TokenManager:  s.tokenManager,
		IsBuy:         false,
		TokenId:       s.tokenId,
		SwapAmount:    tokenAmount,
		fee:           fee,
		totalSupply:   &currentTotalSupply,
		nativeBalance: &currentNativeBalance,
	}, sellGas, nil
}

func (s *PoolSimulator) calculateSellAmount(nativeAmount *uint256.Int) (*uint256.Int, error) {
	sellLimit := getSellLimit(s.totalSupply, s.a, s.b, s.curveScaler,
		s.protocolFeeBasisPoint, s.creatorFeeBasisPoints, s.referralFeeBasisPoint)

	if sellLimit.Lt(nativeAmount) {
		return nil, ErrSellLimitExceeded
	}

	var low, high, mid, amount, result uint256.Int
	high.Div(s.totalSupply, granularityScaler)

	maxIterations := 100

	for !low.Gt(&high) && maxIterations > 0 {
		mid.Add(&low, &high).Div(&mid, u256.U2)

		reward, err := s.calculateRewardWithFees(amount.Mul(&mid, granularityScaler))
		if err != nil {
			return nil, err
		}

		if reward.Lt(nativeAmount) {
			low.Add(&mid, u256.U1)
		} else {
			result.Set(&mid)
			high.Sub(&mid, u256.U1)
		}

		maxIterations--
	}

	return result.Mul(&result, u256.BONE), nil
}

func (s *PoolSimulator) calculateRewardWithFees(amount *uint256.Int) (*uint256.Int, error) {
	if amount.IsZero() {
		return new(uint256.Int), nil
	}

	scaledTotalSupply := new(uint256.Int).Div(s.totalSupply, granularityScaler)
	scaledAmount := new(uint256.Int).Div(amount, granularityScaler)

	if scaledTotalSupply.Lt(scaledAmount) {
		return nil, ErrUnderflow
	}

	reward := integralFloor(scaledTotalSupply, new(uint256.Int).Sub(scaledTotalSupply, scaledAmount), s.a, s.b, s.curveScaler)
	fee := getFee(reward, s.protocolFeeBasisPoint, s.creatorFeeBasisPoints, s.referralFeeBasisPoint)

	return reward.Sub(reward, fee), nil
}

func (s *PoolSimulator) calculateReward(scaledTokenAmount *uint256.Int) (*uint256.Int, error) {
	scaledTotalSupply := new(uint256.Int).Div(s.totalSupply, granularityScaler)
	if scaledTotalSupply.IsZero() {
		return nil, ErrZeroSwap
	}

	if scaledTokenAmount.Gt(scaledTotalSupply) {
		return nil, ErrUnderflow
	}

	return integralFloor(scaledTotalSupply, new(uint256.Int).Sub(scaledTotalSupply, scaledTokenAmount), s.a, s.b, s.curveScaler), nil
}
