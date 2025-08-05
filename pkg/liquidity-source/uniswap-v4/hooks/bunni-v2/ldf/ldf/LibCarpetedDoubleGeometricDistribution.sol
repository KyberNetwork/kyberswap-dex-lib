// SPDX-License-Identifier: BUSL-1.1
pragma solidity ^0.8.19;

import {SafeCastLib} from "solady/utils/SafeCastLib.sol";
import {FixedPointMathLib} from "solady/utils/FixedPointMathLib.sol";

import {TickMath} from "@uniswap/v4-core/src/libraries/TickMath.sol";

import "./ShiftMode.sol";
import "../lib/Math.sol";
import "../lib/ExpMath.sol";
import "../base/Constants.sol";
import "./LibUniformDistribution.sol";
import {LDFType} from "../types/LDFType.sol";
import "./LibDoubleGeometricDistribution.sol";

library LibCarpetedDoubleGeometricDistribution {
    using SafeCastLib for uint256;
    using FixedPointMathLib for uint256;

    uint256 internal constant SCALED_Q96 = 0x10000000000000000000000000; // Q96 << QUERY_SCALE_SHIFT
    uint8 internal constant QUERY_SCALE_SHIFT = 4;

    struct Params {
        int24 minTick;
        int24 length0;
        uint256 alpha0X96;
        uint256 weight0;
        int24 length1;
        uint256 alpha1X96;
        uint256 weight1;
        uint256 weightCarpet;
        ShiftMode shiftMode;
    }

    /// @dev Queries the liquidity density and the cumulative amounts at the given rounded tick.
    /// @param roundedTick The rounded tick to query
    /// @param tickSpacing The spacing of the ticks
    /// @return liquidityDensityX96_ The liquidity density at the given rounded tick. Range is [0, 1]. Scaled by 2^96.
    /// @return cumulativeAmount0DensityX96 The cumulative amount of token0 in the rounded ticks [roundedTick + tickSpacing, minTick + length * tickSpacing)
    /// @return cumulativeAmount1DensityX96 The cumulative amount of token1 in the rounded ticks [minTick, roundedTick - tickSpacing]
    function query(int24 roundedTick, int24 tickSpacing, Params memory params)
        internal
        pure
        returns (uint256 liquidityDensityX96_, uint256 cumulativeAmount0DensityX96, uint256 cumulativeAmount1DensityX96)
    {
        // compute liquidityDensityX96
        liquidityDensityX96_ = liquidityDensityX96(roundedTick, tickSpacing, params);

        // compute cumulativeAmount0DensityX96
        cumulativeAmount0DensityX96 =
            cumulativeAmount0(roundedTick + tickSpacing, SCALED_Q96, tickSpacing, params) >> QUERY_SCALE_SHIFT;

        // compute cumulativeAmount1DensityX96
        cumulativeAmount1DensityX96 =
            cumulativeAmount1(roundedTick - tickSpacing, SCALED_Q96, tickSpacing, params) >> QUERY_SCALE_SHIFT;
    }

    /// @dev Computes the cumulative amount of token0 in the rounded ticks [roundedTick, tickUpper).
    function cumulativeAmount0(int24 roundedTick, uint256 totalLiquidity, int24 tickSpacing, Params memory params)
        internal
        pure
        returns (uint256 amount0)
    {
        int24 length = params.length0 + params.length1;
        (
            uint256 leftCarpetLiquidity,
            uint256 mainLiquidity,
            uint256 rightCarpetLiquidity,
            int24 minUsableTick,
            int24 maxUsableTick
        ) = getCarpetedLiquidity(totalLiquidity, tickSpacing, params.minTick, length, params.weightCarpet);

        return LibUniformDistribution.cumulativeAmount0(
            roundedTick, leftCarpetLiquidity, tickSpacing, minUsableTick, params.minTick, true
        )
            + LibDoubleGeometricDistribution.cumulativeAmount0(
                roundedTick,
                mainLiquidity,
                tickSpacing,
                params.minTick,
                params.length0,
                params.length1,
                params.alpha0X96,
                params.alpha1X96,
                params.weight0,
                params.weight1
            )
            + LibUniformDistribution.cumulativeAmount0(
                roundedTick, rightCarpetLiquidity, tickSpacing, params.minTick + length * tickSpacing, maxUsableTick, true
            );
    }

    /// @dev Computes the cumulative amount of token1 in the rounded ticks [tickLower, roundedTick].
    function cumulativeAmount1(int24 roundedTick, uint256 totalLiquidity, int24 tickSpacing, Params memory params)
        internal
        pure
        returns (uint256 amount1)
    {
        int24 length = params.length0 + params.length1;
        (
            uint256 leftCarpetLiquidity,
            uint256 mainLiquidity,
            uint256 rightCarpetLiquidity,
            int24 minUsableTick,
            int24 maxUsableTick
        ) = getCarpetedLiquidity(totalLiquidity, tickSpacing, params.minTick, length, params.weightCarpet);

        return LibUniformDistribution.cumulativeAmount1(
            roundedTick, leftCarpetLiquidity, tickSpacing, minUsableTick, params.minTick, true
        )
            + LibDoubleGeometricDistribution.cumulativeAmount1(
                roundedTick,
                mainLiquidity,
                tickSpacing,
                params.minTick,
                params.length0,
                params.length1,
                params.alpha0X96,
                params.alpha1X96,
                params.weight0,
                params.weight1
            )
            + LibUniformDistribution.cumulativeAmount1(
                roundedTick, rightCarpetLiquidity, tickSpacing, params.minTick + length * tickSpacing, maxUsableTick, true
            );
    }

    /// @dev Given a cumulativeAmount0, computes the rounded tick whose cumulativeAmount0 is closest to the input. Range is [tickLower, tickUpper].
    ///      The returned tick will be the largest rounded tick whose cumulativeAmount0 is greater than or equal to the input.
    ///      In the case that the input exceeds the cumulativeAmount0 of all rounded ticks, the function will return (false, 0).
    function inverseCumulativeAmount0(
        uint256 cumulativeAmount0_,
        uint256 totalLiquidity,
        int24 tickSpacing,
        Params memory params
    ) internal pure returns (bool success, int24 roundedTick) {
        if (cumulativeAmount0_ == 0) {
            return (true, TickMath.maxUsableTick(tickSpacing));
        }

        // try LDFs in the order of right carpet, main, left carpet
        int24 length = params.length0 + params.length1;
        (
            uint256 leftCarpetLiquidity,
            uint256 mainLiquidity,
            uint256 rightCarpetLiquidity,
            int24 minUsableTick,
            int24 maxUsableTick
        ) = getCarpetedLiquidity(totalLiquidity, tickSpacing, params.minTick, length, params.weightCarpet);
        uint256 rightCarpetCumulativeAmount0 = LibUniformDistribution.cumulativeAmount0(
            params.minTick + length * tickSpacing,
            rightCarpetLiquidity,
            tickSpacing,
            params.minTick + length * tickSpacing,
            maxUsableTick,
            true
        );

        if (cumulativeAmount0_ <= rightCarpetCumulativeAmount0 && rightCarpetLiquidity != 0) {
            // use right carpet
            return LibUniformDistribution.inverseCumulativeAmount0(
                cumulativeAmount0_,
                rightCarpetLiquidity,
                tickSpacing,
                params.minTick + length * tickSpacing,
                maxUsableTick,
                true
            );
        } else {
            uint256 remainder = cumulativeAmount0_ - rightCarpetCumulativeAmount0;
            uint256 mainCumulativeAmount0 = LibDoubleGeometricDistribution.cumulativeAmount0(
                params.minTick,
                mainLiquidity,
                tickSpacing,
                params.minTick,
                params.length0,
                params.length1,
                params.alpha0X96,
                params.alpha1X96,
                params.weight0,
                params.weight1
            );

            if (remainder <= mainCumulativeAmount0) {
                // use main
                return LibDoubleGeometricDistribution.inverseCumulativeAmount0(
                    remainder,
                    mainLiquidity,
                    tickSpacing,
                    params.minTick,
                    params.length0,
                    params.length1,
                    params.alpha0X96,
                    params.alpha1X96,
                    params.weight0,
                    params.weight1
                );
            } else if (leftCarpetLiquidity != 0) {
                // use left carpet
                remainder -= mainCumulativeAmount0;
                return LibUniformDistribution.inverseCumulativeAmount0(
                    remainder, leftCarpetLiquidity, tickSpacing, minUsableTick, params.minTick, true
                );
            }
        }
        return (false, 0);
    }

    /// @dev Given a cumulativeAmount1, computes the rounded tick whose cumulativeAmount1 is closest to the input. Range is [tickLower - tickSpacing, tickUpper - tickSpacing].
    ///      The returned tick will be the smallest rounded tick whose cumulativeAmount1 is greater than or equal to the input.
    ///      In the case that the input exceeds the cumulativeAmount1 of all rounded ticks, the function will return (false, 0).
    function inverseCumulativeAmount1(
        uint256 cumulativeAmount1_,
        uint256 totalLiquidity,
        int24 tickSpacing,
        Params memory params
    ) internal pure returns (bool success, int24 roundedTick) {
        if (cumulativeAmount1_ == 0) {
            return (true, TickMath.minUsableTick(tickSpacing) - tickSpacing);
        }

        // try LDFs in the order of left carpet, main, right carpet
        int24 length = params.length0 + params.length1;
        (
            uint256 leftCarpetLiquidity,
            uint256 mainLiquidity,
            uint256 rightCarpetLiquidity,
            int24 minUsableTick,
            int24 maxUsableTick
        ) = getCarpetedLiquidity(totalLiquidity, tickSpacing, params.minTick, length, params.weightCarpet);
        uint256 leftCarpetCumulativeAmount1 = LibUniformDistribution.cumulativeAmount1(
            params.minTick, leftCarpetLiquidity, tickSpacing, minUsableTick, params.minTick, true
        );

        if (cumulativeAmount1_ <= leftCarpetCumulativeAmount1 && leftCarpetLiquidity != 0) {
            // use left carpet
            return LibUniformDistribution.inverseCumulativeAmount1(
                cumulativeAmount1_, leftCarpetLiquidity, tickSpacing, minUsableTick, params.minTick, true
            );
        } else {
            uint256 remainder = cumulativeAmount1_ - leftCarpetCumulativeAmount1;
            uint256 mainCumulativeAmount1 = LibDoubleGeometricDistribution.cumulativeAmount1(
                params.minTick + length * tickSpacing,
                mainLiquidity,
                tickSpacing,
                params.minTick,
                params.length0,
                params.length1,
                params.alpha0X96,
                params.alpha1X96,
                params.weight0,
                params.weight1
            );

            if (remainder <= mainCumulativeAmount1) {
                // use main
                return LibDoubleGeometricDistribution.inverseCumulativeAmount1(
                    remainder,
                    mainLiquidity,
                    tickSpacing,
                    params.minTick,
                    params.length0,
                    params.length1,
                    params.alpha0X96,
                    params.alpha1X96,
                    params.weight0,
                    params.weight1
                );
            } else if (rightCarpetLiquidity != 0) {
                // use right carpet
                remainder -= mainCumulativeAmount1;
                return LibUniformDistribution.inverseCumulativeAmount1(
                    remainder,
                    rightCarpetLiquidity,
                    tickSpacing,
                    params.minTick + length * tickSpacing,
                    maxUsableTick,
                    true
                );
            }
        }
        return (false, 0);
    }

    function liquidityDensityX96(int24 roundedTick, int24 tickSpacing, Params memory params)
        internal
        pure
        returns (uint256)
    {
        int24 length = params.length0 + params.length1;
        if (roundedTick >= params.minTick && roundedTick < params.minTick + length * tickSpacing) {
            return LibDoubleGeometricDistribution.liquidityDensityX96(
                roundedTick,
                tickSpacing,
                params.minTick,
                params.length0,
                params.length1,
                params.alpha0X96,
                params.alpha1X96,
                params.weight0,
                params.weight1
            ).mulWad(WAD - params.weightCarpet);
        } else {
            (int24 minUsableTick, int24 maxUsableTick) =
                (TickMath.minUsableTick(tickSpacing), TickMath.maxUsableTick(tickSpacing));
            int24 numRoundedTicksCarpeted = (maxUsableTick - minUsableTick) / tickSpacing - length;
            if (numRoundedTicksCarpeted <= 0) {
                return 0;
            }
            uint256 mainLiquidity = Q96.mulWad(WAD - params.weightCarpet);
            uint256 carpetLiquidity = Q96 - mainLiquidity;
            return carpetLiquidity.divUp(uint24(numRoundedTicksCarpeted));
        }
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
        Params memory params
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
            (success, roundedTick) =
                inverseCumulativeAmount0(inverseCumulativeAmountInput, totalLiquidity, tickSpacing, params);
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
                ? cumulativeAmount0(roundedTick + tickSpacing, totalLiquidity, tickSpacing, params)
                : cumulativeAmount0(roundedTick, totalLiquidity, tickSpacing, params);

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
                ? cumulativeAmount1(roundedTick, totalLiquidity, tickSpacing, params)
                : cumulativeAmount1(roundedTick - tickSpacing, totalLiquidity, tickSpacing, params);

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
            swapLiquidity = (liquidityDensityX96(roundedTick, tickSpacing, params) * totalLiquidity) >> 96;
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
            (success, roundedTick) =
                inverseCumulativeAmount1(inverseCumulativeAmountInput, totalLiquidity, tickSpacing, params);
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
                ? cumulativeAmount1(roundedTick - tickSpacing, totalLiquidity, tickSpacing, params)
                : cumulativeAmount1(roundedTick, totalLiquidity, tickSpacing, params);

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
                ? cumulativeAmount0(roundedTick, totalLiquidity, tickSpacing, params)
                : cumulativeAmount0(roundedTick + tickSpacing, totalLiquidity, tickSpacing, params);

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
            swapLiquidity = (liquidityDensityX96(roundedTick, tickSpacing, params) * totalLiquidity) >> 96;
        }
    }

    function getCarpetedLiquidity(
        uint256 totalLiquidity,
        int24 tickSpacing,
        int24 minTick,
        int24 length,
        uint256 weightCarpet
    )
        internal
        pure
        returns (
            uint256 leftCarpetLiquidity,
            uint256 mainLiquidity,
            uint256 rightCarpetLiquidity,
            int24 minUsableTick,
            int24 maxUsableTick
        )
    {
        (minUsableTick, maxUsableTick) = (TickMath.minUsableTick(tickSpacing), TickMath.maxUsableTick(tickSpacing));
        int24 numRoundedTicksCarpeted = (maxUsableTick - minUsableTick) / tickSpacing - length;
        if (numRoundedTicksCarpeted <= 0) {
            return (0, totalLiquidity, 0, minUsableTick, maxUsableTick);
        }
        mainLiquidity = totalLiquidity.mulWad(WAD - weightCarpet);
        uint256 carpetLiquidity = totalLiquidity - mainLiquidity;
        rightCarpetLiquidity = carpetLiquidity.mulDiv(
            uint24((maxUsableTick - minTick) / tickSpacing - length), uint24(numRoundedTicksCarpeted)
        );
        leftCarpetLiquidity = carpetLiquidity - rightCarpetLiquidity;
    }

    function isValidParams(int24 tickSpacing, uint24 twapSecondsAgo, bytes32 ldfParams, LDFType ldfType)
        internal
        pure
        returns (bool)
    {
        // | shiftMode - 1 byte | minTickOrOffset - 3 bytes | length0 - 2 bytes | alpha0 - 4 bytes | weight0 - 4 bytes | length1 - 2 bytes | alpha1 - 4 bytes | weight1 - 4 bytes | weightCarpet - 4 bytes |
        int24 length0 = int24(int16(uint16(bytes2(ldfParams << 32))));
        uint32 alpha0 = uint32(bytes4(ldfParams << 48));
        uint32 weight0 = uint32(bytes4(ldfParams << 80));
        int24 length1 = int24(int16(uint16(bytes2(ldfParams << 112))));
        uint32 alpha1 = uint32(bytes4(ldfParams << 128));
        uint32 weight1 = uint32(bytes4(ldfParams << 160));
        uint32 weightCarpet = uint32(bytes4(ldfParams << 192));

        return LibDoubleGeometricDistribution.isValidParams(tickSpacing, twapSecondsAgo, ldfParams, ldfType)
            && weightCarpet != 0
            && LibDoubleGeometricDistribution.checkMinLiquidityDensity(
                Q96.mulWad(WAD - weightCarpet), tickSpacing, length0, alpha0, weight0, length1, alpha1, weight1
            );
    }

    /// @return params
    /// minTick The minimum rounded tick of the distribution
    /// length0 The length of the right distribution in number of rounded ticks
    /// length1 The length of the left distribution in number of rounded ticks
    /// alpha0X96 The alpha of the right distribution
    /// alpha1X96 The alpha of the left distribution
    /// weight0 The weight of the right distribution
    /// weight1 The weight of the left distribution
    /// weightCarpet The weight of the carpet distribution, 18 decimals. 32 bits means the max weight is 4.295e-9.
    /// shiftMode The shift mode of the distribution
    function decodeParams(int24 twapTick, int24 tickSpacing, bytes32 ldfParams)
        internal
        pure
        returns (Params memory params)
    {
        // | shiftMode - 1 byte | offset - 3 bytes | length0 - 2 bytes | alpha0 - 4 bytes | weight0 - 4 bytes | length1 - 2 bytes | alpha1 - 4 bytes | weight1 - 4 bytes | weightCarpet - 4 bytes |
        params.weightCarpet = uint32(bytes4(ldfParams << 192));
        (
            params.minTick,
            params.length0,
            params.length1,
            params.alpha0X96,
            params.alpha1X96,
            params.weight0,
            params.weight1,
            params.shiftMode
        ) = LibDoubleGeometricDistribution.decodeParams(twapTick, tickSpacing, ldfParams);
    }
}
