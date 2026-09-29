package stake

import (
	"math/big"
	"strings"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var (
	rateUnit    = uint256.NewInt(1e18)
	tenDecimals = uint256.NewInt(1e10) // matches Lista's TEN_DECIMALS fee-rate base
)

type PoolSimulator struct {
	pool.Pool

	paused                  bool
	depositRate             *uint256.Int
	withdrawRate            *uint256.Int
	instantWithdrawFeeRate  *uint256.Int
	minBnb                  *uint256.Int
	instantWithdrawEligible bool
	isNativeUnderlying      bool
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	numTokens := len(entityPool.Tokens)
	if numTokens != 2 || numTokens != len(entityPool.Reserves) {
		return nil, ErrInvalidToken
	}

	tokens := make([]string, numTokens)
	reserves := make([]*uint256.Int, numTokens)
	for i := range entityPool.Tokens {
		tokens[i] = entityPool.Tokens[i].Address
		r, err := uint256.FromDecimal(entityPool.Reserves[i])
		if err != nil {
			return nil, err
		}
		reserves[i] = r
	}

	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:     strings.ToLower(entityPool.Address),
				Exchange:    entityPool.Exchange,
				Type:        entityPool.Type,
				Tokens:      tokens,
				Reserves:    []*big.Int{reserves[0].ToBig(), reserves[1].ToBig()},
				BlockNumber: entityPool.BlockNumber,
			},
		},
		paused:                  extra.Paused,
		depositRate:             extra.DepositRate,
		withdrawRate:            extra.WithdrawRate,
		instantWithdrawFeeRate:  extra.InstantWithdrawFeeRate,
		minBnb:                  extra.MinBnb,
		instantWithdrawEligible: extra.InstantWithdrawEligible,
		isNativeUnderlying:      staticExtra.IsNativeUnderlying,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if p.paused {
		return nil, ErrPoolPaused
	}

	tokenInIndex := p.GetTokenIndex(param.TokenAmountIn.Token)
	tokenOutIndex := p.GetTokenIndex(param.TokenOut)
	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return nil, ErrInvalidToken
	}

	amountIn, overflow := uint256.FromBig(param.TokenAmountIn.Amount)
	if overflow {
		return nil, ErrOverflow
	}

	var amountOut uint256.Int
	gas := int64(gasDeposit)

	if tokenInIndex == 0 {
		// deposit: WBNB -> slisBNB, no fee, no cap (deposit() always mints)
		big256.MulDivDown(&amountOut, amountIn, p.depositRate, rateUnit)
	} else {
		// instant withdraw: slisBNB -> WBNB, fee + eligibility + liquidity cap
		if !p.instantWithdrawEligible {
			return nil, ErrInstantWithdrawNotEligible
		}

		var fee uint256.Int
		big256.MulDivDown(&fee, amountIn, p.instantWithdrawFeeRate, tenDecimals)
		burnAmount := new(uint256.Int).Sub(amountIn, &fee)

		big256.MulDivDown(&amountOut, burnAmount, p.withdrawRate, rateUnit)

		if amountOut.Lt(p.minBnb) {
			return nil, ErrAmountTooSmall
		}
		reserveOut, overflow := uint256.FromBig(p.GetReserves()[tokenOutIndex])
		if overflow {
			return nil, ErrOverflow
		}
		if amountOut.Gt(reserveOut) {
			return nil, ErrInsufficientLiquidity
		}
		gas = gasInstantWithdraw
	}

	if amountOut.IsZero() {
		return nil, ErrZeroAmountOut
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: param.TokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: param.TokenAmountIn.Token, Amount: bignumber.ZeroBI},
		Gas:            gas,
	}, nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	tokenInIndex := p.GetTokenIndex(params.TokenAmountIn.Token)
	tokenOutIndex := p.GetTokenIndex(params.TokenAmountOut.Token)

	// Both legs share the same on-chain buffer (amountToDelegate): deposit tops it up, instant
	// withdraw draws it down -- mirror that so a route quoting multiple hops through this pool
	// doesn't over-withdraw beyond what's actually available.
	reserves := make([]*big.Int, len(p.Info.Reserves))
	copy(reserves, p.Info.Reserves)
	reserves[tokenInIndex] = new(big.Int).Add(reserves[tokenInIndex], params.TokenAmountIn.Amount)
	reserves[tokenOutIndex] = new(big.Int).Sub(reserves[tokenOutIndex], params.TokenAmountOut.Amount)
	p.Info.Reserves = reserves
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.Info.Reserves = make([]*big.Int, len(p.Info.Reserves))
	copy(cloned.Info.Reserves, p.Info.Reserves)
	return &cloned
}

func (p *PoolSimulator) CanSwapFrom(address string) []string {
	if strings.EqualFold(p.Info.Tokens[0], address) {
		if p.paused {
			return nil
		}
		return []string{p.Info.Tokens[1]}
	}
	if strings.EqualFold(p.Info.Tokens[1], address) {
		if p.paused || !p.instantWithdrawEligible {
			return nil
		}
		return []string{p.Info.Tokens[0]}
	}
	return nil
}

func (p *PoolSimulator) CanSwapTo(address string) []string {
	if strings.EqualFold(p.Info.Tokens[1], address) {
		if p.paused {
			return nil
		}
		return []string{p.Info.Tokens[0]}
	}
	if strings.EqualFold(p.Info.Tokens[0], address) {
		if p.paused || !p.instantWithdrawEligible {
			return nil
		}
		return []string{p.Info.Tokens[1]}
	}
	return nil
}

// GetMetaInfo reports IsWithdraw and IsNativeUnderlying so the encoder (aggregator-encoding's
// PackListaStakeManager and swapReceiveNativeIn/swapReturnNativeOut) can pick the right
// IListaStakeManager.Action and wrap/unwrap decision without re-deriving either from the token
// addresses -- both come straight from this pool's own state, so they hold regardless of whether
// the underlying happens to be native, wrapped-native, or any other ERC20.
func (p *PoolSimulator) GetMetaInfo(tokenIn, _ string) any {
	return Meta{
		BlockNumber:        p.Info.BlockNumber,
		IsWithdraw:         !strings.EqualFold(p.Info.Tokens[0], tokenIn),
		IsNativeUnderlying: p.isNativeUnderlying,
	}
}

// SwapReceiveNativeIn reports native support on the deposit leg (token0 in): true only when this
// instance was actually configured with a native underlying (StaticExtra.IsNativeUnderlying), not
// merely because tokenIn happens to be a wrapped-native address -- a variant whose deposit takes
// the token via transferFrom would set IsNativeUnderlying=false and correctly report no support
// even though it trades the same wrapped token.
func (p *PoolSimulator) SwapReceiveNativeIn(tokenIn, _ string, _ valueobject.ChainID) bool {
	return p.isNativeUnderlying && strings.EqualFold(p.Info.Tokens[0], tokenIn)
}

// SwapReturnNativeOut mirrors SwapReceiveNativeIn for the withdraw leg (token0 out): on the one
// real contract today, deposit and instantWithdraw both move native currency, so the same
// IsNativeUnderlying fact governs both directions.
func (p *PoolSimulator) SwapReturnNativeOut(_, tokenOut string, _ valueobject.ChainID) bool {
	return p.isNativeUnderlying && strings.EqualFold(p.Info.Tokens[0], tokenOut)
}
