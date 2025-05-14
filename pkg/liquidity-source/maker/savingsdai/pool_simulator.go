package savingsdai

import (
	"math/big"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/maker/savingsusds"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type (
	PoolSimulator struct {
		pool.Pool

		now         *uint256.Int
		rho         *uint256.Int
		chi         *uint256.Int
		savingsRate *uint256.Int
	}

	SwapInfo struct {
		chi       *uint256.Int
		IsDeposit bool `json:"isDeposit"`
	}

	PoolMetaInfo struct {
		BlockNumber     uint64 `json:"blockNumber"`
		ApprovalAddress string `json:"approvalAddress"`
	}

	Gas struct {
		Deposit int64
		Redeem  int64
	}
)

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	tokens := lo.Map(entityPool.Tokens, func(token *entity.PoolToken, _ int) string {
		return token.Address
	})

	reserves := lo.Map(entityPool.Reserves, func(reserve string, _ int) *big.Int {
		return bignumber.NewBig10(reserve)
	})

	if len(tokens) != 2 && len(reserves) != 2 {
		return nil, ErrInvalidToken
	}

	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     entityPool.Address,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      tokens,
			Reserves:    reserves,
			Checked:     true,
			BlockNumber: entityPool.BlockNumber,
		}},
		now:         extra.BlockTimestamp,
		rho:         extra.RHO,
		chi:         extra.CHI,
		savingsRate: extra.SavingsRate,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := params.TokenAmountIn, params.TokenOut

	if err := s.validate(tokenAmountIn.Token, tokenOut); err != nil {
		return nil, err
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, ErrOverflow
	}

	chi, err := s._chi()
	if err != nil {
		return nil, err
	}

	isDeposit := tokenAmountIn.Token == s.Info.Tokens[0]

	amountOut := lo.Ternary(
		isDeposit,
		s.deposit(amountIn, chi),
		s.redeem(amountIn, chi),
	)

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: bignumber.ZeroBI,
		},
		Gas: s.estimateGas(isDeposit),
		SwapInfo: SwapInfo{
			chi:       chi,
			IsDeposit: isDeposit,
		},
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swapInfo, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		return
	}
	s.chi = swapInfo.chi
	s.rho = s.now
}

func (s *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) interface{} {
	return PoolMetaInfo{
		BlockNumber:     s.Info.BlockNumber,
		ApprovalAddress: s.GetApprovalAddress(tokenIn, tokenOut),
	}
}

func (s *PoolSimulator) GetApprovalAddress(tokenIn, _ string) string {
	return lo.Ternary(tokenIn == s.Info.Tokens[0], s.Info.Tokens[1], "")
}

func (s *PoolSimulator) deposit(assets, chi *uint256.Int) *uint256.Int {
	var shares uint256.Int
	return shares.Mul(assets, RAY).Div(&shares, chi)
}

func (s *PoolSimulator) redeem(shares, chi *uint256.Int) *uint256.Int {
	var assets uint256.Int
	return assets.Mul(shares, chi).Div(&assets, RAY)
}

func (s *PoolSimulator) _chi() (*uint256.Int, error) {
	if s.now.Gt(s.rho) {
		return s.drip()
	}
	return s.chi, nil
}

func (s *PoolSimulator) drip() (*uint256.Int, error) {
	x, err := rpow(s.savingsRate, new(uint256.Int).Sub(s.now, s.rho), RAY)
	if err != nil {
		return nil, err
	}

	return rmul(x, s.chi)
}

func (s *PoolSimulator) validate(tokenIn, tokenOut string) error {
	if tokenIn == tokenOut {
		return ErrInvalidToken
	}
	inIdx, outIdx := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenOut)
	if inIdx < 0 || outIdx < 0 {
		return ErrInvalidToken
	}
	return nil
}

func (s *PoolSimulator) estimateGas(isDeposit bool) int64 {
	var gas Gas
	switch s.GetExchange() {
	case savingsusds.DexType:
		gas = savingsUSDSDefaultGas
	default:
		gas = savingsDAIDefaultGas
	}

	return lo.Ternary(
		isDeposit,
		gas.Deposit,
		gas.Redeem,
	)
}
