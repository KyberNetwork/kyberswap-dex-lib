// SPDX-License-Identifier: BUSL-1.1
pragma solidity ^0.8.19;

import {SafeCastLib} from "solady/utils/SafeCastLib.sol";
import {FixedPointMathLib} from "solady/utils/FixedPointMathLib.sol";

import {TickMath} from "@uniswap/v4-core/src/libraries/TickMath.sol";

import "./ShiftMode.sol";
import "../lib/Math.sol";
import "../lib/ExpMath.sol";
import "../base/Constants.sol";
import {LDFType} from "../types/LDFType.sol";
import {FullMathX96} from "../lib/FullMathX96.sol";

library LibGeometricDistribution {
    using FullMathX96 for *;
    using TickMath for int24;
    using ExpMath for int256;
    using SafeCastLib for *;
    using FixedPointMathLib for *;

    uint256 internal constant ALPHA_BASE = 1e8; // alpha uses 8 decimals in ldfParams
    uint256 internal constant MIN_ALPHA = 1e3;
    uint256 internal constant MAX_ALPHA = 12e8;
    uint256 internal constant MIN_LIQUIDITY_DENSITY = Q96 / 1e3;

    /// @dev Queries the liquidity density and the cumulative amounts at the given rounded tick.
    /// @param roundedTick The rounded tick to query
    /// @param tickSpacing The spacing of the ticks
    /// @return liquidityDensityX96_ The liquidity density at the given rounded tick. Range is [0, 1]. Scaled by 2^96.
    /// @return cumulativeAmount0DensityX96 The cumulative amount of token0 in the rounded ticks [roundedTick + tickSpacing, minTick + length * tickSpacing)
    /// @return cumulativeAmount1DensityX96 The cumulative amount of token1 in the rounded ticks [minTick, roundedTick - tickSpacing]
    function query(int24 roundedTick, int24 tickSpacing, int24 minTick, int24 length, uint256 alphaX96)
        internal
        pure
        returns (uint256 liquidityDensityX96_, uint256 cumulativeAmount0DensityX96, uint256 cumulativeAmount1DensityX96)
    {
        // compute liquidityDensityX96
        liquidityDensityX96_ = liquidityDensityX96(roundedTick, tickSpacing, minTick, length, alphaX96);

        // x is the index of the roundedTick in the distribution
        // should be in the range [0, length)
        int24 x;
        if (roundedTick < minTick) {
            // roundedTick is to the left of the distribution
            // set x = -1
            x = -1;
        } else if (roundedTick >= minTick + length * tickSpacing) {
            // roundedTick is to the right of the distribution
            // set x = length
            x = length;
        } else {
            // roundedTick is in the distribution
            x = (roundedTick - minTick) / tickSpacing;
        }

        uint256 sqrtRatioTickSpacing = tickSpacing.getSqrtPriceAtTick();
        uint256 sqrtRatioNegTickSpacing = (-tickSpacing).getSqrtPriceAtTick();
        uint256 sqrtRatioMinTick = minTick.getSqrtPriceAtTick();
        uint256 sqrtRatioNegMinTick = (-minTick).getSqrtPriceAtTick();

        if (alphaX96 > Q96) {
            // alpha > 1
            // need to make sure that alpha^x doesn't overflow by using alpha^-1 during exponentiation
            uint256 alphaInvX96 = Q96.mulDiv(Q96, alphaX96);

            // compute cumulativeAmount0DensityX96 for the rounded tick to the right of the rounded current tick
            if (x >= length - 1) {
                // roundedTick is the last tick in the distribution
                // cumulativeAmount0DensityX96 is just 0
                cumulativeAmount0DensityX96 = 0;
            } else {
                int24 xPlus1 = x + 1; // the rounded tick to the right of the current rounded tick

                uint24 lengthMinusX = uint24(length - xPlus1);
                bool intermediateTermIsPositive = alphaInvX96 > sqrtRatioNegTickSpacing;
                uint256 numeratorTermLeft = alphaInvX96.rpow(lengthMinusX, Q96);
                uint256 numeratorTermRight = (-tickSpacing * int24(lengthMinusX)).getSqrtPriceAtTick();
                cumulativeAmount0DensityX96 = (Q96 - alphaInvX96).mulDivUp(
                    intermediateTermIsPositive
                        ? numeratorTermLeft - numeratorTermRight
                        : numeratorTermRight - numeratorTermLeft,
                    intermediateTermIsPositive
                        ? alphaInvX96 - sqrtRatioNegTickSpacing
                        : sqrtRatioNegTickSpacing - alphaInvX96
                ).mulDivUp((-tickSpacing * xPlus1).getSqrtPriceAtTick(), Q96 - alphaInvX96.rpow(uint24(length), Q96))
                    .mulDivUp(Q96 - sqrtRatioNegTickSpacing, sqrtRatioMinTick);
            }

            // compute cumulativeAmount1DensityX96 for the rounded tick to the left of the rounded current tick
            if (x <= 0) {
                // roundedTick is the first tick in the distribution
                // cumulativeAmount1DensityX96 is just 0
                cumulativeAmount1DensityX96 = 0;
            } else {
                uint256 alphaInvPowLengthX96 = alphaInvX96.rpow(uint24(length), Q96);

                uint256 baseX96 = alphaX96.mulDiv(sqrtRatioTickSpacing, Q96);
                uint256 numerator1 = alphaX96 - Q96;
                uint256 denominator1 = baseX96 - Q96;
                uint256 numerator2 = alphaInvX96.rpow(uint24(length - x), Q96).mulDivUp(
                    (x * tickSpacing).getSqrtPriceAtTick(), Q96
                ) - alphaInvPowLengthX96;
                uint256 denominator2 = Q96 - alphaInvPowLengthX96;
                cumulativeAmount1DensityX96 = Q96.mulDivUp(numerator2, denominator2).mulDivUp(numerator1, denominator1)
                    .mulDivUp(sqrtRatioTickSpacing - Q96, sqrtRatioNegMinTick);
            }
        } else {
            // alpha <= 1
            // will revert if alpha == 1 but that's ok

            // compute cumulativeAmount0DensityX96 for the rounded tick to the right of the rounded current tick
            if (x >= length - 1) {
                // roundedTick is the last tick in the distribution
                // cumulativeAmount0DensityX96 is just 0
                cumulativeAmount0DensityX96 = 0;
            } else {
                uint256 baseX96 = alphaX96.mulDiv(sqrtRatioNegTickSpacing, Q96);
                int24 xPlus1 = x + 1;
                uint256 alphaPowXX96 = alphaX96.rpow(uint24(xPlus1), Q96);
                uint256 alphaPowLengthX96 = alphaX96.rpow(uint24(length), Q96);
                uint256 numerator = (Q96 - alphaX96)
                    * (
                        alphaPowXX96.mulDivUp((-tickSpacing * xPlus1).getSqrtPriceAtTick(), Q96)
                            - alphaPowLengthX96.mulDivUp((-tickSpacing * length).getSqrtPriceAtTick(), Q96)
                    );
                uint256 denominator = (Q96 - alphaPowLengthX96) * (Q96 - baseX96);
                cumulativeAmount0DensityX96 =
                    (Q96 - sqrtRatioNegTickSpacing).fullMulDivUp(numerator, denominator).mulDivUp(Q96, sqrtRatioMinTick);
            }

            // compute cumulativeAmount1DensityX96 for the rounded tick to the left of the rounded current tick
            if (x <= 0) {
                // roundedTick is the first tick in the distribution
                // cumulativeAmount1DensityX96 is just 0
                cumulativeAmount1DensityX96 = 0;
            } else {
                uint256 baseX96 = alphaX96.mulDiv(sqrtRatioTickSpacing, Q96);
                uint256 numerator = dist(
                    Q96, alphaX96.rpow(uint24(x), Q96).mulDivUp((tickSpacing * x).getSqrtPriceAtTick(), Q96)
                ) * (Q96 - alphaX96);
                uint256 denominator = dist(Q96, baseX96) * (Q96 - alphaX96.rpow(uint24(length), Q96));
                cumulativeAmount1DensityX96 =
                    (sqrtRatioTickSpacing - Q96).fullMulDivUp(numerator, denominator).mulDivUp(sqrtRatioMinTick, Q96);
            }
        }
    }

    /// @dev Computes the cumulative amount of token0 in the rounded ticks [roundedTick, tickUpper).
    function cumulativeAmount0(
        int24 roundedTick,
        uint256 totalLiquidity,
        int24 tickSpacing,
        int24 minTick,
        int24 length,
        uint256 alphaX96
    ) internal pure returns (uint256 amount0) {
        uint256 cumulativeAmount0DensityX96;

        // x is the index of the roundedTick in the distribution
        // should be in the range [0, length)
        int24 x;
        if (roundedTick < minTick) {
            // roundedTick is to the left of the distribution
            x = 0;
        } else if (roundedTick >= minTick + length * tickSpacing) {
            // roundedTick is to the right of the distribution
            return 0;
        } else {
            // roundedTick is in the distribution
            x = (roundedTick - minTick) / tickSpacing;
        }

        uint256 sqrtRatioNegTickSpacing = (-tickSpacing).getSqrtPriceAtTick();
        uint256 sqrtRatioMinTick = minTick.getSqrtPriceAtTick();

        if (alphaX96 > Q96) {
            // alpha > 1
            // need to make sure that alpha^x doesn't overflow by using alpha^-1 during exponentiation
            uint256 alphaInvX96 = Q96.mulDiv(Q96, alphaX96);

            // compute cumulativeAmount0DensityX96 for the rounded tick to the right of the rounded current tick
            if (x >= length) {
                // roundedTick is to the right of the last tick in the distribution
                // amount0 is just 0
                amount0 = 0;
            } else {
                uint24 lengthMinusX = uint24(length - x);
                bool intermediateTermIsPositive = alphaInvX96 > sqrtRatioNegTickSpacing;
                uint256 numeratorTermLeft = alphaInvX96.rpow(lengthMinusX, Q96);
                uint256 numeratorTermRight = (-tickSpacing * int24(lengthMinusX)).getSqrtPriceAtTick();
                cumulativeAmount0DensityX96 = (Q96 - alphaInvX96).mulDivUp(
                    intermediateTermIsPositive
                        ? numeratorTermLeft - numeratorTermRight
                        : numeratorTermRight - numeratorTermLeft,
                    intermediateTermIsPositive
                        ? alphaInvX96 - sqrtRatioNegTickSpacing
                        : sqrtRatioNegTickSpacing - alphaInvX96
                ).mulDivUp((-tickSpacing * x).getSqrtPriceAtTick(), Q96 - alphaInvX96.rpow(uint24(length), Q96))
                    .mulDivUp(Q96 - sqrtRatioNegTickSpacing, sqrtRatioMinTick);
            }
        } else {
            // alpha <= 1
            // will revert if alpha == 1 but that's ok

            // compute cumulativeAmount0DensityX96 for the rounded tick to the right of the rounded current tick
            if (x >= length) {
                // roundedTick is to the right of the last tick in the distribution
                // amount0 is just 0
                amount0 = 0;
            } else {
                uint256 baseX96 = alphaX96.mulDiv(sqrtRatioNegTickSpacing, Q96);
                uint256 alphaPowXX96 = alphaX96.rpow(uint24(x), Q96);
                uint256 alphaPowLengthX96 = alphaX96.rpow(uint24(length), Q96);
                uint256 numerator = (Q96 - alphaX96)
                    * (
                        alphaPowXX96.mulDivUp((-tickSpacing * x).getSqrtPriceAtTick(), Q96)
                            - alphaPowLengthX96.mulDivUp((-tickSpacing * length).getSqrtPriceAtTick(), Q96)
                    );
                uint256 denominator = (Q96 - alphaPowLengthX96) * (Q96 - baseX96);

                cumulativeAmount0DensityX96 = (Q96 - sqrtRatioNegTickSpacing).fullMulDivUp(numerator, denominator)
                    .fullMulDivUp(Q96, sqrtRatioMinTick);
            }
        }

        amount0 = cumulativeAmount0DensityX96.fullMulX96Up(totalLiquidity);
    }

    /// @dev Computes the cumulative amount of token1 in the rounded ticks [tickLower, roundedTick].
    function cumulativeAmount1(
        int24 roundedTick,
        uint256 totalLiquidity,
        int24 tickSpacing,
        int24 minTick,
        int24 length,
        uint256 alphaX96
    ) internal pure returns (uint256 amount1) {
        uint256 cumulativeAmount1DensityX96;

        // x is the index of the roundedTick in the distribution
        // should be in the range [0, length)
        int24 x;
        if (roundedTick < minTick) {
            // roundedTick is to the left of the distribution
            return 0;
        } else if (roundedTick >= minTick + length * tickSpacing) {
            // roundedTick is to the right of the distribution
            // set x = length
            x = length - 1;
        } else {
            // roundedTick is in the distribution
            x = (roundedTick - minTick) / tickSpacing;
        }

        uint256 sqrtRatioTickSpacing = tickSpacing.getSqrtPriceAtTick();

        if (alphaX96 > Q96) {
            // alpha > 1
            // need to make sure that alpha^x doesn't overflow by using alpha^-1 during exponentiation
            uint256 alphaInvX96 = Q96.mulDiv(Q96, alphaX96);

            // compute cumulativeAmount1DensityX96 for the rounded tick to the left of the rounded current tick
            if (x < 0) {
                // roundedTick is to the left of the first tick in the distribution
                // cumulativeAmount1DensityX96 is just 0
                cumulativeAmount1DensityX96 = 0;
            } else {
                uint256 alphaInvPowLengthX96 = alphaInvX96.rpow(uint24(length), Q96);
                uint256 sqrtRatioNegMinTick = (-minTick).getSqrtPriceAtTick();

                uint256 baseX96 = alphaX96.mulDiv(sqrtRatioTickSpacing, Q96);
                uint256 numerator1 = alphaX96 - Q96;
                uint256 denominator1 = baseX96 - Q96;
                uint256 numerator2 = alphaInvX96.rpow(uint24(length - x - 1), Q96).mulDivUp(
                    ((x + 1) * tickSpacing).getSqrtPriceAtTick(), Q96
                ) - alphaInvPowLengthX96;
                uint256 denominator2 = Q96 - alphaInvPowLengthX96;
                cumulativeAmount1DensityX96 = Q96.mulDivUp(numerator2, denominator2).mulDivUp(numerator1, denominator1)
                    .mulDivUp(sqrtRatioTickSpacing - Q96, sqrtRatioNegMinTick);
            }
        } else {
            // alpha <= 1
            // will revert if alpha == 1 but that's ok

            // compute cumulativeAmount1DensityX96 for the rounded tick to the left of the rounded current tick
            if (x < 0) {
                // roundedTick is to the left of the first tick in the distribution
                // cumulativeAmount1DensityX96 is just 0
                cumulativeAmount1DensityX96 = 0;
            } else {
                uint256 sqrtRatioMinTick = minTick.getSqrtPriceAtTick();
                uint256 baseX96 = alphaX96.mulDiv(sqrtRatioTickSpacing, Q96);
                uint256 numerator = dist(
                    Q96, alphaX96.rpow(uint24(x + 1), Q96).mulDivUp((tickSpacing * (x + 1)).getSqrtPriceAtTick(), Q96)
                ) * (Q96 - alphaX96);
                uint256 denominator = dist(Q96, baseX96) * (Q96 - alphaX96.rpow(uint24(length), Q96));
                cumulativeAmount1DensityX96 =
                    (sqrtRatioTickSpacing - Q96).fullMulDivUp(numerator, denominator).mulDivUp(sqrtRatioMinTick, Q96);
            }
        }

        amount1 = cumulativeAmount1DensityX96.fullMulX96Up(totalLiquidity);
    }

    /// @dev Given a cumulativeAmount0, computes the rounded tick whose cumulativeAmount0 is closest to the input. Range is [tickLower, tickUpper].
    ///      The returned tick will be the largest rounded tick whose cumulativeAmount0 is greater than or equal to the input.
    ///      In the case that the input exceeds the cumulativeAmount0 of all rounded ticks, the function will return (false, 0).
    function inverseCumulativeAmount0(
        uint256 cumulativeAmount0_,
        uint256 totalLiquidity,
        int24 tickSpacing,
        int24 minTick,
        int24 length,
        uint256 alphaX96
    ) internal pure returns (bool success, int24 roundedTick) {
        if (cumulativeAmount0_ == 0) {
            // return right boundary of distribution
            return (true, minTick + length * tickSpacing);
        }

        uint256 cumulativeAmount0DensityX96 = cumulativeAmount0_.fullMulDivUp(Q96, totalLiquidity);
        uint256 sqrtRatioNegTickSpacing = (-tickSpacing).getSqrtPriceAtTick();
        uint256 sqrtRatioMinTick = minTick.getSqrtPriceAtTick();
        uint256 baseX96 = alphaX96.mulDiv(sqrtRatioNegTickSpacing, Q96);
        int256 lnBaseX96 = int256(baseX96).lnQ96(); // int256 conversion is safe since baseX96 < Q96

        int256 xWad;
        if (alphaX96 > Q96) {
            // alpha > 1
            // need to make sure that alpha^x doesn't overflow by using alpha^-1 during exponentiation
            uint256 alphaInvX96 = Q96.mulDiv(Q96, alphaX96);

            uint256 alphaInvPowLengthX96 = alphaInvX96.rpow(uint24(length), Q96);
            bool intermediateTermIsPositive = alphaInvX96 > sqrtRatioNegTickSpacing;
            uint256 tmp = cumulativeAmount0DensityX96.mulDivUp(sqrtRatioMinTick, Q96 - sqrtRatioNegTickSpacing).mulDivUp(
                Q96 - alphaInvPowLengthX96, Q96
            ).mulDivUp(
                intermediateTermIsPositive
                    ? alphaInvX96 - sqrtRatioNegTickSpacing
                    : sqrtRatioNegTickSpacing - alphaInvX96,
                Q96 - alphaInvX96
            );
            uint160 sqrtPriceNegTickSpacingMulLength = (-tickSpacing * length).getSqrtPriceAtTick();
            if (!intermediateTermIsPositive && sqrtPriceNegTickSpacingMulLength <= tmp) {
                // this usually happens when the maximum cumulativeAmount0 is very close to zero
                // check to see that cumulativeAmount0_ <= cumulativeAmount0(minTick + (length - 1) * tickSpacing)
                int24 result = minTick + (length - 1) * tickSpacing;
                if (
                    cumulativeAmount0_
                        <= cumulativeAmount0(result, totalLiquidity, tickSpacing, minTick, length, alphaX96)
                ) {
                    return (true, result);
                } else {
                    return (false, 0);
                }
            }
            tmp = intermediateTermIsPositive
                ? tmp + sqrtPriceNegTickSpacingMulLength
                : sqrtPriceNegTickSpacingMulLength - tmp;
            xWad = (tmp.toInt256().lnQ96RoundingUp() + int256(length) * (int256(alphaX96).lnQ96RoundingUp())).sDivWad(
                lnBaseX96
            );
        } else {
            uint256 denominator = (Q96 - alphaX96.rpow(uint24(length), Q96)) * (Q96 - baseX96);
            uint256 numerator = cumulativeAmount0DensityX96.mulDivUp(sqrtRatioMinTick, Q96).fullMulDivUp(
                denominator, Q96 - sqrtRatioNegTickSpacing
            );
            uint256 basePowXX96 = (numerator / (Q96 - alphaX96) + baseX96.rpow(uint24(length), Q96));
            xWad = basePowXX96.toInt256().lnQ96RoundingUp().sDivWad(lnBaseX96);
        }

        // early return if xWad is obviously too small
        // the result (the largest rounded tick whose cumulativeAmount0 is greater than or equal to the input) doesn't exist
        // thus return success = false
        if (xWad < 0) {
            // compare cumulativeAmount0_ with the max value of cumulativeAmount0()
            // due to precision errors sometimes xWad can be negative when cumulativeAmount0_
            // is close to the max value
            uint256 maxCumulativeAmount0 =
                cumulativeAmount0(minTick, totalLiquidity, tickSpacing, minTick, length, alphaX96);
            if (cumulativeAmount0_ > maxCumulativeAmount0) {
                return (false, 0);
            } else {
                // xWad shouldn't actually be negative
                // set it to 0
                xWad = 0;
            }
        }

        // get rounded tick from xWad
        success = true;
        roundedTick = xWadToRoundedTick(xWad, minTick, tickSpacing, false);

        // ensure roundedTick is within the valid range
        int24 maxTick = minTick + length * tickSpacing;
        if (roundedTick < minTick || roundedTick > maxTick) {
            return (false, 0);
        }

        // ensure that roundedTick is not minTick + length * tickSpacing when cumulativeAmount0_ is non-zero
        // this can happen if the corresponding cumulative density is too small
        if (roundedTick == maxTick) {
            return (true, maxTick - tickSpacing);
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
        int24 length,
        uint256 alphaX96
    ) internal pure returns (bool success, int24 roundedTick) {
        if (cumulativeAmount1_ == 0) {
            // return left boundary of distribution
            return (true, minTick - tickSpacing);
        }

        uint256 cumulativeAmount1DensityX96 = cumulativeAmount1_.fullMulDiv(Q96, totalLiquidity);
        uint256 sqrtRatioTickSpacing = tickSpacing.getSqrtPriceAtTick();
        uint256 baseX96 = alphaX96.mulDiv(sqrtRatioTickSpacing, Q96);
        int256 lnBaseX96 = int256(baseX96).lnQ96RoundingUp(); // int256 conversion is safe since baseX96 < Q96

        int256 xWad;
        if (alphaX96 > Q96) {
            // alpha > 1
            // need to make sure that alpha^x doesn't overflow by using alpha^-1 during exponentiation
            uint256 alphaInvX96 = Q96.mulDiv(Q96, alphaX96);
            uint256 alphaInvPowLengthX96 = alphaInvX96.rpow(uint24(length), Q96);
            uint256 sqrtRatioNegMinTick = (-minTick).getSqrtPriceAtTick();

            uint256 numerator1 = alphaX96 - Q96;
            uint256 denominator1 = baseX96 - Q96;
            uint256 denominator2 = Q96 - alphaInvPowLengthX96;
            uint256 numerator2 = cumulativeAmount1DensityX96.mulDiv(sqrtRatioNegMinTick, sqrtRatioTickSpacing - Q96)
                .mulDiv(denominator1, numerator1).mulDiv(denominator2, Q96);
            if (numerator2 + alphaInvPowLengthX96 == 0) return (false, 0);
            xWad = ((numerator2 + alphaInvPowLengthX96).toInt256().lnQ96() + int256(length) * int256(alphaX96).lnQ96())
                .sDivWad(lnBaseX96) - int256(WAD);
        } else {
            uint256 sqrtRatioMinTick = minTick.getSqrtPriceAtTick();

            uint256 denominator = dist(Q96, baseX96) * (Q96 - alphaX96.rpow(uint24(length), Q96));
            uint256 numerator = cumulativeAmount1DensityX96.fullMulDiv(Q96, sqrtRatioMinTick).fullMulDiv(
                denominator, sqrtRatioTickSpacing - Q96
            );
            if (Q96 > baseX96 && Q96 <= numerator / (Q96 - alphaX96)) {
                // this usually happens when the max cumulativeAmount1 is very close to zero
                // return minTick if cumulativeAmount1_ <= cumulativeAmount1(minTick)
                if (
                    cumulativeAmount1_
                        <= cumulativeAmount1(minTick, totalLiquidity, tickSpacing, minTick, length, alphaX96)
                ) {
                    return (true, minTick);
                } else {
                    return (false, 0);
                }
            }
            uint256 basePowXPlusOneX96 =
                Q96 > baseX96 ? Q96 - numerator / (Q96 - alphaX96) : Q96 + numerator / (Q96 - alphaX96);
            xWad = basePowXPlusOneX96.toInt256().lnQ96().sDivWad(lnBaseX96) - int256(WAD);
        }

        // early return if xWad is obviously too large
        // the result (the smallest rounded tick whose cumulativeAmount1 is greater than or equal to the input) doesn't exist
        // thus return success = false
        int256 xWadMax = (length - 1) * int256(WAD);
        if (xWad > xWadMax) {
            // compare cumulativeAmount1_ with the max value of cumulativeAmount1()
            // due to precision errors sometimes xWad can be greater than xWadMax when cumulativeAmount1_
            // is close to the max value
            uint256 maxCumulativeAmount1 = cumulativeAmount1(
                minTick + (length - 1) * tickSpacing, totalLiquidity, tickSpacing, minTick, length, alphaX96
            );
            if (cumulativeAmount1_ > maxCumulativeAmount1) {
                return (false, 0);
            } else {
                // xWad shouldn't actually be greater than xWadMax
                // set it to xWadMax
                xWad = xWadMax;
            }
        }

        // get rounded tick from xWad
        success = true;
        roundedTick = xWadToRoundedTick(xWad, minTick, tickSpacing, true);

        // ensure roundedTick is within the valid range
        if (roundedTick < minTick - tickSpacing || roundedTick >= minTick + length * tickSpacing) {
            return (false, 0);
        }

        // ensure that roundedTick is not (minTick - tickSpacing) when cumulativeAmount1_ is non-zero and rounding up
        // this can happen if the corresponding cumulative density is too small
        if (roundedTick == minTick - tickSpacing) {
            return (true, minTick);
        }
    }

    function liquidityDensityX96(int24 roundedTick, int24 tickSpacing, int24 minTick, int24 length, uint256 alphaX96)
        internal
        pure
        returns (uint256)
    {
        if (roundedTick < minTick || roundedTick >= minTick + length * tickSpacing) {
            // roundedTick is outside of the distribution
            return 0;
        }
        // x is the index of the roundedTick in the distribution
        // should be in the range [0, length)
        uint256 x = uint24((roundedTick - minTick) / tickSpacing);
        if (alphaX96 > Q96) {
            // alpha > 1
            // need to make sure that alpha^x doesn't overflow by using alpha^-1 during exponentiation
            uint256 alphaInvX96 = Q96.mulDiv(Q96, alphaX96);
            return alphaInvX96.rpow(uint24(length) - x, Q96).fullMulDiv(
                alphaX96 - Q96, Q96 - alphaInvX96.rpow(uint24(length), Q96)
            );
        } else {
            // alpha <= 1
            // will revert if alpha == 1 but that's ok
            return (Q96 - alphaX96).mulDiv(alphaX96.rpow(x, Q96), Q96 - alphaX96.rpow(uint24(length), Q96));
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
        int24 minTick,
        int24 length,
        uint256 alphaX96
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
                inverseCumulativeAmountInput, totalLiquidity, tickSpacing, minTick, length, alphaX96
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
                ? cumulativeAmount0(roundedTick + tickSpacing, totalLiquidity, tickSpacing, minTick, length, alphaX96)
                : cumulativeAmount0(roundedTick, totalLiquidity, tickSpacing, minTick, length, alphaX96);

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
                ? cumulativeAmount1(roundedTick, totalLiquidity, tickSpacing, minTick, length, alphaX96)
                : cumulativeAmount1(roundedTick - tickSpacing, totalLiquidity, tickSpacing, minTick, length, alphaX96);

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
            swapLiquidity =
                (liquidityDensityX96(roundedTick, tickSpacing, minTick, length, alphaX96) * totalLiquidity) >> 96;
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
                inverseCumulativeAmountInput, totalLiquidity, tickSpacing, minTick, length, alphaX96
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
                ? cumulativeAmount1(roundedTick - tickSpacing, totalLiquidity, tickSpacing, minTick, length, alphaX96)
                : cumulativeAmount1(roundedTick, totalLiquidity, tickSpacing, minTick, length, alphaX96);

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
                ? cumulativeAmount0(roundedTick, totalLiquidity, tickSpacing, minTick, length, alphaX96)
                : cumulativeAmount0(roundedTick + tickSpacing, totalLiquidity, tickSpacing, minTick, length, alphaX96);

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
            swapLiquidity =
                (liquidityDensityX96(roundedTick, tickSpacing, minTick, length, alphaX96) * totalLiquidity) >> 96;
        }
    }

    function isValidParams(int24 tickSpacing, uint24 twapSecondsAgo, bytes32 ldfParams, LDFType ldfType)
        internal
        pure
        returns (bool)
    {
        (int24 minUsableTick, int24 maxUsableTick) =
            (TickMath.minUsableTick(tickSpacing), TickMath.maxUsableTick(tickSpacing));

        // | shiftMode - 1 byte | minTickOrOffset - 3 bytes | length - 2 bytes | alpha - 4 bytes |
        uint8 shiftMode = uint8(bytes1(ldfParams));
        int24 minTickOrOffset = int24(uint24(bytes3(ldfParams << 8)));
        int24 length = int24(int16(uint16(bytes2(ldfParams << 32))));
        uint256 alpha = uint32(bytes4(ldfParams << 48));

        // ensure shiftMode is within the valid range
        if (shiftMode > uint8(type(ShiftMode).max)) {
            return false;
        }

        if (shiftMode != uint8(ShiftMode.STATIC)) {
            // LDF shifts
            // ensure twapSecondsAgo is non-zero and ldfType is DYNAMIC_AND_STATEFUL
            if (twapSecondsAgo == 0 || ldfType != LDFType.DYNAMIC_AND_STATEFUL) return false;
        }

        // ensure ldfType is STATIC if shiftMode is static
        if (shiftMode == uint8(ShiftMode.STATIC) && ldfType != LDFType.STATIC) {
            return false;
        }

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

        // ensure liquidity density is nowhere equal to zero
        // can check boundaries since function is monotonic
        uint256 minLiquidityDensityX96;
        if (alpha > ALPHA_BASE) {
            // monotonically increasing
            // check left boundary
            minLiquidityDensityX96 =
                liquidityDensityX96(minTickOrOffset, tickSpacing, minTickOrOffset, length, alphaX96);
        } else {
            // monotonically decreasing
            // check right boundary
            minLiquidityDensityX96 = liquidityDensityX96(
                minTickOrOffset + (length - 1) * tickSpacing, tickSpacing, minTickOrOffset, length, alphaX96
            );
        }
        if (minLiquidityDensityX96 < MIN_LIQUIDITY_DENSITY) {
            return false;
        }

        // if all conditions are met, return true
        return true;
    }

    /// @return minTick The minimum rounded tick of the distribution
    /// @return length The length of the distribution in number of rounded ticks (i.e. the number of ticks / tickSpacing)
    /// @return alphaX96 Parameter of the discrete laplace distribution, FixedPoint96
    function decodeParams(int24 twapTick, int24 tickSpacing, bytes32 ldfParams)
        internal
        pure
        returns (int24 minTick, int24 length, uint256 alphaX96, ShiftMode shiftMode)
    {
        // | shiftMode - 1 byte | minTickOrOffset - 3 bytes | length - 2 bytes | alpha - 4 bytes |
        shiftMode = ShiftMode(uint8(bytes1(ldfParams)));
        length = int24(int16(uint16(bytes2(ldfParams << 32))));
        uint256 alpha = uint32(bytes4(ldfParams << 48));
        alphaX96 = alpha.mulDiv(Q96, ALPHA_BASE);

        if (shiftMode != ShiftMode.STATIC) {
            // use rounded TWAP value + offset as minTick
            int24 offset = int24(uint24(bytes3(ldfParams << 8))); // the offset applied to the twap tick to get the minTick
            minTick = roundTickSingle(twapTick + offset, tickSpacing);

            // bound distribution to be within the range of usable ticks
            (int24 minUsableTick, int24 maxUsableTick) =
                (TickMath.minUsableTick(tickSpacing), TickMath.maxUsableTick(tickSpacing));
            if (minTick < minUsableTick) {
                minTick = minUsableTick;
            } else if (minTick > maxUsableTick - length * tickSpacing) {
                minTick = maxUsableTick - length * tickSpacing;
            }
        } else {
            // static minTick set in params
            minTick = int24(uint24(bytes3(ldfParams << 8))); // must be aligned to tickSpacing
        }
    }
}
