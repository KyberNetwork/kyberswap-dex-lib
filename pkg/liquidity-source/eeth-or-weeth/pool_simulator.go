package eethorweeth

import (
	"encoding/json"
	"errors"
	"math/big"
	"time"

	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/plain"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ErrDepositCapReached = errors.New("deposit cap reached")
	ErrInvalidAmount     = errors.New("invalid amount")
)

// PoolSimulator only support deposits ETH and get eETH
type PoolSimulator struct {
	poolpkg.Pool
	PoolExtra
	curveStETHToETHSimulator *plain.PoolSimulator
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra PoolExtra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	curveStETHToETHSimulator, err := plain.NewPoolSimulator(entity.Pool{
		Address:  curveStETHToETHPool,
		Reserves: extra.CurveStETHToETH.Reserves,
		Tokens: []*entity.PoolToken{
			{Address: weth, Decimals: 18, Swappable: true},
			{Address: stETH, Decimals: 18, Swappable: true},
		},
		Extra:       extra.CurveStETHToETH.Extra,
		StaticExtra: extra.CurveStETHToETH.StaticExtra,
	})
	if err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: poolpkg.Pool{Info: poolpkg.PoolInfo{
			Address:     entityPool.Address,
			ReserveUsd:  entityPool.ReserveUsd,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      lo.Map(entityPool.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves:    lo.Map(entityPool.Reserves, func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		PoolExtra:                extra,
		curveStETHToETHSimulator: curveStETHToETHSimulator,
	}, nil
}

func (s *PoolSimulator) CanSwapTo(token string) []string {
	if token == eETH || token == weETH {
		return []string{stETH, wstETH}
	}

	return nil
}

func (s *PoolSimulator) CanSwapFrom(token string) []string {
	if token == stETH || token == wstETH {
		return []string{eETH, weETH}
	}

	return nil
}

func (s *PoolSimulator) CalcAmountOut(param poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	amountIn := new(big.Int).Set(param.TokenAmountIn.Amount)

	if param.TokenAmountIn.Token == wstETH {
		amountIn = s.wstETHUnwrap(amountIn)
	}

	amountOut, err := s.vampireDepositWithERC20StETH(amountIn)
	if err != nil {
		return nil, err
	}

	if param.TokenOut == weETH {
		eETHAmount := s.etherFiAmountForShare(amountOut)
		amountOut = s.etherFiSharesForAmount(eETHAmount)
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{
			Token:  param.TokenOut,
			Amount: amountOut,
		},
		Gas: 0, // TODO: calculate gas
		Fee: &poolpkg.TokenAmount{
			Token:  param.TokenAmountIn.Token,
			Amount: bignumber.ZeroBI,
		},
	}, nil
}

func (s *PoolSimulator) UpdateBalance(param poolpkg.UpdateBalanceParams) {
	// TODO: implement this
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) interface{} {
	return nil
}

func (s *PoolSimulator) wstETHUnwrap(amountIn *big.Int) *big.Int {
	// Mutate the amountIn parameter.
	return amountIn.
		Mul(amountIn, s.StETH.TotalPooledEther).
		Div(amountIn, s.StETH.TotalShares)
}

func (s *PoolSimulator) vampireDepositWithERC20StETH(amountIn *big.Int) (*big.Int, error) {
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
		Sub(bignumber.BasisPoint, big.NewInt(int64(s.StETHTokenInfo.DiscountInBasisPoints))).
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
		return nil, ErrDepositCapReached
	}

	// Step 3: liquidityPool.depositToRecipient
	var eEthShare big.Int
	eEthShare.
		Mul(&dx, s.EETH.TotalShares).
		Div(&eEthShare, s.EtherFiPool.TotalPooledEther)
	var uint128Max big.Int
	uint128Max.SetUint64(1).Lsh(&uint128Max, 128).Sub(&uint128Max, bignumber.One)

	if dx.Cmp(&uint128Max) > 0 || dx.Sign() == 0 || eEthShare.Sign() == 0 {
		return nil, ErrInvalidAmount
	}

	return &eEthShare, nil
}

func (s *PoolSimulator) etherFiAmountForShare(share *big.Int) *big.Int {
	if s.EETH.TotalShares.Sign() == 0 {
		return big.NewInt(0)
	}

	return new(big.Int).Mul(share, s.EtherFiPool.TotalPooledEther).Div(share, s.EETH.TotalShares)
}

func (s *PoolSimulator) etherFiSharesForAmount(amount *big.Int) *big.Int {
	if s.EtherFiPool.TotalPooledEther.Sign() == 0 {
		return big.NewInt(0)
	}

	return new(big.Int).Mul(amount, s.EETH.TotalShares).Div(amount, s.EtherFiPool.TotalPooledEther)
}
