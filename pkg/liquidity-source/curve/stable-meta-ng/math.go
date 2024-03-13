package stablemetang

import (
	"fmt"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/shared"
	stableng "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/stable-ng"
	"github.com/holiman/uint256"
)

func (t *PoolSimulator) GetDyUnderlying(
	i int, j int, _dx *uint256.Int,

	// output
	dy *uint256.Int,
	adminFee *uint256.Int,
	withdrawFee *uint256.Int,
) error {
	var baseNCoins = len(t.basePool.GetInfo().Tokens)
	xp := stableng.XpMem(t.Extra.RateMultipliers, t.Reserves)

	/*
		meta pool has N_COINS tokens where the last one is LPtoken of base pool
		(in stable-meta-ng N_COINS=2 (1 meta coin and 1 LPtoken), in old meta pool there can be more meta coins)
		it can swap between meta pool's coins and base pool's coins like this:
			- if both input and output are base pool's coins:
				just like normal swap at base pool, but will cost more gas
				we'll reject this case at the outer level
			- if input coin is a base pool's coin (output is a meta pool's coin):
				deposit input coin to base pool (add liquidity), get back base pool's LPtoken
				use that to do a normal swap in meta pool, get back another meta pool's coin and return
				i = base_i + N_COINS-1
				meta_i = N_COINS-1
				j = [0, N_COINS-1] (base_j < 0)
				meta_j = j
			- if input coins is a meta pool's coin (output is a base pool's coin):
				swap at meta pool to get base pool's LPtoken
				use that to withdraw coin from base pool (remove liquidity), then return
				i = [0, N_COINS-1] (base_i < 0)
				meta_i = i
				j = base_j + N_COINS-1
				meta_j = N_COINS-1

		if input coin is from meta pool:
			i = [0, N_COINS-1] (last coin in meta pool is LPtoken of base pool, should be excluded)
			meta_i = i
		if input coin is from base pool:
			i = base_i + N_COINS-1
			meta_i = N_COINS-1 (this is the LPtoken, we'll add input coin to base pool to get LPtoken, then use that to do meta swap)

	*/

	var base_i = i - MAX_METAPOOL_COIN_INDEX
	var base_j = j - MAX_METAPOOL_COIN_INDEX
	var meta_i, meta_j int

	input_is_base_coin := base_i >= 0
	output_is_base_coin := base_j >= 0
	if input_is_base_coin && output_is_base_coin {
		// should be rejected at the outer level already
		return ErrAllBasePoolTokens
	}
	if !input_is_base_coin && !output_is_base_coin {
		// all meta coins, should not happen (should be redirected to GetDy instead)
		return ErrAllMetaPoolTokens
	}

	if output_is_base_coin {
		meta_i = i                       // input is meta coin
		meta_j = MAX_METAPOOL_COIN_INDEX // output of meta swap is LPtoken
	} else {
		meta_i = MAX_METAPOOL_COIN_INDEX
		meta_j = j
	}

	// determine input amount
	var x *uint256.Int
	if output_is_base_coin {
		// input is from meta pool, so just add dx directly into meta balances
		// x = xp[i] + dx * rates[0] / 10**18
		x = number.SafeAdd(&xp[i], number.SafeMul(_dx, number.Div(&t.Extra.RateMultipliers[i], Precision)))
	} else {
		// input is base coin, need to call base pool to get amount of LPtoken we'll get after depositing `_dx` input coin to base pool
		// then add that to meta balances of LPtoken
		var base_inputs [shared.MaxTokenCount]uint256.Int

		// x = self._base_calc_token_amount(
		//   dx, base_i, base_n_coins, BASE_POOL, True
		// ) * rates[1] / PRECISION
		for k := 0; k < baseNCoins; k += 1 {
			base_inputs[k].Clear()
		}
		base_inputs[base_i].Set(_dx)
		var mintAmount uint256.Int
		if err := t.basePool.CalculateTokenAmount(base_inputs[:baseNCoins], true, &mintAmount); err != nil {
			return err
		}
		fmt.Println("mintAmount", mintAmount.Dec())

		x = number.Div(number.SafeMul(&mintAmount, &t.Extra.RateMultipliers[MAX_METAPOOL_COIN_INDEX]), Precision)

		// Adding number of pool tokens
		// x += xp[1]
		number.SafeAddZ(x, &xp[MAX_METAPOOL_COIN_INDEX], x)
	}

	// perform normal swap at meta pool
	err := t.PoolSimulator.GetDyByX(meta_i, meta_j, x, xp, nil, dy, adminFee)
	if err != nil {
		return err
	}

	if output_is_base_coin {
		// withdraw output from base pool using `dy` of LPtoken
		var baseDy uint256.Int
		err = t.basePool.CalculateWithdrawOneCoin(dy, base_j, &baseDy, withdrawFee)
		if err != nil {
			return err
		}
		dy.Set(&baseDy)
	}
	return nil
}
