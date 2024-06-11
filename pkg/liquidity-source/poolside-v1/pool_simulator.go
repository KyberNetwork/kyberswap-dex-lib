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
	ErrInvalidLiquidity        = errors.New("INVALID_Liquidity")
	ErrInsufficientLiquidity   = errors.New("INSUFFICIENT_LIQUIDITY")
)

type PoolSimulator struct { //customize
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
		fee:          uint256.NewInt(extra.Fee),
		feePrecision: uint256.NewInt(extra.FeePrecision),
		gas:          defaultGas,
	}, nil
}

func (s *PoolSimulator) getRebaseToken(token string) string {

	if _, exists := s.rebaseTokenInfoMap[token]; exists {
		return token
	}

	for rebaseToken, info := range s.rebaseTokenInfoMap {
		if info.UnderlyingToken == token {
			return rebaseToken
		}
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

	// Have to convert amountIn if tokenIn is underlying token

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

	amountOut, err := s.getAmountOut(amountIn, poolIn, poolOut)

	if err != nil {
		return nil, err
	}

	if amountOut.Cmp(poolOut) > 0 {
		return nil, ErrInsufficientLiquidity
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{
			Token:  s.Pool.Info.Tokens[indexOut],
			Amount: amountOut.ToBig(),
		},
		Fee: &poolpkg.TokenAmount{
			Token:  s.Pool.Info.Tokens[indexIn],
			Amount: integer.Zero(),
		},
		Gas: s.gas.Swap,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params poolpkg.UpdateBalanceParams) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return
	}

	s.Info.Reserves[indexIn] = new(big.Int).Add(s.Info.Reserves[indexIn], params.TokenAmountIn.Amount)
	s.Info.Reserves[indexOut] = new(big.Int).Sub(s.Info.Reserves[indexOut], params.TokenAmountOut.Amount)
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
