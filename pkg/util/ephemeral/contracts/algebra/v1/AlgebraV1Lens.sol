// SPDX-License-Identifier: MIT
pragma solidity ^0.8.37;

import {AlgebraPool} from "../../libraries/AlgebraPool.sol";
import {ChainBlock} from "../../libraries/ChainBlock.sol";
import {TickBitmap} from "../../libraries/TickBitmap.sol";
import {ReturnGas} from "../../libraries/ReturnGas.sol";

/// @title AlgebraV1Lens
/// @notice Deployless lens that returns an Algebra V1 pool's price, active liquidity and initialized
/// ticks from one block, in a single `eth_call`, stopping short of the gas limit rather than
/// running out of it.
///
/// @dev Never deployed. Call with `to == null` and
/// `data == creationCode ++ abi.encode(pool, rawTickTable, wordPosLower, wordPosUpper)`; the
/// constructor reverts with `abi.encode(getSnapshot's return values)`.
///
/// V3Lens for an Algebra V1 pool, read through `globalState()` and `tickTable(int16)`, with
/// the same three properties and the same result. Algebra V1 deployments index the tick table two
/// ways, and nothing a pool exposes says which:
///
/// - by the tick divided by the spacing, as Uniswap V3 does (QuickSwap V3 on Polygon, Zyberswap V3
///   on Arbitrum, THENA Fusion on BSC);
/// - by the tick itself (Camelot V3 on Arbitrum, Zyberswap V3 on Optimism, SwapBased V3 on Base),
///   so `rawTickTable` and words `[-3466, 3465]` whatever the spacing.
///
/// The caller says which. A set bit always marks a tick holding liquidity, so a bit that reads as a
/// tick without any is a table read the wrong way, and the lens reverts TickTableMismatch() rather
/// than return a tick set that only looks plausible.
contract AlgebraV1Lens {
    /// @notice An initialized tick and the liquidity bookkeeping a swap simulator needs.
    /// @dev Algebra calls these liquidityTotal and liquidityDelta; the names here match the other
    /// tick lenses so one decoder serves them all.
    struct PopulatedTick {
        int24 tick;
        int128 liquidityNet;
        uint128 liquidityGross;
    }

    /// @notice `wordPosUpper` is below `wordPosLower`.
    error InvalidWordRange();

    /// @notice The pool's tick spacing is not positive, as on a pool never initialized.
    error InvalidTickSpacing();

    /// @notice A set bit of the tick table read as a tick without liquidity: the table is indexed
    /// the other way than `rawTickTable` says.
    error TickTableMismatch();

    /// @notice Gas kept back for one more tick read and for returning the result.
    /// @dev A `ticks(int24)` getter loads the whole Tick struct, five cold slots on Algebra V1, so
    /// ~13k gas, of which the call may forward only 63/64. 60k covers that and the return several
    /// times over, and is nothing next to any node's eth_call cap.
    uint256 internal constant GAS_RESERVE = 60_000;

    /// @dev Bytes in the static head of the result, the last word being the tick array's offset.
    uint256 private constant HEAD_SIZE = 0x100;

    constructor(address pool, bool rawTickTable, int16 wordPosLower, int16 wordPosUpper) payable {
        bytes memory result = _snapshot(pool, rawTickTable, wordPosLower, wordPosUpper);
        assembly ("memory-safe") {
            revert(add(result, 0x20), mload(result))
        }
    }

    /// @notice Reads the pool's state and every initialized tick in tick-table words
    /// `[wordPosLower, wordPosUpper]`, as far as the call's gas allows.
    /// @dev The same values the constructor reverts with, decoded from the same encoding. Never
    /// reached in a deployless call; it is here so the ABI names and describes the payload.
    /// @param pool The Algebra V1 pool to scan.
    /// @param rawTickTable True when the pool indexes its tick table by tick rather than by tick
    /// divided by the spacing.
    /// @param wordPosLower The first tick-table word to scan, inclusive.
    /// @param wordPosUpper The last tick-table word to scan, inclusive.
    /// @return tickSpacing The pool's on-chain tick spacing.
    /// @return sqrtPriceX96 globalState's price.
    /// @return tick globalState's tick.
    /// @return liquidity The pool's in-range liquidity.
    /// @return blockNumber The block every value here was read at.
    /// @return nextWordPos The word to pass as `wordPosLower` to continue. Meaningless when
    /// `done` is true. Equal to the (clamped) `wordPosLower` when not even one word fit, which
    /// a caller must treat as too little gas rather than resume from, or it will loop.
    /// @return done True when every word in the range was read.
    /// @return ticks The initialized ticks read, ascending by tick index.
    function getSnapshot(address pool, bool rawTickTable, int16 wordPosLower, int16 wordPosUpper)
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
                _snapshot(pool, rawTickTable, wordPosLower, wordPosUpper),
                (int24, uint160, int24, uint128, uint256, int16, bool, PopulatedTick[])
            );
    }

    /// @dev Builds getSnapshot's return values in their ABI encoding, as a `bytes` whose contents
    /// are the payload: an eight-word head, then the tick array's length and its entries, three
    /// words each. Entries are written straight after the head as they are found.
    function _snapshot(address pool, bool rawTickTable, int16 wordPosLower, int16 wordPosUpper)
        private
        view
        returns (bytes memory result)
    {
        require(wordPosLower <= wordPosUpper, InvalidWordRange());

        int24 tickSpacing = AlgebraPool.tickSpacing(pool);
        require(tickSpacing > 0, InvalidTickSpacing());
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
        bool done = true;
        int16 nextWordPos = wordPosUpper;

        // A table indexed by tick is one indexed by tick over a spacing of 1, word range included.
        int24 compression = rawTickTable ? int24(1) : tickSpacing;
        (int16 lower, int16 upper, bool empty) = TickBitmap.clampWordRange(wordPosLower, wordPosUpper, compression);
        // What copying the result out costs grows with the ticks written, and a word adds at most 256:
        // worked out for that many more, and again only after a word that added some, so the words
        // of a wide, sparse range and the ticks within a word are checked against it as it is.
        uint256 reserve = GAS_RESERVE + ReturnGas.copyCost(HEAD_SIZE + 0x20 + 256 * 0x60);
        for (int256 wordPos = lower; !empty && wordPos <= upper; ++wordPos) {
            uint256 wordStart = count;
            bool stopped = gasleft() < reserve;
            if (!stopped) {
                (count, stopped) = _readWord(pool, wordPos, compression, entries, count);
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

    /// @dev Appends every initialized tick of tick-table word `wordPos` to the entries at `entries`,
    /// starting at index `count`. A bit's tick is its position times `compression`, which is the
    /// spacing or 1. Returns `stopped` when the gas left fell below what it keeps back before the word was
    /// finished; the entries it wrote are then the caller's to drop.
    function _readWord(address pool, int256 wordPos, int24 compression, uint256 entries, uint256 count)
        private
        view
        returns (uint256, bool stopped)
    {
        // forge-lint: disable-next-line(unsafe-typecast): the caller's loop keeps wordPos within int16.
        uint256 bitmap = AlgebraPool.tickTableWord(pool, int16(wordPos));
        if (bitmap == 0) return (count, false);
        // A word adds at most 256 ticks: the copy's cost for all of them, worked out once.
        uint256 reserve = GAS_RESERVE + ReturnGas.copyCost(HEAD_SIZE + 0x20 + (count + 256) * 0x60);

        // Visits only the set bits, lowest first: most bits of a word are unset, and stepping through
        // every one up to the highest set cost ~49k on a word whose one tick was its last bit.
        for (uint256 rest = bitmap; rest != 0; rest &= rest - 1) {
            uint256 bitPos = TickBitmap.lowestBit(rest);

            // The edge words straddle the tick bounds, and a bit past them is no tick.
            // forge-lint: disable-next-line(unsafe-typecast): bitPos is below 256.
            int256 raw = (wordPos * 256 + int256(bitPos)) * int256(compression);
            if (raw < TickBitmap.MIN_TICK || raw > TickBitmap.MAX_TICK) continue;

            if (gasleft() < reserve) {
                return (count, true);
            }

            // forge-lint: disable-next-line(unsafe-typecast): raw was just checked against the tick bounds.
            (uint128 liquidityGross, int128 liquidityNet) = AlgebraPool.tickLiquidity(pool, int24(raw));
            // forge-lint: disable-next-line(require-revert-in-loop): a misread table must end the scan.
            require(liquidityGross != 0, TickTableMismatch());
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
