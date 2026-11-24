package daikonprop

import (
	"math/big"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// PoolSimulator prices a Daikon pAMM pair. Tokens are [base, quote].
type PoolSimulator struct {
	pool.Pool
	Extra
	StaticExtra

	// staleCheck gates on the observation TTL. The liquidity indexer runs with it off.
	staleCheck bool
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
	if extra.BaseDecimals > 18 || extra.QuoteDecimals > 18 {
		return nil, ErrInvalidToken
	}
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(ep.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}
	extra.normalize()
	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     ep.Address,
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      lo.Map(ep.Tokens, func(t *entity.PoolToken, _ int) string { return t.Address }),
			Reserves:    lo.Map(ep.Reserves, func(s string, _ int) *big.Int { return bignumber.NewBig(s) }),
			BlockNumber: ep.BlockNumber,
		}},
		Extra:       extra,
		StaticExtra: staticExtra,
		staleCheck:  params.Opts.StaleCheck,
	}, nil
}

// CalcAmountOutWithFee is CalcAmtOutWithFee with big.Int amounts. Both share one quote body.
func (s *PoolSimulator) CalcAmountOutWithFee(params pool.CalcAmountOutParams, feePpm uint64) (*pool.CalcAmountOutResult, error) {
	if s.GetTokenIndex(params.TokenAmountIn.Token) < 0 || s.GetTokenIndex(params.TokenOut) < 0 {
		return nil, ErrInvalidToken
	}
	if params.TokenAmountIn.Amount == nil || params.TokenAmountIn.Amount.Sign() <= 0 {
		return nil, ErrZeroAmountIn
	}
	var amountIn uint256.Int
	if amountIn.SetFromBig(params.TokenAmountIn.Amount) {
		return nil, ErrOverflow
	}
	res, err := s.CalcAmtOutWithFee(pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{Token: params.TokenAmountIn.Token, Amount: amountIn},
		TokenOut:      params.TokenOut,
	}, feePpm)
	if err != nil {
		return nil, err
	}
	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: res.TokenAmountOut.Token, Amount: res.TokenAmountOut.Amount.ToBig()},
		Fee:            &pool.TokenAmount{Token: s.Info.Tokens[1], Amount: big.NewInt(0)},
		Gas:            res.Gas,
		SwapInfo:       res.SwapInfo,
	}, nil
}

// CalcAmtOut quotes in uint256 amounts at the static fee.
func (s *PoolSimulator) CalcAmtOut(params pool.CalcAmtOutParams) (*pool.CalcAmtOutResult, error) {
	return s.CalcAmtOutWithFee(params, s.FeePpm)
}

// CalcAmtOutWithFee is CalcAmtOut with a per-swap swapData fee. feePpm above MaxFeePpm is cut.
func (s *PoolSimulator) CalcAmtOutWithFee(params pool.CalcAmtOutParams, feePpm uint64) (*pool.CalcAmtOutResult, error) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenOut)
	if indexIn < 0 || indexOut < 0 || indexIn == indexOut {
		return nil, ErrInvalidToken
	}
	if params.TokenAmountIn.Amount.IsZero() {
		return nil, ErrZeroAmountIn
	}
	if s.Paused || s.ObservedAtMs == 0 {
		return nil, ErrNotTrading
	}
	// Mirrors the pair: live while observedAtMs/1000 + ttl >= block.timestamp.
	if s.staleCheck && s.ObservedAtMs/1000+s.TTL < uint64(time.Now().Unix()) {
		return nil, ErrExpired
	}

	buy := indexIn == 1 // quote in is BuyBase
	var amountOut uint256.Int
	var info SwapInfo
	if err := s.quoteExactIn(&params.TokenAmountIn.Amount, buy, feePpm, &amountOut, &info); err != nil {
		return nil, err
	}
	if amountOut.IsZero() {
		return nil, ErrZeroAmountOut
	}
	vault := s.VaultBase
	if !buy {
		vault = s.VaultQuote
	}
	if amountOut.Gt(vault) {
		return nil, ErrVaultInventory
	}

	return &pool.CalcAmtOutResult{
		TokenAmountOut: &pool.TokenAmt{Token: params.TokenOut, Amount: amountOut},
		Fee:            &pool.TokenAmt{Token: s.Info.Tokens[1]},
		Gas:            lo.Ternary(buy, gasBuyBase, gasSellBase),
		SwapInfo:       info,
	}, nil
}

// UpdateBalance is UpdateState with big.Int amounts.
func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	var out uint256.Int
	out.SetFromBig(params.TokenAmountOut.Amount)
	s.UpdateState(pool.UpdateStateParams{TokenAmountOut: pool.TokenAmt{Amount: out}, SwapInfo: params.SwapInfo})
}

// UpdateState spends the capacity tiers and custody inventory the swap consumed.
func (s *PoolSimulator) UpdateState(params pool.UpdateStateParams) {
	info, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		return
	}
	c1, c2, vault, vaultIdx := s.SellC1, s.SellC2, s.VaultQuote, 1
	if info.BuyBase {
		c1, c2, vault, vaultIdx = s.BuyC1, s.BuyC2, s.VaultBase, 0
	}
	c1.Sub(c1, info.C1Used)
	c2.Sub(c2, info.C2Used)
	vault.Sub(vault, &params.TokenAmountOut.Amount)

	s.Info.Reserves = lo.Map(s.Info.Reserves, func(r *big.Int, _ int) *big.Int { return new(big.Int).Set(r) })
	s.Info.Reserves[vaultIdx].Sub(s.Info.Reserves[vaultIdx], params.TokenAmountOut.Amount.ToBig())
}

// CloneState deep-copies the capacity and inventory ints UpdateState writes in place.
func (s *PoolSimulator) CloneState() pool.IPoolSim {
	cloned := *s
	cloned.BuyC1, cloned.BuyC2 = s.BuyC1.Clone(), s.BuyC2.Clone()
	cloned.SellC1, cloned.SellC2 = s.SellC1.Clone(), s.SellC2.Clone()
	cloned.VaultBase, cloned.VaultQuote = s.VaultBase.Clone(), s.VaultQuote.Clone()
	return &cloned
}

// GetReserves shadows pool.Pool.GetReserves, which returns big.Int.
func (s *PoolSimulator) GetReserves() []uint256.Int {
	out := make([]uint256.Int, len(s.Info.Reserves))
	for i, r := range s.Info.Reserves {
		out[i].SetFromBig(r)
	}
	return out
}

// CalcLimit returns nil: daikon-prop has no swap limit, as pool.Pool.CalculateLimit.
func (s *PoolSimulator) CalcLimit() map[string]*uint256.Int {
	return nil
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return MetaInfo{
		Router:          s.Router,
		ApprovalAddress: s.Router,
		PartnerID:       s.PartnerID,
		FeePpm:          min(s.FeePpm, MaxFeePpm),
		BlockNumber:     s.Info.BlockNumber,
	}
}
