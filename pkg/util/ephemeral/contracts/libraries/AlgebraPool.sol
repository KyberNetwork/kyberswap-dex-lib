// SPDX-License-Identifier: MIT
pragma solidity ^0.8.37;

/// @notice The accessors a lens reads from an Algebra pool, V1 or Integral, declared with only the
/// leading return words it uses. Both keep `globalState()` starting with `(price, tick)` and
/// `ticks()` starting with `(liquidityTotal, liquidityDelta)`, Algebra's names for liquidityGross
/// and liquidityNet; Integral follows them with `(prevTick, nextTick)`, the links of its tick list.
/// Only the selectors are taken from here; the replies are read word by word.
interface IAlgebraPoolReads {
    function tickSpacing() external view returns (int24);
    function globalState() external view returns (uint160 price, int24 tick);
    function liquidity() external view returns (uint128);
    function tickTable(int16 rowNumber) external view returns (uint256);
    function ticks(int24 tick)
        external
        view
        returns (uint256 liquidityTotal, int128 liquidityDelta, int24 prevTick, int24 nextTick);
}

/// @title AlgebraPool
/// @notice The reads a lens makes of an Algebra pool.
/// @dev Each read copies only the words it needs, so the longer tuples of V1 (eight words of tick)
/// and Integral (six) cost nothing extra. A reply shorter than the words read means the address is
/// not an Algebra pool, and is refused rather than read past its end.
library AlgebraPool {
    /// @notice A reply came back shorter than the words read from it, i.e. the address is not an
    /// Algebra pool.
    error UnexpectedPoolResponse();

    // The narrowing casts below take the low bits of an ABI word, which is where the ABI puts a
    // narrow value; sign-extended or not, those bits are the value. liquidityTotal is a uint256 on
    // Integral, but the liquidity on a tick is bounded by the pool's uint128 liquidity per tick.
    // forge-lint: disable-start(unsafe-typecast)

    /// @dev `pool.tickSpacing()`, as the pool answers it: a pool not yet initialized may answer 0.
    function tickSpacing(address pool) internal view returns (int24 spacing) {
        (uint256 word,) = _read(pool, IAlgebraPoolReads.tickSpacing.selector, 0, false, 1);
        spacing = int24(int256(word));
    }

    /// @dev `pool.globalState()`'s first two words, `(price, tick)`.
    function globalState(address pool) internal view returns (uint160 sqrtPriceX96, int24 tick) {
        (uint256 price, uint256 current) = _read(pool, IAlgebraPoolReads.globalState.selector, 0, false, 2);
        sqrtPriceX96 = uint160(price);
        tick = int24(int256(current));
    }

    /// @dev `pool.liquidity()`.
    function liquidity(address pool) internal view returns (uint128 inRange) {
        (uint256 word,) = _read(pool, IAlgebraPoolReads.liquidity.selector, 0, false, 1);
        inRange = uint128(word);
    }

    /// @dev `pool.tickTable(rowNumber)`.
    function tickTableWord(address pool, int16 rowNumber) internal view returns (uint256 bitmap) {
        (bitmap,) = _read(pool, IAlgebraPoolReads.tickTable.selector, uint256(int256(rowNumber)), true, 1);
    }

    /// @dev `pool.ticks(tick)`'s first two words, `(liquidityTotal, liquidityDelta)`.
    function tickLiquidity(address pool, int24 tick)
        internal
        view
        returns (uint128 liquidityGross, int128 liquidityNet)
    {
        (uint256 gross, uint256 net) = _read(pool, IAlgebraPoolReads.ticks.selector, uint256(int256(tick)), true, 2);
        liquidityGross = uint128(gross);
        liquidityNet = int128(int256(net));
    }

    /// @dev `pool.ticks(tick)`'s first four words on Integral: the liquidity and both links.
    function tickNode(address pool, int24 tick)
        internal
        view
        returns (uint128 liquidityGross, int128 liquidityNet, int24 prevTick, int24 nextTick)
    {
        bytes4 selector = IAlgebraPoolReads.ticks.selector;
        uint256 size;
        uint256 gross;
        uint256 net;
        uint256 prev;
        uint256 next;
        assembly ("memory-safe") {
            // Four words of reply do not fit the scratch space, so take them at the free pointer
            // without moving it: nothing else is allocated before they are read.
            let ptr := mload(0x40)
            mstore(ptr, selector)
            mstore(add(ptr, 4), signextend(2, tick))
            if iszero(staticcall(gas(), pool, ptr, 0x24, ptr, 0x80)) {
                returndatacopy(ptr, 0, returndatasize())
                // The loop is the caller's walk, and a pool that refuses a read must end it.
                // forge-lint: disable-next-line(require-revert-in-loop)
                revert(ptr, returndatasize())
            }
            size := returndatasize()
            gross := mload(ptr)
            net := mload(add(ptr, 0x20))
            prev := mload(add(ptr, 0x40))
            next := mload(add(ptr, 0x60))
        }
        // As above: a short reply ends the walk.
        // forge-lint: disable-next-line(require-revert-in-loop)
        require(size >= 0x80, UnexpectedPoolResponse());
        liquidityGross = uint128(gross);
        liquidityNet = int128(int256(net));
        prevTick = int24(int256(prev));
        nextTick = int24(int256(next));
    }

    // forge-lint: disable-end(unsafe-typecast)

    /// @dev Calls `selector` on `pool`, with `arg` as its one argument when `hasArg`, and returns the
    /// first `words` (1 or 2) words of the reply. A failed call's revert is passed up unchanged.
    function _read(address pool, bytes4 selector, uint256 arg, bool hasArg, uint256 words)
        private
        view
        returns (uint256 first, uint256 second)
    {
        uint256 size;
        assembly ("memory-safe") {
            // Scratch space only: selector at 0, argument at 4, reply over both.
            mstore(0, selector)
            mstore(4, arg)
            if iszero(staticcall(gas(), pool, 0, add(4, mul(hasArg, 0x20)), 0, shl(5, words))) {
                let ptr := mload(0x40)
                returndatacopy(ptr, 0, returndatasize())
                // The loop is the caller's scan, and a pool that refuses a read must end it.
                // forge-lint: disable-next-line(require-revert-in-loop)
                revert(ptr, returndatasize())
            }
            size := returndatasize()
            first := mload(0)
            second := mload(0x20)
        }
        // As above: a short reply ends the scan.
        // forge-lint: disable-next-line(require-revert-in-loop)
        require(size >= words << 5, UnexpectedPoolResponse());
    }
}
