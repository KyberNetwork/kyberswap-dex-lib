// SPDX-License-Identifier: MIT
pragma solidity ^0.8.37;

/// @notice Arbitrum's ArbSys precompile, declared with the one read taken from it.
interface IArbSys {
    function arbBlockNumber() external view returns (uint256);
}

/// @title ChainBlock
/// @notice The number of the block a call runs at, as the chain's own RPC numbers it.
/// @dev On Arbitrum and the Orbit chains built on it, `block.number` is the parent chain's block,
/// not the one the call was pinned to: a lens read at Arbitrum block 517073521 reported 26244634, an
/// Ethereum block. A caller that pins its next page, or its other reads, to the block a lens reports
/// would then read state from years before. ArbSys answers the chain's own number; elsewhere 0x64
/// answers nothing, and `block.number` is the number.
///
/// Elsewhere is not always empty: on BSC 0x64 is a retired precompile that fails, and a failing
/// precompile burns all the gas it is given, which with `gas()` left a lens 1/64 of its call. It has
/// no code, though, while ArbOS gives its precompiles some, so 0x64 is called only where it has code.
/// The read is given ARB_SYS_GAS, far above what ArbSys charges (a few hundred gas, ~3k with the
/// call), should anything else be there.
library ChainBlock {
    address internal constant ARB_SYS = address(100);
    uint256 internal constant ARB_SYS_GAS = 20_000;

    function number() internal view returns (uint256 blockNumber) {
        bytes4 selector = IArbSys.arbBlockNumber.selector;
        address arbSys = ARB_SYS;
        uint256 arbSysGas = ARB_SYS_GAS;
        bool answered = false;
        assembly ("memory-safe") {
            // ArbOS gives its precompiles code (0xfe), and a chain without ArbSys has none at 0x64, not
            // BSC's retired precompile either: called, it would burn its whole ARB_SYS_GAS.
            if extcodesize(arbSys) {
                mstore(0, selector)
                // Its own statement: Yul evaluates arguments right to left, so within one expression
                // returndatasize() would be the previous call's.
                let ok := staticcall(arbSysGas, arbSys, 0, 4, 0, 0x20)
                answered := and(ok, eq(returndatasize(), 0x20))
                blockNumber := mload(0)
            }
        }
        if (!answered) blockNumber = block.number;
    }
}
