// SPDX-License-Identifier: BUSL-1.1
pragma solidity ^0.8.19;

import {PoolKey} from "@uniswap/v4-core/src/interfaces/IPoolManager.sol";

import "./ShiftMode.sol";
import {Guarded} from "../base/Guarded.sol";
import {LDFType} from "../types/LDFType.sol";
import {LibDoubleGeometricDistribution} from "./LibDoubleGeometricDistribution.sol";
import {ILiquidityDensityFunction} from "../interfaces/ILiquidityDensityFunction.sol";

/// @title DoubleGeometricDistribution
/// @author zefram.eth
/// @notice Juxtaposition of two geometric distributions, useful for stable pairs and bid-ask distributions.
/// Should not be used in production when TWAP is enabled, since the price can go out of the range.
contract DoubleGeometricDistribution is ILiquidityDensityFunction, Guarded {
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
        (
            int24 minTick,
            int24 length0,
            int24 length1,
            uint256 alpha0X96,
            uint256 alpha1X96,
            uint256 weight0,
            uint256 weight1,
            ShiftMode shiftMode
        ) = LibDoubleGeometricDistribution.decodeParams(twapTick, key.tickSpacing, ldfParams);
        (bool initialized, int24 lastMinTick) = _decodeState(ldfState);
        if (initialized) {
            minTick = enforceShiftMode(minTick, lastMinTick, shiftMode);
            shouldSurge = minTick != lastMinTick;
        }

        (liquidityDensityX96_, cumulativeAmount0DensityX96, cumulativeAmount1DensityX96) =
        LibDoubleGeometricDistribution.query(
            roundedTick, key.tickSpacing, minTick, length0, length1, alpha0X96, alpha1X96, weight0, weight1
        );
        newLdfState = _encodeState(minTick);
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
        (
            int24 minTick,
            int24 length0,
            int24 length1,
            uint256 alpha0X96,
            uint256 alpha1X96,
            uint256 weight0,
            uint256 weight1,
            ShiftMode shiftMode
        ) = LibDoubleGeometricDistribution.decodeParams(twapTick, key.tickSpacing, ldfParams);
        (bool initialized, int24 lastMinTick) = _decodeState(ldfState);
        if (initialized) {
            minTick = enforceShiftMode(minTick, lastMinTick, shiftMode);
        }

        return LibDoubleGeometricDistribution.computeSwap(
            inverseCumulativeAmountInput,
            totalLiquidity,
            zeroForOne,
            exactIn,
            key.tickSpacing,
            minTick,
            length0,
            length1,
            alpha0X96,
            alpha1X96,
            weight0,
            weight1
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
        (
            int24 minTick,
            int24 length0,
            int24 length1,
            uint256 alpha0X96,
            uint256 alpha1X96,
            uint256 weight0,
            uint256 weight1,
            ShiftMode shiftMode
        ) = LibDoubleGeometricDistribution.decodeParams(twapTick, key.tickSpacing, ldfParams);
        (bool initialized, int24 lastMinTick) = _decodeState(ldfState);
        if (initialized) {
            minTick = enforceShiftMode(minTick, lastMinTick, shiftMode);
        }

        return LibDoubleGeometricDistribution.cumulativeAmount0(
            roundedTick,
            totalLiquidity,
            key.tickSpacing,
            minTick,
            length0,
            length1,
            alpha0X96,
            alpha1X96,
            weight0,
            weight1
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
        (
            int24 minTick,
            int24 length0,
            int24 length1,
            uint256 alpha0X96,
            uint256 alpha1X96,
            uint256 weight0,
            uint256 weight1,
            ShiftMode shiftMode
        ) = LibDoubleGeometricDistribution.decodeParams(twapTick, key.tickSpacing, ldfParams);
        (bool initialized, int24 lastMinTick) = _decodeState(ldfState);
        if (initialized) {
            minTick = enforceShiftMode(minTick, lastMinTick, shiftMode);
        }

        return LibDoubleGeometricDistribution.cumulativeAmount1(
            roundedTick,
            totalLiquidity,
            key.tickSpacing,
            minTick,
            length0,
            length1,
            alpha0X96,
            alpha1X96,
            weight0,
            weight1
        );
    }

    /// @inheritdoc ILiquidityDensityFunction
    function isValidParams(PoolKey calldata key, uint24 twapSecondsAgo, bytes32 ldfParams, LDFType ldfType)
        external
        pure
        override
        returns (bool)
    {
        return LibDoubleGeometricDistribution.isValidParams(key.tickSpacing, twapSecondsAgo, ldfParams, ldfType);
    }

    function _decodeState(bytes32 ldfState) internal pure returns (bool initialized, int24 lastMinTick) {
        // | initialized - 1 byte | lastMinTick - 3 bytes |
        initialized = uint8(bytes1(ldfState)) == 1;
        lastMinTick = int24(uint24(bytes3(ldfState << 8)));
    }

    function _encodeState(int24 lastMinTick) internal pure returns (bytes32 ldfState) {
        // | initialized - 1 byte | lastMinTick - 3 bytes |
        ldfState = bytes32(bytes4(INITIALIZED_STATE + uint32(uint24(lastMinTick))));
    }
}
