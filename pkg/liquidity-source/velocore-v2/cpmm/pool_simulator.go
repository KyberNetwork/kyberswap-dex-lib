package cpmm

import (
	"encoding/json"
	"errors"
	"math/big"
	"strings"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/logger"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/velocore-v2/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/velocore-v2/math/sd59x18"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ErrInvalidToken       = errors.New("invalid token")
	ErrInvalidTokenGrowth = errors.New("invalid token growth")
	ErrInvalidR           = errors.New("invalid r")
	ErrNotFoundR          = errors.New("r not found")
)

type PoolSimulator struct {
	pool.Pool

	poolTokenNumber uint
	weights         []*big.Int
	sumWeight       *big.Int

	fee1e9        uint32
	feeMultiplier *big.Int

	isLastWithdrawInTheSameBlock bool
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var (
		extra       Extra
		staticExtra StaticExtra

		tokens   []string
		reserves []*big.Int
	)

	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	for idx := 0; idx < len(entityPool.Tokens); idx++ {
		tokens = append(tokens, entityPool.Tokens[idx].Address)
		reserves = append(reserves, bignumber.NewBig10(entityPool.Reserves[idx]))
	}

	info := pool.PoolInfo{
		Address:    strings.ToLower(entityPool.Address),
		ReserveUsd: entityPool.ReserveUsd,
		Exchange:   entityPool.Exchange,
		Type:       entityPool.Type,
		Tokens:     tokens,
		Reserves:   reserves,
		Checked:    true,
	}

	return &PoolSimulator{
		Pool:                         pool.Pool{Info: info},
		poolTokenNumber:              staticExtra.PoolTokenNumber,
		weights:                      staticExtra.Weights,
		sumWeight:                    staticExtra.Weights[0],
		fee1e9:                       extra.Fee1e9,
		feeMultiplier:                extra.FeeMultiplier,
		isLastWithdrawInTheSameBlock: false,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	tokens, r := p.newVelocoreExecuteParams(tokenAmountIn, tokenOut)

	result, err := p.velocoreExecute(tokens, r)
	if err != nil {
		return nil, err
	}

	var amountOut *big.Int
	for i, token := range tokens {
		if strings.EqualFold(token, tokenOut) {
			amountOut = new(big.Int).Neg(result.R[i])
			break
		}
	}
	if amountOut == nil {
		return nil, ErrNotFoundR
	}

	swapInfo := SwapInfo{
		IsFeeMultiplierUpdated: result.IsFeeMultiplierUpdated,
		FeeMultiplier:          result.FeeMultiplier.String(),
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut,
		},
		Fee:      &pool.TokenAmount{},
		Gas:      0,
		SwapInfo: swapInfo,
	}, nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	tokenInIdx := p.GetTokenIndex(params.TokenAmountIn.Token)
	if tokenInIdx < 0 {
		logger.WithFields(logger.Fields{
			"dexID":   p.Pool.Info.Exchange,
			"dexType": p.Pool.Info.Type,
			"token":   params.TokenAmountIn.Token,
		}).Error("can not find token in pool")
		return
	}
	tokenOutIdx := p.GetTokenIndex(params.TokenAmountOut.Token)
	if tokenOutIdx < 0 {
		logger.WithFields(logger.Fields{
			"dexID":   p.Pool.Info.Exchange,
			"dexType": p.Pool.Info.Type,
			"token":   params.TokenAmountOut.Token,
		}).Error("can not find token in pool")
	}

	p.Info.Reserves[tokenInIdx] = new(big.Int).Add(p.Info.Reserves[tokenInIdx], params.TokenAmountIn.Amount)
	p.Info.Reserves[tokenOutIdx] = new(big.Int).Sub(p.Info.Reserves[tokenOutIdx], params.TokenAmountOut.Amount)

	swapInfo, ok := params.SwapInfo.(SwapInfo)
	if ok && swapInfo.IsFeeMultiplierUpdated {
		p.feeMultiplier, _ = new(big.Int).SetString(swapInfo.FeeMultiplier, 10)
	}

	p.isLastWithdrawInTheSameBlock = true
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return Meta{}
}

// https://github.com/velocore/velocore-contracts/blob/c29678e5acbe5e60fc018e08289b49e53e1492f3/src/pools/constant-product/ConstantProductPool.sol#L164
func (p *PoolSimulator) velocoreExecute(tokens []string, r []*big.Int) (*velocoreExecuteResult, error) {
	effectiveFee1e9 := p.getEffectiveFee1e9()
	iLp := unknownInt

	// balances of "tokens"
	a, err := p.getPoolBalances(tokens)
	if err != nil {
		return nil, err
	}

	// weights of "tokens"
	weights := make([]*big.Int, len(tokens))
	for i, token := range tokens {
		if p.isLpToken(token) {
			weights[i] = p.sumWeight // p.weights[0]
			iLp = i
			continue
		}

		weights[i], _ = p.getTokenWeight(token)
		a[i] = new(big.Int).Add(a[i], bigint1)
	}

	var (
		invariantMin, invariantMax *big.Int
		k                          = bigint1e18
		lpInvolved                 = iLp != unknownInt
		lpUnknown                  = lpInvolved && (r[iLp].Cmp(unknownBI) == 0)
	)

	// calculate "k"
	// "k" is used to split "r" into taxable and non-taxable components
	if lpInvolved {
		_, invariantMin, invariantMax, err = p.getInvariant()
		if err != nil {
			return nil, err
		}
		if lpUnknown {
			kw := bigint0
			for i := range tokens {
				if r[i].Cmp(unknownBI) == 0 {
					if i != iLp {
						kw = new(big.Int).Add(kw, weights[i])
					}
					continue
				}
				balanceRatio := new(big.Int).Div(
					new(big.Int).Mul(new(big.Int).Add(a[i], r[i]), bigint1e18),
					a[i],
				)
				k = new(big.Int).Add(k, new(big.Int).Mul(weights[i], balanceRatio))
			}
			k = new(big.Int).Div(k, new(big.Int).Sub(p.sumWeight, kw))
		} else {
			k = new(big.Int).Div(
				new(big.Int).Mul(bigint1e18, new(big.Int).Sub(invariantMax, r[iLp])),
				invariantMax,
			)
		}
	}

	// calculate "requestedGrowth1e18"
	// which equals to:
	// Pi[ ((b - fee(b - a)) / a) ^ w ]
	var (
		requestedGrowth1e18 = bigint1e18
		sumUnknownWeight    = bigint0
		sumKnownWeight      = bigint0
	)
	for i := range tokens {
		if r[i].Cmp(unknownBI) == 0 {
			if i != iLp {
				sumUnknownWeight = new(big.Int).Add(sumUnknownWeight, weights[i])
			}
			continue
		}

		var tokenGrowth1e18 *big.Int
		if i == iLp {
			newInvariant := new(big.Int).Sub(invariantMax, r[iLp])
			tokenGrowth1e18 = new(big.Int).Div(
				new(big.Int).Mul(bigint1e18, invariantMin),
				newInvariant,
			)
		} else {
			sumKnownWeight = new(big.Int).Add(sumKnownWeight, weights[i])
			b := new(big.Int).Add(a[i], r[i])

			var (
				aPrime *big.Int
				bPrime *big.Int
				fee    = bigint0
			)
			if k.Cmp(bigint1e18) > 0 {
				aPrime = a[i]
				bPrime = new(big.Int).Div(new(big.Int).Mul(b, bigint1e18), k)
			} else {
				aPrime = new(big.Int).Div(new(big.Int).Mul(a[i], k), bigint1e18)
				bPrime = b
			}

			if bPrime.Cmp(aPrime) > 0 {
				f := math.Common.CeilDivUnsafe(
					uint256.MustFromBig(new(big.Int).Mul(new(big.Int).Sub(bPrime, aPrime), effectiveFee1e9)),
					uint256.NewInt(1e9),
				)

				fee = f.ToBig()
			}

			tokenGrowth1e18 = new(big.Int).Div(
				new(big.Int).Mul(bigint1e18, new(big.Int).Sub(b, fee)),
				a[i],
			)
		}

		oneHundred := big.NewInt(100)
		lo := new(big.Int).Div(bigint1e18, oneHundred) // 0.01e18
		hi := new(big.Int).Mul(bigint1e18, oneHundred) // 100e18
		if tokenGrowth1e18.Cmp(lo) <= 0 || tokenGrowth1e18.Cmp(hi) >= 0 {
			return p.velocoreExecuteFallback(tokens, r)
		}

		rpow, err := math.Common.RPow(
			uint256.MustFromBig(tokenGrowth1e18),
			uint256.MustFromBig(weights[i]),
			uint256.NewInt(1e18),
		)
		if err != nil {
			return nil, err
		}
		requestedGrowth1e18 = new(big.Int).Div(
			new(big.Int).Mul(requestedGrowth1e18, rpow.ToBig()),
			bigint1e18,
		)

		// this statement is not actually needed because we use *big.Int instead of uint256.
		// it's here to make the code more similar to the original.
		if requestedGrowth1e18.Cmp(bigint0) <= 0 {
			return nil, ErrInvalidTokenGrowth
		}
	}

	if k.Cmp(bigint1e18) < 0 {
		x := new(big.Int).Sub(
			bigint1e18,
			new(big.Int).Div(
				new(big.Int).Mul(new(big.Int).Sub(bigint1e18, k), effectiveFee1e9),
				bigint1e9,
			),
		)
		n := new(big.Int).Sub(new(big.Int).Sub(p.sumWeight, sumUnknownWeight), sumKnownWeight)
		unaccountedFeeAsGrowth1e18, err := math.Common.RPow(
			uint256.MustFromBig(x),
			uint256.MustFromBig(n),
			uint256.NewInt(1e18),
		)
		if err != nil {
			return nil, err
		}

		requestedGrowth1e18 = new(big.Int).Div(
			new(big.Int).Mul(requestedGrowth1e18, unaccountedFeeAsGrowth1e18.ToBig()),
			bigint1e18,
		)
	}

	var g_, g *big.Int
	w := sumUnknownWeight
	if lpUnknown {
		w = new(big.Int).Sub(w, p.sumWeight)
	}
	if w.Cmp(bigint0) == 0 {
		return nil, ErrInvalidR
	}

	g_, g, err = powReciprocal(requestedGrowth1e18, new(big.Int).Neg(w))
	if err != nil {
		return nil, err
	}

	// calculate unknown "r_i"
	for i := range tokens {
		if r[i].Cmp(unknownBI) != 0 {
			continue
		}

		if i != iLp {
			bU256, err := math.Common.CeilDiv(
				uint256.MustFromBig(new(big.Int).Mul(g, a[i])),
				uint256.NewInt(1e18),
			)
			b := bU256.ToBig()
			if err != nil {
				return nil, err
			}
			var (
				fee            = bigint0
				aPrime, bPrime *big.Int
			)
			if k.Cmp(bigint1e18) > 0 {
				aPrime = a[i]
				_bPrime, err := math.Common.CeilDiv(
					uint256.MustFromBig(new(big.Int).Mul(b, bigint1e18)),
					uint256.MustFromBig(k),
				)
				if err != nil {
					return nil, err
				}
				bPrime = _bPrime.ToBig()
			} else {
				aPrime = new(big.Int).Div(new(big.Int).Mul(a[i], k), bigint1e18)
				bPrime = b
			}

			if bPrime.Cmp(aPrime) > 0 {
				bPrimeMinusAPrime := new(big.Int).Sub(bPrime, aPrime)

				v, err := math.Common.CeilDiv(
					uint256.MustFromBig(new(big.Int).Mul(bPrimeMinusAPrime, effectiveFee1e9)),
					uint256.MustFromBig(new(big.Int).Sub(bigint1e9, effectiveFee1e9)),
				)
				if err != nil {
					return nil, err
				}
				fee = new(big.Int).Sub(v.ToBig(), bPrimeMinusAPrime)
			}

			r[i] = new(big.Int).Sub(new(big.Int).Add(b, fee), a[i])

			continue
		}

		// case unknown lp token "r"

		b := new(big.Int).Div(new(big.Int).Mul(g_, invariantMin), bigint1e18)
		r[i] = new(big.Int).Neg(new(big.Int).Sub(b, invariantMax))
	}

	var (
		isFeeMultiplierUpdated bool
		newFeeMultiplier       = bigint0
	)
	if lpInvolved && r[iLp].Cmp(bigint0) > 0 {
		newFeeMultiplier = p.feeMultiplier
		if !p.isLastWithdrawInTheSameBlock {
			newFeeMultiplier = bigint1e9
		}
		newFeeMultiplier = new(big.Int).Div(
			new(big.Int).Mul(newFeeMultiplier, invariantMax),
			new(big.Int).Sub(invariantMax, r[iLp]),
		)
		isFeeMultiplierUpdated = true
	}

	return &velocoreExecuteResult{
		Tokens:                 tokens,
		R:                      r,
		FeeMultiplier:          newFeeMultiplier,
		IsFeeMultiplierUpdated: isFeeMultiplierUpdated,
	}, nil
}

// https://github.com/velocore/velocore-contracts/blob/c29678e5acbe5e60fc018e08289b49e53e1492f3/src/pools/constant-product/ConstantProductLibrary.sol#L25
func (p *PoolSimulator) velocoreExecuteFallback(tokens []string, r_ []*big.Int) (*velocoreExecuteResult, error) {
	var (
		t   = p.Info.Tokens
		a   = p.Info.Reserves
		idx = make([]int, len(tokens))
		w   = p.weights

		err error
	)

	fee1e18 := new(big.Int).Mul(big.NewInt(int64(p.fee1e9)), bigint1e9)
	if p.isLastWithdrawInTheSameBlock {
		fee1e18 = new(big.Int).Div(new(big.Int).Mul(fee1e18, p.feeMultiplier), bigint1e9)
	}
	additionalMultiplier := bigint1e9

	r := make([]*big.Int, len(t))
	j := 1
	for i, token := range tokens {
		if tokens[i] == t[0] {
			idx[i] = 0
			r[0] = r_[i]
		} else {
			for token != t[j] {
				j++
			}
			idx[i] = j
			r[j] = r_[i]
		}
	}

	for i := 1; i < len(w); i++ {
		a[i] = new(big.Int).Add(a[i], bigint1)
	}

	// pre convert
	var (
		r_SD59x18 = make([]*sd59x18.SD59x18, len(r))
		a_SD59x18 = make([]*sd59x18.SD59x18, len(a))
		w_SD59x18 = make([]*sd59x18.SD59x18, len(w))
	)
	for i := 0; i < int(p.poolTokenNumber); i++ {
		var err error
		if r[i].Cmp(unknownBI) != 0 {
			if r_SD59x18[i], err = sd59x18.ConvertSD59x18(r[i]); err != nil {
				return nil, err
			}
		}
		if a_SD59x18[i], err = sd59x18.ConvertSD59x18(a[i]); err != nil {
			return nil, err
		}
		if w_SD59x18[i], err = sd59x18.ConvertSD59x18(w[i]); err != nil {
			return nil, err
		}
	}

	logA := make([]*sd59x18.SD59x18, len(w))

	logInvariantMin := sd59x18.Zero
	for i := 1; i < len(w); i++ {
		logA[i], err = new(sd59x18.SD59x18).Log2(a_SD59x18[i])
		if err != nil {
			return nil, err
		}

		logInvariantMin, err = sd59x18.NewExpr(a_SD59x18[i]).Mul(w_SD59x18[i]).Add(logInvariantMin).Result()
		if err != nil {
			return nil, err
		}
	}

	logInvariantMin, err = new(sd59x18.SD59x18).Div(logInvariantMin, w_SD59x18[0])
	if err != nil {
		return nil, err
	}

	var (
		logK             = sd59x18.Zero
		logGrowth        = sd59x18.Zero
		sumUnknownWeight = sd59x18.Zero
	)
	if r[0].Cmp(unknownBI) == 0 {
		kw := bigint0
		for i := 1; i < len(w); i++ {
			if r[i].Cmp(unknownBI) == 0 {
				kw = new(big.Int).Add(kw, w[i])
				continue
			}

			ai_add_ri_sd59x18, err := sd59x18.ConvertSD59x18(new(big.Int).Add(a[i], r[i]))
			if err != nil {
				return nil, err
			}

			logK, err = sd59x18.NewExpr(ai_add_ri_sd59x18).
				Log2().
				Mul(w_SD59x18[i]).
				Sub(logA[i]).
				Add(logK).Result()
			if err != nil {
				return nil, err
			}
		}

		w0_sub_kw_sd59x18, err := sd59x18.ConvertSD59x18(new(big.Int).Sub(w[0], kw))
		if err != nil {
			return nil, err
		}
		logK, err = new(sd59x18.SD59x18).Div(logK, w0_sub_kw_sd59x18)
		if err != nil {
			return nil, err
		}
		sumUnknownWeight = new(sd59x18.SD59x18).Sub(sumUnknownWeight, w_SD59x18[0])

	} else if r[0].Cmp(bigint0) != 0 {
		x, err := sd59x18.NewExpr(logInvariantMin).Exp2().Sub(r_SD59x18[0]).Result()
		if err != nil {
			return nil, err
		}

		one_sd59x18, err := sd59x18.ConvertSD59x18(bigint1)
		if err != nil {
			return nil, err
		}

		v := new(sd59x18.SD59x18).Ternary(
			new(sd59x18.SD59x18).Lt(x, one_sd59x18),
			one_sd59x18,
			x,
		)

		logK, err = sd59x18.NewExpr(v).Log2().Sub(logInvariantMin).Result()
		if err != nil {
			return nil, err
		}

		if new(sd59x18.SD59x18).Lt(logK, sd59x18.Zero) {
			t, err := sd59x18.NewExpr(logK).Neg().Exp2().Result()
			if err != nil {
				return nil, err
			}

			additionalMultiplier = new(big.Int).Quo(sd59x18.IntoInt256(t), bigint1e9)
		}

		logGrowth, err = sd59x18.NewExpr(logK).Neg().Mul(w_SD59x18[0]).Result()
		if err != nil {
			return nil, err
		}
	}

	k, err := new(sd59x18.SD59x18).Exp2(logK)
	if err != nil {
		return nil, err
	}

	for i := 1; i < len(w); i++ {
		if r[i].Cmp(unknownBI) == 0 {
			sumUnknownWeight = new(sd59x18.SD59x18).Add(sumUnknownWeight, w_SD59x18[i])
			continue
		}

		b, err := sd59x18.ConvertSD59x18(new(big.Int).Add(a[i], r[i]))
		if err != nil {
			return nil, err
		}

		fee := sd59x18.Zero

		aPrime, err := new(sd59x18.SD59x18).Mul(
			a_SD59x18[i],
			new(sd59x18.SD59x18).Ternary(
				new(sd59x18.SD59x18).Gt(k, sd59x18.Zero),
				sd59x18.SD(bigint1e18),
				k,
			),
		)
		if err != nil {
			return nil, err
		}

		bPrime, err := new(sd59x18.SD59x18).Div(
			b,
			new(sd59x18.SD59x18).Ternary(
				new(sd59x18.SD59x18).Gt(k, sd59x18.Zero),
				k,
				sd59x18.SD(bigint1e18),
			),
		)
		if err != nil {
			return nil, err
		}

		if new(sd59x18.SD59x18).Gt(bPrime, aPrime) {
			fee, err = sd59x18.NewExpr(bPrime).Sub(aPrime).Mul(sd59x18.SD(fee1e18)).Result()
			if err != nil {
				return nil, err
			}
		}

		logGrowth, err = sd59x18.NewExpr(b).Sub(fee).Log2().Sub(logA[i]).Mul(w_SD59x18[i]).Add(logGrowth).Result()
		if err != nil {
			return nil, err
		}
	}

	logG, err := sd59x18.NewExpr(logGrowth).Neg().Div(sumUnknownWeight).Result()
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(w); i++ {
		if r[i].Cmp(unknownBI) != 0 {
			continue
		}

		if i != 0 {
			logB, err := sd59x18.NewExpr(logG).Add(logA[i]).Add(sd59x18.SD(big.NewInt(100000))).Result()
			if err != nil {
				return nil, err
			}

			b, err := new(sd59x18.SD59x18).Exp2(logB)
			if err != nil {
				return nil, err
			}

			aPrime, err := new(sd59x18.SD59x18).Mul(
				a_SD59x18[i],
				new(sd59x18.SD59x18).Ternary(
					new(sd59x18.SD59x18).Gt(k, sd59x18.Zero),
					sd59x18.SD(bigint1e18),
					k,
				),
			)
			if err != nil {
				return nil, err
			}

			bPrime, err := new(sd59x18.SD59x18).Div(
				b,
				new(sd59x18.SD59x18).Ternary(
					new(sd59x18.SD59x18).Gt(k, sd59x18.Zero),
					k,
					sd59x18.SD(bigint1e18),
				),
			)
			if err != nil {
				return nil, err
			}

			if new(sd59x18.SD59x18).Gt(bPrime, aPrime) {
				b, err = sd59x18.NewExpr(bPrime).Sub(aPrime).Div(
					new(sd59x18.SD59x18).Sub(sd59x18.SD(bigint1e18), sd59x18.SD(fee1e18)),
				).Add(b).Sub(new(sd59x18.SD59x18).Sub(bPrime, aPrime)).Result()
				if err != nil {
					return nil, err
				}
			}

			r[i] = sd59x18.ConvertBI(new(sd59x18.SD59x18).Sub(b, a_SD59x18[i]))

			continue
		}

		logB := new(sd59x18.SD59x18).Add(logG, logInvariantMin)
		t, err := sd59x18.NewExpr(logB).Exp2().SubExpr(
			sd59x18.NewExpr(logInvariantMin).Exp2(),
		).Result()
		if err != nil {
			return nil, err
		}
		r[i] = sd59x18.ConvertBI(t)

		if new(sd59x18.SD59x18).Lt(logG, sd59x18.Zero) {
			u, err := sd59x18.NewExpr(logG).Neg().Exp2().Result()
			if err != nil {
				return nil, err
			}
			additionalMultiplier = new(big.Int).Div(
				sd59x18.IntoInt256(u),
				bigint1e9,
			)
		}
	}

	var (
		isFeeMultiplierUpdated bool
		newFeeMultiplier       = bigint0
	)
	if additionalMultiplier.Cmp(bigint1e9) > 0 {
		newFeeMultiplier = additionalMultiplier
		if p.isLastWithdrawInTheSameBlock {
			newFeeMultiplier = new(big.Int).Div(
				new(big.Int).Mul(additionalMultiplier, p.feeMultiplier),
				bigint1e9,
			)
		}
		isFeeMultiplierUpdated = true
	}
	for i := range tokens {
		r_[i] = r[idx[i]]
	}

	return &velocoreExecuteResult{
		Tokens:                 tokens,
		R:                      r_,
		FeeMultiplier:          newFeeMultiplier,
		IsFeeMultiplierUpdated: isFeeMultiplierUpdated,
	}, nil
}

func (p *PoolSimulator) getEffectiveFee1e9() *big.Int {
	effectiveFee1e9 := big.NewInt(int64(p.fee1e9))
	if !p.isLastWithdrawInTheSameBlock {
		return effectiveFee1e9
	}
	effectiveFee1e9 = new(big.Int).Div(
		new(big.Int).Mul(effectiveFee1e9, p.feeMultiplier),
		bigint1e9,
	)
	return effectiveFee1e9
}

func (p *PoolSimulator) getPoolBalances(tokens []string) ([]*big.Int, error) {
	tokenToReserve := make(map[string]*big.Int)
	for i, token := range p.Info.Tokens {
		tokenToReserve[token] = p.Info.Reserves[i]
	}
	var balances []*big.Int
	for _, token := range tokens {
		balance, ok := tokenToReserve[token]
		if !ok {
			return nil, ErrInvalidToken
		}
		balances = append(balances, balance)
	}
	return balances, nil
}

func (p *PoolSimulator) newVelocoreExecuteParams(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) ([]string, []*big.Int) {
	tokens := []string{tokenAmountIn.Token, tokenOut}
	amounts := []*big.Int{tokenAmountIn.Amount, unknownBI}
	return tokens, amounts
}

func (p *PoolSimulator) isLpToken(token string) bool {
	return token == p.Info.Address
}

func (p *PoolSimulator) getTokenWeight(token string) (*big.Int, error) {
	for i, tok := range p.Info.Tokens {
		if tok == token {
			return p.weights[i], nil
		}
	}
	return nil, ErrInvalidToken
}

func (p *PoolSimulator) getInvariant() (*big.Int, *big.Int, *big.Int, error) {
	/*
		https://docs.velocore.xyz/technical-docs/pool-specifics/generalized-cpmm#calculating-unknown-dimensions-in-a-zero-fee-scenario

		So we have the following equation:
		D = [Pi(xi^wi)]^(1/sum(wi)) (i=1..n)
	*/

	balances := p.Info.Reserves
	if p.poolTokenNumber-lpTokenNumber == 2 && p.weights[1].Cmp(p.weights[2]) == 0 {
		prod := new(big.Int).Mul(
			new(big.Int).Add(balances[1], bigint1),
			new(big.Int).Add(balances[2], bigint1),
		)
		inv := new(big.Int).Sqrt(prod)
		ret0 := balances[0]
		invariantMin := inv
		invariantMax := inv
		invSquare := new(big.Int).Mul(inv, inv)
		if invSquare.Cmp(prod) < 0 {
			invariantMax = new(big.Int).Add(inv, bigint1)
		}
		return ret0, invariantMin, invariantMax, nil
	}

	logInvariant := sd59x18.Zero
	for i := 1; i < len(p.weights); i++ {
		b, err := sd59x18.ConvertSD59x18(new(big.Int).Add(balances[i], bigint1))
		if err != nil {
			return nil, nil, nil, err
		}
		g, err := new(sd59x18.SD59x18).Log2(b)
		if err != nil {
			return nil, nil, nil, err
		}

		w, err := sd59x18.ConvertSD59x18(p.weights[i])
		if err != nil {
			return nil, nil, nil, err
		}

		logInvariant, err = sd59x18.NewExpr(g).Mul(w).Add(logInvariant).Result()
		if err != nil {
			return nil, nil, nil, err
		}
	}

	sumW, err := sd59x18.ConvertSD59x18(p.sumWeight)
	if err != nil {
		return nil, nil, nil, err
	}

	logInvariant, err = new(sd59x18.SD59x18).Div(logInvariant, sumW)
	if err != nil {
		return nil, nil, nil, err
	}

	inv, err := new(sd59x18.SD59x18).Exp2(logInvariant)
	if err != nil {
		return nil, nil, nil, err
	}

	invMin := sd59x18.ConvertBI(inv)

	var invMax *big.Int
	{
		x, err := math.Common.CeilDiv(
			new(uint256.Int).Mul(
				uint256.MustFromBig(invMin),
				uint256.NewInt(1e18+1e5),
			),
			uint256.NewInt(1e18),
		)
		if err != nil {
			return nil, nil, nil, err
		}

		invMax = new(big.Int).Add(x.ToBig(), bigint1)
	}

	return bigint0, invMin, invMax, nil
}

func powReciprocal(x1e18, n *big.Int) (*big.Int, *big.Int, error) {
	if n.Cmp(bigint0) == 0 || x1e18.Cmp(bigint1e18) == 0 {
		return bigint1e18, bigint1e18, nil
	}

	if n.Cmp(bigint1) == 0 {
		return x1e18, x1e18, nil
	}

	if n.Cmp(new(big.Int).Neg(bigint1)) == 0 {
		bigint1e18Square := new(big.Int).Mul(bigint1e18, bigint1e18)

		v, err := math.Common.CeilDiv(
			uint256.MustFromBig(bigint1e18Square),
			uint256.MustFromBig(x1e18),
		)
		if err != nil {
			return nil, nil, err
		}

		return new(big.Int).Quo(bigint1e18Square, x1e18), v.ToBig(), nil
	}

	if n.Cmp(bigint2) == 0 {
		x1e18Mul1e18 := new(big.Int).Mul(x1e18, bigint1e18)
		s := new(big.Int).Sqrt(x1e18Mul1e18)
		if new(big.Int).Mul(s, s).Cmp(x1e18Mul1e18) < 0 {
			return s, new(big.Int).Add(s, bigint1), nil
		}
		return s, s, nil
	}

	if n.Cmp(new(big.Int).Neg(bigint2)) == 0 {
		x1e18Mul1e18 := new(big.Int).Mul(x1e18, bigint1e18)
		s := new(big.Int).Sqrt(x1e18Mul1e18)
		ss := s
		if new(big.Int).Mul(s, s).Cmp(x1e18Mul1e18) < 0 {
			ss = new(big.Int).Add(s, bigint1)
		}
		square1e18 := new(big.Int).Mul(bigint1e18, bigint1e18)

		v, err := math.Common.CeilDiv(
			uint256.MustFromBig(square1e18),
			uint256.MustFromBig(s),
		)
		if err != nil {
			return nil, nil, err
		}

		return new(big.Int).Div(square1e18, ss), v.ToBig(), nil
	}

	var raw *big.Int
	{
		// uint256 raw = uint256((pow(sd(x1e18.toInt256()), sd(1e18) / convert(n))).intoInt256());
		x := sd59x18.SD(x1e18)
		y, err := new(sd59x18.SD59x18).Div(sd59x18.SD(bigint1e18), sd59x18.SD(n))
		if err != nil {
			return nil, nil, err
		}
		z, err := new(sd59x18.SD59x18).Pow(x, y)
		if err != nil {
			return nil, nil, err
		}

		raw = sd59x18.IntoInt256(z)
	}

	var maxError *big.Int
	{
		v, err := math.Common.CeilDiv(
			new(uint256.Int).Mul(
				uint256.MustFromBig(raw),
				uint256.NewInt(1e4),
			),
			number.Number_1e18,
		)
		if err != nil {
			return nil, nil, err
		}

		maxError = new(big.Int).Add(v.ToBig(), bigint1)
	}

	ret0 := bigint0
	if raw.Cmp(maxError) >= 0 {
		ret0 = new(big.Int).Sub(raw, maxError)
	}
	ret1 := new(big.Int).Add(raw, maxError)
	return ret0, ret1, nil
}
