package midas

import (
	"math/big"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

type PoolSimulator struct {
	pool.Pool

	dv IDepositVault
	rv IRedemptionVault

	isDv      bool
	vaultType VaultType
}

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(ep.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var vault VaultState
	if err := json.Unmarshal([]byte(ep.Extra), &vault); err != nil {
		return nil, err
	}

	tokenDecimalsMap := lo.SliceToMap(ep.Tokens, func(item *entity.PoolToken) (string, uint8) {
		return item.Address, item.Decimals
	})
	dv, rv, err := newVault(&vault, staticExtra.VaultType, tokenDecimalsMap)
	if err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     ep.Address,
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      lo.Map(ep.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves:    lo.Map(ep.Reserves, func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: ep.BlockNumber,
		}},
		isDv:      staticExtra.IsDv,
		vaultType: staticExtra.VaultType,
		dv:        dv,
		rv:        rv,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	var (
		amountIn = uint256.MustFromBig(params.TokenAmountIn.Amount)
		tokenIn  = params.TokenAmountIn.Token
		tokenOut = params.TokenOut
	)
	indexIn, indexOut := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	var (
		swapInfo *SwapInfo
		err      error
	)

	isDeposit := indexIn != 0
	if isDeposit && s.isDv {
		swapInfo, err = s.dv.DepositInstant(amountIn, tokenIn, tokenOut)
	} else if !isDeposit && !s.isDv {
		swapInfo, err = s.rv.RedeemInstant(amountIn, tokenIn, tokenOut)
	} else {
		return nil, ErrInvalidSwap
	}
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  s.Info.Tokens[indexOut],
			Amount: swapInfo.amountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  s.Info.Tokens[indexIn],
			Amount: swapInfo.fee.ToBig(),
		},
		Gas:      swapInfo.gas,
		SwapInfo: swapInfo,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swapInfo := params.SwapInfo.(*SwapInfo)
	tokenIn := params.TokenAmountIn.Token
	tokenOut := params.TokenAmountOut.Token
	if swapInfo.IsDeposit {
		s.dv.UpdateState(swapInfo, tokenIn)
	} else {
		s.rv.UpdateState(swapInfo, tokenOut)
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	if s.dv != nil {
		cloned.dv = s.dv.CloneState().(IDepositVault)
	}
	if s.rv != nil {
		cloned.rv = s.rv.CloneState().(IRedemptionVault)
	}

	return &cloned
}

func (p *PoolSimulator) CanSwapTo(address string) []string {
	tokenIndex := p.GetTokenIndex(address)
	if p.isDv && tokenIndex == 0 {
		return p.Info.Tokens[1:]
	} else if !p.isDv && tokenIndex != 0 {
		return []string{p.Info.Tokens[0]}
	}

	return []string{}
}

func (p *PoolSimulator) CanSwapFrom(address string) []string {
	tokenIndex := p.GetTokenIndex(address)
	if p.isDv && tokenIndex != 0 {
		return []string{p.Info.Tokens[0]}
	} else if !p.isDv && tokenIndex == 0 {
		return p.Info.Tokens[1:]
	}

	return []string{}
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return Meta{
		BlockNumber: s.Info.BlockNumber,
	}
}
