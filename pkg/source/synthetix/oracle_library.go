package synthetix

import (
	"math/big"

	"github.com/KyberNetwork/elastic-go-sdk/v2/utils"
	uniswapV3Utils "github.com/daoleno/uniswapv3-sdk/utils"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// =============================================================================================
// Implementation of this contract:
// https://github.com/Uniswap/v3-periphery/blob/5bcdd9f67f9394f3159dad80d0dd01d37ca08c66/contracts/libraries/OracleLibrary.sol

var (
	// MaxUInt128 = 2**128 - 1
	MaxUInt128 = new(big.Int).Sub(new(big.Int).Exp(big.NewInt(2), big.NewInt(128), nil), big.NewInt(1))
)

// / @notice Fetches time-weighted average tick using Uniswap V3 oracle
// / @param pool Address of Uniswap V3 pool that we want to observe
// / @param period Number of seconds in the past to start calculating time-weighted average
// / @return timeWeightedAverageTick The time-weighted average tick from (block.timestamp - period) to block.timestamp
func consult(state *DexPriceAggregatorUniswapV3, pool common.Address, period *big.Int) (int, error) {
	if period.Cmp(bignumber.ZeroBI) == 0 {
		return 0, ErrInvalidPeriod
	}

	tickCumulatives := state.TickCumulatives[pool.String()]
	tickCumulativesDelta := new(big.Int).Sub(tickCumulatives[1], tickCumulatives[0])

	timeWeightedAverageTick := new(big.Int).Div(tickCumulativesDelta, period)

	// Always round to negative infinity
	if tickCumulativesDelta.Cmp(bignumber.ZeroBI) < 0 && new(big.Int).Mod(tickCumulativesDelta, period).Cmp(bignumber.ZeroBI) != 0 {
		new(big.Int).Sub(timeWeightedAverageTick, big.NewInt(1))
	}

	return int(timeWeightedAverageTick.Int64()), nil
}

// / @notice Given a tick and a token amount, calculates the amount of token received in exchange
// / @param tick Tick value used to calculate the quote
// / @param baseAmount Amount of token to be converted
// / @param baseToken Address of an ERC20 token contract used as the baseAmount denomination
// / @param quoteToken Address of an ERC20 token contract used as the quoteAmount denomination
// / @return quoteAmount Amount of quoteToken received for baseAmount of baseToken
func getQuoteAtTick(
	tick int,
	baseAmount *big.Int,
	baseToken common.Address,
	quoteToken common.Address,
) (quoteAmount *big.Int, err error) {
	sqrtRatioX96, err := uniswapV3Utils.GetSqrtRatioAtTick(tick)
	if err != nil {
		return nil, err
	}

	// Calculate quoteAmount with better precision if it doesn't overflow when multiplied by itself
	if sqrtRatioX96.Cmp(MaxUInt128) <= 0 {
		ratioX192 := new(big.Int).Mul(sqrtRatioX96, sqrtRatioX96)

		if baseToken.String() < quoteToken.String() {
			quoteAmount = utils.MulDiv(ratioX192, baseAmount, new(big.Int).Lsh(big.NewInt(1), 192))
		} else {
			quoteAmount = utils.MulDiv(new(big.Int).Lsh(big.NewInt(1), 192), baseAmount, ratioX192)
		}
	} else {
		ratioX128 := utils.MulDiv(sqrtRatioX96, sqrtRatioX96, new(big.Int).Lsh(big.NewInt(1), 64))

		if baseToken.String() < quoteToken.String() {
			quoteAmount = utils.MulDiv(ratioX128, baseAmount, new(big.Int).Lsh(big.NewInt(1), 128))
		} else {
			quoteAmount = utils.MulDiv(new(big.Int).Lsh(big.NewInt(1), 128), baseAmount, ratioX128)
		}
	}

	return quoteAmount, nil
}

// / @notice Given a pool, it returns the tick value as of the start of the current block
// / @param pool Address of Uniswap V3 pool
// / @return The tick that the pool was in at the start of the current block
func getBlockStartingTick(state *DexPriceAggregatorUniswapV3, pool common.Address) (int, error) {
	slot0 := state.UniswapV3Slot0[pool.String()]
	tick := int(slot0.Tick.Int64())
	observationIndex := slot0.ObservationIndex
	observationCardinality := slot0.ObservationCardinality

	// 2 observations are needed to reliably calculate the block starting tick
	if observationCardinality <= 1 {
		return 0, ErrInvalidObservationCardinality
	}

	// If the latest observation occurred in the past, then no tick-changing trades have happened in this block
	// therefore the tick in `slot0` is the same as at the beginning of the current block.
	// We don't need to check if this observation is initialized - it is guaranteed to be.
	observation := state.UniswapV3Observations[pool.String()][observationIndex]
	observationTimestamp := observation.BlockTimestamp
	tickCumulative := observation.TickCumulative

	if uint64(observationTimestamp) != state.BlockTimestamp {
		return tick, nil
	}

	prevIndex := (observationIndex + observationCardinality - 1) % observationCardinality

	prevObservation := state.UniswapV3Observations[pool.String()][prevIndex]
	prevObservationTimestamp := prevObservation.BlockTimestamp
	prevTickCumulative := prevObservation.TickCumulative
	prevInitialized := prevObservation.Initialized

	if !prevInitialized {
		return 0, ErrInvalidPrevInitialized
	}

	return int(new(big.Int).Div(
		new(big.Int).Sub(tickCumulative, prevTickCumulative),
		big.NewInt(int64(observationTimestamp-prevObservationTimestamp)),
	).Int64()), nil
}
