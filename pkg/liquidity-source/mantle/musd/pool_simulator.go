package musd

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/mantle/common"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
)

type (
	PoolSimulator struct {
		poolpkg.Pool

		paused      bool
		oraclePrice *uint256.Int

		gas Gas
	}

	Gas struct {
		Wrap   int64
		Unwrap int64
	}
)

var (
	ErrUnwrapTooSmall = errors.New("unwrap too small")
)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra PoolExtra
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
			Reserves:    lo.Map(entityPool.Reserves, func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		paused:      extra.Paused,
		oraclePrice: extra.OraclePrice,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	if s.paused {

	}

	tokenAmountIn := params.TokenAmountIn
	tokenOut := params.TokenOut

	var tokenInIndex = s.GetTokenIndex(tokenAmountIn.Token)
	var tokenOutIndex = s.GetTokenIndex(tokenOut)

	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return &poolpkg.CalcAmountOutResult{}, fmt.Errorf("invalid tokenIn or tokenOut: %v, %v", tokenAmountIn.Token, tokenOut)
	}

	var (
		amountOut *big.Int
		gas       int64
	)

	if tokenAmountIn.Token == s.Pool.Info.Tokens[0] {
		usdySharesAmount, err := s.unwrap(uint256.MustFromBig(tokenAmountIn.Amount))
		if err != nil {
			return nil, err
		}
		amountOut = usdySharesAmount
		gas = s.gas.Unwrap
	} else {
		amountOut = s.wrap(uint256.MustFromBig(tokenAmountIn.Amount))
		gas = s.gas.Wrap
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: params.TokenOut, Amount: amountOut},
		Fee:            &poolpkg.TokenAmount{Token: params.TokenOut, Amount: bignumber.ZeroBI},
		Gas:            gas,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(_ poolpkg.UpdateBalanceParams) {}

func (s *PoolSimulator) wrap(USDYAmount *uint256.Int) *big.Int {
	return USDYAmount.Mul(USDYAmount, common.BasisPoints).ToBig()
}

func (s *PoolSimulator) unwrap(rUSDYAmount *uint256.Int) (*big.Int, error) {
	usdyAmount := s.getSharesByRUSDY(rUSDYAmount)
	if usdyAmount.Cmp(common.BasisPoints) < 0 {
		return nil, ErrUnwrapTooSmall
	}

	return usdyAmount.Div(usdyAmount, common.BasisPoints).ToBig(), nil
}

// (_rUSDYAmount * 1e18 * BPS_DENOMINATOR) / oracle.getPrice()
func (s *PoolSimulator) getSharesByRUSDY(rUSDYAmount *uint256.Int) *uint256.Int {
	return rUSDYAmount.
		Mul(rUSDYAmount, common.OneE18).
		Mul(rUSDYAmount, common.BasisPoints).
		Div(rUSDYAmount, s.oraclePrice)
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		BlockNumber: s.Pool.Info.BlockNumber,
	}
}
