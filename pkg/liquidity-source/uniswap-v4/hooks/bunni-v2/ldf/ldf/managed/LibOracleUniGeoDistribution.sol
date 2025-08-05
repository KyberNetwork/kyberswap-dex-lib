// SPDX-License-Identifier: BUSL-1.1
pragma solidity ^0.8.19;

import {FixedPointMathLib} from "solady/utils/FixedPointMathLib.sol";

import {TickMath} from "@uniswap/v4-core/src/libraries/TickMath.sol";

import "../../lib/Math.sol";
import "../../base/Constants.sol";
import "../LibUniformDistribution.sol";
import "../LibGeometricDistribution.sol";
import {LDFType} from "../../types/LDFType.sol";

library LibOracleUniGeoDistribution {
    using FixedPointMathLib for uint256;

    enum DistributionType {
        UNIFORM,
        GEOMETRIC
    }

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
        int24 tickLower,
        int24 tickUpper,
        uint256 alphaX96,
        DistributionType distributionType
    )
        internal
        pure
        returns (uint256 liquidityDensityX96_, uint256 cumulativeAmount0DensityX96, uint256 cumulativeAmount1DensityX96)
    {
        // compute liquidityDensityX96
        liquidityDensityX96_ =
            liquidityDensityX96(roundedTick, tickSpacing, tickLower, tickUpper, alphaX96, distributionType);

        // compute cumulativeAmount0DensityX96
        cumulativeAmount0DensityX96 = cumulativeAmount0(
            roundedTick + tickSpacing, Q96, tickSpacing, tickLower, tickUpper, alphaX96, distributionType
        );

        // compute cumulativeAmount1DensityX96
        cumulativeAmount1DensityX96 = cumulativeAmount1(
            roundedTick - tickSpacing, Q96, tickSpacing, tickLower, tickUpper, alphaX96, distributionType
        );
    }

    /// @dev Computes the cumulative amount of token0 in the rounded ticks [roundedTick, tickUpper).
    function cumulativeAmount0(
        int24 roundedTick,
        uint256 totalLiquidity,
        int24 tickSpacing,
        int24 tickLower,
        int24 tickUpper,
        uint256 alphaX96,
        DistributionType distributionType
    ) internal pure returns (uint256 amount0) {
        if (distributionType == DistributionType.UNIFORM) {
            // Uniform
            return LibUniformDistribution.cumulativeAmount0(
                roundedTick, totalLiquidity, tickSpacing, tickLower, tickUpper, false
            );
        } else {
            // Geometric
            int24 length = (tickUpper - tickLower) / tickSpacing;
            return LibGeometricDistribution.cumulativeAmount0(
                roundedTick, totalLiquidity, tickSpacing, tickLower, length, alphaX96
            );
        }
    }

    /// @dev Computes the cumulative amount of token1 in the rounded ticks [tickLower, roundedTick].
    function cumulativeAmount1(
        int24 roundedTick,
        uint256 totalLiquidity,
        int24 tickSpacing,
        int24 tickLower,
        int24 tickUpper,
        uint256 alphaX96,
        DistributionType distributionType
    ) internal pure returns (uint256 amount1) {
        if (distributionType == DistributionType.UNIFORM) {
            // Uniform
            return LibUniformDistribution.cumulativeAmount1(
                roundedTick, totalLiquidity, tickSpacing, tickLower, tickUpper, false
            );
        } else {
            // Geometric
            int24 length = (tickUpper - tickLower) / tickSpacing;
            return LibGeometricDistribution.cumulativeAmount1(
                roundedTick, totalLiquidity, tickSpacing, tickLower, length, alphaX96
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
        int24 tickLower,
        int24 tickUpper,
        uint256 alphaX96,
        DistributionType distributionType
    ) internal pure returns (bool success, int24 roundedTick) {
        if (distributionType == DistributionType.UNIFORM) {
            // Uniform
            return LibUniformDistribution.inverseCumulativeAmount0(
                cumulativeAmount0_, totalLiquidity, tickSpacing, tickLower, tickUpper, false
            );
        } else {
            // Geometric
            int24 length = (tickUpper - tickLower) / tickSpacing;
            return LibGeometricDistribution.inverseCumulativeAmount0(
                cumulativeAmount0_, totalLiquidity, tickSpacing, tickLower, length, alphaX96
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
        int24 tickLower,
        int24 tickUpper,
        uint256 alphaX96,
        DistributionType distributionType
    ) internal pure returns (bool success, int24 roundedTick) {
        if (distributionType == DistributionType.UNIFORM) {
            // Uniform
            return LibUniformDistribution.inverseCumulativeAmount1(
                cumulativeAmount1_, totalLiquidity, tickSpacing, tickLower, tickUpper, false
            );
        } else {
            // Geometric
            int24 length = (tickUpper - tickLower) / tickSpacing;
            return LibGeometricDistribution.inverseCumulativeAmount1(
                cumulativeAmount1_, totalLiquidity, tickSpacing, tickLower, length, alphaX96
            );
        }
    }

    function liquidityDensityX96(
        int24 roundedTick,
        int24 tickSpacing,
        int24 tickLower,
        int24 tickUpper,
        uint256 alphaX96,
        DistributionType distributionType
    ) internal pure returns (uint256) {
        if (distributionType == DistributionType.UNIFORM) {
            // Uniform
            return LibUniformDistribution.liquidityDensityX96(roundedTick, tickSpacing, tickLower, tickUpper);
        } else {
            // Geometric
            int24 length = (tickUpper - tickLower) / tickSpacing;
            return LibGeometricDistribution.liquidityDensityX96(roundedTick, tickSpacing, tickLower, length, alphaX96);
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
        int24 tickLower,
        int24 tickUpper,
        uint256 alphaX96,
        DistributionType distributionType
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
                tickLower,
                tickUpper,
                alphaX96,
                distributionType
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
                    roundedTick + tickSpacing, totalLiquidity, tickSpacing, tickLower, tickUpper, alphaX96, distributionType
                )
                : cumulativeAmount0(
                    roundedTick, totalLiquidity, tickSpacing, tickLower, tickUpper, alphaX96, distributionType
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
                    roundedTick, totalLiquidity, tickSpacing, tickLower, tickUpper, alphaX96, distributionType
                )
                : cumulativeAmount1(
                    roundedTick - tickSpacing, totalLiquidity, tickSpacing, tickLower, tickUpper, alphaX96, distributionType
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
                liquidityDensityX96(roundedTick, tickSpacing, tickLower, tickUpper, alphaX96, distributionType)
                    * totalLiquidity
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
                tickLower,
                tickUpper,
                alphaX96,
                distributionType
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
                    roundedTick - tickSpacing, totalLiquidity, tickSpacing, tickLower, tickUpper, alphaX96, distributionType
                )
                : cumulativeAmount1(
                    roundedTick, totalLiquidity, tickSpacing, tickLower, tickUpper, alphaX96, distributionType
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
                    roundedTick, totalLiquidity, tickSpacing, tickLower, tickUpper, alphaX96, distributionType
                )
                : cumulativeAmount0(
                    roundedTick + tickSpacing, totalLiquidity, tickSpacing, tickLower, tickUpper, alphaX96, distributionType
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
                liquidityDensityX96(roundedTick, tickSpacing, tickLower, tickUpper, alphaX96, distributionType)
                    * totalLiquidity
            ) >> 96;
        }
    }

    function isValidParams(int24 tickSpacing, bytes32 ldfParams, int24 oracleTick, LDFType ldfType)
        internal
        pure
        returns (bool)
    {
        // decode params
        // | shiftMode - 1 byte | distributionType - 1 byte | oracleIsTickLower - 1 byte | oracleTickOffset - 2 bytes | nonOracleTick - 3 bytes | alpha - 4 bytes |
        uint8 shiftMode = uint8(bytes1(ldfParams));
        uint8 distributionType = uint8(bytes1(ldfParams << 8));
        bool oracleIsTickLower = uint8(bytes1(ldfParams << 16)) != 0;
        int24 oracleTickOffset = int24(int16(uint16(bytes2(ldfParams << 24))));
        int24 nonOracleTick = int24(uint24(bytes3(ldfParams << 40)));
        uint32 alpha = uint32(bytes4(ldfParams << 64));

        oracleTick += oracleTickOffset; // apply offset to oracle tick
        (int24 tickLower, int24 tickUpper) =
            oracleIsTickLower ? (oracleTick, nonOracleTick) : (nonOracleTick, oracleTick);
        if (tickLower >= tickUpper) {
            // ensure tickLower < tickUpper
            // use the non oracle tick as the bound
            // LDF needs to be at least one tickSpacing wide
            (tickLower, tickUpper) =
                oracleIsTickLower ? (tickUpper - tickSpacing, tickUpper) : (tickLower, tickLower + tickSpacing);
        }

        bytes32 geometricLdfParams =
            bytes32(abi.encodePacked(shiftMode, tickLower, int16((tickUpper - tickLower) / tickSpacing), alpha));

        (int24 minUsableTick, int24 maxUsableTick) =
            (TickMath.minUsableTick(tickSpacing), TickMath.maxUsableTick(tickSpacing));

        // validity conditions:
        // - geometric LDF params are valid
        // - uniform LDF params are valid
        // - shiftMode is static
        // - ldfType is DYNAMIC_AND_STATEFUL
        // - distributionType is valid
        // - oracleTickOffset is aligned to tickSpacing
        // - nonOracleTick is aligned to tickSpacing
        return LibGeometricDistribution.isValidParams(tickSpacing, 0, geometricLdfParams, LDFType.STATIC) // LDFType.STATIC is used since the geometric LDF doesn't shift
            && tickLower % tickSpacing == 0 && tickUpper % tickSpacing == 0 && tickLower >= minUsableTick
            && tickUpper <= maxUsableTick && shiftMode == uint8(ShiftMode.STATIC) && ldfType == LDFType.DYNAMIC_AND_STATEFUL
            && distributionType <= uint8(type(DistributionType).max) && oracleTickOffset % tickSpacing == 0
            && nonOracleTick % tickSpacing == 0;
    }

    /// @return tickLower The lower tick of the distribution
    /// @return tickUpper The upper tick of the distribution
    /// @return alphaX96 The alpha of the geometric distribution
    /// @return distributionType The distribution type, either UNIFORM or GEOMETRIC
    function decodeParams(bytes32 ldfParams, int24 oracleTick, int24 tickSpacing)
        internal
        pure
        returns (int24 tickLower, int24 tickUpper, uint256 alphaX96, DistributionType distributionType)
    {
        // decode params
        // | shiftMode - 1 byte | distributionType - 1 byte | oracleIsTickLower - 1 byte | oracleTickOffset - 2 bytes | nonOracleTick - 3 bytes | alpha - 4 bytes |
        distributionType = DistributionType(uint8(bytes1(ldfParams << 8)));
        bool oracleIsTickLower = uint8(bytes1(ldfParams << 16)) != 0;
        int24 oracleTickOffset = int24(int16(uint16(bytes2(ldfParams << 24))));
        int24 nonOracleTick = int24(uint24(bytes3(ldfParams << 40)));
        uint256 alpha = uint32(bytes4(ldfParams << 64));

        // compute results
        oracleTick += oracleTickOffset; // apply offset to oracle tick
        (tickLower, tickUpper) = oracleIsTickLower ? (oracleTick, nonOracleTick) : (nonOracleTick, oracleTick);

        // bound tickLower and tickUpper by minUsableTick and maxUsableTick
        // in case oracleTick is some crazy value
        (int24 minUsableTick, int24 maxUsableTick) =
            (TickMath.minUsableTick(tickSpacing), TickMath.maxUsableTick(tickSpacing));
        tickLower = int24(FixedPointMathLib.max(minUsableTick, tickLower));
        tickUpper = int24(FixedPointMathLib.min(maxUsableTick, tickUpper));

        if (tickLower >= tickUpper) {
            // ensure tickLower < tickUpper
            // use the non oracle tick as the bound
            // LDF needs to be at least one tickSpacing wide
            (tickLower, tickUpper) =
                oracleIsTickLower ? (tickUpper - tickSpacing, tickUpper) : (tickLower, tickLower + tickSpacing);
        }
        alphaX96 = alpha.mulDiv(Q96, ALPHA_BASE);
    }
}
