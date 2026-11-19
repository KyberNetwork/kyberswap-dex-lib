// SPDX-License-Identifier: MIT
pragma solidity ^0.8.37;

import {AlgebraPool} from "../../libraries/AlgebraPool.sol";
import {ChainBlock} from "../../libraries/ChainBlock.sol";
import {TickBitmap} from "../../libraries/TickBitmap.sol";
import {ReturnGas} from "../../libraries/ReturnGas.sol";

/// @title AlgebraIntegralLens
/// @notice Deployless lens that returns an Algebra Integral pool's price, active liquidity and
/// initialized ticks from one block, in a single `eth_call`, stopping short of the gas limit rather
/// than running out of it.
///
/// @dev Never deployed. Call with `to == null` and
/// `data == creationCode ++ abi.encode(pool, startTick)`; the constructor reverts with
/// `abi.encode(getSnapshot's return values)`.
///
/// Integral keeps its initialized ticks in an ascending doubly linked list through `ticks()`' third
/// and fourth words, between sentinels at MIN_TICK and MAX_TICK that link to themselves on their
/// outer side. Its tick table is indexed by tick, not by tick over the spacing, so a bitmap scan
/// would read every one of ~6,900 words; the list leads from one tick straight to the next. The walk
/// starts at MIN_TICK and follows `nextTick` to MAX_TICK, and pages by tick: a call that runs short
/// of gas returns the tick it did not get to read.
///
/// The sentinels hold liquidity only when a position reaches the tick bounds; they are returned
/// when they do, and the ticks of a whole walk sum to zero on liquidityNet like any tick set.
contract AlgebraIntegralLens {
    /// @notice An initialized tick and the liquidity bookkeeping a swap simulator needs.
    /// @dev Algebra calls these liquidityTotal and liquidityDelta; the names here match the other
    /// tick lenses so one decoder serves them all.
    struct PopulatedTick {
        int24 tick;
        int128 liquidityNet;
        uint128 liquidityGross;
    }

    /// @notice A tick read is not a node of the list: a link that does not point past it, or a
    /// start that is neither MIN_TICK nor a resumed cursor. Walked on, it would loop or end early
    /// with a tick set that looks complete.
    error BrokenTickLink();

    /// @notice Gas kept back for one more tick read and for returning the result.
    /// @dev A `ticks(int24)` getter loads the whole Tick struct, four cold slots, so ~10k gas, of
    /// which the call may forward only 63/64. 60k covers that and the return several times over.
    uint256 internal constant GAS_RESERVE = 60_000;

    /// @dev Bytes in the static head of the result, the last word being the tick array's offset.
    uint256 private constant HEAD_SIZE = 0x100;

    constructor(address pool, int24 startTick) payable {
        bytes memory result = _snapshot(pool, startTick);
        assembly ("memory-safe") {
            revert(add(result, 0x20), mload(result))
        }
    }

    /// @notice Reads the pool's state and walks its tick list from `startTick`, as far as the
    /// call's gas allows.
    /// @dev The same values the constructor reverts with, decoded from the same encoding. Never
    /// reached in a deployless call; it is here so the ABI names and describes the payload.
    /// @param pool The Algebra Integral pool to walk.
    /// @param startTick MIN_TICK (-887272) to walk from the start, or a previous call's `nextTick`.
    /// @return tickSpacing The pool's on-chain tick spacing.
    /// @return sqrtPriceX96 globalState's price.
    /// @return tick globalState's tick.
    /// @return liquidity The pool's in-range liquidity.
    /// @return blockNumber The block every value here was read at.
    /// @return nextTick The tick to pass as `startTick` to continue. Meaningless when `done` is true.
    /// Equal to `startTick` when not even one tick fit, which a caller must treat as too little gas
    /// rather than resume from, or it will loop.
    /// @return done True when the walk reached MAX_TICK.
    /// @return ticks The initialized ticks read, ascending by tick index.
    function getSnapshot(address pool, int24 startTick)
        public
        view
        returns (
            int24 tickSpacing,
            uint160 sqrtPriceX96,
            int24 tick,
            uint128 liquidity,
            uint256 blockNumber,
            int24 nextTick,
            bool done,
            PopulatedTick[] memory ticks
        )
    {
        (tickSpacing, sqrtPriceX96, tick, liquidity, blockNumber, nextTick, done, ticks) =
            abi.decode(
                _snapshot(pool, startTick), (int24, uint160, int24, uint128, uint256, int24, bool, PopulatedTick[])
            );
    }

    /// @dev Builds getSnapshot's return values in their ABI encoding, as a `bytes` whose contents
    /// are the payload: an eight-word head, then the tick array's length and its entries, three
    /// words each. Entries are written straight after the head as they are found.
    function _snapshot(address pool, int24 startTick) private view returns (bytes memory result) {
        int24 tickSpacing = AlgebraPool.tickSpacing(pool);
        (uint160 sqrtPriceX96, int24 tick) = AlgebraPool.globalState(pool);
        uint128 liquidity = AlgebraPool.liquidity(pool);

        uint256 entries;
        assembly ("memory-safe") {
            result := mload(0x40)
            // length word, head, array length; entries follow.
            entries := add(result, add(0x40, HEAD_SIZE))
            mstore(0x40, entries)
        }

        uint256 count = 0;
        bool done = false;
        int24 cursor = startTick;
        // What copying the result out costs grows with the entries written: worked out for a few
        // more than written, and again once they are.
        uint256 until = ReturnGas.ENTRIES_AHEAD;
        uint256 reserve = GAS_RESERVE + ReturnGas.copyCost(HEAD_SIZE + 0x20 + until * 0x60);
        while (gasleft() >= reserve) {
            (uint128 liquidityGross, int128 liquidityNet, int24 prevTick, int24 next) =
                AlgebraPool.tickNode(pool, cursor);
            // Every node links back below itself and on above itself, but for the sentinels, which
            // link to themselves on their outer side. A tick off the list reads as zero links.
            // forge-lint: disable-start(require-revert-in-loop): a broken list must end the walk.
            require(cursor == TickBitmap.MIN_TICK ? prevTick == cursor : prevTick < cursor, BrokenTickLink());
            require(cursor == TickBitmap.MAX_TICK ? next == cursor : next > cursor, BrokenTickLink());
            // forge-lint: disable-end(require-revert-in-loop)

            if (liquidityGross != 0) {
                assembly ("memory-safe") {
                    let entry := add(entries, mul(count, 0x60))
                    mstore(entry, signextend(2, cursor))
                    mstore(add(entry, 0x20), signextend(15, liquidityNet))
                    mstore(add(entry, 0x40), and(liquidityGross, 0xffffffffffffffffffffffffffffffff))
                    mstore(0x40, add(entry, 0x60))
                }
                if (++count == until) {
                    until += ReturnGas.ENTRIES_AHEAD;
                    reserve = GAS_RESERVE + ReturnGas.copyCost(HEAD_SIZE + 0x20 + until * 0x60);
                }
            }

            if (next == cursor) {
                done = true;
                break;
            }
            cursor = next;
        }

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
            mstore(add(head, 0xa0), signextend(2, cursor))
            mstore(add(head, 0xc0), iszero(iszero(done)))
            mstore(add(head, 0xe0), HEAD_SIZE)
            mstore(add(head, HEAD_SIZE), count)
            mstore(0x40, add(head, length))
        }
    }
}
