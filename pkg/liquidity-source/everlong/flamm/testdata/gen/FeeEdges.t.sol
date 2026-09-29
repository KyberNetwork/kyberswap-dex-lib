// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import {EverlongStrategy as S} from "src/hooks/everlong/EverlongStrategy.sol";
import {FixedPointMathLib} from "solady/utils/FixedPointMathLib.sol";
import {SwapHookEdgesBase} from "./SwapHookEdgesBase.sol";

/// @notice External face over the internal fee law and Solady lnWad, compiled from the c104 source.
contract FeeEdgesHarness {
    function lnWad(uint256 x) external pure returns (uint256) {
        return uint256(FixedPointMathLib.lnWad(int256(x)));
    }

    function logRatioAbsWad(uint256 a, uint256 b) external pure returns (uint256) {
        return S.logRatioAbsWad(a, b);
    }

    function reductionG(uint256 rs, uint256 rv, uint256 anchor, uint256 gamma) external pure returns (uint256, uint256) {
        return S.reductionG(S.FeeState(rs, rv, anchor, 0, 0), gamma);
    }

    function volMultiplier(uint64[8] memory f, uint256 rv, uint256 disloc) external pure returns (uint256) {
        return S.volMultiplier(_p(f), rv, disloc);
    }

    function fillFee(uint64[8] memory f, uint256[5] memory st, bool loanIn, uint256 k, uint256 band)
        external
        pure
        returns (uint256)
    {
        return S.fillFee(_p(f), S.FeeState(st[0], st[1], st[2], st[3], st[4]), loanIn, k, band);
    }

    function _p(uint64[8] memory f) internal pure returns (S.FeeParams memory) {
        return S.FeeParams(f[0], f[1], f[2], f[3], f[4], f[5], f[6], f[7]);
    }
}

/// @notice Edge generator for the fee law: lnWad at every power-of-two boundary, the
///         int256 sign flip, reductionG overflow/zero-denominator panics, volMultiplier clamps and overflow,
///         and fillFee at the tie, the ramp and band thresholds (1-wei neighbours), plus a keyed random grid.
contract FeeEdges is SwapHookEdgesBase {
    uint256 constant WAD = 1e18;
    uint256 constant HALF = 5e17;
    uint256 constant U64 = type(uint64).max;

    FeeEdgesHarness internal h;

    function _ln(uint256 x) internal {
        (bool ok, bytes memory ret) = address(h).call(abi.encodeCall(h.lnWad, (x)));
        _emit("lnWad", _a1(x), ok, ret, 1);
    }

    function _lr(uint256 a, uint256 b) internal {
        (bool ok, bytes memory ret) = address(h).call(abi.encodeCall(h.logRatioAbsWad, (a, b)));
        _emit("logRatioAbsWad", _a2(a, b), ok, ret, 1);
    }

    function _rg(uint256 rs, uint256 rv, uint256 anchor, uint256 gamma) internal {
        uint256[] memory args = new uint256[](4);
        (args[0], args[1], args[2], args[3]) = (rs, rv, anchor, gamma);
        (bool ok, bytes memory ret) = address(h).call(abi.encodeCall(h.reductionG, (rs, rv, anchor, gamma)));
        _emit("reductionG", args, ok, ret, 2);
    }

    function _vm(uint64[8] memory f, uint256 rv, uint256 disloc) internal {
        uint256[] memory args = new uint256[](10);
        for (uint256 i; i < 8; ++i) {
            args[i] = f[i];
        }
        (args[8], args[9]) = (rv, disloc);
        (bool ok, bytes memory ret) = address(h).call(abi.encodeCall(h.volMultiplier, (f, rv, disloc)));
        _emit("volMultiplier", args, ok, ret, 1);
    }

    function _ff(uint64[8] memory f, uint256[5] memory st, bool loanIn, uint256 k, uint256 band) internal {
        uint256[] memory args = new uint256[](16);
        for (uint256 i; i < 8; ++i) {
            args[i] = f[i];
        }
        for (uint256 i; i < 5; ++i) {
            args[8 + i] = st[i];
        }
        (args[13], args[14], args[15]) = (loanIn ? 1 : 0, k, band);
        (bool ok, bytes memory ret) = address(h).call(abi.encodeCall(h.fillFee, (f, st, loanIn, k, band)));
        _emit("fillFee", args, ok, ret, 1);
    }

    function _liveRow() internal pure returns (uint64[8] memory) {
        return [uint64(0.03e18), 0.005e18, 0.05e18, 0.0004e18, 4e18, 0.5e18, 2e18, 0.15e18];
    }

    function test_fee() public {
        h = new FeeEdgesHarness();
        _open("test/kyber/fixtures/fee_edges.json");
        _secLn();
        _secLogRatio();
        _secReductionG();
        _secVol();
        _secFillEdges();
        _secFillRand();
        _close();
        emit log_named_uint("rows", _rows);
    }

    function _secLn() internal {
        _ln(0);
        _ln(type(uint256).max);
        _ln(1 << 255);
        _ln((1 << 255) - 1);
        _ln((1 << 255) + 1);
        _ln(WAD - 1);
        _ln(WAD);
        _ln(WAD + 1);
        _ln(2718281828459045235);
        for (uint256 k; k < 255; ++k) {
            uint256 p = 1 << k;
            if (p > 1) _ln(p - 1);
            _ln(p);
            if (k % 3 == 0) _ln(p + 1);
        }
        // every top-byte pattern of Solady's log2 lookup, at two magnitudes
        for (uint256 v = 1; v < 256; ++v) {
            _ln(v << 60);
            _ln(v << 180);
        }
        for (uint256 i; i < 300; ++i) {
            uint256 r = _rand(11, i);
            uint256 x = i % 2 == 0 ? r >> ((r & 0xff) % 256) : _logRand(r, 76);
            _ln(x);
        }
    }

    function _secLogRatio() internal {
        _lr(0, 0);
        _lr(0, WAD);
        _lr(WAD, 0);
        _lr(WAD, WAD);
        _lr(1, type(uint256).max); // ratio floors to zero
        _lr(1, WAD + 1);
        _lr(1, WAD);
        _lr(type(uint256).max, 1); // mulDiv overflow
        _lr(1 << 255, WAD); // ratio reads negative
        _lr((1 << 255) - 1, WAD);
        _lr(type(uint256).max / WAD, 1);
        _lr(768302232848967000000000000000000, 768302232848967000000000000000000 + 1);
        _lr(768302232848967000000000000000000 - 1, 768302232848967000000000000000000);
        for (uint256 i; i < 200; ++i) {
            uint256 r = _rand(12, i);
            uint256 b = 1 + _logRand(r, 40);
            uint256 a = i % 3 == 0 ? _sub(b + (r >> 100) % 1000, 500) : _logRand(r >> 7, 60);
            _lr(a, b);
        }
    }

    function _secReductionG() internal {
        _rg(0, 1, WAD, WAD);
        _rg(1, 0, WAD, WAD);
        _rg(1, 1, 0, WAD);
        _rg(1, 1, WAD - 1, WAD); // vv floors to zero
        _rg(1, 1, WAD, 0); // K == WAD, gamma 0: zero denominator
        _rg(1e18, 1e18, WAD, 0);
        _rg(1, 2, WAD, 0);
        _rg(1, 1, WAD, 1);
        _rg(1, 1, WAD, U64);
        _rg(1, 2, WAD, U64);
        _rg(1 << 254, 1, WAD, WAD); // 4*vs overflow
        _rg((1 << 254) - 1, 1, WAD, WAD);
        _rg(1 << 255, 1 << 255, WAD, WAD); // total overflow
        _rg(type(uint256).max, 0, WAD, WAD); // one-sided after summing
        _rg(type(uint256).max, 1, WAD, WAD); // total overflow first
        _rg(1, type(uint256).max, 2 * WAD, WAD); // vv mulDiv overflow
        _rg(1, type(uint256).max, WAD, WAD); // total overflow with a small stable leg (4*vs fits)
        _rg(3, type(uint256).max - 2, WAD, WAD); // total == 2^256 exactly
        _rg(157080117684030704791, 234423, 768302232848967000000000000000000, 5e16);
        _rg(157080117684030704791, 234423, 768302232848967000000000000000000, 0);
        _rg(type(uint256).max / 8, type(uint256).max / 8, WAD, U64);
        for (uint256 i; i < 150; ++i) {
            uint256 r = _rand(13, i);
            _rg(_logRand(r, 40), _logRand(r >> 11, 40), _logRand(r >> 23, 36), i % 5 == 0 ? 0 : _logRand(r >> 60, 19) % (U64 + 1));
        }
    }

    function _secVol() internal {
        uint64[8] memory f = _liveRow();
        _vm(f, 0, 0);
        _vm(f, 1.6e11, 0);
        _vm(f, 1.6e11, 1e18);
        _vm(f, type(uint256).max, 0);
        _vm(f, type(uint256).max, type(uint256).max);
        _vm(f, 0, type(uint256).max); // beta*disloc/WAD overflows
        f[3] = 0;
        _vm(f, 5e13, 5e17); // disabled: WAD unclamped
        f = _liveRow();
        f[4] = 1e18;
        _vm(f, 0, type(uint256).max - WAD); // WAD + boost overflow
        _vm(f, 0, type(uint256).max - WAD + 1);
        f = _liveRow();
        f[5] = 2e18;
        f[6] = 1e18; // vMin > vMax
        _vm(f, 1.6e11, 0);
        _vm(f, 1e20, 0);
        // exact clamp boundaries: sigma = 4e14 * ratio/1e9 ... rv = (ratio * sigmaRef / 1e9)^2 at beta 0
        f = [uint64(0.03e18), 0.005e18, 0.05e18, 1e9, 0, 5e17, 2e18, 0.15e18];
        _vm(f, 0.25e18 * 1, 0); // sigma = 5e17 * 1e9 -> ratio 5e17 == vMin
        _vm(f, 0.25e18 - 1, 0);
        _vm(f, 4e18, 0); // ratio == vMax
        _vm(f, 4e18 + 1e10, 0);
        for (uint256 i; i < 120; ++i) {
            uint256 r = _rand(14, i);
            uint64[8] memory g;
            for (uint256 j; j < 8; ++j) {
                g[j] = uint64(_logRand(_rand(r, j), 19) % (U64 + 1));
            }
            _vm(g, _logRand(r >> 5, 40), _logRand(r >> 77, 22));
        }
    }

    function _rows5(uint256 rs, uint256 rv, uint256 anchor, uint256 spot, uint256 rvw)
        internal
        pure
        returns (uint256[5] memory)
    {
        return [rs, rv, anchor, spot, rvw];
    }

    function _secFillEdges() internal {
        uint256[9] memory bands = [uint256(0), 1, 3e16 - 1, 3e16, 3e16 + 1, 6e16, 49e16, 5e17, U64];
        uint256[4] memory kappas = [uint256(0), 1, 4e18, 16e18];
        for (uint256 ri; ri < 6; ++ri) {
            for (uint256 bi; bi < bands.length; ++bi) {
                uint256 band = bands[bi];
                uint256 ramp = band > 3e16 ? band : 3e16;
                // w targets (total = WAD, anchor = WAD so w = rv): tie, +-1, ramp and band neighbours both sides
                uint256[12] memory ws = [
                    HALF, HALF + 1, HALF - 1, _sub(HALF + ramp, 0), _sub(HALF + ramp, 1), HALF + ramp + 1,
                    _sub(HALF, ramp), _sub(HALF + 1, ramp), _sub(HALF, ramp + 1), HALF + band, HALF + band + 1,
                    _sub(HALF, band + 1)
                ];
                for (uint256 wi; wi < ws.length; ++wi) {
                    uint256 w = ws[wi] > WAD ? WAD - 1 : ws[wi];
                    if (w == 0) w = 1;
                    uint256[5] memory st = _rows5(WAD - w, w, WAD, 1.01e18, 1.6e11);
                    uint256 k = kappas[(ri + bi + wi) % kappas.length];
                    if (ri == 5 || bi % 3 == 0 || wi < 3) {
                        _ff(_edgeRow(ri), st, true, k, band);
                        _ff(_edgeRow(ri), st, false, k, band);
                    }
                }
            }
        }
        uint64[8] memory f = _liveRow();
        _ff(f, _rows5(0, 1, WAD, WAD, 0), true, 4e18, 6e16); // one-sided: outFee bare
        _ff(f, _rows5(1, 1, 0, WAD, 0), false, 4e18, 6e16);
        _ff(f, _rows5(1, 1, WAD, 0, 0), false, 4e18, 6e16); // spot 0
        _ff(f, _rows5(1, 1, WAD, 1 << 255, 0), false, 4e18, 6e16);
        _ff(f, _rows5(1, 1, WAD, type(uint256).max, 0), false, 4e18, 6e16); // logRatio overflow
        _ff(f, _rows5(1, 2, WAD, WAD, 0), true, type(uint256).max, 0); // surcharge mulDiv overflow
        _ff(f, _rows5(1, 2, WAD, WAD, 0), true, type(uint256).max / 2, 0); // surcharge add overflow
        _ff(f, _rows5(1, 1, WAD, WAD, type(uint256).max), true, 0, 0);
        _ff(f, _rows5(1e18, 1e18, WAD, WAD, 0), true, 0, 0);
        _ff(_rows6gamma0(), _rows5(1e18, 1e18, WAD, WAD, 0), true, 0, 0); // K == WAD, gamma 0 panics
        _ff(f, _rows5(157080117684030704791, 234423, 768302232848967000000000000000000, 768302232848967000000000000000000, 0), false, 4e18, 6e16);
        _ff(f, _rows5(157080117684030704791, 234423, 768302232848967000000000000000000, 768302232848967000000000000000000, 0), true, 4e18, 6e16);
    }

    function _edgeRow(uint256 i) internal pure returns (uint64[8] memory f) {
        if (i == 0) return _liveRow();
        if (i == 1) return [uint64(0.03e18), 0.005e18, 0, 0.0004e18, 4e18, 0.5e18, 2e18, 0.15e18]; // gamma 0
        if (i == 2) return [uint64(0.005e18), 0.03e18, 0.05e18, 0.0004e18, 4e18, 0.5e18, 2e18, 0.15e18]; // mid < out
        if (i == 3) return [uint64(0.9e18), 0.5e18, 1e18, 0, 0, 0, 0, 1e18]; // skew == WAD
        if (i == 4) return [uint64(0.9e18), 0.5e18, 1e18, 0, 0, 0, 0, 1e18 + 1]; // skew > WAD
        return [uint64(1e18), 1e18, uint64(U64), 1, uint64(U64), 0, uint64(U64), uint64(U64)]; // extremes
    }

    function _sub(uint256 a, uint256 b) internal pure returns (uint256) {
        return a > b ? a - b : 1;
    }

    function _rows6gamma0() internal pure returns (uint64[8] memory) {
        return [uint64(0.03e18), 0.005e18, 0, 0, 0, 0, 0, 0];
    }

    function _secFillRand() internal {
        for (uint256 i; i < 500; ++i) {
            uint256 r = _rand(15, i);
            uint64[8] memory f;
            uint256 mid = (r >> 8) % 2e17;
            uint256 out = i % 17 == 0 ? mid + 1 : (mid == 0 ? 0 : (r >> 72) % (mid + 1));
            f[0] = uint64(mid);
            f[1] = uint64(out);
            f[2] = uint64(i % 9 == 0 ? 0 : (r >> 96) % 1.2e18);
            f[3] = uint64(i % 7 == 0 ? 0 : 1 + (r >> 130) % 1e16);
            f[4] = uint64((r >> 150) % 8e18);
            f[5] = uint64((r >> 170) % 1e18);
            f[6] = uint64(f[5] + (r >> 190) % 8e18);
            f[7] = uint64((r >> 210) % (i % 13 == 0 ? 2e18 : 1e18));
            uint256 r2 = _rand(16, i);
            uint256 anchor = 1 + _logRand(r2, 36);
            uint256 rs = _logRand(r2 >> 13, 30);
            uint256 rv = _logRand(r2 >> 29, 30);
            uint256 spot = i % 5 == 0 ? anchor : _logRand(r2 >> 47, 36);
            uint256 rvw = _logRand(r2 >> 71, 18);
            uint256 k = (r2 >> 99) % 17e18;
            uint256 band = i % 4 == 0 ? 0 : (r2 >> 150) % 5e17;
            _ff(f, _rows5(rs, rv, anchor, spot, rvw), (i & 1) == 0, k, band);
        }
    }
}
