package etherfivampire

import (
	"math/big"
	"time"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/plain"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/etherfi/common"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	PoolExtra
	curveStETHToETHSimulator *plain.PoolSimulator
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra PoolExtra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	curveStETHToETHSimulator, err := plain.NewPoolSimulator(entity.Pool{
		Address:  curveStETHToETHPool,
		Reserves: extra.CurveStETHToETH.Reserves,
		Tokens: []*entity.PoolToken{
			{Address: common.WETH, Decimals: 18, Swappable: true},
			{Address: common.STETH, Decimals: 18, Swappable: true},
		},
		Extra:       extra.CurveStETHToETH.Extra,
		StaticExtra: extra.CurveStETHToETH.StaticExtra,
	})
	if err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Info: pool.PoolInfo{
			Address:     entityPool.Address,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      lo.Map(entityPool.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves:    lo.Map(entityPool.Reserves, func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		},
		PoolExtra:                extra,
		curveStETHToETHSimulator: curveStETHToETHSimulator,
	}, nil
}

func (s *PoolSimulator) CanSwapTo(token string) []string {
	if token == common.EETH || token == common.WEETH {
		return []string{common.STETH, common.WSTETH}
	}

	return nil
}

func (s *PoolSimulator) CanSwapFrom(token string) []string {
	if token == common.STETH || token == common.WSTETH {
		return []string{common.EETH, common.WEETH}
	}

	return nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	gasUsed := int64(0)
	amountIn := new(big.Int).Set(param.TokenAmountIn.Amount)

	if param.TokenAmountIn.Token == common.WSTETH {
		amountIn = s.wstETHUnwrap(amountIn)
		gasUsed += wstETHUnwrapGas
	}

	share, dx, err := s.vampireDepositWithERC20StETH(amountIn)
	if err != nil {
		return nil, err
	}

	if param.TokenAmountIn.Token == common.STETH {
		gasUsed += stETHDepositWithERC20Gas
	} else {
		gasUsed += wstETHDepositWithERC20Gas
	}

	// The executor reads amountForShare(share) after the deposit, so use post-deposit totals.
	var pooledEther, totalShares big.Int
	pooledEther.Add(s.LiquidityPool.TotalPooledEther, dx)
	totalShares.Add(s.EETH.TotalShares, share)
	amountOut := amountForShare(share, &pooledEther, &totalShares)

	if param.TokenOut == common.WEETH {
		// weETH.wrap mints LiquidityPool.sharesForAmount(eETH amount).
		amountOut = sharesForAmount(amountOut, &pooledEther, &totalShares)
		gasUsed += wrapWeETHGas
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  param.TokenOut,
			Amount: amountOut,
		},
		Gas: gasUsed,
		Fee: &pool.TokenAmount{
			Token:  param.TokenAmountIn.Token,
			Amount: bignumber.ZeroBI,
		},
		SwapInfo: SwapInfo{dx: dx, share: share},
	}, nil
}

func (s *PoolSimulator) UpdateBalance(param pool.UpdateBalanceParams) {
	swapInfo := param.SwapInfo.(SwapInfo)
	s.StETHTokenInfo.TotalDepositedThisPeriod.Add(s.StETHTokenInfo.TotalDepositedThisPeriod, swapInfo.dx)
	s.StETHTokenInfo.TotalDeposited.Add(s.StETHTokenInfo.TotalDeposited, swapInfo.dx)
	s.LiquidityPool.TotalPooledEther.Add(s.LiquidityPool.TotalPooledEther, swapInfo.dx)
	s.EETH.TotalShares.Add(s.EETH.TotalShares, swapInfo.share)
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return pool.MetaInfo{BlockNumber: s.Info.BlockNumber}
}

func (s *PoolSimulator) wstETHUnwrap(amountIn *big.Int) *big.Int {
	// Mutate the amountIn parameter.
	return amountIn.
		Mul(amountIn, s.StETH.TotalPooledEther).
		Div(amountIn, s.StETH.TotalShares)
}

func (s *PoolSimulator) vampireDepositWithERC20StETH(amountIn *big.Int) (*big.Int, *big.Int, error) {
	// Step 1: vampire.quoteByDiscountedValue
	// Assume with StETH, `isWhitelisted` is always true & `isL2Eth` is always false.

	// vampire.quoteByMarketValue
	var amount big.Int
	amount.Set(amountIn)
	if s.Vampire.QuoteStEthWithCurve {
		quoteWithCurve, _, _ := s.curveStETHToETHSimulator.GetDy(1, 0, amountIn, nil)
		if quoteWithCurve.Cmp(&amount) < 0 {
			amount.Set(quoteWithCurve)
		}
	}

	// We only need to apply `discountInBasisPoints`.
	var dx big.Int
	dx.
		Sub(bignumber.BasisPoint, s.StETHTokenInfo.DiscountInBasisPoints).
		Mul(&dx, &amount).
		Div(&dx, bignumber.BasisPoint)

	// Step 2: check vampire.isDepositCapReached
	info := s.StETHTokenInfo
	var totalDepositedThisPeriod big.Int
	totalDepositedThisPeriod.Set(info.TotalDepositedThisPeriod)
	if time.Now().Unix() >= int64(info.TimeBoundCapClockStartTime)+int64(s.Vampire.TimeBoundCapRefreshInterval) {
		totalDepositedThisPeriod.SetUint64(0)
	}

	var timeBoundCap, totalCap, tmp big.Int
	timeBoundCap.Mul(big.NewInt(int64(info.TimeBoundCapInEther)), bignumber.BONE)
	totalCap.Mul(big.NewInt(int64(info.TotalCapInEther)), bignumber.BONE)

	if tmp.Add(&totalDepositedThisPeriod, &dx).Cmp(&timeBoundCap) > 0 ||
		tmp.Add(info.TotalDeposited, &dx).Cmp(&totalCap) > 0 {
		return nil, nil, ErrDepositCapReached
	}

	// Step 3: liquidityPool.depositToRecipient
	var eEthShare big.Int
	eEthShare.
		Mul(&dx, s.EETH.TotalShares).
		Div(&eEthShare, s.LiquidityPool.TotalPooledEther)

	if dx.Cmp(bignumber.MaxUint128) > 0 || dx.Sign() == 0 || eEthShare.Sign() == 0 {
		return nil, nil, ErrInvalidAmount
	}

	return &eEthShare, &dx, nil
}

// amountForShare mirrors LiquidityPool.amountForShare.
func amountForShare(share, pooledEther, totalShares *big.Int) *big.Int {
	if totalShares.Sign() == 0 {
		return new(big.Int)
	}
	return bignumber.MulDivDown(new(big.Int), share, pooledEther, totalShares)
}

// sharesForAmount mirrors LiquidityPool.sharesForAmount.
func sharesForAmount(amount, pooledEther, totalShares *big.Int) *big.Int {
	if pooledEther.Sign() == 0 {
		return new(big.Int)
	}
	return bignumber.MulDivDown(new(big.Int), amount, totalShares, pooledEther)
}
