package velocorev2cpmm

import (
	"encoding/json"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/velocorev2-cpmm/sd59x18"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/logger"
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

	sumWeight = bigint0
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
	return Meta{
		Fee1e9:        p.fee1e9,
		FeeMultiplier: p.feeMultiplier.String(),
	}
}

// https://github.com/velocore/velocore-contracts/blob/master/src/pools/constant-product/ConstantProductPool.sol#L164
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
			weights[i] = p.sumWeight
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
		lpUnknown                  = lpInvolved && (a[iLp].Cmp(unknownBI) == 0)
	)

	// TODO: move this to other functions

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
				fee    *big.Int
			)
			if k.Cmp(bigint1e18) > 0 {
				aPrime = a[i]
				bPrime = new(big.Int).Div(new(big.Int).Mul(b, bigint1e18), k)
			} else {
				aPrime = new(big.Int).Div(new(big.Int).Mul(a[i], k), bigint1e18)
				bPrime = b
			}

			if bPrime.Cmp(aPrime) > 0 {
				fee = ceilDivUnsafe(
					new(big.Int).Mul(new(big.Int).Sub(bPrime, aPrime), effectiveFee1e9),
					bigint1e9,
				)
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

		requestedGrowth1e18 = new(big.Int).Div(
			new(big.Int).Mul(requestedGrowth1e18, rpow(tokenGrowth1e18, weights[i], bigint1e18)),
			bigint1e18,
		)

		// this statement is not actually needed because we use *big.Int instead of uint256.
		// it's here to make the code more similar to the original.
		if requestedGrowth1e18.Cmp(bigint0) <= 0 {
			return nil, ErrInvalidTokenGrowth
		}
	}

	unaccountedFeeAsGrowth1e18 := bigint1e18
	if k.Cmp(bigint1e18) < 0 {
		x := new(big.Int).Sub(
			bigint1e18,
			new(big.Int).Div(
				new(big.Int).Mul(new(big.Int).Sub(bigint1e18, k), effectiveFee1e9),
				bigint1e9,
			),
		)
		n := new(big.Int).Sub(new(big.Int).Sub(p.sumWeight, sumUnknownWeight), sumKnownWeight)
		unaccountedFeeAsGrowth1e18 = rpow(x, n, bigint1e18)
		requestedGrowth1e18 = new(big.Int).Div(
			new(big.Int).Mul(requestedGrowth1e18, unaccountedFeeAsGrowth1e18),
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
			b := ceilDiv(new(big.Int).Mul(g, a[i]), bigint1e18)
			var (
				fee            = bigint0
				aPrime, bPrime *big.Int
			)
			if k.Cmp(bigint1e18) > 0 {
				aPrime = a[i]
				bPrime = ceilDiv(new(big.Int).Mul(b, bigint1e18), k)
			} else {
				aPrime = new(big.Int).Div(new(big.Int).Mul(a[i], k), bigint1e18)
				bPrime = b
			}

			bPrimeMinusAPrime := new(big.Int).Sub(bPrime, aPrime)
			if bPrime.Cmp(aPrime) > 0 {
				fee = new(big.Int).Sub(
					ceilDiv(
						new(big.Int).Mul(bPrimeMinusAPrime, bigint1e9),
						new(big.Int).Sub(bigint1e9, effectiveFee1e9),
					),
					bPrimeMinusAPrime,
				)
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

// https://github.com/velocore/velocore-contracts/blob/master/src/pools/constant-product/ConstantProductLibrary.sol#L25
func (p *PoolSimulator) velocoreExecuteFallback(tokens []string, r_ []*big.Int) (*velocoreExecuteResult, error) {
	var (
		t   = p.Info.Tokens
		a   = p.Info.Reserves
		idx = make([]int, len(tokens))
		w   = p.weights
	)

	fee1e18 := new(big.Int).Mul(big.NewInt(int64(p.fee1e9)), bigint1e9)
	if p.isLastWithdrawInTheSameBlock {
		fee1e18 = new(big.Int).Div(new(big.Int).Mul(fee1e18, p.feeMultiplier), bigint1e9)
	}
	additionalMultiplier := bigint1e9

	// map from token to index in "t"
	poolTokenIdx := map[string]int{}
	for i, token := range t {
		poolTokenIdx[token] = i
	}

	// copy "r_" to "r"
	// and map index in "tokens" to index in "t"
	r := make([]*big.Int, len(t))
	for i := range t {
		r[i] = bigint0
	}
	for i, token := range tokens {
		j := poolTokenIdx[token]
		idx[i] = j
		r[j] = r_[i]
	}

	for i := 1; i < len(w); i++ {
		a[i] = new(big.Int).Add(a[i], bigint1)
	}

	var (
		rSD59x18 = make([]sd59x18.SD59x18, len(r))
		aSD59x18 = make([]sd59x18.SD59x18, len(a))
		wSD59x18 = make([]sd59x18.SD59x18, len(w))
	)
	for i := 0; i < int(p.poolTokenNumber); i++ {
		var err error
		if r[i].Cmp(unknownBI) != 0 {
			if rSD59x18[i], err = sd59x18.ConvertToSD59x18(r[i]); err != nil {
				return nil, err
			}
		}
		if aSD59x18[i], err = sd59x18.ConvertToSD59x18(a[i]); err != nil {
			return nil, err
		}
		if wSD59x18[i], err = sd59x18.ConvertToSD59x18(w[i]); err != nil {
			return nil, err
		}
	}

	logA := make([]sd59x18.SD59x18, len(w))
	logInvariantMin := sd59x18.Zero()
	for i := 1; i < len(w); i++ {
		var err error
		if logA[i], err = sd59x18.Log2(aSD59x18[i]); err != nil {
			return nil, err
		}
		logAiMulWi, err := sd59x18.Mul(logA[i], wSD59x18[i])
		if err != nil {
			return nil, err
		}
		logInvariantMin = sd59x18.Add(logInvariantMin, logAiMulWi)
	}

	var err error
	if logInvariantMin, err = sd59x18.Div(logInvariantMin, wSD59x18[0]); err != nil {
		return nil, err
	}

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

			aiAddRiSD59x18, err := sd59x18.ConvertToSD59x18(new(big.Int).Add(a[i], r[i]))
			if err != nil {
				return nil, err
			}
			log2AiAddRiSD59x18, err := sd59x18.Log2(aiAddRiSD59x18)
			if err != nil {
				return nil, err
			}

			v, err := sd59x18.Mul(wSD59x18[i], sd59x18.Sub(log2AiAddRiSD59x18, logA[i]))
			if err != nil {
				return nil, err
			}
			logK = sd59x18.Add(logK, v)
		}
		w0SubKwSD59x18, err := sd59x18.ConvertToSD59x18(new(big.Int).Sub(w[0], kw))
		if err != nil {
			return nil, err
		}
		if logK, err = sd59x18.Div(logK, w0SubKwSD59x18); err != nil {
			return nil, err
		}
		sumUnknownWeight = sd59x18.Sub(sumUnknownWeight, wSD59x18[0])
	} else if r[0].Cmp(bigint0) != 0 {
		exp2LogInvariantMin, err := sd59x18.Exp2(logInvariantMin)
		if err != nil {
			return nil, err
		}
		x := sd59x18.Sub(exp2LogInvariantMin, rSD59x18[0])
		one, err := sd59x18.ConvertToSD59x18(bigint1)
		if err != nil {
			return nil, err
		}

		v := x
		if sd59x18.Lt(x, one) {
			v = one
		}
		logK, err = sd59x18.Log2(v)
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
			wSD59x18[0],
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
			sumUnknownWeight = sd59x18.Add(sumUnknownWeight, wSD59x18[i])
			continue
		}

		b, err := sd59x18.ConvertToSD59x18(new(big.Int).Add(a[i], r[i]))
		if err != nil {
			return nil, err
		}

		var (
			fee            = sd59x18.Zero()
			aPrime, bPrime sd59x18.SD59x18

			sd0    sd59x18.SD59x18 = bigint0
			sd1e18 sd59x18.SD59x18 = bigint1e18
		)
		// calculate aPrime
		{
			v := k
			if sd59x18.Lt(sd0, logK) {
				v = sd1e18
			}
			aPrime, err = sd59x18.Mul(aSD59x18[i], v)
			if err != nil {
				return nil, err
			}
		}

		// calculate bPrime
		{
			v := sd1e18
			if sd59x18.Lt(sd0, logK) {
				v = k
			}
			bPrime, err = sd59x18.Div(b, v)
			if err != nil {
				return nil, err
			}
		}

		if sd59x18.Lt(aPrime, bPrime) {
			var sdFee1e18 sd59x18.SD59x18 = fee1e18
			fee, err = sd59x18.Mul(
				sd59x18.Sub(bPrime, aPrime),
				sdFee1e18,
			)
			if err != nil {
				return nil, err
			}
		}

		// calculate logGrowth
		log2BMinusFee, err := sd59x18.Log2(sd59x18.Sub(b, fee))
		if err != nil {
			return nil, err
		}
		prod, err := sd59x18.Mul(wSD59x18[i], sd59x18.Sub(log2BMinusFee, logA[i]))
		if err != nil {
			return nil, err
		}
		logGrowth = sd59x18.Add(logGrowth, prod)
	}

	logG, err := sd59x18.Div(sd59x18.Sub(sd59x18.Zero(), logGrowth), sumUnknownWeight)
	if err != nil {
		return nil, err
	}

	// calculate unknown "r_i"
	for i := range w {
		if r[i].Cmp(unknownBI) != 0 {
			continue
		}

		var (
			sd0    sd59x18.SD59x18 = bigint0
			sd1e18 sd59x18.SD59x18 = bigint1e18
		)

		if i != 0 {
			var sd1e5 sd59x18.SD59x18 = bigint1e5
			logB := sd59x18.Add(logG, sd59x18.Add(logA[i], sd1e5))
			b, err := sd59x18.Exp2(logB)
			if err != nil {
				return nil, err
			}

			var (
				aPrime, bPrime sd59x18.SD59x18
			)

			// calculate aPrime
			{
				v := k
				if sd59x18.Lt(sd0, logK) {
					v = sd1e18
				}
				aPrime, err = sd59x18.Mul(aSD59x18[i], v)
				if err != nil {
					return nil, err
				}
			}

			// calculate bPrime
			{
				v := sd1e18
				if sd59x18.Lt(sd0, logK) {
					v = k
				}
				bPrime, err = sd59x18.Div(b, v)
				if err != nil {
					return nil, err
				}
			}

			if sd59x18.Lt(aPrime, bPrime) {
				var sdFee1e18 sd59x18.SD59x18 = fee1e18
				u, err := sd59x18.Div(
					sd59x18.Sub(bPrime, aPrime),
					sd59x18.Sub(sd1e18, sdFee1e18),
				)
				if err != nil {
					return nil, err
				}
				v := sd59x18.Sub(bPrime, aPrime)
				b = sd59x18.Add(b, sd59x18.Sub(u, v))
			}

			r[i] = sd59x18.ConvertToBI(sd59x18.Sub(b, aSD59x18[i]))

			continue
		}

		// case unknown lp token "r"

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

		if sd59x18.Lt(logG, sd0) {
			v, err := sd59x18.Exp2(sd59x18.Sub(sd59x18.Zero(), logG))
			if err != nil {
				return nil, err
			}
			additionalMultiplier = new(big.Int).Div(sd59x18.ConvertToBI(v), bigint1e9)
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
	return strings.EqualFold(token, p.Info.Address)
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
	if p.poolTokenNumber-lpTokenNumber == 2 && p.weights[1] == p.weights[2] {
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

	weights := p.weights
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
	sumW, err := sd59x18.ConvertToSD59x18(p.sumWeight)
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
			new(big.Int).Mul(invariant, new(big.Int).Add(bigint1e18, bigint1e5)),
			bigint1e18,
		),
		bigint1,
	)

	return bigint0, invariantMin, invariantMax, nil
}
