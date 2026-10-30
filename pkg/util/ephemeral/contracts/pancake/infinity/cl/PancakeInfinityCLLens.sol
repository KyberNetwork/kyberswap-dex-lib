// SPDX-License-Identifier: MIT
pragma solidity ^0.8.37;

import {CLPoolManager} from "../../../libraries/CLPoolManager.sol";
import {ChainBlock} from "../../../libraries/ChainBlock.sol";
import {PoolManagerStorage} from "../../../libraries/PoolManagerStorage.sol";

/// @title PancakeInfinityCLLens
/// @notice Deployless lens that returns a PancakeSwap Infinity concentrated-liquidity pool's price,
/// active liquidity and initialized ticks from one block, in a single `eth_call`, stopping short of
/// the gas limit rather than running out of it.
///
/// @dev Never deployed. Call with `to == null` and
/// `data == creationCode ++ abi.encode(clPoolManager, poolId, wordPosLower, wordPosUpper)`; the
/// constructor reverts with `abi.encode(getSnapshot's return values)`.
///
/// Infinity CL is a singleton of Uniswap V4's shape, and this is V4Lens for it with the
/// same three properties: state and ticks from one block, checkable by the ticks at or below `tick`
/// summing to `liquidity`; a scan that would exceed the call's gas stops at a word boundary and says
/// where to resume; the result is written in its ABI encoding as the scan goes. It reads
/// CLPoolManager's storage directly, many slots per call (see PoolManagerStorage). Unlike V4,
/// CLPoolManager keeps each pool's key, so the tick spacing is read from it (see CLPoolManager)
/// rather than taken on trust, and the result is V3Lens', spacing first.
contract PancakeInfinityCLLens {
    /// @notice An initialized tick and the liquidity bookkeeping a swap simulator needs.
    struct PopulatedTick {
        int24 tick;
        int128 liquidityNet;
        uint128 liquidityGross;
    }

    /// @notice `wordPosUpper` is below `wordPosLower`.
    error InvalidWordRange();

    /// @notice The pool key's tick spacing is not positive, which is how CLPoolManager answers for
    /// an id it does not know.
    error InvalidTickSpacing();

    /// @notice Slot of CLPoolManager's `mapping(PoolId => CLPool.State) pools`, found and checked
    /// against its getters on BSC: slot0, liquidity, bitmap words and ticks all agree.
    uint256 internal constant POOLS_SLOT = 4;

    /// @dev Bytes in the static head of the result, the last word being the tick array's offset.
    uint256 private constant HEAD_SIZE = 0x100;

    constructor(address clPoolManager, bytes32 poolId, int16 wordPosLower, int16 wordPosUpper) payable {
        bytes memory result = _snapshot(clPoolManager, poolId, wordPosLower, wordPosUpper);
        assembly ("memory-safe") {
            revert(add(result, 0x20), mload(result))
        }
    }

    /// @notice Reads the pool's state and every initialized tick in words
    /// `[wordPosLower, wordPosUpper]`, as far as the call's gas allows.
    /// @dev The same values the constructor reverts with, decoded from the same encoding. Never
    /// reached in a deployless call; it is here so the ABI names and describes the payload.
    /// @param clPoolManager The Infinity CLPoolManager holding the pool's state.
    /// @param poolId The pool id, `keccak256(abi.encode(poolKey))`.
    /// @param wordPosLower The first tick-bitmap word to scan, inclusive.
    /// @param wordPosUpper The last tick-bitmap word to scan, inclusive.
    /// @return tickSpacing The pool key's tick spacing.
    /// @return sqrtPriceX96 slot0's price.
    /// @return tick slot0's tick.
    /// @return liquidity The pool's in-range liquidity.
    /// @return blockNumber The block every value here was read at.
    /// @return nextWordPos The word to pass as `wordPosLower` to continue. Meaningless when
    /// `done` is true. Equal to the (clamped) `wordPosLower` when not even one word fit, which
    /// a caller must treat as too little gas rather than resume from, or it will loop.
    /// @return done True when every word in the range was read.
    /// @return ticks The initialized ticks read, ascending by tick index.
    function getSnapshot(address clPoolManager, bytes32 poolId, int16 wordPosLower, int16 wordPosUpper)
        public
        view
        returns (
            int24 tickSpacing,
            uint160 sqrtPriceX96,
            int24 tick,
            uint128 liquidity,
            uint256 blockNumber,
            int16 nextWordPos,
            bool done,
            PopulatedTick[] memory ticks
        )
    {
        (tickSpacing, sqrtPriceX96, tick, liquidity, blockNumber, nextWordPos, done, ticks) =
            abi.decode(
                _snapshot(clPoolManager, poolId, wordPosLower, wordPosUpper),
                (int24, uint160, int24, uint128, uint256, int16, bool, PopulatedTick[])
            );
    }

    /// @dev Builds getSnapshot's return values in their ABI encoding, as a `bytes` whose contents
    /// are the payload: an eight-word head, then the tick array's length and its entries, three
    /// words each. Entries are written straight after the head as they are found.
    function _snapshot(address clPoolManager, bytes32 poolId, int16 wordPosLower, int16 wordPosUpper)
        private
        view
        returns (bytes memory result)
    {
        require(wordPosLower <= wordPosUpper, InvalidWordRange());

        int24 tickSpacing = CLPoolManager.tickSpacing(clPoolManager, poolId);
        require(tickSpacing > 0, InvalidTickSpacing());
        uint256 stateSlot = PoolManagerStorage.stateSlot(poolId, POOLS_SLOT);
        (uint160 sqrtPriceX96, int24 tick, uint128 liquidity) = PoolManagerStorage.state(clPoolManager, stateSlot);

        uint256 buffers = PoolManagerStorage.allocate();
        uint256 entries;
        assembly ("memory-safe") {
            result := mload(0x40)
            // length word, head, array length; entries follow.
            entries := add(result, add(0x40, HEAD_SIZE))
            mstore(0x40, entries)
        }

        (uint256 count, bool done, int16 nextWordPos) =
            PoolManagerStorage.scan(clPoolManager, stateSlot, tickSpacing, wordPosLower, wordPosUpper, buffers, entries);

        uint256 blockNumber = ChainBlock.number();
        // Narrow values are cleaned on the way out: the ABI wants them sign- or zero-extended, and
        // Solidity does not promise that of a variable seen from assembly.
        assembly ("memory-safe") {
            let length := add(HEAD_SIZE, add(0x20, mul(count, 0x60)))
            mstore(result, length)
            let head := add(result, 0x20)
            mstore(head, signextend(2, tickSpacing))
            mstore(add(head, 0x20), and(sqrtPriceX96, 0xffffffffffffffffffffffffffffffffffffffff))
            mstore(add(head, 0x40), signextend(2, tick))
            mstore(add(head, 0x60), and(liquidity, 0xffffffffffffffffffffffffffffffff))
            mstore(add(head, 0x80), blockNumber)
            mstore(add(head, 0xa0), signextend(1, nextWordPos))
            mstore(add(head, 0xc0), iszero(iszero(done)))
            mstore(add(head, 0xe0), HEAD_SIZE)
            mstore(add(head, HEAD_SIZE), count)
            // Entries dropped by a stop sit past the end; release them with the rest.
            mstore(0x40, add(head, length))
        }
    }
}
