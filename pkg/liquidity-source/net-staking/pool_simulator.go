package netstaking

import (
	"math/big"
	"strings"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// PoolSimulator implements pool.IPoolSimulator for staking/wrapping protocols shaped like
// NET/sNET/wsNET. It has two modes, selected by whether the pool config has a wrap
// contract:
//
//   - With wrap (hasWrap==true, 3 tokens): supports five swap directions between base
//     (NET), staked (sNET), and wrapped (wsNET), spanning two on-chain contracts (Staking
//     and WrappedStakedNET). There is no wsNET->NET direction: the executor helper
//     (INetStaking.NetAction) has no reverse composite action, and pathfinder-lib forbids
//     reusing the same pool twice in a route, so no 2-hop workaround exists either.
//     wsNET->NET is unsupported by design.
//   - Without wrap (hasWrap==false, 2 tokens): only the base<->staked 1:1 leg
//     (Stake/Unstake) is offered. Wrap/Unwrap/StakeAndWrap never appear, and the wsNET
//     token slot does not exist at all.
//
// Pricing:
//   - NET <-> sNET: 1:1 (Staking.stake/unstake, warmupEpochs()==0 so atomic)
//   - sNET -> wsNET: sNetToWs(amount, index) = amount * 1e18 / index
//   - wsNET -> sNET: wsToSNet(amount, index) = amount * index / 1e18
//   - NET -> wsNET: composite, same math as chaining the two legs above
//
// Binding constraints (enforced per CalcAmountOut):
//   - NET->sNET:   amountIn <= SNETStakingReserve (staking must hold enough sNET to pay out)
//   - sNET->NET:   amountIn <= NETReserve (staking must hold enough NET to pay out)
//   - sNET->wsNET: no cap (wrap has no on-chain supply ceiling)
//   - wsNET->sNET: wsToSNet(amountIn) <= SNETWrapReserve (wrap must hold enough sNET to pay out)
//   - NET->wsNET:  amountIn <= SNETStakingReserve (stake leg only; wrap leg uncapped)
//
// Token positions in pool.Info.Tokens — set by the lister, never reordered.
const (
	idxNET   = 0
	idxSNET  = 1
	idxWSNET = 2
)

type PoolSimulator struct {
	pool.Pool

	// hasWrap is true when the pool has a wrap contract (3 tokens: base/staked/wrapped),
	// false when it only offers the base<->staked 1:1 leg (2 tokens).
	hasWrap bool

	index *uint256.Int

	// netReserve is NET.balanceOf(staking); sNetStakingReserve is sNET.balanceOf(staking);
	// sNetWrapReserve is sNET.balanceOf(wrap). All are fetched by the tracker each refresh
	// cycle and used to enforce binding liquidity caps in CalcAmountOut.
	netReserve         *uint256.Int
	sNetStakingReserve *uint256.Int
	sNetWrapReserve    *uint256.Int

	// scratch and scratchIn are pre-allocated uint256.Ints reused across CalcAmountOut
	// calls to avoid per-call heap allocations. scratchIn holds the parsed amountIn;
	// scratch holds the intermediate/final result. Neither must escape across goroutines
	// or be shared between a simulator and its clone.
	scratch   uint256.Int
	scratchIn uint256.Int
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	var extra PoolExtra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}

	// Resolve reserve fields. Pools created before the tracker's first refresh will have
	// nil reserves in their serialized Extra. Use zero as a safe sentinel — the cap check
	// treats zero-reserve as "no liquidity available" which correctly blocks routing until
	// the tracker refreshes with real values.
	netReserve := extra.NETReserve
	if netReserve == nil {
		netReserve = new(uint256.Int)
	}
	sNetStakingReserve := extra.SNETStakingReserve
	if sNetStakingReserve == nil {
		sNetStakingReserve = new(uint256.Int)
	}

	hasWrap := len(ep.Tokens) == 3

	// Only trust the wrap-side fields when the pool actually has a wrap contract; they're
	// never read when !hasWrap (ActionWrap/ActionUnwrap/ActionStakeAndWrap can never be
	// selected), so a nil-safe zero default is fine.
	var (
		index           *uint256.Int
		sNetWrapReserve = new(uint256.Int)
	)
	if hasWrap {
		index = extra.Index
		if extra.SNETWrapReserve != nil {
			sNetWrapReserve = extra.SNETWrapReserve
		}
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     strings.ToLower(ep.Address),
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      lo.Map(ep.Tokens, func(item *entity.PoolToken, _ int) string { return item.Address }),
			Reserves:    lo.Map(ep.Reserves, func(item string, _ int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: ep.BlockNumber,
		}},
		hasWrap:            hasWrap,
		index:              index,
		netReserve:         netReserve,
		sNetStakingReserve: sNetStakingReserve,
		sNetWrapReserve:    sNetWrapReserve,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn := strings.ToLower(params.TokenAmountIn.Token)
	tokenOut := strings.ToLower(params.TokenOut)
	// Use scratchIn to parse amountIn without a heap allocation.
	// SetFromBig returns true on overflow (amount > 2^256-1), which cannot occur for
	// realistic token amounts; treat it the same as invalid input.
	if overflow := s.scratchIn.SetFromBig(params.TokenAmountIn.Amount); overflow {
		return nil, ErrInvalidTokenIn
	}
	amountIn := &s.scratchIn

	if amountIn.IsZero() {
		return nil, ErrZeroAmount
	}

	action, err := ActionFor(tokenIn, tokenOut, s.Info.Tokens[idxNET], s.Info.Tokens[idxSNET], s.wsNetAddr())
	if err != nil {
		return nil, err
	}

	var gasEstimate int64

	switch action {
	case ActionStake:
		// NET->sNET (1:1): cap on outbound sNET.
		if amountIn.Gt(s.sNetStakingReserve) {
			return nil, ErrInsufficientLiquidity
		}
		s.scratch.Set(amountIn)
		gasEstimate = dfGas.Stake
	case ActionUnstake:
		// sNET->NET (1:1): cap on outbound NET.
		if amountIn.Gt(s.netReserve) {
			return nil, ErrInsufficientLiquidity
		}
		s.scratch.Set(amountIn)
		gasEstimate = dfGas.Unstake
	case ActionWrap:
		// sNET->wsNET: no cap (wrap has no on-chain supply ceiling).
		if _, err := sNetToWs(&s.scratch, amountIn, s.index); err != nil {
			return nil, err
		}
		gasEstimate = dfGas.Wrap
	case ActionUnwrap:
		// wsNET->sNET: cap on outbound sNET (wrap's own balance).
		if _, err := wsToSNet(&s.scratch, amountIn, s.index); err != nil {
			return nil, err
		}
		if s.scratch.Gt(s.sNetWrapReserve) {
			return nil, ErrInsufficientLiquidity
		}
		gasEstimate = dfGas.Unwrap
	case ActionStakeAndWrap:
		// NET->wsNET composite: stake leg (1:1, capped by SNETStakingReserve),
		// then wrap leg (ratio, uncapped).
		if amountIn.Gt(s.sNetStakingReserve) {
			return nil, ErrInsufficientLiquidity
		}
		if _, err := sNetToWs(&s.scratch, amountIn, s.index); err != nil {
			return nil, err
		}
		gasEstimate = dfGas.StakeAndWrap
	default:
		return nil, ErrInvalidTokenIn
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: s.scratch.ToBig()},
		Fee:            &pool.TokenAmount{Token: params.TokenOut, Amount: bignumber.ZeroBI},
		Gas:            gasEstimate,
		SwapInfo:       SwapInfo{Action: action},
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	info, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		return
	}
	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow {
		return
	}
	amountOut, overflow := uint256.FromBig(params.TokenAmountOut.Amount)
	if overflow {
		return
	}

	switch info.Action {
	case ActionStake: // NET in, sNET out (1:1) — staking's inventory shifts.
		s.netReserve.Add(s.netReserve, amountIn)
		s.sNetStakingReserve.Sub(s.sNetStakingReserve, amountOut)
	case ActionUnstake: // sNET in, NET out (1:1).
		s.sNetStakingReserve.Add(s.sNetStakingReserve, amountIn)
		s.netReserve.Sub(s.netReserve, amountOut)
	case ActionWrap: // sNET in (held by wrap), wsNET out (minted externally).
		s.sNetWrapReserve.Add(s.sNetWrapReserve, amountIn)
	case ActionUnwrap: // wsNET in (burned externally), sNET out (from wrap's balance).
		s.sNetWrapReserve.Sub(s.sNetWrapReserve, amountOut)
	case ActionStakeAndWrap: // NET in to staking; intermediate sNET (== amountIn, 1:1 stake
		// leg) moves from staking's inventory into wrap's inventory; wsNET out (minted).
		s.netReserve.Add(s.netReserve, amountIn)
		s.sNetStakingReserve.Sub(s.sNetStakingReserve, amountIn)
		s.sNetWrapReserve.Add(s.sNetWrapReserve, amountIn)
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	if s.index != nil {
		cloned.index = new(uint256.Int).Set(s.index)
	}
	if s.netReserve != nil {
		cloned.netReserve = new(uint256.Int).Set(s.netReserve)
	}
	if s.sNetStakingReserve != nil {
		cloned.sNetStakingReserve = new(uint256.Int).Set(s.sNetStakingReserve)
	}
	if s.sNetWrapReserve != nil {
		cloned.sNetWrapReserve = new(uint256.Int).Set(s.sNetWrapReserve)
	}
	if len(s.Info.Reserves) > 0 {
		cloned.Info.Reserves = make([]*big.Int, len(s.Info.Reserves))
		for i, r := range s.Info.Reserves {
			cloned.Info.Reserves[i] = new(big.Int).Set(r)
		}
	}
	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) any {
	return PoolMeta{
		BlockNumber:     s.Info.BlockNumber,
		NET:             s.Info.Tokens[idxNET],
		SNET:            s.Info.Tokens[idxSNET],
		WSNET:           s.wsNetAddr(),
		ApprovalAddress: s.GetApprovalAddress(tokenIn, tokenOut),
	}
}

// wsNetAddr returns the wrapped-token address, or "" when the pool has no wrap contract.
func (s *PoolSimulator) wsNetAddr() string {
	if s.hasWrap {
		return s.Info.Tokens[idxWSNET]
	}
	return ""
}

// GetApprovalAddress returns the first-hop contract the caller must approve tokenIn to:
// the wsNET/wrap contract for ActionWrap (sNET->wsNET) only. Every other action —
// including Unwrap, which burns the caller's own wsNET with no transferFrom and needs
// no allowance at all — approves the pool address (Staking), matching
// ExecutorV3Helper9.executeNetStaking's spender selection.
func (s *PoolSimulator) GetApprovalAddress(tokenIn, tokenOut string) string {
	action, err := ActionFor(strings.ToLower(tokenIn), strings.ToLower(tokenOut),
		s.Info.Tokens[idxNET], s.Info.Tokens[idxSNET], s.wsNetAddr())
	if err != nil {
		return s.Info.Address
	}
	switch action {
	case ActionWrap:
		return s.Info.Tokens[idxWSNET]
	default: // ActionStake, ActionUnstake, ActionUnwrap, ActionStakeAndWrap
		return s.Info.Address
	}
}

// CanSwapTo returns the tokens that can swap to address. wsNET has no path to NET:
// the executor helper has no reverse composite action. When the pool has no wrap
// contract (s.hasWrap==false), wsNET entries are omitted entirely rather than appending
// an empty-string placeholder — pathfinder must never see a "" token.
func (s *PoolSimulator) CanSwapTo(address string) []string {
	net, sNet := s.Info.Tokens[idxNET], s.Info.Tokens[idxSNET]
	switch address {
	case net:
		return []string{sNet}
	case sNet:
		if s.hasWrap {
			return []string{net, s.Info.Tokens[idxWSNET]}
		}
		return []string{net}
	}
	if s.hasWrap && address == s.Info.Tokens[idxWSNET] {
		return []string{net, sNet}
	}
	return nil
}

// CanSwapFrom returns the tokens reachable by swapping from address. NET->wsNET
// (composite stake+wrap) is fine, but wsNET->NET is not: see CanSwapTo. When the pool has
// no wrap contract, wsNET entries are omitted entirely (see CanSwapTo).
func (s *PoolSimulator) CanSwapFrom(address string) []string {
	net, sNet := s.Info.Tokens[idxNET], s.Info.Tokens[idxSNET]
	switch address {
	case net:
		if s.hasWrap {
			return []string{sNet, s.Info.Tokens[idxWSNET]}
		}
		return []string{sNet}
	case sNet:
		if s.hasWrap {
			return []string{net, s.Info.Tokens[idxWSNET]}
		}
		return []string{net}
	}
	if s.hasWrap && address == s.Info.Tokens[idxWSNET] {
		return []string{sNet}
	}
	return nil
}

func ActionFor(tokenIn, tokenOut, net, sNet, wsNet string) (Action, error) {
	switch tokenIn {
	case net:
		switch tokenOut {
		case sNet:
			return ActionStake, nil
		case wsNet:
			return ActionStakeAndWrap, nil
		}
		return 0, ErrInvalidTokenOut
	case sNet:
		switch tokenOut {
		case net:
			return ActionUnstake, nil
		case wsNet:
			return ActionWrap, nil
		}
		return 0, ErrInvalidTokenOut
	case wsNet:
		switch tokenOut {
		case sNet:
			return ActionUnwrap, nil
		}
		return 0, ErrInvalidTokenOut
	}
	return 0, ErrInvalidTokenIn
}
