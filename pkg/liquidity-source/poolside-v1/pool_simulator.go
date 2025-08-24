package poolsidev1

import (
	"encoding/json"
	"errors"
	"math/big"
	"strings"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
)

var (
	ErrInvalidToken            = errors.New("INVALID_TOKEN")
	ErrInvalidAmountIn         = errors.New("INVALID_AMOUNT_IN")
	ErrInsufficientInputAmount = errors.New("INSUFFICIENT_INPUT_AMOUNT")
	ErrInvalidLiquidity        = errors.New("INVALID_LIQUIDITY")
	ErrInsufficientLiquidity   = errors.New("INSUFFICIENT_LIQUIDITY")
	ErrRatioOverFlow           = errors.New("RATIO_OVERFLOW")
)

type PoolSimulator struct {
	poolpkg.Pool

	fee          *uint256.Int
	feePrecision *uint256.Int

	gas Gas

	rebaseTokenInfoMap map[string]RebaseTokenInfo
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
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
		fee:                uint256.NewInt(extra.Fee),
		feePrecision:       uint256.NewInt(extra.FeePrecision),
		gas:                defaultGas,
		rebaseTokenInfoMap: extra.RebaseTokenInfoMap,
	}, nil
}

func (s *PoolSimulator) getRebaseTokenInfo(token string) (RebaseTokenInfo, string, bool) {
	if info, exists := s.rebaseTokenInfoMap[token]; exists {
		return info, token, true
	}

	for rebaseToken, info := range s.rebaseTokenInfoMap {
		if strings.EqualFold(info.UnderlyingToken, token) {
			return info, rebaseToken, true
		}
	}
	return RebaseTokenInfo{}, "", false
}

func (s *PoolSimulator) getRebaseToken(token string) string {
	_, rebaseToken, exists := s.getRebaseTokenInfo(token)
	if exists {
		return rebaseToken
	}
	return token
}

func (s *PoolSimulator) GetTokenIndexNumber(address string) int {
	for i, poolToken := range s.Info.Tokens {
		if strings.EqualFold(poolToken, address) {
			return i
		}
	}
	return -1
}

func (s *PoolSimulator) GetPoolTokenIndexes(tokenIn, tokenOut string) (int, int) {
	rebaseTokenIn := s.getRebaseToken(tokenIn)
	rebaseTokenOut := s.getRebaseToken(tokenOut)

	indexIn, indexOut := s.GetTokenIndexNumber(rebaseTokenIn), s.GetTokenIndexNumber(rebaseTokenOut)

	return indexIn, indexOut
}

func (s *PoolSimulator) CalcAmountOut(param poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := param.TokenAmountIn, param.TokenOut

	indexIn, indexOut := s.GetPoolTokenIndexes(tokenAmountIn.Token, tokenOut)

	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, ErrInvalidAmountIn
	}

	if amountIn.Cmp(number.Zero) <= 0 {
		return nil, ErrInsufficientInputAmount
	}

	convertedAmountIn, err := s.convertAmount(amountIn, tokenAmountIn.Token, false)
	if err != nil {
		return nil, err
	}

	poolIn, overflow := uint256.FromBig(s.Pool.Info.Reserves[indexIn])
	if overflow {
		return nil, ErrInvalidLiquidity
	}

	poolOut, overflow := uint256.FromBig(s.Pool.Info.Reserves[indexOut])
	if overflow {
		return nil, ErrInvalidLiquidity
	}

	if poolIn.Cmp(number.Zero) <= 0 || poolOut.Cmp(number.Zero) <= 0 {
		return nil, ErrInsufficientLiquidity
	}

	amountOut, err := s.getAmountOut(convertedAmountIn, poolIn, poolOut)
	if err != nil {
		return nil, err
	}

	if amountOut.Cmp(poolOut) > 0 {
		return nil, ErrInsufficientLiquidity
	}

	finalAmountOut, err := s.convertAmount(amountOut, tokenOut, true)
	if err != nil {
		return nil, err
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{
			Token:  s.Pool.Info.Tokens[indexOut],
			Amount: finalAmountOut.ToBig(),
		},
		Fee: &poolpkg.TokenAmount{
			Token:  s.Pool.Info.Tokens[indexIn],
			Amount: integer.Zero(),
		},
		Gas: s.gas.Swap,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params poolpkg.UpdateBalanceParams) {
	indexIn, indexOut := s.GetPoolTokenIndexes(params.TokenAmountIn.Token, params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return
	}

	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow {
		return
	}

	amountOut, overflow := uint256.FromBig(params.TokenAmountOut.Amount)
	if overflow {
		return
	}

	convertedAmountIn, err := s.convertAmount(amountIn, params.TokenAmountIn.Token, false)
	if err != nil {
		return
	}

	convertedAmountOut, err := s.convertAmount(amountOut, params.TokenAmountOut.Token, false)
	if err != nil {
		return
	}

	s.Info.Reserves[indexIn] = new(big.Int).Add(s.Info.Reserves[indexIn], convertedAmountIn.ToBig())
	s.Info.Reserves[indexOut] = new(big.Int).Sub(s.Info.Reserves[indexOut], convertedAmountOut.ToBig())
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		Fee:          s.fee.Uint64(),
		FeePrecision: s.feePrecision.Uint64(),
		BlockNumber:  s.Pool.Info.BlockNumber,
	}
}

func (s *PoolSimulator) getAmountOut(amountIn, poolIn, poolOut *uint256.Int) (amountOut *uint256.Int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	amountInWithFee := SafeMul(amountIn, SafeSub(s.feePrecision, s.fee))
	numerator := SafeMul(amountInWithFee, poolOut)
	denominator := SafeAdd(SafeMul(poolIn, s.feePrecision), amountInWithFee)

	return new(uint256.Int).Div(numerator, denominator), nil
}

func (s *PoolSimulator) convertAmount(amount *uint256.Int, token string, isUnwrap bool) (cAmount *uint256.Int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	info, rebaseToken, exists := s.getRebaseTokenInfo(token)

	if !exists || strings.EqualFold(rebaseToken, token) {
		return amount, nil
	}

	var ratio *uint256.Int
	var overflow bool

	if isUnwrap {
		ratio, overflow = uint256.FromBig(info.UnwrapRatio)
	} else {
		ratio, overflow = uint256.FromBig(info.WrapRatio)
	}

	if overflow {
		return nil, ErrRatioOverFlow
	}

	decimals := new(uint256.Int).Exp(uint256.NewInt(10), uint256.NewInt(uint64(info.Decimals)))
	return new(uint256.Int).Div(SafeMul(amount, ratio), decimals), nil
}
