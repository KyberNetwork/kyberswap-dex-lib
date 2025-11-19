package xsolvbtc

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

	Tokens          []*entity.PoolToken
	extra           PoolExtra
	withdrawFeeRate *big.Int
	gas             Gas
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra PoolExtra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     entityPool.Address,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      lo.Map(entityPool.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves:    lo.Map(entityPool.Reserves, func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		Tokens:          entity.ClonePoolTokens(entityPool.Tokens),
		extra:           extra,
		withdrawFeeRate: big.NewInt(int64(extra.WithdrawFeeRate)),
		gas:             defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if s.extra.Nav == nil {
		return nil, ErrNavNotSet
	}

	idIn, idOut := s.Info.GetTokenIndex(params.TokenAmountIn.Token), s.Info.GetTokenIndex(params.TokenOut)
	var amountOut *big.Int
	fee := new(big.Int)
	if idIn == 0 { // solvBTC aka deposit
		if !s.extra.DepositAllowed {
			return nil, ErrDepositNotAllowed
		}
		xSolvBtcAmount := new(big.Int)
		if s.extra.Nav.Sign() != 0 {
			xSolvBtcAmount.Mul(params.TokenAmountIn.Amount, bignumber.TenPowInt(int(s.Tokens[idOut].Decimals))).Div(xSolvBtcAmount, s.extra.Nav)
		}
		if xSolvBtcAmount.Cmp(params.TokenAmountIn.Amount) < 0 {
			return nil, ErrXSolvBTCAmount
		}
		amountOut = xSolvBtcAmount
	} else { // xSolvBTC aka withdraw
		solvBTCAmount, factor := new(big.Int), new(big.Int)
		factor.Mul(params.TokenAmountIn.Amount, s.extra.MaxMultiplier).Div(factor, BasisPoint)
		solvBTCAmount.Mul(params.TokenAmountIn.Amount, s.extra.Nav).Div(solvBTCAmount, bignumber.TenPowInt(int(s.Tokens[idIn].Decimals)))
		fee.Mul(solvBTCAmount, s.withdrawFeeRate).Div(fee, BasisPoint)
		solvBTCAmount.Sub(solvBTCAmount, fee)

		if solvBTCAmount.Cmp(factor) >= 0 {
			return nil, ErrSolvBTCAmount
		}
		amountOut = solvBTCAmount
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: amountOut},
		Fee:            &pool.TokenAmount{Token: params.TokenOut, Amount: fee},
		Gas:            lo.Ternary(idIn == 0, s.gas.Deposit, s.gas.Withdraw),
		SwapInfo: &SwapInfo{
			IsDeposit: idIn == 0,
		},
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	return p
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) any {
	return PoolMeta{
		BlockNumber: s.Info.BlockNumber,
	}
}
