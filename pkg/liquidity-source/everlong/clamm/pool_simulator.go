package everlongclamm

import (
	"math/big"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// defaultGas: a ClammSwapRouter.swap runs the CL swap through the Vault plus the hook's
// beforeSwap (fee-override latch) and afterSwap (output haircut transfer). Base is from
// a live Katana fill (tx 0x7dc8574f…, 522,559 gas total across a few rung crossings).
var defaultGas = uniswapv3.Gas{BaseGas: 450000, CrossInitTickGas: 21000}

// PoolSimulator prices the Everlong CLAMM pool: a PancakeSwap-Infinity CL pool whose sole
// LP is the Everlong ALM (an explicit rung ladder), with a ClammHook that (1) overrides
// the in-pool LP fee to 0, (2) forbids exact-output swaps, and (3) takes the fee as a
// directional OUTPUT haircut latched pre-trade: netOut = grossOut * (WAD - poolFee) / WAD.
type PoolSimulator struct {
	*uniswapv3.PoolSimulator
	StaticExtra     StaticExtra
	PoolFee0For1Wad uint256.Int
	PoolFee1For0Wad uint256.Int
}

var _ = pool.RegisterFactory1(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool, _ valueobject.ChainID) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	// The hook zeroes the in-pool LP fee, so the CL walk always runs at feePips = 0
	// regardless of PoolKey.fee (the haircut is applied on the output instead).
	entityPool.SwapFee = 0
	v3PoolSimulator, err := uniswapv3.NewPoolSimulatorWithExtra(entityPool, &extra.ExtraTickU256,
		uniswapv3.SimulatorConfig{})
	if err != nil {
		return nil, err
	}
	v3PoolSimulator.Gas = defaultGas

	sim := &PoolSimulator{
		PoolSimulator: v3PoolSimulator,
		StaticExtra:   staticExtra,
	}
	if extra.PoolFee0For1Wad != nil {
		sim.PoolFee0For1Wad.Set(extra.PoolFee0For1Wad)
	}
	if extra.PoolFee1For0Wad != nil {
		sim.PoolFee1For0Wad.Set(extra.PoolFee1For0Wad)
	}
	return sim, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	indexIn, indexOut := p.GetTokenIndex(param.TokenAmountIn.Token), p.GetTokenIndex(param.TokenOut)
	if indexIn < 0 || indexOut < 0 || indexIn == indexOut {
		return nil, ErrInvalidToken
	}
	zeroForOne := indexIn == 0

	var amountIn uint256.Int
	if overflow := amountIn.SetFromBig(param.TokenAmountIn.Amount); overflow || amountIn.IsZero() {
		return nil, uniswapv3.ErrOverflow
	}

	// The same unbounded price limits the router uses on-chain (MIN_SQRT_RATIO+1 /
	// MAX_SQRT_RATIO-1): a fill larger than the ALM's liquidity walks past the last rung
	// through empty words (consuming nothing) and reports the unconsumed input.
	sqrtPriceLimitX96 := uniswapv3.MaxSqrtRatioU256
	if zeroForOne {
		sqrtPriceLimitX96 = uniswapv3.MinSqrtRatioU256P1
	}
	state, err := swapExactInput(p.V3Pool, zeroForOne, &amountIn, sqrtPriceLimitX96)
	if err != nil {
		return nil, err
	}

	poolFeeWad := &p.PoolFee1For0Wad
	if zeroForOne {
		poolFeeWad = &p.PoolFee0For1Wad
	}
	var keepWad, netOut uint256.Int
	keepWad.Sub(big256.BONE, poolFeeWad)
	netOut.Div(netOut.Mul(&state.AmountOut, &keepWad), big256.BONE)
	if netOut.IsZero() {
		return nil, ErrZeroAmountOut
	}
	fee := new(uint256.Int).Sub(&state.AmountOut, &netOut)

	remainingTokenAmountIn := &pool.TokenAmount{Token: param.TokenAmountIn.Token, Amount: bignumber.ZeroBI}
	if !state.RemainingAmountIn.IsZero() {
		remainingTokenAmountIn.Amount = state.RemainingAmountIn.ToBig()
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut:         &pool.TokenAmount{Token: param.TokenOut, Amount: netOut.ToBig()},
		Fee:                    &pool.TokenAmount{Token: param.TokenOut, Amount: fee.ToBig()},
		RemainingTokenAmountIn: remainingTokenAmountIn,
		Gas:                    defaultGas.BaseGas + defaultGas.CrossInitTickGas*int64(state.CrossInitTickLoops),
		SwapInfo: uniswapv3.SwapInfo{
			RemainingAmountIn:     &state.RemainingAmountIn,
			NextStateSqrtRatioX96: &state.SqrtPriceX96,
			NextStateLiquidity:    state.Liquidity,
			NextStateTickCurrent:  state.Tick,
		},
	}, nil
}

// CalcAmountIn is intentionally rejected: the ClammHook reverts exact-output swaps.
func (p *PoolSimulator) CalcAmountIn(pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	return nil, ErrExactOutNotSupported
}

// UpdateBalance moves the CL state to the post-swap price/liquidity/tick. Reserves drop
// by the GROSS output (net + haircut): both legs leave the pool's ladder — the haircut
// is the hook's fee take, not retained pool liquidity.
func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	grossOut := new(big.Int).Add(params.TokenAmountOut.Amount, params.Fee.Amount)
	params.TokenAmountOut.Amount = grossOut
	params.Fee.Amount = big.NewInt(0)
	p.PoolSimulator.UpdateBalance(params)
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.PoolSimulator = p.PoolSimulator.CloneState().(*uniswapv3.PoolSimulator)
	return &cloned
}

func (p *PoolSimulator) GetMetaInfo(tokenIn, _ string) any {
	return PoolMeta{
		Router:      p.StaticExtra.Router,
		PoolManager: p.StaticExtra.PoolManager,
		Hook:        p.StaticExtra.Hook,
		Fee:         p.StaticExtra.Fee,
		Parameters:  p.StaticExtra.Parameters,
		PriceLimit:  p.GetSqrtPriceLimit(tokenIn == p.Info.Tokens[0]),
	}
}

// GetApprovalAddress: the ClammSwapRouter pulls the input token from the payer.
func (p *PoolSimulator) GetApprovalAddress(_, _ string) string {
	return p.StaticExtra.Router
}
