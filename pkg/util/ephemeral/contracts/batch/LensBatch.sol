// SPDX-License-Identifier: MIT
pragma solidity ^0.8.37;

import {ChainBlock} from "../libraries/ChainBlock.sol";

/// @title LensBatchRunner
/// @notice Runs one lens for LensBatch and hands back what it reverted with.
/// @dev It exists for its gas: CREATE forwards all but 1/64 of what the creating frame has, and only
/// a CALL takes a limit, so LensBatch calls this with each item's budget and this creates the lens.
/// A lens always reverts with its result; one that deploys is not a lens, and this reverts with
/// nothing, which LensBatch records as an item that did not run.
contract LensBatchRunner {
    fallback() external {
        assembly {
            calldatacopy(0, 0, calldatasize())
            if create(0, 0, calldatasize()) { revert(0, 0) }
            returndatacopy(0, 0, returndatasize())
            return(0, returndatasize())
        }
    }
}

/// @title LensBatch
/// @notice Deployless contract that runs many lens reads in one `eth_call`, at one block, as far as
/// the call's gas allows, and returns what each one reverted with.
///
/// @dev Never deployed. Call with `to == null` and
/// `data == creationCode ++ abi.encode(codes, items, minItemGas)`; the constructor reverts with
/// `abi.encode(getResults' return values)`.
///
/// An item is a lens's creation code, by its index in `codes`, and the lens's constructor
/// arguments. Items share their codes because a node refuses init code over 49152 bytes (EIP-3860)
/// for the whole call: a lens is ~1.3KB of code and ~0.1KB of arguments, so sharing it is what lets
/// a batch hold a hundred pools of a dex rather than thirty.
///
/// Every lens pages itself by the gas it is given, so an item given less than its whole read
/// still returns a page of its own, `done` false and its cursor, which the caller resumes from as
/// after any page. Items run in order, each given at most its `gas` (0 for no limit) and at most
/// what is left once the batch keeps what it needs to finish. An item that would get less than
/// `minItemGas` is not started, nor is any after it: a lens needs enough for its largest step, a
/// bitmap word of 256 ticks, ~4.7M gas on the forks whose tick getter loads the most, or it
/// returns no progress at all.
contract LensBatch {
    /// @notice One lens read: `codes[code] ++ args`, given at most `gas` (0 for no limit).
    struct Item {
        uint256 code;
        bytes args;
        uint256 gas;
    }

    /// @notice What an item reverted with, whether it ran, and the gas it cost the batch, its lens,
    /// the call to it and copying its result, by which a caller sizes its next batches. An item that did not run, cut
    /// off by the batch's gas or not a lens, has no data; one that ran out of gas inside the lens ran
    /// and has none either. One whose result was too large for the batch to copy is reported as not
    /// run, with no data but the gas it cost, which is what reading it alone takes at least.
    struct Result {
        bool ran;
        uint256 gas;
        bytes data;
    }

    /// @notice An item names a code the batch was not given.
    error UnknownCode();

    /// @notice Gas kept back, past the last item, for writing the results' head and returning.
    uint256 internal constant FINISH_RESERVE = 30_000;

    /// @notice Gas writing one item not started costs besides the memory its head takes.
    uint256 internal constant TAIL_ITEM_GAS = 300;

    /// @notice Gas an item costs besides what the lens is given and the memory its init code is built
    /// in: the call to the runner and the runner's own copies, for a lens of a few kilobytes.
    uint256 internal constant ITEM_OVERHEAD = 20_000;

    constructor(bytes[] memory codes, Item[] memory items, uint256 minItemGas) payable {
        bytes memory result = _batch(codes, items, minItemGas);
        assembly ("memory-safe") {
            revert(add(result, 0x20), mload(result))
        }
    }

    /// @notice Runs `items` in order and returns what each reverted with.
    /// @dev The same values the constructor reverts with, decoded from the same encoding. Never
    /// reached in a deployless call; it is here so the ABI names and describes the payload. Not a
    /// view: the batch creates its lenses.
    /// @param codes The lenses' creation codes.
    /// @param items The reads, each a code index and its constructor arguments.
    /// @param minItemGas The least an item is started with.
    /// @return blockNumber The block every item ran at.
    /// @return count How many items, from the first, were started; the rest did not run.
    /// @return results One per item, in order.
    function getResults(bytes[] memory codes, Item[] memory items, uint256 minItemGas)
        public
        returns (uint256 blockNumber, uint256 count, Result[] memory results)
    {
        (blockNumber, count, results) = abi.decode(_batch(codes, items, minItemGas), (uint256, uint256, Result[]));
    }

    /// @dev Builds getResults' return values in their ABI encoding: a three-word head, the result
    /// array's length and one offset per item, then each item's `(ran, gas, 0x60, length, data)` as
    /// it comes back, the revert data copied once, straight into place.
    function _batch(bytes[] memory codes, Item[] memory items, uint256 minItemGas)
        private
        returns (bytes memory result)
    {
        for (uint256 i; i < items.length; ++i) {
            // forge-lint: disable-next-line(require-revert-in-loop): an item naming no code is a bad batch, refused whole.
            require(items[i].code < codes.length, UnknownCode());
        }

        uint256 blockNumber = ChainBlock.number();
        address runner = address(new LensBatchRunner());

        uint256 n = items.length;
        uint256 offsets;
        uint256 cursor;
        assembly ("memory-safe") {
            result := mload(0x40)
            // length word, three-word head, array length; offsets follow, then the items.
            offsets := add(result, 0x80)
            cursor := add(offsets, add(0x20, shl(5, n)))
            mstore(0x40, cursor)
        }

        uint256 count = 0;
        for (; count < n; ++count) {
            Item memory item = items[count];
            uint256 left = gasleft();
            // The heads of this item and every one after it may still have to be written as not run,
            // and its init code is built at the cursor, past every result read so far.
            uint256 initSize = codes[item.code].length + item.args.length;
            uint256 keep = FINISH_RESERVE + ITEM_OVERHEAD + _tailCost(cursor, n - count)
                + _memoryCost(cursor, cursor + 0x80 + initSize) + ((initSize + 31) >> 5) * 3 + left / 64;
            if (left <= keep) break;
            uint256 budget = left - keep;
            if (item.gas != 0 && item.gas < budget) budget = item.gas;
            uint256 least = item.gas != 0 && item.gas < minItemGas ? item.gas : minItemGas;
            if (budget < least) break;

            bool written;
            (cursor, written) = _run(runner, codes[item.code], item.args, budget, offsets, count, cursor, n);
            if (!written) {
                // Started, and its gas spent, but its result is not in the batch.
                ++count;
                break;
            }
        }

        // The items not started are written as not run, so the array still has one per item.
        for (uint256 i = count; i < n; ++i) {
            cursor = _write(offsets, i, cursor, false, 0, 0);
        }

        assembly ("memory-safe") {
            let head := add(result, 0x20)
            mstore(head, blockNumber)
            mstore(add(head, 0x20), count)
            mstore(add(head, 0x40), 0x60)
            mstore(add(head, 0x60), n)
            mstore(result, sub(cursor, head))
            mstore(0x40, cursor)
        }
    }

    /// @dev Runs one item through the runner under `budget` and writes its result at `cursor`.
    /// Returns the cursor past it, and false when copying what came back would leave the batch short
    /// of what it needs to finish: memory grows quadratically, and a dense pool's result is
    /// megabytes. The item is then written as not run, with the gas it cost and no data.
    function _run(
        address runner,
        bytes memory code,
        bytes memory args,
        uint256 budget,
        uint256 offsets,
        uint256 index,
        uint256 cursor,
        uint256 n
    ) private returns (uint256, bool) {
        bool ran;
        uint256 size;
        uint256 before = gasleft();
        assembly ("memory-safe") {
            // The init code is built where the item's data will go, past the free memory pointer:
            // nothing is allocated before the call returns, and the data then overwrites it.
            let initCode := add(cursor, 0x80)
            let codeSize := mload(code)
            let argsSize := mload(args)
            mcopy(initCode, add(code, 0x20), codeSize)
            mcopy(add(initCode, codeSize), add(args, 0x20), argsSize)
            ran := call(budget, runner, 0, initCode, add(codeSize, argsSize), 0, 0)
            size := returndatasize()
        }
        uint256 used = before - gasleft();
        // A runner that failed ran out of gas or created something that is not a lens; either way
        // nothing it returned is a lens's.
        if (!ran) size = 0;

        uint256 padded = (size + 31) & ~uint256(31);
        uint256 end = cursor + 0x80 + padded + 0x20;
        // returndatacopy charges 3 gas a word on top of the memory it grows into, and the items after
        // this one must still fit as not run past it.
        uint256 tail = n - index - 1;
        uint256 cost = _memoryCost(cursor, end + tail * 0x80) + tail * TAIL_ITEM_GAS + (padded >> 5) * 3;
        if (gasleft() < cost + FINISH_RESERVE) return (_write(offsets, index, cursor, false, used, 0), false);

        // The copy is the item's cost too: a caller sizing its batches by the gas items report would
        // otherwise miss a third of a large result's.
        uint256 copying = gasleft();
        assembly ("memory-safe") {
            returndatacopy(add(cursor, 0x80), 0, size)
            // Zero the padding after the data, which the ABI wants and the init code left dirty.
            mstore(add(add(cursor, 0x80), size), 0)
        }
        used += copying - gasleft();
        return (_write(offsets, index, cursor, ran, used, size), true);
    }

    /// @dev Writes item `index`'s offset and the head of its tuple at `cursor`, its data already in
    /// place after it, and returns the cursor past the data.
    function _write(uint256 offsets, uint256 index, uint256 cursor, bool ran, uint256 used, uint256 size)
        private
        pure
        returns (uint256 next)
    {
        assembly ("memory-safe") {
            mstore(add(offsets, add(0x20, shl(5, index))), sub(cursor, add(offsets, 0x20)))
            mstore(cursor, iszero(iszero(ran)))
            mstore(add(cursor, 0x20), used)
            mstore(add(cursor, 0x40), 0x60)
            mstore(add(cursor, 0x60), size)
            next := add(add(cursor, 0x80), and(add(size, 31), not(31)))
            mstore(0x40, next)
        }
    }

    /// @dev What writing `items` heads of items not started at `cursor` costs at most.
    function _tailCost(uint256 cursor, uint256 items) private pure returns (uint256) {
        return _memoryCost(cursor, cursor + items * 0x80) + items * TAIL_ITEM_GAS;
    }

    /// @dev What growing memory from `start` to `end` bytes would cost at most: the EVM charges
    /// 3 * words + words^2 / 512 for the memory in use, and only for the words added. Memory is in
    /// use at least up to `start`, the cursor, so counting from there never undercounts.
    function _memoryCost(uint256 start, uint256 end) private pure returns (uint256) {
        uint256 to = (end + 31) / 32;
        uint256 from = start / 32;
        if (to <= from) return 0;
        return _memoryFee(to) - _memoryFee(from);
    }

    /// @dev The EVM's fee for `words` words of memory, rounded down as the EVM rounds it.
    function _memoryFee(uint256 words) private pure returns (uint256) {
        uint256 quadratic = words * words;
        return words * 3 + quadratic / 512;
    }
}
