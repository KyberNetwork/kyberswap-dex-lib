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

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	return s.CalcAmountOutWithFee(params, s.FeePpm)
}

// CalcAmountOutWithFee quotes with a per-swap swapData fee instead of the static one, which is
// how the RFQ handler prices the alpha fee. feePpm above MaxFeePpm is cut, as the pair does.
func (s *PoolSimulator) CalcAmountOutWithFee(params pool.CalcAmountOutParams, feePpm uint64) (*pool.CalcAmountOutResult, error) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenOut)
	if indexIn < 0 || indexOut < 0 || indexIn == indexOut {
		return nil, ErrInvalidToken
	}
	if params.TokenAmountIn.Amount.Sign() <= 0 {
		return nil, ErrZeroAmountIn
	}
	var amountIn uint256.Int
	if amountIn.SetFromBig(params.TokenAmountIn.Amount) {
		return nil, ErrOverflow
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
	if err := s.quoteExactIn(&amountIn, buy, feePpm, &amountOut, &info); err != nil {
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

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: s.Info.Tokens[1], Amount: big.NewInt(0)},
		Gas:            lo.Ternary(buy, gasBuyBase, gasSellBase),
		SwapInfo:       info,
	}, nil
}

// UpdateBalance spends the capacity tiers and custody inventory the swap consumed.
func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	info, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		return
	}
	var out uint256.Int
	out.SetFromBig(params.TokenAmountOut.Amount)
	c1, c2, vault, vaultIdx := s.SellC1, s.SellC2, s.VaultQuote, 1
	if info.BuyBase {
		c1, c2, vault, vaultIdx = s.BuyC1, s.BuyC2, s.VaultBase, 0
	}
	c1.Sub(c1, info.C1Used)
	c2.Sub(c2, info.C2Used)
	vault.Sub(vault, &out)

	s.Info.Reserves = lo.Map(s.Info.Reserves, func(r *big.Int, _ int) *big.Int { return new(big.Int).Set(r) })
	s.Info.Reserves[vaultIdx].Sub(s.Info.Reserves[vaultIdx], params.TokenAmountOut.Amount)
}

// CloneState deep-copies the capacity and inventory ints UpdateBalance writes in place.
func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.BuyC1, cloned.BuyC2 = s.BuyC1.Clone(), s.BuyC2.Clone()
	cloned.SellC1, cloned.SellC2 = s.SellC1.Clone(), s.SellC2.Clone()
	cloned.VaultBase, cloned.VaultQuote = s.VaultBase.Clone(), s.VaultQuote.Clone()
	return &cloned
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
