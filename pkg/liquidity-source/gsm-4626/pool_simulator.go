package gsm4626

import (
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
)

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

type PoolSimulator struct {
	pool.Pool
	Extra
	StaticExtra
	underlyingAssetUnits *uint256.Int
}

func NewPoolSimulator(p entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     p.Address,
			Exchange:    p.Exchange,
			Type:        p.Type,
			Tokens:      lo.Map(p.Tokens, func(t *entity.PoolToken, _ int) string { return t.Address }),
			Reserves:    lo.Map(p.Reserves, func(r string, _ int) *big.Int { return bignum.NewBig(r) }),
			BlockNumber: p.BlockNumber,
		}},
		Extra:                extra,
		StaticExtra:          staticExtra,
		underlyingAssetUnits: u256.TenPow(p.Tokens[1].Decimals),
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if !s.CanSwap {
		return nil, ErrCannotSwap
	}

	tokenIn, tokenOut := params.TokenAmountIn.Token, params.TokenOut
	amountIn := uint256.MustFromBig(params.TokenAmountIn.Amount)

	var (
		amountOut *uint256.Int
		fee       *uint256.Int
	)

	isBuy := strings.EqualFold(tokenIn, s.Info.Tokens[0])
	if isBuy {
		amountOut, fee = s.getAssetAmountForBuyAsset(amountIn)

		if amountOut.Sign() <= 0 {
			return nil, ErrInvalidAmount
		}
		if s.CurrentExposure.Lt(amountOut) {
			return nil, ErrInsufficientAvailableExogenousAssetLiquidity
		}
	} else {
		amountOut, fee = s.getGhoAmountForSellAsset(amountIn)

		if new(uint256.Int).Add(s.CurrentExposure, amountIn).Gt(s.ExposureCap) {
			return nil, ErrExogenousAssetExposureTooHigh
		}
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  s.Info.Tokens[0],
			Amount: fee.ToBig(),
		},
		Gas:      lo.Ternary(isBuy, getAssetAmountForBuyAssetGas+buyAssetGas, sellAssetGas),
		SwapInfo: &SwapInfo{isBuy},
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swapInfo := params.SwapInfo.(*SwapInfo)
	if swapInfo.IsBuy {
		s.CurrentExposure = new(uint256.Int).Sub(s.CurrentExposure, uint256.MustFromBig(params.TokenAmountOut.Amount))
	} else {
		s.CurrentExposure = new(uint256.Int).Add(s.CurrentExposure, uint256.MustFromBig(params.TokenAmountIn.Amount))
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.CurrentExposure = new(uint256.Int).Set(s.CurrentExposure)

	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return Meta{
		BlockNumber: s.Info.BlockNumber,
	}
}

func (s *PoolSimulator) getGhoAmountForSellAsset(assetAmount *uint256.Int) (*uint256.Int, *uint256.Int) {
	grossAmount := s.getAssetPriceInGho(assetAmount, false)

	// getSellFee
	fee := new(uint256.Int)
	u256.MulDivUp(fee, grossAmount, s.SellFee, percentageFactor)

	ghoBought := new(uint256.Int).Sub(grossAmount, fee)
	finalGrossAmount := s.getGrossAmountFromTotalSold(ghoBought)
	finalFee := new(uint256.Int).Sub(finalGrossAmount, ghoBought)

	return finalGrossAmount.Sub(finalGrossAmount, finalFee), finalFee
}

func (s *PoolSimulator) getGrossAmountFromTotalSold(totalAmount *uint256.Int) *uint256.Int {
	if s.SellFee.Sign() == 0 {
		return totalAmount.Clone()
	}

	return u256.MulDivUp(new(uint256.Int), totalAmount, percentageFactor, new(uint256.Int).Sub(percentageFactor, s.SellFee))
}

func (s *PoolSimulator) getAssetPriceInGho(assetAmount *uint256.Int, roundUp bool) *uint256.Int {
	var temp uint256.Int
	vaultAssets := lo.Ternary(roundUp, u256.MulDivUp, u256.MulDivDown)(&temp, assetAmount, s.Rate, ray)
	vaultAssets = lo.Ternary(roundUp, u256.MulDivUp, u256.MulDivDown)(&temp, vaultAssets, s.PriceRatio, s.underlyingAssetUnits)

	return vaultAssets
}

func (s *PoolSimulator) getAssetAmountForBuyAsset(maxGhoAmount *uint256.Int) (*uint256.Int, *uint256.Int) {
	grossAmount := s.getGrossAmountFromTotalBought(maxGhoAmount)
	assetAmount := s.getGhoPriceInAsset(grossAmount, false)
	finalGrossAmount := s.getAssetPriceInGho(assetAmount, true)

	// getBuyFee
	finalFee := new(uint256.Int)
	u256.MulDivUp(finalFee, finalGrossAmount, s.BuyFee, percentageFactor)

	return assetAmount, finalFee
}

func (s *PoolSimulator) getGrossAmountFromTotalBought(totalAmount *uint256.Int) *uint256.Int {
	if s.BuyFee.Sign() == 0 {
		return totalAmount.Clone()
	}

	return u256.MulDivDown(new(uint256.Int), totalAmount, percentageFactor, new(uint256.Int).Add(percentageFactor, s.BuyFee))
}

func (s *PoolSimulator) getGhoPriceInAsset(ghoAmount *uint256.Int, roundUp bool) *uint256.Int {
	var temp uint256.Int
	vaultAssets := lo.Ternary(roundUp, u256.MulDivUp, u256.MulDivDown)(&temp, ghoAmount, s.underlyingAssetUnits, s.PriceRatio)
	vaultAssets = lo.Ternary(roundUp, u256.MulDivUp, u256.MulDivDown)(&temp, vaultAssets, ray, s.Rate)

	return vaultAssets
}
