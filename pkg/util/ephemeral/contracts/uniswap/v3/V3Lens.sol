// SPDX-License-Identifier: MIT
pragma solidity ^0.8.37;

import {ChainBlock} from "../../libraries/ChainBlock.sol";
import {TickBitmap} from "../../libraries/TickBitmap.sol";
import {V3Pool} from "../../libraries/V3Pool.sol";
import {ReturnGas} from "../../libraries/ReturnGas.sol";

/// @title V3Lens
/// @notice Deployless lens that returns a Uniswap V3 pool's price, active liquidity and initialized
/// ticks from one block, in a single `eth_call`, stopping short of the gas limit rather than
/// running out of it.
///
/// @dev Never deployed. Call with `to == null` and
/// `data == creationCode ++ abi.encode(pool, wordPosLower, wordPosUpper)`; the constructor reverts
/// with `abi.encode(getSnapshot's return values)`, which is how an `eth_call` hands them back.
///
/// Three properties, each aimed at a tracker:
///
/// - The pool state comes back with the ticks. Read in calls of their own, slot0 and liquidity can
///   come from a different block than the ticks; here they cannot, and the reply carries what is
///   needed to check it: the ticks at or below `tick` must sum to `liquidity`, and over a full
///   range all of them to zero. That check is what catches a tick set missing an entry, which
///   nothing else in the reply would reveal.
///
/// - A scan that would exceed the call's gas stops at a word boundary instead. The ticks of the
///   word it was in are dropped, `done` is false and `nextWordPos` names that word, so the
///   caller resumes from it. A lens without that exit runs out of gas, the node returns nothing,
///   and the gas spent is lost along with every tick already read; the caller can only halve the
///   range and try again. Stopping on a word boundary keeps the cursor a plain word position, the
///   unit the caller already pages in, at the cost of re-reading at most one word.
///
/// - The result is written in its ABI encoding as the scan goes, so there is no bitmap array, no
///   index array and no re-encoding copy. Memory is one three-word entry per tick, and its
///   quadratic expansion cost is paid once rather than three times.
///
/// Pool reads take leading words only (see V3Pool), so this serves V3 forks without a per-DEX
/// branch.
contract V3Lens {
    /// @notice An initialized tick and the liquidity bookkeeping a swap simulator needs.
    struct PopulatedTick {
        int24 tick;
        int128 liquidityNet;
        uint128 liquidityGross;
    }

    /// @notice `wordPosUpper` is below `wordPosLower`.
    error InvalidWordRange();

    /// @notice Gas kept back for one more tick read and for returning the result.
    /// @dev A `ticks(int24)` getter loads the whole Tick.Info struct: four cold slots on Uniswap V3,
    /// six or seven on forks that append fields, so up to ~16k gas, of which the call may forward
    /// only 63/64. Then the block number, a few thousand on Arbitrum (see ChainBlock), and returning,
    /// a few hundred: the result is already encoded. 60k covers them twice over, and is nothing next
    /// to any node's eth_call cap. What copying the result out costs is kept on top (see ReturnGas).
    uint256 internal constant GAS_RESERVE = 60_000;

    /// @dev Bytes in the static head of the result, the last word being the tick array's offset.
    uint256 private constant HEAD_SIZE = 0x100;

    constructor(address pool, int16 wordPosLower, int16 wordPosUpper) payable {
        bytes memory result = _snapshot(pool, wordPosLower, wordPosUpper);
        assembly ("memory-safe") {
            revert(add(result, 0x20), mload(result))
        }
    }

    /// @notice Reads the pool's state and every initialized tick in words
    /// `[wordPosLower, wordPosUpper]`, as far as the call's gas allows.
    /// @dev The same values the constructor reverts with, decoded from the same encoding. Never
    /// reached in a deployless call; it is here so the ABI names and describes the payload.
    /// @param pool The Uniswap V3 pool to scan.
    /// @param wordPosLower The first tick-bitmap word to scan, inclusive.
    /// @param wordPosUpper The last tick-bitmap word to scan, inclusive.
    /// @return tickSpacing The pool's on-chain tick spacing.
    /// @return sqrtPriceX96 slot0's price.
    /// @return tick slot0's tick.
    /// @return liquidity The pool's in-range liquidity.
    /// @return blockNumber The block every value here was read at.
    /// @return nextWordPos The word to pass as `wordPosLower` to continue. Meaningless when
    /// `done` is true. Equal to the (clamped) `wordPosLower` when not even one word fit, which
    /// a caller must treat as too little gas rather than resume from, or it will loop.
    /// @return done True when every word in the range was read.
    /// @return ticks The initialized ticks read, ascending by tick index.
    function getSnapshot(address pool, int16 wordPosLower, int16 wordPosUpper)
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
                _snapshot(pool, wordPosLower, wordPosUpper),
                (int24, uint160, int24, uint128, uint256, int16, bool, PopulatedTick[])
            );
    }

    /// @dev Builds getSnapshot's return values in their ABI encoding, as a `bytes` whose contents
    /// are the payload: an eight-word head, then the tick array's length and its entries, three
    /// words each. Entries are written straight after the head as they are found, so nothing is
    /// copied afterwards.
    function _snapshot(address pool, int16 wordPosLower, int16 wordPosUpper)
        private
        view
        returns (bytes memory result)
    {
        require(wordPosLower <= wordPosUpper, InvalidWordRange());

        int24 tickSpacing = V3Pool.tickSpacing(pool);
        (uint160 sqrtPriceX96, int24 tick) = V3Pool.slot0(pool);
        uint128 liquidity = V3Pool.liquidity(pool);

        uint256 entries;
        assembly ("memory-safe") {
            result := mload(0x40)
            // length word, head, array length; entries follow.
            entries := add(result, add(0x40, HEAD_SIZE))
            mstore(0x40, entries)
        }

        uint256 count = 0;
        bool done = true;
        int16 nextWordPos = wordPosUpper;

        (int16 lower, int16 upper, bool empty) = TickBitmap.clampWordRange(wordPosLower, wordPosUpper, tickSpacing);
        // What copying the result out costs grows with the ticks written, and a word adds at most 256:
        // worked out for that many more, and again only after a word that added some, so the words
        // of a wide, sparse range and the ticks within a word are checked against it as it is.
        uint256 reserve = GAS_RESERVE + ReturnGas.copyCost(HEAD_SIZE + 0x20 + 256 * 0x60);
        for (int256 wordPos = lower; !empty && wordPos <= upper; ++wordPos) {
            uint256 wordStart = count;
            bool stopped = gasleft() < reserve;
            if (!stopped) {
                (count, stopped) = _readWord(pool, wordPos, tickSpacing, entries, count);
                if (count != wordStart) {
                    reserve = GAS_RESERVE + ReturnGas.copyCost(HEAD_SIZE + 0x20 + (count + 256) * 0x60);
                }
            }
            if (!stopped) continue;

            // Ticks are dropped a whole word at a time, so the cursor stays a word position.
            count = wordStart;
            done = false;
            // forge-lint: disable-next-line(unsafe-typecast): wordPos lies in [lower, upper], both int16.
            nextWordPos = int16(wordPos);
            break;
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
            mstore(add(head, 0xa0), signextend(1, nextWordPos))
            mstore(add(head, 0xc0), iszero(iszero(done)))
            mstore(add(head, 0xe0), HEAD_SIZE)
            mstore(add(head, HEAD_SIZE), count)
            // Entries dropped by a stop sit past the end; release them with the rest.
            mstore(0x40, add(head, length))
        }
    }

    /// @dev Appends every initialized tick of bitmap word `wordPos` to the entries at `entries`,
    /// starting at index `count`. Returns `stopped` when the gas left fell below what it keeps back before
    /// the word was finished; the entries it wrote are then the caller's to drop.
    function _readWord(address pool, int256 wordPos, int24 tickSpacing, uint256 entries, uint256 count)
        private
        view
        returns (uint256, bool stopped)
    {
        // forge-lint: disable-next-line(unsafe-typecast): the caller's loop keeps wordPos within int16.
        uint256 bitmap = V3Pool.bitmapWord(pool, int16(wordPos));
        if (bitmap == 0) return (count, false);
        int256 spacing = int256(tickSpacing);
        // A word adds at most 256 ticks: the copy's cost for all of them, worked out once.
        uint256 reserve = GAS_RESERVE + ReturnGas.copyCost(HEAD_SIZE + 0x20 + (count + 256) * 0x60);

        // Visits only the set bits, lowest first: most bits of a word are unset, and stepping through
        // every one up to the highest set cost ~49k on a word whose one tick was its last bit.
        for (uint256 rest = bitmap; rest != 0; rest &= rest - 1) {
            uint256 bitPos = TickBitmap.lowestBit(rest);

            // The edge words straddle the tick bounds, and a bit past them is no tick.
            // forge-lint: disable-next-line(unsafe-typecast): bitPos is below 256.
            int256 raw = (wordPos * 256 + int256(bitPos)) * spacing;
            if (raw < TickBitmap.MIN_TICK || raw > TickBitmap.MAX_TICK) continue;

            if (gasleft() < reserve) {
                return (count, true);
            }

            // forge-lint: disable-next-line(unsafe-typecast): raw was just checked against the tick bounds.
            (uint128 liquidityGross, int128 liquidityNet) = V3Pool.tickLiquidity(pool, int24(raw));
            assembly ("memory-safe") {
                let entry := add(entries, mul(count, 0x60))
                mstore(entry, raw)
                mstore(add(entry, 0x20), signextend(15, liquidityNet))
                mstore(add(entry, 0x40), and(liquidityGross, 0xffffffffffffffffffffffffffffffff))
                mstore(0x40, add(entry, 0x60))
            }
            ++count;
        }

        return (count, false);
    }
}
