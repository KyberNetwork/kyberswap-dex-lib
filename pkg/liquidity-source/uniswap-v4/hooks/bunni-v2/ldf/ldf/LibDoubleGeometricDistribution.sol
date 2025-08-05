// SPDX-License-Identifier: BUSL-1.1
pragma solidity ^0.8.19;

import {SafeCastLib} from "solady/utils/SafeCastLib.sol";
import {FixedPointMathLib} from "solady/utils/FixedPointMathLib.sol";

import "./ShiftMode.sol";
import "../lib/Math.sol";
import "../lib/ExpMath.sol";
import "../base/Constants.sol";
import "./LibGeometricDistribution.sol";
import {LDFType} from "../types/LDFType.sol";

library LibDoubleGeometricDistribution {
    using SafeCastLib for uint256;
    using FixedPointMathLib for uint256;

    uint256 internal constant ALPHA_BASE = 1e8; // alpha uses 8 decimals in ldfParams
    uint256 internal constant MIN_LIQUIDITY_DENSITY = Q96 / 1e3;

    /// @dev Queries the liquidity density and the cumulative amounts at the given rounded tick.
    /// @param roundedTick The rounded tick to query
    /// @param tickSpacing The spacing of the ticks
    /// @return liquidityDensityX96_ The liquidity density at the given rounded tick. Range is [0, 1]. Scaled by 2^96.
    /// @return cumulativeAmount0DensityX96 The cumulative amount of token0 in the rounded ticks [roundedTick + tickSpacing, minTick + length * tickSpacing)
    /// @return cumulativeAmount1DensityX96 The cumulative amount of token1 in the rounded ticks [minTick, roundedTick - tickSpacing]
    function query(
        int24 roundedTick,
        int24 tickSpacing,
        int24 minTick,
        int24 length0,
        int24 length1,
        uint256 alpha0X96,
        uint256 alpha1X96,
        uint256 weight0,
        uint256 weight1
    )
        internal
        pure
        returns (uint256 liquidityDensityX96_, uint256 cumulativeAmount0DensityX96, uint256 cumulativeAmount1DensityX96)
    {
        // compute liquidityDensityX96
        liquidityDensityX96_ = liquidityDensityX96(
            roundedTick, tickSpacing, minTick, length0, length1, alpha0X96, alpha1X96, weight0, weight1
        );

        // compute cumulativeAmount0DensityX96
        cumulativeAmount0DensityX96 = cumulativeAmount0(
            roundedTick + tickSpacing,
            Q96,
            tickSpacing,
            minTick,
            length0,
            length1,
            alpha0X96,
            alpha1X96,
            weight0,
            weight1
        );

        // compute cumulativeAmount1DensityX96
        cumulativeAmount1DensityX96 = cumulativeAmount1(
            roundedTick - tickSpacing,
            Q96,
            tickSpacing,
            minTick,
            length0,
            length1,
            alpha0X96,
            alpha1X96,
            weight0,
            weight1
        );
    }

    /// @dev Computes the cumulative amount of token0 in the rounded ticks [roundedTick, tickUpper).
    function cumulativeAmount0(
        int24 roundedTick,
        uint256 totalLiquidity,
        int24 tickSpacing,
        int24 minTick,
        int24 length0,
        int24 length1,
        uint256 alpha0X96,
        uint256 alpha1X96,
        uint256 weight0,
        uint256 weight1
    ) internal pure returns (uint256 amount0) {
        uint256 totalLiquidity0 = totalLiquidity.mulDiv(weight0, weight0 + weight1);
        uint256 totalLiquidity1 = totalLiquidity.mulDiv(weight1, weight0 + weight1);
        amount0 = LibGeometricDistribution.cumulativeAmount0(
            roundedTick, totalLiquidity0, tickSpacing, minTick + length1 * tickSpacing, length0, alpha0X96
        )
            + LibGeometricDistribution.cumulativeAmount0(
                roundedTick, totalLiquidity1, tickSpacing, minTick, length1, alpha1X96
            );
    }

    /// @dev Computes the cumulative amount of token1 in the rounded ticks [tickLower, roundedTick].
    function cumulativeAmount1(
        int24 roundedTick,
        uint256 totalLiquidity,
        int24 tickSpacing,
        int24 minTick,
        int24 length0,
        int24 length1,
        uint256 alpha0X96,
        uint256 alpha1X96,
        uint256 weight0,
        uint256 weight1
    ) internal pure returns (uint256 amount1) {
        uint256 totalLiquidity0 = totalLiquidity.mulDiv(weight0, weight0 + weight1);
        uint256 totalLiquidity1 = totalLiquidity.mulDiv(weight1, weight0 + weight1);
        amount1 = LibGeometricDistribution.cumulativeAmount1(
            roundedTick, totalLiquidity0, tickSpacing, minTick + length1 * tickSpacing, length0, alpha0X96
        )
            + LibGeometricDistribution.cumulativeAmount1(
                roundedTick, totalLiquidity1, tickSpacing, minTick, length1, alpha1X96
            );
    }

    /// @dev Given a cumulativeAmount0, computes the rounded tick whose cumulativeAmount0 is closest to the input. Range is [tickLower, tickUpper].
    ///      The returned tick will be the largest rounded tick whose cumulativeAmount0 is greater than or equal to the input.
    ///      In the case that the input exceeds the cumulativeAmount0 of all rounded ticks, the function will return (false, 0).
    function inverseCumulativeAmount0(
        uint256 cumulativeAmount0_,
        uint256 totalLiquidity,
        int24 tickSpacing,
        int24 minTick,
        int24 length0,
        int24 length1,
        uint256 alpha0X96,
        uint256 alpha1X96,
        uint256 weight0,
        uint256 weight1
    ) internal pure returns (bool success, int24 roundedTick) {
        // try ldf0 first, if fails then try ldf1 with remainder
        int24 minTick0 = minTick + length1 * tickSpacing;
        uint256 totalLiquidity0 = totalLiquidity.mulDiv(weight0, weight0 + weight1);
        uint256 ldf0CumulativeAmount0 = LibGeometricDistribution.cumulativeAmount0(
            minTick0, totalLiquidity0, tickSpacing, minTick0, length0, alpha0X96
        );

        if (cumulativeAmount0_ <= ldf0CumulativeAmount0) {
            return LibGeometricDistribution.inverseCumulativeAmount0(
                cumulativeAmount0_, totalLiquidity0, tickSpacing, minTick0, length0, alpha0X96
            );
        } else {
            uint256 remainder = cumulativeAmount0_ - ldf0CumulativeAmount0;
            uint256 totalLiquidity1 = totalLiquidity.mulDiv(weight1, weight0 + weight1);
            return LibGeometricDistribution.inverseCumulativeAmount0(
                remainder, totalLiquidity1, tickSpacing, minTick, length1, alpha1X96
            );
        }
    }

    /// @dev Given a cumulativeAmount1, computes the rounded tick whose cumulativeAmount1 is closest to the input. Range is [tickLower - tickSpacing, tickUpper - tickSpacing].
    ///      The returned tick will be the smallest rounded tick whose cumulativeAmount1 is greater than or equal to the input.
    ///      In the case that the input exceeds the cumulativeAmount1 of all rounded ticks, the function will return (false, 0).
    function inverseCumulativeAmount1(
        uint256 cumulativeAmount1_,
        uint256 totalLiquidity,
        int24 tickSpacing,
        int24 minTick,
        int24 length0,
        int24 length1,
        uint256 alpha0X96,
        uint256 alpha1X96,
        uint256 weight0,
        uint256 weight1
    ) internal pure returns (bool success, int24 roundedTick) {
        // try ldf1 first, if fails then try ldf0 with remainder
        uint256 totalLiquidity1 = totalLiquidity.mulDiv(weight1, weight0 + weight1);
        uint256 ldf1CumulativeAmount1 = LibGeometricDistribution.cumulativeAmount1(
            minTick + length1 * tickSpacing, totalLiquidity1, tickSpacing, minTick, length1, alpha1X96
        );

        if (cumulativeAmount1_ <= ldf1CumulativeAmount1) {
            return LibGeometricDistribution.inverseCumulativeAmount1(
                cumulativeAmount1_, totalLiquidity1, tickSpacing, minTick, length1, alpha1X96
            );
        } else {
            uint256 remainder = cumulativeAmount1_ - ldf1CumulativeAmount1;
            uint256 totalLiquidity0 = totalLiquidity.mulDiv(weight0, weight0 + weight1);
            return LibGeometricDistribution.inverseCumulativeAmount1(
                remainder, totalLiquidity0, tickSpacing, minTick + length1 * tickSpacing, length0, alpha0X96
            );
        }
    }

    function checkMinLiquidityDensity(
        uint256 totalLiquidity,
        int24 tickSpacing,
        int24 length0,
        uint256 alpha0,
        uint256 weight0,
        int24 length1,
        uint256 alpha1,
        uint256 weight1
    ) internal pure returns (bool) {
        // ensure liquidity density is nowhere equal to zero
        // can check boundaries since function is monotonic
        int24 minTick = 0; // no loss of generality since shifting doesn't change the min liquidity density
        {
            uint256 alpha0X96 = uint256(alpha0).mulDiv(Q96, ALPHA_BASE);
            uint256 minLiquidityDensityX96;
            int24 minTick0 = minTick + length1 * tickSpacing;
            if (alpha0 > ALPHA_BASE) {
                // monotonically increasing
                // check left boundary
                minLiquidityDensityX96 =
                    LibGeometricDistribution.liquidityDensityX96(minTick0, tickSpacing, minTick0, length0, alpha0X96);
            } else {
                // monotonically decreasing
                // check right boundary
                minLiquidityDensityX96 = LibGeometricDistribution.liquidityDensityX96(
                    minTick0 + (length0 - 1) * tickSpacing, tickSpacing, minTick0, length0, alpha0X96
                );
            }
            minLiquidityDensityX96 =
                minLiquidityDensityX96.mulDiv(weight0, weight0 + weight1).mulDiv(totalLiquidity, Q96);
            if (minLiquidityDensityX96 < MIN_LIQUIDITY_DENSITY) {
                return false;
            }
        }

        {
            uint256 alpha1X96 = uint256(alpha1).mulDiv(Q96, ALPHA_BASE);
            uint256 minLiquidityDensityX96;
            if (alpha1 > ALPHA_BASE) {
                // monotonically increasing
                // check left boundary
                minLiquidityDensityX96 =
                    LibGeometricDistribution.liquidityDensityX96(minTick, tickSpacing, minTick, length1, alpha1X96);
            } else {
                // monotonically decreasing
                // check right boundary
                minLiquidityDensityX96 = LibGeometricDistribution.liquidityDensityX96(
                    minTick + (length1 - 1) * tickSpacing, tickSpacing, minTick, length1, alpha1X96
                );
            }
            minLiquidityDensityX96 =
                minLiquidityDensityX96.mulDiv(weight1, weight0 + weight1).mulDiv(totalLiquidity, Q96);
            if (minLiquidityDensityX96 < MIN_LIQUIDITY_DENSITY) {
                return false;
            }
        }

        return true;
    }

    function liquidityDensityX96(
        int24 roundedTick,
        int24 tickSpacing,
        int24 minTick,
        int24 length0,
        int24 length1,
        uint256 alpha0X96,
        uint256 alpha1X96,
        uint256 weight0,
        uint256 weight1
    ) internal pure returns (uint256) {
        return weightedSum(
            LibGeometricDistribution.liquidityDensityX96(
                roundedTick, tickSpacing, minTick + length1 * tickSpacing, length0, alpha0X96
            ),
            weight0,
            LibGeometricDistribution.liquidityDensityX96(roundedTick, tickSpacing, minTick, length1, alpha1X96),
            weight1
        );
    }

    /// @dev Combines several operations used during a swap into one function to save gas.
    ///      Given a cumulative amount, it computes its inverse to find the closest rounded tick, then computes the cumulative amount at that tick,
    ///      and finally computes the liquidity of the tick that will handle the remainder of the swap.
    function computeSwap(
        uint256 inverseCumulativeAmountInput,
        uint256 totalLiquidity,
        bool zeroForOne,
        bool exactIn,
        int24 tickSpacing,
        int24 minTick,
        int24 length0,
        int24 length1,
        uint256 alpha0X96,
        uint256 alpha1X96,
        uint256 weight0,
        uint256 weight1
    )
        internal
        pure
        returns (
            bool success,
            int24 roundedTick,
            uint256 cumulativeAmount0_,
            uint256 cumulativeAmount1_,
            uint256 swapLiquidity
        )
    {
        if (exactIn == zeroForOne) {
            // compute roundedTick by inverting the cumulative amount
            // below is an illustration of 4 rounded ticks, the input amount, and the resulting roundedTick (rick)
            // notice that the inverse tick is between two rounded ticks, and we round down to the rounded tick to the left
            // e.g. go from 1.5 to 1
            //       input
            //      ├──────┤
            // ┌──┬──┬──┬──┐
            // │  │ █│██│██│
            // │  │ █│██│██│
            // └──┴──┴──┴──┘
            // 0  1  2  3  4
            //    │
            //    ▼
            //   rick
            (success, roundedTick) = inverseCumulativeAmount0(
                inverseCumulativeAmountInput,
                totalLiquidity,
                tickSpacing,
                minTick,
                length0,
                length1,
                alpha0X96,
                alpha1X96,
                weight0,
                weight1
            );
            if (!success) return (false, 0, 0, 0, 0);

            // compute the cumulative amount up to roundedTick
            // below is an illustration of the cumulative amount at roundedTick
            // notice that exactIn ? (input - cum) : (cum - input) is the remainder of the swap that will be handled by Uniswap math
            // exactIn:
            //         cum
            //       ├─────┤
            // ┌──┬──┬──┬──┐
            // │  │ █│██│██│
            // │  │ █│██│██│
            // └──┴──┴──┴──┘
            // 0  1  2  3  4
            //       │
            //       ▼
            //      rick + tickSpacing
            // exactOut:
            //        cum
            //    ├────────┤
            // ┌──┬──┬──┬──┐
            // │  │ █│██│██│
            // │  │ █│██│██│
            // └──┴──┴──┴──┘
            // 0  1  2  3  4
            //    │
            //    ▼
            //   rick
            cumulativeAmount0_ = exactIn
                ? cumulativeAmount0(
                    roundedTick + tickSpacing,
                    totalLiquidity,
                    tickSpacing,
                    minTick,
                    length0,
                    length1,
                    alpha0X96,
                    alpha1X96,
                    weight0,
                    weight1
                )
                : cumulativeAmount0(
                    roundedTick,
                    totalLiquidity,
                    tickSpacing,
                    minTick,
                    length0,
                    length1,
                    alpha0X96,
                    alpha1X96,
                    weight0,
                    weight1
                );

            // compute the cumulative amount of the complementary token
            // below is an illustration
            // exactIn:
            //   cum
            // ├─────┤
            // ┌──┬──┬──┬──┐
            // │  │ █│██│██│
            // │  │ █│██│██│
            // └──┴──┴──┴──┘
            // 0  1  2  3  4
            //    │
            //    ▼
            //   rick
            // exactOut:
            //  cum
            // ├──┤
            // ┌──┬──┬──┬──┐
            // │  │ █│██│██│
            // │  │ █│██│██│
            // └──┴──┴──┴──┘
            // 0  1  2  3  4
            // │
            // ▼
            //rick - tickSpacing
            cumulativeAmount1_ = exactIn
                ? cumulativeAmount1(
                    roundedTick,
                    totalLiquidity,
                    tickSpacing,
                    minTick,
                    length0,
                    length1,
                    alpha0X96,
                    alpha1X96,
                    weight0,
                    weight1
                )
                : cumulativeAmount1(
                    roundedTick - tickSpacing,
                    totalLiquidity,
                    tickSpacing,
                    minTick,
                    length0,
                    length1,
                    alpha0X96,
                    alpha1X96,
                    weight0,
                    weight1
                );

            // compute liquidity of the rounded tick that will handle the remainder of the swap
            // below is an illustration of the liquidity of the rounded tick that will handle the remainder of the swap
            //    liq
            //    ├──┤
            // ┌──┬──┬──┬──┐
            // │  │ █│██│██│
            // │  │ █│██│██│
            // └──┴──┴──┴──┘
            // 0  1  2  3  4
            //    │
            //    ▼
            //   rick
            swapLiquidity = (
                liquidityDensityX96(
                    roundedTick, tickSpacing, minTick, length0, length1, alpha0X96, alpha1X96, weight0, weight1
                ) * totalLiquidity
            ) >> 96;
        } else {
            // compute roundedTick by inverting the cumulative amount
            // below is an illustration of 4 rounded ticks, the input amount, and the resulting roundedTick (rick)
            // notice that the inverse tick is between two rounded ticks, and we round up to the rounded tick to the right
            // e.g. go from 1.5 to 2
            //  input
            // ├──────┤
            // ┌──┬──┬──┬──┐
            // │██│██│█ │  │
            // │██│██│█ │  │
            // └──┴──┴──┴──┘
            // 0  1  2  3  4
            //       │
            //       ▼
            //      rick
            (success, roundedTick) = inverseCumulativeAmount1(
                inverseCumulativeAmountInput,
                totalLiquidity,
                tickSpacing,
                minTick,
                length0,
                length1,
                alpha0X96,
                alpha1X96,
                weight0,
                weight1
            );
            if (!success) return (false, 0, 0, 0, 0);

            // compute the cumulative amount up to roundedTick
            // below is an illustration of the cumulative amount at roundedTick
            // notice that exactIn ? (input - cum) : (cum - input) is the remainder of the swap that will be handled by Uniswap math
            // exactIn:
            //   cum
            // ├─────┤
            // ┌──┬──┬──┬──┐
            // │██│██│█ │  │
            // │██│██│█ │  │
            // └──┴──┴──┴──┘
            // 0  1  2  3  4
            //    │
            //    ▼
            //   rick - tickSpacing
            // exactOut:
            //     cum
            // ├────────┤
            // ┌──┬──┬──┬──┐
            // │██│██│█ │  │
            // │██│██│█ │  │
            // └──┴──┴──┴──┘
            // 0  1  2  3  4
            //       │
            //       ▼
            //      rick
            cumulativeAmount1_ = exactIn
                ? cumulativeAmount1(
                    roundedTick - tickSpacing,
                    totalLiquidity,
                    tickSpacing,
                    minTick,
                    length0,
                    length1,
                    alpha0X96,
                    alpha1X96,
                    weight0,
                    weight1
                )
                : cumulativeAmount1(
                    roundedTick,
                    totalLiquidity,
                    tickSpacing,
                    minTick,
                    length0,
                    length1,
                    alpha0X96,
                    alpha1X96,
                    weight0,
                    weight1
                );

            // compute the cumulative amount of the complementary token
            // below is an illustration
            // exactIn:
            //         cum
            //       ├─────┤
            // ┌──┬──┬──┬──┐
            // │██│██│█ │  │
            // │██│██│█ │  │
            // └──┴──┴──┴──┘
            // 0  1  2  3  4
            //       │
            //       ▼
            //      rick
            // exactOut:
            //           cum
            //          ├──┤
            // ┌──┬──┬──┬──┐
            // │██│██│█ │  │
            // │██│██│█ │  │
            // └──┴──┴──┴──┘
            // 0  1  2  3  4
            //          │
            //          ▼
            //         rick + tickSpacing
            cumulativeAmount0_ = exactIn
                ? cumulativeAmount0(
                    roundedTick,
                    totalLiquidity,
                    tickSpacing,
                    minTick,
                    length0,
                    length1,
                    alpha0X96,
                    alpha1X96,
                    weight0,
                    weight1
                )
                : cumulativeAmount0(
                    roundedTick + tickSpacing,
                    totalLiquidity,
                    tickSpacing,
                    minTick,
                    length0,
                    length1,
                    alpha0X96,
                    alpha1X96,
                    weight0,
                    weight1
                );

            // compute liquidity of the rounded tick that will handle the remainder of the swap
            // below is an illustration of the liquidity of the rounded tick that will handle the remainder of the swap
            //       liq
            //       ├──┤
            // ┌──┬──┬──┬──┐
            // │██│██│█ │  │
            // │██│██│█ │  │
            // └──┴──┴──┴──┘
            // 0  1  2  3  4
            //       │
            //       ▼
            //      rick
            swapLiquidity = (
                liquidityDensityX96(
                    roundedTick, tickSpacing, minTick, length0, length1, alpha0X96, alpha1X96, weight0, weight1
                ) * totalLiquidity
            ) >> 96;
        }
    }

    function isValidParams(int24 tickSpacing, uint24 twapSecondsAgo, bytes32 ldfParams, LDFType ldfType)
        internal
        pure
        returns (bool)
    {
        // | shiftMode - 1 byte | minTickOrOffset - 3 bytes | length0 - 2 bytes | alpha0 - 4 bytes | weight0 - 4 bytes | length1 - 2 bytes | alpha1 - 2 bytes | weight1 - 4 bytes |
        uint8 shiftMode = uint8(bytes1(ldfParams));
        int24 minTickOrOffset = int24(uint24(bytes3(ldfParams << 8)));
        int24 length0 = int24(int16(uint16(bytes2(ldfParams << 32))));
        uint32 alpha0 = uint32(bytes4(ldfParams << 48));
        uint256 weight0 = uint32(bytes4(ldfParams << 80));
        int24 length1 = int24(int16(uint16(bytes2(ldfParams << 112))));
        uint32 alpha1 = uint32(bytes4(ldfParams << 128));
        uint256 weight1 = uint32(bytes4(ldfParams << 160));

        // ensure length doesn't overflow when multiplied by tickSpacing
        // ensure length can be contained between minUsableTick and maxUsableTick
        (int24 minUsableTick, int24 maxUsableTick) =
            (TickMath.minUsableTick(tickSpacing), TickMath.maxUsableTick(tickSpacing));
        int24 length = length0 + length1;
        if (
            int256(length) * int256(tickSpacing) > type(int24).max || length > maxUsableTick / tickSpacing
                || -length < minUsableTick / tickSpacing
        ) return false;

        return LibGeometricDistribution.isValidParams(
            tickSpacing,
            twapSecondsAgo,
            bytes32(abi.encodePacked(shiftMode, minTickOrOffset, int16(length1), alpha1)),
            ldfType
        )
            && LibGeometricDistribution.isValidParams(
                tickSpacing,
                twapSecondsAgo,
                bytes32(abi.encodePacked(shiftMode, minTickOrOffset + length1 * tickSpacing, int16(length0), alpha0)),
                ldfType
            ) && weight0 != 0 && weight1 != 0
            && checkMinLiquidityDensity(Q96, tickSpacing, length0, alpha0, weight0, length1, alpha1, weight1);
    }

    /// @return minTick The minimum rounded tick of the distribution
    /// @return length0 The length of the right distribution in number of rounded ticks
    /// @return length1 The length of the left distribution in number of rounded ticks
    /// @return alpha0X96 The alpha of the right distribution
    /// @return alpha1X96 The alpha of the left distribution
    /// @return weight0 The weight of the right distribution
    /// @return weight1 The weight of the left distribution
    /// @return shiftMode The shift mode of the distribution
    function decodeParams(int24 twapTick, int24 tickSpacing, bytes32 ldfParams)
        internal
        pure
        returns (
            int24 minTick,
            int24 length0,
            int24 length1,
            uint256 alpha0X96,
            uint256 alpha1X96,
            uint256 weight0,
            uint256 weight1,
            ShiftMode shiftMode
        )
    {
        // | shiftMode - 1 byte | minTickOrOffset - 3 bytes | length0 - 2 bytes | alpha0 - 4 bytes | weight0 - 4 bytes | length1 - 2 bytes | alpha1 - 4 bytes | weight1 - 4 bytes |
        shiftMode = ShiftMode(uint8(bytes1(ldfParams)));
        length0 = int24(int16(uint16(bytes2(ldfParams << 32))));
        uint256 alpha0 = uint32(bytes4(ldfParams << 48));
        weight0 = uint32(bytes4(ldfParams << 80));
        length1 = int24(int16(uint16(bytes2(ldfParams << 112))));
        uint256 alpha1 = uint32(bytes4(ldfParams << 128));
        weight1 = uint32(bytes4(ldfParams << 160));

        alpha0X96 = alpha0.mulDiv(Q96, ALPHA_BASE);
        alpha1X96 = alpha1.mulDiv(Q96, ALPHA_BASE);

        if (shiftMode != ShiftMode.STATIC) {
            // use rounded TWAP value + offset as minTick
            int24 offset = int24(uint24(bytes3(ldfParams << 8))); // the offset applied to the twap tick to get the minTick
            minTick = roundTickSingle(twapTick + offset, tickSpacing);

            // bound distribution to be within the range of usable ticks
            (int24 minUsableTick, int24 maxUsableTick) =
                (TickMath.minUsableTick(tickSpacing), TickMath.maxUsableTick(tickSpacing));
            if (minTick < minUsableTick) {
                minTick = minUsableTick;
            } else if (minTick > maxUsableTick - (length0 + length1) * tickSpacing) {
                minTick = maxUsableTick - (length0 + length1) * tickSpacing;
            }
        } else {
            // static minTick set in params
            minTick = int24(uint24(bytes3(ldfParams << 8))); // must be aligned to tickSpacing
        }
    }
}
