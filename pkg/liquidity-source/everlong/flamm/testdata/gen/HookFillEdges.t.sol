// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import {FillResult, PoolContext, SwapContext} from "src/interfaces/core/flamm/IFLAMMHooks.sol";
import {SwapHookEdgesBase} from "./SwapHookEdgesBase.sol";

struct HookFillEdgesBook {
    uint256 kappa;
    uint256 rs;
    uint256 is_;
    uint256 rv;
    uint256 iv;
    uint256 x;
}

interface IHookFillEdgesHook {
    function spot(PoolContext calldata ctx) external view returns (uint256);
    function bookFor(PoolContext calldata ctx) external view returns (HookFillEdgesBook memory);
    function previewFeeWad(SwapContext calldata ctx, uint256 spotBefore) external view returns (uint256);
    function previewExactIn(SwapContext calldata ctx, uint256 feeWad) external view returns (FillResult memory);
    function executeExactIn(SwapContext calldata ctx, uint256 feeWad) external returns (FillResult memory);
    function LOAN_SCALE() external view returns (uint256);
}

/// @notice Delegating tap etched over the hook for the live-transaction replay: it logs the exact calldata and
///         return data of executeExactIn (a CALL, so logging is legal) and forwards everything else untouched.
contract HookFillEdgesTap {
    address constant IMPL = address(0x00000000000000000000000000000000bE11c0dE);

    event Tap(bytes data, bool ok, bytes ret);

    fallback(bytes calldata data) external returns (bytes memory) {
        (bool ok, bytes memory ret) = IMPL.delegatecall(data);
        if (bytes4(data[:4]) == IHookFillEdgesHook.executeExactIn.selector) emit Tap(data, ok, ret);
        if (!ok) {
            assembly ("memory-safe") {
                revert(add(ret, 32), mload(ret))
            }
        }
        return ret;
    }
}

/// @notice Edge generator for the LIVE Base EverlongHook swap path. Every state is written
///         into the hook's storage (curve row, band, anchor, book, variance, fee row) and every context is run
///         through spot, bookFor, previewFeeWad, previewExactIn and a pool-pranked executeExactIn whose committed
///         book is read back. Covers the lazy-rescale branches (acc == 0 re-seed, HALF reset, empty seed,
///         gross == acc, clamp), retracted/invalid books, the cap re-solve at 1-wei neighbours of the net,
///         the fee edges (0, WAD-1, WAD), the loan-grid snap, LOAN_SCALE patched to 1 and 1e10, and a keyed
///         random grid; plus the real 15000-sat sell replayed through a calldata tap.
contract HookFillEdges is SwapHookEdgesBase {
    address constant HOOK = 0x65CBD227cBC61248ae77a5fC813A29C54C092134;
    address constant POOL = 0xc0fdCB1799cCc2CEBaA1fe247157b0dF33D57572;
    address constant ALM = 0xf82DdF0A8a50bc2C3F163997766bA1839E527A17;
    bytes4 constant SEL_RESERVES_AT = 0xfa07504e;
    uint256 constant WAD = 1e18;
    uint256 constant HALF = 5e17;
    uint256 constant U64 = type(uint64).max;
    uint256 constant LIVE_X = 532533306204662064;
    uint256 constant LIVE_KAPPA = 12779948558379;
    uint256 constant LIVE_ANCHOR = 2196067230099687190109974130321092041;
    uint256 constant LIVE_RP = 768302232848967000000000000000000;
    uint256 constant XLO = 24096289941794299;
    uint256 constant XHI = 1099912607172170593;
    uint256 constant YHI = 24096289941794300;

    IHookFillEdgesHook constant H = IHookFillEdgesHook(HOOK);

    uint256 internal _slot0Hi;
    uint256 internal _slot6Hi;
    bytes internal _liveCode;
    uint256 internal _loanScale;

    // ------------------------------------------------------------------ state plumbing
    // s: 0 aWad, 1-4 sup, 5 anchor, 6 rp, 7 kappa, 8 x, 9 rs, 10 is, 11 rv, 12 iv, 13 rvWad, 14-21 fee row,
    //    22 invKappa, 23 invBand, 24 loanScale
    // c: 0 phys, 1 posted, 2 poolAssetIn, 3 amountIn, 4 maxAmountOut, 5 feeWad

    function _put(uint256 slot, uint256 v) internal {
        vm.store(HOOK, bytes32(slot), bytes32(v));
    }

    function _load(uint256 slot) internal view returns (uint256) {
        return uint256(vm.load(HOOK, bytes32(slot)));
    }

    function _liveState() internal pure returns (uint256[25] memory s) {
        s[0] = 34e18;
        (s[1], s[2], s[3], s[4]) = (34e18, XLO, XHI, YHI);
        (s[5], s[6], s[7], s[8]) = (LIVE_ANCHOR, LIVE_RP, LIVE_KAPPA, LIVE_X);
        (s[9], s[10], s[11], s[12], s[13]) = (157080117684030704791, 201304154388180895, 234423, 0, 0);
        (s[14], s[15], s[16], s[17]) = (0.03e18, 0.005e18, 0.05e18, 0.0004e18);
        (s[18], s[19], s[20], s[21]) = (4e18, 0.5e18, 2e18, 0.15e18);
        (s[22], s[23], s[24]) = (4e18, 6e16, 1e12);
    }

    function _cp(uint256[25] memory a) internal pure returns (uint256[25] memory b) {
        for (uint256 i; i < 25; ++i) {
            b[i] = a[i];
        }
    }

    function _apply(uint256[25] memory s) internal {
        require(s[0] < 1 << 128 && s[5] <= type(uint160).max, "state range");
        for (uint256 i = 14; i < 24; ++i) {
            require(s[i] <= U64, "u64");
        }
        _put(0, _slot0Hi | s[0]);
        _put(4, s[14] | (s[15] << 64) | (s[16] << 128) | (s[17] << 192));
        _put(5, s[18] | (s[19] << 64) | (s[20] << 128) | (s[21] << 192));
        _put(6, _slot6Hi | s[22] | (s[23] << 64));
        for (uint256 i; i < 4; ++i) {
            _put(10 + i, s[1 + i]);
        }
        for (uint256 i; i < 8; ++i) {
            _put(14 + i, s[5 + i]); // 14 anchor .. 21 iv
        }
        _put(23, s[13]);
        require(s[24] == _loanScale, "loan scale");
    }

    function _pctx(uint256[6] memory c) internal pure returns (PoolContext memory p) {
        p.physicalPoolAsset = c[0];
        p.postedPoolAsset = c[1];
        p.priceWad = 76830;
        p.loanCount = 1;
    }

    function _sctx(uint256[6] memory c) internal pure returns (SwapContext memory x) {
        x.pool = _pctx(c);
        x.poolAssetIn = c[2] != 0;
        x.amountIn = c[3];
        x.maxAmountOut = c[4];
        x.crossWad = WAD;
    }

    function _args(uint256[25] memory s, uint256[6] memory c) internal pure returns (uint256[] memory a) {
        a = new uint256[](31);
        for (uint256 i; i < 25; ++i) {
            a[i] = s[i];
        }
        for (uint256 i; i < 6; ++i) {
            a[25 + i] = c[i];
        }
    }

    function _reservesAt(uint256[25] memory s, uint256 kappa, uint256 x)
        internal
        view
        returns (bool ok, uint256 st, uint256 vol)
    {
        bytes memory ret;
        (ok, ret) = ALM.staticcall(
            abi.encodeWithSelector(SEL_RESERVES_AT, s[1], s[2], s[3], s[4], uint160(s[5]), kappa, x)
        );
        if (ok) (st, vol) = abi.decode(ret, (uint256, uint256));
    }

    function _onCurve(uint256[25] memory base, uint256 x, uint256 kappa) internal view returns (uint256[25] memory s) {
        s = _cp(base);
        (s[7], s[8]) = (kappa, x);
        (bool ok, uint256 st, uint256 vol) = _reservesAt(s, kappa, x);
        require(ok, "onCurve");
        (s[9], s[11], s[10], s[12]) = (st, vol, 0, 0);
    }

    /// @dev The accounted volatile the book will rescale from (the re-seed's when rv + iv == 0).
    function _acc(uint256[25] memory s) internal view returns (uint256) {
        if (s[11] + s[12] != 0) return s[11] + s[12];
        (bool ok,, uint256 vol) = _reservesAt(s, 1e30, s[8]);
        if (!ok) return 0;
        if (vol == 0) (ok,, vol) = _reservesAt(s, 1e30, HALF);
        return ok ? vol : 0;
    }

    // ------------------------------------------------------------------ recorded calls
    function _spot(uint256[25] memory s) internal {
        uint256[6] memory c;
        (bool ok, bytes memory ret) = HOOK.staticcall(abi.encodeCall(H.spot, (_pctx(c))));
        _emit("hook.spot", _args(s, c), ok, ret, 1);
    }

    function _book(uint256[25] memory s, uint256[6] memory c) internal {
        (bool ok, bytes memory ret) = HOOK.staticcall(abi.encodeCall(H.bookFor, (_pctx(c))));
        _emit("hook.bookFor", _args(s, c), ok, ret, 6);
    }

    function _fee(uint256[25] memory s, uint256[6] memory c) internal returns (bool ok, uint256 fee) {
        bytes memory ret;
        (ok, ret) = HOOK.staticcall(abi.encodeCall(H.previewFeeWad, (_sctx(c), 12345)));
        _emit("hook.previewFeeWad", _args(s, c), ok, ret, 1);
        if (ok) fee = abi.decode(ret, (uint256));
    }

    function _fill(uint256[25] memory s, uint256[6] memory c) internal returns (bool ok, FillResult memory fr) {
        bytes memory ret;
        (ok, ret) = HOOK.staticcall(abi.encodeCall(H.previewExactIn, (_sctx(c), c[5])));
        _emit("hook.previewExactIn", _args(s, c), ok, ret, 4);
        if (ok) fr = abi.decode(ret, (FillResult));

        vm.prank(POOL);
        (bool eok, bytes memory eret) = HOOK.call(abi.encodeCall(H.executeExactIn, (_sctx(c), c[5])));
        uint256[] memory outs = new uint256[](10);
        if (eok) {
            uint256[] memory w = _words(eret, 4);
            for (uint256 i; i < 4; ++i) {
                outs[i] = w[i];
            }
            for (uint256 i; i < 6; ++i) {
                outs[4 + i] = _load(16 + i); // kappa, rs, is, rv, iv, x
            }
        }
        _emitRaw("hook.executeExactIn", _args(s, c), eok, outs, eret);
        for (uint256 i; i < 6; ++i) {
            _put(16 + i, s[7 + i]); // restore kappa .. iv, x
        }
        _put(21, s[12]);
        _put(17, s[8]);
    }

    function _sellAmts() internal pure returns (uint256[11] memory) {
        return [uint256(0), 1, 999, 15000, 139999, 234423, 1e7, 1e10, 1e14, 1 << 128, type(uint256).max];
    }

    function _buyAmts() internal pure returns (uint256[12] memory) {
        return [
            uint256(0), 1, 1e12 - 1, 1e12, 1e12 + 1, 1e17, 11301759e12, 190e18, 1e24, 1e30, 1 << 128,
            type(uint256).max
        ];
    }

    /// @dev Full context sweep for one state.
    function _sweep(uint256[25] memory s, bool deep) internal {
        _apply(s);
        _spot(s);
        uint256 acc = _acc(s);
        uint256[5] memory grosses = [acc, acc + 1, acc > 0 ? acc - 1 : 3, 0, (acc * 7) / 3 + 1];
        for (uint256 gi; gi < grosses.length; ++gi) {
            if (!deep && gi > 1) break;
            uint256[6] memory c;
            c[1] = gi == 4 ? grosses[gi] / 3 : 0;
            c[0] = grosses[gi] - c[1];
            _book(s, c);
            _dirSweep(s, c, true, gi == 0 && deep);
            _dirSweep(s, c, false, gi == 0 && deep);
        }
    }

    function _dirSweep(uint256[25] memory s, uint256[6] memory c, bool sell, bool deep) internal {
        c[2] = sell ? 1 : 0;
        c[3] = 1;
        c[4] = type(uint256).max;
        (bool fok, uint256 fee) = _fee(s, c);
        uint256 n = sell ? 11 : 12;
        for (uint256 j; j < n; ++j) {
            c[3] = sell ? _sellAmts()[j] : _buyAmts()[j];
            c[4] = type(uint256).max;
            c[5] = fok ? fee : 17499999999999999;
            (bool ok, FillResult memory fr) = _fill(s, c);
            if (!deep) continue;
            // cap neighbours of the realised net
            if (ok && fr.grossOut != 0) {
                uint256 net = fr.grossOut - fr.feeOut;
                uint256[5] memory caps = [net > 0 ? net - 1 : 0, net, net + 1, net / 2, 0];
                for (uint256 k; k < caps.length; ++k) {
                    c[4] = caps[k];
                    _fill(s, c);
                }
                c[4] = type(uint256).max;
            }
            if (j % 2 == 1 || j == 6) {
                uint256[4] memory fees = [uint256(0), WAD - 1, WAD, type(uint256).max];
                for (uint256 k; k < fees.length; ++k) {
                    c[5] = fees[k];
                    _fill(s, c);
                }
            }
        }
    }

    function setUp() public {
        vm.createSelectFork("https://mainnet.base.org", 51310000);
        _slot0Hi = _load(0) & ~uint256(type(uint128).max);
        _slot6Hi = _load(6) & ~uint256(type(uint128).max);
        _liveCode = HOOK.code;
        _loanScale = H.LOAN_SCALE();
    }

    function _begin(string memory tag) internal returns (uint256[25] memory L) {
        _open(string.concat("test/kyber/fixtures/hook_fill_edges_", tag, ".json"));
        L = _liveState();
        require(_load(16) == L[7] && _load(17) == L[8] && _load(18) == L[9] && _load(20) == L[11], "live drift");
    }

    function _end() internal {
        _close();
        emit log_named_uint("rows", _rows);
    }

    function test_hs01() public {
        uint256[25] memory L = _begin("s01");
        _sweep(L, true);
        _end();
    }

    function test_hs02() public {
        uint256[25] memory L = _begin("s02");
        uint256[25] memory s = _cp(L);
        s[13] = 1.6e11;
        _sweep(s, false);
        _sweep(_onCurve(L, HALF, 1e30), true); // the constructor seed book
        _end();
    }

    function test_hs03() public {
        uint256[25] memory L = _begin("s03");
        _sweep(_onCurve(L, XLO, LIVE_KAPPA), true); // rv == 0: one-sided
        _end();
    }

    function test_hs04() public {
        uint256[25] memory L = _begin("s04");
        _sweep(_onCurve(L, XHI, LIVE_KAPPA), true); // rs == 0
        _end();
    }

    function test_hs05() public {
        uint256[25] memory L = _begin("s05");
        _sweep(_onCurve(L, XLO + 1, LIVE_KAPPA), false);
        _sweep(_onCurve(L, XHI - 1, LIVE_KAPPA), false);
        uint256[25] memory s = _onCurve(L, 3e17, 1e18);
        (s[10], s[12]) = (5e12 + 7, 13);
        _sweep(s, true);
        _end();
    }

    function test_hs06() public {
        uint256[25] memory L = _begin("s06");
        uint256[25] memory s = _cp(L);
        (s[11], s[12]) = (0, 0);
        _sweep(s, true);
        _end();
    }

    function test_hs07() public {
        uint256[25] memory L = _begin("s07");
        uint256[25] memory s = _cp(L);
        (s[11], s[12]) = (0, 0);
        s[8] = XLO; // seed holds no volatile: HALF reset
        _sweep(s, false);
        s[8] = 0;
        _sweep(s, false);
        s[8] = type(uint256).max;
        _sweep(s, false);
        s = _cp(L);
        (s[11], s[12], s[2], s[8]) = (0, 0, 6e17, 6e17); // re-seed empty at both coordinates
        _sweep(s, false);
        s = _cp(L);
        (s[11], s[12], s[5]) = (0, 0, 0); // anchor 0 on the re-seed
        _sweep(s, false);
        _end();
    }

    function test_hs08() public {
        uint256[25] memory L = _begin("s08");
        uint256[25] memory s = _cp(L);
        s[7] = 0; // retracted
        _sweep(s, false);
        s = _cp(L);
        s[0] = 5e17; // invalid _p.aWad: spot / fee / spotAfter revert
        _sweep(s, false);
        s[0] = 1e18; // _p.aWad differs from the support's
        _sweep(s, false);
        s = _cp(L);
        s[9] = 1; // rs clamp on sells
        _sweep(s, false);
        s = _cp(L);
        (s[11], s[12]) = (0, 234423); // all idle
        _sweep(s, false);
        _end();
    }

    function test_hs09() public {
        uint256[25] memory L = _begin("s09");
        uint256[25] memory s = _cp(L);
        s[6] = 0;
        _sweep(s, false);
        s = _onCurve(L, 3e17, LIVE_KAPPA);
        s[6] = type(uint256).max; // spotAt overflow
        _sweep(s, false);
        s = _cp(L);
        s[5] = 0; // anchor 0 with a live book
        _sweep(s, false);
        s = _cp(L);
        s[8] = 1999e15 + 1;
        _sweep(s, false);
        s = _cp(L);
        s[8] = XLO - 1e9;
        _sweep(s, false);
        s[8] = XHI + 1e9;
        _sweep(s, false);
        _end();
    }

    function test_hs10() public {
        uint256[25] memory L = _begin("s10");
        uint256[25] memory s = _onCurve(L, 7e17, 1e15);
        s[13] = 5e13;
        (s[16], s[17], s[21], s[22], s[23]) = (0, 0, 0, 0, 0); // bare
        _sweep(s, false);
        s = _onCurve(L, 7e17, 1e15);
        (s[14], s[15], s[21]) = (0.005e18, 0.03e18, 1e18 + 5); // mid < out, skew > WAD
        _sweep(s, false);
        s = _onCurve(L, 2e17, 1e15);
        (s[13], s[22], s[23]) = (1 << 100, U64, 0);
        _sweep(s, false);
        s = _onCurve(L, 45e16, 1e15);
        (s[22], s[23]) = (16e18, 49e16);
        _sweep(s, false);
        s = _cp(L);
        (s[1], s[2], s[3], s[4]) = (3e18, 150000000000000000, 900000000000000000, 1);
        s = _onCurve(s, 5e17, 1e14); // a different curve row under the live _p.aWad
        _sweep(s, false);
        _end();
    }

    function test_hs11() public {
        uint256[25] memory L = _begin("s11");
        // gross overflows: phys + posted > 2^256 - 1
        uint256[6] memory c = [type(uint256).max, 1, 1, 15000, type(uint256).max, 17499999999999999];
        _apply(L);
        _book(L, c);
        _fee(L, c);
        _fill(L, c);
        c = [type(uint256).max, 0, 0, 1e18, type(uint256).max, 17499999999999999];
        _book(L, c);
        _fee(L, c);
        _fill(L, c);
        // invalid support amplification under a valid _p.aWad
        uint256[25] memory s = _cp(L);
        s[1] = 5e17;
        _sweep(s, false);
        // malformed band (xLo > xHi) on the re-seed path and on a live book
        s = _cp(L);
        (s[2], s[3], s[11], s[12]) = (9e17, 2e17, 0, 0);
        _sweep(s, false);
        s = _cp(L);
        (s[2], s[3]) = (9e17, 2e17);
        _sweep(s, false);
        // rs == 0 and rv == 0 with only idle on both legs
        s = _cp(L);
        (s[9], s[10], s[11], s[12]) = (0, 1e20, 0, 5);
        _sweep(s, false);
        _end();
    }

    /// @dev Books sized so fills are material: kappa large, amounts a sizable fraction of the book, and caps
    ///      drawn around the uncapped net.
    function test_hr5() public {
        uint256[25] memory L = _begin("r5");
        for (uint256 i; i < 120; ++i) {
            uint256 r = _rand(31, i);
            uint256 x = XLO + (r >> 8) % (XHI - XLO + 1);
            uint256[25] memory s = _onCurve(L, x, 1e12 + (r >> 72) % 1e20);
            s[13] = (r >> 140) % 1e14;
            s[10] = (r >> 30) % 1e18;
            _apply(s);
            uint256[6] memory c;
            c[0] = s[11] + s[12];
            if (i % 3 == 1) c[0] = c[0] + (r >> 200) % (c[0] / 50 + 1);
            c[2] = i & 1;
            (bool fok, uint256 fee) = _fee(s, c);
            uint256 leg = c[2] == 1 ? s[11] : s[9];
            uint256 scale = c[2] == 1 ? 1 : 1e0;
            c[3] = ((leg / 4 + 1) * (1 + (r >> 100) % 400)) / 100 / scale + (r >> 60) % 1000;
            c[4] = type(uint256).max;
            c[5] = fok ? fee : 3e15;
            (bool ok, FillResult memory fr) = _fill(s, c);
            if (ok && fr.grossOut > fr.feeOut) {
                uint256 net = fr.grossOut - fr.feeOut;
                c[4] = (net * (1 + (r >> 40) % 99)) / 100;
                _fill(s, c);
                c[4] = net - 1;
                _fill(s, c);
            }
        }
        _end();
    }

    function test_hr1() public {
        _randomGrid(_begin("r1"), 0, 110);
        _end();
    }

    function test_hr2() public {
        _randomGrid(_begin("r2"), 110, 220);
        _end();
    }

    function test_hr3() public {
        _randomGrid(_begin("r3"), 220, 330);
        _end();
    }

    function test_hr4() public {
        _randomGrid(_begin("r4"), 330, 440);
        _end();
    }

    function _randomGrid(uint256[25] memory L, uint256 from, uint256 to) internal {
        for (uint256 i = from; i < to; ++i) {            uint256 r = _rand(21, i);
            uint256 x = XLO + (r >> 8) % (XHI - XLO + 1);
            uint256 kappa = 1e6 + _logRand(r >> 72, 24);
            uint256[25] memory s = _onCurve(L, x, kappa);
            s[10] = _logRand(r >> 120, 16);
            s[12] = i % 3 == 0 ? _logRand(r >> 140, 6) : 0;
            s[13] = _logRand(r >> 160, 15);
            uint256 r2 = _rand(22, i);
            s[14] = (r2 >> 8) % 1e17;
            s[15] = (r2 >> 72) % (s[14] + 1);
            s[16] = i % 7 == 0 ? 0 : (r2 >> 96) % 1e18;
            s[17] = i % 5 == 0 ? 0 : 1 + (r2 >> 130) % 1e15;
            s[18] = (r2 >> 150) % 8e18;
            s[19] = (r2 >> 170) % 1e18;
            s[20] = s[19] + (r2 >> 190) % 7e18;
            s[21] = (r2 >> 210) % 9e17;
            s[22] = i % 4 == 0 ? 0 : (r2 >> 20) % 16e18;
            s[23] = s[22] == 0 ? 0 : (r2 >> 40) % 49e16;
            _apply(s);
            uint256 acc = s[11] + s[12];
            uint256[6] memory c;
            uint256 mode = (r >> 250) % 4;
            uint256 gross = mode == 0 ? acc : (mode == 1 ? acc + (r >> 180) % 1000 : (acc * (90 + (r >> 190) % 20)) / 100);
            c[1] = i % 5 == 0 ? gross / 4 : 0;
            c[0] = gross - c[1];
            c[2] = i & 1;
            if (i % 6 == 0) _book(s, c);
            (bool fok, uint256 fee) = _fee(s, c);
            c[3] = c[2] == 1 ? _logRand(r2 >> 60, 12) : _logRand(r2 >> 60, 30);
            c[4] = i % 3 == 0 ? _logRand(r2 >> 100, 30) : type(uint256).max;
            c[5] = fok && i % 8 != 0 ? fee : (r2 >> 30) % WAD;
            _fill(s, c);
        }
    }

    /// @dev LOAN_SCALE is an immutable: patch its three PUSH32 sites to model a hook for another loan decimals.
    function test_hookLoanScale1() public {
        _scaleSweep(1, "scale1");
    }

    function test_hookLoanScale1e10() public {
        _scaleSweep(1e10, "scale1e10");
    }

    function _scaleSweep(uint256 scale, string memory tag) internal {
        _open(string.concat("test/kyber/fixtures/hook_fill_edges_", tag, ".json"));
        bytes memory code = _liveCode;
        require(_patchWord(code, 1e12, scale) == 3, "immutable sites");
        vm.etch(HOOK, code);
        require(H.LOAN_SCALE() == scale, "scale patch");
        _loanScale = scale;
        uint256[25] memory L = _liveState();
        L[24] = scale;
        _sweep(L, true);
        uint256[25] memory s = _onCurve(L, 3e17, 1e18);
        (s[10], s[12]) = (5e12 + 7, 13);
        _sweep(s, true);
        _end();
    }

    function _patchWord(bytes memory code, uint256 from, uint256 to) internal pure returns (uint256 n) {
        for (uint256 i; i + 33 <= code.length; ++i) {
            if (uint8(code[i]) != 0x7f) continue;
            uint256 w;
            assembly ("memory-safe") {
                w := mload(add(add(code, 33), i))
            }
            if (w == from) {
                assembly ("memory-safe") {
                    mstore(add(add(code, 33), i), to)
                }
                ++n;
            }
        }
    }

    /// @dev The one real swap. The transaction is replayed as-is for the committed book, then re-driven from the
    ///      taker (same call, same pre-state) through a calldata tap to capture the exact hook context; the two
    ///      committed books must agree. The hook's own previews are then recorded at the untouched pre-state.
    function test_hookLiveTx() public {
        bytes32 txh = 0x46c3cd72a5860b2fe546e5a2130e066314e3777027151661e1e4f19a935901fa;
        vm.createSelectFork("https://mainnet.base.org", txh);
        _open("test/kyber/fixtures/hook_live_tx_edges.json");
        _slot0Hi = _load(0) & ~uint256(type(uint128).max);
        _slot6Hi = _load(6) & ~uint256(type(uint128).max);
        _loanScale = H.LOAN_SCALE();
        uint256[25] memory s = _readState();
        uint256 snap = vm.snapshotState();
        vm.transact(txh);
        uint256[6] memory realPost;
        for (uint256 i; i < 6; ++i) {
            realPost[i] = _load(16 + i);
        }
        require(realPost[3] != s[11], "tx did not fill");
        vm.revertToState(snap);
        (uint256[6] memory c, uint256[] memory outs) = _tapSwap();
        for (uint256 i; i < 6; ++i) {
            require(outs[4 + i] == realPost[i], "tap replay diverged from the transaction");
        }
        _emitRaw("hook.executeExactIn", _args(s, c), true, outs, "");
        vm.revertToState(snap);
        (_firstRow, _rows) = (false, 1); // the snapshot also rewound this contract's writer state
        require(_load(20) == s[11], "revert");
        _fee(s, c);
        _fill(s, c);
        _close();
    }

    function _readState() internal view returns (uint256[25] memory s) {
        s[0] = _load(0) & type(uint128).max;
        for (uint256 i; i < 4; ++i) {
            s[1 + i] = _load(10 + i);
        }
        for (uint256 i; i < 8; ++i) {
            s[5 + i] = _load(14 + i);
        }
        s[13] = _load(23);
        uint256 w4 = _load(4);
        uint256 w5 = _load(5);
        uint256 w6 = _load(6);
        for (uint256 i; i < 4; ++i) {
            s[14 + i] = (w4 >> (64 * i)) & U64;
            s[18 + i] = (w5 >> (64 * i)) & U64;
        }
        (s[22], s[23], s[24]) = (w6 & U64, (w6 >> 64) & U64, _loanScale);
    }

    function _tapSwap() internal returns (uint256[6] memory c, uint256[] memory outs) {
        address taker = 0xbae5642ec964388995e92796ba15E7Fd3ce63484;
        address cbBTC = 0xcbB7C0000aB88B473b1f5aFd9ef808440eed33Bf;
        address usdc = 0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913;
        vm.etch(address(0x00000000000000000000000000000000bE11c0dE), HOOK.code);
        vm.etch(HOOK, address(new HookFillEdgesTap()).code);
        vm.recordLogs();
        vm.startPrank(taker);
        (bool ok,) = cbBTC.call(abi.encodeWithSignature("approve(address,uint256)", POOL, 15000));
        require(ok, "approve");
        bytes memory ret;
        (ok, ret) = POOL.call(
            abi.encodeWithSignature(
                "swap(address,address,uint256,uint256,address,uint256)", cbBTC, usdc, 15000, 14985, taker, 1789995085
            )
        );
        vm.stopPrank();
        if (!ok) {
            assembly ("memory-safe") {
                revert(add(ret, 32), mload(ret))
            }
        }
        outs = new uint256[](10);
        for (uint256 i; i < 6; ++i) {
            outs[4 + i] = _load(16 + i);
        }
        bytes memory data;
        (data, ret) = _tapped();
        uint256[] memory w = _words(ret, 4);
        for (uint256 i; i < 4; ++i) {
            outs[i] = w[i];
        }
        bytes memory body = new bytes(data.length - 4);
        for (uint256 i; i < body.length; ++i) {
            body[i] = data[i + 4];
        }
        (SwapContext memory x, uint256 feeWad) = abi.decode(body, (SwapContext, uint256));
        c = [x.pool.physicalPoolAsset, x.pool.postedPoolAsset, x.poolAssetIn ? 1 : 0, x.amountIn, x.maxAmountOut, feeWad];
    }

    function _tapped() internal returns (bytes memory data, bytes memory ret) {
        VmSafeLog[] memory logs = _logs();
        for (uint256 i; i < logs.length; ++i) {
            if (logs[i].emitter == HOOK && logs[i].topic0 == HookFillEdgesTap.Tap.selector) {
                bool ok;
                (data, ok, ret) = abi.decode(logs[i].data, (bytes, bool, bytes));
                require(ok, "tap exec failed");
            }
        }
        require(data.length > 4, "no tap");
    }

    struct VmSafeLog {
        address emitter;
        bytes32 topic0;
        bytes data;
    }

    function _logs() internal returns (VmSafeLog[] memory out) {
        (bool ok, bytes memory raw) = address(vm).call(abi.encodeWithSignature("getRecordedLogs()"));
        require(ok, "logs");
        Log[] memory logs = abi.decode(raw, (Log[]));
        out = new VmSafeLog[](logs.length);
        for (uint256 i; i < logs.length; ++i) {
            out[i] = VmSafeLog(logs[i].emitter, logs[i].topics.length > 0 ? logs[i].topics[0] : bytes32(0), logs[i].data);
        }
    }

    struct Log {
        bytes32[] topics;
        bytes data;
        address emitter;
    }
}
