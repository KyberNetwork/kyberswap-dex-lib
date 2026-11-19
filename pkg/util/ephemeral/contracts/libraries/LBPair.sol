// SPDX-License-Identifier: MIT
pragma solidity ^0.8.37;

/// @notice The accessors a lens reads from a Liquidity Book v2.1 pair, declared with only the
/// leading return words it uses. v2.2 pairs answer the same. Only the selectors are taken from
/// here; the replies are read word by word.
interface ILBPairReads {
    function getActiveId() external view returns (uint24 activeId);
    function getReserves() external view returns (uint128 reserveX, uint128 reserveY);
    function getBin(uint24 id) external view returns (uint128 binReserveX, uint128 binReserveY);
    function getNextNonEmptyBin(bool swapForY, uint24 id) external view returns (uint24 nextId);
}

/// @title LBPair
/// @notice The reads a lens makes of a Liquidity Book v2.1 pair.
/// @dev Each read copies at most two words of the reply; a reply shorter than that means the
/// address is not a pair, and is refused rather than read past its end.
library LBPair {
    /// @notice A reply came back shorter than the words read from it, i.e. the address is not a
    /// Liquidity Book pair.
    error UnexpectedPairResponse();

    // The narrowing casts below take the low bits of an ABI word, which is where the ABI puts a
    // narrow value; zero-extended or not, those bits are the value.
    // forge-lint: disable-start(unsafe-typecast)

    /// @dev `pair.getActiveId()`.
    function activeId(address pair) internal view returns (uint24 id) {
        (uint256 word,) = _read(pair, ILBPairReads.getActiveId.selector, 0, 0, 0, 1);
        id = uint24(word);
    }

    /// @dev `pair.getReserves()`, the reserves of every bin, protocol fees left out.
    function reserves(address pair) internal view returns (uint128 reserveX, uint128 reserveY) {
        (uint256 x, uint256 y) = _read(pair, ILBPairReads.getReserves.selector, 0, 0, 0, 2);
        reserveX = uint128(x);
        reserveY = uint128(y);
    }

    /// @dev `pair.getBin(id)`, `(binReserveX, binReserveY)`.
    function bin(address pair, uint24 id) internal view returns (uint128 reserveX, uint128 reserveY) {
        (uint256 x, uint256 y) = _read(pair, ILBPairReads.getBin.selector, id, 0, 1, 2);
        reserveX = uint128(x);
        reserveY = uint128(y);
    }

    /// @dev `pair.getNextNonEmptyBin(swapForY, id)`: the closest non-empty bin below `id` when
    /// `swapForY`, above it otherwise. Where there is none it answers a sentinel, which differs by
    /// direction; the caller tells the end of the list by the step not moving the way it asked.
    function nextNonEmptyBin(address pair, bool swapForY, uint24 id) internal view returns (uint24 nextId) {
        uint256 direction = swapForY ? 1 : 0;
        (uint256 word,) = _read(pair, ILBPairReads.getNextNonEmptyBin.selector, direction, id, 2, 1);
        nextId = uint24(word);
    }

    // forge-lint: disable-end(unsafe-typecast)

    /// @dev Calls `selector` on `pair` with the first `argc` (0 to 2) of `arg0, arg1`, and returns
    /// the first `words` (1 or 2) words of the reply. A failed call's revert is passed up unchanged.
    function _read(address pair, bytes4 selector, uint256 arg0, uint256 arg1, uint256 argc, uint256 words)
        private
        view
        returns (uint256 first, uint256 second)
    {
        uint256 size;
        assembly ("memory-safe") {
            // Two arguments do not fit the scratch space, so build the calldata at the free pointer
            // without moving it: nothing else is allocated before the call returns.
            let ptr := mload(0x40)
            mstore(ptr, selector)
            mstore(add(ptr, 4), arg0)
            mstore(add(ptr, 0x24), arg1)
            if iszero(staticcall(gas(), pair, ptr, add(4, shl(5, argc)), 0, shl(5, words))) {
                returndatacopy(ptr, 0, returndatasize())
                // The loop is the caller's walk, and a pair that refuses a read must end it.
                // forge-lint: disable-next-line(require-revert-in-loop)
                revert(ptr, returndatasize())
            }
            size := returndatasize()
            first := mload(0)
            second := mload(0x20)
        }
        // As above: a short reply ends the walk.
        // forge-lint: disable-next-line(require-revert-in-loop)
        require(size >= words << 5, UnexpectedPairResponse());
    }
}
