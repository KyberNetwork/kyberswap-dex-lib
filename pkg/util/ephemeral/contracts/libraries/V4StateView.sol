// SPDX-License-Identifier: MIT
pragma solidity ^0.8.37;

/// @notice The accessor a lens reads from Uniswap V4's StateView. Only the selector is taken from
/// here.
interface IStateViewReads {
    function poolManager() external view returns (address);
}

/// @title V4StateView
/// @notice The PoolManager a Uniswap V4 StateView reads, which is where a lens reads the pools.
/// @dev A tracker is configured with StateView's address, so the lens asks it for the manager's
/// rather than taking a second address.
library V4StateView {
    /// @notice A reply came back shorter than a word, or naming no manager, i.e. the address is not
    /// a StateView.
    error UnexpectedStateViewResponse();

    function poolManager(address stateView) internal view returns (address manager) {
        bytes4 selector = IStateViewReads.poolManager.selector;
        uint256 size;
        assembly ("memory-safe") {
            mstore(0, selector)
            if iszero(staticcall(gas(), stateView, 0, 4, 0, 0x20)) {
                let ptr := mload(0x40)
                returndatacopy(ptr, 0, returndatasize())
                revert(ptr, returndatasize())
            }
            size := returndatasize()
            manager := and(mload(0), 0xffffffffffffffffffffffffffffffffffffffff)
        }
        require(size >= 0x20 && manager != address(0), UnexpectedStateViewResponse());
    }
}
