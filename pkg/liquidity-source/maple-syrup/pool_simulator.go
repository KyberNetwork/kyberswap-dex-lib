package maplesyrup

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/erc4626"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
)

type PoolSimulator struct {
	erc4626.PoolSimulator
	router       string
	active       bool
	liquidityCap *uint256.Int
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(p entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return nil, err
	}

	erc4626PoolSimulator, err := erc4626.NewPoolSimulator(p)
	if err != nil {
		return nil, err
	}

	return &PoolSimulator{
		PoolSimulator: *erc4626PoolSimulator,
		router:        extra.Router,
		active:        extra.Active,
		liquidityCap:  extra.LiquidityCap,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if !s.active {
		return nil, ErrNotActive
	}

	amountIn := uint256.MustFromBig(params.TokenAmountIn.Amount)
	if new(uint256.Int).Add(amountIn, s.PoolSimulator.TotalAssets).Gt(s.liquidityCap) {
		return nil, ErrDepositGtLiqCap
	}

	return s.PoolSimulator.CalcAmountOut(params)
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.MaxDeposit = new(uint256.Int).Set(s.MaxDeposit)
	cloned.TotalAssets = new(uint256.Int).Set(s.TotalAssets)
	return &cloned
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	s.PoolSimulator.UpdateBalance(params)
	s.TotalAssets = new(uint256.Int).Add(s.TotalAssets, uint256.MustFromBig(params.TokenAmountIn.Amount))
}

func (s *PoolSimulator) GetApprovalAddress(_, _ string) string {
	return s.router
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return Meta{
		BlockNumber: s.Info.BlockNumber,
		Router:      s.router,
	}
}
