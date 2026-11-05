// SPDX-License-Identifier: MIT
pragma solidity ^0.8.37;

import {ChainBlock} from "../../libraries/ChainBlock.sol";
import {FluidDexV2} from "../../libraries/FluidDexV2.sol";
import {ReturnGas} from "../../libraries/ReturnGas.sol";
import {TickBitmap} from "../../libraries/TickBitmap.sol";

/// @title FluidDexV2Lens
/// @notice Deployless lens that returns a Fluid DEX v2 pool's price, active liquidity and initialized
/// ticks from one block, in a single `eth_call`, stopping short of the gas limit rather than
/// running out of it.
///
/// @dev Never deployed. Call with `to == null` and
/// `data == creationCode ++ abi.encode(dex, dexType, dexId, wordPosLower, wordPosUpper)`; the
/// constructor reverts with `abi.encode(getSnapshot's return values)`.
///
/// A Fluid DEX v2 pool is a Uniswap V3-shaped pool kept in the DEX's own storage, keyed by dex type
/// (3 for D3, 4 for D4) and dex id, and readable only through `readFromStorage` (see FluidDexV2).
/// Its tick bitmap is indexed by tick over the spacing, as Uniswap's, within tighter tick bounds
/// (-524287 to 524287, words [-2048, 2047] at spacing 1), so this is V3Lens over raw storage,
/// with the same three properties and the same result: state and ticks from one block, checkable by
/// the ticks at or below `tick` summing to `liquidity`; a scan that would exceed the call's gas
/// stops at a word boundary and says where to resume; the result is written in its ABI encoding as
/// the scan goes. The tick spacing is read from the pool's key.
///
/// Reading raw storage leans on the DEX's layout, so the lens checks what a changed layout would
/// break: the spacing must be one a pool can have, and every set bit must mark a tick holding
/// liquidity.
contract FluidDexV2Lens {
    /// @notice An initialized tick and the liquidity bookkeeping a swap simulator needs.
    struct PopulatedTick {
        int24 tick;
        int128 liquidityNet;
        uint128 liquidityGross;
    }

    /// @notice `wordPosUpper` is below `wordPosLower`.
    error InvalidWordRange();

    /// @notice The pool's key holds no tick spacing a pool can have: the pool does not exist, or the
    /// key is not where the lens reads it.
    error UnknownPool();

    /// @notice A set bit of the tick bitmap marks a tick without liquidity: the storage is not laid
    /// out the way the lens reads it.
    error UnexpectedTickState();

    /// @notice Gas kept back for one more tick read and for returning the result.
    /// @dev A tick is two `readFromStorage` calls through the DEX's proxy, each a cold slot and a
    /// delegatecall, so ~6k gas, of which a call may forward only 63/64. 60k covers that and the
    /// return several times over, and is nothing next to any node's eth_call cap.
    uint256 internal constant GAS_RESERVE = 60_000;

    /// @dev Bytes in the static head of the result, the last word being the tick array's offset.
    uint256 private constant HEAD_SIZE = 0x100;

    /// @dev The mapping bases of one pool's bitmap and ticks, computed once per call.
    struct Bases {
        bytes32 bitmap;
        bytes32 gross;
        bytes32 data;
    }

    constructor(address dex, uint256 dexType, bytes32 dexId, int16 wordPosLower, int16 wordPosUpper) payable {
        bytes memory result = _snapshot(dex, dexType, dexId, wordPosLower, wordPosUpper);
        assembly ("memory-safe") {
            revert(add(result, 0x20), mload(result))
        }
    }

    /// @notice Reads the pool's state and every initialized tick in bitmap words
    /// `[wordPosLower, wordPosUpper]`, as far as the call's gas allows.
    /// @dev The same values the constructor reverts with, decoded from the same encoding. Never
    /// reached in a deployless call; it is here so the ABI names and describes the payload.
    /// @param dex The Fluid DEX v2 contract.
    /// @param dexType The pool's dex type: 3 (D3) or 4 (D4).
    /// @param dexId The pool's id, `keccak256(abi.encode(dexKey))`.
    /// @param wordPosLower The first bitmap word to scan, inclusive.
    /// @param wordPosUpper The last bitmap word to scan, inclusive.
    /// @return tickSpacing The pool's tick spacing.
    /// @return sqrtPriceX96 The pool's price.
    /// @return tick The pool's tick.
    /// @return liquidity The pool's active liquidity.
    /// @return blockNumber The block every value here was read at.
    /// @return nextWordPos The word to pass as `wordPosLower` to continue. Meaningless when
    /// `done` is true. Equal to the (clamped) `wordPosLower` when not even one word fit, which
    /// a caller must treat as too little gas rather than resume from, or it will loop.
    /// @return done True when every word in the range was read.
    /// @return ticks The initialized ticks read, ascending by tick index.
    function getSnapshot(address dex, uint256 dexType, bytes32 dexId, int16 wordPosLower, int16 wordPosUpper)
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
                _snapshot(dex, dexType, dexId, wordPosLower, wordPosUpper),
                (int24, uint160, int24, uint128, uint256, int16, bool, PopulatedTick[])
            );
    }

    /// @dev Builds getSnapshot's return values in their ABI encoding, as a `bytes` whose contents
    /// are the payload: an eight-word head, then the tick array's length and its entries, three
    /// words each. Entries are written straight after the head as they are found.
    function _snapshot(address dex, uint256 dexType, bytes32 dexId, int16 wordPosLower, int16 wordPosUpper)
        private
        view
        returns (bytes memory result)
    {
        require(wordPosLower <= wordPosUpper, InvalidWordRange());

        (uint160 sqrtPriceX96, int24 tick, uint128 liquidity, int24 tickSpacing) = FluidDexV2.state(dex, dexType, dexId);
        require(tickSpacing >= FluidDexV2.MIN_TICK_SPACING && tickSpacing <= FluidDexV2.MAX_TICK_SPACING, UnknownPool());

        Bases memory bases = Bases({
            bitmap: FluidDexV2.poolSlot(FluidDexV2.TICK_BITMAP_SLOT, dexType, dexId),
            gross: FluidDexV2.poolSlot(FluidDexV2.TICK_LIQUIDITY_GROSS_SLOT, dexType, dexId),
            data: FluidDexV2.poolSlot(FluidDexV2.TICK_DATA_SLOT, dexType, dexId)
        });

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

        (int16 lower, int16 upper, bool empty) = FluidDexV2.clampWordRange(wordPosLower, wordPosUpper, tickSpacing);
        // What copying the result out costs grows with the ticks written, and a word adds at most 256:
        // worked out for that many more, and again only after a word that added some, so the words
        // of a wide, sparse range and the ticks within a word are checked against it as it is.
        uint256 reserve = GAS_RESERVE + ReturnGas.copyCost(HEAD_SIZE + 0x20 + 256 * 0x60);
        for (int256 wordPos = lower; !empty && wordPos <= upper; ++wordPos) {
            uint256 wordStart = count;
            bool stopped = gasleft() < reserve;
            if (!stopped) {
                (count, stopped) = _readWord(dex, bases, wordPos, tickSpacing, entries, count);
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
    function _readWord(
        address dex,
        Bases memory bases,
        int256 wordPos,
        int24 tickSpacing,
        uint256 entries,
        uint256 count
    ) private view returns (uint256, bool stopped) {
        uint256 bitmap = FluidDexV2.bitmapWord(dex, bases.bitmap, wordPos);
        if (bitmap == 0) return (count, false);
        // A word adds at most 256 ticks: the copy's cost for all of them, worked out once.
        uint256 reserve = GAS_RESERVE + ReturnGas.copyCost(HEAD_SIZE + 0x20 + (count + 256) * 0x60);
        int256 spacing = int256(tickSpacing);

        // Visits only the set bits, lowest first: most bits of a word are unset, and stepping through
        // every one up to the highest set cost ~49k on a word whose one tick was its last bit.
        for (uint256 rest = bitmap; rest != 0; rest &= rest - 1) {
            uint256 bitPos = TickBitmap.lowestBit(rest);

            // The edge words straddle the tick bounds, and a bit past them is no tick.
            // forge-lint: disable-next-line(unsafe-typecast): bitPos is below 256.
            int256 raw = (wordPos * 256 + int256(bitPos)) * spacing;
            if (raw < FluidDexV2.MIN_TICK || raw > FluidDexV2.MAX_TICK) continue;

            if (gasleft() < reserve) {
                return (count, true);
            }

            (uint128 liquidityGross, int128 liquidityNet) = FluidDexV2.tickLiquidity(dex, bases.gross, bases.data, raw);
            // forge-lint: disable-next-line(require-revert-in-loop): a misread layout must end the scan.
            require(liquidityGross != 0, UnexpectedTickState());
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
