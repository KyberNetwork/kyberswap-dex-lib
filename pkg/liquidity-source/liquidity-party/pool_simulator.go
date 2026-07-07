package liquidityparty

import (
	"math/big"

	"github.com/KyberNetwork/int256"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	abdk "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abdkmath64x64"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// PoolSimulator prices Liquidity Party (LMSR) swaps purely in-memory, reproducing the on-chain swap
// kernel to the wei. Only the swap kernel is modeled — mint/burn/swapMint/burnSwap are LP ops the
// aggregator does not route. Pricing is driven solely by the LMSR internal state (kappa,
// effectiveSigmaQ, qInternal); b = κ·effectiveSigmaQ is block-frozen so effectiveSigmaQ stays
// constant across every quote/update within a routing pass.
type PoolSimulator struct {
	pool.Pool

	kappa           *int256.Int   // κ, Q64.64 (immutable within a block)
	effectiveSigmaQ *int256.Int   // effectiveSigmaQ, Q64.64 (block-frozen)
	qInternal       []*int256.Int // per-token internal balances, Q64.64 (mutated by UpdateBalance)
	bases           []*uint256.Int
	feesPpm         []uint64
	killed          bool
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

// estimateGas models a single swap's gas as a linear function of the pool's token count
// (swapBaseGas + swapGasPerToken·nTokens); swap() recomputes the size metric across every token, so
// gas grows with pool width. See the constants in constant.go for the fit against measured traces.
func (p *PoolSimulator) estimateGas() int64 {
	return swapBaseGas + swapGasPerToken*int64(len(p.qInternal))
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	n := len(entityPool.Tokens)
	if extra.Kappa == nil || extra.EffectiveSigmaQ == nil ||
		len(extra.QInternal) != n || len(extra.Bases) != n || len(extra.FeesPpm) != n {
		return nil, ErrInvalidExtra
	}

	kappa, err := int256.FromBig(extra.Kappa)
	if err != nil {
		return nil, err
	}
	effectiveSigmaQ, err := int256.FromBig(extra.EffectiveSigmaQ)
	if err != nil {
		return nil, err
	}

	qInternal := make([]*int256.Int, n)
	bases := make([]*uint256.Int, n)
	for i := 0; i < n; i++ {
		if extra.QInternal[i] == nil || extra.Bases[i] == nil {
			return nil, ErrInvalidExtra
		}
		q, err := int256.FromBig(extra.QInternal[i])
		if err != nil {
			return nil, err
		}
		base, overflow := uint256.FromBig(extra.Bases[i])
		if overflow {
			return nil, ErrOverflow
		}
		qInternal[i] = q
		bases[i] = base
	}

	reserves := make([]*big.Int, len(entityPool.Reserves))
	for i, r := range entityPool.Reserves {
		reserves[i] = bignumber.NewBig(r)
	}

	feesPpm := make([]uint64, n)
	copy(feesPpm, extra.FeesPpm)

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:  entityPool.Address,
			Exchange: entityPool.Exchange,
			Type:     entityPool.Type,
			Tokens: lo.Map(entityPool.Tokens,
				func(item *entity.PoolToken, _ int) string { return item.Address }),
			Reserves:    reserves,
			BlockNumber: entityPool.BlockNumber,
		}},
		kappa:           kappa,
		effectiveSigmaQ: effectiveSigmaQ,
		qInternal:       qInternal,
		bases:           bases,
		feesPpm:         feesPpm,
		killed:          extra.Killed,
	}, nil
}

// CalcAmountOut reproduces PartyInfo.swapAmounts (exact-in) to the wei, plus the on-chain
// applySwap "pool drained" feasibility guard so a returned quote can actually execute.
func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if p.killed {
		return nil, ErrPoolKilled
	}

	indexIn := p.GetTokenIndex(params.TokenAmountIn.Token)
	indexOut := p.GetTokenIndex(params.TokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}
	if indexIn == indexOut {
		return nil, ErrSameToken
	}

	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow {
		return nil, ErrOverflow
	}

	amountOut, outFee, swapInfo, err := p.quoteExactIn(indexIn, indexOut, amountIn)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: p.Info.Tokens[indexOut], Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: p.Info.Tokens[indexOut], Amount: outFee.ToBig()},
		Gas:            p.estimateGas(),
		SwapInfo:       swapInfo,
	}, nil
}

// CalcAmountIn reproduces PartyInfo.swapAmountsForExactOutput to the wei. The SwapInfo it emits
// mirrors the actual exact-in swap of the resolved amountIn (the path the adapter executes), so
// UpdateBalance evolves state identically to an on-chain swap().
func (p *PoolSimulator) CalcAmountIn(params pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	if p.killed {
		return nil, ErrPoolKilled
	}

	indexIn := p.GetTokenIndex(params.TokenIn)
	indexOut := p.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}
	if indexIn == indexOut {
		return nil, ErrSameToken
	}

	amountOut, overflow := uint256.FromBig(params.TokenAmountOut.Amount)
	if overflow {
		return nil, ErrOverflow
	}
	if amountOut.IsZero() {
		return nil, ErrInvalidAmount
	}

	amountIn, outFee, swapInfo, err := p.quoteExactOut(indexIn, indexOut, amountOut)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: p.Info.Tokens[indexIn], Amount: amountIn.ToBig()},
		Fee:           &pool.TokenAmount{Token: p.Info.Tokens[indexOut], Amount: outFee.ToBig()},
		Gas:           p.estimateGas(),
		SwapInfo:      swapInfo,
	}, nil
}

// quoteExactIn implements PartyInfo.swapAmounts steps 1-6.
func (p *PoolSimulator) quoteExactIn(i, j int, amountIn *uint256.Int) (amountOut, outFee uint256.Int, swapInfo SwapInfo, err error) {
	feePpm := p.feesPpm[i] + p.feesPpm[j]

	// deltaInternalI = floor(amountIn / base_i); require > 0 ("too small").
	deltaInternalI, err := abdk.DivU(amountIn, p.bases[i])
	if err != nil {
		return
	}
	if deltaInternalI.Sign() <= 0 {
		err = ErrTooSmall
		return
	}

	amountOutInternal, err := swapAmountsForExactInput(p.kappa, p.effectiveSigmaQ, p.qInternal, i, j, &deltaInternalI)
	if err != nil {
		return
	}

	if err = p.checkDrained(j, &amountOutInternal); err != nil {
		return
	}

	grossOut, err := internalToUintFloor(&amountOutInternal, p.bases[j])
	if err != nil {
		return
	}
	if grossOut.IsZero() {
		err = ErrTooSmall
		return
	}

	outFee, err = ceilFee(&grossOut, feePpm)
	if err != nil {
		return
	}
	amountOut.Sub(&grossOut, &outFee)
	if amountOut.IsZero() {
		err = ErrTooSmall
		return
	}

	swapInfo = SwapInfo{
		TokenInIndex:  i,
		TokenOutIndex: j,
		DeltaInternal: deltaInternalI.ToBig(),
		GrossInternal: amountOutInternal.ToBig(),
	}
	return
}

// quoteExactOut implements PartyInfo.swapAmountsForExactOutput, then rebuilds the SwapInfo from the
// forward exact-in swap of the resolved amountIn so UpdateBalance matches the executed swap().
func (p *PoolSimulator) quoteExactOut(i, j int, amountOut *uint256.Int) (amountIn, outFee uint256.Int, swapInfo SwapInfo, err error) {
	feePpm := p.feesPpm[i] + p.feesPpm[j]

	// grossOut = ceil(amountOut·1e6 / (1e6 − feePpm)); the smallest gross whose net (after ceilFee)
	// still covers the requested amountOut.
	var grossOut uint256.Int
	if feePpm == 0 {
		grossOut.Set(amountOut)
	} else {
		denom := uint256.NewInt(1_000_000 - feePpm)
		var num uint256.Int
		if _, over := num.MulOverflow(amountOut, u1e6); over {
			err = ErrOverflow
			return
		}
		if _, over := num.AddOverflow(&num, denom); over {
			err = ErrOverflow
			return
		}
		num.Sub(&num, uOne)
		grossOut.Div(&num, denom)
	}

	yInternal, err := internalCeilFromUint(&grossOut, p.bases[j])
	if err != nil {
		return
	}
	if yInternal.Sign() <= 0 {
		err = ErrTooSmall
		return
	}

	amountInInternal, err := amountInForExactOutput(p.kappa, p.effectiveSigmaQ, p.qInternal, i, j, &yInternal)
	if err != nil {
		return
	}

	amountIn, err = internalToUintCeil(&amountInInternal, p.bases[i])
	if err != nil {
		return
	}
	if amountIn.IsZero() {
		err = ErrTooSmall
		return
	}

	outFee, err = ceilFee(&grossOut, feePpm)
	if err != nil {
		return
	}

	// The adapter executes an exact-in swap of `amountIn` wei, so derive the state deltas from that
	// forward quote (not the exact-out internals) to keep UpdateBalance wei-consistent with swap().
	deltaInternalI, err := abdk.DivU(&amountIn, p.bases[i])
	if err != nil {
		return
	}
	if deltaInternalI.Sign() <= 0 {
		err = ErrTooSmall
		return
	}
	fwdOutInternal, err := swapAmountsForExactInput(p.kappa, p.effectiveSigmaQ, p.qInternal, i, j, &deltaInternalI)
	if err != nil {
		return
	}
	if err = p.checkDrained(j, &fwdOutInternal); err != nil {
		return
	}

	swapInfo = SwapInfo{
		TokenInIndex:  i,
		TokenOutIndex: j,
		DeltaInternal: deltaInternalI.ToBig(),
		GrossInternal: fwdOutInternal.ToBig(),
	}
	return
}

// checkDrained enforces the on-chain applySwap invariant qInternal[j] − amountOutInternal > 0
// ("pool drained"), rejecting quotes that would revert when executed.
func (p *PoolSimulator) checkDrained(j int, amountOutInternal *int256.Int) error {
	var newQj int256.Int
	newQj.Sub(p.qInternal[j], amountOutInternal)
	if newQj.Sign() <= 0 {
		return ErrTooLarge
	}
	return nil
}

// UpdateBalance mirrors LMSRKernel.applySwap: qInternal[i] += deltaInternalI (input); qInternal[j]
// -= amountOutInternal (GROSS kernel output — the withheld fee stays in the pool). Reserves track
// the net user-facing token flow (+amountIn, −amountOut); pricing depends only on qInternal, so the
// protocol-fee sliver excluded from reserves does not affect quotes.
func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swapInfo, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		return
	}
	i, j := swapInfo.TokenInIndex, swapInfo.TokenOutIndex
	n := len(p.qInternal)
	if i < 0 || i >= n || j < 0 || j >= n {
		return
	}

	delta, err := int256.FromBig(swapInfo.DeltaInternal)
	if err != nil {
		return
	}
	gross, err := int256.FromBig(swapInfo.GrossInternal)
	if err != nil {
		return
	}

	p.qInternal[i].Add(p.qInternal[i], delta)
	p.qInternal[j].Sub(p.qInternal[j], gross)

	if i < len(p.Info.Reserves) && p.Info.Reserves[i] != nil && params.TokenAmountIn.Amount != nil {
		p.Info.Reserves[i].Add(p.Info.Reserves[i], params.TokenAmountIn.Amount)
	}
	if j < len(p.Info.Reserves) && p.Info.Reserves[j] != nil && params.TokenAmountOut.Amount != nil {
		p.Info.Reserves[j].Sub(p.Info.Reserves[j], params.TokenAmountOut.Amount)
	}
}

// CloneState deep-copies the mutable state UpdateBalance writes in place (qInternal + Reserves).
// kappa/effectiveSigmaQ/bases/feesPpm are block-immutable and safely shared (copy-on-write).
func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p

	cloned.qInternal = make([]*int256.Int, len(p.qInternal))
	for i, q := range p.qInternal {
		cloned.qInternal[i] = q.Clone()
	}

	cloned.Info.Reserves = make([]*big.Int, len(p.Info.Reserves))
	for i, r := range p.Info.Reserves {
		if r != nil {
			cloned.Info.Reserves[i] = new(big.Int).Set(r)
		}
	}

	return &cloned
}

// GetMetaInfo emits the (i, j) token indices the aggregator needs to build the adapter calldata —
// word-aligned abi.encode(pool, indexIn, indexOut), matching the in-repo adapter convention.
// PartyPool.swap takes token indices, not addresses.
func (p *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) any {
	return Meta{
		TokenInIndex:  p.GetTokenIndex(tokenIn),
		TokenOutIndex: p.GetTokenIndex(tokenOut),
	}
}
