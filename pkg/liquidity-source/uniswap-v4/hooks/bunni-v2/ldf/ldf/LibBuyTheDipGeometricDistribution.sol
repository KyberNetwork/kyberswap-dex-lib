// SPDX-License-Identifier: BUSL-1.1
pragma solidity ^0.8.19;

import {FixedPointMathLib} from "solady/utils/FixedPointMathLib.sol";

import {TickMath} from "@uniswap/v4-core/src/libraries/TickMath.sol";

import "../lib/Math.sol";
import "../base/Constants.sol";
import "./LibGeometricDistribution.sol";
import {LDFType} from "../types/LDFType.sol";

library LibBuyTheDipGeometricDistribution {
    using TickMath for *;
    using FixedPointMathLib for uint256;

    uint256 internal constant MIN_ALPHA = 1e3;
    uint256 internal constant MAX_ALPHA = 12e8;
    uint256 internal constant ALPHA_BASE = 1e8; // alpha uses 8 decimals in ldfParams

    /// @dev Queries the liquidity density and the cumulative amounts at the given rounded tick.
    /// @param roundedTick The rounded tick to query
    /// @param tickSpacing The spacing of the ticks
    /// @return liquidityDensityX96_ The liquidity density at the given rounded tick. Range is [0, 1]. Scaled by 2^96.
    /// @return cumulativeAmount0DensityX96 The cumulative amount of token0 in the rounded ticks [roundedTick + tickSpacing, minTick + length * tickSpacing)
    /// @return cumulativeAmount1DensityX96 The cumulative amount of token1 in the rounded ticks [minTick, roundedTick - tickSpacing]
    function query(
        int24 roundedTick,
        int24 tickSpacing,
        int24 twapTick,
        int24 minTick,
        int24 length,
        uint256 alphaX96,
        uint256 altAlphaX96,
        int24 altThreshold,
        bool altThresholdDirection
    )
        internal
        pure
        returns (uint256 liquidityDensityX96_, uint256 cumulativeAmount0DensityX96, uint256 cumulativeAmount1DensityX96)
    {
        // compute liquidityDensityX96
        liquidityDensityX96_ = liquidityDensityX96(
            roundedTick,
            tickSpacing,
            twapTick,
            minTick,
            length,
            alphaX96,
            altAlphaX96,
            altThreshold,
            altThresholdDirection
        );

        // compute cumulativeAmount0DensityX96
        cumulativeAmount0DensityX96 = cumulativeAmount0(
            roundedTick + tickSpacing,
            Q96,
            tickSpacing,
            twapTick,
            minTick,
            length,
            alphaX96,
            altAlphaX96,
            altThreshold,
            altThresholdDirection
        );

        // compute cumulativeAmount1DensityX96
        cumulativeAmount1DensityX96 = cumulativeAmount1(
            roundedTick - tickSpacing,
            Q96,
            tickSpacing,
            twapTick,
            minTick,
            length,
            alphaX96,
            altAlphaX96,
            altThreshold,
            altThresholdDirection
        );
    }

    /// @dev Computes the cumulative amount of token0 in the rounded ticks [roundedTick, tickUpper).
    function cumulativeAmount0(
        int24 roundedTick,
        uint256 totalLiquidity,
        int24 tickSpacing,
        int24 twapTick,
        int24 minTick,
        int24 length,
        uint256 alphaX96,
        uint256 altAlphaX96,
        int24 altThreshold,
        bool altThresholdDirection
    ) internal pure returns (uint256 amount0) {
        if (shouldUseAltAlpha(twapTick, altThreshold, altThresholdDirection)) {
            return LibGeometricDistribution.cumulativeAmount0(
                roundedTick, totalLiquidity, tickSpacing, minTick, length, altAlphaX96
            );
        } else {
            return LibGeometricDistribution.cumulativeAmount0(
                roundedTick, totalLiquidity, tickSpacing, minTick, length, alphaX96
            );
        }
    }

    /// @dev Computes the cumulative amount of token1 in the rounded ticks [tickLower, roundedTick].
    function cumulativeAmount1(
        int24 roundedTick,
        uint256 totalLiquidity,
        int24 tickSpacing,
        int24 twapTick,
        int24 minTick,
        int24 length,
        uint256 alphaX96,
        uint256 altAlphaX96,
        int24 altThreshold,
        bool altThresholdDirection
    ) internal pure returns (uint256 amount1) {
        if (shouldUseAltAlpha(twapTick, altThreshold, altThresholdDirection)) {
            return LibGeometricDistribution.cumulativeAmount1(
                roundedTick, totalLiquidity, tickSpacing, minTick, length, altAlphaX96
            );
        } else {
            return LibGeometricDistribution.cumulativeAmount1(
                roundedTick, totalLiquidity, tickSpacing, minTick, length, alphaX96
            );
        }
    }

    /// @dev Given a cumulativeAmount0, computes the rounded tick whose cumulativeAmount0 is closest to the input. Range is [tickLower, tickUpper].
    ///      The returned tick will be the largest rounded tick whose cumulativeAmount0 is greater than or equal to the input.
    ///      In the case that the input exceeds the cumulativeAmount0 of all rounded ticks, the function will return (false, 0).
    function inverseCumulativeAmount0(
        uint256 cumulativeAmount0_,
        uint256 totalLiquidity,
        int24 tickSpacing,
        int24 twapTick,
        int24 minTick,
        int24 length,
        uint256 alphaX96,
        uint256 altAlphaX96,
        int24 altThreshold,
        bool altThresholdDirection
    ) internal pure returns (bool success, int24 roundedTick) {
        if (shouldUseAltAlpha(twapTick, altThreshold, altThresholdDirection)) {
            return LibGeometricDistribution.inverseCumulativeAmount0(
                cumulativeAmount0_, totalLiquidity, tickSpacing, minTick, length, altAlphaX96
            );
        } else {
            return LibGeometricDistribution.inverseCumulativeAmount0(
                cumulativeAmount0_, totalLiquidity, tickSpacing, minTick, length, alphaX96
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
        int24 twapTick,
        int24 minTick,
        int24 length,
        uint256 alphaX96,
        uint256 altAlphaX96,
        int24 altThreshold,
        bool altThresholdDirection
    ) internal pure returns (bool success, int24 roundedTick) {
        if (shouldUseAltAlpha(twapTick, altThreshold, altThresholdDirection)) {
            return LibGeometricDistribution.inverseCumulativeAmount1(
                cumulativeAmount1_, totalLiquidity, tickSpacing, minTick, length, altAlphaX96
            );
        } else {
            return LibGeometricDistribution.inverseCumulativeAmount1(
                cumulativeAmount1_, totalLiquidity, tickSpacing, minTick, length, alphaX96
            );
        }
    }

    function liquidityDensityX96(
        int24 roundedTick,
        int24 tickSpacing,
        int24 twapTick,
        int24 minTick,
        int24 length,
        uint256 alphaX96,
        uint256 altAlphaX96,
        int24 altThreshold,
        bool altThresholdDirection
    ) internal pure returns (uint256) {
        if (shouldUseAltAlpha(twapTick, altThreshold, altThresholdDirection)) {
            return LibGeometricDistribution.liquidityDensityX96(roundedTick, tickSpacing, minTick, length, altAlphaX96);
        } else {
            return LibGeometricDistribution.liquidityDensityX96(roundedTick, tickSpacing, minTick, length, alphaX96);
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
        int24 twapTick,
        int24 minTick,
        int24 length,
        uint256 alphaX96,
        uint256 altAlphaX96,
        int24 altThreshold,
        bool altThresholdDirection
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
                twapTick,
                minTick,
                length,
                alphaX96,
                altAlphaX96,
                altThreshold,
                altThresholdDirection
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
                    twapTick,
                    minTick,
                    length,
                    alphaX96,
                    altAlphaX96,
                    altThreshold,
                    altThresholdDirection
                )
                : cumulativeAmount0(
                    roundedTick,
                    totalLiquidity,
                    tickSpacing,
                    twapTick,
                    minTick,
                    length,
                    alphaX96,
                    altAlphaX96,
                    altThreshold,
                    altThresholdDirection
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
                    twapTick,
                    minTick,
                    length,
                    alphaX96,
                    altAlphaX96,
                    altThreshold,
                    altThresholdDirection
                )
                : cumulativeAmount1(
                    roundedTick - tickSpacing,
                    totalLiquidity,
                    tickSpacing,
                    twapTick,
                    minTick,
                    length,
                    alphaX96,
                    altAlphaX96,
                    altThreshold,
                    altThresholdDirection
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
                    roundedTick,
                    tickSpacing,
                    twapTick,
                    minTick,
                    length,
                    alphaX96,
                    altAlphaX96,
                    altThreshold,
                    altThresholdDirection
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
                twapTick,
                minTick,
                length,
                alphaX96,
                altAlphaX96,
                altThreshold,
                altThresholdDirection
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
                    twapTick,
                    minTick,
                    length,
                    alphaX96,
                    altAlphaX96,
                    altThreshold,
                    altThresholdDirection
                )
                : cumulativeAmount1(
                    roundedTick,
                    totalLiquidity,
                    tickSpacing,
                    twapTick,
                    minTick,
                    length,
                    alphaX96,
                    altAlphaX96,
                    altThreshold,
                    altThresholdDirection
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
                    twapTick,
                    minTick,
                    length,
                    alphaX96,
                    altAlphaX96,
                    altThreshold,
                    altThresholdDirection
                )
                : cumulativeAmount0(
                    roundedTick + tickSpacing,
                    totalLiquidity,
                    tickSpacing,
                    twapTick,
                    minTick,
                    length,
                    alphaX96,
                    altAlphaX96,
                    altThreshold,
                    altThresholdDirection
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
                    roundedTick,
                    tickSpacing,
                    twapTick,
                    minTick,
                    length,
                    alphaX96,
                    altAlphaX96,
                    altThreshold,
                    altThresholdDirection
                ) * totalLiquidity
            ) >> 96;
        }
    }

    function isValidParams(int24 tickSpacing, uint24 twapSecondsAgo, bytes32 ldfParams, LDFType ldfType)
        internal
        pure
        returns (bool)
    {
        // decode params
        // | shiftMode - 1 byte | minTick - 3 bytes | length - 2 bytes | alpha - 4 bytes | altAlpha - 4 bytes | altThreshold - 3 bytes | altThresholdDirection - 1 byte |
        uint8 shiftMode = uint8(bytes1(ldfParams));
        int24 minTick = int24(uint24(bytes3(ldfParams << 8)));
        int24 length = int24(int16(uint16(bytes2(ldfParams << 32))));
        uint32 alpha = uint32(bytes4(ldfParams << 48));
        uint32 altAlpha = uint32(bytes4(ldfParams << 80));
        int24 altThreshold = int24(uint24(bytes3(ldfParams << 112)));

        bytes32 altLdfParams = bytes32(abi.encodePacked(shiftMode, minTick, int16(length), altAlpha));

        // validity conditions:
        // - need TWAP to be enabled to trigger the alt alpha switch
        // - shiftMode is static
        // - ldfType is DYNAMIC_AND_STATEFUL
        // - both LDFs are valid
        // - threshold makes sense i.e. both LDFs can be used at some point
        // - alpha and altAlpha are on different sides of 1
        return (twapSecondsAgo != 0) && (shiftMode == uint8(ShiftMode.STATIC))
            && ldfType == LDFType.DYNAMIC_AND_STATEFUL && geometricIsValidParams(tickSpacing, ldfParams)
            && geometricIsValidParams(tickSpacing, altLdfParams) && altThreshold < minTick + length * tickSpacing
            && altThreshold > minTick && ((alpha < ALPHA_BASE) != (altAlpha < ALPHA_BASE));
    }

    /// @dev Should be the same as LibGeometricDistribution.isValidParams but without checks for minimum liquidity.
    /// This LDF requires one end of the distribution to have essentially 0 liquidity so that when the alt LDF
    /// is activated liquidity can move to a specified price to "buy the dip".
    function geometricIsValidParams(int24 tickSpacing, bytes32 ldfParams) internal pure returns (bool) {
        (int24 minUsableTick, int24 maxUsableTick) =
            (TickMath.minUsableTick(tickSpacing), TickMath.maxUsableTick(tickSpacing));

        // | shiftMode - 1 byte | minTickOrOffset - 3 bytes | length - 2 bytes | alpha - 4 bytes |
        uint8 shiftMode = uint8(bytes1(ldfParams));
        int24 minTickOrOffset = int24(uint24(bytes3(ldfParams << 8)));
        int24 length = int24(int16(uint16(bytes2(ldfParams << 32))));
        uint256 alpha = uint32(bytes4(ldfParams << 48));

        // ensure minTickOrOffset is aligned to tickSpacing
        if (minTickOrOffset % tickSpacing != 0) {
            return false;
        }

        // ensure length > 0 and doesn't overflow when multiplied by tickSpacing
        // ensure length can be contained between minUsableTick and maxUsableTick
        if (
            length <= 0 || int256(length) * int256(tickSpacing) > type(int24).max
                || length > maxUsableTick / tickSpacing || -length < minUsableTick / tickSpacing
        ) return false;

        // ensure alpha is in range
        if (alpha < MIN_ALPHA || alpha > MAX_ALPHA || alpha == ALPHA_BASE) return false;

        // ensure alpha != sqrtRatioTickSpacing which would cause cum0 to always be 0
        uint256 alphaX96 = alpha.mulDiv(Q96, ALPHA_BASE);
        uint160 sqrtRatioTickSpacing = tickSpacing.getSqrtPriceAtTick();
        if (alphaX96 == sqrtRatioTickSpacing) return false;

        // ensure the ticks are within the valid range
        if (shiftMode == uint8(ShiftMode.STATIC)) {
            // static minTick set in params
            int24 maxTick = minTickOrOffset + length * tickSpacing;
            if (minTickOrOffset < minUsableTick || maxTick > maxUsableTick) return false;
        }

        // if all conditions are met, return true
        return true;
    }

    /// @return minTick The minimum rounded tick of the distribution
    /// @return length The length of the geometric distribution in number of rounded ticks
    /// @return alphaX96 The alpha of the geometric distribution
    /// @return altAlphaX96 The alternative alpha value used when (altThresholdDirection ? twapTick <= altThreshold : twapTick >= altThreshold)
    /// @return altThreshold The threshold used to switch to the alternative alpha value
    /// @return altThresholdDirection The direction of the threshold. True if the alternative alpha value is used when twapTick < altThreshold, false if when twapTick > altThreshold
    function decodeParams(bytes32 ldfParams)
        internal
        pure
        returns (
            int24 minTick,
            int24 length,
            uint256 alphaX96,
            uint256 altAlphaX96,
            int24 altThreshold,
            bool altThresholdDirection
        )
    {
        // static minTick set in params
        // | shiftMode - 1 byte | minTick - 3 bytes | length - 2 bytes | alpha - 4 bytes | altAlpha - 4 bytes | altThreshold - 3 bytes | altThresholdDirection - 1 byte |
        minTick = int24(uint24(bytes3(ldfParams << 8))); // must be aligned to tickSpacing
        length = int24(int16(uint16(bytes2(ldfParams << 32))));
        uint256 alpha = uint32(bytes4(ldfParams << 48));
        alphaX96 = alpha.mulDiv(Q96, ALPHA_BASE);
        uint256 altAlpha = uint32(bytes4(ldfParams << 80));
        altAlphaX96 = altAlpha.mulDiv(Q96, ALPHA_BASE);
        altThreshold = int24(uint24(bytes3(ldfParams << 112)));
        altThresholdDirection = uint8(bytes1(ldfParams << 136)) != 0;
    }

    /// @dev Whether the alternative alpha value should be used based on the TWAP tick and the threshold.
    function shouldUseAltAlpha(int24 twapTick, int24 altThreshold, bool altThresholdDirection)
        internal
        pure
        returns (bool)
    {
        return altThresholdDirection ? twapTick <= altThreshold : twapTick >= altThreshold;
    }
}
