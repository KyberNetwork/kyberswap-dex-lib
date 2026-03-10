package poe

import (
	"math/big"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	reserveX *uint256.Int
	reserveY *uint256.Int
	price    *uint256.Int
	feeHbps  *uint256.Int
	alpha    *uint256.Int
	expiry   uint64
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:     ep.Address,
				Exchange:    ep.Exchange,
				Type:        ep.Type,
				Tokens:      lo.Map(ep.Tokens, func(t *entity.PoolToken, _ int) string { return t.Address }),
				Reserves:    lo.Map(ep.Reserves, func(r string, _ int) *big.Int { return bignumber.NewBig(r) }),
				BlockNumber: ep.BlockNumber,
			},
		},
		reserveX: uint256.MustFromDecimal(ep.Reserves[0]),
		reserveY: uint256.MustFromDecimal(ep.Reserves[1]),
		price:    extra.Price,
		feeHbps:  extra.FeeHbps,
		alpha:    extra.Alpha,
		expiry:   extra.Expiry,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut

	indexIn, indexOut := s.GetTokenIndex(tokenAmountIn.Token), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	amountIn := uint256.MustFromBig(tokenAmountIn.Amount)
	if amountIn.Sign() <= 0 {
		return nil, ErrInvalidAmountIn
	}

	if err := s.validate(); err != nil {
		return nil, err
	}

	isXtoY := indexIn == 0

	vr := computeVirtualReserves(s.reserveX, s.reserveY, s.price, s.alpha)
	if vr.xv.IsZero() || vr.yv.IsZero() {
		return nil, ErrZeroVirtualReserve
	}

	feeIn := applyFeeCeil(amountIn, s.feeHbps)
	netAmountIn := new(uint256.Int).Sub(amountIn, feeIn)

	xvIn, xvOut := lo.Ternary(isXtoY, vr.xv, vr.yv), lo.Ternary(isXtoY, vr.yv, vr.xv)
	realReserveOut := lo.Ternary(isXtoY, s.reserveY, s.reserveX)
	amountOut := calcAmountOutCPMM(xvIn, xvOut, netAmountIn)
	if amountOut.IsZero() {
		return nil, ErrInvalidAmountOut
	}

	if amountOut.Gt(realReserveOut) {
		return nil, ErrInsufficientLiquidity
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: tokenAmountIn.Token, Amount: feeIn.ToBig()},
		Gas:            defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	tokenAmountOut := param.TokenAmountOut
	tokenIn := param.TokenIn

	indexIn, indexOut := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	amountOut := uint256.MustFromBig(tokenAmountOut.Amount)
	if amountOut.Sign() <= 0 {
		return nil, ErrInvalidAmountOut
	}

	if err := s.validate(); err != nil {
		return nil, err
	}

	isXtoY := indexIn == 0

	realReserveOut := lo.Ternary(isXtoY, s.reserveY, s.reserveX)
	if amountOut.Cmp(realReserveOut) >= 0 {
		return nil, ErrInsufficientLiquidity
	}

	vr := computeVirtualReserves(s.reserveX, s.reserveY, s.price, s.alpha)
	if vr.xv.IsZero() || vr.yv.IsZero() {
		return nil, ErrZeroVirtualReserve
	}

	xvIn, xvOut := lo.Ternary(isXtoY, vr.xv, vr.yv), lo.Ternary(isXtoY, vr.yv, vr.xv)
	netAmountIn := calcAmountInCPMM(xvIn, xvOut, amountOut)
	if netAmountIn == nil {
		return nil, ErrInsufficientLiquidity
	}

	feeForNet := deductFeeCeil(netAmountIn, s.feeHbps)
	if feeForNet == nil {
		return nil, ErrInsufficientLiquidity
	}
	amountIn := new(uint256.Int).Add(netAmountIn, feeForNet)
	feeIn := applyFeeCeil(amountIn, s.feeHbps)

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: tokenIn, Amount: amountIn.ToBig()},
		Fee:           &pool.TokenAmount{Token: tokenIn, Amount: feeIn.ToBig()},
		Gas:           defaultGas,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return
	}

	s.Info.Reserves[indexIn] = new(big.Int).Add(s.Info.Reserves[indexIn], params.TokenAmountIn.Amount)
	s.Info.Reserves[indexOut] = new(big.Int).Sub(s.Info.Reserves[indexOut], params.TokenAmountOut.Amount)

	amountIn := uint256.MustFromBig(params.TokenAmountIn.Amount)
	amountOut := uint256.MustFromBig(params.TokenAmountOut.Amount)
	fee := uint256.MustFromBig(params.Fee.Amount)
	netIn := new(uint256.Int).Sub(amountIn, fee)

	isXtoY := indexIn == 0
	if isXtoY {
		s.reserveX.Add(s.reserveX, netIn)
		s.reserveY.Sub(s.reserveY, amountOut)
	} else {
		s.reserveY.Add(s.reserveY, netIn)
		s.reserveX.Sub(s.reserveX, amountOut)
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.reserveX = new(uint256.Int).Set(s.reserveX)
	cloned.reserveY = new(uint256.Int).Set(s.reserveY)

	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(tokenIn, _ string) any {
	return PoolMeta{
		BlockNumber: s.Info.BlockNumber,
		IsXtoY:      s.GetTokenIndex(tokenIn) == 0,
	}
}

func (s *PoolSimulator) validate() error {
	if s.alpha.Cmp(bps) <= 0 {
		return ErrInvalidAlpha
	}

	if s.price.IsZero() {
		return ErrZeroReserve
	}

	now := uint64(time.Now().Unix())
	if now > s.expiry {
		return ErrExpiredOracle
	}

	return nil
}
