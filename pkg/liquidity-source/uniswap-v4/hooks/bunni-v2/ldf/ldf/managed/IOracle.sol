pragma solidity ^0.8.19;

interface IOracle {
    /// @dev Scaled by WAD (1e18)
    function getFloorPrice() external view returns (uint256);
}
