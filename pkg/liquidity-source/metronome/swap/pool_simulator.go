package metronomeswap

import (
	"math/big"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	bignumber "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// defaultGas is the gasUsed of a real swap() execution on a Tenderly vnet fork (chain head +1,
// msETH->msUSD, tx 0x33d20226a9ae4e35ae4649372c23d0e6e949d35713b25e63e24a8aebc0a50608 on that
// vnet) during dex-verify — replaces an earlier unverified 150000 placeholder from the math
// stage, which undershot the real cost of two mint()s (fee + net) plus the burn and the
// fee/oracle/cap reads.
const defaultGas = 262490

type PoolSimulator struct {
	pool.Pool
	Extra    Extra
	Decimals []uint8
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(p entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return nil, err
	}

	tokens := lo.Map(p.Tokens, func(e *entity.PoolToken, _ int) string { return e.Address })
	reserves := lo.Map(p.Reserves, func(e string, _ int) *big.Int { return bignumber.NewBig(e) })
	decimals := lo.Map(p.Tokens, func(e *entity.PoolToken, _ int) uint8 { return e.Decimals })

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     p.Address,
			Exchange:    p.Exchange,
			Type:        p.Type,
			Tokens:      tokens,
			Reserves:    reserves,
			BlockNumber: p.BlockNumber,
		}},
		Extra:    extra,
		Decimals: decimals,
	}, nil
}

// CalcAmountOut mirrors Pool.quoteSwapOut / Pool.swap: a pure oracle-rate conversion
// (MasterOracle.quote) minus a static per-ordered-pair fee taken from the gross output.
// There is no bonding curve and no pooled reserves — the only capacity constraint is
// syntheticTokenOut's maxTotalSupply headroom.
func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn, tokenOut := params.TokenAmountIn.Token, params.TokenOut
	if tokenIn == tokenOut {
		return nil, ErrSameToken
	}

	indexIn, indexOut := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	if !s.Extra.SwapActive {
		return nil, ErrSwapInactive
	}

	stateIn, ok := s.Extra.Tokens[tokenIn]
	if !ok || !stateIn.IsActive {
		return nil, ErrTokenInactive
	}
	stateOut, ok := s.Extra.Tokens[tokenOut]
	if !ok || !stateOut.IsActive {
		return nil, ErrTokenInactive
	}

	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow || amountIn.Sign() <= 0 {
		return nil, ErrInvalidAmountIn
	}

	grossAmountOut := quote(amountIn, s.Decimals[indexIn], stateIn.PriceInUsd, s.Decimals[indexOut], stateOut.PriceInUsd)

	feeBps := s.Extra.SwapFeesBps[tokenIn+"-"+tokenOut]
	if feeBps == nil {
		feeBps = uint256.NewInt(0)
	}
	fee := swapFee(grossAmountOut, feeBps)

	if fee.Gt(grossAmountOut) {
		return nil, ErrZeroAmountOut
	}
	amountOut := new(uint256.Int).Sub(grossAmountOut, fee)
	if amountOut.Sign() <= 0 {
		return nil, ErrZeroAmountOut
	}

	// Pool.swap() mints TWICE on the output token — once to the fee collector (fee), once to
	// the caller (net) — and SyntheticToken._mint checks totalSupply<=maxTotalSupply after
	// EACH mint. The binding constraint is the second (larger) check, equivalent to gating on
	// the FULL gross amount, not just the net amount the trader receives. Verified against a
	// real vnet swap(): totalSupply(msUSD) increased by the gross amount, with two separate
	// Transfer(0x0, ...) mint events (one to the fee collector, one to the caller) summing to
	// it — confirmed the cap isn't just against the trader-visible net output.
	newTotalSupplyOut := new(uint256.Int).Add(stateOut.TotalSupply, grossAmountOut)
	if newTotalSupplyOut.Gt(stateOut.MaxTotalSupply) {
		return nil, ErrExceedsMaxTotalSupply
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: tokenOut, Amount: fee.ToBig()},
		Gas:            defaultGas,
	}, nil
}

// UpdateBalance mirrors the on-chain burn(tokenIn) / mint(feeCollector, fee) / mint(caller, net)
// sequence and keeps Info.Reserves (used here as remaining mint headroom, not AMM reserves — see
// dex-explorer's shared_vault notes) in sync for router-service consumers that read it directly.
//
// totalSupply(tokenOut) grows by the FULL gross amount (net+fee) — the fee portion is minted to
// the fee collector, not burned or skipped, confirmed via two separate Transfer(0x0, ...) mint
// events on a real vnet swap(). Using params.TokenAmountOut.Amount (net only) here would
// under-count totalSupply and silently let CalcAmountOut over-quote against the mint cap on
// subsequent swaps.
func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	tokenIn, tokenOut := params.TokenAmountIn.Token, params.TokenAmountOut.Token
	amountIn := uint256.MustFromBig(params.TokenAmountIn.Amount)
	netAmountOut := uint256.MustFromBig(params.TokenAmountOut.Amount)
	fee := uint256.MustFromBig(params.Fee.Amount)
	grossAmountOut := new(uint256.Int).Add(netAmountOut, fee)

	stateIn := s.Extra.Tokens[tokenIn]
	stateIn.TotalSupply = new(uint256.Int).Sub(stateIn.TotalSupply, amountIn)
	s.Extra.Tokens[tokenIn] = stateIn

	stateOut := s.Extra.Tokens[tokenOut]
	stateOut.TotalSupply = new(uint256.Int).Add(stateOut.TotalSupply, grossAmountOut)
	s.Extra.Tokens[tokenOut] = stateOut

	if indexIn := s.GetTokenIndex(tokenIn); indexIn >= 0 {
		s.Info.Reserves[indexIn] = headroom(stateIn.MaxTotalSupply, stateIn.TotalSupply).ToBig()
	}
	if indexOut := s.GetTokenIndex(tokenOut); indexOut >= 0 {
		s.Info.Reserves[indexOut] = headroom(stateOut.MaxTotalSupply, stateOut.TotalSupply).ToBig()
	}
}

// headroom returns max(0, maxTotalSupply-totalSupply) — plain Sub would underflow-wrap a
// uint256 if totalSupply ever exceeds maxTotalSupply (stale data race), so guard explicitly.
func headroom(maxTotalSupply, totalSupply *uint256.Int) *uint256.Int {
	if maxTotalSupply.Lt(totalSupply) {
		return uint256.NewInt(0)
	}
	return new(uint256.Int).Sub(maxTotalSupply, totalSupply)
}

// CanSwapTo/CanSwapFrom: complete-graph topology — every token in a Metronome pool swaps
// to every other token in the same pool (unlike e.g. angle-transmuter's star topology).
func (s *PoolSimulator) CanSwapTo(address string) []string {
	if s.GetTokenIndex(address) < 0 {
		return nil
	}
	return lo.Filter(s.Info.Tokens, func(token string, _ int) bool { return token != address })
}

func (s *PoolSimulator) CanSwapFrom(address string) []string {
	return s.CanSwapTo(address)
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	// swap() pulls syntheticTokenIn_ directly on the Pool contract itself — no router,
	// no separate approval target.
	return nil
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.Info.Reserves = lo.Map(s.Info.Reserves, func(r *big.Int, _ int) *big.Int { return new(big.Int).Set(r) })

	cloned.Extra.Tokens = make(map[string]TokenState, len(s.Extra.Tokens))
	for token, state := range s.Extra.Tokens {
		cloned.Extra.Tokens[token] = TokenState{
			IsActive:       state.IsActive,
			MaxTotalSupply: new(uint256.Int).Set(state.MaxTotalSupply),
			TotalSupply:    new(uint256.Int).Set(state.TotalSupply),
			PriceInUsd:     new(uint256.Int).Set(state.PriceInUsd),
		}
	}

	cloned.Extra.SwapFeesBps = make(map[string]*uint256.Int, len(s.Extra.SwapFeesBps))
	for pair, feeBps := range s.Extra.SwapFeesBps {
		cloned.Extra.SwapFeesBps[pair] = new(uint256.Int).Set(feeBps)
	}

	return &cloned
}
