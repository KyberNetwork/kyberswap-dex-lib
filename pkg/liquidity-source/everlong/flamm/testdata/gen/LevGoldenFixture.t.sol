// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import {LevBase} from "test/flamm/lev/LevBase.sol";
import {LevRecorder, LevRecLeverFill} from "test/kyber/LevRecorder.sol";
import {EverlongLeverageHook} from "src/hooks/everlong/lev/EverlongLeverageHook.sol";
import {CollRebalancerMath} from "src/hooks/everlong/lev/CollRebalancerMath.sol";

/// @notice Leverage-hook fixture on the local c104 stack (LevBase: genesis 10 BTC at $100k, spread 5,000 ppm).
///         Scenario 0 replays VenueGolden.test_golden_venueSequence_bitForBit step for step, recording the hook's
///         inputs and fill before every step and asserting the golden pins, so the Go port can reproduce the
///         pinned frames and fills. Scenarios 1-3 record amount grids on displaced states from VenueFrame: the
///         mark lifted 50% (CR above the 2.2 ceiling clamp), the mark cut 25% (far below target) and a pool
///         displaced by a 5 BTC sell. Scenarios 4-5 feed the hook synthetic contexts derived from a captured one
///         (the book and mark stay the chain's): a CR ladder from beyond the wall to far above target through the
///         loan legs, the unquotable and overflow edges, at the displaced state and at a mark cut to 60%.
/// @dev Exposes the hook's internal anchor-and-band assertion, which no reachable fill trips.
contract LevAssertBandHarness is EverlongLeverageHook {
    constructor(address pool, address hook) EverlongLeverageHook(pool, hook) {}

    function assertAnchorAndBand(uint256 xAnchorBefore, uint256 cvAfter, uint256 dAfter) external pure {
        _assertAnchorAndBand(xAnchorBefore, cvAfter, dAfter);
    }
}

contract LevGoldenFixtureTest is LevBase, LevRecorder {
    uint256 constant CV0 = 1999999999999999997898758;
    uint256 constant D0 = 999999999999999997898758;
    uint256 constant OUT1 = 4937593496;
    uint256 constant CV1 = 2009999999999999997888251;
    uint256 constant D1 = 1009937593495999997888251;
    uint256 constant OUT2 = 1230883;
    uint256 constant CV2 = 2007538233999999997890838;
    uint256 constant D2 = 1007468796959999997890838;
    uint256 constant OUT3 = 4919275043;
    uint256 constant CV3 = 2017538233999999997880332;
    uint256 constant D3 = 1017388072002999997880332;
    uint256 constant OUT4 = 922647;
    uint256 constant CV4 = 2015692939999999997882270;
    uint256 constant D4 = 1015541054419999997882270;

    function _rec(uint256 scenario, bool up, uint256 amount, uint256 hookOverride)
        internal
        returns (LevRecLeverFill memory f, bool ok)
    {
        (, f, ok) = _levRecord(scenario, address(pool), address(lhook), address(ehook), up, amount, hookOverride);
    }

    function _assertFrame(uint256 cv, uint256 d) internal view {
        EverlongLeverageHook.Frame memory f = _frame();
        assertEq(f.cv, cv, "golden cv");
        assertEq(uint256(f.d), d, "golden d");
    }

    function _grid(uint256 scenario) internal {
        uint256 btc = 1;
        for (uint256 i; i < 19; ++i) {
            _rec(scenario, true, btc, 0);
            btc = i % 2 == 0 ? btc * 3 : (btc * 10) / 3;
        }
        uint256 usd = 1;
        for (uint256 i; i < 25; ++i) {
            _rec(scenario, false, usd, 0);
            usd = i % 2 == 0 ? usd * 3 : (usd * 10) / 3;
        }
        // off-grid L18 lever-down amounts, hook level only
        _rec(scenario, false, 1, 1);
        _rec(scenario, false, 1, 999_999_999_999);
        _rec(scenario, false, 1, 123_456_789_012_345_678_901);
        _rec(scenario, false, 1, 4_937_593_496_000_000_000_001);
    }

    function test_levGoldenFixture() public {
        _levInitFields();

        // Scenario 0: the golden sequence.
        _assertFrame(CV0, D0);
        (LevRecLeverFill memory f1,) = _rec(0, true, 5e6, 0);
        (, uint256 out1) = _leverUp(trader, 5e6);
        assertEq(out1, OUT1, "out1");
        assertEq((f1.grossOut - f1.virtualLegL18) / LOAN_SCALE, out1, "up1 netting");
        _assertFrame(CV1, D1);

        (LevRecLeverFill memory f2,) = _rec(0, false, out1 / 2, 0);
        (, uint256 out2) = _leverDown(trader, out1 / 2);
        assertEq(out2, OUT2, "out2");
        assertEq(f2.grossOut, out2, "down1 out");
        _assertFrame(CV2, D2);

        (LevRecLeverFill memory f3,) = _rec(0, true, 5e6, 0);
        (, uint256 out3) = _leverUp(trader, 5e6);
        assertEq(out3, OUT3, "out3");
        assertEq((f3.grossOut - f3.virtualLegL18) / LOAN_SCALE, out3, "up2 netting");
        _assertFrame(CV3, D3);

        uint256 in4 = (out1 - out1 / 2 + out3) / 4;
        (LevRecLeverFill memory f4,) = _rec(0, false, in4, 0);
        (, uint256 out4) = _leverDown(trader, in4);
        assertEq(out4, OUT4, "out4");
        assertEq(f4.grossOut, out4, "down2 out");
        _assertFrame(CV4, D4);
        // the post-sequence frame, carried by a one-unit probe row
        _rec(0, true, 1, 0);

        // Scenario 1: every mark lifted 50%, above the CR ceiling clamp.
        uint256 snap = vm.snapshotState();
        _liftMark(150);
        _grid(1);
        _levRevertKeepingRows(snap);

        // Scenario 2: every mark cut to 75%, far below target.
        snap = vm.snapshotState();
        _liftMark(75);
        _grid(2);
        _levRevertKeepingRows(snap);

        // Scenario 3: a pool displaced by a 5 BTC sell.
        _sell(trader, 5 * BTC);
        _grid(3);

        // Scenarios 4-5: synthetic contexts at the displaced state and at a mark cut to 60%.
        _levSynthetic(4, address(pool), address(lhook), address(ehook));
        _liftMark(60);
        _levSynthetic(5, address(pool), address(lhook), address(ehook));

        string[] memory scenarios = new string[](6);
        scenarios[0] = "golden: VenueGolden sequence, pre-step rows then a post-sequence probe";
        scenarios[1] = "lift150: mark lifted to 150%";
        scenarios[2] = "lift75: mark cut to 75%";
        scenarios[3] = "sell5btc: pool displaced by a 5 BTC sell";
        scenarios[4] = "synthetic-sell5btc: hook-only CR ladder and edge contexts at the displaced state";
        scenarios[5] = "synthetic-lift60: hook-only CR ladder and edge contexts after the mark is cut to 60%";
        _levWrite("lev_golden", "test/kyber/lev_hook_local_fixture.json", scenarios, block.number);
    }

    /// @notice _assertAnchorAndBand over dust and in-domain (cv, D) states with the pre-anchor at, just above and
    ///         below the post-state's own anchor; one row per case with its revert data.
    function test_levAssertBandFixture() public {
        LevAssertBandHarness h = new LevAssertBandHarness(address(pool), address(ehook));
        uint256[] memory rows = new uint256[](0);
        uint256[14] memory cvs = [uint256(1), 2, 3, 5, 7, 10, 13, 30, 101, 1_000, 1e18, 2e24, 1e38, 1e38 + 1];
        uint256[10] memory crBps = [uint256(0), 10_000, 12_000, 15_000, 15_500, 16_000, 19_800, 20_000, 30_000, 100_000];
        uint256[5] memory pre = [uint256(0), 1, 2, 3, 4]; // 0, anchor-1, anchor, anchor+1, 2^255
        uint256 n;
        rows = new uint256[](cvs.length * (crBps.length + 3) * pre.length * 6);
        for (uint256 i; i < cvs.length; ++i) {
            for (uint256 j; j < crBps.length + 3; ++j) {
                uint256 d;
                if (j < crBps.length) d = crBps[j] == 0 ? 0 : cvs[i] * 10_000 / crBps[j];
                else d = cvs[i] + (j - crBps.length); // debt at, and past, the mark
                (uint256 xa,) = CollRebalancerMath.anchorAndBase(cvs[i], d, 1e18, CollRebalancerMath.LEVERAGE_RATIO_WAD);
                for (uint256 k; k < pre.length; ++k) {
                    uint256 xb = k == 0 ? 0 : k == 1 ? (xa == 0 ? 0 : xa - 1) : k == 2 ? xa : k == 3 ? xa + 1 : 1 << 255;
                    bytes memory err;
                    bool ok = true;
                    try h.assertAnchorAndBand(xb, cvs[i], d) {}
                    catch (bytes memory e) {
                        ok = false;
                        err = e;
                    }
                    rows[n++] = xb;
                    rows[n++] = cvs[i];
                    rows[n++] = d;
                    rows[n++] = ok ? 1 : 0;
                    (rows[n++], rows[n++]) = _selArg(err);
                }
            }
        }
        string memory obj = "lev_band";
        string[] memory fields = new string[](6);
        fields[0] = "xAnchorBefore";
        fields[1] = "cvAfter";
        fields[2] = "dAfter";
        fields[3] = "ok";
        fields[4] = "revertLen";
        fields[5] = "revertSelector";
        vm.serializeString(obj, "fields", fields);
        vm.serializeUint(obj, "stride", 6);
        string memory out = vm.serializeUint(obj, "v", rows);
        vm.writeJson(out, "test/kyber/lev_hook_band_fixture.json");
    }

    function _selArg(bytes memory err) internal pure returns (uint256 len, uint256 sel) {
        len = err.length;
        if (len >= 4) sel = uint256(uint32(bytes4(err)));
    }
}
