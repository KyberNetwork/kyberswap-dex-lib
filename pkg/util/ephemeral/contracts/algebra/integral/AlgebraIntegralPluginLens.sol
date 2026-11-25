// SPDX-License-Identifier: MIT
pragma solidity ^0.8.37;

import {AlgebraPlugin} from "../../libraries/AlgebraPlugin.sol";
import {ChainBlock} from "../../libraries/ChainBlock.sol";
import {ReturnGas} from "../../libraries/ReturnGas.sol";

/// @title AlgebraIntegralPluginLens
/// @notice Deployless lens that reads an Algebra Integral pool's plugin, its fee settings and the
/// volatility oracle's timepoints a tracker keeps, in one `eth_call`, stopping short of the gas
/// limit rather than running out of it.
///
/// @dev Never deployed. Call with `to == null` and
/// `data == creationCode ++ abi.encode(plugin, calls, walk)`; the constructor reverts with
/// `abi.encode(getSnapshot's return values)`.
///
/// A tracker reads a plugin in rounds: one multicall of its settings, then the oracle's timepoints,
/// paged backwards from the newest until one is older than a day, up to sixteen more multicalls,
/// then a few slots past both ends. This lens makes the settings calls the tracker gives it and
/// returns each one's raw `(success, returnData)`, as `tryAggregate` would, then walks the
/// timepoints from the newest itself and returns each one's seven words raw too, so the tracker
/// decodes everything as it does today, in one place.
///
/// The walk steps back one slot at a time (the ring of 65536 wraps) and stops at the first slot
/// that is uninitialized or older than `cutoff`, which it returns, as the tracker's paging finds
/// it; or at a slot the caller already holds, given as `knownIndex` and its `knownTimestamp`, when
/// the slot still holds that timepoint; or after `maxCount` slots; or, short of gas, at a slot it
/// did not read, `next`, to resume from. Walking from the newest, it also returns the slots the
/// tracker reads past both ends: 0, newest + 1, newest + 2 and newest - 1.
contract AlgebraIntegralPluginLens {
    /// @notice How far to walk the timepoints.
    struct Walk {
        // Start at the plugin's newest timepoint, read here, rather than at `start`.
        bool fromHead;
        uint16 start;
        // Stop at the first timepoint older than this.
        uint32 cutoff;
        // Read at most this many.
        uint256 maxCount;
        // Stop at slot knownIndex when it still holds the caller's timepoint of knownTimestamp.
        bool hasKnown;
        uint16 knownIndex;
        uint32 knownTimestamp;
    }

    /// @notice One call's outcome, as Multicall's tryAggregate reports it.
    struct Result {
        bool success;
        bytes returnData;
    }

    /// @notice The call ran out of gas for the next settings read. Recording it as failed, the way
    /// tryAggregate would, reads as a plugin that lacks the setting.
    error GasExhausted();

    /// @notice Gas kept back for one more timepoint, the four slots read past the walk's ends, and
    /// returning the result.
    /// @dev A timepoint is one storage slot behind a warm call, ~5k gas with the copy, and the
    /// result is already encoded. 100k covers a read and the four after the walk several times over.
    uint256 internal constant GAS_RESERVE = 100_000;

    /// @dev Headroom for copying a settings read's result back and moving on to the next.
    uint256 private constant READ_OVERHEAD = 20_000;

    /// @dev Bytes in the static head of the result: ten words, three of them offsets.
    uint256 private constant HEAD_SIZE = 0x140;

    /// @dev The slots read past the walk's ends: 0, newest + 1, newest + 2 and newest - 1.
    uint256 private constant EXTRA_SLOTS = 4;

    constructor(address plugin, bytes[] memory calls, Walk memory walk) payable {
        bytes memory result = _snapshot(plugin, calls, walk);
        assembly ("memory-safe") {
            revert(add(result, 0x20), mload(result))
        }
    }

    /// @notice Makes the settings calls and walks the timepoints, as far as the call's gas allows.
    /// @dev The same values the constructor reverts with, decoded from the same encoding. Never
    /// reached in a deployless call; it is here so the ABI names and describes the payload.
    /// @param plugin The pool's plugin.
    /// @param calls The settings calls to make of it, each its calldata.
    /// @param walk How far to walk the timepoints.
    /// @return blockNumber The block every value here was read at.
    /// @return reads One result per call, in order.
    /// @return walked False when walking from the newest and the plugin has no oracle to walk.
    /// @return head The plugin's newest timepoint, when walking from it.
    /// @return first The slot of the first timepoint returned.
    /// @return next The slot to pass as `start` to go on. Meaningless when `done` is true.
    /// @return done False when the walk stopped short of gas.
    /// @return count The timepoints returned.
    /// @return timepoints Their replies, seven words each, from `first` down.
    /// @return extras The replies for slots 0, head + 1, head + 2 and head - 1, when walking from the
    /// newest.
    function getSnapshot(address plugin, bytes[] memory calls, Walk memory walk)
        public
        view
        returns (
            uint256 blockNumber,
            Result[] memory reads,
            bool walked,
            uint16 head,
            uint16 first,
            uint16 next,
            bool done,
            uint256 count,
            bytes memory timepoints,
            bytes memory extras
        )
    {
        (blockNumber, reads, walked, head, first, next, done, count, timepoints, extras) = abi.decode(
            _snapshot(plugin, calls, walk),
            (uint256, Result[], bool, uint16, uint16, uint16, bool, uint256, bytes, bytes)
        );
    }

    /// @dev Builds getSnapshot's return values in their ABI encoding: a ten-word head, the results'
    /// encoding, then the timepoints, written straight after it as the walk reads them, then the
    /// extra slots.
    function _snapshot(address plugin, bytes[] memory calls, Walk memory walk)
        private
        view
        returns (bytes memory result)
    {
        bytes memory reads = abi.encode(_read(plugin, calls));

        bool walked = true;
        uint16 head = 0;
        uint16 first = walk.start;
        if (walk.fromHead) {
            (walked, head) = AlgebraPlugin.timepointIndex(plugin);
            first = head;
        }

        // reads is abi.encode((Result[])): an offset word, then the array's own encoding.
        uint256 readsSize = reads.length - 0x20;
        uint256 timepointsAt;
        assembly ("memory-safe") {
            result := mload(0x40)
            // length word, head, the results; the timepoints' length word, then the timepoints.
            mcopy(add(result, add(0x20, HEAD_SIZE)), add(reads, 0x40), readsSize)
            timepointsAt := add(result, add(add(0x20, HEAD_SIZE), add(readsSize, 0x20)))
            mstore(0x40, timepointsAt)
        }

        (uint16 next, bool done, uint256 count) = _walk(plugin, walk, walked, first, timepointsAt);

        uint256 extrasAt = timepointsAt + count * AlgebraPlugin.TIMEPOINT_SIZE + 0x20;
        uint256 extraCount = 0;
        if (walked && walk.fromHead) {
            extraCount = EXTRA_SLOTS;
            assembly ("memory-safe") {
                mstore(0x40, add(extrasAt, mul(EXTRA_SLOTS, 0xe0)))
            }
            unchecked {
                AlgebraPlugin.readTimepoint(plugin, 0, extrasAt);
                AlgebraPlugin.readTimepoint(plugin, head + 1, extrasAt + AlgebraPlugin.TIMEPOINT_SIZE);
                AlgebraPlugin.readTimepoint(plugin, head + 2, extrasAt + 2 * AlgebraPlugin.TIMEPOINT_SIZE);
                AlgebraPlugin.readTimepoint(plugin, head - 1, extrasAt + 3 * AlgebraPlugin.TIMEPOINT_SIZE);
            }
        }

        uint256 blockNumber = ChainBlock.number();
        // Narrow values are cleaned on the way out: the ABI wants them zero-extended, and Solidity
        // does not promise that of a variable seen from assembly.
        assembly ("memory-safe") {
            let h := add(result, 0x20)
            let timepointsOffset := add(HEAD_SIZE, readsSize)
            let timepointsSize := mul(count, 0xe0)
            let extrasOffset := add(timepointsOffset, add(0x20, timepointsSize))
            let extrasSize := mul(extraCount, 0xe0)
            mstore(h, blockNumber)
            mstore(add(h, 0x20), HEAD_SIZE)
            mstore(add(h, 0x40), iszero(iszero(walked)))
            mstore(add(h, 0x60), and(head, 0xffff))
            mstore(add(h, 0x80), and(first, 0xffff))
            mstore(add(h, 0xa0), and(next, 0xffff))
            mstore(add(h, 0xc0), iszero(iszero(done)))
            mstore(add(h, 0xe0), count)
            mstore(add(h, 0x100), timepointsOffset)
            mstore(add(h, 0x120), extrasOffset)
            mstore(add(h, timepointsOffset), timepointsSize)
            mstore(add(h, extrasOffset), extrasSize)
            let length := add(extrasOffset, add(0x20, extrasSize))
            mstore(result, length)
            mstore(0x40, add(h, length))
        }
    }

    /// @dev Makes each settings call of the plugin, as tryAggregate does.
    function _read(address plugin, bytes[] memory calls) private view returns (Result[] memory reads) {
        reads = new Result[](calls.length);
        for (uint256 i; i < calls.length; ++i) {
            // Start a read only with its whole budget in hand: one that failed for want of gas would
            // read as a setting the plugin lacks.
            // forge-lint: disable-next-line(require-revert-in-loop)
            require(gasleft() >= AlgebraPlugin.READ_GAS + AlgebraPlugin.READ_GAS / 63 + READ_OVERHEAD, GasExhausted());
            // The calls are the tracker's few reads of the plugin's settings, each a word or a few.
            // forge-lint: disable-next-line(calls-loop, return-bomb)
            (reads[i].success, reads[i].returnData) = plugin.staticcall{gas: AlgebraPlugin.READ_GAS}(calls[i]);
        }
    }

    /// @dev Walks the timepoints down from `first`, writing each one's seven words at `to` in turn.
    /// Returns the slot to resume from, whether the walk is over, and how many it read.
    function _walk(address plugin, Walk memory walk, bool walked, uint16 first, uint256 to)
        private
        view
        returns (uint16 next, bool done, uint256 count)
    {
        next = first;
        if (!walked) return (next, true, 0);
        while (count < walk.maxCount) {
            if (gasleft() < GAS_RESERVE + ReturnGas.copyCostOfMemory(AlgebraPlugin.TIMEPOINT_SIZE)) {
                return (next, false, count);
            }
            uint256 slot = to + count * AlgebraPlugin.TIMEPOINT_SIZE;
            assembly ("memory-safe") {
                mstore(0x40, add(slot, 0xe0))
            }
            (bool initialized, uint32 blockTimestamp) = AlgebraPlugin.readTimepoint(plugin, next, slot);
            ++count;
            if (!initialized || blockTimestamp < walk.cutoff) return (next, true, count);
            if (walk.hasKnown && next == walk.knownIndex && blockTimestamp == walk.knownTimestamp) {
                return (next, true, count);
            }
            unchecked {
                --next;
            }
        }
        return (next, true, count);
    }
}
