// SPDX-License-Identifier: MIT
pragma solidity ^0.8.37;

/// @notice The accessors a lens reads from a Liquidity Book v2.0 pair, and the error its tree
/// search raises when there is no bin further on. Only the selectors are taken from here; the
/// replies are read word by word.
interface ILBPairV20Reads {
    error TreeMath__ErrorDepthSearch();

    function getReservesAndId() external view returns (uint256 reserveX, uint256 reserveY, uint256 activeId);
    function getBin(uint24 id) external view returns (uint256 reserveX, uint256 reserveY);
    function findFirstNonEmptyBinId(uint24 id, bool sentTokenY) external view returns (uint24 nextId);
}

/// @title LBPairV20
/// @notice The reads a lens makes of a Liquidity Book v2.0 pair.
/// @dev v2.0 differs from v2.1 in three reads: the active id comes with the reserves, the step takes
/// its arguments the other way round, and a step with nowhere to go reverts rather than answering a
/// sentinel. Reserves are uint256 in the ABI, but stored as uint112 per bin and uint136 per pair, so
/// one wider than the uint128 the result holds is refused rather than truncated.
library LBPairV20 {
    /// @notice A reply came back shorter than the words read from it, i.e. the address is not a
    /// Liquidity Book v2.0 pair.
    error UnexpectedPairResponse();

    /// @notice A reserve does not fit the result's uint128.
    error ReserveOverflow();

    // The narrowing casts below take the low bits of an ABI word, which is where the ABI puts a
    // narrow value; the reserves are checked to fit first.
    // forge-lint: disable-start(unsafe-typecast)

    /// @dev `pair.getReservesAndId()`: the pair's reserves, fees left out, and its active bin.
    function reservesAndId(address pair) internal view returns (uint128 reserveX, uint128 reserveY, uint24 activeId) {
        bytes4 selector = ILBPairV20Reads.getReservesAndId.selector;
        uint256 size;
        uint256 x;
        uint256 y;
        uint256 id;
        assembly ("memory-safe") {
            // Three words of reply do not fit the scratch space, so take them at the free pointer
            // without moving it: nothing else is allocated before they are read.
            let ptr := mload(0x40)
            mstore(ptr, selector)
            if iszero(staticcall(gas(), pair, ptr, 4, ptr, 0x60)) {
                returndatacopy(ptr, 0, returndatasize())
                revert(ptr, returndatasize())
            }
            size := returndatasize()
            x := mload(ptr)
            y := mload(add(ptr, 0x20))
            id := mload(add(ptr, 0x40))
        }
        require(size >= 0x60, UnexpectedPairResponse());
        require(x >> 128 == 0 && y >> 128 == 0, ReserveOverflow());
        reserveX = uint128(x);
        reserveY = uint128(y);
        activeId = uint24(id);
    }

    /// @dev `pair.getBin(id)`, `(reserveX, reserveY)`.
    function bin(address pair, uint24 id) internal view returns (uint128 reserveX, uint128 reserveY) {
        bytes4 selector = ILBPairV20Reads.getBin.selector;
        uint256 size;
        uint256 x;
        uint256 y;
        assembly ("memory-safe") {
            // Scratch space only: selector at 0, argument at 4, reply over both.
            mstore(0, selector)
            mstore(4, and(id, 0xffffff))
            if iszero(staticcall(gas(), pair, 0, 0x24, 0, 0x40)) {
                let ptr := mload(0x40)
                returndatacopy(ptr, 0, returndatasize())
                // The loop is the caller's walk, and a pair that refuses a read must end it.
                // forge-lint: disable-next-line(require-revert-in-loop)
                revert(ptr, returndatasize())
            }
            size := returndatasize()
            x := mload(0)
            y := mload(0x20)
        }
        // As above: a short or overwide reply ends the walk.
        // forge-lint: disable-start(require-revert-in-loop)
        require(size >= 0x40, UnexpectedPairResponse());
        require(x >> 128 == 0 && y >> 128 == 0, ReserveOverflow());
        // forge-lint: disable-end(require-revert-in-loop)
        reserveX = uint128(x);
        reserveY = uint128(y);
    }

    /// @dev `pair.findFirstNonEmptyBinId(id, false)`: the closest non-empty bin above `id`, and
    /// whether there is one. The pair says there is none by reverting TreeMath__ErrorDepthSearch(),
    /// and only that revert is taken as the end: any other failure, an out-of-gas included, which
    /// carries no data, is passed up, so a walk never ends early with bins left unread.
    function nextNonEmptyBinAbove(address pair, uint24 id) internal view returns (uint24 nextId, bool found) {
        bytes4 selector = ILBPairV20Reads.findFirstNonEmptyBinId.selector;
        bytes4 depthSearch = ILBPairV20Reads.TreeMath__ErrorDepthSearch.selector;
        uint256 size;
        uint256 word;
        assembly ("memory-safe") {
            // Two arguments do not fit the scratch space, so build the calldata at the free pointer
            // without moving it: nothing else is allocated before the call returns.
            let ptr := mload(0x40)
            mstore(ptr, selector)
            mstore(add(ptr, 4), and(id, 0xffffff))
            mstore(add(ptr, 0x24), 0)
            found := staticcall(gas(), pair, ptr, 0x44, 0, 0x20)
            if iszero(found) {
                returndatacopy(ptr, 0, returndatasize())
                let theEnd := and(eq(returndatasize(), 4), eq(shr(224, mload(ptr)), shr(224, depthSearch)))
                if iszero(theEnd) {
                    // The loop is the caller's walk, and a pair that refuses a step must end it.
                    // forge-lint: disable-next-line(require-revert-in-loop)
                    revert(ptr, returndatasize())
                }
            }
            size := returndatasize()
            word := mload(0)
        }
        if (!found) return (0, false);
        // As above: a short reply ends the walk.
        // forge-lint: disable-next-line(require-revert-in-loop)
        require(size >= 0x20, UnexpectedPairResponse());
        nextId = uint24(word);
    }

    // forge-lint: disable-end(unsafe-typecast)
}
