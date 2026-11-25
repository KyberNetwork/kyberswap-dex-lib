package cl

import (
	"fmt"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/shared"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var (
	defaultGas = uniswapv3.Gas{BaseGas: 75000, CrossInitTickGas: 21000, CrossEmptyWordGas: uniswapv3.CrossEmptyWordGas}
)

type PoolSimulator struct {
	*uniswapv3.PoolSimulator
	staticExtra StaticExtra
	hook        Hook
	chainID     valueobject.ChainID
}

var _ = pool.RegisterFactory1(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool, chainID valueobject.ChainID) (*PoolSimulator, error) {
	var extra ExtraU256
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, fmt.Errorf("unmarshal static extra: %w", err)
	}

	hook, ok := GetHook(staticExtra.HooksAddress, &HookParam{
		Cfg:       &Config{ChainID: int(chainID), DexID: entityPool.Exchange},
		Pool:      &entityPool,
		HookExtra: extra.HookExtra,
	})
	if !ok && staticExtra.HasSwapPermissions {
		return nil, shared.ErrUnsupportedHook
	}

	allowEmptyTicks := false
	if hook != nil {
		allowEmptyTicks = hook.AllowEmptyTicks()
	}

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
	v3PoolSimulator.Gas = defaultGas

	return &PoolSimulator{
		PoolSimulator: v3PoolSimulator,
		staticExtra:   staticExtra,
		hook:          hook,
		chainID:       chainID,
	}, nil
}

// CalcAmtOut is CalcAmountOut with uint256 amounts. It shadows the embedded v3 CalcAmtOut, which would skip
// this pool's hook logic.
func (p *PoolSimulator) CalcAmtOut(params pool.CalcAmtOutParams) (*pool.CalcAmtOutResult, error) {
	originalTokenIn, originalTokenOut := params.TokenAmountIn.Token, params.TokenOut
	amountIn := params.TokenAmountIn.Amount

	// A native-flagged side trades at native's own decimals on-chain, not the wrapped-native
	// address's used in p.Tokens; only Arc's differ (18 vs 6). Rescale around the v3 math below.
	if idx := p.GetTokenIndex(originalTokenIn); idx >= 0 && p.staticExtra.IsNative[idx] &&
		p.unwrapNativeU256(&amountIn) {
		return nil, uniswapv3.ErrOverflow
	}

	var beforeSwapResult *BeforeSwapResult
	var afterSwapResult *AfterSwapResult
	var swapResult *pool.CalcAmtOutResult
	var err error
	if p.hook == nil {
		if swapResult, err = p.PoolSimulator.CalcAmtOut(pool.CalcAmtOutParams{
			TokenAmountIn: pool.TokenAmt{Token: originalTokenIn, Amount: amountIn},
			TokenOut:      originalTokenOut,
		}); err != nil {
			return nil, err
		}
	} else if swapResult, beforeSwapResult, afterSwapResult, err = p.calcAmtOutHooked(originalTokenIn, originalTokenOut,
		amountIn); err != nil {
		return nil, err
	}

	// Hook deltas are signed: out -= DeltaUnspecified, out -= HookFee. A negative value below zero is
	// checked after the native rescale, like the big.Int body.
	swapResult.TokenAmountOut.Token = originalTokenOut
	amountOut := &swapResult.TokenAmountOut.Amount
	var outNeg, overflow, outOverflow bool
	v4SwapInfo := SwapInfo{PoolSwapInfo: swapResult.SwapInfo.(PoolSwapInfo)}
	if beforeSwapResult != nil {
		outNeg, overflow = big256.SubSigned(amountOut, outNeg, beforeSwapResult.DeltaUnspecified)
		outOverflow = outOverflow || overflow
		swapResult.Gas += beforeSwapResult.Gas
		v4SwapInfo.HookSwapInfo = beforeSwapResult.SwapInfo
	}
	if afterSwapResult != nil {
		outNeg, overflow = big256.SubSigned(amountOut, outNeg, afterSwapResult.HookFee)
		outOverflow = outOverflow || overflow
		swapResult.Gas += afterSwapResult.Gas
	}
	if outOverflow {
		return nil, uniswapv3.ErrOverflow
	}
	if idx := p.GetTokenIndex(originalTokenOut); idx >= 0 && p.staticExtra.IsNative[idx] {
		if p.chainID == valueobject.ChainIDArc {
			amountOut.Div(amountOut, arcNativeScale)
			outNeg = outNeg && !amountOut.IsZero() // big.Int Quo truncates a small negative to zero
		}
	}
	swapResult.SwapInfo = v4SwapInfo
	if swapResult.RemainingTokenAmountIn != nil {
		swapResult.RemainingTokenAmountIn.Token = originalTokenIn
	}
	if outNeg {
		return nil, ErrInvalidAmountOut
	}
	return swapResult, nil
}

// calcAmtOutHooked runs the hook's BeforeSwap, the v3 swap on the adjusted input, and AfterSwap. It returns
// the v3 result before the hook output deltas are applied.
func (p *PoolSimulator) calcAmtOutHooked(tokenIn, tokenOut string, amountIn uint256.Int) (swapResult *pool.CalcAmtOutResult,
	beforeSwapResult *BeforeSwapResult, afterSwapResult *AfterSwapResult, err error) {
	zeroForOne := p.GetTokenIndex(tokenIn) == 0
	if beforeSwapResult, err = p.hook.BeforeSwap(&BeforeSwapParams{
		CalcOut:         true,
		ZeroForOne:      zeroForOne,
		AmountSpecified: new(uint256.Int).Set(&amountIn),
	}); err != nil {
		return nil, nil, nil, fmt.Errorf("[BeforeSwap] %w", err)
	} else if err = ValidateBeforeSwapResult(beforeSwapResult); err != nil {
		return nil, nil, nil, fmt.Errorf("[BeforeSwap] validation failed: %w", err)
	}

	neg, amountInOverflow := big256.SubSigned(&amountIn, false, beforeSwapResult.DeltaSpecified)
	if neg {
		return nil, nil, nil, ErrInvalidAmountIn
	}

	poolSim := p.PoolSimulator
	if beforeSwapResult.SwapFee >= FeeMax {
		return nil, nil, nil, ErrInvalidFee
	} else if beforeSwapResult.SwapFee > 0 && beforeSwapResult.SwapFee != p.V3Pool.Fee {
		cloned := *poolSim
		clonedV3Pool := *poolSim.V3Pool
		cloned.V3Pool = &clonedV3Pool
		cloned.V3Pool.Fee = beforeSwapResult.SwapFee
		poolSim = &cloned
	}

	if amountInOverflow {
		return nil, nil, nil, uniswapv3.ErrOverflow
	} else if amountIn.IsZero() {
		// The hook consumed the whole input, so its BeforeSwap delta fully sets the output. Skip the v3
		// tick math, which errors on a zero input. The all-nil PoolSwapInfo tells UpdateState v3 was skipped.
		swapResult = &pool.CalcAmtOutResult{
			TokenAmountOut: &pool.TokenAmt{Token: tokenOut},
			SwapInfo:       PoolSwapInfo{},
		}
	} else if swapResult, err = poolSim.CalcAmtOut(pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{Token: tokenIn, Amount: amountIn},
		TokenOut:      tokenOut,
	}); err != nil {
		return nil, nil, nil, err
	}

	if afterSwapResult, err = p.hook.AfterSwap(&AfterSwapParams{
		BeforeSwapParams: &BeforeSwapParams{
			CalcOut:         true,
			ZeroForOne:      zeroForOne,
			AmountSpecified: new(uint256.Int).Set(&amountIn),
		},
		AmountIn:  new(uint256.Int).Set(&amountIn),
		AmountOut: new(uint256.Int).Set(&swapResult.TokenAmountOut.Amount),
	}); err != nil {
		return nil, nil, nil, fmt.Errorf("[AfterSwap] %w", err)
	} else if err = ValidateAfterSwapResult(afterSwapResult); err != nil {
		return nil, nil, nil, fmt.Errorf("[AfterSwap] validation failed: %w", err)
	}
	return swapResult, beforeSwapResult, afterSwapResult, nil
}

// CalcAmtIn reverses exact-in with uint256 amounts. It shadows the embedded v3 CalcAmtIn, which would skip this
// pool's hook logic.
func (p *PoolSimulator) CalcAmtIn(params pool.CalcAmtInParams) (*pool.CalcAmtInResult, error) {
	originalTokenOut, originalTokenIn := params.TokenAmountOut.Token, params.TokenIn
	amountOut := params.TokenAmountOut.Amount

	// A native-flagged side trades at native's own decimals on-chain, not the wrapped-native
	// address's used in p.Tokens; only Arc's differ (18 vs 6). Rescale around the v3 math below.
	if idx := p.GetTokenIndex(originalTokenOut); idx >= 0 && p.staticExtra.IsNative[idx] &&
		p.unwrapNativeU256(&amountOut) {
		amountOut.Set(big256.UMax) // the big.Int body kept the exact product, which v3 rejects as above the reserves
	}

	var beforeSwapResult *BeforeSwapResult
	var afterSwapResult *AfterSwapResult
	var swapResult *pool.CalcAmtInResult
	var err error
	if p.hook == nil {
		if swapResult, err = p.PoolSimulator.CalcAmtIn(pool.CalcAmtInParams{
			TokenAmountOut: pool.TokenAmt{Token: originalTokenOut, Amount: amountOut},
			TokenIn:        originalTokenIn,
		}); err != nil {
			return nil, err
		}
	} else if swapResult, beforeSwapResult, afterSwapResult, err = p.calcAmtInHooked(originalTokenIn, originalTokenOut,
		amountOut); err != nil {
		return nil, err
	}

	// Hook deltas are signed: in += DeltaUnspecified, in += HookFee. A negative value below zero is
	// checked after the native rescale, like the big.Int body.
	swapResult.TokenAmountIn.Token = originalTokenIn
	amountIn := &swapResult.TokenAmountIn.Amount
	var inNeg, inOverflow, overflow bool
	v4SwapInfo := SwapInfo{PoolSwapInfo: swapResult.SwapInfo.(PoolSwapInfo)}
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
	if idx := p.GetTokenIndex(originalTokenIn); idx >= 0 && p.staticExtra.IsNative[idx] &&
		p.chainID == valueobject.ChainIDArc {
		amountIn.Div(amountIn, arcNativeScale)
		inNeg = inNeg && !amountIn.IsZero() // big.Int Quo truncates a small negative to zero
	}
	swapResult.SwapInfo = v4SwapInfo
	if swapResult.RemainingTokenAmountOut != nil {
		swapResult.RemainingTokenAmountOut.Token = originalTokenOut
	}
	if inNeg {
		return nil, ErrInvalidAmountIn
	}
	return swapResult, nil
}

// calcAmtInHooked runs the hook's BeforeSwap, the v3 swap on the adjusted output, and AfterSwap. It returns
// the v3 result before the hook input deltas are applied.
func (p *PoolSimulator) calcAmtInHooked(tokenIn, tokenOut string, amountOut uint256.Int) (swapResult *pool.CalcAmtInResult,
	beforeSwapResult *BeforeSwapResult, afterSwapResult *AfterSwapResult, err error) {
	zeroForOne := p.GetTokenIndex(tokenOut) == 1
	if beforeSwapResult, err = p.hook.BeforeSwap(&BeforeSwapParams{
		CalcOut:         false,
		ZeroForOne:      zeroForOne,
		AmountSpecified: new(uint256.Int).Set(&amountOut),
	}); err != nil {
		return nil, nil, nil, fmt.Errorf("[BeforeSwap] %w", err)
	} else if err = ValidateBeforeSwapResult(beforeSwapResult); err != nil {
		return nil, nil, nil, fmt.Errorf("[BeforeSwap] validation failed: %w", err)
	}

	neg, amountOutOverflow := big256.AddSigned(&amountOut, false, beforeSwapResult.DeltaSpecified)
	if neg {
		return nil, nil, nil, ErrInvalidAmountOut
	}

	poolSim := p.PoolSimulator
	if beforeSwapResult.SwapFee >= FeeMax {
		return nil, nil, nil, ErrInvalidFee
	} else if beforeSwapResult.SwapFee > 0 && beforeSwapResult.SwapFee != p.V3Pool.Fee {
		cloned := *poolSim
		clonedV3Pool := *poolSim.V3Pool
		cloned.V3Pool = &clonedV3Pool
		cloned.V3Pool.Fee = beforeSwapResult.SwapFee
		poolSim = &cloned
	}

	if amountOutOverflow {
		// The big.Int body kept the exact sum, which v3 rejects as above the pool's reserves.
		amountOut.Set(big256.UMax)
	}
	if amountOut.IsZero() {
		// The hook fully provides the output: skip the v3 tick math. The all-nil PoolSwapInfo tells
		// UpdateState v3 was skipped, and the hook's DeltaUnspecified becomes the whole input.
		swapResult = &pool.CalcAmtInResult{
			TokenAmountIn: &pool.TokenAmt{Token: tokenIn},
			SwapInfo:      PoolSwapInfo{},
		}
	} else if swapResult, err = poolSim.CalcAmtIn(pool.CalcAmtInParams{
		TokenAmountOut: pool.TokenAmt{Token: tokenOut, Amount: amountOut},
		TokenIn:        tokenIn,
	}); err != nil {
		return nil, nil, nil, err
	}

	if afterSwapResult, err = p.hook.AfterSwap(&AfterSwapParams{
		BeforeSwapParams: &BeforeSwapParams{
			CalcOut:         false,
			ZeroForOne:      zeroForOne,
			AmountSpecified: new(uint256.Int).Set(&amountOut),
		},
		AmountIn:  new(uint256.Int).Set(&swapResult.TokenAmountIn.Amount),
		AmountOut: new(uint256.Int).Set(&amountOut),
	}); err != nil {
		return nil, nil, nil, fmt.Errorf("[AfterSwap] %w", err)
	} else if err = ValidateAfterSwapResult(afterSwapResult); err != nil {
		return nil, nil, nil, fmt.Errorf("[AfterSwap] validation failed: %w", err)
	}
	return swapResult, beforeSwapResult, afterSwapResult, nil
}

func (p *PoolSimulator) GetExchange() string {
	return p.hook.GetExchange()
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
	if v4SwapInfo.NextStateSqrtRatioX96 == nil {
		return
	}
	params.SwapInfo = v4SwapInfo.PoolSwapInfo

	// p.PoolSimulator tracks reserves at each currency's real on-chain decimals (see
	// NewPoolSimulator); TokenAmountIn/Out here are in the wrapped-native address's decimals
	// for a native-flagged leg, so rescale before applying, mirroring the Unwrap/WrapNativeAmount
	// pair CalcAmountOut applies around the same v3 math.
	if idx := p.GetTokenIndex(params.TokenAmountIn.Token); idx >= 0 && p.staticExtra.IsNative[idx] {
		p.unwrapNativeU256(&params.TokenAmountIn.Amount)
	}
	if idx := p.GetTokenIndex(params.TokenAmountOut.Token); idx >= 0 && p.staticExtra.IsNative[idx] {
		p.unwrapNativeU256(&params.TokenAmountOut.Amount)
	}

	p.PoolSimulator.UpdateState(params)
}

// unwrapNativeU256 rescales a native-flagged amount to the wrapped-native decimals, Arc only.
// Mirrors valueobject.UnwrapNativeAmount on uint256. It reports overflow; UpdateState has no error
// return, so it leaves an overflowing product truncated.
func (p *PoolSimulator) unwrapNativeU256(amount *uint256.Int) (overflow bool) {
	if p.chainID != valueobject.ChainIDArc {
		return false
	}
	_, overflow = amount.MulOverflow(amount, arcNativeScale)
	return overflow
}

var arcNativeScale = uint256.NewInt(1e12)

func (p *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) any {
	tokenInAddress, tokenOutAddress := valueobject.AddrZero, valueobject.AddrZero
	if !p.staticExtra.IsNative[p.GetTokenIndex(tokenIn)] {
		tokenInAddress = common.HexToAddress(tokenIn)
	}
	if !p.staticExtra.IsNative[p.GetTokenIndex(tokenOut)] {
		tokenOutAddress = common.HexToAddress(tokenOut)
	}

	return PoolMetaInfo{
		BlockNumber: p.Info.BlockNumber,
		Vault:       p.staticExtra.VaultAddress,
		PoolManager: p.staticExtra.PoolManagerAddress,
		Permit2Addr: p.staticExtra.Permit2Address,
		TokenIn:     tokenInAddress,
		TokenOut:    tokenOutAddress,
		Fee:         p.staticExtra.Fee,
		Parameters:  p.staticExtra.Parameters,
		HookAddress: p.staticExtra.HooksAddress,
		HookData:    []byte{},
		PriceLimit:  p.GetSqrtPriceLimit(tokenIn == p.GetTokens()[0]),
		SwapFee:     p.Info.SwapFee.Uint64(),

		TicksBasedStateBlock: p.StateBlockMeta(),
	}
}

func (s *PoolSimulator) SwapReceiveNativeIn(tokenIn, tokenOut string, _ valueobject.ChainID) bool {
	meta := s.GetMetaInfo(tokenIn, tokenOut).(PoolMetaInfo)
	return meta.TokenIn == valueobject.AddrZero
}

func (s *PoolSimulator) SwapReturnNativeOut(tokenIn, tokenOut string, _ valueobject.ChainID) bool {
	meta := s.GetMetaInfo(tokenIn, tokenOut).(PoolMetaInfo)
	return meta.TokenOut == valueobject.AddrZero
}
