// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import {CoreE2EBase, EPool, EPoolAdmin, EFeed, EAggregator} from "test/kyber/CoreE2EBase.sol";

/// @notice Core end-to-end preview grids on Base forks: per block and per scenario, one complete state (views only,
///         see CoreE2EBase) and pool.previewSwap / pool.previewLever over a log-spaced grid in both directions,
///         the degenerate amounts, and every class boundary the grid brackets (bisected to one unit, then scanned
///         unit by unit over a window on both sides: the accepted set is not an interval near a band edge).
///         Scenarios arm the leverage venue (curator setLevPaused(false), keeper setSpread), let the spread go
///         stale, cap the notional, tighten the band, clip / floor the fee, pause, clear feature bits, age or
///         break the feed (mocked aggregator / sequencer answers), take the IRM down inside and past the account's
///         grace, and let interest accrue. Rows land in kyber-out/core_e2e/grid_<block>.jsonl.
contract CoreE2EGridTest is CoreE2EBase {
    uint256 constant B1 = 51_302_915;
    uint256 constant B2 = 51_313_000;
    uint256 constant B3 = 51_324_800;
    uint256 constant W = 600;

    // ------------------------------------------------------------------ amount grids
    function _logGrid(uint256 decades) internal pure returns (uint256[] memory a) {
        uint16[8] memory m = [uint16(100), 133, 178, 237, 316, 421, 562, 750];
        uint256[] memory tmp = new uint256[](decades * 8 + 1);
        uint256 n;
        uint256 base = 1;
        for (uint256 e; e < decades; ++e) {
            for (uint256 j; j < 8; ++j) {
                uint256 v = base * m[j] / 100;
                if (v != 0 && (n == 0 || v > tmp[n - 1])) tmp[n++] = v;
            }
            base *= 10;
        }
        tmp[n++] = base;
        a = new uint256[](n);
        for (uint256 i; i < n; ++i) a[i] = tmp[i];
    }

    function _huge() internal pure returns (uint256[8] memory) {
        return [uint256(0), 1 << 64, 1 << 100, 1 << 128, 1 << 160, 1 << 200, 1 << 255, type(uint256).max];
    }

    function _class(bool lever, bool dir, uint256 amount) internal view returns (bytes4) {
        (bool ok, bytes memory ret) = POOL.staticcall(
            lever ? abi.encodeCall(EPool.previewLever, (dir, amount)) : abi.encodeCall(EPool.previewSwap, (dir, amount))
        );
        if (ok) return bytes4(0);
        if (ret.length < 4) return bytes4(0xffffffff);
        return bytes4(ret);
    }

    function _rec(string memory tag, bool lever, bool dir, uint256 amount) internal {
        if (lever) _pLever(tag, dir, amount);
        else _pSwap(tag, dir, amount);
    }

    /// @dev The grid, the degenerate amounts, and (windows) every class boundary scanned unit by unit.
    function _grid(string memory tag, bool lever, bool dir, uint256 decades, bool windows) internal {
        uint256[8] memory h = _huge();
        for (uint256 i; i < h.length; ++i) _rec(tag, lever, dir, h[i]);
        uint256[] memory a = _logGrid(decades);
        bytes4[] memory c = new bytes4[](a.length);
        for (uint256 i; i < a.length; ++i) {
            _rec(tag, lever, dir, a[i]);
            c[i] = _class(lever, dir, a[i]);
        }
        if (!windows) return;
        uint256 lastHi;
        for (uint256 i = 1; i < a.length; ++i) {
            if (c[i] == c[i - 1]) continue;
            (uint256 lo, uint256 hi) = (a[i - 1], a[i]);
            bytes4 cl = c[i - 1];
            while (hi - lo > 1) {
                uint256 mid = (lo + hi) / 2;
                if (_class(lever, dir, mid) == cl) lo = mid;
                else hi = mid;
            }
            uint256 from = lo > W ? lo - W : 1;
            if (from <= lastHi) from = lastHi + 1;
            uint256 to = hi + W;
            for (uint256 x = from; x <= to; ++x) _rec(tag, lever, dir, x);
            lastHi = to;
        }
    }

    function _swapGrids(string memory tag, bool windows) internal {
        _grid(tag, false, true, 12, windows);
        _grid(tag, false, false, 14, windows);
    }

    function _leverGrids(string memory tag, bool windows) internal {
        _grid(tag, true, true, 12, windows);
        _grid(tag, true, false, 14, windows);
    }

    // ------------------------------------------------------------------ scenario setup
    /// @dev Re-answer an aggregator at `answer`, observed `age` seconds ago (a mocked latestRoundData).
    /// @dev Back to the block's state: snapshots do not undo mocked calls, so those are cleared too.
    function _scenario(string memory tag) internal {
        _state(tag);
    }

    // ------------------------------------------------------------------ the block
    function _run(uint256 blockNumber, bool full) internal {
        vm.createSelectFork(RPC, blockNumber);
        _out = string.concat("kyber-out/core_e2e/grid_", vm.toString(blockNumber), ".jsonl");
        if (vm.exists(_out)) vm.removeFile(_out);
        uint256 snap = vm.snapshotState();
        (uint64 band, uint64 floorWad, uint256 maxN, uint256 reserve) = _liveLoanCfg();

        _scenario("live");
        _swapGrids("live", true);
        _leverGrids("live", false);

        _reset(snap);
        _arm(17_500);
        _scenario("armed");
        _leverGrids("armed", true);

        _reset(snap);
        _arm(90_000);
        _scenario("armed_hi");
        _leverGrids("armed_hi", false);

        _reset(snap);
        _arm(0);
        _scenario("stale_spread");
        _grid("stale_spread", true, true, 12, false);
        _grid("stale_spread", true, false, 14, full);

        _reset(snap);
        _loanCfg(band, floorWad, 5e6, reserve);
        _scenario("capped");
        _swapGrids("capped", full);

        _reset(snap);
        _loanCfg(1e16, floorWad, maxN, reserve);
        _arm(17_500);
        _scenario("band_tight");
        _swapGrids("band_tight", false);
        _leverGrids("band_tight", false);

        _reset(snap);
        vm.prank(_curator());
        EPoolAdmin(POOL).setFeeBounds(0, 1e16);
        _scenario("fee_cap");
        _swapGrids("fee_cap", false);

        _reset(snap);
        vm.prank(_curator());
        EPoolAdmin(POOL).setFeeBounds(5e16, 1e17);
        _scenario("fee_floor");
        _swapGrids("fee_floor", false);

        _reset(snap);
        _loanCfg(band, 2e16, maxN, reserve);
        _scenario("fee_floor_loan");
        _swapGrids("fee_floor_loan", false);

        _reset(snap);
        _arm(17_500);
        vm.prank(_curator());
        EPoolAdmin(POOL).setPaused(true);
        _scenario("paused");
        _swapGrids("paused", false);
        _leverGrids("paused", false);

        _features(snap, "no_sell", 63 & ~uint256(2));
        _features(snap, "no_buy", 63 & ~uint256(4));
        _features(snap, "no_lev", 63 & ~uint256(32));

        _reset(snap);
        {
            (,,, uint256 ts,) = EAggregator(_agg(CBBTC)).latestRoundData();
            vm.warp(ts + 3601);
        }
        _arm(17_500);
        _scenario("stale_feed");
        _swapGrids("stale_feed", false);
        _leverGrids("stale_feed", false);

        _sequencer(snap, "seq_grace", 0, 100);
        _sequencer(snap, "seq_down", 1, 100_000);

        _reset(snap);
        _freshFeeds(5);
        _mockRound(_agg(USDC), 97_000_000, 5);
        _arm(17_500);
        _scenario("peg_broken");
        _swapGrids("peg_broken", false);
        _leverGrids("peg_broken", false);

        _reset(snap);
        _irmDown();
        vm.warp(block.timestamp + 1_800);
        _freshFeeds(7);
        _scenario("irm_grace");
        _swapGrids("irm_grace", false);

        _reset(snap);
        _irmDown();
        vm.warp(block.timestamp + 7_200);
        _freshFeeds(9);
        _arm(17_500);
        _scenario("irm_quarantine");
        _swapGrids("irm_quarantine", false);
        _leverGrids("irm_quarantine", false);

        // The Router pin stored below the pool's ltv (governance keeps them equal): the sell's funding binds on
        // collateral before the gate room, so a partial fill re-plans over several funding passes.
        _reset(snap);
        _storePin(0.2e18);
        _scenario("pin_low");
        _grid("pin_low", false, true, 12, true);

        // The pool's ltv dial stored below its standing exposure: no credit-zero room and no sell room.
        _reset(snap);
        _storeLtv(0.05e18);
        _arm(17_500);
        _scenario("ltv_low");
        _swapGrids("ltv_low", false);
        _leverGrids("ltv_low", false);

        _feedMove(snap, "btc_down10", 90);
        _feedMove(snap, "btc_down25", 75);
        _feedMove(snap, "btc_up10", 110);

        _reset(snap);
        _mockRound(_agg(CBBTC), 0, 5);
        _arm(17_500);
        _scenario("feed_invalid");
        _grid("feed_invalid", false, true, 6, false);
        _grid("feed_invalid", false, false, 8, false);
        _grid("feed_invalid", true, true, 6, false);
        _grid("feed_invalid", true, false, 8, false);

        _reset(snap);
        vm.warp(block.timestamp + 30 days);
        _freshFeeds(11);
        _arm(17_500);
        _scenario("accrued");
        _swapGrids("accrued", full);
        _leverGrids("accrued", false);
    }

    /// @dev The cbBTC USD answer moved to `pct` percent (both rounds fresh), leverage armed.
    function _feedMove(uint256 snap, string memory tag, int256 pct) internal {
        _reset(snap);
        address a = _agg(CBBTC);
        int256 x = _answer(a);
        _freshFeeds(4);
        _mockRound(a, x * pct / 100, 4);
        _arm(17_500);
        _scenario(tag);
        _swapGrids(tag, false);
        _leverGrids(tag, false);
    }

    /// @dev FLAMMStore slot base + 13: phiWad (bits 0..63), phiMinWad, phiMaxWad, ltvWad (192..255).
    function _storeLtv(uint64 ltvWad) internal {
        bytes32 slot = bytes32(FLAMM_STORE + 13);
        uint256 w = uint256(vm.load(POOL, slot));
        require(uint64(w >> 192) == EPool(POOL).dials().ltvWad, "slot:ltv");
        w = (w & ~(uint256(type(uint64).max) << 192)) | (uint256(ltvWad) << 192);
        vm.store(POOL, slot, bytes32(w));
        require(EPool(POOL).dials().ltvWad == ltvWad, "slot:ltv write");
    }

    function _features(uint256 snap, string memory tag, uint256 bitmap) internal {
        _reset(snap);
        _arm(17_500);
        vm.prank(_curator());
        EPoolAdmin(POOL).setFeatures(bitmap);
        _scenario(tag);
        _grid(tag, false, true, 12, false);
        _grid(tag, false, false, 14, false);
        _grid(tag, true, true, 6, false);
        _grid(tag, true, false, 8, false);
    }

    function _sequencer(uint256 snap, string memory tag, int256 answer, uint256 startedAgo) internal {
        _reset(snap);
        _arm(17_500);
        address seq = EFeed(FEED).SEQUENCER_FEED();
        (uint80 id,,,,) = EAggregator(seq).latestRoundData();
        uint256 st = block.timestamp - startedAgo;
        vm.mockCall(seq, abi.encodeWithSelector(SEL_ROUND), abi.encode(id, answer, st, st, id));
        _scenario(tag);
        _grid(tag, false, true, 6, false);
        _grid(tag, false, false, 8, false);
        _grid(tag, true, true, 6, false);
        _grid(tag, true, false, 8, false);
    }

    function test_grid_b1() public {
        _run(B1, true);
    }

    function test_grid_b2() public {
        _run(B2, false);
    }

    function test_grid_b3() public {
        _run(B3, false);
    }
}
