package uniswapv4

import (
	"fmt"
	"maps"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/shared"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	few_v1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/few/v1"
	few_v2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/few/v2"
	sharedv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// tokenWrappers lists few_v2 before few_v1 so, for a token pair with pools in both
// generations (e.g. WBTC/fwWBTC), the current/active generation wins.
var tokenWrappers = []ITokenWrapper{few_v2.NewTokenWrapper(), few_v1.NewTokenWrapper()}

// fewWrapped returns the wrap metadata of token if it is a Few-wrapped token.
func fewWrapped(chainID valueobject.ChainID, token string) (sharedv4.IWrapMetadata, bool) {
	for _, wrapper := range tokenWrappers {
		if metadata, ok := wrapper.IsWrapped(chainID, token); ok {
			return metadata, true
		}
	}
	return nil, false
}

type PoolSimulator struct {
	*uniswapv3.PoolSimulator
	staticExtra   StaticExtra
	hook          Hook
	chainID       valueobject.ChainID
	tokenWrappers []ITokenWrapper
	// fewReserves caps how much of each Few-wrapped pool token can be unwrapped (see
	// Extra.FewReserves); nil for a non-Few token or a pool not yet re-tracked.
	fewReserves [2]*uint256.Int
	// fewWrapperIdx is the Few token's index when this pool is that token's own wrap/unwrap
	// pool (so swapping out of it unwraps it), else -1.
	fewWrapperIdx int

	// tokens and canSwapTo cache GetTokens and CanSwapTo (keyed by each token of GetTokens): both
	// depend only on the pool's tokens and the token wrappers, and the route finder calls them far
	// more often than CalcAmountOut. Not serialized; a decoded simulator computes them per call.
	exchange  string              `msgpack:"-"` // hook exchange with the fork dexID
	tokens    []string            `msgpack:"-"`
	canSwapTo map[string][]string `msgpack:"-"`

	// protocolFee is slot0's packed protocol fee and lpFee the LP fee it applies on top of
	// (slot0.lpFee, or the pool's SwapFee while the protocol fee is off).
	protocolFee uint32
	lpFee       FeeAmount
}

var _ = pool.RegisterFactory(DexType, func(params pool.FactoryParams) (*PoolSimulator, error) {
	return newPoolSimulator(params.EntityPool, params.ChainID, params.Opts.StaleCheck)
})

func NewPoolSimulator(entityPool entity.Pool, chainID valueobject.ChainID) (*PoolSimulator, error) {
	return newPoolSimulator(entityPool, chainID, false)
}

func gasForChain(chainID valueobject.ChainID) uniswapv3.Gas {
	gas := DefaultGas
	if chainID == valueobject.ChainIDPolygon {
		gas.CrossEmptyWordGas = polygonCrossEmptyWordGas
	}
	return gas
}

func newPoolSimulator(entityPool entity.Pool, chainID valueobject.ChainID, staleCheck bool) (*PoolSimulator, error) {
	var extra ExtraU256
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, fmt.Errorf("unmarshal static extra: %w", err)
	}

	// Indexing prices at the snapshot time; route finding keeps the wall clock (QuoteAt 0).
	var quoteAt int64
	if t := pool.QuoteTime(staleCheck, entityPool.Timestamp); !t.IsZero() {
		quoteAt = t.Unix()
	}

	hook, ok := GetHook(staticExtra.HooksAddress, &HookParam{
		Cfg:        &Config{ChainID: chainID},
		Pool:       &entityPool,
		HookExtra:  HookExtra(extra.HookExtra),
		TrackedAt:  extra.TrackedAt,
		StaleCheck: staleCheck,
		QuoteAt:    quoteAt,
	})
	if !ok && HasSwapPermissions(staticExtra.HooksAddress) {
		return nil, shared.ErrUnsupportedHook
	}

	allowEmptyTicks := hook.AllowEmptyTicks()

	// V3Pool's tick/liquidity math -- and the reserve-based insufficient-balance check inside
	// it -- always operates in each currency's real on-chain decimals. entityPool.Reserves
	// stores a native-flagged index at the wrapped-native address's decimals instead (only
	// differs from native's own decimals on Arc); convert it back before building the v3
	// simulator, matching the Unwrap/WrapNativeAmount pair CalcAmountOut applies below.
	entityPool.Reserves = slices.Clone(entityPool.Reserves)
	for i, isNative := range staticExtra.IsNative {
		if isNative && i < len(entityPool.Reserves) {
			if amount, ok := new(big.Int).SetString(entityPool.Reserves[i], 10); ok {
				entityPool.Reserves[i] = valueobject.UnwrapNativeAmount(chainID, amount).String()
			}
		}
	}

	v3PoolSimulator, err := uniswapv3.NewPoolSimulatorWithExtra(entityPool, extra.ExtraTickU256,
		uniswapv3.SimulatorConfig{AllowEmptyTicks: allowEmptyTicks})
	if err != nil {
		return nil, err
	}
	v3PoolSimulator.Gas = gasForChain(chainID)

	var fewReserves [2]*uint256.Int
	copy(fewReserves[:], extra.FewReserves)
	fewWrapperIdx := -1
	for i, reserve := range fewReserves {
		if reserve == nil {
			continue
		}
		for _, wrapper := range tokenWrappers {
			if metadata, ok := wrapper.IsWrapped(chainID, entityPool.Tokens[i].Address); ok &&
				metadata.GetPool() == entityPool.Address {
				fewWrapperIdx = i
			}
		}
	}

	// Without pF (protocol fee off, or an Extra written before pF existed) SwapFee is the fee for
	// both directions, as before.
	lpFee, protocolFee := v3PoolSimulator.V3Pool.Fee, uint32(0)
	if extra.Fees != 0 {
		lpFee, protocolFee = FeeAmount(extra.Fees>>24), uint32(extra.Fees&0xffffff)
	}

	p := &PoolSimulator{
		PoolSimulator: v3PoolSimulator,
		staticExtra:   staticExtra,
		hook:          hook,
		exchange:      ExchangeOf(hook, "", entityPool.Exchange),
		chainID:       chainID,
		tokenWrappers: tokenWrappers,
		fewReserves:   fewReserves,
		fewWrapperIdx: fewWrapperIdx,
		protocolFee:   protocolFee,
		lpFee:         lpFee,
	}
	p.tokens = p.computeTokens()
	p.canSwapTo = make(map[string][]string, len(p.tokens))
	for _, token := range p.tokens {
		p.canSwapTo[token] = p.computeCanSwapTo(token)
	}
	return p, nil
}

// arcNativeScale is the 1e12 decimals gap of valueobject's arcNativeDecimalsScale, for uint256 math.
var arcNativeScale = uint256.NewInt(1e12)

// hookSwaps reports whether the pool's hook runs in BeforeSwap or AfterSwap for this pool.
func (p *PoolSimulator) hookSwaps() bool {
	return p.hook != nil && (p.hook.CanBeforeSwap(p.staticExtra.HooksAddress) ||
		p.hook.CanAfterSwap(p.staticExtra.HooksAddress))
}

// CalcAmtOut is CalcAmountOut with uint256 amounts, including the BeforeSwap and AfterSwap hook terms.
func (p *PoolSimulator) CalcAmtOut(params pool.CalcAmtOutParams) (*pool.CalcAmtOutResult, error) {
	originalTokenIn, originalTokenOut := params.TokenAmountIn.Token, params.TokenOut
	amountIn := params.TokenAmountIn.Amount
	var wrapAdditionalGas int64
	unwrapIdx := -1

	if idx := p.GetTokenIndex(originalTokenIn); idx >= 0 && p.staticExtra.IsNative[idx] {
		if overflow := p.unwrapNativeU256(&amountIn); overflow {
			return nil, uniswapv3.ErrOverflow
		}
	}

	tokenIn, tokenOut := originalTokenIn, originalTokenOut
	if p.GetTokenIndex(tokenIn) == -1 {
		for _, wrapper := range p.tokenWrappers {
			metadata, canWrap := wrapper.CanWrap(p.chainID, tokenIn)
			if canWrap {
				tokenIn = metadata.GetWrapToken()
				wrapAdditionalGas += p.Gas.BaseGas
				break
			}
		}
	}
	if p.GetTokenIndex(tokenOut) == -1 {
		for _, wrapper := range p.tokenWrappers {
			metadata, canUnwrap := wrapper.CanWrap(p.chainID, tokenOut)
			if canUnwrap {
				tokenOut = metadata.GetWrapToken()
				unwrapIdx = p.GetTokenIndex(tokenOut)
				wrapAdditionalGas += p.Gas.BaseGas
				break
			}
		}
	} else if p.fewWrapperIdx >= 0 && p.GetTokenIndex(tokenIn) == p.fewWrapperIdx {
		unwrapIdx = p.fewWrapperIdx
	}

	zeroForOne := p.GetTokenIndex(tokenIn) == 0
	var beforeSwapResult *BeforeSwapResult
	var afterSwapResult *AfterSwapResult
	var amountInOverflow bool // the hook's negative DeltaSpecified pushed the input past 256 bits
	hooked := p.hookSwaps()

	if hooked && p.hook.CanBeforeSwap(p.staticExtra.HooksAddress) {
		var err error
		if beforeSwapResult, err = p.hook.BeforeSwap(&BeforeSwapParams{
			CalcOut:         true,
			ZeroForOne:      zeroForOne,
			AmountSpecified: new(uint256.Int).Set(&amountIn),
			SqrtPriceX96:    &p.V3Pool.SqrtRatioX96,
		}); err != nil {
			return nil, fmt.Errorf("[BeforeSwap] %w", err)
		} else if err = ValidateBeforeSwapResult(beforeSwapResult); err != nil {
			return nil, fmt.Errorf("[BeforeSwap] validation failed: %w", err)
		}

		var neg bool
		if neg, amountInOverflow = big256.SubSigned(&amountIn, false, beforeSwapResult.DeltaSpecified); neg {
			return nil, ErrInvalidAmountIn
		}
	}

	swapFee, err := p.swapFee(zeroForOne, beforeSwapResult)
	if err != nil {
		return nil, err
	} else if amountInOverflow {
		return nil, uniswapv3.ErrOverflow
	}

	swapResult, err := p.withSwapFee(swapFee).CalcAmtOut(pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{Token: tokenIn, Amount: amountIn},
		TokenOut:      tokenOut,
	})
	if err != nil {
		return nil, err
	}

	if hooked && p.hook.CanAfterSwap(p.staticExtra.HooksAddress) {
		afterSwapResult, err = p.hook.AfterSwap(&AfterSwapParams{
			BeforeSwapParams: &BeforeSwapParams{
				CalcOut:         true,
				ZeroForOne:      zeroForOne,
				AmountSpecified: new(uint256.Int).Set(&amountIn),
				SqrtPriceX96:    &p.V3Pool.SqrtRatioX96,
			},
			AmountIn:         new(uint256.Int).Set(&amountIn),
			AmountOut:        new(uint256.Int).Set(&swapResult.TokenAmountOut.Amount),
			BeforeSwapResult: beforeSwapResult,
		})
		if err != nil {
			return nil, fmt.Errorf("[AfterSwap] %w", err)
		} else if err = ValidateAfterSwapResult(afterSwapResult); err != nil {
			return nil, fmt.Errorf("[AfterSwap] validation failed: %w", err)
		}
	}

	swapResult.TokenAmountOut.Token = originalTokenOut
	swapResult.RemainingTokenAmountIn.Token = originalTokenIn
	swapResult.Gas += wrapAdditionalGas

	// Hook deltas are signed: out -= DeltaUnspecified, out -= HookFee. A negative value below zero is
	// checked after the native rescale, like the big.Int body.
	amountOut := &swapResult.TokenAmountOut.Amount
	var outNeg, outOverflow bool
	v4SwapInfo := SwapInfo{PoolSwapInfo: swapResult.SwapInfo.(PoolSwapInfo)}
	if beforeSwapResult != nil {
		var overflow bool
		outNeg, overflow = big256.SubSigned(amountOut, outNeg, beforeSwapResult.DeltaUnspecified)
		outOverflow = outOverflow || overflow
		swapResult.Gas += beforeSwapResult.Gas
		v4SwapInfo.HookSwapInfo = beforeSwapResult.SwapInfo
	}
	if afterSwapResult != nil {
		var overflow bool
		outNeg, overflow = big256.SubSigned(amountOut, outNeg, afterSwapResult.HookFee)
		outOverflow = outOverflow || overflow
		swapResult.Gas += afterSwapResult.Gas
	}
	if outOverflow {
		return nil, uniswapv3.ErrOverflow
	}
	if idx := p.GetTokenIndex(originalTokenOut); idx >= 0 && p.staticExtra.IsNative[idx] {
		p.wrapNativeU256(amountOut)
		outNeg = outNeg && !amountOut.IsZero() // big.Int Quo truncates a small negative to zero
	}
	if outNeg {
		return nil, ErrInvalidAmountOut
	}
	fewReserve, err := p.unwrapFewU256(unwrapIdx, &swapResult.TokenAmountOut.Amount)
	if err != nil {
		return nil, err
	}
	if fewReserve != nil {
		v4SwapInfo.FewReserve, v4SwapInfo.FewIdx = fewReserve, unwrapIdx
	}
	swapResult.SwapInfo = v4SwapInfo
	return swapResult, nil
}

// tokenAmtFromBigV4 converts a big.Int amount to uint256 at the hook boundary. Amounts from the shared
// CalcAmountOut body are non-negative and fit 256 bits, so a nil result is returned only for a nil input.
func tokenAmtFromBigV4(t *pool.TokenAmount) *pool.TokenAmt {
	if t == nil {
		return nil
	}
	out := &pool.TokenAmt{Token: t.Token, AmountUsd: t.AmountUsd}
	if t.Amount != nil {
		out.Amount = *uint256.MustFromBig(t.Amount)
	}
	return out
}

// unwrapNativeU256 rescales a native-flagged amount to the wrapped-native decimals of valueobject's
// UnwrapNativeAmount. Only Arc rescales; it reports overflow like a failed big.Int product would revert.
func (p *PoolSimulator) unwrapNativeU256(amount *uint256.Int) (overflow bool) {
	if p.chainID != valueobject.ChainIDArc {
		return false
	}
	_, overflow = amount.MulOverflow(amount, arcNativeScale)
	return overflow
}

// wrapNativeU256 is the inverse of unwrapNativeU256, matching valueobject's WrapNativeAmount (truncating).
func (p *PoolSimulator) wrapNativeU256(amount *uint256.Int) {
	if p.chainID == valueobject.ChainIDArc {
		amount.Div(amount, arcNativeScale)
	}
}

// unwrapFewU256 is unwrapFew for a uint256 amount.
func (p *PoolSimulator) unwrapFewU256(idx int, amount *uint256.Int) (*uint256.Int, error) {
	if idx < 0 || p.fewReserves[idx] == nil {
		return nil, nil
	}
	if amount.Gt(p.fewReserves[idx]) {
		return nil, ErrFewUnwrapExceeded
	}
	return new(uint256.Int).Sub(p.fewReserves[idx], amount), nil
}

// swapFee is the fee PoolManager.swap charges in the given direction: the LP fee (a dynamic-fee
// pool's BeforeSwap override when the hook returns one, else slot0.lpFee) combined with that
// direction's protocol fee via ProtocolFeeLibrary.calculateSwapFee.
func (p *PoolSimulator) swapFee(zeroForOne bool, beforeSwapResult *BeforeSwapResult) (FeeAmount, error) {
	lpFee := p.lpFee
	if beforeSwapResult != nil && shared.IsDynamicFee(p.staticExtra.Fee) { // ignore if not dynamic fee
		if fee, ok := lpFeeOverride(beforeSwapResult.SwapFee); fee >= FeeMax {
			return 0, ErrInvalidFee
		} else if ok {
			lpFee = fee
		}
	}
	if protocolFee := directionalProtocolFee(p.protocolFee, zeroForOne); protocolFee != 0 {
		return CalculateSwapFee(protocolFee, lpFee), nil
	}
	return lpFee, nil
}

// withSwapFee returns the v3 simulator to quote with, a shallow clone when fee differs from the
// pool's stored fee.
func (p *PoolSimulator) withSwapFee(fee FeeAmount) *uniswapv3.PoolSimulator {
	if fee == p.V3Pool.Fee {
		return p.PoolSimulator
	}
	cloned := *p.PoolSimulator
	clonedV3Pool := *p.V3Pool
	clonedV3Pool.Fee = fee
	cloned.V3Pool = &clonedV3Pool
	return &cloned
}

// CalcAmtIn reverses exact-in with uint256 amounts: hooks get CalcOut=false and must invert their exact-in
// fees, not simulate on-chain exact-out, since the swap is always executed exact-in.
func (p *PoolSimulator) CalcAmtIn(params pool.CalcAmtInParams) (*pool.CalcAmtInResult, error) {
	originalTokenOut, originalTokenIn := params.TokenAmountOut.Token, params.TokenIn
	amountOut := params.TokenAmountOut.Amount
	var wrapAdditionalGas int64
	unwrapIdx := -1 // pool index of the Few token this swap unwraps

	// A native-flagged side trades at native's own decimals on-chain, not the wrapped-native
	// address's used in p.Tokens; only Arc's differ (18 vs 6). Rescale around the v3 math below.
	if idx := p.GetTokenIndex(originalTokenOut); idx >= 0 && p.staticExtra.IsNative[idx] &&
		p.unwrapNativeU256(&amountOut) {
		amountOut.Set(big256.UMax) // the big.Int body kept the exact product, which v3 rejects as above the reserves
	}

	// Wrap/unwrap tokens if needed and calculate wrap gas
	tokenIn, tokenOut := originalTokenIn, originalTokenOut
	if p.GetTokenIndex(tokenOut) == -1 {
		for _, wrapper := range p.tokenWrappers {
			metadata, canWrap := wrapper.CanWrap(p.chainID, tokenOut)
			if canWrap {
				tokenOut = metadata.GetWrapToken()
				unwrapIdx = p.GetTokenIndex(tokenOut)
				wrapAdditionalGas += p.Gas.BaseGas
				break
			}
		}
	} else if p.fewWrapperIdx >= 0 && p.GetTokenIndex(tokenIn) == p.fewWrapperIdx {
		unwrapIdx = p.fewWrapperIdx
	}
	fewReserve, err := p.unwrapFewU256(unwrapIdx, &amountOut)
	if err != nil {
		return nil, err
	}
	if p.GetTokenIndex(tokenIn) == -1 {
		for _, wrapper := range p.tokenWrappers {
			metadata, canUnwrap := wrapper.CanWrap(p.chainID, tokenIn)
			if canUnwrap {
				tokenIn = metadata.GetWrapToken()
				wrapAdditionalGas += p.Gas.BaseGas
				break
			}
		}
	}

	zeroForOne := p.GetTokenIndex(tokenOut) == 1
	var beforeSwapResult *BeforeSwapResult
	var afterSwapResult *AfterSwapResult
	var amountOutOverflow bool // the hook's positive DeltaSpecified pushed the output past 256 bits

	if p.hook != nil && p.hook.CanBeforeSwap(p.staticExtra.HooksAddress) {
		if beforeSwapResult, err = p.hook.BeforeSwap(&BeforeSwapParams{
			CalcOut:         false,
			ZeroForOne:      zeroForOne,
			AmountSpecified: new(uint256.Int).Set(&amountOut),
			SqrtPriceX96:    &p.V3Pool.SqrtRatioX96,
		}); err != nil {
			return nil, fmt.Errorf("[BeforeSwap] %w", err)
		} else if err = ValidateBeforeSwapResult(beforeSwapResult); err != nil {
			return nil, fmt.Errorf("[BeforeSwap] validation failed: %w", err)
		}

		var neg bool
		if neg, amountOutOverflow = big256.AddSigned(&amountOut, false, beforeSwapResult.DeltaSpecified); neg {
			return nil, ErrInvalidAmountOut
		}
	}

	swapFee, err := p.swapFee(zeroForOne, beforeSwapResult)
	if err != nil {
		return nil, err
	} else if amountOutOverflow {
		// The big.Int body kept the exact sum, which v3 rejects as above the pool's reserves.
		amountOut.Set(big256.UMax)
	}

	swapResult, err := p.withSwapFee(swapFee).CalcAmtIn(pool.CalcAmtInParams{
		TokenAmountOut: pool.TokenAmt{Token: tokenOut, Amount: amountOut},
		TokenIn:        tokenIn,
	})
	if err != nil {
		return nil, err
	}

	if p.hook != nil && p.hook.CanAfterSwap(p.staticExtra.HooksAddress) {
		if afterSwapResult, err = p.hook.AfterSwap(&AfterSwapParams{
			BeforeSwapParams: &BeforeSwapParams{
				CalcOut:         false,
				ZeroForOne:      zeroForOne,
				AmountSpecified: new(uint256.Int).Set(&amountOut),
				SqrtPriceX96:    &p.V3Pool.SqrtRatioX96,
			},
			AmountIn:         new(uint256.Int).Set(&swapResult.TokenAmountIn.Amount),
			AmountOut:        new(uint256.Int).Set(&amountOut),
			BeforeSwapResult: beforeSwapResult,
		}); err != nil {
			return nil, fmt.Errorf("[AfterSwap] %w", err)
		} else if err = ValidateAfterSwapResult(afterSwapResult); err != nil {
			return nil, fmt.Errorf("[AfterSwap] validation failed: %w", err)
		}
	}

	// Hook deltas are signed: in += DeltaUnspecified, in += HookFee. A negative value below zero is
	// checked after the native rescale, like the big.Int body.
	swapResult.TokenAmountIn.Token = originalTokenIn
	swapResult.RemainingTokenAmountOut.Token = originalTokenOut
	swapResult.Gas += wrapAdditionalGas
	amountIn := &swapResult.TokenAmountIn.Amount
	var inNeg, inOverflow, overflow bool
	v4SwapInfo := SwapInfo{PoolSwapInfo: swapResult.SwapInfo.(PoolSwapInfo), FewReserve: fewReserve, FewIdx: unwrapIdx}
	if beforeSwapResult != nil {
		inNeg, overflow = big256.AddSigned(amountIn, inNeg, beforeSwapResult.DeltaUnspecified)
		inOverflow = inOverflow || overflow
		swapResult.Gas += beforeSwapResult.Gas
		v4SwapInfo.HookSwapInfo = beforeSwapResult.SwapInfo
	}
	if afterSwapResult != nil {
		inNeg, overflow = big256.AddSigned(amountIn, inNeg, afterSwapResult.HookFee)
		inOverflow = inOverflow || overflow
		swapResult.Gas += afterSwapResult.Gas
	}
	if inOverflow {
		return nil, uniswapv3.ErrOverflow
	}
	if idx := p.GetTokenIndex(originalTokenIn); idx >= 0 && p.staticExtra.IsNative[idx] {
		p.wrapNativeU256(amountIn)
		inNeg = inNeg && !amountIn.IsZero() // big.Int Quo truncates a small negative to zero
	}
	if inNeg {
		return nil, ErrInvalidAmountIn
	}
	swapResult.SwapInfo = v4SwapInfo
	return swapResult, nil
}

func (p *PoolSimulator) CanSwapFrom(address string) []string {
	return p.CanSwapTo(address)
}

// CanSwapTo returns a slice shared with the simulator and its clones: callers must not modify it.
func (p *PoolSimulator) CanSwapTo(address string) []string {
	if tokens, ok := p.canSwapTo[address]; ok {
		return tokens
	}
	return p.computeCanSwapTo(address)
}

func (p *PoolSimulator) computeCanSwapTo(address string) []string {
	tokenIndex := p.GetTokenIndex(address)

	result := map[string]struct{}{}
	if tokenIndex == -1 {
		for _, wrapper := range p.tokenWrappers {
			metadata, canWrap := wrapper.CanWrap(p.chainID, address)
			if !canWrap {
				continue
			}

			wrapTokenIndex := p.GetTokenIndex(metadata.GetWrapToken())
			if wrapTokenIndex == -1 {
				continue
			}

			res := p.CanSwapTo(metadata.GetWrapToken())
			for _, token := range res {
				result[token] = struct{}{}
			}
		}
	} else { // tokenIndex >= 0
		for _, token := range p.Info.Tokens {
			if token != address {
				result[token] = struct{}{}

				for _, wrapper := range p.tokenWrappers {
					metadata, canUnwrap := wrapper.IsWrapped(p.chainID, token)
					if canUnwrap && metadata.GetUnwrapToken() != address {
						result[metadata.GetUnwrapToken()] = struct{}{}
					}
				}
			}
		}
	}

	return slices.Collect(maps.Keys(result))
}

func (p *PoolSimulator) GetExchange() string {
	if p.exchange != "" {
		return p.exchange
	}
	// Unexported fields are empty after a msgpack decode; Info.Exchange keeps the fork name.
	return ExchangeOf(p.hook, "", p.Info.Exchange)
}

// GetTokens returns a slice shared with the simulator and its clones: callers must not modify it.
func (p *PoolSimulator) GetTokens() []string {
	if p.tokens != nil {
		return p.tokens
	}
	return p.computeTokens()
}

func (p *PoolSimulator) computeTokens() []string {
	seen := make(map[string]struct{})
	tokens := make([]string, 0, len(p.Info.Tokens))

	for _, token := range p.Info.Tokens {
		if _, exists := seen[token]; !exists {
			seen[token] = struct{}{}
			tokens = append(tokens, token)
		}

		for _, wrapper := range p.tokenWrappers {
			if metadata, isWrapped := wrapper.IsWrapped(p.chainID, token); isWrapped {
				unwrappedToken := metadata.GetUnwrapToken()
				if _, exists := seen[unwrappedToken]; !exists {
					seen[unwrappedToken] = struct{}{}
					tokens = append(tokens, unwrappedToken)
				}
			}
		}
	}

	return tokens
}

func (p *PoolSimulator) CloneState() pool.IPoolSim {
	cloned := *p
	cloned.PoolSimulator = p.PoolSimulator.CloneState().(*uniswapv3.PoolSimulator)
	if cloned.hook != nil {
		cloned.hook = p.hook.CloneState()
		if _, ok := cloned.hook.(*BaseHook); ok {
			if _, ok = p.hook.(*BaseHook); !ok {
				cloned.hook = p.hook
			}
		}
	}
	return &cloned
}

func (p *PoolSimulator) UpdateState(params pool.UpdateStateParams) {
	if params.SwapInfo == nil {
		return
	}
	v4SwapInfo, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		return
	}
	if p.hook != nil {
		p.hook.UpdateBalance(v4SwapInfo.HookSwapInfo)
	}
	if v4SwapInfo.FewReserve != nil {
		p.fewReserves[v4SwapInfo.FewIdx] = new(uint256.Int).Set(v4SwapInfo.FewReserve)
	}
	params.SwapInfo = v4SwapInfo.PoolSwapInfo

	// p.PoolSimulator tracks reserves at each currency's real on-chain decimals (see
	// NewPoolSimulator); TokenAmountIn/Out here are in the wrapped-native address's decimals
	// for a native-flagged leg, so rescale before applying, mirroring the Unwrap/WrapNativeAmount
	// pair CalcAmountOut/CalcAmountIn apply around the same v3 math.
	if idx := p.GetTokenIndex(params.TokenAmountIn.Token); idx >= 0 && p.staticExtra.IsNative[idx] {
		p.unwrapNativeU256(&params.TokenAmountIn.Amount)
	}
	if idx := p.GetTokenIndex(params.TokenAmountOut.Token); idx >= 0 && p.staticExtra.IsNative[idx] {
		p.unwrapNativeU256(&params.TokenAmountOut.Amount)
	}

	p.PoolSimulator.UpdateState(params)
	if provider, ok := p.hook.(HookPoolStateProvider); ok {
		state := provider.PoolState()
		p.Info.Reserves = []*big.Int{new(big.Int).Set(state.Reserves[0]), new(big.Int).Set(state.Reserves[1])}
		p.V3Pool.SqrtRatioX96.Set(&state.SqrtPriceX96)
		p.V3Pool.Liquidity.Set(&state.Liquidity)
		p.V3Pool.TickCurrent = state.Tick
	}
}

// GetMetaInfo
// adapt from https://github.com/KyberNetwork/kyberswap-dex-lib-private/blob/c1877a8c19759faeb7d82b6902ed335f0657ce3e/pkg/liquidity-source/uniswap-v4/pool_simulator.go#L201
func (p *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) any {
	tokenInAfterWrap, tokenOutBeforeUnwrap := tokenIn, tokenOut
	var wrapMetadata *TokenWrapMetadata

	if p.GetTokenIndex(tokenIn) == -1 {
		for _, wrapper := range p.tokenWrappers {
			if metadata, canWrap := wrapper.CanWrap(p.chainID, tokenIn); canWrap {
				tokenInAfterWrap = metadata.GetWrapToken()
				if metadata.IsUnwrapNative() {
					tokenIn = NativeTokenAddress.Hex()
				}

				wrapMetadata = &TokenWrapMetadata{WrapInfo: &WrapInfo{
					TokenIn:     tokenIn,
					TokenOut:    metadata.GetWrapToken(),
					HookAddress: metadata.GetHook(),
					PoolAddress: metadata.GetPool(),
					TickSpacing: metadata.GetTickSpacing(),
					Fee:         metadata.GetFee(),
					HookData:    metadata.GetHookData(),
				}}
				break
			}
		}
	}

	if p.GetTokenIndex(tokenOut) == -1 {
		for _, wrapper := range p.tokenWrappers {
			if metadata, canUnwrap := wrapper.CanWrap(p.chainID, tokenOut); canUnwrap {
				tokenOutBeforeUnwrap = metadata.GetWrapToken()
				if metadata.IsUnwrapNative() {
					tokenOut = NativeTokenAddress.Hex()
				}

				if wrapMetadata == nil {
					wrapMetadata = &TokenWrapMetadata{}
				}
				wrapMetadata.UnwrapInfo = &WrapInfo{
					TokenIn:     tokenOutBeforeUnwrap,
					TokenOut:    tokenOut,
					HookAddress: metadata.GetHook(),
					PoolAddress: metadata.GetPool(),
					TickSpacing: metadata.GetTickSpacing(),
					Fee:         metadata.GetFee(),
					HookData:    metadata.GetHookData(),
				}
			}
		}
	}

	tokenInAddress, tokenOutAddress := NativeTokenAddress, NativeTokenAddress
	if !p.staticExtra.IsNative[p.GetTokenIndex(tokenInAfterWrap)] {
		tokenInAddress = common.HexToAddress(tokenInAfterWrap)
	}
	if !p.staticExtra.IsNative[p.GetTokenIndex(tokenOutBeforeUnwrap)] {
		tokenOutAddress = common.HexToAddress(tokenOutBeforeUnwrap)
	}

	zeroForOne := tokenInAfterWrap == p.Info.Tokens[0]
	priceLimit := p.GetSqrtPriceLimit(zeroForOne)
	if provider, ok := p.hook.(HookPriceLimitProvider); ok {
		if hookLimit := provider.SqrtPriceLimit(zeroForOne); hookLimit != nil {
			priceLimit = hookLimit
		}
	}

	return PoolMetaInfo{
		BlockNumber:       p.Info.BlockNumber,
		Router:            p.staticExtra.UniversalRouterAddress,
		Permit2Addr:       p.staticExtra.Permit2Address,
		TokenIn:           tokenInAddress,
		TokenOut:          tokenOutAddress,
		Fee:               p.staticExtra.Fee,
		TickSpacing:       p.staticExtra.TickSpacing,
		HookAddress:       p.staticExtra.HooksAddress,
		HookData:          p.hook.GetHookData(),
		PriceLimit:        priceLimit,
		PoolManager:       p.poolManager(),
		TokenWrapMetadata: wrapMetadata,

		TicksBasedStateBlock: p.StateBlockMeta(),
	}
}

func (s *PoolSimulator) SwapReceiveNativeIn(tokenIn, tokenOut string, _ valueobject.ChainID) bool {
	meta := s.GetMetaInfo(tokenIn, tokenOut).(PoolMetaInfo)
	return meta.TokenIn == NativeTokenAddress
}

func (s *PoolSimulator) SwapReturnNativeOut(tokenIn, tokenOut string, _ valueobject.ChainID) bool {
	meta := s.GetMetaInfo(tokenIn, tokenOut).(PoolMetaInfo)
	return meta.TokenOut == NativeTokenAddress
}

// poolManager is the pool's own manager, else the official one for the chain, else nil.
func (p *PoolSimulator) poolManager() *common.Address {
	if p.staticExtra.PoolManager != nil {
		return p.staticExtra.PoolManager
	}
	if pm, ok := PoolManager[p.chainID]; ok {
		return &pm
	}
	return nil
}
