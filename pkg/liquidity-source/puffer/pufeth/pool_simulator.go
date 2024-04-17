package pufeth

import (
	"errors"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ErrUnsupportedSwap = errors.New("unsupported swap")
	ErrInvalidAmountIn = errors.New("invalid amountIn")
)

// Depositor: https://etherscan.io/address/0x4aa799c5dfc01ee7d790e3bf1a7c2257ce1dceff
// Vault: https://etherscan.io/address/0xD9A442856C234a39a81a089C06451EBAa4306a72
type PoolSimulator struct {
	poolpkg.Pool

	// totalSupply: PufferVaultMethodTotalSupply
	totalSupply *uint256.Int

	// totalAssets: PufferVaultMethodTotalAssets
	totalAssets *uint256.Int

	// totalPooledEther: LidoMethodGetTotalPooledEther
	totalPooledEther *uint256.Int

	// totalShares: LidoMethodGetTotalShares
	totalShares *uint256.Int

	gas Gas
}

type Gas struct {
	depositStETH  int64 // 250000
	depositWstETH int64 // 280000
}

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
		totalSupply:      extra.TotalSupply,
		totalAssets:      extra.TotalAssets,
		totalPooledEther: extra.TotalPooledEther,
		totalShares:      extra.TotalShares,
		gas:              defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	// NOTE: only support tokenIn = stETH, wstETH and tokenOut is pufETH
	if !((params.TokenAmountIn.Token == s.Info.Tokens[1] || params.TokenAmountIn.Token == s.Info.Tokens[2]) && params.TokenOut == s.Info.Tokens[0]) {
		return nil, ErrUnsupportedSwap
	}

	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow {
		return nil, ErrInvalidAmountIn
	}

	gas := s.gas.depositStETH
	if params.TokenAmountIn.Token == s.Info.Tokens[2] {
		amountIn = s.unwrap(amountIn)
		gas = s.gas.depositWstETH
	}

	amountOut, err := s.convertToShares(amountIn)
	if err != nil {
		return nil, err
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: params.TokenOut, Amount: amountOut.ToBig()},
		Fee:            &poolpkg.TokenAmount{Token: params.TokenOut, Amount: bignumber.ZeroBI},
		Gas:            gas,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params poolpkg.UpdateBalanceParams) {
	amountIn, _ := uint256.FromBig(params.TokenAmountIn.Amount)
	amountOut, _ := uint256.FromBig(params.TokenAmountOut.Amount)

	s.totalSupply = new(uint256.Int).Add(s.totalSupply, amountOut)

	if params.TokenAmountIn.Token == s.Info.Tokens[2] {
		s.totalAssets = new(uint256.Int).Add(s.totalAssets, s.unwrap(amountIn))
	} else {
		s.totalAssets = new(uint256.Int).Add(s.totalAssets, amountIn)
	}
}

// NOTE: only support tokenIn = stETH, wstETH and tokenOut is pufETH
func (s *PoolSimulator) CanSwapTo(token string) []string {
	if token == PUFETH {
		return []string{STETH, WSTETH}
	}
	return []string{}
}

func (s *PoolSimulator) CanSwapFrom(token string) []string {
	if token == STETH || token == WSTETH {
		return []string{PUFETH}
	}
	return []string{}
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		BlockNumber: s.Pool.Info.BlockNumber,
	}
}

func (s *PoolSimulator) unwrap(amount *uint256.Int) *uint256.Int {
	return new(uint256.Int).Div(
		new(uint256.Int).Mul(amount, s.totalPooledEther),
		s.totalShares,
	)
}

func (s *PoolSimulator) convertToShares(amount *uint256.Int) (*uint256.Int, error) {
	return Math.MulDivF(amount, new(uint256.Int).Add(s.totalSupply, number.Number_1), new(uint256.Int).Add(s.totalAssets, number.Number_1))
}
