// SPDX-License-Identifier: BUSL-1.1
pragma solidity ^0.8.19;

import {Currency} from "@uniswap/v4-core/src/types/Currency.sol";
import {TickMath} from "@uniswap/v4-core/src/libraries/TickMath.sol";
import {PoolKey} from "@uniswap/v4-core/src/interfaces/IPoolManager.sol";
import {PoolId, PoolIdLibrary} from "@uniswap/v4-core/src/types/PoolId.sol";

import {Ownable} from "solady/auth/Ownable.sol";
import {SafeCastLib} from "solady/utils/SafeCastLib.sol";
import {FixedPointMathLib} from "solady/utils/FixedPointMathLib.sol";

import {IOracle} from "./IOracle.sol";
import {ShiftMode} from "../ShiftMode.sol";
import {WAD} from "../../base/Constants.sol";
import {Guarded} from "../../base/Guarded.sol";
import {LDFType} from "../../types/LDFType.sol";
import {roundTickSingle} from "../../lib/Math.sol";
import {LibOracleUniGeoDistribution} from "./LibOracleUniGeoDistribution.sol";
import {ILiquidityDensityFunction} from "../../interfaces/ILiquidityDensityFunction.sol";

/// @title OracleUniGeoDistribution
/// @author zefram.eth
/// @notice A Uniform distribution where one side is bounded by an oracle-determined rick. It is managed
/// by an owner who can switch the distribution to a geometric distribution or back to a uniform distribution.
/// The alpha of the geometric distribution can also be set by the owner.
contract OracleUniGeoDistribution is ILiquidityDensityFunction, Guarded, Ownable {
    using TickMath for *;
    using SafeCastLib for *;
    using FixedPointMathLib for *;
    using PoolIdLibrary for PoolKey;

    IOracle public immutable oracle;

    bool public immutable bondLtStablecoin;
    Currency public immutable bond;
    Currency public immutable stablecoin;

    struct LdfParamsOverride {
        bool overridden;
        bytes12 ldfParams;
    }

    mapping(PoolId => LdfParamsOverride) public ldfParamsOverride;

    event SetLdfParamsOverride(PoolId indexed id, bytes32 indexed ldfParams);

    error InvalidLdfParams();

    constructor(
        address hub_,
        address hook_,
        address quoter_,
        address initialOwner_,
        IOracle oracle_,
        Currency bond_,
        Currency stablecoin_
    ) Guarded(hub_, hook_, quoter_) {
        _initializeOwner(initialOwner_);
        oracle = oracle_;
        bond = bond_;
        stablecoin = stablecoin_;
        bondLtStablecoin = bond_ < stablecoin_;
    }

    /// @inheritdoc ILiquidityDensityFunction
    function query(
        PoolKey calldata key,
        int24 roundedTick,
        int24 twapTick,
        int24 spotPriceTick,
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
        // override ldf params if needed
        PoolId id = key.toId();
        LdfParamsOverride memory ldfParamsOverride_ = ldfParamsOverride[id];
        if (ldfParamsOverride_.overridden) {
            ldfParams = bytes32(ldfParamsOverride_.ldfParams);
        }

        // decode ldf params and check surge
        int24 oracleRick = floorPriceToRick(oracle.getFloorPrice(), key.tickSpacing);
        (
            int24 tickLower,
            int24 tickUpper,
            uint256 alphaX96,
            LibOracleUniGeoDistribution.DistributionType distributionType
        ) = LibOracleUniGeoDistribution.decodeParams({
            ldfParams: ldfParams,
            oracleTick: oracleRick,
            tickSpacing: key.tickSpacing
        });
        (bool initialized, int24 lastOracleRick, bytes32 lastLdfParams) = _decodeState(ldfState);
        if (initialized) {
            // should surge if param was updated or oracle rick has updated
            shouldSurge = lastLdfParams != ldfParams || oracleRick != lastOracleRick;
        }

        // compute results
        (liquidityDensityX96_, cumulativeAmount0DensityX96, cumulativeAmount1DensityX96) = LibOracleUniGeoDistribution
            .query({
            roundedTick: roundedTick,
            tickSpacing: key.tickSpacing,
            tickLower: tickLower,
            tickUpper: tickUpper,
            alphaX96: alphaX96,
            distributionType: distributionType
        });

        // update ldf state
        newLdfState = _encodeState(oracleRick, ldfParams);
    }

    /// @inheritdoc ILiquidityDensityFunction
    function computeSwap(
        PoolKey calldata key,
        uint256 inverseCumulativeAmountInput,
        uint256 totalLiquidity,
        bool zeroForOne,
        bool exactIn,
        int24, /* twapTick */
        int24, /* spotPriceTick */
        bytes32 ldfParams,
        bytes32 /* ldfState */
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
        // override ldf params if needed
        PoolId id = key.toId();
        LdfParamsOverride memory ldfParamsOverride_ = ldfParamsOverride[id];
        if (ldfParamsOverride_.overridden) {
            ldfParams = bytes32(ldfParamsOverride_.ldfParams);
        }

        // decode ldf params
        int24 oracleRick = floorPriceToRick(oracle.getFloorPrice(), key.tickSpacing);
        (
            int24 tickLower,
            int24 tickUpper,
            uint256 alphaX96,
            LibOracleUniGeoDistribution.DistributionType distributionType
        ) = LibOracleUniGeoDistribution.decodeParams({
            ldfParams: ldfParams,
            oracleTick: oracleRick,
            tickSpacing: key.tickSpacing
        });

        return LibOracleUniGeoDistribution.computeSwap({
            inverseCumulativeAmountInput: inverseCumulativeAmountInput,
            totalLiquidity: totalLiquidity,
            zeroForOne: zeroForOne,
            exactIn: exactIn,
            tickSpacing: key.tickSpacing,
            tickLower: tickLower,
            tickUpper: tickUpper,
            alphaX96: alphaX96,
            distributionType: distributionType
        });
    }

    /// @inheritdoc ILiquidityDensityFunction
    function cumulativeAmount0(
        PoolKey calldata key,
        int24 roundedTick,
        uint256 totalLiquidity,
        int24, /* twapTick */
        int24, /* spotPriceTick */
        bytes32 ldfParams,
        bytes32 /* ldfState */
    ) external view override guarded returns (uint256) {
        // override ldf params if needed
        PoolId id = key.toId();
        LdfParamsOverride memory ldfParamsOverride_ = ldfParamsOverride[id];
        if (ldfParamsOverride_.overridden) {
            ldfParams = bytes32(ldfParamsOverride_.ldfParams);
        }

        // decode ldf params
        int24 oracleRick = floorPriceToRick(oracle.getFloorPrice(), key.tickSpacing);
        (
            int24 tickLower,
            int24 tickUpper,
            uint256 alphaX96,
            LibOracleUniGeoDistribution.DistributionType distributionType
        ) = LibOracleUniGeoDistribution.decodeParams({
            ldfParams: ldfParams,
            oracleTick: oracleRick,
            tickSpacing: key.tickSpacing
        });

        return LibOracleUniGeoDistribution.cumulativeAmount0({
            roundedTick: roundedTick,
            totalLiquidity: totalLiquidity,
            tickSpacing: key.tickSpacing,
            tickLower: tickLower,
            tickUpper: tickUpper,
            alphaX96: alphaX96,
            distributionType: distributionType
        });
    }

    /// @inheritdoc ILiquidityDensityFunction
    function cumulativeAmount1(
        PoolKey calldata key,
        int24 roundedTick,
        uint256 totalLiquidity,
        int24, /* twapTick */
        int24, /* spotPriceTick */
        bytes32 ldfParams,
        bytes32 /* ldfState */
    ) external view override guarded returns (uint256) {
        // override ldf params if needed
        PoolId id = key.toId();
        LdfParamsOverride memory ldfParamsOverride_ = ldfParamsOverride[id];
        if (ldfParamsOverride_.overridden) {
            ldfParams = bytes32(ldfParamsOverride_.ldfParams);
        }

        // decode ldf params
        int24 oracleRick = floorPriceToRick(oracle.getFloorPrice(), key.tickSpacing);
        (
            int24 tickLower,
            int24 tickUpper,
            uint256 alphaX96,
            LibOracleUniGeoDistribution.DistributionType distributionType
        ) = LibOracleUniGeoDistribution.decodeParams({
            ldfParams: ldfParams,
            oracleTick: oracleRick,
            tickSpacing: key.tickSpacing
        });

        return LibOracleUniGeoDistribution.cumulativeAmount1({
            roundedTick: roundedTick,
            totalLiquidity: totalLiquidity,
            tickSpacing: key.tickSpacing,
            tickLower: tickLower,
            tickUpper: tickUpper,
            alphaX96: alphaX96,
            distributionType: distributionType
        });
    }

    /// @inheritdoc ILiquidityDensityFunction
    function isValidParams(PoolKey calldata key, uint24, /* twapSecondsAgo */ bytes32 ldfParams, LDFType ldfType)
        public
        view
        override
        returns (bool)
    {
        // only allow the bond-stablecoin pairing
        (Currency currency0, Currency currency1) = bond < stablecoin ? (bond, stablecoin) : (stablecoin, bond);

        return LibOracleUniGeoDistribution.isValidParams(
            key.tickSpacing, ldfParams, floorPriceToRick(oracle.getFloorPrice(), key.tickSpacing), ldfType
        ) && key.currency0 == currency0 && key.currency1 == currency1;
    }

    /// @notice Sets the ldf params for the given pool. Only callable by the owner.
    /// @param key The PoolKey of the Uniswap v4 pool
    /// @param distributionType The distribution type, either UNIFORM or GEOMETRIC
    /// @param oracleIsTickLower Whether the oracle tick is used to compute the lower bound of the distribution (or the upper bound)
    /// @param oracleTickOffset The offset applied to the oracle tick to compute the lower bound of the distribution (or the upper bound)
    /// @param nonOracleTick The boundary tick that's not computed from the oracle tick, AKA the fixed boundary.
    /// @param alpha The alpha of the geometric distribution, scaled by 1e8
    function setLdfParams(
        PoolKey calldata key,
        LibOracleUniGeoDistribution.DistributionType distributionType,
        bool oracleIsTickLower,
        int16 oracleTickOffset,
        int24 nonOracleTick,
        uint32 alpha
    ) public {
        // onlyOwner check is done in setLdfParams(PoolKey calldata key, bytes32 ldfParams)
        setLdfParams(key, encodeLdfParams(distributionType, oracleIsTickLower, oracleTickOffset, nonOracleTick, alpha));
    }

    /// @notice Sets the ldf params for the given pool. Only callable by the owner.
    /// @param key The PoolKey of the Uniswap v4 pool
    /// @param ldfParams The ldf params
    function setLdfParams(PoolKey calldata key, bytes32 ldfParams) public onlyOwner {
        // ensure new params are valid
        bool isValid = isValidParams(key, 0, ldfParams, LDFType.DYNAMIC_AND_STATEFUL);
        if (!isValid) {
            revert InvalidLdfParams();
        }

        // override ldf params
        PoolId id = key.toId();
        ldfParamsOverride[id] = LdfParamsOverride({overridden: true, ldfParams: bytes12(ldfParams)});
        emit SetLdfParamsOverride(id, ldfParams);
    }

    /// @notice Encodes the ldf params using the given parameters.
    /// @param distributionType The distribution type, either UNIFORM or GEOMETRIC
    /// @param oracleIsTickLower Whether the oracle tick is used to compute the lower bound of the distribution
    /// @param oracleTickOffset The offset applied to the oracle tick to compute the lower bound of the distribution
    /// @param nonOracleTick The non-oracle tick
    /// @param alpha The alpha of the geometric distribution, scaled by 1e8
    /// @return ldfParams The encoded ldf params
    function encodeLdfParams(
        LibOracleUniGeoDistribution.DistributionType distributionType,
        bool oracleIsTickLower,
        int16 oracleTickOffset,
        int24 nonOracleTick,
        uint32 alpha
    ) public pure returns (bytes32 ldfParams) {
        return bytes32(
            abi.encodePacked(
                ShiftMode.STATIC, distributionType, oracleIsTickLower, oracleTickOffset, nonOracleTick, alpha
            )
        );
    }

    /// @notice Computes the rick that the given floor price corresponds to.
    /// @param floorPriceWad The price of the bond token in stablecoin terms, scaled by WAD (1e18)
    /// @param tickSpacing The tick spacing of the Uniswap v4 pool
    /// @return rick The rounded tick that the given floor price corresponds to
    function floorPriceToRick(uint256 floorPriceWad, int24 tickSpacing) public view returns (int24 rick) {
        // convert floor price to sqrt price
        // assume bond is currency0, floor price's unit is (currency1 / currency0)
        // unscale by WAD then rescale by 2**(96*2), then take the sqrt to get sqrt(floorPrice) * 2**96
        uint160 sqrtPriceX96 = ((floorPriceWad << 192) / WAD).sqrt().toUint160();

        // convert sqrt price to rick
        rick = sqrtPriceX96.getTickAtSqrtPrice();
        rick = bondLtStablecoin ? rick : -rick; // need to invert the sqrt price if bond is currency1
        rick = roundTickSingle(rick, tickSpacing);
    }

    function _decodeState(bytes32 ldfState)
        internal
        pure
        returns (bool initialized, int24 lastOracleRick, bytes32 lastLdfParams)
    {
        // | initialized - 1 byte | lastOracleRick - 3 bytes | lastLdfParams - 12 bytes |
        initialized = uint8(bytes1(ldfState)) != 0;
        lastOracleRick = int24(uint24(bytes3(ldfState << 8)));
        lastLdfParams = ldfState << 32;
    }

    function _encodeState(int24 lastOracleRick, bytes32 lastLdfParams) internal pure returns (bytes32 ldfState) {
        // | initialized - 1 byte | lastOracleRick - 3 bytes | lastLdfParams - 12 bytes |
        ldfState = bytes32(abi.encodePacked(true, lastOracleRick, bytes12(lastLdfParams)));
    }
}
