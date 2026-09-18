// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import {Vm} from "forge-std/Vm.sol";
import {CoreE2EBase, EPool, EPoolAdmin, ESpreadHook, EAggregator} from "test/kyber/CoreE2EBase.sol";
import {KERC20} from "test/kyber/MMFixtureBase.sol";

/// @notice Core end-to-end EXECUTED sequences on Base forks. A funded account trades directly on the deployed pool
///         (swap, leverUp, leverDown, each its own call with deadline = block.timestamp) with governance moves
///         (curator setLoanConfig / setLevPaused / setMaxSpreadAge / setPaused / setFeeBounds, keeper setSpread), time
///         warps, mocked feed rounds and an IRM outage between them. Per step: the op and its arguments, the block
///         timestamp, the call's return words or revert data, the pool's Swap / LeverUp / LeverDown event words, and
///         the complete post-state (CoreE2EBase._dump). Interleaved previewSwap / previewLever rows are recorded at
///         the state they see. Rows land in kyber-out/core_e2e/seq_<block>.jsonl.
contract CoreE2ESeqTest is CoreE2EBase {
    uint256 constant B1 = 51_302_915;
    uint256 constant B2 = 51_313_000;
    uint256 constant B3 = 51_324_800;

    bytes32 constant SWAP_SIG = keccak256("Swap(address,address,bool,uint256,uint256,uint256,uint256,uint256)");
    bytes32 constant LEVER_UP_SIG = keccak256("LeverUp(address,address,uint256,uint256,uint256,uint256)");
    bytes32 constant LEVER_DOWN_SIG = keccak256("LeverDown(address,address,uint256,uint256,uint256,uint256)");

    address internal _taker;
    string internal _seq;
    uint256 internal _stepNo;

    // ------------------------------------------------------------------ recording
    function _begin(string memory name) internal {
        _seq = name;
        _stepNo = 0;
        _line(string.concat('{"k":"seq","seq":"', name, '","s":', _dump(), "}"));
    }

    function _events() internal returns (string memory s) {
        Vm.Log[] memory logs = vm.getRecordedLogs();
        s = "[";
        bool first = true;
        for (uint256 i; i < logs.length; ++i) {
            if (logs[i].emitter != POOL || logs[i].topics.length == 0) continue;
            bytes32 t0 = logs[i].topics[0];
            string memory name;
            uint256 n;
            if (t0 == SWAP_SIG) (name, n) = ("Swap", 6);
            else if (t0 == LEVER_UP_SIG) (name, n) = ("LeverUp", 4);
            else if (t0 == LEVER_DOWN_SIG) (name, n) = ("LeverDown", 4);
            else continue;
            s = string.concat(s, first ? "" : ",", '{"name":"', name, '","w":', _words(logs[i].data, n), "}");
            first = false;
        }
        s = string.concat(s, "]");
    }

    function _args(uint256[] memory a) internal pure returns (string memory) {
        return _ua(a);
    }

    function _record(string memory op, uint256[] memory args, bool ok, bytes memory ret, uint256 nWords) internal {
        string memory ev = _events();
        _line(
            string.concat(
                '{"k":"step","seq":"', _seq, '","i":', vm.toString(_stepNo++), ',"op":"', op, '","args":', _args(args),
                string.concat(
                    ',"ts":', vm.toString(block.timestamp), ',"ok":', _b(ok),
                    ok ? string.concat(',"r":', _words(ret, nWords)) : string.concat(',"e":', _h(ret)),
                    string.concat(',"ev":', ev, ',"post":', _dump(), "}")
                )
            )
        );
    }

    function _a1(uint256 x) internal pure returns (uint256[] memory a) {
        a = new uint256[](1);
        a[0] = x;
    }

    function _a2(uint256 x, uint256 y) internal pure returns (uint256[] memory a) {
        a = new uint256[](2);
        (a[0], a[1]) = (x, y);
    }

    function _a4(uint256 x, uint256 y, uint256 z, uint256 w) internal pure returns (uint256[] memory a) {
        a = new uint256[](4);
        (a[0], a[1], a[2], a[3]) = (x, y, z, w);
    }

    // ------------------------------------------------------------------ trader ops
    /// @dev args: [sell, amountIn, minOut, deadlineOffset(0 = block.timestamp, 1 = one second ago), pairKind]
    ///      pairKind 0 is the real pair; 1 sends USDC for USDC (InvalidPair).
    function _swapFull(bool sell, uint256 amount, uint256 minOut, uint256 late, uint256 pairKind) internal returns (bool ok) {
        (address tin, address tout) = sell ? (CBBTC, USDC) : (USDC, CBBTC);
        if (pairKind == 1) (tin, tout) = (USDC, USDC);
        uint256 deadline = block.timestamp - late;
        vm.recordLogs();
        vm.prank(_taker);
        bytes memory ret;
        (ok, ret) = POOL.call(abi.encodeCall(EPool.swap, (tin, tout, amount, minOut, _taker, deadline)));
        uint256[] memory a = new uint256[](5);
        (a[0], a[1], a[2], a[3], a[4]) = (sell ? 1 : 0, amount, minOut, late, pairKind);
        _record("swap", a, ok, ret, 2);
    }

    function _swap(bool sell, uint256 amount) internal returns (bool) {
        return _swapFull(sell, amount, 0, 0, 0);
    }

    function _lever(bool up, uint256 amount, uint256 minOut) internal returns (bool ok) {
        vm.recordLogs();
        vm.prank(_taker);
        bytes memory ret;
        (ok, ret) = POOL.call(
            up
                ? abi.encodeCall(EPool.leverUp, (amount, minOut, _taker, block.timestamp))
                : abi.encodeCall(EPool.leverDown, (amount, minOut, _taker, block.timestamp))
        );
        _record(up ? "leverUp" : "leverDown", _a2(amount, minOut), ok, ret, 2);
    }

    function _previews(bool sell, uint256 amount) internal {
        _pSwap(_seq, sell, amount);
    }

    // ------------------------------------------------------------------ governance and environment ops
    function _admin(string memory op, uint256[] memory args, address who, bytes memory data, address target) internal {
        vm.recordLogs();
        vm.prank(who);
        (bool ok, bytes memory ret) = target.call(data);
        require(ok, string.concat("admin op failed: ", op));
        _record(op, args, ok, ret, 0);
    }

    function _setLoanConfig(uint64 band, uint64 floorWad, uint256 maxN, uint256 reserve) internal {
        _admin(
            "setLoanConfig", _a4(band, floorWad, maxN, reserve), _curator(),
            abi.encodeCall(EPool.setLoanConfig, (0, band, floorWad, maxN, reserve)), POOL
        );
    }

    function _setLevPaused(bool p) internal {
        _admin("setLevPaused", _a1(p ? 1 : 0), _curator(), abi.encodeCall(EPool.setLevPaused, (p)), POOL);
    }

    function _setPaused(bool p) internal {
        _admin("setPaused", _a1(p ? 1 : 0), _curator(), abi.encodeCall(EPoolAdmin.setPaused, (p)), POOL);
    }

    function _setFeeBounds(uint64 f, uint64 c) internal {
        _admin("setFeeBounds", _a2(f, c), _curator(), abi.encodeCall(EPoolAdmin.setFeeBounds, (f, c)), POOL);
    }

    function _setSpread(uint24 ppm) internal {
        _admin("setSpread", _a1(ppm), _keeper(), abi.encodeCall(ESpreadHook.setSpread, (ppm)), SPREAD_HOOK);
    }

    function _setMaxSpreadAge(uint32 age) internal {
        _admin("setMaxSpreadAge", _a1(age), _curator(), abi.encodeCall(ESpreadHook.setMaxSpreadAge, (age)), SPREAD_HOOK);
    }

    function _warp(uint256 dt) internal {
        vm.recordLogs();
        vm.warp(block.timestamp + dt);
        _record("warp", _a1(dt), true, "", 0);
    }

    /// @dev Tracker-input moves the replay copies from the post-state (new rounds, a rate model outage, a pin).
    function _env(string memory op, uint256 arg) internal {
        vm.recordLogs();
        _record(op, _a1(arg), true, "", 0);
    }

    function _cbAge() internal view returns (uint256) {
        (,,, uint256 ts,) = EAggregator(_agg(CBBTC)).latestRoundData();
        return block.timestamp - ts;
    }

    // ------------------------------------------------------------------ helpers
    function _physical() internal view returns (uint256 p) {
        (p,) = EPool(POOL).poolAssetPosition();
    }

    /// @dev The largest amount in [lo, hi] (a doubling scan, then bisection) whose preview answers; 0 when none.
    function _maxOk(bool lever, bool dir, uint256 lo, uint256 hi) internal view returns (uint256 best) {
        uint256 x = lo;
        while (x <= hi && !(lever ? _okLever(dir, x) : _okSwap(dir, x))) x = x * 3 / 2 + 1;
        while (x <= hi) {
            bool ok = lever ? _okLever(dir, x) : _okSwap(dir, x);
            if (!ok) break;
            best = x;
            x *= 2;
        }
        if (best == 0) return 0;
        uint256 bad = x;
        while (bad - best > 1) {
            uint256 mid = (best + bad) / 2;
            if (lever ? _okLever(dir, mid) : _okSwap(dir, mid)) best = mid;
            else bad = mid;
        }
    }

    // ------------------------------------------------------------------ sequences
    function _seqBasic() internal {
        _begin("basic");
        _swap(true, 15_000);
        _swap(false, 20e6);
        _swap(true, 50_000);
        _swapFull(true, 10_000, type(uint256).max, 0, 0); // Slippage
        _swapFull(true, 10_000, 0, 1, 0); // Expired
        _swapFull(true, 10_000, 0, 0, 1); // InvalidPair
        _swapFull(false, 0, 0, 0, 0); // InvalidAmount
        _swap(false, 40e6); // repays the debt, lends the surplus
        _previews(true, 50_000);
        _swap(true, 50_000); // withdraws the supply, then borrows
        _swap(false, 1); // dust
        _swap(true, 1_000_000); // band
        uint256 m = _maxOk(false, true, 1_000, 10_000_000);
        if (m != 0) _swap(true, m);
        _swap(false, 5e6);
        uint256 mb = _maxOk(false, false, 1e6, 1e12);
        if (mb != 0) _swap(false, mb);
        _previews(true, 30_000);
        _previews(false, 30e6);
    }

    function _seqReclaim() internal {
        _begin("reclaim");
        for (uint256 k; k < 4; ++k) {
            uint256 m = _maxOk(false, true, 1_000, 10_000_000);
            if (m == 0) break;
            _swap(true, m);
        }
        // A buy paying out more poolAsset than is held physically: repay, reclaim the shortfall, release excess.
        uint256 phys = _physical();
        uint256 amt = 1e6;
        for (uint256 k; k < 40; ++k) {
            (bool ok, bytes memory ret) = POOL.staticcall(abi.encodeCall(EPool.previewSwap, (false, amt)));
            if (ok) {
                (, uint256 out,) = abi.decode(ret, (uint256, uint256, uint256));
                if (out > phys) break;
            }
            amt = amt * 5 / 4;
        }
        _swap(false, amt);
        _swap(false, 10e6);
        uint256 mb = _maxOk(false, false, 1e6, 1e12);
        if (mb != 0) _swap(false, mb);
        _swap(true, 20_000);
    }

    function _seqNotional() internal {
        _begin("notional");
        (uint64 band, uint64 floorWad, uint256 maxN, uint256 reserve) = _liveLoanCfg();
        _swap(true, 30_000);
        _setLoanConfig(band, floorWad, 10e6, reserve);
        _previews(true, 50_000);
        _swap(true, 50_000); // partial fill at the notional cap
        _swap(false, 12e6); // NotionalCap
        _swap(false, 9e6);
        _setLoanConfig(band, floorWad, maxN, 5e6);
        _swap(false, 30e6); // repay, keep the reserve liquid, lend the rest
        _swap(true, 20_000); // pays from liquid first
        _swap(true, 8_000);
        _setLoanConfig(band, 3e16, maxN, reserve);
        _swap(true, 8_000); // FeeOutOfBounds
        _setFeeBounds(0, 1e16);
        _setLoanConfig(band, floorWad, maxN, reserve);
        _swap(true, 8_000); // fee clipped at the cap
        _swap(false, 8e6);
    }

    function _seqLever() internal {
        _begin("lever");
        _lever(true, 5_000, 0); // LevPaused
        _setLevPaused(false);
        _pLever(_seq, true, 5_000);
        _pLever(_seq, false, 5e6);
        _lever(false, 5e6, 0); // degraded to the venue ceiling
        _setSpread(17_500);
        _lever(true, 5_000, 0);
        _lever(false, 5e6, 0);
        _lever(true, 5_000, type(uint256).max); // Slippage
        _swap(true, 10_000);
        _lever(true, 8_000, 0);
        _swap(false, 10e6);
        _lever(false, 3e6, 0);
        uint256 mu = _maxOk(true, true, 100, 10_000_000);
        if (mu != 0) _lever(true, mu, 0);
        uint256 md = _maxOk(true, false, 10_000, 1e12);
        if (md != 0) _lever(false, md, 0);
        _setMaxSpreadAge(1);
        _warp(2);
        _pLever(_seq, false, 3e6);
        _lever(false, 2e6, 0); // degraded to the last live spread
        uint256 dd = _maxOk(true, false, 1_000, 1e12);
        if (dd != 0) _lever(false, dd, 0); // a degraded fill: the stored spread does not move
        _lever(true, 5_000, 0); // SpreadUnavailable
        _setSpread(90_000); // live again, clamped to the band
        _lever(false, 2e6, 0);
        _lever(true, 2_000, 0);
        uint256 dc = _maxOk(true, false, 1_000, 1e12);
        if (dc != 0) _lever(false, dc, 0); // live at the band ceiling: stored as the next degrade value
        _setSpread(17_500);
        uint256 uc = _maxOk(true, true, 100, 10_000_000);
        if (uc != 0) _lever(true, uc, 0);
        _setPaused(true);
        _lever(true, 2_000, 0); // Paused
        _swap(true, 2_000); // Paused
        _lever(false, 2e6, 0);
        _swap(false, 2e6);
    }

    function _seqWarp() internal {
        _begin("warp");
        _swap(true, 20_000);
        _warp(600);
        _swap(false, 10e6);
        uint256 age = _cbAge();
        if (age + 60 < 3600) _warp(3600 - age - 30);
        _swap(true, 10_000);
        _warp(60); // past the heartbeat
        _swap(true, 10_000); // StalePrice
        _swap(false, 5e6); // StalePrice
        _previews(true, 10_000);
        _warp(30 days);
        _freshFeeds(3);
        _env("mockFeeds", 3);
        _swap(false, 10e6); // repays with a month of interest
        _swap(true, 30_000);
        _warp(1 days);
        _freshFeeds(1);
        _env("mockFeeds", 1);
        _swap(true, 10_000);
    }

    function _seqPinLow() internal {
        _begin("pin_low");
        _storePin(0.2e18);
        _env("storePin", 0.2e18);
        _previews(true, 60_000);
        _previews(true, 100_000);
        uint256 m = _maxOk(false, true, 1_000, 10_000_000);
        if (m != 0) _swap(true, m);
        _swap(true, 60_000);
        _swap(false, 20e6);
        _swap(true, 90_000);
    }

    function _seqIrm() internal {
        _begin("irm");
        _swap(true, 30_000);
        _warp(1_800);
        _irmDown();
        _freshFeeds(2);
        _env("irmDown", 1_800);
        _swap(false, 5e6); // repays into a stale rate model inside the grace
        _swap(true, 10_000); // no funding
        _warp(7_200);
        _freshFeeds(2);
        _env("mockFeeds", 2);
        _swap(false, 5e6); // quarantined venue
        _swap(true, 10_000);
    }

    function _run(uint256 blockNumber) internal {
        vm.createSelectFork(RPC, blockNumber);
        _out = string.concat("kyber-out/core_e2e/seq_", vm.toString(blockNumber), ".jsonl");
        if (vm.exists(_out)) vm.removeFile(_out);
        _taker = makeAddr("taker");
        deal(CBBTC, _taker, 100e8);
        deal(USDC, _taker, 10_000_000e6);
        vm.startPrank(_taker);
        KERC20(CBBTC).approve(POOL, type(uint256).max);
        KERC20(USDC).approve(POOL, type(uint256).max);
        vm.stopPrank();
        uint256 snap = vm.snapshotState();
        if (blockNumber == B1) {
            _begin("real_sell");
            _swap(true, 15_000);
            _reset(snap);
        }
        _seqBasic();
        _reset(snap);
        _seqReclaim();
        _reset(snap);
        _seqNotional();
        _reset(snap);
        _seqLever();
        _reset(snap);
        _seqWarp();
        _reset(snap);
        _seqPinLow();
        _reset(snap);
        _seqIrm();
    }

    function test_seq_b1() public {
        _run(B1);
    }

    function test_seq_b2() public {
        _run(B2);
    }

    function test_seq_b3() public {
        _run(B3);
    }
}
