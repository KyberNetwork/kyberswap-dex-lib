// SPDX-License-Identifier: MIT
pragma solidity ^0.8.37;

/// @notice The accessors a lens reads from a Uniswap V3-shaped pool, declared with only the leading
/// return words it uses. Forks append or reorder what follows (SlipStream inserts
/// stakedLiquidityNet third in ticks, PancakeSwap widens slot0's feeProtocol, Solidly replaces
/// slot0's oracle fields with a fee), but every fork checked keeps these prefixes. Only the
/// selectors are taken from here; the replies are read word by word, never ABI-decoded whole.
interface IUniswapV3PoolReads {
    function tickSpacing() external view returns (int24);
    function tickBitmap(int16 wordPosition) external view returns (uint256);
    function ticks(int24 tick) external view returns (uint128 liquidityGross, int128 liquidityNet);
    function slot0() external view returns (uint160 sqrtPriceX96, int24 tick);
    function liquidity() external view returns (uint128);
}

/// @title V3Pool
/// @notice The reads a lens makes of a Uniswap V3-shaped pool.
/// @dev Each read is a staticcall that copies at most two words of the reply, so a fork's longer
/// tuple costs nothing extra and needs no per-fork decoder. A reply shorter than the words read
/// means the address is not a V3-shaped pool, and is refused rather than read past its end.
library V3Pool {
    /// @notice A reply came back shorter than the words read from it, or with a non-positive tick
    /// spacing, i.e. the address is not a Uniswap V3-shaped pool.
    error UnexpectedPoolResponse();

    // The narrowing casts below take the low bits of an ABI word, which is where the ABI puts a
    // narrow value; sign-extended or not, those bits are the value.
    // forge-lint: disable-start(unsafe-typecast)

    /// @dev `pool.tickSpacing()`, required positive: word ranges are derived by dividing by it.
    function tickSpacing(address pool) internal view returns (int24 spacing) {
        (uint256 word,) = _read(pool, IUniswapV3PoolReads.tickSpacing.selector, 0, false, 1);
        spacing = int24(int256(word));
        require(spacing > 0, UnexpectedPoolResponse());
    }

    /// @dev `pool.tickBitmap(wordPos)`.
    function bitmapWord(address pool, int16 wordPos) internal view returns (uint256 bitmap) {
        (bitmap,) = _read(pool, IUniswapV3PoolReads.tickBitmap.selector, uint256(int256(wordPos)), true, 1);
    }

    /// @dev `pool.ticks(tick)`'s first two words, `(liquidityGross, liquidityNet)`.
    function tickLiquidity(address pool, int24 tick)
        internal
        view
        returns (uint128 liquidityGross, int128 liquidityNet)
    {
        (uint256 gross, uint256 net) = _read(pool, IUniswapV3PoolReads.ticks.selector, uint256(int256(tick)), true, 2);
        liquidityGross = uint128(gross);
        liquidityNet = int128(int256(net));
    }

    /// @dev `pool.slot0()`'s first two words, `(sqrtPriceX96, tick)`.
    function slot0(address pool) internal view returns (uint160 sqrtPriceX96, int24 tick) {
        (uint256 price, uint256 current) = _read(pool, IUniswapV3PoolReads.slot0.selector, 0, false, 2);
        sqrtPriceX96 = uint160(price);
        tick = int24(int256(current));
    }

    /// @dev `pool.liquidity()`.
    function liquidity(address pool) internal view returns (uint128 inRange) {
        (uint256 word,) = _read(pool, IUniswapV3PoolReads.liquidity.selector, 0, false, 1);
        inRange = uint128(word);
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
