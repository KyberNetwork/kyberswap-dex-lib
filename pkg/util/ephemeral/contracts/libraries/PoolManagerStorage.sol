// SPDX-License-Identifier: MIT
pragma solidity ^0.8.37;

import {TickBitmap} from "./TickBitmap.sol";
import {ReturnGas} from "./ReturnGas.sol";

/// @notice The batched storage read that Uniswap V4's PoolManager and PancakeSwap Infinity's
/// CLPoolManager both expose. Only the selector is taken from here.
interface IExtsload {
    function extsload(bytes32[] calldata slots) external view returns (bytes32[] memory values);
}

/// @title PoolManagerStorage
/// @notice Reads a concentrated-liquidity pool of a singleton manager straight from the manager's
/// storage, many slots per call, and scans its tick bitmap.
/// @dev Uniswap V4's PoolManager and PancakeSwap Infinity's CLPoolManager keep every pool's state in
/// a `mapping(PoolId => Pool.State)`, at slot 6 and slot 4 respectively, and lay the state out alike:
///
///     state       = keccak256(poolId . poolsSlot)
///     slot0       = state + 0     sqrtPriceX96 in bits 0..159, tick in 160..183
///     liquidity   = state + 3
///     ticks       = state + 4     mapping(int24 => Tick.Info), liquidityGross in bits 0..127 and
///                                 liquidityNet in 128..255 of its first word
///     tickBitmap  = state + 5     mapping(int16 => uint256)
///
/// Their getters read the same slots one call at a time: a bitmap word through Uniswap's StateView
/// is two calls deep, ~4.4k gas, and a Pancake tick returns its whole Tick.Info, three cold slots
/// for the one word read. `extsload(bytes32[])` reads a slot for little more than its cold SLOAD.
///
/// Reading raw storage leans on that layout, so the reads check what a different one would look
/// like: an uninitialized pool reads as a zero slot0 and is refused, and a set bitmap bit whose tick
/// holds no liquidity, which the managers never leave, reverts rather than being returned.
library PoolManagerStorage {
    /// @notice An extsload reply was not the ABI encoding of as many words as slots asked for.
    error UnexpectedManagerResponse();

    /// @notice The pool's slot0 is zero: the manager holds no initialized pool under this id.
    error UnknownPool();

    /// @notice A set bitmap bit's tick holds no liquidity, which the manager never leaves: the
    /// storage is not laid out as this library reads it.
    error UnexpectedTickState();

    uint256 private constant LIQUIDITY_OFFSET = 3;
    uint256 private constant TICKS_OFFSET = 4;
    uint256 private constant TICK_BITMAP_OFFSET = 5;

    /// @notice Bitmap words read per extsload call.
    uint256 internal constant WORDS_PER_READ = 64;

    /// @notice Gas kept back, past every read the scan makes, for what follows it: the block number,
    /// whose ArbSys read may burn its 20k cap and a cold access (see ChainBlock), and returning the
    /// result, which is already encoded.
    uint256 internal constant GAS_RESERVE = 60_000;

    /// @dev The most a lens's result holds before its ticks: its head and the array's length.
    uint256 private constant RESULT_HEAD = 0x120;

    /// @dev What one slot of an extsload call costs, set high: a read the gas does not cover runs
    /// the manager out of gas, and the call fails rather than stopping. The manager's loop pays a
    /// cold SLOAD (2,100) and a few hundred more per slot, and the lens a keccak and the bit loop.
    /// Some chains price a cold SLOAD higher, 5,460 on Polygon and 8,100 on Monad, so a scan plans
    /// by what its first read cost when that is more.
    uint256 private constant SLOT_COST = 3_500;

    /// @dev Words in a scan's first read, the read that tells what a slot costs on this chain. Enough
    /// that the call's fixed ~4k gas adds little to each slot, and few enough that what the read
    /// costs past SLOT_COST, ~30k at Monad's prices, stays within GAS_RESERVE.
    uint256 private constant FIRST_READ_WORDS = 8;

    /// @dev What one extsload call costs besides its slots: the call, its first access to the
    /// manager when cold (2,600), and copying the reply.
    uint256 private constant READ_COST = 10_000;

    /// @dev The buffers a scan reads into: the bitmap words of one read, a word's ticks, then an
    /// extsload call's calldata and reply for up to a word's 256 ticks.
    uint256 private constant TICK_BUFFER = WORDS_PER_READ << 5;
    uint256 private constant CALL_BUFFER = TICK_BUFFER + (256 << 5);
    uint256 private constant BUFFERS_SIZE = CALL_BUFFER + 0x44 + (256 << 5);

    /// @notice Allocates the buffers a scan reads into. A lens allocates them before its result, whose
    /// entries grow over whatever lies above them: a buffer placed past the entries would also make
    /// every read expand memory, quadratically in the size of the result.
    function allocate() internal pure returns (uint256 buffers) {
        uint256 size = BUFFERS_SIZE;
        assembly ("memory-safe") {
            buffers := mload(0x40)
            mstore(0x40, add(buffers, size))
        }
    }

    /// @notice Where the manager keeps the state of `poolId`.
    function stateSlot(bytes32 poolId, uint256 poolsSlot) internal pure returns (uint256 slot) {
        assembly ("memory-safe") {
            mstore(0, poolId)
            mstore(0x20, poolsSlot)
            slot := keccak256(0, 0x40)
        }
    }

    /// @notice The pool's price, tick and in-range liquidity, from one extsload call.
    function state(address manager, uint256 stateSlot_)
        internal
        view
        returns (uint160 sqrtPriceX96, int24 tick, uint128 liquidity)
    {
        uint256 buf;
        assembly ("memory-safe") {
            buf := mload(0x40)
            mstore(add(buf, 0x44), stateSlot_)
            mstore(add(buf, 0x64), add(stateSlot_, LIQUIDITY_OFFSET))
        }
        _extsload(manager, buf, 2);
        uint256 slot0;
        uint256 inRange;
        assembly ("memory-safe") {
            slot0 := mload(add(buf, 0x40))
            inRange := mload(add(buf, 0x60))
        }
        require(slot0 != 0, UnknownPool());
        // forge-lint: disable-start(unsafe-typecast): fields of a packed slot, at their bit offsets.
        sqrtPriceX96 = uint160(slot0);
        tick = int24(int256(slot0 >> 160));
        liquidity = uint128(inRange);
        // forge-lint: disable-end(unsafe-typecast)
    }

    /// @notice Appends every initialized tick in bitmap words `[wordPosLower, wordPosUpper]` to the
    /// entries at `entries`, three words each, as far as the call's gas allows, and moves the free
    /// memory pointer past them.
    /// @param buffers What allocate() returned, below the entries.
    /// @return count The entries written.
    /// @return done True when every word in the range was read.
    /// @return nextWordPos The word a call resuming the scan starts from; meaningless when done.
    function scan(
        address manager,
        uint256 stateSlot_,
        int24 tickSpacing,
        int16 wordPosLower,
        int16 wordPosUpper,
        uint256 buffers,
        uint256 entries
    ) internal view returns (uint256 count, bool done, int16 nextWordPos) {
        (int16 lower, int16 upper, bool empty) = TickBitmap.clampWordRange(wordPosLower, wordPosUpper, tickSpacing);
        if (empty) return (0, true, wordPosUpper);

        uint256 slotCost = SLOT_COST;
        for (int256 cursor = lower; cursor <= upper;) {
            uint256 left = gasleft();
            uint256 reserve = GAS_RESERVE + ReturnGas.copyCost(RESULT_HEAD + count * 0x60);
            if (left < reserve + READ_COST + slotCost) return (count, false, _wordPos(cursor));
            // forge-lint: disable-next-line(unsafe-typecast): cursor is at most upper, so this is positive.
            uint256 words = uint256(int256(upper) - cursor + 1);
            uint256 most = cursor == lower ? FIRST_READ_WORDS : WORDS_PER_READ;
            words = _min(words, most, (left - reserve - READ_COST) / slotCost);

            _readWords(manager, stateSlot_ + TICK_BITMAP_OFFSET, cursor, words, buffers);
            if (cursor == lower) slotCost = _slotCost(left - gasleft(), words);
            // Most words of a wide range are empty; passed over here rather than through a call each.
            for (uint256 i = _nextWord(buffers, 0, words); i < words; i = _nextWord(buffers, i + 1, words)) {
                uint256 bitmap;
                assembly ("memory-safe") {
                    bitmap := mload(add(buffers, shl(5, i)))
                }
                // forge-lint: disable-next-line(unsafe-typecast): i is below WORDS_PER_READ.
                int256 wordPos = cursor + int256(i);
                bool stopped;
                (count, stopped) =
                    _readWord(manager, stateSlot_, wordPos, bitmap, tickSpacing, buffers, entries, count, slotCost);
                // Ticks are dropped a whole word at a time, so the cursor stays a word position.
                if (stopped) return (count, false, _wordPos(wordPos));
            }
            // forge-lint: disable-next-line(unsafe-typecast): words is at most WORDS_PER_READ.
            cursor += int256(words);
        }
        return (count, true, wordPosUpper);
    }

    /// @dev Reads `words` bitmap words from `firstWordPos` to the start of `buffers`, in one
    /// extsload call.
    function _readWords(address manager, uint256 bitmapSlot, int256 firstWordPos, uint256 words, uint256 buffers)
        private
        view
    {
        uint256 buf = buffers + CALL_BUFFER;
        assembly ("memory-safe") {
            for { let i := 0 } lt(i, words) { i := add(i, 1) } {
                mstore(0, add(firstWordPos, i))
                mstore(0x20, bitmapSlot)
                mstore(add(add(buf, 0x44), shl(5, i)), keccak256(0, 0x40))
            }
        }
        _extsload(manager, buf, words);
        assembly ("memory-safe") {
            mcopy(buffers, add(buf, 0x40), shl(5, words))
        }
    }

    /// @dev Appends the initialized ticks of word `wordPos`, whose bitmap is `bitmap`, from one
    /// extsload call for all of their liquidity slots. Returns `stopped`, having written nothing,
    /// when the gas left does not cover the word: a slot for each set bit, an upper bound, and the
    /// memory its entries take, which costs more per tick the larger the result already is.
    ///
    /// Only the set bits are visited, lowest first: the bits of a word are mostly unset, and
    /// walking all 256 positions twice cost ~75k gas on a word whose one tick was its last bit.
    function _readWord(
        address manager,
        uint256 stateSlot_,
        int256 wordPos,
        uint256 bitmap,
        int24 tickSpacing,
        uint256 buffers,
        uint256 entries,
        uint256 count,
        uint256 slotCost
    ) private view returns (uint256, bool stopped) {
        uint256 bits = _popCount(bitmap);
        uint256 needed = GAS_RESERVE + READ_COST + bits * slotCost + _growth(bits * 0x60);
        if (gasleft() < needed + ReturnGas.copyCost(RESULT_HEAD + (count + bits) * 0x60)) return (count, true);

        uint256 ticksSlot = stateSlot_ + TICKS_OFFSET;
        uint256 ticks = buffers + TICK_BUFFER;
        uint256 buf = buffers + CALL_BUFFER;
        int256 first = wordPos * 256;
        int256 spacing = int256(tickSpacing);
        int256 minTick = TickBitmap.MIN_TICK;
        int256 maxTick = TickBitmap.MAX_TICK;
        uint256 slots = 0;
        assembly ("memory-safe") {
            for { let x := bitmap } x {} {
                let lowest := and(x, sub(0, x))
                x := xor(x, lowest)
                // The position of the one bit of `lowest`, by halving.
                let bitPos := shl(7, lt(0xffffffffffffffffffffffffffffffff, lowest))
                bitPos := or(bitPos, shl(6, lt(0xffffffffffffffff, shr(bitPos, lowest))))
                bitPos := or(bitPos, shl(5, lt(0xffffffff, shr(bitPos, lowest))))
                bitPos := or(bitPos, shl(4, lt(0xffff, shr(bitPos, lowest))))
                bitPos := or(bitPos, shl(3, lt(0xff, shr(bitPos, lowest))))
                bitPos := or(bitPos, shl(2, lt(0xf, shr(bitPos, lowest))))
                bitPos := or(bitPos, shl(1, lt(0x3, shr(bitPos, lowest))))
                bitPos := or(bitPos, lt(0x1, shr(bitPos, lowest)))
                let raw := mul(add(first, bitPos), spacing)
                // The edge words straddle the tick bounds, and a bit past them is no tick.
                if iszero(or(slt(raw, minTick), sgt(raw, maxTick))) {
                    mstore(add(ticks, shl(5, slots)), raw)
                    mstore(0, raw)
                    mstore(0x20, ticksSlot)
                    mstore(add(add(buf, 0x44), shl(5, slots)), keccak256(0, 0x40))
                    slots := add(slots, 1)
                }
            }
        }
        if (slots == 0) return (count, false);

        _extsload(manager, buf, slots);

        for (uint256 i = 0; i < slots; ++i) {
            int256 raw = 0;
            uint256 info = 0;
            assembly ("memory-safe") {
                raw := mload(add(ticks, shl(5, i)))
                info := mload(add(add(buf, 0x40), shl(5, i)))
            }
            // liquidityGross is the word's low half; a misread layout must end the scan.
            // forge-lint: disable-next-line(unsafe-typecast,require-revert-in-loop)
            require(uint128(info) != 0, UnexpectedTickState());
            assembly ("memory-safe") {
                let entry := add(entries, mul(count, 0x60))
                mstore(entry, raw)
                mstore(add(entry, 0x20), signextend(15, shr(128, info)))
                mstore(add(entry, 0x40), and(info, 0xffffffffffffffffffffffffffffffff))
                mstore(0x40, add(entry, 0x60))
            }
            ++count;
        }
        return (count, false);
    }

    /// @dev Calls `manager.extsload(slots)` with the `n` slots at `buf + 0x44`, and leaves the reply
    /// at `buf`, its values from `buf + 0x40`. A failed call's revert is passed up unchanged.
    function _extsload(address manager, uint256 buf, uint256 n) private view {
        bytes4 selector = IExtsload.extsload.selector;
        uint256 size;
        uint256 length = 0;
        assembly ("memory-safe") {
            mstore(buf, selector)
            mstore(add(buf, 0x04), 0x20)
            mstore(add(buf, 0x24), n)
            if iszero(staticcall(gas(), manager, buf, add(0x44, shl(5, n)), 0, 0)) {
                returndatacopy(buf, 0, returndatasize())
                // The loop is the caller's scan, and a refused read must end it.
                // forge-lint: disable-next-line(require-revert-in-loop)
                revert(buf, returndatasize())
            }
            size := returndatasize()
            if eq(size, add(0x40, shl(5, n))) {
                returndatacopy(buf, 0, size)
                length := mload(add(buf, 0x20))
            }
        }
        // forge-lint: disable-next-line(require-revert-in-loop): a short reply must end the scan.
        require(size == 0x40 + (n << 5) && length == n, UnexpectedManagerResponse());
    }

    // forge-lint: disable-next-line(unsafe-typecast): every word position here comes from int16 bounds.
    function _wordPos(int256 wordPos) private pure returns (int16) {
        // forge-lint: disable-next-line(unsafe-typecast)
        return int16(wordPos);
    }

    /// @dev What expanding memory by `size` bytes past the free memory pointer costs: 3 gas a word
    /// and a word squared over 512, from the yellow paper. Nothing above the pointer has been touched,
    /// the scan's buffers lying below the result.
    function _growth(uint256 size) private pure returns (uint256 cost) {
        assembly ("memory-safe") {
            let from := shr(5, mload(0x40))
            let to := add(from, shr(5, add(size, 31)))
            cost := sub(add(mul(3, to), shr(9, mul(to, to))), add(mul(3, from), shr(9, mul(from, from))))
        }
    }

    /// @dev The number of set bits, by the usual parallel sum; a full word's 256 overflows the byte
    /// the sum ends in, so it is answered first.
    function _popCount(uint256 x) private pure returns (uint256 c) {
        if (x == type(uint256).max) return 256;
        unchecked {
            x -= (x >> 1) & 0x5555555555555555555555555555555555555555555555555555555555555555;
            x = (x & 0x3333333333333333333333333333333333333333333333333333333333333333)
                + ((x >> 2) & 0x3333333333333333333333333333333333333333333333333333333333333333);
            x = (x + (x >> 4)) & 0x0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f;
            c = (x * 0x0101010101010101010101010101010101010101010101010101010101010101) >> 248;
        }
    }

    /// @dev The slot cost a scan plans its reads by: what its first read cost for each of `words`
    /// slots, with a quarter more as a margin, and never below SLOT_COST.
    function _slotCost(uint256 used, uint256 words) private pure returns (uint256) {
        uint256 measured = used / words;
        measured += measured / 4;
        return measured > SLOT_COST ? measured : SLOT_COST;
    }

    /// @dev The first bitmap word in `buffers`, from `i` on, with a bit set, or `words` if none is.
    /// A loop of its own, in assembly: a scan's own loop keeps too much on the stack to be as cheap
    /// for each of the thousands of empty words that a spacing-1 range has.
    function _nextWord(uint256 buffers, uint256 i, uint256 words) private pure returns (uint256 next) {
        assembly ("memory-safe") {
            next := i
            for {} and(lt(next, words), iszero(mload(add(buffers, shl(5, next))))) { next := add(next, 1) } {}
        }
    }

    function _min(uint256 a, uint256 b, uint256 c) private pure returns (uint256) {
        uint256 m = a < b ? a : b;
        return m < c ? m : c;
    }
}
