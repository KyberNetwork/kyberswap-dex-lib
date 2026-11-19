// SPDX-License-Identifier: MIT
pragma solidity ^0.8.37;

import {ChainBlock} from "../../libraries/ChainBlock.sol";
import {GasPrice} from "../../libraries/GasPrice.sol";
import {LBPair} from "../../libraries/LBPair.sol";
import {ReturnGas} from "../../libraries/ReturnGas.sol";

/// @title LiquidityBookV21Lens
/// @notice Deployless lens that returns a Liquidity Book v2.1 pair's active bin, reserves and
/// non-empty bins from one block, in a single `eth_call`, stopping short of the gas limit rather
/// than running out of it.
///
/// @dev Never deployed. Call with `to == null` and `data == creationCode ++ abi.encode(pair, startId)`;
/// the constructor reverts with `abi.encode(getSnapshot's return values)`.
///
/// Liquidity sits in fixed-price bins holding reserves of both tokens, and the pair exposes no
/// bitmap, only `getNextNonEmptyBin`, which jumps from a bin to the closest non-empty one. So the
/// lens walks: from `startId` towards higher ids, reading each bin it lands on, and pages by bin id.
/// Starting from 0 reads every bin in one direction, the active one included, so a whole read
/// counts each bin once.
///
/// The walk ends when a step does not move up. The pair answers 0 when there is no bin above, but
/// that is a convention of its tree search rather than a documented promise, and a sentinel taken
/// on trust once made a walk wrap around and read the same bins again; the direction of the step is
/// what is checked.
///
/// The reply checks itself: `getReserves()` is the sum of every bin's reserves, protocol fees left
/// out of both, so over a whole walk the bins' reserves add up to `reserveX` and `reserveY`. A bin
/// missing from the walk, or read at another block, shows as a difference.
contract LiquidityBookV21Lens {
    /// @notice A non-empty bin and the reserves a swap simulator needs.
    struct PopulatedBin {
        uint24 id;
        uint128 reserveX;
        uint128 reserveY;
    }

    /// @notice Gas kept back for one more bin and for returning the result.
    /// @dev A bin costs two calls to the pair: `getBin`, one cold slot, and `getNextNonEmptyBin`,
    /// whose tree search reads up to two cold slots per level of a three-level tree. That is ~20k gas
    /// at most, of which a call may forward only 63/64. Returning costs a few hundred: the result is
    /// already encoded. 60k covers both several times over, and is nothing next to any node's
    /// eth_call cap. That is at Ethereum's prices: on a chain whose first read of the pair costs more
    /// than FIRST_READ, as Monad's 8,100 a cold slot does, the lens keeps back that much more.
    uint256 internal constant GAS_RESERVE = 60_000;

    /// @dev The most the pair's first read, `getActiveId` with the pair and its implementation cold,
    /// costs where storage is priced as on Ethereum: 7,904 to 8,356 gas on the pairs measured,
    /// against 28,878 and 48,503 on Monad's.
    uint256 internal constant FIRST_READ = 9_000;

    /// @dev Bytes in the static head of the result, the last word being the bin array's offset.
    uint256 private constant HEAD_SIZE = 0xe0;

    constructor(address pair, uint24 startId) payable {
        bytes memory result = _snapshot(pair, startId);
        assembly ("memory-safe") {
            revert(add(result, 0x20), mload(result))
        }
    }

    /// @notice Reads the pair's active bin and reserves, and every non-empty bin from `startId`
    /// up, as far as the call's gas allows.
    /// @dev The same values the constructor reverts with, decoded from the same encoding. Never
    /// reached in a deployless call; it is here so the ABI names and describes the payload.
    /// @param pair The Liquidity Book v2.1 or v2.2 pair to walk.
    /// @param startId The first bin to read: 0 for the whole pair, or a previous call's `nextId`.
    /// @return activeId The pair's active bin.
    /// @return reserveX The pair's reserve of token X, which every bin's reserveX sums to.
    /// @return reserveY The pair's reserve of token Y, which every bin's reserveY sums to.
    /// @return blockNumber The block every value here was read at.
    /// @return nextId The bin to pass as `startId` to continue. Meaningless when `done` is true.
    /// Equal to `startId` when not even one bin fit, which a caller must treat as too little gas
    /// rather than resume from, or it will loop.
    /// @return done True when no non-empty bin is left above the last one read.
    /// @return bins The non-empty bins read, ascending by id.
    function getSnapshot(address pair, uint24 startId)
        public
        view
        returns (
            uint24 activeId,
            uint128 reserveX,
            uint128 reserveY,
            uint256 blockNumber,
            uint24 nextId,
            bool done,
            PopulatedBin[] memory bins
        )
    {
        (activeId, reserveX, reserveY, blockNumber, nextId, done, bins) = abi.decode(
            _snapshot(pair, startId), (uint24, uint128, uint128, uint256, uint24, bool, PopulatedBin[])
        );
    }

    /// @dev Builds getSnapshot's return values in their ABI encoding, as a `bytes` whose contents
    /// are the payload: a seven-word head, then the bin array's length and its entries, three words
    /// each. Entries are written straight after the head as they are found.
    function _snapshot(address pair, uint24 startId) private view returns (bytes memory result) {
        uint256 before = gasleft();
        uint24 activeId = LBPair.activeId(pair);
        uint256 gasReserve = GasPrice.scale(GAS_RESERVE, before - gasleft(), FIRST_READ);
        (uint128 reserveX, uint128 reserveY) = LBPair.reserves(pair);

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
        // What copying the result out costs grows with the entries written: worked out for a few
        // more than written, and again once they are.
        uint256 until = ReturnGas.ENTRIES_AHEAD;
        uint256 reserve = gasReserve + ReturnGas.copyCost(HEAD_SIZE + 0x20 + until * 0x60);
        while (gasleft() >= reserve) {
            (uint128 binReserveX, uint128 binReserveY) = LBPair.bin(pair, cursor);
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

            uint24 next = LBPair.nextNonEmptyBin(pair, false, cursor);
            if (next <= cursor) {
                done = true;
                break;
            }
            cursor = next;
        }

        uint256 blockNumber = ChainBlock.number();
        // Narrow values are cleaned on the way out: the ABI wants them zero-extended, and Solidity
        // does not promise that of a variable seen from assembly.
        assembly ("memory-safe") {
            let length := add(HEAD_SIZE, add(0x20, mul(count, 0x60)))
            mstore(result, length)
            let head := add(result, 0x20)
            mstore(head, and(activeId, 0xffffff))
            mstore(add(head, 0x20), and(reserveX, 0xffffffffffffffffffffffffffffffff))
            mstore(add(head, 0x40), and(reserveY, 0xffffffffffffffffffffffffffffffff))
            mstore(add(head, 0x60), blockNumber)
            mstore(add(head, 0x80), and(cursor, 0xffffff))
            mstore(add(head, 0xa0), iszero(iszero(done)))
            mstore(add(head, 0xc0), HEAD_SIZE)
            mstore(add(head, HEAD_SIZE), count)
            mstore(0x40, add(head, length))
        }
    }
}
