// SPDX-License-Identifier: MIT
pragma solidity ^0.8.37;

import {BinPoolManager} from "../../../libraries/BinPoolManager.sol";
import {ChainBlock} from "../../../libraries/ChainBlock.sol";
import {GasPrice} from "../../../libraries/GasPrice.sol";
import {ReturnGas} from "../../../libraries/ReturnGas.sol";

/// @title PancakeInfinityBinLens
/// @notice Deployless lens that returns a PancakeSwap Infinity bin pool's active bin and non-empty
/// bins from one block, in a single `eth_call`, stopping short of the gas limit rather than running
/// out of it.
///
/// @dev Never deployed. Call with `to == null` and
/// `data == creationCode ++ abi.encode(binPoolManager, poolId, startId)`; the constructor reverts
/// with `abi.encode(getSnapshot's return values)`.
///
/// LiquidityBookV21Lens for an Infinity bin pool, read through BinPoolManager (see
/// BinPoolManager): the same upward walk from `startId` through `getNextNonEmptyBin`, the same stop
/// short of the gas, the same paging by bin id, and the end of the walk told by a step that does
/// not move up rather than by the manager's sentinel. Three things differ:
///
/// - The tree `getNextNonEmptyBin` searches leaves out a bin whose only shares left are the locked
///   minimum, though `getBin` still answers its dust reserves. A swap never steps onto such a bin,
///   but it starts at the active one whatever the tree says, so the walk never steps over the
///   active bin: it is read in its place, and returned when it holds reserves. Seen live on Base,
///   an active bin of 279,563,274 units of X that the tree skipped.
/// - Starting from 0 steps before it reads: `getBin(0)` reverts on Infinity, the price of bin 0
///   underflowing, and no liquidity can be added there for the same reason.
/// - There is no pool total to check the bins against, as a pair's `getReserves()` is: the Vault
///   counts the tokens of every pool of a manager together. The walk is complete when `done`.
///
/// The manager answers an id it does not know with a zeroed slot0 and no bins, which reads as an
/// empty pool; the lens refuses such an id with UnknownPool() instead.
contract PancakeInfinityBinLens {
    /// @notice A non-empty bin and the reserves a swap simulator needs.
    struct PopulatedBin {
        uint24 id;
        uint128 reserveX;
        uint128 reserveY;
    }

    /// @notice `poolId` is not a pool of `binPoolManager`.
    error UnknownPool();

    /// @notice Gas kept back for one more bin and for returning the result.
    /// @dev A bin costs two calls to the manager: `getBin`, which loads the bin's reserves and its
    /// total shares, two cold slots, and works its liquidity out of its price, and
    /// `getNextNonEmptyBin`, whose tree search reads up to two cold slots per level of a three-level
    /// tree. That is ~30k gas at most, of which a call may forward only 63/64. Returning costs a few
    /// hundred: the result is already encoded. 60k covers both, and is nothing next to any node's
    /// eth_call cap. That is at Ethereum's prices: on a chain whose first read of the manager costs
    /// more than FIRST_READ, as Monad's 8,100 a cold slot would, the lens keeps back that much more.
    uint256 internal constant GAS_RESERVE = 60_000;

    /// @dev The most the manager's first read, `poolIdToPoolKey` with the manager and the key cold,
    /// costs where storage is priced as on Ethereum: 13,639 gas on every pool measured.
    uint256 internal constant FIRST_READ = 15_000;

    /// @dev Bytes in the static head of the result, the last word being the bin array's offset.
    uint256 private constant HEAD_SIZE = 0xa0;

    constructor(address binPoolManager, bytes32 poolId, uint24 startId) payable {
        bytes memory result = _snapshot(binPoolManager, poolId, startId);
        assembly ("memory-safe") {
            revert(add(result, 0x20), mload(result))
        }
    }

    /// @notice Reads the pool's active bin and every non-empty bin from `startId` up, as far as the
    /// call's gas allows.
    /// @dev The same values the constructor reverts with, decoded from the same encoding. Never
    /// reached in a deployless call; it is here so the ABI names and describes the payload.
    /// @param binPoolManager The Infinity BinPoolManager holding the pool's state.
    /// @param poolId The pool id, `keccak256(abi.encode(poolKey))`.
    /// @param startId 0 for the whole pool, or a previous call's `nextId`, which is the next bin to
    /// read.
    /// @return activeId The pool's active bin.
    /// @return blockNumber The block every value here was read at.
    /// @return nextId The bin to pass as `startId` to continue. Meaningless when `done` is true.
    /// Equal to `startId` when not even one bin fit, which a caller must treat as too little gas
    /// rather than resume from, or it will loop.
    /// @return done True when no non-empty bin is left above the last one read.
    /// @return bins The non-empty bins read, ascending by id, the active one included when it holds
    /// reserves.
    function getSnapshot(address binPoolManager, bytes32 poolId, uint24 startId)
        public
        view
        returns (uint24 activeId, uint256 blockNumber, uint24 nextId, bool done, PopulatedBin[] memory bins)
    {
        (activeId, blockNumber, nextId, done, bins) =
            abi.decode(_snapshot(binPoolManager, poolId, startId), (uint24, uint256, uint24, bool, PopulatedBin[]));
    }

    /// @dev Builds getSnapshot's return values in their ABI encoding, as a `bytes` whose contents
    /// are the payload: a five-word head, then the bin array's length and its entries, three words
    /// each. Entries are written straight after the head as they are found.
    function _snapshot(address manager, bytes32 poolId, uint24 startId) private view returns (bytes memory result) {
        uint256 before = gasleft();
        bool known = BinPoolManager.knows(manager, poolId);
        uint256 gasReserve = GasPrice.scale(GAS_RESERVE, before - gasleft(), FIRST_READ);
        require(known, UnknownPool());
        uint24 activeId = BinPoolManager.activeId(manager, poolId);

        uint256 entries;
        assembly ("memory-safe") {
            result := mload(0x40)
            // length word, head, array length; entries follow.
            entries := add(result, add(0x40, HEAD_SIZE))
            mstore(0x40, entries)
        }

        uint256 count = 0;
        bool done = false;
        uint24 cursor = startId;
        // Bin 0 is where a whole walk starts, not a bin to read: see the contract's notes.
        bool unread = startId != 0;
        // What copying the result out costs grows with the entries written: worked out for a few
        // more than written, and again once they are.
        uint256 until = ReturnGas.ENTRIES_AHEAD;
        uint256 reserve = gasReserve + ReturnGas.copyCost(HEAD_SIZE + 0x20 + until * 0x60);
        while (gasleft() >= reserve) {
            if (unread) {
                (uint128 binReserveX, uint128 binReserveY) = BinPoolManager.bin(manager, poolId, cursor);
                if (binReserveX != 0 || binReserveY != 0) {
                    assembly ("memory-safe") {
                        let entry := add(entries, mul(count, 0x60))
                        mstore(entry, and(cursor, 0xffffff))
                        mstore(add(entry, 0x20), and(binReserveX, 0xffffffffffffffffffffffffffffffff))
                        mstore(add(entry, 0x40), and(binReserveY, 0xffffffffffffffffffffffffffffffff))
                        mstore(0x40, add(entry, 0x60))
                    }
                    if (++count == until) {
                        until += ReturnGas.ENTRIES_AHEAD;
                        reserve = gasReserve + ReturnGas.copyCost(HEAD_SIZE + 0x20 + until * 0x60);
                    }
                }
            }

            uint24 next = BinPoolManager.nextNonEmptyBin(manager, poolId, false, cursor);
            // The tree may leave the active bin out: see the contract's notes.
            if (cursor < activeId && (next <= cursor || next > activeId)) next = activeId;
            if (next <= cursor) {
                done = true;
                break;
            }
            cursor = next;
            unread = true;
        }

        uint256 blockNumber = ChainBlock.number();
        // Narrow values are cleaned on the way out: the ABI wants them zero-extended, and Solidity
        // does not promise that of a variable seen from assembly.
        assembly ("memory-safe") {
            let length := add(HEAD_SIZE, add(0x20, mul(count, 0x60)))
            mstore(result, length)
            let head := add(result, 0x20)
            mstore(head, and(activeId, 0xffffff))
            mstore(add(head, 0x20), blockNumber)
            mstore(add(head, 0x40), and(cursor, 0xffffff))
            mstore(add(head, 0x60), iszero(iszero(done)))
            mstore(add(head, 0x80), HEAD_SIZE)
            mstore(add(head, HEAD_SIZE), count)
            mstore(0x40, add(head, length))
        }
    }
}
