package twocryptong

import (
	"fmt"
	"math/big"
	"slices"
	"strings"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	pool.Pool

	precisionMultipliers []uint256.Int
	Reserves             []uint256.Int // same as pool.Reserves but use uint256.Int

	Extra       Extra
	StaticExtra StaticExtra

	// stateAfterTrade marks a v3.0.0 pool's state after a trade whose result is not known (see exchangeV3),
	// which is not quoted.
	stateAfterTrade bool

	gas int64
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	sim := &PoolSimulator{}
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &sim.StaticExtra); err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(entityPool.Extra), &sim.Extra); err != nil {
		return nil, err
	}

	var numTokens = len(entityPool.Tokens)
	if numTokens != NumTokens {
		return nil, ErrInvalidNumToken
	}

	if entityPool.Reserves == nil || len(entityPool.Reserves) != numTokens {
		return nil, ErrInvalidReserve
	}

	var tokens = make([]string, numTokens)
	var reservesBI = make([]*big.Int, numTokens)

	sim.Reserves = make([]uint256.Int, numTokens)
	sim.precisionMultipliers = make([]uint256.Int, numTokens)

	for i := range numTokens {
		tokens[i] = entityPool.Tokens[i].Address

		reservesBI[i] = bignumber.NewBig10(entityPool.Reserves[i])
		if err := sim.Reserves[i].SetFromDecimal(entityPool.Reserves[i]); err != nil {
			return nil, err
		}

		sim.precisionMultipliers[i].Exp(
			number.Number_10,
			uint256.NewInt(uint64(18-entityPool.Tokens[i].Decimals)),
		)
	}

	sim.Pool = pool.Pool{
		Info: pool.PoolInfo{
			Address:     strings.ToLower(entityPool.Address),
			SwapFee:     bignumber.ZeroBI,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      tokens,
			Reserves:    reservesBI,
			BlockNumber: entityPool.BlockNumber,
		},
	}

	sim.gas = DefaultGas

	return sim, nil
}

// nativeFeePolicies are v3.0.0 POLICY contracts whose get_fee is pure and returns 0, so the pool charges its own
// dynamic fee (verified source, YBTwocryptoPolicy, Ethereum). Their get_price_scale steers rebalancing only,
// which a first quote does not reach. Any other policy (YBOraclizedPolicy prices its fee off the trade and a
// signed report) is refused.
var nativeFeePolicies = map[string]struct{}{
	"0x0436c0648afd66de23f86f23cb42884304743efa": {}, // crvUSD/WBTC 0x31369866
	"0x038f549d21b9c22a26ada4da2ebf7cfd1c3ca22d": {}, // crvUSD/cbBTC 0x862cb4e9
	"0xe99ee179f1cc62d4a1d3e67dd5541ecb63aca396": {}, // crvUSD/tBTC 0x4f52c3a8
}

// isV3 reports state of a v3.0.0 pool: it has no allowed_extra_profit view, so the trackers store none. v3.0.0
// rebalances on lp_xcp_profit and donation shares within a step range, books admin fees during the swap, and may
// defer its fee and price target to a policy. A pool with a native-fee policy is quoted (v3PreTradeQuotable);
// with the full tracked state its trades also run v3.0.0's tweak_price (exchangeV3), so later quotes see the
// state after them, and without it the state after a trade is refused.
func (t *PoolSimulator) isV3() bool {
	return t.Extra.AllowedExtraProfit == nil
}

// v3PreTradeQuotable reports whether a v3.0.0 pool's first quote is known: the stableswap math, a native-fee
// policy, and no A/gamma ramp ever scheduled (a ramp makes the pool re-solve D on its next trade).
func (t *PoolSimulator) v3PreTradeQuotable() bool {
	_, nativeFee := nativeFeePolicies[t.Extra.Policy]
	return t.Extra.UseCustomMath && nativeFee && t.Extra.FutureAGammaTime == 0
}

// v3Refusal returns why a v3.0.0 pool's state is not quoted, or nil.
func (t *PoolSimulator) v3Refusal() error {
	if t.stateAfterTrade {
		return ErrStateAfterTrade
	}
	if !t.v3PreTradeQuotable() {
		return ErrPoolVersionNotSupported
	}
	return nil
}

// assertBalance is v3.0.0's _assert_balance on the balances a trade leaves, for state without the full v3.0.0
// fields (exchangeV3 checks exactly otherwise): both positive and less than 1000:1 once scaled by price_scale.
// The pool checks after moving its admin share of the fee out of balances[j] and, when the trade rebalances, at
// the new price_scale; neither is modelled here, so a trade within that margin of 1000:1 may pass and revert on
// chain.
func (t *PoolSimulator) assertBalance(i, j int, dx, dy *uint256.Int) error {
	var xp [NumTokens]uint256.Int
	for k := range NumTokens {
		xp[k].Set(&t.Reserves[k])
	}
	xp[i].Add(&xp[i], dx)
	xp[j].Sub(&xp[j], dy)
	xp[0].Mul(&xp[0], &t.precisionMultipliers[0])
	xp[1].Div(number.SafeMul(number.SafeMul(&xp[1], &t.precisionMultipliers[1]), &t.Extra.PriceScale[0]), Precision)
	hi, lo := &xp[0], &xp[1]
	if hi.Lt(lo) {
		hi, lo = lo, hi
	}
	if lo.IsZero() || !new(uint256.Int).Div(hi, lo).Lt(uint256.NewInt(1000)) {
		return ErrUnbalanced
	}
	return nil
}

func (t *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	v3 := t.isV3()
	if v3 {
		if err := t.v3Refusal(); err != nil {
			return nil, err
		}
	}
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut

	var tokenIndexFrom = t.Info.GetTokenIndex(tokenAmountIn.Token)
	var tokenIndexTo = t.Info.GetTokenIndex(tokenOut)
	if tokenIndexFrom < 0 || tokenIndexTo < 0 {
		return nil, fmt.Errorf("tokenIndexFrom %v or tokenIndexTo %v is not correct",
			tokenIndexFrom, tokenIndexTo)
	}

	var amountOut, fee, amount uint256.Int
	amount.SetFromBig(tokenAmountIn.Amount)
	var swapInfo SwapInfo
	err := t.GetDy(
		tokenIndexFrom,
		tokenIndexTo,
		&amount,
		&amountOut, &fee, &swapInfo.K0, swapInfo.Xp[:],
	)
	if err != nil {
		return nil, err
	} else if amountOut.IsZero() {
		return nil, ErrZero
	}
	if v3 {
		if t.v3PostTradeTracked() {
			A, gamma := t._A_gamma()
			err = t.exchangeV3(A, gamma, tokenIndexTo, &amountOut, &fee, &swapInfo)
		} else {
			// Without the full v3.0.0 state only the amount out is known: UpdateBalance marks the state after
			// it as not quotable.
			err = t.assertBalance(tokenIndexFrom, tokenIndexTo, &amount, &amountOut)
		}
		if err != nil {
			return nil, err
		}
		return &pool.CalcAmountOutResult{
			TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut.ToBig()},
			Fee:            &pool.TokenAmount{Token: tokenOut, Amount: fee.ToBig()},
			Gas:            t.gas,
			SwapInfo:       swapInfo,
		}, nil
	}
	A, gamma := t._A_gamma()
	if t.isV21d() {
		err = t.tweakPriceV21d(A, gamma, swapInfo.Xp, &swapInfo.K0, &swapInfo)
	} else {
		err = t.tweak_price(A, gamma, swapInfo.Xp, nil, &swapInfo.K0,
			swapInfo.LastPrices[:], swapInfo.PriceScale[:], swapInfo.PriceOracle[:], &swapInfo.PriceOracleTimestamp,
			&swapInfo.XcpProfit, &swapInfo.D, &swapInfo.VirtualPrice)
	}
	if err != nil {
		return nil, errors.WithMessage(err, "tweak price")
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: fee.ToBig(),
		},
		Gas:      t.gas,
		SwapInfo: swapInfo,
	}, nil
}

func (t *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	if t.isV3() {
		// v3.0.0's get_dx is not checked against the chain.
		if t.stateAfterTrade {
			return nil, ErrStateAfterTrade
		}
		return nil, ErrPoolVersionNotSupported
	}
	tokenIn := param.TokenIn
	tokenAmountOut := param.TokenAmountOut

	var tokenIndexFrom = t.Info.GetTokenIndex(tokenIn)
	var tokenIndexTo = t.Info.GetTokenIndex(tokenAmountOut.Token)
	if tokenIndexFrom < 0 || tokenIndexTo < 0 {
		return nil, fmt.Errorf("tokenIndexFrom %v or tokenIndexTo %v is not correct",
			tokenIndexFrom, tokenIndexTo)
	}

	var amountIn, feeDy, amountOut uint256.Int
	amountOut.SetFromBig(tokenAmountOut.Amount)

	var swapInfo SwapInfo
	err := t.GetDx(
		tokenIndexFrom,
		tokenIndexTo,
		&amountOut,
		&amountIn,
		&feeDy,
		&swapInfo.K0,
		swapInfo.Xp[:],
	)
	if err != nil {
		return nil, err
	} else if amountIn.IsZero() {
		return nil, ErrZero
	}
	// tweak_price only prevents small swaps. commented out as needed updates were not implemented in GetDx
	// A, gamma := t._A_gamma()
	// if err = t.tweak_price(A, gamma, swapInfo.Xp, nil, &swapInfo.K0,
	// 	swapInfo.LastPrices[:], swapInfo.PriceScale[:], &swapInfo.XcpProfit, &swapInfo.D,
	// 	&swapInfo.VirtualPrice); err != nil {
	// 	return nil, errors.WithMessage(err, "tweak price")
	// }

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: amountIn.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenAmountOut.Token,
			Amount: feeDy.ToBig(),
		},
		Gas:      t.gas,
		SwapInfo: swapInfo,
	}, nil
}

func (t *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *t
	cloned.Info.Reserves = slices.Clone(t.Info.Reserves)
	cloned.Reserves = slices.Clone(t.Reserves)
	cloned.Extra.XcpProfit = t.Extra.XcpProfit
	cloned.Extra.D = t.Extra.D
	cloned.Extra.VirtualPrice = t.Extra.VirtualPrice
	return &cloned
}

func (t *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swapInfo, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		logger.Warnf("failed to UpdateBalance for curve-twocrypto-ng %v %v pool, wrong swapInfo type", t.Info.Address,
			t.Info.Exchange)
		return
	}

	input, output := params.TokenAmountIn, params.TokenAmountOut
	inputAmount, outputAmount := input.Amount, output.Amount
	inputIndex, outputIndex := t.GetTokenIndex(input.Token), t.GetTokenIndex(output.Token)

	t.Info.Reserves[inputIndex] = new(big.Int).Add(t.Info.Reserves[inputIndex], inputAmount)
	t.Reserves[inputIndex] = *new(uint256.Int).Add(&t.Reserves[inputIndex], number.SetFromBig(inputAmount))

	t.Info.Reserves[outputIndex] = new(big.Int).Sub(t.Info.Reserves[outputIndex], outputAmount)
	t.Reserves[outputIndex] = *new(uint256.Int).Sub(&t.Reserves[outputIndex], number.SetFromBig(outputAmount))

	if t.isV3() {
		// The admin share of the fee leaves the balances.
		if !swapInfo.AdminFee.IsZero() {
			t.Info.Reserves[outputIndex] = new(big.Int).Sub(t.Info.Reserves[outputIndex], swapInfo.AdminFee.ToBig())
			t.Reserves[outputIndex] = *new(uint256.Int).Sub(&t.Reserves[outputIndex], &swapInfo.AdminFee)
		}
		if !swapInfo.V3Exact {
			// The state after the trade is not known: keep the balances, refuse quotes on this state.
			t.stateAfterTrade = true
			return
		}
		t.Extra.LpXcpProfit = &swapInfo.LpXcpProfit
		t.Extra.PolicyState = swapInfo.PolicyState
	}

	t.Extra.LastPrices = swapInfo.LastPrices[:]
	t.Extra.PriceScale = swapInfo.PriceScale[:]
	t.Extra.PriceOracle = swapInfo.PriceOracle[:]
	t.Extra.PriceOracleTimestamp = swapInfo.PriceOracleTimestamp
	t.Extra.XcpProfit = &swapInfo.XcpProfit
	t.Extra.D = &swapInfo.D
	t.Extra.VirtualPrice = &swapInfo.VirtualPrice
	if t.isV21d() || t.isV3() {
		t.Extra.LpSupply = &swapInfo.LpSupply
		t.Extra.DonationShares = &swapInfo.DonationShares
		t.Extra.LastDonationReleaseTs = swapInfo.LastDonationReleaseTs
		t.Extra.LastTimestamp = swapInfo.LastTimestamp
	}
}

func (t *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) any {
	var fromId = t.GetTokenIndex(tokenIn)
	var toId = t.GetTokenIndex(tokenOut)
	meta := curve.Meta{
		TokenInIndex:  fromId,
		TokenOutIndex: toId,
		Underlying:    false,
		BlockNumber:   t.Info.BlockNumber,
	}
	if len(t.StaticExtra.IsNativeCoins) == len(t.Info.Tokens) {
		meta.TokenInIsNative = &t.StaticExtra.IsNativeCoins[fromId]
		meta.TokenOutIsNative = &t.StaticExtra.IsNativeCoins[toId]
	}
	return meta
}

func (s *PoolSimulator) SwapReceiveNativeIn(tokenIn, tokenOut string, _ valueobject.ChainID) bool {
	meta := s.GetMetaInfo(tokenIn, tokenOut).(curve.Meta)
	return meta.TokenInIsNative != nil && *meta.TokenInIsNative
}

func (s *PoolSimulator) SwapReturnNativeOut(tokenIn, tokenOut string, _ valueobject.ChainID) bool {
	meta := s.GetMetaInfo(tokenIn, tokenOut).(curve.Meta)
	return meta.TokenOutIsNative != nil && *meta.TokenOutIsNative
}
