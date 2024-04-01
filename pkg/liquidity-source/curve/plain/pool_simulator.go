package plain

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/holiman/uint256"
)

func init() {
	gob.Register(&PoolSimulator{})
}

type PoolSimulator struct {
	pool.Pool

	PrecisionMultipliers []uint256.Int
	Reserves             []uint256.Int // same as pool.Reserves but use uint256.Int

	LpSupply uint256.Int
	Gas      Gas

	NumTokens     int
	NumTokensU256 uint256.Int

	Extra       Extra
	StaticExtra StaticExtra
}

type Gas struct {
	Exchange int64
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {

	/*
		Curve StableSwap Plain pools are the most basic Curve pools that implement StableSwap invariant
		(see the whitepaper for more details https://docs.curve.fi/references/whitepapers/stableswap/#constructing-the-stableswap-invariant )
		A*n^n * sum(x_i) + D = A*D*n^n + (D^{n+1}) / (n^n * prod(x_i))

		There are many variants of Curve StableSwap Plain pool:

		The first group are pools created and owned by Curve themselves
		the deployed pools will be put to https://github.com/curvefi/curve-contract/tree/master/contracts/pools
		(they are based on these templates but might come with some modifications https://github.com/curvefi/curve-contract/tree/master/contracts/pool-templates)

		The second group are pools created from various Factories (permissionless deployment)
			Plain2Basic.vy: standard
			Plain2Optimized.vy: same as Plain2Basic, but optimized for case when all coins have 18 decimals
			Plain2Balances.vy: support positive-rebasing & FOT tokens (call to coin to get pool's balance, instead of storing/calculating in pool itself)
				(correct balances should have been filled in by pool-tracker already)
			Plain2BasicEMA.vy: support EMA (exponential moving average) price (seems not affecting CalcAmountOut/UpdateBalance)
			Plain2ETH.vy: Uses (and optimized for) native Ether as coins[0]
			Plain2ETHEMA.vy: Plain2ETH with moving average price
			Plain2Price.vy: call to external contract to get rate multipliers (instead of hardcoding in contract)
				(should have been filled in by pool-tracker already)

			Plain3-xxx, Plain4-xxx: same as Plain2 but for 3/4 coins (Plain2 is more optimized)
	*/

	sim := &PoolSimulator{}

	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &sim.StaticExtra); err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(entityPool.Extra), &sim.Extra); err != nil {
		return nil, err
	}

	var numTokens = len(entityPool.Tokens)
	// Reserves: N tokens & lpSupply
	if entityPool.Reserves == nil || len(entityPool.Reserves) != numTokens+1 {
		return nil, ErrInvalidReserve
	}

	if numTokens > shared.MaxTokenCount {
		return nil, ErrInvalidNumToken
	}

	var tokens = make([]string, numTokens)
	var reservesBI = make([]*big.Int, numTokens)

	sim.Reserves = make([]uint256.Int, numTokens)
	sim.PrecisionMultipliers = make([]uint256.Int, numTokens)

	/*
		most of Plain pools use standard rate 10^(36 - token_decimal)
		- Factory pools: https://github.com/curvefi/curve-factory/blob/master/contracts/Factory.vy#L558
		- Curve-owned pools: see explanation of ___RATES___ in https://github.com/curvefi/curve-contract/blob/master/contracts/pool-templates/README.md
		some pools (ETH/rETH and ETH/aETH) have rates changed overtime, they have been filled in by pool-tracker already
	*/
	useStandardRate := false
	if len(sim.Extra.RateMultipliers) == 0 {
		sim.Extra.RateMultipliers = make([]uint256.Int, numTokens)
		useStandardRate = true
	}

	for i := 0; i < numTokens; i += 1 {
		tokens[i] = entityPool.Tokens[i].Address

		reservesBI[i] = bignumber.NewBig10(entityPool.Reserves[i])
		if err := sim.Reserves[i].SetFromDecimal(entityPool.Reserves[i]); err != nil {
			return nil, err
		}

		if useStandardRate {
			sim.Extra.RateMultipliers[i].Exp(
				uint256.NewInt(10),
				uint256.NewInt(uint64(36-entityPool.Tokens[i].Decimals)),
			)
		}

		/*
			different Plain variants have slightly different way to deal with this
			but they can all be expressed as 10^(18 - token_decimal)
			- Curve-owned pools: see explanation of ___PRECISION_MUL___ in https://github.com/curvefi/curve-contract/blob/master/contracts/pool-templates/README.md
			- Factory pools: see code, for example Plain3Basic _calc_withdraw_one_coin function:
					dy_0: uint256 = (xp[i] - new_y) * PRECISION / rates[i]  # w/o fees
					dy = (dy - 1) * PRECISION / rates[i]  # Withdraw less to account for rounding errors
				(something * 10^18 / 10^(36-decimal) --> something / 10^(18-decimal))
		*/
		sim.PrecisionMultipliers[i].Exp(
			uint256.NewInt(10),
			uint256.NewInt(uint64(18-entityPool.Tokens[i].Decimals)),
		)
	}

	sim.Pool = pool.Pool{
		Info: pool.PoolInfo{
			Address:    strings.ToLower(entityPool.Address),
			ReserveUsd: entityPool.ReserveUsd,
			SwapFee:    sim.Extra.SwapFee.ToBig(),
			Exchange:   entityPool.Exchange,
			Type:       entityPool.Type,
			Tokens:     tokens,
			Reserves:   reservesBI,
			Checked:    false,
		},
	}

	sim.Gas = DefaultGas

	if err := sim.LpSupply.SetFromDecimal(entityPool.Reserves[numTokens]); err != nil {
		return nil, err
	}

	sim.NumTokens = numTokens
	sim.NumTokensU256.SetUint64(uint64(numTokens))
	return sim, nil
}

func (t *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut
	// swap from token to token
	var tokenIndexFrom = t.Info.GetTokenIndex(tokenAmountIn.Token)
	var tokenIndexTo = t.Info.GetTokenIndex(tokenOut)
	if tokenIndexFrom >= 0 && tokenIndexTo >= 0 {
		var amountOut, fee, amount uint256.Int
		amount.SetFromBig(tokenAmountIn.Amount)
		err := t.GetDyU256(
			tokenIndexFrom,
			tokenIndexTo,
			&amount,
			nil,
			&amountOut, &fee,
		)
		if err != nil {
			return &pool.CalcAmountOutResult{}, err
		}
		if !amountOut.IsZero() {
			return &pool.CalcAmountOutResult{
				TokenAmountOut: &pool.TokenAmount{
					Token:  tokenOut,
					Amount: amountOut.ToBig(),
				},
				Fee: &pool.TokenAmount{
					Token:  tokenOut,
					Amount: fee.ToBig(),
				},
				Gas: t.Gas.Exchange,
			}, nil
		}
	}
	return &pool.CalcAmountOutResult{}, fmt.Errorf("tokenIndexFrom %v or TokenOutIndex %v is not correct", tokenIndexFrom, tokenIndexTo)
}

func (t *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output := params.TokenAmountIn, params.TokenAmountOut
	var inputAmount = input.Amount
	var outputAmount = output.Amount
	// swap fee
	// output = output + output * swapFee * adminFee
	outputAmount = new(big.Int).Add(
		outputAmount,
		new(big.Int).Div(
			new(big.Int).Mul(
				new(big.Int).Div(new(big.Int).Mul(outputAmount, t.Info.SwapFee), FeeDenominator.ToBig()),
				t.Extra.AdminFee.ToBig(),
			),
			FeeDenominator.ToBig(),
		),
	)
	for i := range t.Info.Tokens {
		if t.Info.Tokens[i] == input.Token {
			t.Info.Reserves[i] = new(big.Int).Add(t.Info.Reserves[i], inputAmount)
			t.Reserves[i].Add(&t.Reserves[i], number.SetFromBig(inputAmount))
		}
		if t.Info.Tokens[i] == output.Token {
			t.Info.Reserves[i] = new(big.Int).Sub(t.Info.Reserves[i], outputAmount)
			t.Reserves[i].Sub(&t.Reserves[i], number.SetFromBig(outputAmount))
		}
	}
}

func (t *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	var fromId = t.GetTokenIndex(tokenIn)
	var toId = t.GetTokenIndex(tokenOut)
	meta := curve.Meta{
		TokenInIndex:  fromId,
		TokenOutIndex: toId,
		Underlying:    false,
	}
	if len(t.StaticExtra.IsNativeCoin) == t.NumTokens {
		meta.TokenInIsNative = &t.StaticExtra.IsNativeCoin[fromId]
		meta.TokenOutIsNative = &t.StaticExtra.IsNativeCoin[toId]
	}
	return meta
}
