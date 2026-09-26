// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import {AlmCurve} from "src/hooks/everlong/AlmCurve.sol";
import {SwapHookEdgesBase} from "./SwapHookEdgesBase.sol";

/// @notice External face over AlmCurve's INTERNAL functions, compiled from the c104 source.
contract AlmCurveEdgesHarness {
    function cWad(uint256 a) external pure returns (int256) {
        return AlmCurve.cWad(a);
    }

    function yAtX(uint256 x, uint256 a) external pure returns (uint256) {
        return AlmCurve.yAtX(x, a);
    }

    function priceAtX(uint256 x, uint256 a) external pure returns (uint256) {
        return AlmCurve.priceAtX(x, a);
    }

    function xAtPrice(uint256 t, uint256 a) external pure returns (uint256) {
        return AlmCurve.xAtPrice(t, a);
    }

    function supportFor(uint256 a, uint256 up, uint256 dn) external pure returns (AlmCurve.Support memory) {
        return AlmCurve.supportFor(a, up, dn);
    }

    function heldAt(AlmCurve.Support memory s, uint256 x) external pure returns (uint256, uint256) {
        return AlmCurve.heldAt(s, x);
    }

    function swapExactIn(AlmCurve.Support memory s, uint256 x, bool volIn, uint256 amt)
        external
        pure
        returns (uint256, uint256, uint256)
    {
        return AlmCurve.swapExactIn(s, x, volIn, amt);
    }
}

/// @notice Edge generator for AlmCurve: domain and amplification edges, the b = 0 branch,
///         the seed-bracket gates, band truncation, and a keyed pseudo-random grid. Internal functions run
///         through the source harness; reservesAt and swapExactInX96 run against the DEPLOYED Base library.
contract AlmCurveEdges is SwapHookEdgesBase {
    address constant ALM = 0xf82DdF0A8a50bc2C3F163997766bA1839E527A17;
    bytes4 constant SEL_RESERVES_AT = 0xfa07504e;
    bytes4 constant SEL_SWAP_X96 = 0x1432583d;
    uint256 constant WAD = 1e18;
    uint256 constant MIN_X = 1e15;
    uint256 constant MAX_X = 1999e15;
    uint256 constant MIN_A = 5e17 + 1;
    uint256 constant MAX_A = 1000e18;
    uint256 constant Q96 = 1 << 96;
    uint256 constant LIVE_ANCHOR = 2196067230099687190109974130321092041;
    uint256 constant LIVE_KAPPA = 12779948558379;

    AlmCurveEdgesHarness internal h;

    function _negC(uint256 a) internal pure returns (uint256) {
        return WAD - (WAD * WAD) / (2 * a);
    }

    function _call1(string memory f, uint256[] memory args, bytes memory cd, uint256 n) internal {
        (bool ok, bytes memory ret) = address(h).call(cd);
        _emit(f, args, ok, ret, n);
    }

    function _y(uint256 x, uint256 a) internal {
        _call1("yAtX", _a2(x, a), abi.encodeCall(h.yAtX, (x, a)), 1);
    }

    function _p(uint256 x, uint256 a) internal {
        _call1("priceAtX", _a2(x, a), abi.encodeCall(h.priceAtX, (x, a)), 1);
    }

    function _sup(uint256 a, uint256 up, uint256 dn) internal returns (bool ok, AlmCurve.Support memory s) {
        bytes memory ret;
        (ok, ret) = address(h).call(abi.encodeCall(h.supportFor, (a, up, dn)));
        _emit("supportFor", _a3(a, up, dn), ok, ret, 4);
        if (ok) s = abi.decode(ret, (AlmCurve.Support));
    }

    function _supArgs(AlmCurve.Support memory s, uint256 extra) internal pure returns (uint256[] memory x) {
        x = new uint256[](4 + extra);
        (x[0], x[1], x[2], x[3]) = (s.aWad, s.xLo, s.xHi, s.yHi);
    }

    function _held(AlmCurve.Support memory s, uint256 x) internal {
        uint256[] memory args = _supArgs(s, 1);
        args[4] = x;
        _call1("heldAt", args, abi.encodeCall(h.heldAt, (s, x)), 2);
    }

    function _swapN(AlmCurve.Support memory s, uint256 x, bool volIn, uint256 amt) internal {
        uint256[] memory args = _supArgs(s, 3);
        (args[4], args[5], args[6]) = (x, volIn ? 1 : 0, amt);
        _call1("swapExactIn", args, abi.encodeCall(h.swapExactIn, (s, x, volIn, amt)), 3);
    }

    function _res(AlmCurve.Support memory s, uint256 anchor, uint256 kappa, uint256 x) internal {
        uint256[] memory args = _supArgs(s, 3);
        (args[4], args[5], args[6]) = (anchor, kappa, x);
        (bool ok, bytes memory ret) =
            ALM.staticcall(abi.encodeWithSelector(SEL_RESERVES_AT, s, uint160(anchor), kappa, x));
        _emit("reservesAt", args, ok, ret, 2);
    }

    function _x96(AlmCurve.Support memory s, uint256 anchor, uint256 kappa, uint256 x, bool stableIn, uint256 amt)
        internal
    {
        uint256[] memory args = _supArgs(s, 5);
        (args[4], args[5], args[6], args[7], args[8]) = (anchor, kappa, x, stableIn ? 1 : 0, amt);
        (bool ok, bytes memory ret) =
            ALM.staticcall(abi.encodeWithSelector(SEL_SWAP_X96, s, uint160(anchor), kappa, x, stableIn, amt));
        _emit("swapExactInX96", args, ok, ret, 3);
    }

    AlmCurve.Support[] internal _sups;

    function _live() internal pure returns (AlmCurve.Support memory) {
        return AlmCurve.Support(34e18, 24096289941794299, 1099912607172170593, 24096289941794300);
    }

    function _full() internal pure returns (AlmCurve.Support memory) {
        return AlmCurve.Support(34e18, MIN_X, MAX_X, 0);
    }

    function _bad() internal pure returns (AlmCurve.Support memory) {
        return AlmCurve.Support(34e18, 9e17, 2e17, 0); // xLo > xHi
    }

    function _badA() internal pure returns (AlmCurve.Support memory) {
        return AlmCurve.Support(5e17, 24096289941794299, 1099912607172170593, 1);
    }

    function _pick(uint256 r, uint256 i, uint256 m) internal view returns (AlmCurve.Support memory) {
        if (_sups.length == 0 || i % m == 0) return _live();
        return _sups[(r >> 32) % _sups.length];
    }

    function _As() internal pure returns (uint256[20] memory) {
        return [
            uint256(0), 5e17, MIN_A, MIN_A + 1, MIN_A + 2, 6e17, 666666666666666667, 1e18, 1e18 + 1, 3e18,
            34e18, 34e18 + 1, 333333333333333333333, MAX_A - 1, MAX_A, MAX_A + 1, 1 << 128, type(uint256).max,
            7e17 + 3, 123456789012345678901
        ];
    }

    function test_curve() public {
        vm.createSelectFork("https://mainnet.base.org", 51310000);
        h = new AlmCurveEdgesHarness();
        _open("test/kyber/fixtures/alm_curve_edges.json");
        _secY();
        _secYRand();
        _secXAtPrice();
        _secSupport();
        _secHeld();
        _secSwapEdges();
        _secSwapRand();
        _secReserves();
        _secX96Edges();
        _secX96Rand();
        _close();
        emit log_named_uint("rows", _rows);
    }

    function _secY() internal {
        uint256[20] memory As = _As();
        for (uint256 i; i < As.length; ++i) {
            _call1("cWad", _a1(As[i]), abi.encodeCall(h.cWad, (As[i])), 1);
        }
        uint256[14] memory xs = [
            uint256(0), MIN_X - 1, MIN_X, MIN_X + 1, 5e17 - 1, 5e17, 5e17 + 1, MAX_X - 1, MAX_X, MAX_X + 1,
            type(uint256).max, 1e18, 1, 2e18
        ];
        for (uint256 i; i < As.length; ++i) {
            for (uint256 j; j < xs.length; ++j) {
                _y(xs[j], As[i]);
                _p(xs[j], As[i]);
            }
            if (As[i] >= MIN_A && As[i] <= MAX_A) {
                uint256 nc = _negC(As[i]);
                for (uint256 d; d < 5; ++d) {
                    _y(nc + d - 2, As[i]);
                    _p(nc + d - 2, As[i]);
                }
            }
        }
    }

    function _secYRand() internal {
        for (uint256 i; i < 360; ++i) {
            uint256 r = _rand(1, i);
            uint256 a = i % 3 == 0
                ? MIN_A + (r >> 64) % (MAX_A - MIN_A + 1)
                : MIN_A + _logRand(r >> 8, 21) % (MAX_A - MIN_A + 1);
            uint256 x = MIN_X + (r >> 128) % (MAX_X - MIN_X + 1);
            if (i % 5 == 0) x = _negC(a) + ((r >> 16) % 7) - 3;
            _y(x, a);
            _p(x, a);
        }
    }

    function _secXAtPrice() internal {
        uint256[6] memory Ax = [MIN_A, 6e17, 1e18, 34e18, MAX_A, 5e17];
        for (uint256 i; i < Ax.length; ++i) {
            uint256[] memory ts = _xTargets(Ax[i]);
            for (uint256 j; j < ts.length; ++j) {
                _call1("xAtPrice", _a2(ts[j], Ax[i]), abi.encodeCall(h.xAtPrice, (ts[j], Ax[i])), 1);
            }
        }
        for (uint256 i; i < 40; ++i) {
            uint256 r = _rand(2, i);
            uint256 a = MIN_A + _logRand(r, 21) % (MAX_A - MIN_A + 1);
            uint256 t = _logRand(r >> 3, 22);
            _call1("xAtPrice", _a2(t, a), abi.encodeCall(h.xAtPrice, (t, a)), 1);
        }
    }

    function _xTargets(uint256 a) internal returns (uint256[] memory ts) {
        (bool okLo, bytes memory rLo) = address(h).call(abi.encodeCall(h.priceAtX, (MIN_X, a)));
        (bool okHi, bytes memory rHi) = address(h).call(abi.encodeCall(h.priceAtX, (MAX_X, a)));
        ts = new uint256[](10);
        ts[1] = 1;
        ts[2] = WAD;
        ts[3] = type(uint256).max;
        if (okLo && okHi) {
            uint256 pLo = abi.decode(rLo, (uint256));
            uint256 pHi = abi.decode(rHi, (uint256));
            (ts[4], ts[5], ts[6]) = (pLo - 1, pLo, pLo + 1);
            (ts[7], ts[8], ts[9]) = (pHi - 1, pHi, pHi + 1);
        }
    }

    function _secSupport() internal {
        uint256[9] memory spans = [WAD - 1, WAD, WAD + 1, 1.0001e18, 1.06e18, 6e18, 1e21, 1e30, type(uint256).max];
        for (uint256 i; i < spans.length; ++i) {
            for (uint256 j; j < spans.length; ++j) {
                if ((i + j) % 2 == 1 && i != 5 && j != 5) continue;
                _supKeep(34e18, spans[i], spans[j]);
            }
        }
        _sup(5e17, 6e18, 6e18);
        _sup(MAX_A + 1, 6e18, 6e18);
        _sup(5e17, WAD, 6e18);
        for (uint256 i; i < 40; ++i) {
            uint256 r = _rand(3, i);
            uint256 a = MIN_A + _logRand(r, 21) % (MAX_A - MIN_A + 1);
            _supKeep(a, WAD + 1 + _logRand(r >> 5, 19), WAD + 1 + _logRand(r >> 9, 19));
        }
    }

    function _supKeep(uint256 a, uint256 up, uint256 dn) internal {
        (bool ok, AlmCurve.Support memory s) = _sup(a, up, dn);
        if (ok) _sups.push(s);
    }

    function _secHeld() internal {
        AlmCurve.Support[4] memory hs = [_live(), _full(), _bad(), _badA()];
        uint256[9] memory hx = [
            uint256(0), 2e17, 24096289941794298, 24096289941794299, 5e17, 9e17, 1099912607172170593,
            1099912607172170594, type(uint256).max
        ];
        for (uint256 i; i < hs.length; ++i) {
            for (uint256 j; j < hx.length; ++j) {
                _held(hs[i], hx[j]);
            }
        }
    }

    function _yOf(uint256 x) internal returns (uint256) {
        return abi.decode(_mustCall(abi.encodeCall(h.yAtX, (x, 34e18))), (uint256));
    }

    function _secSwapEdges() internal {
        AlmCurve.Support memory live = _live();
        uint256[6] memory sx = [live.xLo, live.xLo + 1, uint256(5e17), 532533306204662064, live.xHi - 1, live.xHi];
        for (uint256 i; i < sx.length; ++i) {
            _swapEdgesAt(sx[i]);
        }
        _swapN(live, 0, true, 0);
        _swapN(live, 0, false, 5);
        _swapN(live, 1e15 - 1, true, 7);
        _swapN(live, live.xLo - 5, false, 1e12);
        _swapN(live, live.xHi + 5, true, 1e12);
        _swapN(live, live.xHi + 5, false, 1e12);
        _swapN(_bad(), 5e17, true, 1e12);
        _swapN(_bad(), 5e17, false, 1e12);
        _swapN(_badA(), 5e17, false, 1e12);
        _swapN(_full(), MIN_X, false, 1);
        _swapN(_full(), MAX_X, true, 1);
        _swapN(_full(), MAX_X, false, 1e30);
        _swapN(_full(), MIN_X + 1, false, 1e30);
    }

    function _swapEdgesAt(uint256 x) internal {
        AlmCurve.Support memory live = _live();
        uint256 y = _yOf(x);
        uint256 yLo = _yOf(live.xLo);
        uint256 room = live.xHi - x;
        uint256 reach = yLo > y ? yLo - y : 0;
        uint256[12] memory amts =
            [uint256(0), 1, 2, 65535, 65536, 1e9, room, room + 1, reach, reach + 1, type(uint256).max, 0];
        if (reach > 1) amts[11] = reach - 1;
        for (uint256 j; j < amts.length; ++j) {
            if (amts[j] != 0 || j == 0) {
                _swapN(live, x, true, amts[j]);
                _swapN(live, x, false, amts[j]);
            }
        }
        if (MIN_X > y) {
            _swapN(live, x, false, MIN_X - y - 1);
            _swapN(live, x, false, MIN_X - y);
        }
        if (MAX_X + 1 > y) {
            _swapN(live, x, false, MAX_X - y);
            _swapN(live, x, false, MAX_X - y + 1);
        }
    }

    function _secSwapRand() internal {
        for (uint256 i; i < 420; ++i) {
            uint256 r = _rand(4, i);
            AlmCurve.Support memory s = _pick(r, i, 4);
            uint256 x = s.xLo + (r >> 64) % (s.xHi - s.xLo + 1);
            if (i % 11 == 0) x = s.xLo;
            if (i % 13 == 0) x = s.xHi;
            uint256 amt = i % 7 == 0 ? (r >> 100) % 200 : _logRand(r >> 128, 19);
            _swapN(s, x, (i & 1) == 0, amt);
        }
    }

    function _secReserves() internal {
        uint256[6] memory anchors =
            [uint256(0), 1, Q96, LIVE_ANCHOR, type(uint160).max, 79228162514264337593543950336000];
        uint256[5] memory kappas = [uint256(0), 1, LIVE_KAPPA, 1e30, type(uint256).max];
        AlmCurve.Support memory live = _live();
        for (uint256 i; i < anchors.length; ++i) {
            for (uint256 j; j < kappas.length; ++j) {
                _res(live, anchors[i], kappas[j], 5e17);
                if (j == 2) {
                    _res(live, anchors[i], kappas[j], live.xLo);
                    _res(live, anchors[i], kappas[j], live.xHi);
                    _res(_bad(), anchors[i], kappas[j], 5e17);
                }
            }
        }
        for (uint256 i; i < 120; ++i) {
            uint256 r = _rand(5, i);
            AlmCurve.Support memory s = _pick(r, i, 3);
            uint256 x = (r >> 64) % (MAX_X + 2e17);
            uint256 anchor = i % 2 == 0 ? LIVE_ANCHOR : 1 + _logRand(r >> 16, 48) % type(uint160).max;
            _res(s, anchor, _logRand(r >> 128, 40), x);
        }
    }

    function _x96Amts() internal pure returns (uint256[16] memory) {
        return [
            uint256(0), 1, 2, 9, 10, 1e6, 1e12 - 1, 1e12, 1e12 + 1, 15000, 1e18, 11301759e12, 1e24, 1e30,
            1 << 200, type(uint256).max
        ];
    }

    function _secX96Edges() internal {
        uint256[16] memory amts = _x96Amts();
        uint256[4] memory kappas = [uint256(0), 1, LIVE_KAPPA, 1e30];
        uint256[3] memory anchors = [uint256(0), LIVE_ANCHOR, type(uint160).max];
        AlmCurve.Support memory live = _live();
        for (uint256 k; k < kappas.length; ++k) {
            for (uint256 an; an < anchors.length; ++an) {
                for (uint256 j; j < amts.length; ++j) {
                    if (k != 2 && an != 1 && j % 3 != 0) continue;
                    _x96(live, anchors[an], kappas[k], 532533306204662064, true, amts[j]);
                    _x96(live, anchors[an], kappas[k], 532533306204662064, false, amts[j]);
                }
            }
        }
        uint256[4] memory ex = [live.xLo, live.xLo + 3, live.xHi - 3, live.xHi];
        for (uint256 i; i < ex.length; ++i) {
            for (uint256 j; j < amts.length; ++j) {
                _x96(live, LIVE_ANCHOR, LIVE_KAPPA, ex[i], true, amts[j]);
                _x96(live, LIVE_ANCHOR, LIVE_KAPPA, ex[i], false, amts[j]);
            }
        }
        _x96(live, LIVE_ANCHOR, LIVE_KAPPA, 0, true, 1e18);
        _x96(live, LIVE_ANCHOR, LIVE_KAPPA, 0, true, 0);
        _x96(live, LIVE_ANCHOR, LIVE_KAPPA, MAX_X + 1, false, 1e18);
        _x96(_bad(), LIVE_ANCHOR, LIVE_KAPPA, 5e17, false, 1e18);
        _x96(_badA(), LIVE_ANCHOR, LIVE_KAPPA, 5e17, true, 1e18);
    }

    function _secX96Rand() internal {
        for (uint256 i; i < 480; ++i) {
            uint256 r = _rand(6, i);
            AlmCurve.Support memory s = _pick(r, i, 3);
            uint256 x = s.xLo + (r >> 64) % (s.xHi - s.xLo + 1);
            uint256 anchor = i % 4 == 0 ? LIVE_ANCHOR : (Q96 >> 20) + _logRand(r >> 16, 40);
            uint256 kappa = i % 4 == 1 ? LIVE_KAPPA : 1 + _logRand(r >> 128, 32);
            uint256 amt = i % 9 == 0 ? (r >> 90) % 1000 : _logRand(r >> 180, 30);
            _x96(s, anchor, kappa, x, (i & 1) == 0, amt);
        }
    }

    function _mustCall(bytes memory cd) internal returns (bytes memory ret) {
        bool ok;
        (ok, ret) = address(h).call(cd);
        require(ok, "mustCall");
    }
}
