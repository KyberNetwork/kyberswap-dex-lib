// SPDX-License-Identifier: MIT
pragma solidity ^0.8.37;

/// @title GasPrice
/// @notice How much more a chain charges than Ethereum, told by what one read cost.
/// @dev A lens keeps back gas for its next step and its return, planned at Ethereum's prices: a
/// cold SLOAD at 2,100 and a cold account at 2,600. Polygon charges 5,460 for that SLOAD, and Monad
/// 8,100 and 10,100 for the account, which can take a step of several cold slots past what a lens
/// keeps back. A lens times one read it makes anyway, whose cost on Ethereum it knows, and keeps
/// back that much more.
library GasPrice {
    /// @notice `amount` scaled by `used` over `ethereum`: the gas a read took on this chain against
    /// the most it takes on Ethereum. Never less than `amount`.
    function scale(uint256 amount, uint256 used, uint256 ethereum) internal pure returns (uint256) {
        return used > ethereum ? amount * used / ethereum : amount;
    }
}
