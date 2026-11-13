package doppleramm

import (
	"math/big"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	v3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	extra       Extra
	staticExtra StaticExtra

	reserves [2]*uint256.Int

	// Launch Markets only; see launchSwap. bounds/nets are immutable and shared by clones.
	launch    bool
	bounds    []uint256.Int
	nets      []uint256.Int
	sqrtP     uint256.Int
	liquidity uint256.Int
}

var _ = pool.RegisterFactory(DexType, NewPoolSimulator)

func NewPoolSimulator(params pool.FactoryParams) (*PoolSimulator, error) {
	ep := params.EntityPool
	if len(ep.Tokens) != 2 || len(ep.Reserves) != 2 {
		return nil, ErrInvalidToken
	}

	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(ep.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var reserves [2]*uint256.Int
	for i, r := range ep.Reserves {
		v, err := uint256.FromDecimal(r)
		if err != nil {
			return nil, err
		}
		reserves[i] = v
	}

	s := &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     ep.Address,
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      lo.Map(ep.Tokens, func(t *entity.PoolToken, _ int) string { return t.Address }),
			Reserves:    lo.Map(ep.Reserves, func(s string, _ int) *big.Int { return bignum.NewBig(s) }),
			BlockNumber: ep.BlockNumber,
		}},
		extra:       extra,
		staticExtra: staticExtra,
		reserves:    reserves,
	}
	if staticExtra.Module != "" {
		if err := s.initLaunch(); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *PoolSimulator) initLaunch() error {
	levels, extra := s.staticExtra.Levels, &s.extra
	if len(levels) == 0 || len(levels) != len(extra.Nets) || extra.SqrtPriceX96 == nil || extra.Liquidity == nil ||
		extra.SwapFeePips >= feePipsDenominator {
		return ErrUnsupportedMarket
	}
	s.launch = true
	s.bounds = make([]uint256.Int, len(levels))
	s.nets = make([]uint256.Int, len(levels))
	for i, level := range levels {
		if err := v3.GetSqrtRatioAtTick(int(level), &s.bounds[i]); err != nil {
			return err
		}
		s.nets[i].Set((*uint256.Int)(extra.Nets[i]))
	}
	s.sqrtP.Set(extra.SqrtPriceX96)
	s.liquidity.Set(extra.Liquidity)
	return nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenOut)
	if indexIn < 0 || indexOut < 0 || indexIn == indexOut {
		return nil, ErrInvalidToken
	}

	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow || amountIn.IsZero() {
		return nil, ErrInvalidAmountIn
	}

	var amountOut uint256.Int
	var swapInfo any = s.extra
	if s.launch {
		var info Extra
		var err error
		if amountOut, info, err = s.launchOut(amountIn, indexIn == 0); err != nil {
			return nil, err
		}
		swapInfo = info
	} else {
		var err error
		if amountOut, err = getAmountOut(amountIn, s.reserves[indexIn], s.reserves[indexOut], s.extra.BaseFeeBps, s.extra.LpFeeBps); err != nil {
			return nil, err
		}
	}
	if !amountOut.Lt(s.reserves[indexOut]) || amountOut.IsZero() {
		return nil, ErrInsufficientLiquidity
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: params.TokenAmountIn.Token, Amount: bignum.ZeroBI},
		Gas:            defaultGas,
		// The encoder needs tokenX to tell which side the input is, and the
		// route summary carries SwapInfo but not the pool's Extra.
		SwapInfo: swapInfo,
	}, nil
}

// launchOut quotes a Launch Market swap. The returned Extra carries the post-trade
// price and liquidity for UpdateBalance; the level nets are left out of the route summary.
func (s *PoolSimulator) launchOut(amountIn *uint256.Int, zeroForOne bool) (uint256.Int, Extra, error) {
	var fee, eff, sqrtP, liquidity uint256.Int
	// the protocol base fee comes off the input first, floored
	fee.Mul(amountIn, uint256.NewInt(uint64(s.extra.BaseFeeBps))) // bounded: bps < 2^16
	fee.Div(&fee, bpsDenomU256)
	eff.Sub(amountIn, &fee)
	sqrtP.Set(&s.sqrtP)
	liquidity.Set(&s.liquidity)
	out, err := launchSwap(s.bounds, s.nets, s.extra.SwapFeePips, &sqrtP, &liquidity, &eff, zeroForOne)
	info := Extra{TokenX: s.extra.TokenX, BaseFeeBps: s.extra.BaseFeeBps, SwapFeePips: s.extra.SwapFeePips,
		SqrtPriceX96: &sqrtP, Liquidity: &liquidity}
	return out, info, err
}

// getAmountOut mirrors DopplerAMM's swap: the protocol base fee comes off the
// input first, then the module LP fee, each with its own floor, then constant
// product. Verified wei-exact against the deployed quoter on a fork.
func getAmountOut(
	amountIn, reserveIn, reserveOut *uint256.Int,
	baseFeeBps, lpFeeBps uint16,
) (uint256.Int, error) {
	var fee, effective, num, den, out uint256.Int

	var baseBps, lpBps uint256.Int
	baseBps.SetUint64(uint64(baseFeeBps))
	lpBps.SetUint64(uint64(lpFeeBps))

	if _, overflow := fee.MulOverflow(amountIn, &baseBps); overflow {
		return out, ErrOverflow
	}
	fee.Div(&fee, bpsDenomU256)
	effective.Sub(amountIn, &fee)

	if _, overflow := fee.MulOverflow(&effective, &lpBps); overflow {
		return out, ErrOverflow
	}
	fee.Div(&fee, bpsDenomU256)
	effective.Sub(&effective, &fee)

	if effective.IsZero() {
		return out, ErrInsufficientOutput
	}
	if _, overflow := num.MulOverflow(reserveOut, &effective); overflow {
		return out, ErrOverflow
	}
	den.Add(reserveIn, &effective)
	out.Div(&num, &den)
	return out, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return
	}

	amountIn, overflowIn := uint256.FromBig(params.TokenAmountIn.Amount)
	amountOut, overflowOut := uint256.FromBig(params.TokenAmountOut.Amount)
	if overflowIn || overflowOut {
		return
	}

	if info, ok := params.SwapInfo.(Extra); ok && s.launch && info.SqrtPriceX96 != nil && info.Liquidity != nil {
		s.sqrtP.Set(info.SqrtPriceX96)
		s.liquidity.Set(info.Liquidity)
	}

	s.reserves[indexIn].Add(s.reserves[indexIn], amountIn)
	if amountOut.Lt(s.reserves[indexOut]) {
		s.reserves[indexOut].Sub(s.reserves[indexOut], amountOut)
	} else {
		s.reserves[indexOut].Clear()
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.reserves = [2]*uint256.Int{new(uint256.Int).Set(s.reserves[0]), new(uint256.Int).Set(s.reserves[1])}
	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return pool.MetaInfo{ApprovalAddress: s.staticExtra.AMM, BlockNumber: s.Info.BlockNumber}
}
