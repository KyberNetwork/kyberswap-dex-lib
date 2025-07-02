package arenabc

import (
	"math/big"
	"strings"

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
	return lo.Ternary(s.isBuy(tokenIn), "", s.tokenManager)
}

func (s *PoolSimulator) isBuy(tokenIn string) bool {
	return strings.EqualFold(tokenIn, s.Info.Tokens[0])
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
		if s.isLpTokenThresholdReached() {
			s.lpDeployed = true
		}
	}
}

func (s *PoolSimulator) sell(amount *uint256.Int) (*uint256.Int, *SwapInfo, int64, error) {
	scaledAmount, remaining := new(uint256.Int).DivMod(amount, granularityScaler, new(uint256.Int))
	if scaledAmount.IsZero() {
		return nil, nil, 0, ErrZeroSwap
	}

	reward, err := s.calculateReward(scaledAmount)
	if err != nil {
		return nil, nil, 0, err
	}

	fee := getFee(reward, s.protocolFeeBasisPoint, s.creatorFeeBasisPoints, s.referralFeeBasisPoint)

	amountOut := new(uint256.Int).Sub(reward, fee)

	var currentNativeBalance, currentTotalSupply uint256.Int
	if _, underflow := currentNativeBalance.SubOverflow(s.nativeBalance, reward); underflow {
		return nil, nil, 0, ErrNativeBalanceOverflowOrUnderflow
	} else if _, underflow = currentTotalSupply.SubOverflow(s.totalSupply, amount); underflow {
		return nil, nil, 0, ErrTotalSupplyOverflowOrUnderflow
	}

	return amountOut, &SwapInfo{
		TokenManager:  s.tokenManager,
		IsBuy:         false,
		TokenId:       s.tokenId,
		SwapAmount:    new(uint256.Int).Sub(amount, remaining),
		fee:           fee,
		totalSupply:   &currentTotalSupply,
		nativeBalance: &currentNativeBalance,
		remainingTokenIn: &pool.TokenAmount{
			Token:  s.Info.Tokens[1],
			Amount: remaining.ToBig(),
		},
	}, sellGas, nil
}

func (s *PoolSimulator) calculateReward(scaledAmount *uint256.Int) (*uint256.Int, error) {
	scaledTotalSupply := new(uint256.Int).Div(s.totalSupply, granularityScaler)

	if scaledTotalSupply.IsZero() {
		return nil, ErrZeroSwap
	}

	if scaledAmount.Gt(scaledTotalSupply) {
		return nil, ErrUnderflow
	}

	return integralFloor(scaledTotalSupply, new(uint256.Int).Sub(scaledTotalSupply, scaledAmount), s.a, s.b, s.curveScaler), nil
}

func (s *PoolSimulator) buyAndCreateLpIfPossible(nativeAmount *uint256.Int) (*uint256.Int, *SwapInfo, int64, error) {
	tokenAmount := s.calculatePurchaseAmountParametric(nativeAmount)

	scaledTokenAmount := new(uint256.Int).Div(tokenAmount, granularityScaler)
	if scaledTokenAmount.IsZero() {
		return nil, nil, 0, ErrZeroSwap
	}

	cost, _ := s.calculateCost(scaledTokenAmount)
	fee := getFee(cost, s.protocolFeeBasisPoint, s.creatorFeeBasisPoints, s.referralFeeBasisPoint)
	totalCost := new(uint256.Int).Add(cost, fee)
	gas := buyGas

	if s.isLpTokenThresholdReached() {
		if !s.canDeployLp {
			return nil, nil, 0, ErrLpDeployNotAllowedRightNow
		}
		gas += createLpGas
	}

	var currentNativeBalance, currentTotalSupply uint256.Int
	if _, overflow := currentNativeBalance.AddOverflow(s.nativeBalance, totalCost); overflow {
		return nil, nil, 0, ErrNativeBalanceOverflowOrUnderflow
	} else if _, overflow = currentTotalSupply.AddOverflow(s.totalSupply, tokenAmount); overflow {
		return nil, nil, 0, ErrTotalSupplyOverflowOrUnderflow
	}

	return tokenAmount, &SwapInfo{
		TokenManager: s.tokenManager,
		IsBuy:        true,
		TokenId:      s.tokenId,
		SwapAmount:   nativeAmount,
		fee:          fee,
		remainingTokenIn: &pool.TokenAmount{
			Token:  s.Info.Tokens[0],
			Amount: new(uint256.Int).Sub(nativeAmount, totalCost).ToBig(),
		},
		totalSupply:   &currentTotalSupply,
		nativeBalance: &currentNativeBalance,
	}, int64(gas), nil
}

func (s *PoolSimulator) getMaxTokensForSale() *uint256.Int {
	var maxSupplyForSale, buyLimit uint256.Int
	maxSupplyForSale.MulDivOverflow(s.allowedTotalSupply, s.salePercentage, U100)

	if !maxSupplyForSale.Lt(s.totalSupply) {
		buyLimit.Sub(&maxSupplyForSale, s.totalSupply)
	}

	return &buyLimit
}

func (s *PoolSimulator) getMaxNativeTokensOut() *uint256.Int {
	maxNativeOut := integralFloor(
		new(uint256.Int).Div(s.totalSupply, granularityScaler),
		new(uint256.Int),
		s.a, s.b, s.curveScaler,
	)
	fee := getFee(
		maxNativeOut,
		s.protocolFeeBasisPoint,
		s.creatorFeeBasisPoints,
		s.referralFeeBasisPoint,
	)

	return new(uint256.Int).Sub(maxNativeOut, fee)
}

func (s *PoolSimulator) calculateCost(amount *uint256.Int) (*uint256.Int, error) {
	totalSupply := new(uint256.Int).Div(s.totalSupply, granularityScaler)

	return integralCeil(new(uint256.Int).Add(amount, totalSupply), totalSupply, s.a, s.b, s.curveScaler), nil
}

func (s *PoolSimulator) isLpTokenThresholdReached() bool {
	lhs := new(uint256.Int).Mul(s.allowedTotalSupply, s.salePercentage)
	rhs := new(uint256.Int).Mul(s.totalSupply, U100)

	return lhs.Eq(rhs)
}

func (s *PoolSimulator) calculatePurchaseAmountParametric(nativeAmount *uint256.Int) *uint256.Int {
	maxTokensForSaleInWei := s.getMaxTokensForSale()

	cost := s.calculateCostScaledParametricWithFees(maxTokensForSaleInWei, s.totalSupply)

	if nativeAmount.Gt(cost) {
		return maxTokensForSaleInWei
	}

	var low, high, mid, amountInWei uint256.Int
	high.Div(maxTokensForSaleInWei, granularityScaler)

	maxIterations := 100

	for low.Lt(&high) && maxIterations > 0 {
		mid.Add(&low, &high).Add(&mid, u256.U1).Div(&mid, u256.U2)

		cost.Set(s.calculateCostScaledParametricWithFees(amountInWei.Mul(&mid, granularityScaler), s.totalSupply))
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

func (s *PoolSimulator) calculateCostScaledParametricWithFees(amountInWei, supplyInWei *uint256.Int) *uint256.Int {
	rawCosts := s.calculateCostScaledParametric(amountInWei, supplyInWei)
	fee := getFee(rawCosts, s.protocolFeeBasisPoint, s.creatorFeeBasisPoints, s.referralFeeBasisPoint)

	return rawCosts.Add(rawCosts, fee)
}

func (s *PoolSimulator) calculateCostScaledParametric(amountInWei, supplyInWei *uint256.Int) *uint256.Int {
	amountInTokens := new(uint256.Int).Div(amountInWei, granularityScaler)
	supplyInTokens := new(uint256.Int).Div(supplyInWei, granularityScaler)

	upperBound := new(uint256.Int).Add(supplyInTokens, amountInTokens)
	lowerBound := supplyInTokens

	return integralCeil(upperBound, lowerBound, s.a, s.b, s.curveScaler)
}
