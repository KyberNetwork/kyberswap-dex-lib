package beets_ss

import (
	"errors"
	"math/big"
	"strings"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var (
	ErrInvalidToken            = errors.New("invalid token")
	ErrInvalidReserve          = errors.New("invalid reserve")
	ErrInvalidAmountIn         = errors.New("invalid amount in")
	ErrInsufficientInputAmount = errors.New("INSUFFICIENT_INPUT_AMOUNT")

	ErrDepositTooSmall = errors.New("deposit too small")
	ErrDepositPaused   = errors.New("deposit paused")
	ErrOverflow        = errors.New("overflow")
)

type (
	PoolSimulator struct {
		poolpkg.Pool

		totalSupply   *uint256.Int
		totalAssets   *uint256.Int
		depositPaused bool

		gas Gas
	}

	Gas struct {
		Swap int64
	}
)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	totalAssets, overflow := uint256.FromBig(extra.TotalAssets)
	if overflow {
		return nil, ErrOverflow
	}

	totalSupply, overflow := uint256.FromBig(extra.TotalSupply)
	if overflow {
		return nil, ErrOverflow
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
		totalAssets:   totalAssets,
		totalSupply:   totalSupply,
		depositPaused: extra.DepositPaused,

		gas: defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	var (
		tokenAmountIn = param.TokenAmountIn
		tokenOut      = param.TokenOut
	)

	indexIn, indexOut := s.GetTokenIndex(tokenAmountIn.Token), s.GetTokenIndex(tokenOut)
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

	amountOut, err := s.deposit(amountIn)
	if err != nil {
		return nil, err
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: s.Pool.Info.Tokens[indexOut], Amount: amountOut.ToBig()},
		Fee:            &poolpkg.TokenAmount{Token: s.Pool.Info.Tokens[indexIn], Amount: integer.Zero()},
		Gas:            s.gas.Swap,
	}, nil
}

func (s *PoolSimulator) deposit(amountIn *uint256.Int) (*uint256.Int, error) {
	if amountIn.Cmp(MIN_DEPOSIT) < 0 {
		return nil, ErrDepositTooSmall
	}

	if s.depositPaused {
		return nil, ErrDepositPaused
	}

	sharesAmount := s.convertToShares(amountIn)

	return sharesAmount, nil
}

func (s *PoolSimulator) convertToShares(assetAmount *uint256.Int) *uint256.Int {
	if s.totalAssets.IsZero() || s.totalSupply.IsZero() {
		return assetAmount
	}

	return new(uint256.Int).Div(
		new(uint256.Int).Mul(assetAmount, s.totalSupply),
		s.totalAssets,
	)
}

func (s *PoolSimulator) UpdateBalance(params poolpkg.UpdateBalanceParams) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return
	}

	s.Pool.Info.Reserves[indexIn] = new(big.Int).Add(s.Pool.Info.Reserves[indexIn], params.TokenAmountIn.Amount)
	s.Pool.Info.Reserves[indexOut] = new(big.Int).Sub(s.Pool.Info.Reserves[indexOut], params.TokenAmountOut.Amount)

	s.totalAssets = new(uint256.Int).Add(s.totalAssets, uint256.MustFromBig(params.TokenAmountIn.Amount))
	s.totalSupply = new(uint256.Int).Add(s.totalSupply, uint256.MustFromBig(params.TokenAmountOut.Amount))
}

func (s *PoolSimulator) CanSwapTo(token string) []string {
	if strings.EqualFold(token, Beets_Staked_Sonic_Address) {
		return []string{strings.ToLower(valueobject.WrappedNativeMap[valueobject.ChainIDSonic])}
	}
	return []string{}
}

func (s *PoolSimulator) CanSwapFrom(token string) []string {
	if strings.EqualFold(token, valueobject.WrappedNativeMap[valueobject.ChainIDSonic]) {
		return []string{Beets_Staked_Sonic_Address}
	}
	return []string{}
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		BlockNumber: s.Pool.Info.BlockNumber,
	}
}
