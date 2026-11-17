// SPDX-License-Identifier: MIT
pragma solidity ^0.8.37;

/// @notice The accessors a lens reads from an Algebra Integral plugin's volatility oracle,
/// declared with only the leading return words it uses: `timepoints()` answers seven words
/// (initialized, blockTimestamp, tickCumulative, volatilityCumulative, tick, averageTick,
/// windowStartIndex), of which the walk decides on the first two. Only the selectors are taken
/// from here; the replies are read word by word.
interface IAlgebraPluginReads {
    function timepointIndex() external view returns (uint16);
    function timepoints(uint256 index) external view returns (bool initialized, uint32 blockTimestamp);
}

/// @title AlgebraPlugin
/// @notice The reads a lens makes of an Algebra Integral plugin's volatility oracle.
library AlgebraPlugin {
    /// @notice A timepoint came back shorter than its seven words, i.e. the plugin's oracle is not
    /// the one the caller decodes.
    error UnexpectedPluginResponse();

    /// @notice Bytes of one timepoint as `timepoints()` answers it: seven words.
    uint256 internal constant TIMEPOINT_SIZE = 0xe0;

    /// @notice Gas one read may use: a timepoint is a single storage slot, and a plugin that burns
    /// what it is given (an INVALID opcode) costs this much rather than the whole call.
    uint256 internal constant READ_GAS = 1_000_000;

    /// @dev `plugin.timepointIndex()`, and whether the plugin answered it: one without a volatility
    /// oracle does not, and has no timepoints to read.
    function timepointIndex(address plugin) internal view returns (bool answered, uint16 index) {
        bytes4 selector = IAlgebraPluginReads.timepointIndex.selector;
        uint256 readGas = READ_GAS;
        uint256 word;
        assembly ("memory-safe") {
            mstore(0, selector)
            let ok := staticcall(readGas, plugin, 0, 4, 0, 0x20)
            answered := and(ok, iszero(lt(returndatasize(), 0x20)))
            word := mload(0)
        }
        // forge-lint: disable-next-line(unsafe-typecast): the low bits of an ABI word are the value.
        index = uint16(word);
    }

    /// @dev Copies `plugin.timepoints(index)`'s seven words to memory at `to`, which the caller has
    /// set aside, and returns the first two. A failed read's revert is passed up unchanged: the walk
    /// cannot step over a slot it did not read.
    function readTimepoint(address plugin, uint16 index, uint256 to)
        internal
        view
        returns (bool initialized, uint32 blockTimestamp)
    {
        bytes4 selector = IAlgebraPluginReads.timepoints.selector;
        uint256 readGas = READ_GAS;
        uint256 size;
        uint256 first;
        uint256 second;
        assembly ("memory-safe") {
            mstore(0, selector)
            mstore(4, and(index, 0xffff))
            if iszero(staticcall(readGas, plugin, 0, 0x24, 0, 0)) {
                let ptr := mload(0x40)
                returndatacopy(ptr, 0, returndatasize())
                // The loop is the caller's walk, and a plugin that refuses a read must end it.
                // forge-lint: disable-next-line(require-revert-in-loop)
                revert(ptr, returndatasize())
            }
            size := returndatasize()
        }
        // As above: a short reply ends the walk.
        // forge-lint: disable-next-line(require-revert-in-loop)
        require(size >= TIMEPOINT_SIZE, UnexpectedPluginResponse());
        assembly ("memory-safe") {
            returndatacopy(to, 0, 0xe0)
            first := mload(to)
            second := mload(add(to, 0x20))
        }
        initialized = first != 0;
        // forge-lint: disable-next-line(unsafe-typecast): the low bits of an ABI word are the value.
        blockTimestamp = uint32(second);
    }
}
