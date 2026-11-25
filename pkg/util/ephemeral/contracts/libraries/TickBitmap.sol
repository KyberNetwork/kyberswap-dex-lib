// SPDX-License-Identifier: MIT
pragma solidity ^0.8.37;

/// @title TickBitmap
/// @notice The part of a tick-bitmap scan that does not depend on where the bitmap is read from:
/// the tick bounds, and narrowing a word range to the words a tick spacing can populate.
library TickBitmap {
    /// @notice The Uniswap V3 tick bounds. A bit outside them cannot denote a real tick.
    int256 internal constant MIN_TICK = -887272;
    int256 internal constant MAX_TICK = 887272;

    /// @notice Intersects `[wordPosLower, wordPosUpper]` with the words `tickSpacing` can populate.
    /// @dev A word holds 256 compressed ticks and a compressed tick is `tick / tickSpacing` rounded
    /// toward negative infinity, so only `[floor(MIN_TICK / s) >> 8, floor(MAX_TICK / s) >> 8]` can
    /// hold a set bit. Every word outside that is still a cold read: the spacing-1 range asked of a
    /// spacing-10 pool is ~6,200 of them, about 39M gas spent learning nothing.
    /// @param tickSpacing Must be positive; the callers check before calling.
    /// @return lower The first word worth reading.
    /// @return upper The last word worth reading.
    /// @return empty True when the ranges do not overlap, so there is nothing to read at all.
    function clampWordRange(int16 wordPosLower, int16 wordPosUpper, int24 tickSpacing)
        internal
        pure
        returns (int16 lower, int16 upper, bool empty)
    {
        int256 spacing = int256(tickSpacing);
        int256 minWord = _floorDiv(MIN_TICK, spacing) >> 8;
        int256 maxWord = (MAX_TICK / spacing) >> 8;

        int256 lo = int256(wordPosLower) > minWord ? int256(wordPosLower) : minWord;
        int256 hi = int256(wordPosUpper) < maxWord ? int256(wordPosUpper) : maxWord;
        if (lo > hi) return (0, 0, true);

        // Both lie between the caller's int16 bounds, so they fit.
        // forge-lint: disable-next-line(unsafe-typecast)
        return (int16(lo), int16(hi), false);
    }

    /// @dev Division rounding toward negative infinity, as `compress` does on-chain; Solidity's `/`
    /// truncates toward zero. Only called with a positive divisor.
    function _floorDiv(int256 a, int256 b) private pure returns (int256 q) {
        q = a / b;
        if (a % b != 0 && a < 0) --q;
    }

    /// @notice The position of the lowest set bit of x, which is not 0, found by halving.
    function lowestBit(uint256 x) internal pure returns (uint256 bitPos) {
        assembly ("memory-safe") {
            let lowest := and(x, sub(0, x))
            bitPos := shl(7, lt(0xffffffffffffffffffffffffffffffff, lowest))
            bitPos := or(bitPos, shl(6, lt(0xffffffffffffffff, shr(bitPos, lowest))))
            bitPos := or(bitPos, shl(5, lt(0xffffffff, shr(bitPos, lowest))))
            bitPos := or(bitPos, shl(4, lt(0xffff, shr(bitPos, lowest))))
            bitPos := or(bitPos, shl(3, lt(0xff, shr(bitPos, lowest))))
            bitPos := or(bitPos, shl(2, lt(0xf, shr(bitPos, lowest))))
            bitPos := or(bitPos, shl(1, lt(0x3, shr(bitPos, lowest))))
            bitPos := or(bitPos, lt(0x1, shr(bitPos, lowest)))
        }
    }
}
