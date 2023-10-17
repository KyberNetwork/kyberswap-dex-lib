package velocorev2cpmm

import (
	"encoding/json"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/velocorev2-cpmm/sd59x18"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
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

		poolTokenNumber uint
		tokens          []string
		weights         []*big.Int
		sumWeight       *big.Int
		reserves        []*big.Int

		fee1e9        uint32
		feeMultiplier *big.Int
	)

	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}
	fee1e9 = extra.Fee1e9
	feeMultiplier = bignumber.NewBig10(extra.FeeMultiplier)

	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}
	poolTokenNumber = staticExtra.PoolTokenNumber

	sumWeight = zero
	for _, token := range entityPool.Tokens {
		tokens = append(tokens, token.Address)
		weightBI := big.NewInt(int64(token.Weight))
		weights = append(weights, weightBI)
		sumWeight = new(big.Int).Add(sumWeight, weightBI)
	}
	for _, reserve := range entityPool.Reserves {
		reserves = append(reserves, bignumber.NewBig10(reserve))
	}

	info := pool.PoolInfo{
		Address:    strings.ToLower(entityPool.Address),
		ReserveUsd: entityPool.ReserveUsd,
		Exchange:   entityPool.Exchange,
		Type:       entityPool.Type,
		Tokens:     tokens,
		Reserves:   reserves,
		Checked:    false,
	}

	return &PoolSimulator{
		Pool:                         pool.Pool{Info: info},
		poolTokenNumber:              poolTokenNumber,
		weights:                      weights,
		sumWeight:                    sumWeight,
		fee1e9:                       fee1e9,
		feeMultiplier:                feeMultiplier,
		isLastWithdrawInTheSameBlock: false,
	}, nil
}

func (t *PoolSimulator) CalcAmountOut(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	tokens, r := t.newVelocoreExecuteParams(tokenAmountIn, tokenOut)

	effectiveFee1e9 := t.getEffectiveFee1e9()

	iLp := unknownInt
	a, err := t.getPoolBalances(tokens)
	if err != nil {
		return nil, err
	}
	weights := make([]*big.Int, len(tokens))
	for i, token := range tokens {
		if t.isLpToken(token) {
			weights[i] = t.sumWeight
			iLp = i
		} else {
			weights[i], _ = t.getTokenWeight(token)
			a[i] = new(big.Int).Add(a[i], one)
		}
	}

	var (
		invariantMin, invariantMax *big.Int
		k                          = bigint1e18
		lpInvolved                 = iLp != unknownInt
		lpUnknown                  = lpInvolved && (a[iLp].Cmp(unknownBI) == 0)
	)
	if lpInvolved {
		_, invariantMin, invariantMax, err = t.getInvariant()
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
					new(big.Int).Mul(
						new(big.Int).Add(a[i], r[i]),
						bigint1e18,
					),
					a[i],
				)
				k = new(big.Int).Add(k, new(big.Int).Mul(weights[i], balanceRatio))
			}

			k = new(big.Int).Div(k, new(big.Int).Sub(t.sumWeight, kw))
		} else {
			k = new(big.Int).Div(
				new(big.Int).Mul(
					bigint1e18,
					new(big.Int).Sub(invariantMax, r[iLp]),
				),
				invariantMax,
			)
		}
	}

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
				fee    *big.Int
			)

			if k.Cmp(bigint1e18) > 0 {
				aPrime = a[i]
				bPrime = new(big.Int).Div(
					new(big.Int).Mul(b, bigint1e18),
					k,
				)
			} else {
				aPrime = new(big.Int).Div(
					new(big.Int).Mul(a[i], k),
					bigint1e18,
				)
				bPrime = b
			}

			if bPrime.Cmp(aPrime) > 0 {
				fee = ceilDivUnsafe(
					new(big.Int).Mul(
						new(big.Int).Sub(bPrime, aPrime),
						effectiveFee1e9,
					),
					bigint1e9,
				)
			}

			tokenGrowth1e18 = new(big.Int).Div(
				new(big.Int).Mul(
					bigint1e18,
					new(big.Int).Sub(b, fee),
				),
				a[i],
			)
		}

		oneHundred := big.NewInt(100)
		lo := new(big.Int).Div(bigint1e18, oneHundred) // 0.01e18
		hi := new(big.Int).Mul(bOne, oneHundred)       // 100e18
		if tokenGrowth1e18.Cmp(lo) <= 0 || tokenGrowth1e18.Cmp(hi) >= 0 {
			return t.returnLogarithmicSwap(tokenOut, tokens, r)
		}

		requestedGrowth1e18 = new(big.Int).Div(
			new(big.Int).Mul(
				requestedGrowth1e18,
				rpow(tokenGrowth1e18, weights[i], bigint1e18),
			),
			bigint1e18,
		)
		if requestedGrowth1e18.Cmp(zero) <= 0 {
			return nil, ErrInvalidTokenGrowth
		}
	}

	unaccountedFeeAsGrowth1e18 := bigint1e18
	if k.Cmp(bigint1e18) < 0 {
		x := new(big.Int).Sub(
			bigint1e18,
			new(big.Int).Div(
				new(big.Int).Mul(
					new(big.Int).Sub(bigint1e18, k),
					effectiveFee1e9,
				),
				bigint1e9,
			),
		)
		n := new(big.Int).Sub(
			new(big.Int).Sub(t.sumWeight, sumUnknownWeight),
			sumKnownWeight,
		)
		unaccountedFeeAsGrowth1e18 = rpow(x, n, bigint1e18)
		requestedGrowth1e18 = new(big.Int).Div(
			new(big.Int).Mul(
				requestedGrowth1e18,
				unaccountedFeeAsGrowth1e18,
			),
			bigint1e18,
		)
	}

	var g_, g *big.Int
	w := sumUnknownWeight
	if lpUnknown {
		w = new(big.Int).Sub(w, t.sumWeight)
	}
	if w.Cmp(zero) == 0 {
		return nil, ErrInvalidR
	}
	g_, g, err = powReciprocal(requestedGrowth1e18, new(big.Int).Neg(w)) // TODO: check me!
	if err != nil {
		return nil, err
	}

	for i := range tokens {
		if r[i].Cmp(unknownBI) != 0 {
			continue
		}

		if i != iLp {
			b := ceilDiv(new(big.Int).Mul(g, a[i]), bigint1e18)
			var (
				fee            = bigint0
				aPrime, bPrime *big.Int
			)
			if k.Cmp(bigint1e18) > 0 {
				aPrime = a[i]
				bPrime = ceilDiv(new(big.Int).Mul(b, bigint1e18), k)
			} else {
				aPrime = ceilDiv(new(big.Int).Mul(a[i], k), bigint1e18)
				bPrime = b
			}

			bPrimeMinusAPrime := new(big.Int).Sub(bPrime, aPrime)
			if bPrime.Cmp(aPrime) > 0 {
				fee = new(big.Int).Sub(
					ceilDiv(
						new(big.Int).Mul(
							bPrimeMinusAPrime,
							bigint1e9,
						),
						new(big.Int).Sub(bigint1e9, effectiveFee1e9),
					),
					bPrimeMinusAPrime,
				)
			}

			r[i] = new(big.Int).Sub( // TODO: this is amountIn
				new(big.Int).Add(b, fee),
				a[i],
			)

			continue
		}

		b := new(big.Int).Div(
			new(big.Int).Mul(g_, invariantMin),
			bigint1e18,
		)
		r[i] = new(big.Int).Neg(new(big.Int).Sub(b, invariantMax))
	}

	var swapInfo SwapInfo
	if iLp != unknownInt && r[iLp].Cmp(bigint0) > 0 {
		feeMultiplier := t.feeMultiplier
		if !t.isLastWithdrawInTheSameBlock {
			feeMultiplier = bigint1e9
		}
		feeMultiplier = new(big.Int).Div(
			new(big.Int).Mul(feeMultiplier, invariantMax),
			new(big.Int).Sub(invariantMax, r[iLp]),
		)
		swapInfo.NeedToUpdateFeeMultiplier = true
		swapInfo.FeeMultiplierUpdated = feeMultiplier.String()
	}

	var amountOut *big.Int
	for i, token := range tokens {
		if strings.EqualFold(token, tokenOut) {
			amountOut = new(big.Int).Neg(r[i])
			break
		}
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut,
		},
		Fee:      nil, //TODO: implement me!
		Gas:      0,
		SwapInfo: &swapInfo,
	}, nil
}

func (t *PoolSimulator) returnLogarithmicSwap(
	tokenOut string,
	tokens []string, r_ []*big.Int,
) (*pool.CalcAmountOutResult, error) {
	var (
		tt  = t.Info.Tokens
		a   = t.Info.Reserves
		idx = make([]int, len(tokens))
		w   = t.weights
	)

	fee1e18 := new(big.Int).Mul(
		big.NewInt(int64(t.fee1e9)),
		bigint1e9,
	)
	if t.isLastWithdrawInTheSameBlock {
		fee1e18 = new(big.Int).Div(
			new(big.Int).Mul(fee1e18, t.feeMultiplier),
			bigint1e9,
		)
	}
	additionalMultiplier := bigint1e9

	r := make([]*big.Int, len(tt))
	poolTokenIdx := map[string]int{}
	for i, token := range tt {
		poolTokenIdx[token] = i
	}
	for i, token := range tokens {
		j := poolTokenIdx[token]
		idx[i] = j
		r[j] = r_[i]
	}

	logA := make([]sd59x18.SD59x18, len(w))
	logInvariantMin := sd59x18.Zero()
	for i := 1; i < len(w); i++ {
		a[i] = new(big.Int).Add(a[i], bigint1)
		ai, err := sd59x18.ConvertToSD59x18(a[i])
		if err != nil {
			return nil, err
		}
		logA[i], err = sd59x18.Log2(ai)
		if err != nil {
			return nil, err
		}
		logAiMulWi, err := sd59x18.Mul(logA[i], w[i])
		if err != nil {
			return nil, err
		}
		logInvariantMin = sd59x18.Add(logInvariantMin, logAiMulWi)
	}
	w0SD59x18, err := sd59x18.ConvertToSD59x18(w[0])
	if err != nil {
		return nil, err
	}
	logInvariantMin = new(big.Int).Div(logInvariantMin, w0SD59x18)

	var (
		logK             = sd59x18.Zero()
		logGrowth        = sd59x18.Zero()
		sumUnknownWeight = sd59x18.Zero()
	)
	if r[0].Cmp(unknownBI) == 0 {
		kw := bigint0
		for i := 1; i < len(w); i++ {
			if r[i].Cmp(unknownBI) == 0 {
				kw = new(big.Int).Add(kw, w[i])
				continue
			}
			wi, err := sd59x18.ConvertToSD59x18(w[i])
			if err != nil {
				return nil, err
			}
			aiAddRi, err := sd59x18.ConvertToSD59x18(new(big.Int).Add(a[i], r[i]))
			if err != nil {
				return nil, err
			}
			log2AiAddRi, err := sd59x18.Log2(aiAddRi)
			if err != nil {
				return nil, err
			}

			d, err := sd59x18.Mul(wi, sd59x18.Sub(log2AiAddRi, logA[i]))
			if err != nil {
				return nil, err
			}
			logK = sd59x18.Add(logK, d)
		}
		w0SubKw, err := sd59x18.ConvertToSD59x18(new(big.Int).Sub(w[0], kw))
		if err != nil {
			return nil, err
		}
		logK, err = sd59x18.Div(logK, w0SubKw)
		if err != nil {
			return nil, err
		}
		sumUnknownWeight = sd59x18.Sub(sumUnknownWeight, w0SD59x18)
	} else if r[0].Cmp(bigint0) != 0 {
		r0, err := sd59x18.ConvertToSD59x18(r[0])
		if err != nil {
			return nil, err
		}
		exp2LogInvariantMin, err := sd59x18.Exp2(logInvariantMin)
		if err != nil {
			return nil, err
		}
		x := sd59x18.Sub(exp2LogInvariantMin, r0)
		one, err := sd59x18.ConvertToSD59x18(bigint1)
		if err != nil {
			return nil, err
		}
		val := x
		if sd59x18.Lt(x, one) {
			val = one
		}
		logK, err := sd59x18.Log2(val)
		if err != nil {
			return nil, err
		}
		logK = sd59x18.Sub(logK, logInvariantMin)
		if sd59x18.Lt(logK, sd59x18.Zero()) {
			v, err := sd59x18.Exp2(sd59x18.Sub(sd59x18.Zero(), logK))
			if err != nil {
				return nil, err
			}
			var vBI *big.Int = v
			additionalMultiplier = new(big.Int).Div(vBI, bigint1e9)
		}

		logGrowth, err = sd59x18.Mul(
			sd59x18.Sub(sd59x18.Zero(), logK),
			w0SD59x18,
		)
		if err != nil {
			return nil, err
		}
	}

	k, err := sd59x18.Exp2(logK)
	if err != nil {
		return nil, err
	}

	for i := 1; i < len(w); i++ {
		if r[i].Cmp(unknownBI) == 0 {
			wi, err := sd59x18.ConvertToSD59x18(w[i])
			if err != nil {
				return nil, err
			}
			sumUnknownWeight = sd59x18.Add(sumUnknownWeight, wi)
			continue
		}

		b, err := sd59x18.ConvertToSD59x18(new(big.Int).Add(a[i], r[i]))
		if err != nil {
			return nil, err
		}
		var (
			fee            sd59x18.SD59x18 = sd59x18.Zero()
			aPrime, bPrime sd59x18.SD59x18

			zeroSD59x18 = sd59x18.Zero()
		)

		// calculate aPrime
		{
			v := k
			if sd59x18.Lt(zeroSD59x18, logK) {
				v, err = sd59x18.ConvertToSD59x18(bigint1e18)
				if err != nil {
					return nil, err
				}
			}
			ai, err := sd59x18.ConvertToSD59x18(a[i])
			if err != nil {
				return nil, err
			}
			aPrime, err = sd59x18.Mul(ai, v)
			if err != nil {
				return nil, err
			}
		}

		// calculate bPrime
		{
			v, err := sd59x18.ConvertToSD59x18(bigint1e18)
			if err != nil {
				return nil, err
			}
			if sd59x18.Lt(zeroSD59x18, logK) {
				v = k
			}
			bPrime, err = sd59x18.Div(b, v)
			if err != nil {
				return nil, err
			}
		}

		// TODO: check prime comparison

		if sd59x18.Lt(aPrime, bPrime) {
			v, err := sd59x18.ConvertToSD59x18(bigint1e18)
			if err != nil {
				return nil, err
			}

			fee, err = sd59x18.Mul(
				sd59x18.Sub(bPrime, aPrime),
				v,
			)
			if err != nil {
				return nil, err
			}
		}

		wi, err := sd59x18.ConvertToSD59x18(w[i])
		if err != nil {
			return nil, err
		}

		log2BMinusFee, err := sd59x18.Log2(sd59x18.Sub(b, fee))
		if err != nil {
			return nil, err
		}

		prod, err := sd59x18.Mul(wi, sd59x18.Sub(log2BMinusFee, logA[i]))
		if err != nil {
			return nil, err
		}

		logGrowth = sd59x18.Add(logGrowth, prod)
	}

	logG, err := sd59x18.Div(sd59x18.Sub(sd59x18.Zero(), logGrowth), sumUnknownWeight)
	if err != nil {
		return nil, err
	}

	for i := range w {
		if r[i].Cmp(unknownBI) != 0 {
			continue
		}

		if i != 0 {
			v, err := sd59x18.ConvertToSD59x18(bigint1e5)
			if err != nil {
				return nil, err
			}
			logB := sd59x18.Add(logG, sd59x18.Add(logA[i], v))
			b, err := sd59x18.Exp2(logB)
			if err != nil {
				return nil, err
			}

			var (
				aPrime, bPrime sd59x18.SD59x18

				zeroSD59x18 = sd59x18.Zero()
			)

			// calculate aPrime
			{
				v := k
				if sd59x18.Lt(zeroSD59x18, logK) {
					v, err = sd59x18.ConvertToSD59x18(bigint1e18)
					if err != nil {
						return nil, err
					}
				}
				ai, err := sd59x18.ConvertToSD59x18(a[i])
				if err != nil {
					return nil, err
				}
				aPrime, err = sd59x18.Mul(ai, v)
				if err != nil {
					return nil, err
				}
			}

			// calculate bPrime
			{
				v, err := sd59x18.ConvertToSD59x18(bigint1e18)
				if err != nil {
					return nil, err
				}
				if sd59x18.Lt(zeroSD59x18, logK) {
					v = k
				}
				bPrime, err = sd59x18.Div(b, v)
				if err != nil {
					return nil, err
				}
			}

			if sd59x18.Lt(aPrime, bPrime) {
				// b = b + ((b_prime - a_prime) / (sd(1e18) - sd(int256(fee1e18)))) - (b_prime - a_prime);

				v0, err := sd59x18.ConvertToSD59x18(bigint1e18)
				if err != nil {
					return nil, err
				}

				v1, err := sd59x18.ConvertToSD59x18(fee1e18)
				if err != nil {
					return nil, err
				}

				v2, err := sd59x18.Div(
					sd59x18.Sub(bPrime, aPrime),
					sd59x18.Sub(v0, v1),
				)
				if err != nil {
					return nil, err
				}

				b = sd59x18.Add(b, sd59x18.Sub(v2, sd59x18.Sub(bPrime, aPrime)))
			}

			// r[i] = convert(b - convert(int256(a[i]))).toInt128();

			ai, err := sd59x18.ConvertToSD59x18(a[i])
			if err != nil {
				return nil, err
			}

			r[i] = sd59x18.ConvertToBI(sd59x18.Sub(b, ai))
		} else {

			// SD59x18 logB = logG + logInvariantMin;
			// r[i] = -convert(exp2(logB) - exp2(logInvariantMin)).toInt128();
			// if (logG < sd(0)) {
			// 	additionalMultiplier = uint256(exp2(-logG).intoInt256() / 1e9);
			// }

			logB := sd59x18.Add(logG, logInvariantMin)
			exp2LogB, err := sd59x18.Exp2(logB)
			if err != nil {
				return nil, err
			}

			exp2LogInvariantMin, err := sd59x18.Exp2(logInvariantMin)
			if err != nil {
				return nil, err
			}

			r[i] = new(big.Int).Neg(sd59x18.ConvertToBI(sd59x18.Sub(exp2LogB, exp2LogInvariantMin)))

			if sd59x18.Lt(logG, sd59x18.Zero()) {
				v, err := sd59x18.Exp2(sd59x18.Sub(sd59x18.Zero(), logG))
				if err != nil {
					return nil, err
				}

				additionalMultiplier = new(big.Int).Div(sd59x18.ConvertToBI(v), bigint1e9)
			}
		}
	}

	swapInfo := SwapInfo{}
	if additionalMultiplier.Cmp(bigint1e9) > 0 {
		feeMultiplier := additionalMultiplier
		if t.isLastWithdrawInTheSameBlock {
			feeMultiplier = new(big.Int).Div(
				new(big.Int).Mul(additionalMultiplier, t.feeMultiplier),
				bigint1e9,
			)
		}
		swapInfo.NeedToUpdateFeeMultiplier = true
		swapInfo.FeeMultiplierUpdated = feeMultiplier.String()
	}

	// for (uint256 i = 0; i < tokens.length; i++) {
	// 	r_.u(i, r.u(idx.u(i)));
	// }
	for i := range tokens {
		r_[i] = r[idx[i]]
	}

	var amountOut *big.Int
	for i, token := range tokens {
		if strings.EqualFold(token, tokenOut) {
			amountOut = new(big.Int).Neg(r_[i])
			break
		}
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut,
		},
		Fee:      nil, //TODO: implement me!
		Gas:      0,
		SwapInfo: &swapInfo,
	}, nil
}

func (t *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	
}

func (t *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return Meta{
		Fee1e9:        t.fee1e9,
		FeeMultiplier: t.feeMultiplier.String(),
	}
}

func (t *PoolSimulator) getEffectiveFee1e9() *big.Int {
	effectiveFee1e9 := big.NewInt(int64(t.fee1e9))
	if !t.isLastWithdrawInTheSameBlock {
		return effectiveFee1e9
	}
	effectiveFee1e9 = new(big.Int).Div(
		new(big.Int).Mul(effectiveFee1e9, t.feeMultiplier),
		bigint1e9,
	)
	return effectiveFee1e9
}

func (t *PoolSimulator) getPoolBalances(tokens []string) ([]*big.Int, error) {
	tokenToReserve := make(map[string]*big.Int)
	for i, token := range t.Info.Tokens {
		tokenToReserve[token] = t.Info.Reserves[i]
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

func (t *PoolSimulator) newVelocoreExecuteParams(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) ([]string, []*big.Int) {
	tokens := []string{tokenAmountIn.Token, tokenOut}
	amounts := []*big.Int{tokenAmountIn.Amount, unknownBI}
	return tokens, amounts
}

func (t *PoolSimulator) isLpToken(token string) bool {
	return strings.EqualFold(token, t.Info.Address)
}

func (t *PoolSimulator) getTokenWeight(token string) (*big.Int, error) {
	for i, tok := range t.Info.Tokens {
		if tok == token {
			return t.weights[i], nil
		}
	}
	return nil, ErrInvalidToken
}

func (t *PoolSimulator) getInvariant() (*big.Int, *big.Int, *big.Int, error) {
	balances := t.Info.Reserves
	if t.poolTokenNumber-lpTokenNumber == 2 && t.weights[1] == t.weights[2] {
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

	weights := t.weights
	logInvariant := sd59x18.Zero()
	for i := 1; i < len(weights); i++ {
		g, err := sd59x18.ConvertToSD59x18(new(big.Int).Add(balances[i], bigint1))
		if err != nil {
			return nil, nil, nil, err
		}
		g, err = sd59x18.Log2(g)
		if err != nil {
			return nil, nil, nil, err
		}

		w, err := sd59x18.ConvertToSD59x18(weights[i])
		if err != nil {
			return nil, nil, nil, err
		}

		gw, err := sd59x18.Mul(g, w)
		if err != nil {
			return nil, nil, nil, err
		}
		logInvariant = sd59x18.Add(logInvariant, gw)
	}
	sumW, err := sd59x18.ConvertToSD59x18(t.sumWeight)
	if err != nil {
		return nil, nil, nil, err
	}
	logInvariant, err = sd59x18.Div(logInvariant, sumW)
	if err != nil {
		return nil, nil, nil, err
	}

	invariant, err := sd59x18.Exp2(logInvariant)
	if err != nil {
		return nil, nil, nil, err
	}
	invariantMin := sd59x18.ConvertToBI(invariant)

	invariantMax := new(big.Int).Add(
		ceilDiv(
			new(big.Int).Mul(
				invariant,
				new(big.Int).Add(
					bigint1e18,
					bigint1e5,
				),
			),
			bigint1e18,
		),
		bigint1,
	)

	return zero, invariantMin, invariantMax, nil
}
