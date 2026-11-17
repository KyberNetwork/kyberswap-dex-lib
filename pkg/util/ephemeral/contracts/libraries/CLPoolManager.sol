// SPDX-License-Identifier: MIT
pragma solidity ^0.8.37;

/// @notice The accessor a lens reads from PancakeSwap Infinity's CLPoolManager: the pool key, whose
/// last word is the one read. Only the selector is taken from here.
interface ICLPoolManagerReads {
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

/// @title CLPoolManager
/// @notice The tick spacing of a PancakeSwap Infinity concentrated-liquidity pool, from the key
/// CLPoolManager keeps under its id. The pool's state is read from storage (see PoolManagerStorage).
library CLPoolManager {
    /// @notice The pool key came back shorter than its six words, i.e. the address is not a
    /// CLPoolManager.
    error UnexpectedPoolManagerResponse();

    /// @dev The tick spacing in `poolIdToPoolKey(poolId)`. The key is six words, and its last,
    /// `parameters`, keeps the hook permission flags in bits 0..15 and the tick spacing in bits
    /// 16..39. An id the manager does not know reads back as a zeroed key, so as spacing 0.
    function tickSpacing(address manager, bytes32 poolId) internal view returns (int24 spacing) {
        bytes4 selector = ICLPoolManagerReads.poolIdToPoolKey.selector;
        uint256 size;
        uint256 parameters;
        assembly ("memory-safe") {
            // Six words of reply do not fit the scratch space, so take them at the free pointer
            // without moving it: nothing else is allocated before the last one is read.
            let ptr := mload(0x40)
            mstore(ptr, selector)
            mstore(add(ptr, 4), poolId)
            if iszero(staticcall(gas(), manager, ptr, 0x24, ptr, 0xc0)) {
                returndatacopy(ptr, 0, returndatasize())
                revert(ptr, returndatasize())
            }
            size := returndatasize()
            parameters := mload(add(ptr, 0xa0))
        }
        require(size >= 0xc0, UnexpectedPoolManagerResponse());
        // forge-lint: disable-next-line(unsafe-typecast): bits 16..39 of the word are the spacing.
        spacing = int24(int256(parameters >> 16));
    }
}
