package synthetix

import (
	"encoding/hex"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
)

// =============================================================================================
// Implementation of this contract:
// https://github.com/sohkai/uniswap-v3-spot-twap-oracle/blob/8f9777a6160a089c99f39f2ee297119ee293bc4b/contracts/DexPriceAggregatorUniswapV3.sol

/********************
 * Oracle functions *
 ********************/

func (dp *DexPriceAggregatorUniswapV3) assetToAsset(
	_tokenIn common.Address,
	_amountIn *big.Int,
	_tokenOut common.Address,
	_twapPeriod *big.Int,
) (*big.Int, error) {
	if _tokenIn == dp.Weth {
		return dp.ethToAsset(_amountIn, _tokenOut, _twapPeriod)
	} else if _tokenOut == dp.Weth {
		return dp.assetToEth(_tokenIn, _amountIn, _twapPeriod)
	} else {
		return dp._fetchAmountCrossingPools(_tokenIn, _amountIn, _tokenOut, _twapPeriod)
	}
}

// @notice Given a token and its amount, return the equivalent value in ETH
//
//	@param _tokenIn Address of an ERC20 token contract to be converted
//	@param _amountIn Amount of tokenIn to be converted
//	@param _twapPeriod Number of seconds in the past to consider for the TWAP rate
//	@return ethAmountOut Amount of ETH received for amountIn of tokenIn
func (dp *DexPriceAggregatorUniswapV3) assetToEth(
	_tokenIn common.Address,
	_amountIn *big.Int,
	_twapPeriod *big.Int,
) (*big.Int, error) {
	tokenOut := dp.Weth
	pool, err := dp._getPoolForRoute(getPoolKey(_tokenIn, tokenOut, dp.DefaultPoolFee))
	if err != nil {
		return nil, err
	}

	return dp._fetchAmountFromSinglePool(_tokenIn, _amountIn, tokenOut, pool, _twapPeriod)
}

// @notice Given an amount of ETH, return the equivalent value in another token
//
//	@param _ethAmountIn Amount of ETH to be converted
//	@param _tokenOut Address of an ERC20 token contract to convert into
//	@param _twapPeriod Number of seconds in the past to consider for the TWAP rate
//	@return amountOut Amount of tokenOut received for ethAmountIn of ETH
func (dp *DexPriceAggregatorUniswapV3) ethToAsset(
	_ethAmountIn *big.Int,
	_tokenOut common.Address,
	_twapPeriod *big.Int,
) (*big.Int, error) {
	tokenIn := dp.Weth
	pool, err := dp._getPoolForRoute(getPoolKey(tokenIn, _tokenOut, dp.DefaultPoolFee))
	if err != nil {
		return nil, err
	}

	return dp._fetchAmountFromSinglePool(tokenIn, _ethAmountIn, _tokenOut, pool, _twapPeriod)
}

// @notice Given a token and amount, return the equivalent value in another token by exchanging
//
//	within a single liquidity pool
//
// @dev _pool _must_ be previously checked to contain _tokenIn and _tokenOut.
//
//	It is exposed as a parameter only as a gas optimization.
//
// @param _tokenIn Address of an ERC20 token contract to be converted
// @param _amountIn Amount of tokenIn to be converted
// @param _tokenOut Address of an ERC20 token contract to convert into
// @param _pool Address of a Uniswap V3 pool containing _tokenIn and _tokenOut
// @param _twapPeriod Number of seconds in the past to consider for the TWAP rate
// @return amountOut Amount of _tokenOut received for _amountIn of _tokenIn
func (dp *DexPriceAggregatorUniswapV3) _fetchAmountFromSinglePool(
	_tokenIn common.Address,
	_amountIn *big.Int,
	_tokenOut common.Address,
	_pool common.Address,
	_twapPeriod *big.Int,
) (*big.Int, error) {
	spotTick, err := getBlockStartingTick(dp, _pool)
	if err != nil {
		return nil, err
	}

	twapTick, err := consult(dp, _pool, _twapPeriod)
	if err != nil {
		return nil, err
	}

	// Return min amount between spot price and twap
	// Ticks are based on the ratio between token0:token1 so if the input token is token1 then
	// we need to treat the tick as an inverse
	var minTick int
	if _tokenIn.String() < _tokenOut.String() {
		if spotTick < twapTick {
			minTick = spotTick
		} else {
			minTick = twapTick
		}
	} else {
		if spotTick > twapTick {
			minTick = spotTick
		} else {
			minTick = twapTick
		}
	}

	return getQuoteAtTick(
		minTick, // can assume safe being result from consult()
		_amountIn,
		_tokenIn,
		_tokenOut,
	)
}

// @notice Given a token and amount, return the equivalent value in another token by "crossing"
// liquidity across an intermediary pool with ETH (ie. _tokenIn:ETH and ETH:_tokenOut)
//
// @dev If an overridden pool has been set for _tokenIn and _tokenOut, this pool will be used
// directly in lieu of "crossing" against an intermediary pool with ETH
//
// @param _tokenIn Address of an ERC20 token contract to be converted
// @param _amountIn Amount of tokenIn to be converted
// @param _tokenOut Address of an ERC20 token contract to convert into
// @param _twapPeriod Number of seconds in the past to consider for the TWAP rate
// @return amountOut Amount of _tokenOut received for _amountIn of _tokenIn
func (dp *DexPriceAggregatorUniswapV3) _fetchAmountCrossingPools(
	_tokenIn common.Address,
	_amountIn *big.Int,
	_tokenOut common.Address,
	_twapPeriod *big.Int,
) (*big.Int, error) {
	// If the tokenIn:tokenOut route was overridden to use a single pool, derive price directly from that pool
	overriddenPool := dp._getOverriddenPool(
		getPoolKey(_tokenIn, _tokenOut, bignumber.ZeroBI), // pool fee is unused
	)

	if !eth.IsZeroAddress(overriddenPool) {
		return dp._fetchAmountFromSinglePool(_tokenIn, _amountIn, _tokenOut, overriddenPool, _twapPeriod)
	}

	// Otherwise, derive the price by "crossing" through tokenIn:ETH -> ETH:tokenOut
	// To keep consistency, we cross through with the same price source (spot vs. twap)
	pool1, err := dp._getPoolForRoute(getPoolKey(_tokenIn, dp.Weth, dp.DefaultPoolFee))
	if err != nil {
		return nil, err
	}

	pool2, err := dp._getPoolForRoute(getPoolKey(_tokenOut, dp.Weth, dp.DefaultPoolFee))
	if err != nil {
		return nil, err
	}

	spotTick1, err := getBlockStartingTick(dp, pool1)
	if err != nil {
		return nil, err
	}

	spotTick2, err := getBlockStartingTick(dp, pool2)
	if err != nil {
		return nil, err
	}

	spotAmountOut, err := dp._getQuoteCrossingTicksThroughWeth(_tokenIn, _amountIn, _tokenOut, spotTick1, spotTick2)
	if err != nil {
		return nil, err
	}

	castedTwapPeriod := _twapPeriod
	twapTick1, err := consult(dp, pool1, castedTwapPeriod)
	if err != nil {
		return nil, err
	}

	twapTick2, err := consult(dp, pool2, castedTwapPeriod)
	if err != nil {
		return nil, err
	}

	twapAmountOut, err := dp._getQuoteCrossingTicksThroughWeth(_tokenIn, _amountIn, _tokenOut, twapTick1, twapTick2)
	if err != nil {
		return nil, err
	}

	// Return min amount between spot price and twap
	if spotAmountOut.Cmp(twapAmountOut) < 0 {
		return spotAmountOut, nil
	}

	return twapAmountOut, nil
}

// @notice Similar to OracleLibrary#getQuoteAtTick but calculates the amount of token received
// in exchange by first adjusting into ETH
// (i.e. when a route goes through an intermediary pool with ETH)
// @param _tokenIn Address of an ERC20 token contract to be converted
// @param _amountIn Amount of tokenIn to be converted
// @param _tokenOut Address of an ERC20 token contract to convert into
// @param _tick1 First tick value used to adjust from _tokenIn to ETH
// @param _tick2 Second tick value used to adjust from ETH to _tokenOut
// @return amountOut Amount of _tokenOut received for _amountIn of _tokenIn
func (dp *DexPriceAggregatorUniswapV3) _getQuoteCrossingTicksThroughWeth(
	_tokenIn common.Address,
	_amountIn *big.Int,
	_tokenOut common.Address,
	_tick1 int,
	_tick2 int,
) (*big.Int, error) {
	ethAmountOut, err := getQuoteAtTick(_tick1, _amountIn, _tokenIn, dp.Weth)
	if err != nil {
		return nil, err
	}

	return getQuoteAtTick(_tick2, ethAmountOut, dp.Weth, _tokenOut)
}

// @notice Fetch the Uniswap V3 pool to be queried for a route denoted by a PoolKey
// @param _poolKey PoolKey representing the route
// @return pool Address of the Uniswap V3 pool to use for the route
func (dp *DexPriceAggregatorUniswapV3) _getPoolForRoute(_poolKey PoolKey) (common.Address, error) {
	pool := dp._getOverriddenPool(_poolKey)
	if !eth.IsZeroAddress(pool) {
		return pool, nil
	}

	pool, err := computeAddress(dp.UniswapV3Factory, _poolKey)
	if err != nil {
		return common.Address{}, err
	}

	return pool, nil
}

// @notice Obtain the canonical identifier for a route denoted by a PoolKey
// @param _poolKey PoolKey representing the route
// @return id identifier for the route
func _identifyRouteFromPoolKey(_poolKey PoolKey) string {
	return hex.EncodeToString(crypto.Keccak256(abi.EncodePacked(_poolKey.token0.Bytes(), _poolKey.token1.Bytes())))
}

// @notice Fetch an overridden pool for a route denoted by a PoolKey, if any
// @param _poolKey PoolKey representing the route
// @return pool Address of the Uniswap V3 pool overridden for the route.
// address(0) if no overridden pool has been set.
func (dp *DexPriceAggregatorUniswapV3) _getOverriddenPool(_poolKey PoolKey) common.Address {
	return dp.OverriddenPoolForRoute[_identifyRouteFromPoolKey(_poolKey)]
}
