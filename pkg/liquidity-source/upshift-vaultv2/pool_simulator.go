package upshiftvaultv2

import (
	"math/big"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// PoolSimulator models Tokenized Vault V2 deposit(asset, amount, receiver) of the reference
// asset. Tokens are [asset, LP token]; deposit only.
type PoolSimulator struct {
	pool.Pool
	Extra
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}
	if extra.Supply == nil || extra.Valuation == nil || extra.TotalAssets == nil || extra.MaxDeposit == nil || extra.DepositCap == nil ||
		len(entityPool.Tokens) != 2 {
		return nil, ErrInvalidExtra
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:  entityPool.Address,
			Exchange: entityPool.Exchange,
			Type:     entityPool.Type,
			Tokens: lo.Map(entityPool.Tokens,
				func(item *entity.PoolToken, _ int) string { return item.Address }),
			Reserves: lo.Map(entityPool.Reserves,
				func(item string, _ int) *big.Int { return bignumber.NewBig10(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		Extra: extra,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn, tokenOut := param.TokenAmountIn.Token, param.TokenOut
	if tokenIn != s.Info.Tokens[0] || tokenOut != s.Info.Tokens[1] {
		return nil, ErrInvalidSwapPath
	}
	if param.TokenAmountIn.Amount == nil || param.TokenAmountIn.Amount.Sign() <= 0 {
		return nil, ErrZeroAmount
	}
	amount, overflow := uint256.FromBig(param.TokenAmountIn.Amount)
	if overflow {
		return nil, ErrOverflow
	}

	var shares uint256.Int
	if err := s.depositShares(&shares, amount); err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: shares.ToBig()},
		Fee:            &pool.TokenAmount{Token: tokenOut, Amount: bignumber.ZeroBI},
		Gas:            gasOf(s.Info.Address),
		SwapInfo:       SwapInfo{Shares: &shares},
	}, nil
}

// UpdateBalance mints the shares and adds the deposit to the valuation, as deposit() does.
// MaxDeposit is per call and does not move.
func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	info, ok := params.SwapInfo.(SwapInfo)
	if !ok || info.Shares == nil {
		return
	}
	amount, _ := uint256.FromBig(params.TokenAmountIn.Amount)
	// Copy-on-write so clones sharing the old pointers are unaffected.
	s.Supply = new(uint256.Int).Add(s.Supply, info.Shares)
	s.Valuation = new(uint256.Int).Add(s.Valuation, amount)
	s.TotalAssets = new(uint256.Int).Add(s.TotalAssets, amount)
	a, l := s.Reserves()
	s.Info.Reserves = []*big.Int{a.ToBig(), l.ToBig()}
}

// CloneState is shallow: UpdateBalance replaces fields wholesale and never writes through pointers.
func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return PoolMeta{BlockNumber: s.Info.BlockNumber}
}

func (s *PoolSimulator) CanSwapTo(token string) []string {
	if token == s.Info.Tokens[1] {
		return []string{s.Info.Tokens[0]}
	}
	return []string{}
}

func (s *PoolSimulator) CanSwapFrom(token string) []string {
	if token == s.Info.Tokens[0] {
		return []string{s.Info.Tokens[1]}
	}
	return []string{}
}

func gasOf(vault string) int64 {
	switch vault {
	case earnAUSDVault:
		return gasEarnAUSD
	case hlpeVault:
		return gasHLPe
	}
	return defaultGas
}
