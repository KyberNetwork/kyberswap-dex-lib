package obric

import (
	"math/big"
	"slices"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	decimalsX, decimalsY uint8
	multYBase            *uint256.Int
	currentXK            *uint256.Int
	preK                 *uint256.Int
	feeMillionth         *uint256.Int
	priceMaxAge          uint64
	priceUpdateTime      uint64
	isLocked             bool
	enable               bool
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
		decimalsX:       ep.Tokens[0].Decimals,
		decimalsY:       ep.Tokens[1].Decimals,
		multYBase:       bignumber.NewUint256(staticExtra.MultYBase),
		currentXK:       bignumber.NewUint256(extra.CurrentXK),
		preK:            bignumber.NewUint256(extra.PreK),
		feeMillionth:    uint256.NewInt(extra.FeeMillionth),
		priceMaxAge:     extra.PriceMaxAge,
		priceUpdateTime: extra.PriceUpdateTime,
		isLocked:        extra.IsLocked,
		enable:          extra.Enable,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	var (
		tokenAmountIn = param.TokenAmountIn
		tokenOut      = param.TokenOut
	)

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

	k := s.calculateK()
	if k.IsZero() {
		return nil, ErrZeroCurrentXK
	}

	currentYK := new(uint256.Int).Div(k, s.currentXK)

	isXtoY := indexIn == 0

	var currentLK, currentRK *uint256.Int
	if isXtoY {
		currentLK = new(uint256.Int).Set(s.currentXK)
		currentRK = currentYK
	} else {
		currentLK = currentYK
		currentRK = new(uint256.Int).Set(s.currentXK)
	}

	newLK := new(uint256.Int).Add(currentLK, amountIn)
	newRK := new(uint256.Int).Div(k, newLK)

	if currentRK.Cmp(newRK) <= 0 {
		return nil, ErrInsufficientLiquidity
	}
	outputBeforeFee := new(uint256.Int).Sub(currentRK, newRK)

	reserveOut, overflow := uint256.FromBig(s.Info.Reserves[indexOut])
	if overflow || reserveOut.Sign() <= 0 {
		return nil, ErrInsufficientLiquidity
	}

	if outputBeforeFee.Cmp(reserveOut) > 0 {
		return nil, ErrInsufficientLiquidity
	}

	fee := u256.MulDiv(outputBeforeFee, s.feeMillionth, millionth)

	amountOut := new(uint256.Int).Sub(outputBeforeFee, fee)

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: tokenOut, Amount: fee.ToBig()},
		Gas:            defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	var (
		tokenAmountOut = param.TokenAmountOut
		tokenIn        = param.TokenIn
	)

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

	k := s.calculateK()
	if k.IsZero() {
		return nil, ErrZeroCurrentXK
	}

	currentYK := new(uint256.Int).Div(k, s.currentXK)

	isXtoY := indexIn == 0

	var currentLK, currentRK *uint256.Int
	if isXtoY {
		currentLK = new(uint256.Int).Set(s.currentXK)
		currentRK = currentYK
	} else {
		currentLK = currentYK
		currentRK = new(uint256.Int).Set(s.currentXK)
	}

	reserveOut, overflow := uint256.FromBig(s.Info.Reserves[indexOut])
	if overflow || reserveOut.Sign() <= 0 {
		return nil, ErrInsufficientLiquidity
	}

	if amountOut.Cmp(reserveOut) > 0 {
		return nil, ErrInsufficientLiquidity
	}

	feeDenom := new(uint256.Int).Sub(millionth, s.feeMillionth)
	if feeDenom.IsZero() {
		return nil, ErrInsufficientLiquidity
	}
	outputPlusFee := u256.MulDiv(amountOut, millionth, feeDenom)

	if outputPlusFee.Cmp(currentRK) >= 0 {
		return nil, ErrInsufficientLiquidity
	}

	newRK := new(uint256.Int).Sub(currentRK, outputPlusFee)
	newLK := new(uint256.Int).Div(k, newRK)

	if newLK.Cmp(currentLK) <= 0 {
		return nil, ErrInsufficientLiquidity
	}

	amountIn := new(uint256.Int).Sub(newLK, currentLK)

	fee := u256.MulDiv(outputPlusFee, s.feeMillionth, millionth)

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: tokenIn, Amount: amountIn.ToBig()},
		Fee:           &pool.TokenAmount{Token: tokenAmountOut.Token, Amount: fee.ToBig()},
		Gas:           defaultGas,
	}, nil
}

func (s *PoolSimulator) validate() error {
	if !s.enable {
		return ErrPoolDisabled
	}

	if s.currentXK.IsZero() {
		return ErrZeroCurrentXK
	}

	if s.isLocked {
		return ErrPoolLocked
	}

	now := uint64(time.Now().Unix())
	if now+priceBufferSeconds > s.priceUpdateTime+s.priceMaxAge {
		return ErrPriceStale
	}

	return nil
}

func (s *PoolSimulator) calculateK() *uint256.Int {
	k := s.preK.Clone()

	if s.multYBase.IsZero() {
		return u256.New0()
	}

	if s.decimalsX > s.decimalsY {
		multFactor := u256.TenPow(uint64(s.decimalsX - s.decimalsY))
		k.Div(k, multFactor)
		k.Div(k, s.multYBase)
	} else {
		multFactor := u256.TenPow(uint64(s.decimalsY - s.decimalsX))
		k.MulDivOverflow(k, multFactor, s.multYBase)
	}

	return k
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return
	}

	s.Info.Reserves[indexIn] = new(big.Int).Add(s.Info.Reserves[indexIn], params.TokenAmountIn.Amount)
	s.Info.Reserves[indexOut] = new(big.Int).Sub(s.Info.Reserves[indexOut], params.TokenAmountOut.Amount)

	amountIn := uint256.MustFromBig(params.TokenAmountIn.Amount)

	k := s.calculateK()
	if k.IsZero() {
		return
	}

	isXtoY := indexIn == 0
	if isXtoY {
		s.currentXK.Add(s.currentXK, amountIn)
	} else {
		currentYK := new(uint256.Int).Div(k, s.currentXK)
		newYK := new(uint256.Int).Add(currentYK, amountIn)
		s.currentXK.Div(k, newYK)
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.Info.Reserves = slices.Clone(s.Info.Reserves)
	cloned.currentXK = new(uint256.Int).Set(s.currentXK)
	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(tokenIn, _ string) any {
	return PoolMeta{
		BlockNumber: s.Info.BlockNumber,
		IsXtoY:      s.GetTokenIndex(tokenIn) == 0,
	}
}
