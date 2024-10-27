package rseth

import (
	"errors"
	"math/big"

	"github.com/bytedance/sonic"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ErrInvalidTokenOut            = errors.New("invalid tokenOut")
	ErrInvalidAmountToDeposit     = errors.New("invalid amount to deposit")
	ErrMaximumDepositLimitReached = errors.New("maximum deposit limit reached")
)

type PoolSimulator struct {
	poolpkg.Pool

	// minAmountToDeposit: minAmountToDeposit
	minAmountToDeposit *big.Int

	// totalDepositByAsset: getTotalAssetDeposits
	totalDepositByAsset map[string]*big.Int

	// depositLimitByAsset: lrtConfig.depositLimitByAsset
	depositLimitByAsset map[string]*big.Int

	// priceByAsset: lrtOracle.getAssetPrice
	priceByAsset map[string]*big.Int

	// rsETHPrice: lrtOracle.rsETHPrice
	rsETHPrice *big.Int

	gas Gas
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra PoolExtra
	if err := sonic.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
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
		minAmountToDeposit:  extra.MinAmountToDeposit,
		totalDepositByAsset: extra.TotalDepositByAsset,
		depositLimitByAsset: extra.DepositLimitByAsset,
		priceByAsset:        extra.PriceByAsset,
		rsETHPrice:          extra.RSETHPrice,
		gas:                 defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	if param.TokenOut != s.Info.Tokens[0] {
		return nil, ErrInvalidTokenOut
	}

	amountOut, err := s._beforeDeposit(param.TokenAmountIn.Token, param.TokenAmountIn.Amount)
	if err != nil {
		return nil, err
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: param.TokenOut, Amount: amountOut},
		Fee:            &poolpkg.TokenAmount{Token: param.TokenOut, Amount: bignumber.ZeroBI},
		Gas:            s.gas.DepositAsset,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(param poolpkg.UpdateBalanceParams) {
	totalDeposit := s.totalDepositByAsset[param.TokenAmountIn.Token]

	newTotalDeposit := new(big.Int).Add(totalDeposit, param.TokenAmountIn.Amount)

	s.totalDepositByAsset[param.TokenAmountIn.Token] = newTotalDeposit
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		BlockNumber: s.Pool.Info.BlockNumber,
	}
}

func (s *PoolSimulator) _beforeDeposit(
	asset string,
	depositAmount *big.Int,
) (*big.Int, error) {
	if depositAmount.Cmp(bignumber.ZeroBI) == 0 || depositAmount.Cmp(s.minAmountToDeposit) < 0 {
		return nil, ErrInvalidAmountToDeposit
	}

	if depositAmount.Cmp(s.getAssetCurrentLimit(asset)) > 0 {
		return nil, ErrMaximumDepositLimitReached
	}

	return s.getRsETHAmountToMint(asset, depositAmount), nil
}

func (s *PoolSimulator) getAssetCurrentLimit(asset string) *big.Int {
	totalDeposit := s.totalDepositByAsset[asset]
	depositLimit := s.depositLimitByAsset[asset]

	if totalDeposit.Cmp(depositLimit) > 0 {
		return bignumber.ZeroBI
	}

	return new(big.Int).Sub(depositLimit, totalDeposit)
}

func (s *PoolSimulator) getRsETHAmountToMint(asset string, amount *big.Int) *big.Int {
	return new(big.Int).Div(new(big.Int).Mul(amount, s.priceByAsset[asset]), s.rsETHPrice)
}
