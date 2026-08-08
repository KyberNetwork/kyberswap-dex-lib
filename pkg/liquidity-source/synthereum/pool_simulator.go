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
	// wrapFactor converts collateral amounts to synthetic amounts for wrapper pools (10^(synthDec-collateralDec))
	wrapFactor *uint256.Int
	gas        Gas
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
	}

	collateralDecimals := entityPool.Tokens[0].Decimals
	synthDecimals := entityPool.Tokens[1].Decimals
	if synthDecimals < collateralDecimals {
		return nil, fmt.Errorf("unexpected token decimals: %d < %d", synthDecimals, collateralDecimals)
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
		poolType:   staticExtra.PoolType,
		extra:      extra,
		wrapFactor: big256.TenPow(synthDecimals - collateralDecimals),
		gas:        defaultGas,
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
	case poolTypeMultiLP:
		if indexIn == 0 {
			return p.mint(amountIn, params.TokenAmountIn.Token, params.TokenOut)
		}
		return p.redeem(amountIn, params.TokenOut)
	case poolTypeWrapper:
		if indexIn == 0 {
			return p.wrap(amountIn, params.TokenAmountIn.Token, params.TokenOut)
		}
		return p.unwrap(amountIn, params.TokenOut)
	default:
		return nil, ErrUnsupportedPoolType
	}
}

// mint quotes collateral -> synthetic on a multi-lp pool, linear in the tracked
// probe rate (net of fee), bounded by maxTokensCapacity.
func (p *PoolSimulator) mint(amountIn *uint256.Int, tokenIn, tokenOut string) (*pool.CalcAmountOutResult, error) {
	e := &p.extra
	if e.MintProbeIn == nil || e.MintProbeOut == nil || e.MintProbeIn.IsZero() || e.MaxSynthCap == nil {
		return nil, ErrTradeUnavailable
	}

	var amountOut uint256.Int
	if _, overflow := amountOut.MulDivOverflow(amountIn, e.MintProbeOut, e.MintProbeIn); overflow {
		return nil, ErrOverflow
	}
	if amountOut.IsZero() {
		return nil, ErrZeroAmountOut
	}
	if amountOut.Gt(e.MaxSynthCap) {
		return nil, ErrExceedsMaxCapacity
	}

	var fee uint256.Int
	if e.MintProbeFee != nil {
		big256.MulDivDown(&fee, amountIn, e.MintProbeFee, e.MintProbeIn)
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: tokenIn, Amount: fee.ToBig()},
		Gas:            p.gas.Mint,
	}, nil
}

// redeem quotes synthetic -> collateral on a multi-lp pool, linear in the tracked
// probe rate (net of fee), bounded by totalSyntheticTokens.
func (p *PoolSimulator) redeem(amountIn *uint256.Int, tokenOut string) (*pool.CalcAmountOutResult, error) {
	e := &p.extra
	if e.RedeemProbeIn == nil || e.RedeemProbeOut == nil || e.RedeemProbeIn.IsZero() || e.TotalSynth == nil {
		return nil, ErrTradeUnavailable
	}
	if amountIn.Gt(e.TotalSynth) {
		return nil, ErrExceedsRedeemCapacity
	}

	var amountOut uint256.Int
	if _, overflow := amountOut.MulDivOverflow(amountIn, e.RedeemProbeOut, e.RedeemProbeIn); overflow {
		return nil, ErrOverflow
	}
	if amountOut.IsZero() {
		return nil, ErrZeroAmountOut
	}

	var fee uint256.Int
	if e.RedeemProbeFee != nil {
		big256.MulDivDown(&fee, amountIn, e.RedeemProbeFee, e.RedeemProbeIn)
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: tokenOut, Amount: fee.ToBig()},
		Gas:            p.gas.Redeem,
	}, nil
}

// wrap quotes collateral -> synthetic on the fixed-rate wrapper: exactly 1:1 in
// value (scaled by decimals), zero fee, unbounded capacity.
func (p *PoolSimulator) wrap(amountIn *uint256.Int, tokenIn, tokenOut string) (*pool.CalcAmountOutResult, error) {
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

// unwrap quotes synthetic -> collateral on the fixed-rate wrapper: exactly 1:1 in
// value (scaled by decimals), zero fee, bounded by the collateral the wrapper can
// redeem from its vault.
func (p *PoolSimulator) unwrap(amountIn *uint256.Int, tokenOut string) (*pool.CalcAmountOutResult, error) {
	var amountOut uint256.Int
	amountOut.Div(amountIn, p.wrapFactor)
	if amountOut.IsZero() {
		return nil, ErrZeroAmountOut
	}
	if p.extra.WrapperReserve == nil || amountOut.Gt(p.extra.WrapperReserve) {
		return nil, ErrInsufficientWrapReserve
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: tokenOut, Amount: bignumber.ZeroBI},
		Gas:            p.gas.Unwrap,
	}, nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn := p.GetTokenIndex(params.TokenAmountIn.Token)
	amountIn := fromBig(params.TokenAmountIn.Amount)
	amountOut := fromBig(params.TokenAmountOut.Amount)
	if indexIn < 0 || amountIn == nil || amountOut == nil {
		return
	}

	// fields are reassigned wholesale (copy-on-write) so CloneState can stay shallow
	switch p.poolType {
	case poolTypeMultiLP:
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
	case poolTypeWrapper:
		if p.extra.WrapperReserve != nil {
			if indexIn == 0 { // wrap: collateral is deposited into the vault
				p.extra.WrapperReserve = new(uint256.Int).Add(p.extra.WrapperReserve, amountIn)
			} else { // unwrap: collateral is redeemed from the vault
				p.extra.WrapperReserve = subClamped(p.extra.WrapperReserve, amountOut)
			}
		}
	}
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	return &cloned
}

func (p *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) any {
	return PoolMeta{
		BlockNumber:     p.Info.BlockNumber,
		ApprovalAddress: p.GetApprovalAddress(tokenIn, tokenOut),
	}
}

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
