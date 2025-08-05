// SPDX-License-Identifier: BUSL-1.1
pragma solidity ^0.8.19;

import {SafeCastLib} from "solady/utils/SafeCastLib.sol";
import {FixedPointMathLib} from "solady/utils/FixedPointMathLib.sol";

import {TickMath} from "@uniswap/v4-core/src/libraries/TickMath.sol";

import "./ShiftMode.sol";
import "../lib/Math.sol";
import "../base/Constants.sol";
import {LDFType} from "../types/LDFType.sol";
import {FullMathX96} from "../lib/FullMathX96.sol";
import {SqrtPriceMath} from "../lib/SqrtPriceMath.sol";

library LibUniformDistribution {
    using TickMath for int24;
    using TickMath for uint160;
    using FixedPointMathLib for *;
    using SafeCastLib for uint256;
    using FullMathX96 for uint256;

    /// @dev Queries the liquidity density and the cumulative amounts at the given rounded tick.
    /// @param roundedTick The rounded tick to query
    /// @param tickSpacing The spacing of the ticks
    /// @return liquidityDensityX96_ The liquidity density at the given rounded tick. Range is [0, 1]. Scaled by 2^96.
    /// @return cumulativeAmount0DensityX96 The cumulative amount of token0 in the rounded ticks [roundedTick + tickSpacing, tickUpper)
    /// @return cumulativeAmount1DensityX96 The cumulative amount of token1 in the rounded ticks [tickLower, roundedTick - tickSpacing]
    function query(int24 roundedTick, int24 tickSpacing, int24 tickLower, int24 tickUpper)
        internal
        pure
        returns (uint256 liquidityDensityX96_, uint256 cumulativeAmount0DensityX96, uint256 cumulativeAmount1DensityX96)
    {
        // compute liquidityDensityX96
        liquidityDensityX96_ = liquidityDensityX96(roundedTick, tickSpacing, tickLower, tickUpper);

        uint24 length = uint24((tickUpper - tickLower) / tickSpacing);
        uint256 liquidity = Q96.divUp(length);

        uint160 sqrtRatioTickLower = tickLower.getSqrtPriceAtTick();
        uint160 sqrtRatioTickUpper = tickUpper.getSqrtPriceAtTick();

        // compute cumulativeAmount0DensityX96 for the rounded tick to the right of the rounded current tick
        if (roundedTick + tickSpacing >= tickUpper) {
            // cumulativeAmount0DensityX96 is just 0
            cumulativeAmount0DensityX96 = 0;
        } else if (roundedTick + tickSpacing <= tickLower) {
            cumulativeAmount0DensityX96 =
                SqrtPriceMath.getAmount0Delta(sqrtRatioTickLower, sqrtRatioTickUpper, liquidity, true);
        } else {
            cumulativeAmount0DensityX96 = SqrtPriceMath.getAmount0Delta(
                (roundedTick + tickSpacing).getSqrtPriceAtTick(), sqrtRatioTickUpper, liquidity, true
            );
        }

        // compute cumulativeAmount1DensityX96 for the rounded tick to the left of the rounded current tick
        if (roundedTick - tickSpacing < tickLower) {
            // cumulativeAmount1DensityX96 is just 0
            cumulativeAmount1DensityX96 = 0;
        } else if (roundedTick >= tickUpper) {
            cumulativeAmount1DensityX96 =
                SqrtPriceMath.getAmount1Delta(sqrtRatioTickLower, sqrtRatioTickUpper, liquidity, true);
        } else {
            cumulativeAmount1DensityX96 =
                SqrtPriceMath.getAmount1Delta(sqrtRatioTickLower, roundedTick.getSqrtPriceAtTick(), liquidity, true);
        }
    }

    /// @dev Computes the cumulative amount of token0 in the rounded ticks [roundedTick, tickUpper).
    function cumulativeAmount0(
        int24 roundedTick,
        uint256 totalLiquidity,
        int24 tickSpacing,
        int24 tickLower,
        int24 tickUpper,
        bool isCarpet
    ) internal pure returns (uint256 amount0) {
        if (roundedTick >= tickUpper || tickLower >= tickUpper) {
            // cumulativeAmount0DensityX96 is just 0
            return 0;
        } else if (roundedTick < tickLower) {
            roundedTick = tickLower;
        }

        uint24 length = uint24((tickUpper - tickLower) / tickSpacing);
        uint160 sqrtRatioTickUpper = tickUpper.getSqrtPriceAtTick();
        amount0 = isCarpet
            ? SqrtPriceMath.getAmount0Delta(
                roundedTick.getSqrtPriceAtTick(), sqrtRatioTickUpper, totalLiquidity.divUp(length), true
            )
            : totalLiquidity.fullMulX96Up(
                SqrtPriceMath.getAmount0Delta(roundedTick.getSqrtPriceAtTick(), sqrtRatioTickUpper, Q96.divUp(length), true)
            );
    }

    /// @dev Computes the cumulative amount of token1 in the rounded ticks [tickLower, roundedTick].
    function cumulativeAmount1(
        int24 roundedTick,
        uint256 totalLiquidity,
        int24 tickSpacing,
        int24 tickLower,
        int24 tickUpper,
        bool isCarpet
    ) internal pure returns (uint256 amount1) {
        if (roundedTick < tickLower || tickLower >= tickUpper) {
            // cumulativeAmount1DensityX96 is just 0
            return 0;
        } else if (roundedTick > tickUpper - tickSpacing) {
            roundedTick = tickUpper - tickSpacing;
        }

        uint24 length = uint24((tickUpper - tickLower) / tickSpacing);
        uint160 sqrtRatioTickLower = tickLower.getSqrtPriceAtTick();
        amount1 = isCarpet
            ? SqrtPriceMath.getAmount1Delta(
                sqrtRatioTickLower, (roundedTick + tickSpacing).getSqrtPriceAtTick(), totalLiquidity.divUp(length), true
            )
            : totalLiquidity.fullMulX96Up(
                SqrtPriceMath.getAmount1Delta(
                    sqrtRatioTickLower, (roundedTick + tickSpacing).getSqrtPriceAtTick(), Q96.divUp(length), true
                )
            );
    }

    /// @dev Given a cumulativeAmount0, computes the rounded tick whose cumulativeAmount0 is closest to the input. Range is [tickLower, tickUpper].
    ///      The returned tick will be the largest rounded tick whose cumulativeAmount0 is greater than or equal to the input.
    ///      In the case that the input exceeds the cumulativeAmount0 of all rounded ticks, the function will return (false, 0).
    function inverseCumulativeAmount0(
        uint256 cumulativeAmount0_,
        uint256 totalLiquidity,
        int24 tickSpacing,
        int24 tickLower,
        int24 tickUpper,
        bool isCarpet
    ) internal pure returns (bool success, int24 roundedTick) {
        // short circuit if cumulativeAmount0_ is 0
        if (cumulativeAmount0_ == 0) return (true, tickUpper);

        uint24 length = uint24((tickUpper - tickLower) / tickSpacing);

        uint160 sqrtRatioTickLower = tickLower.getSqrtPriceAtTick();
        uint160 sqrtRatioTickUpper = tickUpper.getSqrtPriceAtTick();
        uint160 sqrtPrice = isCarpet
            ? SqrtPriceMath.getNextSqrtPriceFromAmount0RoundingUp(
                sqrtRatioTickUpper, totalLiquidity.divUp(length), cumulativeAmount0_, true
            )
            : SqrtPriceMath.getNextSqrtPriceFromAmount0RoundingUp(
                sqrtRatioTickUpper, Q96.divUp(length), cumulativeAmount0_.fullMulDiv(Q96, totalLiquidity), true
            );
        if (sqrtPrice < sqrtRatioTickLower) {
            return (false, 0);
        }
        int24 tick = sqrtPrice.getTickAtSqrtPrice();
        success = true;
        roundedTick = roundTickSingle(tick, tickSpacing);

        // ensure roundedTick is within the valid range
        if (roundedTick < tickLower || roundedTick > tickUpper) {
            return (false, 0);
        }

        // ensure that roundedTick is not tickUpper when cumulativeAmount0_ is non-zero
        // this can happen if the corresponding cumulative density is too small
        if (roundedTick == tickUpper) {
            return (true, tickUpper - tickSpacing);
        }
    }

    /// @dev Given a cumulativeAmount1, computes the rounded tick whose cumulativeAmount1 is closest to the input. Range is [tickLower - tickSpacing, tickUpper - tickSpacing].
    ///      The returned tick will be the smallest rounded tick whose cumulativeAmount1 is greater than or equal to the input.
    ///      In the case that the input exceeds the cumulativeAmount1 of all rounded ticks, the function will return (false, 0).
    function inverseCumulativeAmount1(
        uint256 cumulativeAmount1_,
        uint256 totalLiquidity,
        int24 tickSpacing,
        int24 tickLower,
        int24 tickUpper,
        bool isCarpet
    ) internal pure returns (bool success, int24 roundedTick) {
        // short circuit if cumulativeAmount1_ is 0
        if (cumulativeAmount1_ == 0) return (true, tickLower - tickSpacing);

        uint24 length = uint24((tickUpper - tickLower) / tickSpacing);

        uint160 sqrtRatioTickLower = tickLower.getSqrtPriceAtTick();
        uint160 sqrtRatioTickUpper = tickUpper.getSqrtPriceAtTick();
        uint160 sqrtPrice = isCarpet
            ? SqrtPriceMath.getNextSqrtPriceFromAmount1RoundingDown(
                sqrtRatioTickLower, totalLiquidity.divUp(length), cumulativeAmount1_, true
            )
            : SqrtPriceMath.getNextSqrtPriceFromAmount1RoundingDown(
                sqrtRatioTickLower, Q96.divUp(length), cumulativeAmount1_.fullMulDiv(Q96, totalLiquidity), true
            );
        if (sqrtPrice > sqrtRatioTickUpper) {
            return (false, 0);
        }
        int24 tick = sqrtPrice.getTickAtSqrtPrice();
        // handle the edge case where cumulativeAmount1_ is exactly the
        // cumulative amount in [tickLower, tickUpper]
        if (tick == tickUpper) {
            tick -= 1;
        }
        success = true;
        roundedTick = roundTickSingle(tick, tickSpacing);

        // ensure roundedTick is within the valid range
        if (roundedTick < tickLower - tickSpacing || roundedTick >= tickUpper) {
            return (false, 0);
        }

        // ensure that roundedTick is not (tickLower - tickSpacing) when cumulativeAmount1_ is non-zero and rounding up
        // this can happen if the corresponding cumulative density is too small
        if (roundedTick == tickLower - tickSpacing) {
            return (true, tickLower);
        }
    }

    function liquidityDensityX96(int24 roundedTick, int24 tickSpacing, int24 tickLower, int24 tickUpper)
        internal
        pure
        returns (uint256)
    {
        if (roundedTick < tickLower || roundedTick >= tickUpper) {
            // roundedTick is outside of the distribution
            return 0;
        }
        uint256 length = uint24((tickUpper - tickLower) / tickSpacing);
        return Q96 / length;
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
        int24 tickLower,
        int24 tickUpper
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
                inverseCumulativeAmountInput, totalLiquidity, tickSpacing, tickLower, tickUpper, false
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
                ? cumulativeAmount0(roundedTick + tickSpacing, totalLiquidity, tickSpacing, tickLower, tickUpper, false)
                : cumulativeAmount0(roundedTick, totalLiquidity, tickSpacing, tickLower, tickUpper, false);

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
                ? cumulativeAmount1(roundedTick, totalLiquidity, tickSpacing, tickLower, tickUpper, false)
                : cumulativeAmount1(roundedTick - tickSpacing, totalLiquidity, tickSpacing, tickLower, tickUpper, false);

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
            swapLiquidity = (liquidityDensityX96(roundedTick, tickSpacing, tickLower, tickUpper) * totalLiquidity) >> 96;
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
                inverseCumulativeAmountInput, totalLiquidity, tickSpacing, tickLower, tickUpper, false
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
                ? cumulativeAmount1(roundedTick - tickSpacing, totalLiquidity, tickSpacing, tickLower, tickUpper, false)
                : cumulativeAmount1(roundedTick, totalLiquidity, tickSpacing, tickLower, tickUpper, false);

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
                ? cumulativeAmount0(roundedTick, totalLiquidity, tickSpacing, tickLower, tickUpper, false)
                : cumulativeAmount0(roundedTick + tickSpacing, totalLiquidity, tickSpacing, tickLower, tickUpper, false);

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
            swapLiquidity = (liquidityDensityX96(roundedTick, tickSpacing, tickLower, tickUpper) * totalLiquidity) >> 96;
        }
    }

    function isValidParams(int24 tickSpacing, uint24 twapSecondsAgo, bytes32 ldfParams, LDFType ldfType)
        internal
        pure
        returns (bool)
    {
        uint8 shiftMode = uint8(bytes1(ldfParams)); // use uint8 since we don't know if the value is in range yet
        (int24 minUsableTick, int24 maxUsableTick) =
            (TickMath.minUsableTick(tickSpacing), TickMath.maxUsableTick(tickSpacing));
        if (shiftMode != uint8(ShiftMode.STATIC)) {
            // Shifting
            // | shiftMode - 1 byte | offset - 3 bytes | length - 3 bytes |
            int24 offset = int24(uint24(bytes3(ldfParams << 8))); // offset (in rounded ticks) of tickLower from the twap tick
            int24 length = int24(uint24(bytes3(ldfParams << 32))); // length of the position in rounded ticks

            return twapSecondsAgo != 0 && ldfType == LDFType.DYNAMIC_AND_STATEFUL && length > 0
                && offset % tickSpacing == 0 && int256(length) * int256(tickSpacing) <= type(int24).max
                && int256(length) * int256(tickSpacing) <= (maxUsableTick - minUsableTick)
                && shiftMode <= uint8(type(ShiftMode).max);
        } else {
            // Static
            // | shiftMode - 1 byte | tickLower - 3 bytes | tickUpper - 3 bytes |
            int24 tickLower = int24(uint24(bytes3(ldfParams << 8)));
            int24 tickUpper = int24(uint24(bytes3(ldfParams << 32)));

            return ldfType == LDFType.STATIC && tickLower % tickSpacing == 0 && tickUpper % tickSpacing == 0
                && tickLower < tickUpper && tickLower >= minUsableTick && tickUpper <= maxUsableTick;
        }
    }

    /// @return tickLower The lower tick of the distribution
    /// @return tickUpper The upper tick of the distribution
    function decodeParams(int24 twapTick, int24 tickSpacing, bytes32 ldfParams)
        internal
        pure
        returns (int24 tickLower, int24 tickUpper, ShiftMode shiftMode)
    {
        shiftMode = ShiftMode(uint8(bytes1(ldfParams)));

        if (shiftMode != ShiftMode.STATIC) {
            // | shiftMode - 1 byte | offset - 3 bytes | length - 3 bytes |
            int24 offset = int24(uint24(bytes3(ldfParams << 8))); // offset of tickLower from the twap tick
            int24 length = int24(uint24(bytes3(ldfParams << 32))); // length of the position in rounded ticks
            tickLower = roundTickSingle(twapTick + offset, tickSpacing);
            tickUpper = tickLower + length * tickSpacing;

            // bound distribution to be within the range of usable ticks
            (int24 minUsableTick, int24 maxUsableTick) =
                (TickMath.minUsableTick(tickSpacing), TickMath.maxUsableTick(tickSpacing));
            if (tickLower < minUsableTick) {
                tickLower = minUsableTick;
                tickUpper = int24(FixedPointMathLib.min(tickLower + length * tickSpacing, maxUsableTick));
            } else if (tickUpper > maxUsableTick) {
                tickUpper = maxUsableTick;
                tickLower = int24(FixedPointMathLib.max(tickUpper - length * tickSpacing, minUsableTick));
            }
        } else {
            // | shiftMode - 1 byte | tickLower - 3 bytes | tickUpper - 3 bytes |
            tickLower = int24(uint24(bytes3(ldfParams << 8)));
            tickUpper = int24(uint24(bytes3(ldfParams << 32)));
        }
    }
}
