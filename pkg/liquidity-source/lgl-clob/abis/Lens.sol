// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

/// @title Lens
/// @notice A deployless helper that reads a Hanji pool's state in one eth_call, its order book and what its market
/// maker's quote costs. Its creation code runs as a to-less eth_call and reverts with the state, never deployed.
/// @dev Written in assembly, as the code goes out with every call and the state comes back with it. The code is
/// followed by abi.encode(address pool, address helper, uint24 maxPriceLevels, bool measure).
///
/// The state is packed, in this order, the tag "Lens", the block number in 8 bytes, 1 when the pool has a market
/// maker, then for a sell at one level, a sell at two, a buy at one and a buy at two the quote's gas in 4 bytes and
/// the levels it quoted in 1, zero for a quote not made or that failed, then the bids and the asks, each a byte
/// counting its levels followed by every level's price in 9 bytes and shares in 16, then a byte, 1 when the pool
/// quotes its market maker through a quoter, followed by what the market maker quotes and then what the quoter quotes,
/// each of the bids and then the asks for any quantity, as a side is packed or the byte 0xff alone for a quote that
/// failed, and 0 followed by nothing for a pool without one. A revert without data means the book could not be read.
///
/// The book shows the market maker's own quote. Where a pool with a quoter has the quoter price the levels the market
/// maker posts for an order, an order fills what the quoter quotes, and elsewhere it fills the book. The quotes are
/// read last, in place.
///
/// The book is read in a probe that reverts, so the storage it warms is cold again for the market maker's quotes,
/// as it is for a real order. Each side is quoted at half its first level, and at its first level and half its
/// second. Without measure, the market maker is neither looked up nor quoted, and as nothing then needs cold storage
/// the book is read in place rather than in a probe.
contract Lens {
    constructor() payable {
        bytes memory probe = type(Probe).creationCode;
        assembly {
            let code := add(probe, 32)
            let codeLen := mload(probe)
            let probeArgs := add(code, codeLen)
            let args := add(probeArgs, 160)
            codecopy(args, sub(codesize(), 128), 128)
            let pool := mload(args)
            let helper := mload(add(args, 32))
            if iszero(helper) { revert(0, 0) }

            // Packs the bids and then the asks the helper reads of lob from dst, each a byte counting its levels
            // followed by every level's price in 9 bytes and shares in 16, and returns their length. buf holds each
            // call. The helper answers abi.encode(uint72[] prices, uint128[] shares).
            function readBook(h, lob, maxLevels, buf, dst) -> length {
                let cursor := dst
                for { let isAsk := 0 } lt(isAsk, 2) { isAsk := add(isAsk, 1) } {
                    mstore(buf, shl(224, 0x1c5f5589))
                    mstore(add(buf, 4), lob)
                    mstore(add(buf, 36), isAsk)
                    mstore(add(buf, 68), maxLevels)
                    if iszero(staticcall(gas(), h, buf, 100, 0, 0)) { revert(0, 0) }
                    returndatacopy(0, 0, 64)
                    let prices := mload(0)
                    let shares := mload(32)
                    returndatacopy(0, prices, 32)
                    let n := mload(0)
                    mstore8(cursor, n)
                    cursor := add(cursor, 1)
                    for { let i := 0 } lt(i, n) { i := add(i, 1) } {
                        let at := add(32, mul(i, 32))
                        returndatacopy(0, add(prices, at), 32)
                        mstore(cursor, shl(184, mload(0)))
                        returndatacopy(0, add(shares, at), 32)
                        mstore(add(cursor, 9), shl(128, mload(0)))
                        cursor := add(cursor, 25)
                    }
                }
                length := sub(cursor, dst)
            }

            // The word target answers selector with, and whether it answered one.
            function word(target, selector) -> ok, value {
                mstore(0, shl(224, selector))
                ok := staticcall(gas(), target, 0, 4, 0, 32)
                ok := and(ok, eq(returndatasize(), 32))
                value := mload(0)
            }

            // Packs at cursor the levels target quotes for a side of market at any quantity, as readBook packs a side,
            // or the byte 0xff alone when the quote fails or answers otherwise, and returns where it ended. The quote
            // is called from past cursor, and answers abi.encode(uint256[] prices, uint256[] shares).
            function packQuote(target, market, buy, cursor) -> next {
                let buf := add(cursor, 32)
                mstore(buf, shl(224, 0x24675a09))
                mstore(add(buf, 4), market)
                mstore(add(buf, 36), buy)
                mstore(add(buf, 68), 0xffffffffffffffffffffffffffffffff)
                mstore(add(buf, 100), 0xffffffffffffffffffffffffffffffff)
                mstore(add(buf, 132), 1)
                if buy { mstore(add(buf, 132), 0xffffffffffffffffff) }
                mstore8(cursor, 0xff)
                next := add(cursor, 1)
                if iszero(staticcall(gas(), target, buf, 164, 0, 0)) { leave }
                let size := returndatasize()
                if lt(size, 128) { leave }
                returndatacopy(0, 0, 64)
                let prices := mload(0)
                let shares := mload(32)
                if or(gt(prices, sub(size, 32)), gt(shares, sub(size, 32))) { leave }
                returndatacopy(0, prices, 32)
                let n := mload(0)
                returndatacopy(0, shares, 32)
                if or(gt(n, 254), iszero(eq(n, mload(0)))) { leave }
                let end := mul(add(n, 1), 32)
                if or(gt(add(prices, end), size), gt(add(shares, end), size)) { leave }
                mstore8(cursor, n)
                for { let i := 0 } lt(i, n) { i := add(i, 1) } {
                    let at := add(32, mul(i, 32))
                    returndatacopy(0, add(prices, at), 32)
                    mstore(next, shl(184, mload(0)))
                    returndatacopy(0, add(shares, at), 32)
                    mstore(add(next, 9), shl(128, mload(0)))
                    next := add(next, 25)
                }
            }

            // Packs at cursor whether pool quotes its market maker through a quoter and, when it does, what each of
            // them quotes, and returns where it ended.
            function readQuotes(p, cursor) -> next {
                mstore8(cursor, 0)
                next := add(cursor, 1)
                let quoterOk, quoter := word(p, 0xf8bed25f)
                if iszero(and(quoterOk, iszero(iszero(quoter)))) { leave }
                let marketOk, market := word(p, 0xb6903418)
                let makerOk, maker := word(p, 0x938c5132)
                if iszero(and(marketOk, makerOk)) { leave }
                mstore8(cursor, 1)
                next := packQuote(maker, market, 0, next)
                next := packQuote(maker, market, 1, next)
                next := packQuote(quoter, market, 0, next)
                next := packQuote(quoter, market, 1, next)
            }

            // Runs the probe with its five arguments, and leaves what it reverted with as return data.
            function run(c, cLen, at, a0, a1, a2, a3, a4) {
                mstore(at, a0)
                mstore(add(at, 32), a1)
                mstore(add(at, 64), a2)
                mstore(add(at, 96), a3)
                mstore(add(at, 128), a4)
                pop(create(0, c, add(cLen, 160)))
            }

            let out := add(args, 128)
            mstore(out, shl(224, 0x4c656e73))
            mstore(add(out, 4), shl(192, number()))
            mstore8(add(out, 12), 0)
            // The four quotes, zero until made.
            mstore(add(out, 13), 0)

            if iszero(mload(add(args, 96))) {
                // The arguments are read, so their memory holds the helper's calls.
                let bookLen := readBook(helper, pool, mload(add(args, 64)), args, add(out, 33))
                revert(out, sub(readQuotes(pool, add(add(out, 33), bookLen)), out))
            }

            run(code, codeLen, probeArgs, pool, helper, mload(add(args, 64)), 0, 0)
            let bookLen := returndatasize()
            if iszero(bookLen) { revert(0, 0) }
            let book := add(out, add(64, bookLen))
            returndatacopy(book, 0, bookLen)

            {
                // Yul evaluates arguments right to left, so each answer's size is read in a statement of its own.
                mstore(0, shl(224, 0xb6903418))
                let marketOk := staticcall(gas(), pool, 0, 4, 0, 32)
                marketOk := and(marketOk, eq(returndatasize(), 32))
                let market := mload(0)
                mstore(0, shl(224, 0x938c5132))
                let makerOk := staticcall(gas(), pool, 0, 4, 0, 32)
                makerOk := and(makerOk, eq(returndatasize(), 32))
                let maker := mload(0)
                if and(marketOk, makerOk) {
                    mstore8(add(out, 12), 1)

                    // Quotes a side at half its first level, and at its first and half its second, writing each
                    // quote's gas and levels from slot.
                    function quoteSide(c, cLen, at, mm, m, buy, side, slot) {
                        let n := byte(0, mload(side))
                        if iszero(n) { leave }
                        let first := shr(128, mload(add(side, 10)))
                        if iszero(first) { leave }
                        quote(c, cLen, at, mm, m, buy, half(first), slot)
                        if gt(n, 1) {
                            let second := shr(128, mload(add(side, 35)))
                            if second { quote(c, cLen, at, mm, m, buy, add(first, half(second)), add(slot, 5)) }
                        }
                    }

                    function half(x) -> h {
                        h := x
                        if gt(x, 1) { h := shr(1, x) }
                    }

                    function quote(c, cLen, at, mm, m, buy, quantity, slot) {
                        run(c, cLen, at, mm, 0, m, buy, quantity)
                        if eq(returndatasize(), 64) {
                            returndatacopy(0, 0, 64)
                            mstore(slot, or(shl(224, mload(0)), shl(216, mload(32))))
                        }
                    }

                    let asks := add(book, add(1, mul(25, byte(0, mload(book)))))
                    quoteSide(code, codeLen, probeArgs, maker, market, 0, book, add(out, 13))
                    quoteSide(code, codeLen, probeArgs, maker, market, 1, asks, add(out, 23))
                }
            }

            // The book follows the header, copied last as the header's 32-byte writes reach past it.
            for { let i := 0 } lt(i, bookLen) { i := add(i, 32) } { mstore(add(add(out, 33), i), mload(add(book, i))) }
            revert(out, sub(readQuotes(pool, add(add(out, 33), bookLen)), out))
        }
    }
}

/// @notice Reads a pool's book, or runs one market maker quote, and reverts with what it found, so the storage it
/// warmed is cold again. Its code is followed by five words, the target, the helper, then maxPriceLevels for a book
/// read, or for a quote a zero helper, the market, whether it buys and the quantity.
contract Probe {
    constructor() payable {
        assembly {
            let args := 0x80
            codecopy(args, sub(codesize(), 160), 160)
            let target := mload(args)
            let helper := mload(add(args, 32))
            let buf := add(args, 160)

            if helper {
                // The bids and then the asks, each a byte counting its levels followed by every level's price in 9
                // bytes and shares in 16. The helper answers abi.encode(uint72[] prices, uint128[] shares).
                let out := add(buf, 128)
                let cursor := out
                for { let isAsk := 0 } lt(isAsk, 2) { isAsk := add(isAsk, 1) } {
                    mstore(buf, shl(224, 0x1c5f5589))
                    mstore(add(buf, 4), target)
                    mstore(add(buf, 36), isAsk)
                    mstore(add(buf, 68), mload(add(args, 64)))
                    if iszero(staticcall(gas(), helper, buf, 100, 0, 0)) { revert(0, 0) }
                    returndatacopy(0, 0, 64)
                    let prices := mload(0)
                    let shares := mload(32)
                    returndatacopy(0, prices, 32)
                    let n := mload(0)
                    mstore8(cursor, n)
                    cursor := add(cursor, 1)
                    for { let i := 0 } lt(i, n) { i := add(i, 1) } {
                        let at := add(32, mul(i, 32))
                        returndatacopy(0, add(prices, at), 32)
                        mstore(cursor, shl(184, mload(0)))
                        returndatacopy(0, add(shares, at), 32)
                        mstore(add(cursor, 9), shl(128, mload(0)))
                        cursor := add(cursor, 25)
                    }
                }
                revert(out, sub(cursor, out))
            }

            // A sell takes any price above zero, a buy any price a level can hold.
            let buy := mload(add(args, 96))
            let quote := buf
            mstore(quote, 164)
            mstore(add(quote, 32), shl(224, 0x24675a09))
            mstore(add(quote, 36), mload(add(args, 64)))
            mstore(add(quote, 68), buy)
            mstore(add(quote, 100), mload(add(args, 128)))
            mstore(add(quote, 132), 0xffffffffffffffffffffffffffffffff)
            mstore(add(quote, 164), 1)
            if buy { mstore(add(quote, 164), 0xffffffffffffffffff) }

            // Only the call is measured.
            let before := gas()
            let ok := staticcall(gas(), target, add(quote, 32), mload(quote), 0, 0)
            let used := sub(before, gas())

            // The quote answers (uint256[] prices, uint256[] shares), and each price is a level.
            let levels := 0
            if and(ok, gt(returndatasize(), 127)) {
                returndatacopy(0, 0, 32)
                let offset := mload(0)
                if lt(offset, sub(returndatasize(), 31)) {
                    returndatacopy(0, offset, 32)
                    levels := mload(0)
                }
            }
            mstore(0, mul(ok, used))
            mstore(32, levels)
            revert(0, 64)
        }
    }
}
