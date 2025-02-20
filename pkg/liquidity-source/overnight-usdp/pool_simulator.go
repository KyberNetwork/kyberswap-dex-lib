package overnightusdp

import (
	"math/big"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	isPaused  bool
	buyFee    *big.Int
	redeemFee *big.Int

	usdPlusDecimals int64
	assetDecimals   int64 // USDC

	gas int64
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     entityPool.Address,
			ReserveUsd:  entityPool.ReserveUsd,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      lo.Map(entityPool.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves:    lo.Map(entityPool.Reserves, func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		isPaused:        extra.IsPaused,
		buyFee:          extra.BuyFee,
		redeemFee:       extra.RedeemFee,
		usdPlusDecimals: staticExtra.UsdPlusDecimals,
		assetDecimals:   staticExtra.AssetDecimals,
		gas:             defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if s.isPaused {
		return nil, ErrPoolIsPaused
	}

	if params.TokenAmountIn.Amount.Sign() <= 0 {
		return nil, ErrorInvalidAmountIn
	}

	var amountOut, feeAmount *big.Int
	if params.TokenAmountIn.Token == s.Pool.Info.Tokens[0] {
		amountOut, feeAmount = s.takeFee(s.mint(params.TokenAmountIn.Amount), true)
	} else if params.TokenAmountIn.Token == s.Pool.Info.Tokens[1] {
		amountOut, feeAmount = s.takeFee(s.redeem(params.TokenAmountIn.Amount), false)
	} else {
		return nil, ErrorInvalidTokenIn
	}

	if amountOut.Sign() <= 0 {
		return nil, ErrorInvalidAmountOut
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: amountOut},
		Fee:            &pool.TokenAmount{Token: params.TokenOut, Amount: feeAmount},
		Gas:            s.gas,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(_ pool.UpdateBalanceParams) {}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		Asset:       s.Info.Tokens[0],
		UsdPlus:     s.Info.Tokens[1],
		BlockNumber: s.Pool.Info.BlockNumber,
	}
}

func (s *PoolSimulator) takeFee(amount *big.Int, isBuy bool) (*big.Int, *big.Int) {
	var feeAmount = new(big.Int)

	if isBuy {
		feeAmount.Mul(amount, s.buyFee)
		feeAmount.Div(feeAmount, buyFeeDenominator)
	} else {
		feeAmount.Mul(amount, s.redeemFee)
		feeAmount.Div(feeAmount, redeemFeeDenominator)
	}

	return new(big.Int).Sub(amount, feeAmount), feeAmount
}

func (s *PoolSimulator) mint(amountIn *big.Int) *big.Int {
	divisor := new(big.Int)

	if s.assetDecimals > s.usdPlusDecimals {
		divisor = divisor.Exp(
			bignumber.Ten,
			big.NewInt(s.assetDecimals-s.usdPlusDecimals),
			nil,
		)

		return new(big.Int).Mul(amountIn, divisor)
	}

	divisor = divisor.Exp(
		bignumber.Ten,
		big.NewInt(s.usdPlusDecimals-s.assetDecimals),
		nil,
	)

	return new(big.Int).Div(amountIn, divisor)
}

func (s *PoolSimulator) redeem(amountIn *big.Int) *big.Int {
	amountOut := new(big.Int)
	divisor := new(big.Int)

	if s.assetDecimals > s.usdPlusDecimals {
		divisor = divisor.Exp(
			bignumber.Ten,
			big.NewInt(s.assetDecimals-s.usdPlusDecimals),
			nil,
		)

		return amountOut.Div(amountIn, divisor)
	}
	divisor = divisor.Exp(
		bignumber.Ten,
		big.NewInt(s.usdPlusDecimals-s.assetDecimals),
		nil,
	)

	return amountOut.Mul(amountIn, divisor)
}
