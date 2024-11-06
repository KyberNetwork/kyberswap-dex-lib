package staderethx

import (
	"errors"
	"math/big"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type (
	PoolSimulator struct {
		poolpkg.Pool

		paused          bool
		minDeposit      *uint256.Int
		maxDeposit      *uint256.Int
		exchangeRate    *uint256.Int
		totalETHXSupply *uint256.Int
		totalETHBalance *uint256.Int

		gas Gas
	}
)

var (
	ErrPoolPaused           = errors.New("pool is paused")
	ErrInvalidTokenIn       = errors.New("invalid tokenIn")
	ErrInvalidTokenOut      = errors.New("invalid tokenOut")
	ErrInvalidDepositAmount = errors.New("invalid deposit amount")
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
		paused:          extra.Paused,
		minDeposit:      extra.MinDeposit,
		maxDeposit:      extra.MaxDeposit,
		totalETHXSupply: extra.TotalETHXSupply,
		totalETHBalance: extra.TotalETHBalance,
		gas:             defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	if s.paused {
		return nil, ErrPoolPaused
	}

	if params.TokenAmountIn.Token != s.Pool.Info.Tokens[0] {
		return nil, ErrInvalidTokenIn
	}

	if params.TokenOut != s.Info.Tokens[1] {
		return nil, ErrInvalidTokenOut
	}

	amountIn := new(uint256.Int).Set(uint256.MustFromBig(params.TokenAmountIn.Amount))
	if amountIn.Cmp(s.minDeposit) < 0 || amountIn.Cmp(s.maxDeposit) > 0 {
		return nil, ErrInvalidDepositAmount
	}

	amountOut, err := s.previewDeposit(amountIn)
	if err != nil {
		return nil, err
	}

	// NOTE: using previewDeposit instead of calculating amountOut = amountIn * 1e18 / exchangeRate
	// (exchangeRate from staderStakePoolsManagerMethodGetExchangeRate) to avoid precision issue
	// amountOut = new(uint256.Int).Mul(amountIn, number.Number_1e18)
	// amountOut.Div(amountOut, s.exchangeRate)

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: params.TokenOut, Amount: amountOut.ToBig()},
		Fee:            &poolpkg.TokenAmount{Token: params.TokenOut, Amount: bignumber.ZeroBI},
		Gas:            s.gas.Deposit,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params poolpkg.UpdateBalanceParams) {

}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		BlockNumber: s.Pool.Info.BlockNumber,
	}
}

func (s *PoolSimulator) CanSwapTo(token string) []string {
	if token == ETHx {
		return []string{WETH}
	}
	return []string{}
}

func (s *PoolSimulator) CanSwapFrom(token string) []string {
	if token == WETH {
		return []string{ETHx}
	}
	return []string{}
}

func (s *PoolSimulator) previewDeposit(assets *uint256.Int) (*uint256.Int, error) {
	supply := s.totalETHXSupply
	if assets.IsZero() || s.totalETHXSupply.IsZero() {
		return assets, nil
	} else {
		return mulDiv(assets, supply, s.totalETHBalance)
	}
}
