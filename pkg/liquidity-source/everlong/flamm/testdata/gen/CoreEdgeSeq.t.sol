// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import {Vm} from "forge-std/Vm.sol";
import {CoreEdgesBase, VPool, VSpread, VRouter, VMorpho, VIrm, VMarketParams, VMarket, VERC20, VAgg} from "test/kyber/CoreEdgesBase.sol";

/// @notice Edge EXECUTED sequences: a dealt taker calls pool.swap / leverUp / leverDown (deadline = now)
///         interleaved with warps (heartbeat, sequencer grace, spread age and IRM grace crossings to the second),
///         curator / keeper moves, feed / sequencer / IRM / oracle mocks and third-party Morpho activity. Rows:
///         begin (dump), pv (preview at the current state), x (execution: return words or revert, pool events, post
///         dump), warp (post dump; nothing but time moved), re (a tracker re-read after a non-core move, with the
///         move's name). Output: kyber-out/core_edges/seq_<block>.jsonl.
contract CoreEdgeSeqTest is CoreEdgesBase {
    address taker;
    string seq;
    uint256 stepNo;
    uint256 base;
    uint256 seedBase;

    bytes32 constant SWAP_T = keccak256("Swap(address,address,bool,uint256,uint256,uint256,uint256,uint256)");
    bytes32 constant UP_T = keccak256("LeverUp(address,address,uint256,uint256,uint256,uint256)");
    bytes32 constant DOWN_T = keccak256("LeverDown(address,address,uint256,uint256,uint256,uint256)");

    // ------------------------------------------------------------------ recording
    function _begin(string memory name) internal {
        vm.revertToState(base);
        vm.clearMockedCalls();
        seq = name;
        stepNo = 0;
        rng = uint256(keccak256(abi.encode(name, block.number, seedBase)));
        _emit(string.concat('{"k":"begin","seq":"', name, '","s":', _dump(), "}"));
    }

    function _head(string memory k) internal returns (string memory) {
        return string.concat('{"k":"', k, '","seq":"', seq, '","i":', _u(stepNo++), ',"ts":', _u(block.timestamp));
    }

    function _logs() internal returns (string memory s) {
        Vm.Log[] memory logs = vm.getRecordedLogs();
        s = "[";
        bool first = true;
        for (uint256 i; i < logs.length; ++i) {
            if (logs[i].emitter != POOL || logs[i].topics.length == 0) continue;
            bytes32 t0 = logs[i].topics[0];
            if (t0 != SWAP_T && t0 != UP_T && t0 != DOWN_T) continue;
            s = string.concat(s, first ? "" : ",", '{"t":"', t0 == SWAP_T ? "Swap" : t0 == UP_T ? "LeverUp" : "LeverDown", '","data":', _hex(logs[i].data), "}");
            first = false;
        }
        s = string.concat(s, "]");
    }

    function _pv(bool lever, bool dir, uint256 a) internal {
        (bool ok, bytes memory ret) = _probe(lever, dir, a);
        _emit(
            string.concat(
                _head("pv"), ',"lever":', _b(lever), ',"d":', dir ? "1" : "0", ',"a":', _u(a),
                ok ? ',"r":' : ',"e":', _hex(ret), "}"
            )
        );
    }

    function _x(bool lever, bool dir, uint256 a, uint256 minOut) internal returns (bool ok) {
        return _xd(lever, dir, a, minOut, 0);
    }

    function _xd(bool lever, bool dir, uint256 a, uint256 minOut, uint256 late) internal returns (bool ok) {
        _pv(lever, dir, a);
        uint256 dl = block.timestamp - late;
        vm.recordLogs();
        bytes memory data;
        if (!lever) {
            (address tin, address tout) = dir ? (CBBTC, USDC) : (USDC, CBBTC);
            data = abi.encodeCall(VPool.swap, (tin, tout, a, minOut, taker, dl));
        } else if (dir) {
            data = abi.encodeCall(VPool.leverUp, (a, minOut, taker, dl));
        } else {
            data = abi.encodeCall(VPool.leverDown, (a, minOut, taker, dl));
        }
        vm.prank(taker);
        bytes memory ret;
        (ok, ret) = POOL.call(data);
        string memory ev = _logs();
        _emit(
            string.concat(
                _head("x"), ',"lever":', _b(lever), ',"d":', dir ? "1" : "0", ',"a":', _u(a), ',"m":', _u(minOut), ',"late":', _u(late),
                string.concat(ok ? ',"r":' : ',"e":', _hex(ret), ',"ev":', ev, ',"post":', _dump(), "}")
            )
        );
    }

    function _warp(uint256 dt) internal {
        vm.warp(block.timestamp + dt);
        _emit(string.concat(_head("warp"), ',"dt":', _u(dt), ',"post":', _dump(), "}"));
    }

    function _re(string memory op, bool ok) internal {
        _emit(string.concat(_head("re"), ',"op":"', op, '","ok":', _b(ok), ',"post":', _dump(), "}"));
    }

    // ------------------------------------------------------------------ moves
    function _cur(string memory op, bytes memory data) internal returns (bool ok) {
        address c = _curator();
        vm.prank(c);
        (ok,) = POOL.call(data);
        _re(op, ok);
    }

    function _spread(uint24 v) internal returns (bool ok) {
        address k = _keeper();
        vm.prank(k);
        (ok,) = SPREAD.call(abi.encodeCall(VSpread.setSpread, (v)));
        _re("setSpread", ok);
    }

    function _maxAge(uint32 v) internal {
        address c = _curator();
        vm.prank(c);
        (bool ok,) = SPREAD.call(abi.encodeCall(VSpread.setMaxSpreadAge, (v)));
        _re("setMaxSpreadAge", ok);
    }

    function _fresh(uint256 age) internal {
        address a = _aggOf(CBBTC);
        address b = _aggOf(USDC);
        (uint80 ia, int256 xa,) = _answerOf(a);
        (uint80 ib, int256 xb,) = _answerOf(b);
        _mockRound(a, ia, xa, block.timestamp - age, block.timestamp - age);
        _mockRound(b, ib, xb, block.timestamp - age, block.timestamp - age);
        _re("feeds", true);
    }

    function _setFeeds(int256 cb, uint256 cbAge, int256 us, uint256 usAge) internal {
        address a = _aggOf(CBBTC);
        address b = _aggOf(USDC);
        (uint80 ia,,) = _answerOf(a);
        (uint80 ib,,) = _answerOf(b);
        _mockRound(a, ia, cb, block.timestamp - cbAge, block.timestamp - cbAge);
        _mockRound(b, ib, us, block.timestamp - usAge, block.timestamp - usAge);
        _re("feeds", true);
    }

    function _arm(uint24 s) internal {
        _cur("setLevPaused", abi.encodeCall(VPool.setLevPaused, (false)));
        _spread(s);
    }

    function _band(uint64 band, uint64 fl, uint256 mx, uint256 rt) internal returns (bool) {
        return _cur("setLoanConfig", abi.encodeCall(VPool.setLoanConfig, (0, band, fl, mx, rt)));
    }

    /// @dev An accepted amount at the top of the accepted region: the largest accepted point of a geometric scan
    ///      of [lo, hi], bisected up to the next refusal (the accepted set need not be an interval).
    function _edge(bool lever, bool dir, uint256 lo, uint256 hi) internal view returns (uint256) {
        uint256[] memory pts = _geo(lo, hi, 36);
        uint256 j = type(uint256).max;
        for (uint256 k; k < pts.length; ++k) {
            (bool ok,) = _probe(lever, dir, pts[k]);
            if (ok) j = k;
        }
        if (j == type(uint256).max) return 0;
        if (j == pts.length - 1) return pts[j];
        uint256 a = pts[j];
        uint256 b = pts[j + 1];
        while (b - a > 1) {
            uint256 mid = a + (b - a) / 2;
            (bool ok,) = _probe(lever, dir, mid);
            if (ok) a = mid;
            else b = mid;
        }
        return a;
    }

    function _xEdge(bool lever, bool dir, uint256 lo, uint256 hi) internal returns (bool) {
        uint256 e = _edge(lever, dir, lo, hi);
        if (e == 0) return _x(lever, dir, lo + _next() % (hi - lo), 0);
        if (_next() % 3 == 0) _x(lever, dir, e + 1, 0); // just past the edge (may still fill)
        return _x(lever, dir, e, 0);
    }

    function _debt() internal view returns (uint256) {
        (,, uint256 d) = _loanPos();
        return d;
    }

    function _loanPos() internal view returns (uint256, uint256, uint256) {
        (bool ok, bytes memory ret) = POOL.staticcall(abi.encodeWithSignature("loanPosition()"));
        if (!ok) return (0, 0, 0);
        return abi.decode(ret, (uint256, uint256, uint256));
    }

    // ------------------------------------------------------------------ named sequences
    function _seqHeartbeat() internal {
        _begin("heartbeat");
        _arm(21_000);
        _fresh(3000);
        _x(false, true, 40_000, 0);
        _x(true, true, 20_000, 0);
        _warp(600); // age 3600: the last fresh second
        _x(false, true, 30_000, 0);
        _x(true, false, 5e6, 0);
        _x(false, false, 20e6, 0);
        _warp(1); // age 3601: stale everywhere
        _x(false, true, 30_000, 0);
        _x(false, false, 20e6, 0);
        _x(true, false, 5e6, 0);
        _x(true, true, 20_000, 0);
        _fresh(0);
        _x(true, true, 20_000, 0); // spread posted 601 s ago, still live
        _x(false, false, 30e6, 0);
        _warp(89_999);
        _setFeeds(7_700_000_000_000, 3599, 100_000_000, 90_000);
        _x(false, false, 10e6, 0);
        _warp(1); // USDC 90_001 s old
        _x(false, false, 10e6, 0);
        _x(false, true, 10_000, 0);
    }

    function _seqSequencer() internal {
        _begin("sequencer");
        _arm(9_000);
        _fresh(0);
        address sq = abi.decode(_sq(), (address));
        (uint80 id,,,,) = VAgg(sq).latestRoundData();
        _mockRound(sq, id, 0, block.timestamp - 3000, block.timestamp - 3000);
        _re("sequencer", true);
        _x(false, true, 20_000, 0);
        _x(false, false, 10e6, 0);
        _warp(600); // exactly SEQUENCER_GRACE: still grace
        _x(false, true, 20_000, 0);
        _x(true, false, 3e6, 0);
        _warp(1);
        _spread(9_000);
        _x(false, true, 20_000, 0);
        _x(true, true, 10_000, 0);
        _x(true, false, 3e6, 0);
        _mockRound(sq, id, 1, block.timestamp - 99_999, block.timestamp);
        _re("sequencer", true);
        _x(false, false, 10e6, 0);
        _x(true, false, 3e6, 0);
    }

    function _sq() internal view returns (bytes memory ret) {
        (, ret) = FEED.staticcall(abi.encodeWithSignature("SEQUENCER_FEED()"));
    }

    function _seqSpread() internal {
        _begin("spread_degrade");
        _arm(18_000);
        _x(true, true, 30_000, 0); // stores 18_000
        _xEdge(true, true, 1_000, 3e6);
        _warp(3599 - 0);
        _fresh(0);
        _x(true, true, 10_000, 0); // lastSetTs + 3599 + (feeds re-read) : live
        _warp(1); // lastSetTs + 3600: live (<=)
        _x(true, true, 10_000, 0);
        _warp(1); // stale
        _x(true, true, 10_000, 0);
        _x(true, false, 4e6, 0); // degrade at 18_000
        _xEdge(true, false, 1e6, 3e9);
        _spread(95_000); // live post above the band: clamp 80_000, stored on a live fill
        _x(true, false, 2e6, 0);
        _x(true, true, 5_000, 0);
        _warp(3601);
        _fresh(0);
        _x(true, false, 2e6, 0); // degrade at the stored 80_000
        _x(true, false, 2e6, type(uint256).max); // Slippage
        _band(1.5e15, 0, 1e12, 0); // band 0.15%: live ceiling 1_500; degrade value ignores it
        _x(true, false, 2e6, 0);
        _spread(2_500);
        _x(true, true, 5_000, 0);
        _x(true, false, 2e6, 0);
        _maxAge(0);
        _warp(100_000);
        _fresh(0);
        _x(true, false, 2e6, 0);
        _x(true, true, 5_000, 0);
    }

    function _seqLtvWalk() internal {
        _begin("ltv_walk");
        _arm(12_000);
        _band(0.1e18, 0, 1e12, 0);
        uint64[6] memory ltvs = [uint64(0.45e18), 0.35e18, 0.25e18, 0.15e18, 0.1e18, 0.2e18];
        uint64[6] memory phis = [uint64(1e18), 0.9e18, 0.5e18, 0.6e18, 0.5e18, 1e18];
        for (uint256 k; k < 6; ++k) {
            vm.warp(block.timestamp + 3600);
            _re("warpquiet", true);
            _fresh(1);
            _spread(12_000);
            _cur("setDials", abi.encodeCall(VPool.setDials, (phis[k], ltvs[k])));
            _xEdge(false, true, 100, 3e6);
            _xEdge(true, true, 100, 3e6);
            _x(false, false, 7e6, 0);
            _xEdge(false, true, 100, 3e6);
            _xEdge(true, false, 1e5, 3e9);
        }
    }

    function _seqIrm() internal {
        _begin("irm_outage");
        _arm(12_000);
        _band(0.1e18, 0, 1e12, 0);
        _xEdge(false, true, 100, 3e6);
        _xEdge(false, true, 100, 3e6);
        VRouter.VenueView memory v = _venue0();
        VMarketParams memory p = _params(v.id);
        vm.mockCallRevert(p.irm, abi.encodeWithSelector(VIrm.borrowRateView.selector), "down");
        vm.mockCallRevert(p.irm, abi.encodeWithSignature("borrowRate((address,address,address,address,uint256),(uint128,uint128,uint128,uint128,uint128,uint128))"), "down");
        _re("irm", true);
        uint256 lu = _mkt(v.id).lastUpdate;
        _x(false, true, 10_000, 0);
        _x(false, false, 5e6, 0);
        vm.warp(lu + 3599);
        _re("warpquiet", true);
        _fresh(0);
        _spread(12_000);
        _x(false, true, 10_000, 0);
        _x(false, false, 3e6, 0);
        _x(true, false, 3e6, 0);
        _warp(1); // IRM_STALE_GRACE exactly
        _x(false, false, 3e6, 0);
        _x(false, true, 10_000, 0);
        _warp(1); // past the grace: quarantined
        _x(false, false, 3e6, 0);
        _x(false, true, 10_000, 0);
        _x(true, false, 3e6, 0);
        _x(true, true, 10_000, 0);
        _xEdge(false, false, 1e5, 1e10);
        _x(false, false, 500e6, 0);
    }

    function _seqOracle() internal {
        _begin("oracle");
        _arm(12_000);
        _band(0.1e18, 0, 1e12, 0);
        _xEdge(false, true, 100, 3e6);
        _xEdge(false, true, 100, 3e6);
        VMarketParams memory p = _params(_venue0().id);
        uint256 snapO = vm.snapshotState();
        vm.mockCallRevert(p.oracle, abi.encodeWithSignature("price()"), "oracle down");
        _re("oracle", true);
        _x(false, true, 10_000, 0);
        _x(false, false, 2e6, 0);
        _x(true, false, 2e6, 0);
        uint256 d = _debt();
        _x(false, false, d + 1, 0); // covers the whole debt
        _xEdge(false, false, 1e5, 1e10);
        _xEdge(false, true, 100, 3e6);
        vm.revertToState(snapO);
        vm.clearMockedCalls();
        _re("rewind", true);
        vm.mockCall(p.oracle, abi.encodeWithSignature("price()"), abi.encode(uint256(0)));
        _re("oracle", true);
        _x(false, true, 10_000, 0);
        _x(false, false, 2e6, 0);
        _x(false, false, _debt() + 7, 0);
        _xEdge(false, false, 1e5, 1e10);
    }

    function _seqWhale() internal {
        _begin("whale");
        _arm(12_000);
        _band(0.1e18, 0, 1e12, 0);
        _xEdge(false, false, 1e5, 1e10); // lend first if possible
        VRouter.VenueView memory v = _venue0();
        VMarketParams memory p = _params(v.id);
        VMarket memory m = _mkt(v.id);
        address whale = makeAddr("whale");
        uint256 free = m.totalSupplyAssets - m.totalBorrowAssets;
        (, bytes memory r) = p.oracle.staticcall(abi.encodeWithSignature("price()"));
        uint256 op = abi.decode(r, (uint256));
        uint256 amt = free > 40e6 ? free - 40e6 : free / 2;
        uint256 coll = amt * 1e36 / op * 2 + 1e8;
        deal(CBBTC, whale, coll);
        vm.startPrank(whale);
        VERC20(CBBTC).approve(MORPHO, type(uint256).max);
        VMorpho(MORPHO).supplyCollateral(p, coll, whale, "");
        VMorpho(MORPHO).borrow(p, amt, 0, whale, whale);
        vm.stopPrank();
        _re("morpho", true);
        _xEdge(false, true, 100, 3e6);
        _xEdge(false, true, 100, 3e6);
        _xEdge(true, true, 100, 3e6);
        _warp(1800);
        _fresh(0);
        _spread(12_000);
        _xEdge(false, false, 1e5, 1e10);
        _xEdge(false, true, 100, 3e6);
        _warp(2 days);
        _fresh(0);
        _spread(12_000);
        _xEdge(false, true, 100, 3e6);
        _xEdge(false, false, 1e5, 1e10);
        _xEdge(true, false, 1e5, 1e10);
        // the whale repays half: cash returns
        deal(USDC, whale, amt);
        vm.startPrank(whale);
        VERC20(USDC).approve(MORPHO, type(uint256).max);
        (bool ok,) = MORPHO.call(abi.encodeWithSignature("repay((address,address,address,address,uint256),uint256,uint256,address,bytes)", p, amt / 2, 0, whale, ""));
        vm.stopPrank();
        _re("morpho", ok);
        _xEdge(false, true, 100, 3e6);
        _xEdge(true, true, 100, 3e6);
    }

    function _seqLending() internal {
        _begin("lending");
        _arm(12_000);
        _band(0.1e18, 0, 1e12, 5e6);
        _xEdge(false, true, 100, 3e6);
        _x(false, false, 3e6, 0);
        _x(false, false, _debt() + 9e6, 0); // repay, fill the reserve, lend the rest
        _xEdge(false, false, 1e5, 1e10);
        _x(false, true, 7_000, 0); // liquid, then withdraw
        VRouter.VenueView memory v = _venue0();
        (, uint256 sup,) = _loanPos();
        _cur("setVenueCaps", abi.encodeCall(VPool.setVenueCaps, (0, v.debtCap, uint128(sup + 2e6), v.maxBorrowRateWad)));
        _xEdge(false, false, 1e5, 1e10);
        _cur("setFeatures", abi.encodeCall(VPool.setFeatures, (63 & ~uint256(8))));
        _xEdge(false, false, 1e5, 1e10);
        _xEdge(false, true, 100, 3e6);
        _cur("setFeatures", abi.encodeCall(VPool.setFeatures, (63)));
        _x(false, false, 20e6, 0);
        _warp(7 days);
        _fresh(0);
        _spread(12_000);
        _xEdge(false, true, 100, 3e6);
        _xEdge(true, true, 100, 3e6);
        _cur("setVenueFlags", abi.encodeCall(VPool.setVenueFlags, (0, false, false)));
        _xEdge(false, true, 100, 3e6);
        _xEdge(false, false, 1e5, 1e10);
        _cur("setVenueFlags", abi.encodeCall(VPool.setVenueFlags, (0, true, true)));
        _xEdge(false, false, 1e5, 1e10);
    }

    function _seqMorphoFee() internal {
        _begin("morpho_fee");
        _arm(12_000);
        _band(0.1e18, 0, 1e12, 0);
        VMarketParams memory p = _params(_venue0().id);
        address o = VMorpho(MORPHO).owner();
        vm.prank(o);
        VMorpho(MORPHO).setFee(p, 0.25e18);
        _re("morpho", true);
        _xEdge(false, true, 100, 3e6);
        _xEdge(false, true, 100, 3e6);
        _warp(9 days + 3);
        _fresh(0);
        _spread(12_000);
        _x(false, false, 1e6, 0);
        _xEdge(false, false, 1e5, 1e10);
        _xEdge(true, true, 100, 3e6);
        _warp(1);
        _xEdge(true, false, 1e5, 1e10);
    }

    function _seqSameBlock() internal {
        _begin("same_block");
        _arm(40_000);
        for (uint256 k; k < 14; ++k) {
            uint256 r = _next();
            if (r % 4 == 0) _xEdge(false, true, 1 + r % 50, 1e6);
            else if (r % 4 == 1) _xEdge(false, false, 1e4 + r % 1e5, 1e9);
            else if (r % 4 == 2) _x(true, true, 1 + (r >> 8) % 60_000, 0);
            else _x(true, false, 1 + (r >> 8) % 60e6, 0);
        }
    }

    function _seqRandom(string memory name, uint256 steps) internal {
        _begin(name);
        _arm(uint24(2500 + _next() % 97_500));
        for (uint256 k; k < steps; ++k) {
            uint256 r = _next() % 100;
            uint256 z = _next();
            if (r < 18) {
                if (z % 2 == 0) _xEdge(false, true, 1 + z % 97, 5e6);
                else _x(false, true, _logRand(1, 400_000), (z >> 7) % 5 == 0 ? 1e9 : 0);
            } else if (r < 36) {
                if (z % 2 == 0) _xEdge(false, false, 1e3 + z % 1e5, 5e10);
                else _x(false, false, _logRand(1, 2e9), 0);
            } else if (r < 46) {
                if (z % 2 == 0) _xEdge(true, true, 1 + z % 97, 5e6);
                else _x(true, true, _logRand(1, 400_000), 0);
            } else if (r < 56) {
                if (z % 2 == 0) _xEdge(true, false, 1e3 + z % 1e5, 5e10);
                else _x(true, false, _logRand(1, 2e9), 0);
            } else if (r < 66) {
                uint256[9] memory dts = [uint256(1), 13, 59, 600, 1800, 3599, 3600, 3601, 86_400];
                _warp(dts[z % 9]);
                if ((z >> 9) % 3 != 0) _fresh((z >> 20) % 1200);
            } else if (r < 72) {
                uint24[5] memory sp = [uint24(2500), 2501, 79_999, 80_000, 100_000];
                _spread(z % 2 == 0 ? sp[(z >> 3) % 5] : uint24(2500 + (z >> 5) % 97_501));
            } else if (r < 78) {
                uint64[4] memory bands = [uint64(1.5e15), 5e16, 8e16, 0.1e18];
                uint256[4] memory mx = [uint256(0), 5e6, 50e6, 1e12];
                uint256[3] memory rt = [uint256(0), 1e6, 60e6];
                uint64 fl = (z >> 11) % 3 == 0 ? uint64(17e15) : 0;
                _band(bands[z % 4], fl, mx[(z >> 4) % 4], rt[(z >> 8) % 3]);
            } else if (r < 82) {
                uint64[4] memory fl = [uint64(0), 1e15, 17e15, 3e16];
                uint64[4] memory cp = [uint64(1e17), 1e16, 175e14, 1e18];
                _cur("setFeeBounds", abi.encodeCall(VPool.setFeeBounds, (fl[z % 4], cp[(z >> 2) % 4])));
            } else if (r < 86) {
                uint256[5] memory bits = [uint256(63), 63 & ~uint256(2), 63 & ~uint256(4), 63 & ~uint256(8), 63 & ~uint256(32)];
                _cur("setFeatures", abi.encodeCall(VPool.setFeatures, (bits[z % 5])));
            } else if (r < 90) {
                _cur("setPaused", abi.encodeCall(VPool.setPaused, (z % 2 == 0)));
                _cur("setLevPaused", abi.encodeCall(VPool.setLevPaused, ((z >> 1) % 3 == 0)));
            } else if (r < 95) {
                uint64[5] memory lt = [uint64(0.1e18), 0.3e18, 0.55e18, 0.62e18, 0.7e18];
                uint64[3] memory ph = [uint64(0.5e18), 0.77e18, 1e18];
                _cur("setDials", abi.encodeCall(VPool.setDials, (ph[z % 3], lt[(z >> 2) % 5])));
            } else {
                VRouter.VenueView memory v = _venue0();
                uint256 d = _debt();
                uint128[3] memory dc = [uint128(d + 1), uint128(d + 20e6), 1.5e12];
                uint64 rc = (z >> 5) % 2 == 0 ? v.maxBorrowRateWad : uint64(1e9 + (z >> 9) % 1e9);
                _cur("setVenueCaps", abi.encodeCall(VPool.setVenueCaps, (0, dc[z % 3], v.supplyCap, rc)));
            }
        }
    }


    /// @dev Output of a preview (0 when refused).
    function _outOf(bool lever, bool dir, uint256 a) internal view returns (uint256 o) {
        (bool ok, bytes memory ret) = _probe(lever, dir, a);
        if (ok) (, o) = abi.decode(ret, (uint256, uint256));
    }

    function _seqPingPong(string memory name, uint24 spread, uint256 steps) internal {
        _begin(name);
        _arm(spread);
        _band(0.1e18, 0, 1e12, (_next() % 2) * 3e6);
        for (uint256 k; k < steps; ++k) {
            uint256 z = _next();
            uint256 kind = z % 4;
            bool lever = kind >= 2;
            bool dir = kind % 2 == 0;
            uint256 e = dir ? _edge(lever, true, 1 + z % 300, 4e6) : _edge(lever, false, 1e4 + z % 1e6, 4e10);
            if (e == 0) {
                _x(lever, dir, dir ? 1 + (z >> 8) % 200_000 : 1 + (z >> 8) % 300e6, 0);
            } else if ((z >> 4) % 11 == 0) {
                uint256 o = _outOf(lever, dir, e);
                _x(lever, dir, e, o + 1); // Slippage
                _x(lever, dir, e, o); // exact limit
            } else if ((z >> 4) % 13 == 1) {
                _xd(lever, dir, e, 0, 1); // Expired
            } else {
                _x(lever, dir, (z >> 12) % 4 == 0 ? e / 2 + 1 : e, 0);
            }
            if ((z >> 20) % 5 == 0) {
                _warp(1 + (z >> 24) % 700);
                _fresh((z >> 40) % 300);
                if ((z >> 50) % 2 == 0) _spread(spread);
            }
        }
    }

    function _seqDonation() internal {
        _begin("donation");
        _arm(15_000);
        _band(0.1e18, 0, 1e12, 0);
        _xEdge(false, true, 100, 3e6);
        VRouter.VenueView memory v = _venue0();
        VMarketParams memory p = _params(v.id);
        address donor = makeAddr("donor");
        deal(CBBTC, donor, 1e8);
        deal(USDC, donor, 1000e6);
        vm.startPrank(donor);
        VERC20(CBBTC).approve(MORPHO, type(uint256).max);
        VERC20(USDC).approve(MORPHO, type(uint256).max);
        VMorpho(MORPHO).supplyCollateral(p, 50_000, v.account, "");
        vm.stopPrank();
        _re("donate", true);
        _xEdge(false, true, 100, 3e6);
        _xEdge(false, false, 1e5, 1e10);
        _xEdge(true, true, 100, 3e6);
        _xEdge(false, false, 1e5, 1e10);
        vm.prank(donor);
        VMorpho(MORPHO).supply(p, 7e6, 0, v.account, "");
        _re("donate", true);
        _xEdge(false, true, 100, 3e6);
        _xEdge(false, false, 1e5, 1e10);
        _xEdge(true, false, 1e5, 1e10);
        _xEdge(false, true, 100, 3e6);
        _warp(3 days);
        _fresh(0);
        _spread(15_000);
        _xEdge(false, false, 1e5, 1e10);
        _xEdge(false, true, 100, 3e6);
    }

    function _seqPriceMove() internal {
        _begin("price_move");
        _arm(20_000);
        _band(0.1e18, 0, 1e12, 0);
        _xEdge(false, true, 100, 3e6);
        _xEdge(false, true, 100, 3e6);
        address a = _aggOf(CBBTC);
        (, int256 x,) = _answerOf(a);
        VMarketParams memory p = _params(_venue0().id);
        (, bytes memory r) = p.oracle.staticcall(abi.encodeWithSignature("price()"));
        uint256 op = abi.decode(r, (uint256));
        int256[4] memory pct = [int256(97), 103, 90, 115];
        for (uint256 k; k < 4; ++k) {
            (, int256 us,) = _answerOf(_aggOf(USDC));
            _setFeeds(x * pct[k] / 100, 1, us, 1);
            if (k % 2 == 1) {
                vm.mockCall(p.oracle, abi.encodeWithSignature("price()"), abi.encode(op * uint256(pct[k]) / 100));
                _re("oracle", true);
            }
            _spread(20_000);
            _xEdge(false, true, 100, 3e6);
            _xEdge(false, false, 1e5, 1e10);
            _xEdge(true, true, 100, 3e6);
            _xEdge(true, false, 1e5, 1e10);
            _warp(61);
        }
    }

    function _seqBig() internal {
        _begin("big");
        bool ok = _depositBig(makeAddr("depositor"), 25e8);
        _re("deposit", ok);
        if (!ok) return;
        _arm(14_000);
        _band(0.1e18, 0, 1e12, 20_000e6);
        for (uint256 k; k < 70; ++k) {
            uint256 z = _next();
            uint256 kind = z % 4;
            bool lever = kind >= 2;
            bool dir = kind % 2 == 0;
            uint256 e = dir ? _edge(lever, true, 1 + z % 3000, 4e10) : _edge(lever, false, 1e5 + z % 1e7, 4e13);
            if (e == 0) _x(lever, dir, dir ? 1 + (z >> 8) % 1e9 : 1 + (z >> 8) % 1e12, 0);
            else _x(lever, dir, (z >> 12) % 3 == 0 ? e / 3 + 1 : e, 0);
            if ((z >> 20) % 4 == 0) {
                _warp(1 + (z >> 24) % 3000);
                _fresh((z >> 40) % 300);
                _spread(14_000);
            }
            if ((z >> 60) % 9 == 0) {
                uint64[3] memory lt = [uint64(0.45e18), 0.55e18, 0.62e18];
                vm.warp(block.timestamp + 3601);
                _re("warpquiet", true);
                _fresh(0);
                _spread(14_000);
                _cur("setDials", abi.encodeCall(VPool.setDials, (1e18, lt[(z >> 64) % 3])));
            }
        }
    }

    /// @dev Executes every preview-accepted point of a geometric sweep from the SAME state (rewinding after each),
    ///      to catch a settlement leg refusing what the plan accepted.
    function _tryAll(bool lever, bool dir, uint256 lo, uint256 hi, uint256 n) internal {
        uint256[] memory pts = _geo(lo, hi, n);
        for (uint256 k; k < pts.length; ++k) {
            (bool ok,) = _probe(lever, dir, pts[k]);
            if (!ok) continue;
            uint256 sn = vm.snapshotState();
            _x(lever, dir, pts[k], 0);
            vm.revertToState(sn);
            _re("rewind", true);
        }
    }

    function _seqSettleHunt() internal {
        _begin("settle_hunt");
        _arm(12_000);
        _band(0.1e18, 0, 1e12, 0);
        _xEdge(false, true, 100, 3e6);
        _xEdge(false, true, 100, 3e6);
        _xEdge(true, true, 100, 3e6);
        VRouter.VenueView memory v = _venue0();
        VMarketParams memory p = _params(v.id);
        _tryAll(false, false, 1e5, 1e10, 40);
        _tryAll(true, false, 1e5, 1e10, 40);
        vm.mockCallRevert(p.oracle, abi.encodeWithSignature("price()"), "oracle down");
        _re("oracle", true);
        _tryAll(false, false, 1e5, 1e10, 40);
        _tryAll(true, false, 1e5, 1e10, 40);
        _tryAll(false, true, 1, 1e7, 30);
        vm.clearMockedCalls();
        _re("oracle", true);
        vm.mockCall(p.oracle, abi.encodeWithSignature("price()"), abi.encode(uint256(0)));
        _re("oracle", true);
        _tryAll(false, false, 1e5, 1e10, 40);
        _tryAll(true, false, 1e5, 1e10, 40);
        vm.clearMockedCalls();
        _re("oracle", true);
        // oracle 4% under the feed: borrows refused by the router band, reclaim still reads Morpho's own price
        (, bytes memory r) = p.oracle.staticcall(abi.encodeWithSignature("price()"));
        uint256 op = abi.decode(r, (uint256));
        vm.mockCall(p.oracle, abi.encodeWithSignature("price()"), abi.encode(op * 96 / 100));
        _re("oracle", true);
        _tryAll(false, true, 1, 1e7, 30);
        _tryAll(true, true, 1, 1e7, 30);
        _tryAll(false, false, 1e5, 1e10, 30);
        vm.clearMockedCalls();
        _re("oracle", true);
        // a debt cap one unit above the debt and a supply cap at the supply
        (,, uint256 debt) = _loanPos();
        _cur("setVenueCaps", abi.encodeCall(VPool.setVenueCaps, (0, uint128(debt + 1), 0, v.maxBorrowRateWad)));
        _tryAll(false, true, 1, 1e7, 30);
        _tryAll(true, true, 1, 1e7, 30);
        _tryAll(false, false, 1e5, 1e10, 30);
    }

    /// @dev Router loan-level caps set by pranking the pool (unreachable through c104 governance): a debt cap one
    ///      unit above the loan's debt and a supply cap of 1 USDC, with trades that borrow, repay and lend.
    function _seqLoanCaps() internal {
        _begin("loancaps");
        _arm(11_000);
        _band(0.1e18, 0, 1e12, 0);
        _xEdge(false, true, 100, 3e6);
        (,, uint256 debt) = _loanPos();
        vm.prank(POOL);
        (bool ok,) = ROUTER.call(abi.encodeWithSignature("setLoanCaps(uint8,uint128,uint128,bool)", 0, uint128(debt + 1), uint128(1e6), true));
        _re("loanCaps", ok);
        _xEdge(false, true, 100, 3e6);
        _xEdge(true, true, 100, 3e6);
        _x(false, false, debt / 2, 0);
        _xEdge(false, false, 1e5, 1e10);
        _xEdge(false, true, 100, 3e6);
        _warp(3 days);
        _fresh(0);
        _spread(11_000);
        _xEdge(false, true, 100, 3e6);
        _xEdge(true, false, 1e5, 1e10);
    }

    function _run(uint256 blk, uint256 seed) internal {
        vm.createSelectFork(RPC, blk);
        seedBase = seed;
        out = string.concat("kyber-out/core_edges/seq_", vm.toString(blk), ".jsonl");
        if (vm.exists(out)) vm.removeFile(out);
        taker = makeAddr("vtaker");
        deal(CBBTC, taker, 1000e8);
        deal(USDC, taker, 50_000_000e6);
        vm.startPrank(taker);
        VERC20(CBBTC).approve(POOL, type(uint256).max);
        VERC20(USDC).approve(POOL, type(uint256).max);
        vm.stopPrank();
        base = vm.snapshotState();
        _seqHeartbeat();
        _seqSequencer();
        _seqSpread();
        _seqLtvWalk();
        _seqIrm();
        _seqOracle();
        _seqWhale();
        _seqLending();
        _seqMorphoFee();
        _seqSameBlock();
        _seqDonation();
        _seqPriceMove();
        _seqPingPong("pingpong_a", 2_500, 60);
        _seqPingPong("pingpong_b", 40_000, 60);
        _seqPingPong("pingpong_c", 80_000, 60);
        _seqSettleHunt();
        _seqLoanCaps();
        _seqBig();
        _seqRandom("random_a", 120);
        _seqRandom("random_b", 120);
        _seqRandom("random_c", 120);
    }

    function test_seq_51302915() public {
        _run(51_302_915, 0x5EED1);
    }

    function test_seq_51324800() public {
        _run(51_324_800, 0x5EED2);
    }

    function test_seq_51326000() public {
        _run(51_326_000, 0x5EED3);
    }
}
