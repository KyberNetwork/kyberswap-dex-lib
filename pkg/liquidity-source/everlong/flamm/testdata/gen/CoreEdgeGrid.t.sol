// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import {CoreEdgesBase, VPool, VSpread, VRouter, VMorpho, VIrm, VMarketParams, VMarket, VERC20, VAgg} from "test/kyber/CoreEdgesBase.sol";

/// @notice Edge preview grids: per scenario one state dump and previewSwap / previewLever rows (seeded
///         log-random, geometric sweeps with every class transition bisected to one unit and a unit window around
///         it) plus router.fundingCeiling probes. Output: kyber-out/core_edges/grid_<block>.jsonl.
contract CoreEdgeGridTest is CoreEdgesBase {
    uint256 snap;

    function _swapSweep(string memory tag, uint256 edges, uint256 rand) internal {
        _edges(tag, false, true, _geo(1, 4e8, 40), 12, edges);
        _edges(tag, false, false, _geo(1, 4e11, 40), 12, edges);
        _randGrid(tag, false, true, 1, 1e9, rand);
        _randGrid(tag, false, false, 1, 1e12, rand);
        _row(tag, false, true, type(uint256).max);
        _row(tag, false, false, type(uint256).max);
        _row(tag, false, false, type(uint256).max / 1e12 + 1);
        _row(tag, false, true, 0);
    }

    function _leverSweep(string memory tag, uint256 edges, uint256 rand) internal {
        _edges(tag, true, true, _geo(1, 4e8, 40), 12, edges);
        _edges(tag, true, false, _geo(1, 4e11, 40), 12, edges);
        _randGrid(tag, true, true, 1, 1e9, rand);
        _randGrid(tag, true, false, 1, 1e12, rand);
        _row(tag, true, true, type(uint256).max);
        _row(tag, true, false, type(uint256).max / 1e12 + 1);
    }

    function _small(string memory tag) internal {
        uint256[6] memory s = [uint256(1), 77, 5_000, 60_000, 139_000, 2_000_000];
        uint256[6] memory b = [uint256(1), 999, 1e6, 50e6, 150e6, 3000e6];
        for (uint256 k; k < 6; ++k) {
            _row(tag, false, true, s[k]);
            _row(tag, false, false, b[k]);
            _row(tag, true, true, s[k]);
            _row(tag, true, false, b[k]);
        }
    }

    function _fcSweep(string memory tag) internal {
        (, uint256 gross) = VPool(POOL).poolAssetPosition();
        (bool ok, bytes memory ret) = FEED.staticcall(abi.encodeWithSignature("peekCross(address,address)", CBBTC, USDC));
        uint256 p;
        if (ok) (, p,) = abi.decode(ret, (bool, uint256, uint48));
        if (p == 0) p = 776903017291400;
        uint256[9] memory c = [uint256(0), 1, 1000, 30_000, gross / 2, gross, gross * 3, 5e8, 1e30];
        for (uint256 k; k < c.length; ++k) {
            _fcRow(tag, c[k], p);
            _fcRow(tag, c[k], p / 2);
            _fcRow(tag, c[k], p * 101 / 100);
        }
        // unit steps of collateral around the pin-bound region
        for (uint256 k; k < 12; ++k) _fcRow(tag, gross + k * 7919, p);
        _fcRow(tag, gross, 1);
        _fcRow(tag, gross, 0);
    }

    function _fresh(uint256 age) internal {
        address a = _aggOf(CBBTC);
        address b = _aggOf(USDC);
        vm.clearMockedCalls();
        (uint80 ia, int256 xa,) = _answerOf(a);
        (uint80 ib, int256 xb,) = _answerOf(b);
        _mockRound(a, ia, xa, block.timestamp - age, block.timestamp - age);
        _mockRound(b, ib, xb, block.timestamp - age, block.timestamp - age);
    }

    function _arm(uint24 spread) internal {
        vm.prank(_curator());
        VPool(POOL).setLevPaused(false);
        vm.prank(_keeper());
        VSpread(SPREAD).setSpread(spread);
    }

    function _reset() internal {
        vm.revertToState(snap);
        vm.clearMockedCalls();
    }

    function _loanCfg(uint64 band, uint64 fl, uint256 mx, uint256 rt) internal {
        vm.prank(_curator());
        VPool(POOL).setLoanConfig(0, band, fl, mx, rt);
    }

    // ------------------------------------------------------------------ scenarios
    function _live() internal {
        _state("live");
        _swapSweep("live", 8, 60);
        _small("live");
        _fcSweep("live");
    }

    function _armed() internal {
        _arm(2500);
        _state("armed_min");
        _leverSweep("armed_min", 8, 50);
        _swapSweep("armed_min", 2, 10);
        _reset();
        _arm(100_000);
        _state("armed_max");
        _leverSweep("armed_max", 6, 30);
        _reset();
        // band 5% -> ceiling 50_000; a 60_000 post clamps to it
        _arm(60_000);
        _loanCfg(5e16, 0, 1e12, 0);
        _state("armed_band5");
        _leverSweep("armed_band5", 6, 30);
        _swapSweep("armed_band5", 4, 20);
        _reset();
        // band 0.15% -> ceiling 1_500 below the 2_500 floor: the ceiling wins
        _arm(2500);
        _loanCfg(1.5e15, 0, 1e12, 0);
        _state("armed_band_tiny");
        _leverSweep("armed_band_tiny", 6, 30);
        _swapSweep("armed_band_tiny", 4, 20);
        _reset();
        // band at the max 10%: wider accepted sets, output ceilings reachable
        _arm(99_999);
        _loanCfg(0.1e18, 0, 1e12, 0);
        _state("armed_band_max");
        _leverSweep("armed_band_max", 8, 30);
        _swapSweep("armed_band_max", 8, 30);
        _reset();
    }

    function _spreadAge() internal {
        _arm(33_333);
        uint256 t0 = block.timestamp;
        vm.warp(t0 + 3600); // lastSetTs + maxSpreadAge: still live
        _fresh(0);
        _state("spread_age_eq");
        _leverSweep("spread_age_eq", 3, 10);
        vm.warp(t0 + 3601);
        _fresh(0);
        _state("spread_age_over");
        _leverSweep("spread_age_over", 4, 10);
        _reset();
        // stale from the start, never filled: lever-down degrades to the 100_000 venue ceiling
        vm.prank(_curator());
        VPool(POOL).setLevPaused(false);
        _state("degrade_zero");
        _leverSweep("degrade_zero", 4, 20);
        _reset();
        // max age zero disables staleness
        vm.prank(_curator());
        VPool(POOL).setLevPaused(false);
        vm.prank(_curator());
        VSpread(SPREAD).setMaxSpreadAge(0);
        _state("age_zero");
        _leverSweep("age_zero", 4, 10);
        _reset();
    }

    function _dialsDown(uint64 phi, uint64 target) internal returns (bool) {
        for (uint256 k; k < 8; ++k) {
            uint64 ltv = VPool(POOL).dials().ltvWad;
            if (ltv == target && VPool(POOL).dials().phiWad == phi) return true;
            uint64 next = ltv > target + 0.1e18 ? ltv - 0.1e18 : target;
            vm.warp(block.timestamp + 3601);
            vm.prank(_curator());
            (bool ok,) = POOL.call(abi.encodeCall(VPool.setDials, (phi, next)));
            if (!ok) return false;
        }
        return VPool(POOL).dials().ltvWad == target;
    }

    function _ltvLow() internal {
        _arm(20_000);
        uint64[3] memory targets = [uint64(0.25e18), 0.12e18, 0.1e18];
        uint64[3] memory phis = [uint64(1e18), 0.5e18, 0.73e18];
        for (uint256 k; k < 3; ++k) {
            if (!_dialsDown(phis[k], targets[k])) {
                _emit(string.concat('{"k":"note","tag":"ltv_low","msg":"setDials refused at ', vm.toString(k), '"}'));
                break;
            }
            vm.prank(_keeper());
            VSpread(SPREAD).setSpread(20_000);
            _fresh(5);
            string memory tag = string.concat("ltv_low_", vm.toString(k));
            _state(tag);
            _swapSweep(tag, 8, 30);
            _leverSweep(tag, 8, 30);
            _fcSweep(tag);
        }
        _reset();
    }

    function _notional() internal {
        _loanCfg(0.1e18, 0, 20e6, 0);
        _state("notional_20");
        _swapSweep("notional_20", 8, 30);
        _reset();
        _loanCfg(0.1e18, 0, 1, 0);
        _state("notional_1");
        _swapSweep("notional_1", 6, 20);
        _reset();
    }

    function _feeBounds() internal {
        (, uint256 fee) = _tryFee();
        uint64 f = uint64(fee);
        vm.prank(_curator());
        VPool(POOL).setFeeBounds(0, f / 2);
        _state("fee_clip");
        _swapSweep("fee_clip", 6, 30);
        _reset();
        vm.prank(_curator());
        VPool(POOL).setFeeBounds(f, f);
        _state("fee_eq");
        _swapSweep("fee_eq", 6, 30);
        _reset();
        vm.prank(_curator());
        VPool(POOL).setFeeBounds(f + 1, 1e18);
        _state("fee_floor_over");
        _swapSweep("fee_floor_over", 6, 20);
        _reset();
        // loan floor set under a high cap, then the pool cap lowered below it
        vm.prank(_curator());
        VPool(POOL).setFeeBounds(0, 1e18);
        _loanCfg(8e16, f - 1, 1e12, 0);
        vm.prank(_curator());
        VPool(POOL).setFeeBounds(0, f / 3);
        _state("fee_loanfloor_cap");
        _swapSweep("fee_loanfloor_cap", 6, 20);
        _reset();
        vm.prank(_curator());
        VPool(POOL).setFeeBounds(0, 1e18);
        _state("fee_cap_wad");
        _swapSweep("fee_cap_wad", 6, 20);
        _reset();
    }

    function _tryFee() internal view returns (bool ok, uint256 fee) {
        bytes memory ret;
        (ok, ret) = _pvSwap(true, 1000);
        if (ok) (,, fee) = abi.decode(ret, (uint256, uint256, uint256));
        if (fee < 4) fee = 17499999999999999;
    }

    function _caps() internal {
        VRouter.VenueView memory v = _venue0();
        bytes32 id = v.id;
        (, uint128 bs,) = VMorpho(MORPHO).position(id, v.account);
        VMarket memory m = _mkt(id);
        uint256 debt = bs == 0 ? 0 : (uint256(bs) * (m.totalBorrowAssets + 1) + m.totalBorrowShares + 1e6 - 1) / (m.totalBorrowShares + 1e6);
        vm.prank(_curator());
        VPool(POOL).setVenueCaps(0, uint128(debt + 3e6), v.supplyCap, v.maxBorrowRateWad);
        _loanCfg(0.1e18, 0, 1e12, 0);
        _state("debtcap_tight");
        _swapSweep("debtcap_tight", 8, 30);
        _fcSweep("debtcap_tight");
        _reset();
        // rate cap at the current rate: the borrow leg becomes non-monotone in size
        VMarketParams memory p = _params(id);
        uint256 rate = VIrm(p.irm).borrowRateView(p, _mkt(id));
        uint256[4] memory d = [uint256(0), 1, 1000, 1e6];
        for (uint256 k; k < d.length; ++k) {
            vm.prank(_curator());
            VPool(POOL).setVenueCaps(0, v.debtCap, v.supplyCap, uint64(rate + d[k]));
            _loanCfg(0.1e18, 0, 1e12, 0);
            string memory tag = string.concat("ratecap_", vm.toString(d[k]));
            _state(tag);
            _swapSweep(tag, 6, 20);
            _fcSweep(tag);
            _reset();
        }
        vm.prank(_curator());
        VPool(POOL).setVenueCaps(0, v.debtCap, v.supplyCap, uint64(rate - 1));
        _state("ratecap_below");
        _swapSweep("ratecap_below", 6, 20);
        _fcSweep("ratecap_below");
        _reset();
        vm.prank(_curator());
        VPool(POOL).setVenueFlags(0, false, true);
        _arm(10_000);
        _state("borrow_off");
        _swapSweep("borrow_off", 6, 20);
        _leverSweep("borrow_off", 4, 10);
        _fcSweep("borrow_off");
        _reset();
        vm.prank(VRouter(ROUTER).protocolSafe());
        VRouter(ROUTER).setGlobalPaused(true);
        _state("global_paused");
        _swapSweep("global_paused", 6, 20);
        _fcSweep("global_paused");
        _reset();
    }

    function _feeds() internal {
        _arm(15_000);
        address a = _aggOf(CBBTC);
        address b = _aggOf(USDC);
        address sq = abi.decode(_call(FEED, abi.encodeWithSignature("SEQUENCER_FEED()")), (address));
        (uint80 ia, int256 xa,) = _answerOf(a);
        (uint80 ib, int256 xb,) = _answerOf(b);
        uint256 t = block.timestamp;
        uint256 s2 = vm.snapshotState();
        // cbBTC heartbeat 3600, USDC 90000
        _feedCase(s2, "cb_age_eq", a, ia, xa, t - 3600, b, ib, xb, t - 10);
        _feedCase(s2, "cb_age_over", a, ia, xa, t - 3601, b, ib, xb, t - 10);
        _feedCase(s2, "cb_future", a, ia, xa, t + 1, b, ib, xb, t - 10);
        _feedCase(s2, "cb_round0", a, 0, xa, t - 1, b, ib, xb, t - 10);
        _feedCase(s2, "cb_neg", a, ia, -1, t - 1, b, ib, xb, t - 10);
        _feedCase(s2, "cb_one", a, ia, 1, t - 1, b, ib, xb, t - 10);
        _feedCase(s2, "cb_maxprice_eq", a, ia, int256(uint256(1 << 200) / 100), t - 1, b, ib, 1e8, t - 10);
        _feedCase(s2, "cb_maxprice_under", a, ia, int256(uint256(1 << 200) / 100 - 1), t - 1, b, ib, 1e8, t - 10);
        _feedCase(s2, "usdc_age_eq", a, ia, xa, t - 5, b, ib, xb, t - 90000);
        _feedCase(s2, "usdc_age_over", a, ia, xa, t - 5, b, ib, xb, t - 90001);
        _feedCase(s2, "peg_lo_eq", a, ia, xa, t - 5, b, ib, 99000000, t - 10);
        _feedCase(s2, "peg_lo_over", a, ia, xa, t - 5, b, ib, 98999999, t - 10);
        _feedCase(s2, "peg_hi_eq", a, ia, xa, t - 5, b, ib, 101000000, t - 10);
        _feedCase(s2, "peg_hi_over", a, ia, xa, t - 5, b, ib, 101000001, t - 10);
        _feedCase(s2, "btc_x3", a, ia, xa * 3, t - 5, b, ib, xb, t - 10);
        _feedCase(s2, "btc_half", a, ia, xa / 2, t - 5, b, ib, xb, t - 10);
        // sequencer: answer 0 = up; startedAt within grace (<=) / just past / future / down / never started
        _seqCase(s2, "seq_grace_eq", sq, 0, t - 3600);
        _seqCase(s2, "seq_grace_past", sq, 0, t - 3601);
        _seqCase(s2, "seq_future", sq, 0, t + 5);
        _seqCase(s2, "seq_down", sq, 1, t - 100000);
        _seqCase(s2, "seq_zero_started", sq, 0, 0);
        _reset();
    }

    function _call(address to, bytes memory data) internal view returns (bytes memory ret) {
        bool ok;
        (ok, ret) = to.staticcall(data);
        require(ok, "call");
    }

    function _feedCase(uint256 s2, string memory tag, address a, uint80 ia, int256 xa, uint256 ta, address b, uint80 ib, int256 xb, uint256 tb)
        internal
    {
        _mockRound(a, ia, xa, ta, ta);
        _mockRound(b, ib, xb, tb, tb);
        _state(tag);
        _swapSweep(tag, 3, 8);
        _leverSweep(tag, 3, 8);
        vm.revertToState(s2);
        vm.clearMockedCalls();
    }

    function _seqCase(uint256 s2, string memory tag, address sq, int256 ans, uint256 startedAt) internal {
        (uint80 id,,,,) = VAgg(sq).latestRoundData();
        _mockRound(sq, id, ans, startedAt, block.timestamp - 1);
        _state(tag);
        _swapSweep(tag, 2, 6);
        _leverSweep(tag, 2, 6);
        vm.revertToState(s2);
        vm.clearMockedCalls();
    }

    function _morphoFee() internal {
        VRouter.VenueView memory v = _venue0();
        VMarketParams memory p = _params(v.id);
        vm.prank(VMorpho(MORPHO).owner());
        VMorpho(MORPHO).setFee(p, 0.25e18);
        vm.warp(block.timestamp + 1 days + 17);
        _fresh(7);
        _arm(12_345);
        _loanCfg(0.1e18, 0, 1e12, 0);
        _state("morpho_fee");
        _swapSweep("morpho_fee", 8, 30);
        _leverSweep("morpho_fee", 6, 20);
        _fcSweep("morpho_fee");
        _reset();
    }

    function _irmGrace() internal {
        VRouter.VenueView memory v = _venue0();
        VMarketParams memory p = _params(v.id);
        VMarket memory m = _mkt(v.id);
        _arm(12_000);
        _loanCfg(0.1e18, 0, 1e12, 0);
        vm.mockCallRevert(p.irm, abi.encodeWithSelector(VIrm.borrowRateView.selector), "down");
        vm.mockCallRevert(p.irm, abi.encodeWithSignature("borrowRate((address,address,address,address,uint256),(uint128,uint128,uint128,uint128,uint128,uint128))"), "down");
        uint256 t0 = m.lastUpdate;
        uint256[3] memory dt = [uint256(3599), 3600, 3601];
        for (uint256 k; k < 3; ++k) {
            if (t0 + dt[k] < block.timestamp) continue;
            vm.warp(t0 + dt[k]);
            address a = _aggOf(CBBTC);
            address b = _aggOf(USDC);
            _mockRound(a, 1, 7769030172914, block.timestamp - 3, block.timestamp - 3);
            _mockRound(b, 1, 99990000, block.timestamp - 3, block.timestamp - 3);
            vm.prank(_keeper());
            VSpread(SPREAD).setSpread(12_000);
            string memory tag = string.concat("irm_dt_", vm.toString(dt[k]));
            _state(tag);
            _swapSweep(tag, 6, 20);
            _leverSweep(tag, 6, 20);
            _fcSweep(tag);
        }
        _reset();
    }

    function _oracle() internal {
        VRouter.VenueView memory v = _venue0();
        VMarketParams memory p = _params(v.id);
        _arm(12_000);
        _loanCfg(0.1e18, 0, 1e12, 0);
        uint256 s2 = vm.snapshotState();
        vm.mockCallRevert(p.oracle, abi.encodeWithSignature("price()"), "oracle down");
        _state("oracle_revert");
        _swapSweep("oracle_revert", 6, 20);
        _leverSweep("oracle_revert", 6, 20);
        _fcSweep("oracle_revert");
        vm.revertToState(s2);
        vm.clearMockedCalls();
        vm.mockCall(p.oracle, abi.encodeWithSignature("price()"), abi.encode(uint256(0)));
        _state("oracle_zero");
        _swapSweep("oracle_zero", 6, 20);
        _leverSweep("oracle_zero", 6, 20);
        vm.revertToState(s2);
        vm.clearMockedCalls();
        // market oracle 3% below the feed: outside the router's 2% oracle band
        uint256 op = abi.decode(_call(p.oracle, abi.encodeWithSignature("price()")), (uint256));
        vm.mockCall(p.oracle, abi.encodeWithSignature("price()"), abi.encode(op * 97 / 100));
        _state("oracle_band_out");
        _swapSweep("oracle_band_out", 6, 20);
        _fcSweep("oracle_band_out");
        vm.revertToState(s2);
        vm.clearMockedCalls();
        _reset();
    }

    function _whale() internal {
        VRouter.VenueView memory v = _venue0();
        VMarketParams memory p = _params(v.id);
        VMarket memory m = _mkt(v.id);
        address whale = makeAddr("whale");
        uint256 free = m.totalSupplyAssets - m.totalBorrowAssets;
        uint256 op = abi.decode(_call(p.oracle, abi.encodeWithSignature("price()")), (uint256));
        // borrow all but 1_000 USDC; collateral at 50% of the market lltv
        uint256 borrowAmt = free > 1000e6 ? free - 1000e6 : free / 2;
        uint256 coll = borrowAmt * 1e36 / op * 2 + 1e8;
        deal(CBBTC, whale, coll);
        vm.startPrank(whale);
        VERC20(CBBTC).approve(MORPHO, type(uint256).max);
        VMorpho(MORPHO).supplyCollateral(p, coll, whale, "");
        VMorpho(MORPHO).borrow(p, borrowAmt, 0, whale, whale);
        vm.stopPrank();
        _arm(12_000);
        _loanCfg(0.1e18, 0, 1e12, 0);
        _state("whale_t0");
        _swapSweep("whale_t0", 8, 30);
        _leverSweep("whale_t0", 4, 10);
        _fcSweep("whale_t0");
        uint256[2] memory dt = [uint256(3000), 3 days];
        for (uint256 k; k < 2; ++k) {
            vm.warp(block.timestamp + dt[k]);
            _fresh(1);
            vm.prank(_keeper());
            VSpread(SPREAD).setSpread(12_000);
            string memory tag = string.concat("whale_dt", vm.toString(k));
            _state(tag);
            _swapSweep(tag, 8, 30);
            _leverSweep(tag, 4, 10);
            _fcSweep(tag);
        }
        _reset();
    }

    function _big() internal {
        if (!_depositBig(makeAddr("depositor"), 25e8)) {
            _reset();
            return;
        }
        _arm(13_000);
        _state("big");
        _edges("big", false, true, _geo(1, 4e10, 48), 10, 10);
        _edges("big", false, false, _geo(1, 4e13, 48), 10, 10);
        _edges("big", true, true, _geo(1, 4e10, 48), 10, 10);
        _edges("big", true, false, _geo(1, 4e13, 48), 10, 10);
        _randGrid("big", false, true, 1, 1e10, 60);
        _randGrid("big", false, false, 1, 1e13, 60);
        _randGrid("big", true, true, 1, 1e10, 40);
        _randGrid("big", true, false, 1, 1e13, 40);
        _fcSweep("big");
        _loanCfg(0.1e18, 0, 150_000e6, 0);
        _state("big_notional");
        _edges("big_notional", false, true, _geo(1, 4e10, 48), 10, 10);
        _edges("big_notional", false, false, _geo(1, 4e13, 48), 10, 10);
        _reset();
    }

    /// @dev Router loan-level caps: only the pool may set them and c104 never does (unreachable through governance),
    ///      so this covers the port's loan-level branches, not a live configuration.
    function _loanCaps() internal {
        (bool okp, bytes memory ret) = POOL.staticcall(abi.encodeWithSignature("loanPosition()"));
        (,, uint256 debt) = okp ? abi.decode(ret, (uint256, uint256, uint256)) : (0, 0, 0);
        _loanCfg(0.1e18, 0, 1e12, 0);
        vm.prank(POOL);
        (bool ok,) = ROUTER.call(abi.encodeWithSignature("setLoanCaps(uint8,uint128,uint128,bool)", 0, uint128(debt + 2e6), uint128(0), true));
        if (ok) {
            _state("loancap_debt");
            _swapSweep("loancap_debt", 6, 20);
            _fcSweep("loancap_debt");
        }
        vm.prank(POOL);
        (ok,) = ROUTER.call(abi.encodeWithSignature("setLoanCaps(uint8,uint128,uint128,bool)", 0, uint128(0), uint128(1), false));
        if (ok) {
            _arm(11_000);
            _state("loan_borrow_off");
            _swapSweep("loan_borrow_off", 6, 20);
            _leverSweep("loan_borrow_off", 4, 10);
            _fcSweep("loan_borrow_off");
        }
        _reset();
    }

    function _run(uint256 blk, uint256 seed) internal {
        vm.createSelectFork(RPC, blk);
        rng = seed;
        out = string.concat("kyber-out/core_edges/grid_", vm.toString(blk), ".jsonl");
        if (vm.exists(out)) vm.removeFile(out);
        snap = vm.snapshotState();
        _live();
        _reset();
        _armed();
        _spreadAge();
        _ltvLow();
        _notional();
        _feeBounds();
        _caps();
        _feeds();
        _morphoFee();
        _irmGrace();
        _oracle();
        _whale();
        _big();
        _loanCaps();
    }

    function test_grid_51302915() public {
        _run(51_302_915, 0xA11CE);
    }

    function test_grid_51324800() public {
        _run(51_324_800, 0xB0B);
    }

    function test_grid_51326000() public {
        _run(51_326_000, 0xC0FFEE);
    }
}
