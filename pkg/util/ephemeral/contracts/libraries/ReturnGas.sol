// SPDX-License-Identifier: MIT
pragma solidity ^0.8.37;

/// @title ReturnGas
/// @notice Gas a lens keeps back for the frame that runs it to copy its result out.
/// @dev The frame that creates a lens and returns what it reverted with, LensBatch's runner, must
/// copy that into its own memory, paying 3 gas a word and the memory's quadratic fee, with only
/// what the lens left plus the 1/64 CREATE kept from it: a lens that spent its budget and then
/// reverted with a 1MB page left its runner ~0.5M of the ~2.1M the copy costs, and the page was
/// lost. A lens that stops for gas keeps this back too, and the gas comes back to the runner when
/// the lens reverts.
library ReturnGas {
    /// @notice Entries a walk keeps the copy's cost worked out ahead for, so that it does not work
    /// it out again at every step: 64 entries of three words overstate it by ~6KB.
    uint256 internal constant ENTRIES_AHEAD = 64;

    /// @notice What copying out a result of `size` bytes costs.
    /// @dev Counted from the result's size, not the memory in use: a lens's reads leave scratch
    /// behind, ~400KB over the 6,932 words of a spacing-1 scan, which the runner never copies.
    function copyCost(uint256 size) internal pure returns (uint256 cost) {
        assembly ("memory-safe") {
            let words := shr(5, add(size, 31))
            cost := add(mul(6, words), shr(9, mul(words, words)))
        }
    }

    /// @notice What copying out the memory in use, plus `extra` bytes about to be written, costs:
    /// for a lens whose memory is its result.
    function copyCostOfMemory(uint256 extra) internal pure returns (uint256) {
        uint256 end;
        assembly ("memory-safe") {
            end := add(mload(0x40), extra)
        }
        return copyCost(end);
    }
}
