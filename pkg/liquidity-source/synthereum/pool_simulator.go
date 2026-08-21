package synthereum

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// PoolSimulator prices Synthereum swaps purely from tracked state.
// Token order convention: Info.Tokens[0] = collateral (USDC/EURC), Info.Tokens[1] = synthetic (jEUR).
type PoolSimulator struct {
	pool.Pool
	poolType string
	extra    Extra
	// wrapFactor converts collateral amounts to synthetic amounts for wrapper pools,
	// and doubles as the multi-lp pool's SCALING_FACTOR equivalent (10^(synthDec-collateralDec)).
	wrapFactor *uint256.Int
	// mintScaleBone = 10^(18-collateralDec) * 1e18, the combined multi-lp conversion
	// constant so mint/redeem reproduce _calculateNumberOfTokens/_calculateCollateralAmount
	// (both PreciseUnitMath-floor-rounded) in a single MulDivDown call.
	mintScaleBone *uint256.Int
	gas           Gas
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	if len(entityPool.Tokens) != 2 || len(entityPool.Reserves) != 2 {
		return nil, fmt.Errorf("invalid pool tokens/reserves length: %d/%d",
			len(entityPool.Tokens), len(entityPool.Reserves))
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	tokens := make([]string, len(entityPool.Tokens))
	reserves := make([]*big.Int, len(entityPool.Tokens))
	for i, token := range entityPool.Tokens {
		tokens[i] = token.Address
		reserves[i] = bignumber.NewBig10(entityPool.Reserves[i])
		if reserves[i] == nil {
			// NewBig10 yields nil on a malformed reserve string, and UpdateBalance/
			// CloneState read Info.Reserves by index -- a nil there would panic.
			reserves[i] = new(big.Int)
		}
	}

	collateralDecimals := entityPool.Tokens[0].Decimals
	synthDecimals := entityPool.Tokens[1].Decimals

	// wrapFactor/mintScaleBone are pool-type-specific: only the wrapper's
	// wrap()/unwrap() consume wrapFactor, and only mint()/redeem() consume
	// mintScaleBone. Validating both decimal relationships regardless of
	// poolType would wrongly reject, e.g., a multi-lp pool whose synthetic
	// token happens to use fewer decimals than its collateral (wrapFactor is
	// never read on that path).
	var wrapFactor, mintScaleBone *uint256.Int
	switch staticExtra.PoolType {
	case PoolTypeWrapper:
		if synthDecimals < collateralDecimals {
			return nil, fmt.Errorf("unexpected token decimals: %d < %d", synthDecimals, collateralDecimals)
		}
		wrapFactor = big256.TenPow(synthDecimals - collateralDecimals)
	case PoolTypeMultiLP:
		if collateralDecimals > 18 {
			return nil, fmt.Errorf("unexpected collateral decimals: %d > 18", collateralDecimals)
		}
		mintScaleBone = big256.TenPow(36 - collateralDecimals) // 10^(18-collateralDec) * 1e18
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:     strings.ToLower(entityPool.Address),
				Exchange:    entityPool.Exchange,
				Type:        entityPool.Type,
				Tokens:      tokens,
				Reserves:    reserves,
				BlockNumber: entityPool.BlockNumber,
			},
		},
		poolType:      staticExtra.PoolType,
		extra:         extra,
		wrapFactor:    wrapFactor,
		mintScaleBone: mintScaleBone,
		gas:           defaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	indexIn := p.GetTokenIndex(params.TokenAmountIn.Token)
	indexOut := p.GetTokenIndex(params.TokenOut)
	if indexIn < 0 || indexOut < 0 || indexIn == indexOut {
		return nil, ErrInvalidToken
	}

	if params.TokenAmountIn.Amount == nil {
		return nil, ErrInvalidAmountIn
	}
	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow || amountIn.IsZero() {
		return nil, ErrInvalidAmountIn
	}

	switch p.poolType {
	case PoolTypeMultiLP:
		if indexIn == 0 {
			return p.mint(amountIn, params.TokenAmountIn.Token, params.TokenOut)
		}
		return p.redeem(amountIn, params.TokenOut)
	case PoolTypeWrapper:
		if indexIn == 0 {
			return p.wrap(amountIn, params.TokenAmountIn.Token, params.TokenOut)
		}
		return p.unwrap(amountIn, params.TokenOut)
	default:
		return nil, ErrUnsupportedPoolType
	}
}

// mint quotes collateral -> synthetic on a multi-lp pool, replicating
// SynthereumMultiLpLiquidityPoolLib._calculateMint exactly (PreciseUnitMath floor
// rounding throughout), bounded by maxTokensCapacity. feeAmount = floor(amountIn *
// fee / 1e18); numTokens = floor((amountIn - feeAmount) * 10^(18-collateralDec) *
// 1e18 / price).
func (p *PoolSimulator) mint(amountIn *uint256.Int, tokenIn, tokenOut string) (*pool.CalcAmountOutResult, error) {
	e := &p.extra
	if e.Price == nil || e.Price.IsZero() || e.FeePercentage == nil || e.MaxSynthCap == nil {
		return nil, ErrTradeUnavailable
	}

	var fee uint256.Int
	big256.MulWadDown(&fee, amountIn, e.FeePercentage)
	if fee.Gt(amountIn) {
		return nil, ErrOverflow
	}
	var netCollateral uint256.Int
	netCollateral.Sub(amountIn, &fee)

	var amountOut uint256.Int
	big256.MulDivDown(&amountOut, &netCollateral, p.mintScaleBone, e.Price)
	if amountOut.IsZero() {
		return nil, ErrZeroAmountOut
	}
	if amountOut.Gt(e.MaxSynthCap) {
		return nil, ErrExceedsMaxCapacity
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: tokenIn, Amount: fee.ToBig()},
		Gas:            p.gas.Mint,
	}, nil
}

// redeem quotes synthetic -> collateral on a multi-lp pool, replicating
// SynthereumMultiLpLiquidityPoolLib._calculateRedeem exactly, bounded by
// totalSyntheticTokens. totCollateral = floor(amountIn * price / (10^(18-collateralDec)
// * 1e18)); feeAmount = floor(totCollateral * fee / 1e18); out = totCollateral - feeAmount.
func (p *PoolSimulator) redeem(amountIn *uint256.Int, tokenOut string) (*pool.CalcAmountOutResult, error) {
	e := &p.extra
	if e.Price == nil || e.Price.IsZero() || e.FeePercentage == nil || e.TotalSynth == nil {
		return nil, ErrTradeUnavailable
	}
	if amountIn.Gt(e.TotalSynth) {
		return nil, ErrExceedsRedeemCapacity
	}

	var totCollateral uint256.Int
	big256.MulDivDown(&totCollateral, amountIn, e.Price, p.mintScaleBone)

	var fee uint256.Int
	big256.MulWadDown(&fee, &totCollateral, e.FeePercentage)
	if fee.Gt(&totCollateral) {
		return nil, ErrOverflow
	}
	var amountOut uint256.Int
	amountOut.Sub(&totCollateral, &fee)
	if amountOut.IsZero() {
		return nil, ErrZeroAmountOut
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: tokenOut, Amount: fee.ToBig()},
		Gas:            p.gas.Redeem,
	}, nil
}

// wrap quotes collateral -> synthetic on the fixed-rate wrapper: exactly 1:1 in
// value (scaled by decimals), zero fee. wrap() delegates the deposit leg into the
// underlying ERC4626 vault, which reverts above vault.maxDeposit(wrapper) — bounded
// here when that figure is tracked (nil, i.e. not read this refresh, is treated as
// unbounded rather than failing the quote outright).
func (p *PoolSimulator) wrap(amountIn *uint256.Int, tokenIn, tokenOut string) (*pool.CalcAmountOutResult, error) {
	if p.extra.WrapperMaxDeposit != nil && amountIn.Gt(p.extra.WrapperMaxDeposit) {
		return nil, ErrExceedsWrapCapacity
	}

	var amountOut uint256.Int
	if _, overflow := amountOut.MulOverflow(amountIn, p.wrapFactor); overflow {
		return nil, ErrOverflow
	}
	if amountOut.IsZero() {
		return nil, ErrZeroAmountOut
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: tokenIn, Amount: bignumber.ZeroBI},
		Gas:            p.gas.Wrap,
	}, nil
}

// unwrap quotes synthetic -> collateral on the fixed-rate wrapper: zero fee, bounded
// first by the wrapper's own outstanding-synthetic accounting (totalSyntheticTokens,
// the exact check FixedRateLendingWrapper.unwrap enforces via
// 'require(_synthTokenAmount <= totSynthToken_, "Synth tokens amount too high")'),
// and independently by the collateral actually withdrawable from the vault it
// deposits into (vault.maxWithdraw(wrapper)). Both are exact on-chain limits and
// either can bind: exceeding the first reverts "Synth tokens amount too high", the
// second reverts Morpho's NotEnoughLiquidity() — verified to the unit on Base.
//
// When conversionRate() == 1e18 (the deployed default), unwrap REVERTS on-chain for
// any amount that isn't an exact multiple of the scaling factor ('Wrong synth token
// rounding') rather than flooring the remainder — mirrored here, not silently
// truncated.
func (p *PoolSimulator) unwrap(amountIn *uint256.Int, tokenOut string) (*pool.CalcAmountOutResult, error) {
	e := &p.extra
	if e.WrapperSynthCap == nil || e.WrapperRate == nil {
		return nil, ErrTradeUnavailable
	}

	cap := e.WrapperSynthCap
	if e.WrapperReserve != nil {
		var vaultCap uint256.Int
		if _, overflow := vaultCap.MulOverflow(e.WrapperReserve, p.wrapFactor); !overflow && vaultCap.Lt(cap) {
			cap = &vaultCap
		}
	}
	if amountIn.Gt(cap) {
		return nil, ErrInsufficientWrapReserve
	}

	var amountOut uint256.Int
	if e.WrapperRate.Eq(big256.BONE) {
		var rem uint256.Int
		rem.Mod(amountIn, p.wrapFactor)
		if !rem.IsZero() {
			return nil, ErrWrongSynthTokenRounding
		}
		amountOut.Div(amountIn, p.wrapFactor)
	} else {
		var scaled uint256.Int
		big256.MulDivDown(&scaled, amountIn, big256.BONE, e.WrapperRate)
		amountOut.Div(&scaled, p.wrapFactor)
	}
	if amountOut.IsZero() {
		return nil, ErrZeroAmountOut
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: tokenOut, Amount: bignumber.ZeroBI},
		Gas:            p.gas.Unwrap,
	}, nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn := p.GetTokenIndex(params.TokenAmountIn.Token)
	indexOut := p.GetTokenIndex(params.TokenAmountOut.Token)
	amountIn := fromBig(params.TokenAmountIn.Amount)
	amountOut := fromBig(params.TokenAmountOut.Amount)
	if indexIn < 0 || indexOut < 0 || amountIn == nil || amountOut == nil {
		return
	}

	// Info.Reserves is display/heuristic-only for this protocol (mint/redeem/wrap/
	// unwrap are bounded by the extra.* capacity fields below, not by Reserves), but
	// kept live so it doesn't go stale between tracker refreshes -- same convention
	// as other integrations' UpdateBalance (increase the input side, decrease the
	// output side), clamped at zero.
	p.Info.Reserves[indexIn] = new(big.Int).Add(p.Info.Reserves[indexIn], params.TokenAmountIn.Amount)
	if out := new(big.Int).Sub(p.Info.Reserves[indexOut], params.TokenAmountOut.Amount); out.Sign() >= 0 {
		p.Info.Reserves[indexOut] = out
	} else {
		p.Info.Reserves[indexOut] = big.NewInt(0)
	}

	// fields are reassigned wholesale (copy-on-write) so CloneState can stay shallow
	switch p.poolType {
	case PoolTypeMultiLP:
		if indexIn == 0 { // mint: consumes mint capacity, adds outstanding synthetic supply
			if p.extra.MaxSynthCap != nil {
				p.extra.MaxSynthCap = subClamped(p.extra.MaxSynthCap, amountOut)
			}
			if p.extra.TotalSynth != nil {
				p.extra.TotalSynth = new(uint256.Int).Add(p.extra.TotalSynth, amountOut)
			}
		} else { // redeem: burns outstanding synthetic supply (mint capacity increase is ignored, conservative)
			if p.extra.TotalSynth != nil {
				p.extra.TotalSynth = subClamped(p.extra.TotalSynth, amountIn)
			}
		}
	case PoolTypeWrapper:
		if indexIn == 0 { // wrap: collateral deposited into the vault, outstanding synthetic supply grows
			if p.extra.WrapperReserve != nil {
				p.extra.WrapperReserve = new(uint256.Int).Add(p.extra.WrapperReserve, amountIn)
			}
			if p.extra.WrapperSynthCap != nil {
				p.extra.WrapperSynthCap = new(uint256.Int).Add(p.extra.WrapperSynthCap, amountOut)
			}
			if p.extra.WrapperMaxDeposit != nil {
				p.extra.WrapperMaxDeposit = subClamped(p.extra.WrapperMaxDeposit, amountIn)
			}
		} else { // unwrap: collateral redeemed from the vault, outstanding synthetic supply shrinks
			if p.extra.WrapperReserve != nil {
				p.extra.WrapperReserve = subClamped(p.extra.WrapperReserve, amountOut)
			}
			if p.extra.WrapperSynthCap != nil {
				p.extra.WrapperSynthCap = subClamped(p.extra.WrapperSynthCap, amountIn)
			}
		}
	}
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	// Info.Reserves is written by index in UpdateBalance (unlike extra.*, which is
	// always reassigned wholesale), so the shallow copy above is not enough on its
	// own -- it would leave the clone sharing the original's backing array.
	cloned.Info.Reserves = make([]*big.Int, len(p.Info.Reserves))
	for i, r := range p.Info.Reserves {
		cloned.Info.Reserves[i] = new(big.Int).Set(r)
	}
	return &cloned
}

func (p *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) any {
	return PoolMeta{
		BlockNumber:    p.Info.BlockNumber,
		PoolType:       p.poolType,
		IsCollateralIn: p.GetTokenIndex(tokenIn) == 0,
	}
}

// GetApprovalAddress reports the pool itself: every entry point pulls its input with
// transferFrom or burns it from msg.sender, so the pool is the spender. The encoder
// gets the same answer from its useApproveMaxDexes entry, which is why PoolMeta does
// not repeat it.
func (p *PoolSimulator) GetApprovalAddress(_, _ string) string {
	return p.Info.Address
}

func subClamped(a, b *uint256.Int) *uint256.Int {
	res := new(uint256.Int)
	if b.Gt(a) {
		return res
	}
	return res.Sub(a, b)
}
