package flowstatec1

import (
	"errors"
	"math/big"
	"strings"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	StaticExtra
	Extra
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

var (
	ErrPoolNotAvailable      = errors.New("pool is currently not available")
	ErrInsufficientLiquidity = errors.New("amount exceeds available tokens")
	ErrInvalidProbe          = errors.New("invalid probe rate")
	ErrZeroAmountOut         = errors.New("amountOut is zero")
)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	info := pool.PoolInfo{
		Address:     strings.ToLower(entityPool.Address),
		Exchange:    entityPool.Exchange,
		Type:        entityPool.Type,
		Tokens:      []string{entityPool.Tokens[0].Address, entityPool.Tokens[1].Address},
		Reserves:    []*big.Int{bignumber.NewBig10(entityPool.Reserves[0]), bignumber.NewBig10(entityPool.Reserves[1])},
		BlockNumber: entityPool.BlockNumber,
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool:        pool.Pool{Info: info},
		StaticExtra: staticExtra,
		Extra:       extra,
	}, nil
}

// CanSwapFrom: buy-only, quote asset in (token[0]), inventory token out (token[1]).
func (s *PoolSimulator) CanSwapFrom(token string) []string {
	if token == s.Info.Tokens[0] {
		return []string{s.Info.Tokens[1]}
	}

	return nil
}

func (s *PoolSimulator) CanSwapTo(token string) []string {
	if token == s.Info.Tokens[1] {
		return []string{s.Info.Tokens[0]}
	}

	return nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if !s.Available {
		return nil, ErrPoolNotAvailable
	}

	if s.ProbeQuoteCost.IsZero() || s.ProbeAmount.IsZero() {
		return nil, ErrInvalidProbe
	}

	amountIn, overflow := uint256.FromBig(param.TokenAmountIn.Amount)
	if overflow {
		return nil, ErrInsufficientLiquidity
	}

	// No curve price impact up to FillableAmount (vendor spec 3.1): the unit rate
	// sampled at ProbeAmount holds for any size, so scale linearly.
	var amountOut uint256.Int
	big256.MulDivDown(&amountOut, amountIn, s.ProbeAmount, s.ProbeQuoteCost)

	if amountOut.IsZero() {
		return nil, ErrZeroAmountOut
	}

	if amountOut.Gt(s.FillableAmount) {
		return nil, ErrInsufficientLiquidity
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  s.Info.Tokens[1],
			Amount: amountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  s.Info.Tokens[0],
			Amount: nil,
		},
		Gas: defaultGas,
	}, nil
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	return &cloned
}

// UpdateBalance reassigns FillableAmount to a new *uint256.Int rather than mutating
// the existing one in place: CloneState is a shallow copy, and a clone's pointer
// would otherwise alias the original's.
func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	amountOut, overflow := uint256.FromBig(params.TokenAmountOut.Amount)
	if overflow {
		return
	}
	s.FillableAmount = new(uint256.Int).Sub(s.FillableAmount, amountOut)
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return MetaInfo{
		Market:      s.StaticExtra.Market,
		Pool:        s.StaticExtra.Pool,
		QuoteAsset:  s.StaticExtra.QuoteAsset,
		BlockNumber: s.Info.BlockNumber,
	}
}
