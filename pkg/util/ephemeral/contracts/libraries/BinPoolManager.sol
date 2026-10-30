// SPDX-License-Identifier: MIT
pragma solidity ^0.8.37;

/// @notice The accessors a lens reads from PancakeSwap Infinity's BinPoolManager, declared with only
/// the leading return words it uses. Every read is keyed by the pool id, the manager holding every
/// bin pool's state. Only the selectors are taken from here; the replies are read word by word.
interface IBinPoolManagerReads {
    function getSlot0(bytes32 id) external view returns (uint24 activeId);
    function getBin(bytes32 id, uint24 binId) external view returns (uint128 binReserveX, uint128 binReserveY);
    function getNextNonEmptyBin(bytes32 id, bool swapForY, uint24 binId) external view returns (uint24 nextId);
    function poolIdToPoolKey(bytes32 id)
        external
        view
        returns (
            address currency0,
            address currency1,
            address hooks,
            address poolManager,
            uint24 fee,
            bytes32 parameters
        );
}

/// @title BinPoolManager
/// @notice The reads a lens makes of a PancakeSwap Infinity bin pool, through BinPoolManager.
/// @dev Each read copies only the words it needs, so the longer tuples (`getBin` returns four words,
/// `getSlot0` three) cost nothing extra. A reply shorter than the words read means the address is
/// not a BinPoolManager, and is refused rather than read past its end.
library BinPoolManager {
    /// @notice A reply came back shorter than the words read from it, i.e. the address is not a
    /// BinPoolManager.
    error UnexpectedPoolManagerResponse();

    // The narrowing casts below take the low bits of an ABI word, which is where the ABI puts a
    // narrow value; zero-extended or not, those bits are the value.
    // forge-lint: disable-start(unsafe-typecast)

    /// @dev `getSlot0(poolId)`'s first word, the active bin.
    function activeId(address manager, bytes32 poolId) internal view returns (uint24 id) {
        (uint256 word,) = _read(manager, IBinPoolManagerReads.getSlot0.selector, poolId, 0, 0, 1);
        id = uint24(word);
    }

    /// @dev `getBin(poolId, id)`'s first two words, `(binReserveX, binReserveY)`.
    function bin(address manager, bytes32 poolId, uint24 id)
        internal
        view
        returns (uint128 reserveX, uint128 reserveY)
    {
        (uint256 x, uint256 y) = _read(manager, IBinPoolManagerReads.getBin.selector, poolId, id, 1, 2);
        reserveX = uint128(x);
        reserveY = uint128(y);
    }

    /// @dev `getNextNonEmptyBin(poolId, swapForY, id)`: the closest non-empty bin below `id` when
    /// `swapForY`, above it otherwise. Where there is none it answers a sentinel, which differs by
    /// direction; the caller tells the end of the list by the step not moving the way it asked.
    function nextNonEmptyBin(address manager, bytes32 poolId, bool swapForY, uint24 id)
        internal
        view
        returns (uint24 nextId)
    {
        uint256 word;
        bytes4 selector = IBinPoolManagerReads.getNextNonEmptyBin.selector;
        uint256 size;
        assembly ("memory-safe") {
            // Three arguments do not fit the scratch space, so build the calldata at the free pointer
            // without moving it: nothing else is allocated before the call returns.
            let ptr := mload(0x40)
            mstore(ptr, selector)
            mstore(add(ptr, 4), poolId)
            mstore(add(ptr, 0x24), iszero(iszero(swapForY)))
            mstore(add(ptr, 0x44), and(id, 0xffffff))
            if iszero(staticcall(gas(), manager, ptr, 0x64, 0, 0x20)) {
                returndatacopy(ptr, 0, returndatasize())
                // The loop is the caller's walk, and a manager that refuses a read must end it.
                // forge-lint: disable-next-line(require-revert-in-loop)
                revert(ptr, returndatasize())
            }
            size := returndatasize()
            word := mload(0)
        }
        // As above: a short reply ends the walk.
        // forge-lint: disable-next-line(require-revert-in-loop)
        require(size >= 0x20, UnexpectedPoolManagerResponse());
        nextId = uint24(word);
    }

    /// @dev Whether `poolIdToPoolKey(poolId)` names `manager` as the pool's manager. The manager
    /// answers a zeroed key for an id it does not know, and a zeroed slot0 and no bins with it, so
    /// this is what tells an unknown pool from an empty one. The currencies cannot: the native
    /// currency is address 0.
    function knows(address manager, bytes32 poolId) internal view returns (bool) {
        bytes4 selector = IBinPoolManagerReads.poolIdToPoolKey.selector;
        uint256 size;
        uint256 keyManager;
        assembly ("memory-safe") {
            // Six words of reply do not fit the scratch space, so take them at the free pointer
            // without moving it: nothing else is allocated before they are read.
            let ptr := mload(0x40)
            mstore(ptr, selector)
            mstore(add(ptr, 4), poolId)
            if iszero(staticcall(gas(), manager, ptr, 0x24, ptr, 0xc0)) {
                returndatacopy(ptr, 0, returndatasize())
                revert(ptr, returndatasize())
            }
            size := returndatasize()
            keyManager := mload(add(ptr, 0x60))
        }
        require(size >= 0xc0, UnexpectedPoolManagerResponse());
        return address(uint160(keyManager)) == manager;
    }

    // forge-lint: disable-end(unsafe-typecast)

    /// @dev Calls `selector` on `manager` with `poolId` and, when `hasArg`, `arg`, and returns the
    /// first `words` (1 or 2) words of the reply. A failed call's revert is passed up unchanged.
    function _read(address manager, bytes4 selector, bytes32 poolId, uint256 arg, uint256 hasArg, uint256 words)
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
            mstore(add(ptr, 4), poolId)
            mstore(add(ptr, 0x24), arg)
            if iszero(staticcall(gas(), manager, ptr, add(0x24, shl(5, hasArg)), 0, shl(5, words))) {
                returndatacopy(ptr, 0, returndatasize())
                // The loop is the caller's walk, and a manager that refuses a read must end it.
                // forge-lint: disable-next-line(require-revert-in-loop)
                revert(ptr, returndatasize())
            }
            size := returndatasize()
            first := mload(0)
            second := mload(0x20)
        }
        // As above: a short reply ends the walk.
        // forge-lint: disable-next-line(require-revert-in-loop)
        require(size >= words << 5, UnexpectedPoolManagerResponse());
    }
}
