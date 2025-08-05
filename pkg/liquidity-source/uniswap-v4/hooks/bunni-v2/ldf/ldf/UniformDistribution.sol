// SPDX-License-Identifier: BUSL-1.1
pragma solidity ^0.8.19;

import {TickMath} from "@uniswap/v4-core/src/libraries/TickMath.sol";
import {PoolKey} from "@uniswap/v4-core/src/interfaces/IPoolManager.sol";

import {FixedPointMathLib} from "solady/utils/FixedPointMathLib.sol";

import "./ShiftMode.sol";
import {Guarded} from "../base/Guarded.sol";
import {LDFType} from "../types/LDFType.sol";
import {LibUniformDistribution} from "./LibUniformDistribution.sol";
import {ILiquidityDensityFunction} from "../interfaces/ILiquidityDensityFunction.sol";

/// @title UniformDistribution
/// @author zefram.eth
/// @notice Uniform distribution between two ticks, equivalent to a basic Uniswap v3 position.
/// Can shift using TWAP.
contract UniformDistribution is ILiquidityDensityFunction, Guarded {
    uint32 internal constant INITIALIZED_STATE = 1 << 24;

    constructor(address hub_, address hook_, address quoter_) Guarded(hub_, hook_, quoter_) {}

    /// @inheritdoc ILiquidityDensityFunction
    function query(
        PoolKey calldata key,
        int24 roundedTick,
        int24 twapTick,
        int24, /* spotPriceTick */
        bytes32 ldfParams,
        bytes32 ldfState
    )
        external
        view
        override
        guarded
        returns (
            uint256 liquidityDensityX96_,
            uint256 cumulativeAmount0DensityX96,
            uint256 cumulativeAmount1DensityX96,
            bytes32 newLdfState,
            bool shouldSurge
        )
    {
        (int24 tickLower, int24 tickUpper, ShiftMode shiftMode) =
            LibUniformDistribution.decodeParams(twapTick, key.tickSpacing, ldfParams);
        (bool initialized, int24 lastTickLower) = _decodeState(ldfState);
        if (initialized) {
            int24 tickLength = tickUpper - tickLower;
            (int24 minUsableTick, int24 maxUsableTick) =
                (TickMath.minUsableTick(key.tickSpacing), TickMath.maxUsableTick(key.tickSpacing));
            tickLower =
                int24(FixedPointMathLib.max(minUsableTick, enforceShiftMode(tickLower, lastTickLower, shiftMode)));
            tickUpper = int24(FixedPointMathLib.min(maxUsableTick, tickLower + tickLength));
            shouldSurge = tickLower != lastTickLower;
        }

        (liquidityDensityX96_, cumulativeAmount0DensityX96, cumulativeAmount1DensityX96) =
            LibUniformDistribution.query(roundedTick, key.tickSpacing, tickLower, tickUpper);
        newLdfState = _encodeState(tickLower);
    }

    /// @inheritdoc ILiquidityDensityFunction
    function computeSwap(
        PoolKey calldata key,
        uint256 inverseCumulativeAmountInput,
        uint256 totalLiquidity,
        bool zeroForOne,
        bool exactIn,
        int24 twapTick,
        int24, /* spotPriceTick */
        bytes32 ldfParams,
        bytes32 ldfState
    )
        external
        view
        override
        guarded
        returns (
            bool success,
            int24 roundedTick,
            uint256 cumulativeAmount0_,
            uint256 cumulativeAmount1_,
            uint256 swapLiquidity
        )
    {
        (int24 tickLower, int24 tickUpper, ShiftMode shiftMode) =
            LibUniformDistribution.decodeParams(twapTick, key.tickSpacing, ldfParams);
        (bool initialized, int24 lastTickLower) = _decodeState(ldfState);
        if (initialized) {
            int24 tickLength = tickUpper - tickLower;
            tickLower = enforceShiftMode(tickLower, lastTickLower, shiftMode);
            tickUpper = tickLower + tickLength;
        }

        return LibUniformDistribution.computeSwap(
            inverseCumulativeAmountInput, totalLiquidity, zeroForOne, exactIn, key.tickSpacing, tickLower, tickUpper
        );
    }

    /// @inheritdoc ILiquidityDensityFunction
    function cumulativeAmount0(
        PoolKey calldata key,
        int24 roundedTick,
        uint256 totalLiquidity,
        int24 twapTick,
        int24, /* spotPriceTick */
        bytes32 ldfParams,
        bytes32 ldfState
    ) external view override guarded returns (uint256) {
        (int24 tickLower, int24 tickUpper, ShiftMode shiftMode) =
            LibUniformDistribution.decodeParams(twapTick, key.tickSpacing, ldfParams);
        (bool initialized, int24 lastTickLower) = _decodeState(ldfState);
        if (initialized) {
            int24 tickLength = tickUpper - tickLower;
            tickLower = enforceShiftMode(tickLower, lastTickLower, shiftMode);
            tickUpper = tickLower + tickLength;
        }

        return LibUniformDistribution.cumulativeAmount0(
            roundedTick, totalLiquidity, key.tickSpacing, tickLower, tickUpper, false
        );
    }

    /// @inheritdoc ILiquidityDensityFunction
    function cumulativeAmount1(
        PoolKey calldata key,
        int24 roundedTick,
        uint256 totalLiquidity,
        int24 twapTick,
        int24, /* spotPriceTick */
        bytes32 ldfParams,
        bytes32 ldfState
    ) external view override guarded returns (uint256) {
        (int24 tickLower, int24 tickUpper, ShiftMode shiftMode) =
            LibUniformDistribution.decodeParams(twapTick, key.tickSpacing, ldfParams);
        (bool initialized, int24 lastTickLower) = _decodeState(ldfState);
        if (initialized) {
            int24 tickLength = tickUpper - tickLower;
            tickLower = enforceShiftMode(tickLower, lastTickLower, shiftMode);
            tickUpper = tickLower + tickLength;
        }

        return LibUniformDistribution.cumulativeAmount1(
            roundedTick, totalLiquidity, key.tickSpacing, tickLower, tickUpper, false
        );
    }

    /// @inheritdoc ILiquidityDensityFunction
    function isValidParams(PoolKey calldata key, uint24 twapSecondsAgo, bytes32 ldfParams, LDFType ldfType)
        external
        pure
        override
        returns (bool)
    {
        return LibUniformDistribution.isValidParams(key.tickSpacing, twapSecondsAgo, ldfParams, ldfType);
    }

    function _decodeState(bytes32 ldfState) internal pure returns (bool initialized, int24 lastTickLower) {
        // | initialized - 1 byte | lastTickLower - 3 bytes |
        initialized = uint8(bytes1(ldfState)) == 1;
        lastTickLower = int24(uint24(bytes3(ldfState << 8)));
    }

    function _encodeState(int24 lastTickLower) internal pure returns (bytes32 ldfState) {
        // | initialized - 1 byte | lastTickLower - 3 bytes |
        ldfState = bytes32(bytes4(INITIALIZED_STATE + uint32(uint24(lastTickLower))));
    }
}
