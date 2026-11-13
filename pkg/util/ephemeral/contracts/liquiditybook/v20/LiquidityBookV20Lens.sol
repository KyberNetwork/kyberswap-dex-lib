// SPDX-License-Identifier: MIT
pragma solidity ^0.8.37;

import {ChainBlock} from "../../libraries/ChainBlock.sol";
import {GasPrice} from "../../libraries/GasPrice.sol";
import {LBPairV20} from "../../libraries/LBPairV20.sol";
import {ReturnGas} from "../../libraries/ReturnGas.sol";

/// @title LiquidityBookV20Lens
/// @notice Deployless lens that returns a Liquidity Book v2.0 pair's active bin, reserves and
/// non-empty bins from one block, in a single `eth_call`, stopping short of the gas limit rather
/// than running out of it.
///
/// @dev Never deployed. Call with `to == null` and `data == creationCode ++ abi.encode(pair, startId)`;
/// the constructor reverts with `abi.encode(getSnapshot's return values)`.
///
/// LiquidityBookV21Lens for a v2.0 pair, with the same walk and the same result: from
/// `startId` towards higher ids, reading each bin it lands on, paging by bin id. What differs is
/// how the walk ends. v2.0's `findFirstNonEmptyBinId` has no sentinel: with nowhere to go it
/// reverts TreeMath__ErrorDepthSearch(), and only that revert ends the walk (see LBPairV20). Any
/// other failure ends the call, so a walk never stops short with bins left that it reports as all.
///
/// The reply checks itself: `getReservesAndId()` is the sum of every bin's reserves, fees left out
/// of both, so over a whole walk the bins' reserves add up to `reserveX` and `reserveY`.
contract LiquidityBookV20Lens {
    /// @notice A non-empty bin and the reserves a swap simulator needs.
    struct PopulatedBin {
        uint24 id;
        uint128 reserveX;
        uint128 reserveY;
    }

    /// @notice Gas kept back for one more bin and for returning the result.
    /// @dev A bin costs two calls to the pair, each through its EIP-1167 proxy: `getBin`, one cold
    /// slot, and `findFirstNonEmptyBinId`, whose tree search reads up to two cold slots per level of
    /// a three-level tree. That is ~20k gas at most, of which a call may forward only 63/64.
    /// Returning costs a few hundred: the result is already encoded. 60k covers both several times
    /// over, and is nothing next to any node's eth_call cap. That is at Ethereum's prices: on a chain
    /// whose first read of the pair costs more than FIRST_READ, the lens keeps back that much more.
    uint256 internal constant GAS_RESERVE = 60_000;

    /// @dev The most the pair's first read, `getReservesAndId` with the pair, its proxy's
    /// implementation and their slots cold, costs where storage is priced as on Ethereum: 9,983 gas
    /// on every pair measured.
    uint256 internal constant FIRST_READ = 11_000;

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
    /// @param pair The Liquidity Book v2.0 pair to walk.
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
        (uint128 reserveX, uint128 reserveY, uint24 activeId) = LBPairV20.reservesAndId(pair);
        uint256 gasReserve = GasPrice.scale(GAS_RESERVE, before - gasleft(), FIRST_READ);

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
            (uint128 binReserveX, uint128 binReserveY) = LBPairV20.bin(pair, cursor);
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

            (uint24 next, bool found) = LBPairV20.nextNonEmptyBinAbove(pair, cursor);
            // A step that does not move up is no step, whatever the pair answered.
            if (!found || next <= cursor) {
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
