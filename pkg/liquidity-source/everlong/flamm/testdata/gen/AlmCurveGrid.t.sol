// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import {Test} from "forge-std/Test.sol";

/// @notice Kyber fixture generator: the AlmCurve swap/book math on SHIPPED bytecode. A Base fork pinned at
///         BLOCK; every public library entry is called on the deployed AlmCurve library, and the internal
///         {priceAtX} is read exactly through the live EverlongHook's `spot()` with its price row overwritten
///         (`reservationPriceWad = WAD^2`, so `spot = priceAtX`). {yAtX} is read exactly through
///         `reservesAt` on a full-domain support with `yHi = 0`, `anchor = Q96`, `kappa = WAD`, and the
///         normalized {swapExactIn} through `swapExactInX96` with the same identity scale. Reverts are recorded
///         as raw revert data.
contract KyberAlmCurveGrid is Test {
    address constant ALM = 0xf82DdF0A8a50bc2C3F163997766bA1839E527A17;
    address constant HOOK = 0x65CBD227cBC61248ae77a5fC813A29C54C092134;
    uint256 constant BLOCK = 51310000;
    uint256 constant WAD = 1e18;
    uint256 constant HALF = 5e17;
    uint256 constant MIN_X = 1e15;
    uint256 constant MAX_X = 1999e15;
    uint256 constant Q96 = 1 << 96;
    uint256 constant LIVE_ANCHOR = 2196067230099687190109974130321092041;
    bytes4 constant SEL_SUPPORT_FOR = 0x573f89ed;
    bytes4 constant SEL_RESERVES_AT = 0xfa07504e;
    bytes4 constant SEL_SWAP_X96 = 0x1432583d;
    bytes4 constant SEL_SPOT = 0x310e7c56;
    string constant OUT = "test/kyber/fixtures/alm_curve_grid.json";

    struct Sup {
        uint256 aWad;
        uint256 xLo;
        uint256 xHi;
        uint256 yHi;
    }

    struct Cfg {
        uint128 a;
        uint128 up;
        uint128 dn;
    }

    bool internal _first = true;
    uint256 internal _rows;

    function _row(string memory r) internal {
        vm.writeLine(OUT, string.concat(_first ? "[" : ",", r));
        _first = false;
        ++_rows;
    }

    function _u(uint256 v) internal pure returns (string memory) {
        return string.concat('"', vm.toString(v), '"');
    }

    function _kv(string memory k, uint256 v) internal pure returns (string memory) {
        return string.concat('"', k, '":', _u(v), ",");
    }

    function _res(bool ok, bytes memory ret) internal pure returns (string memory) {
        return string.concat('"ok":', ok ? "true" : "false", ',"err":"', ok ? "0x" : vm.toString(ret), '"');
    }

    function _supJson(Sup memory s) internal pure returns (string memory) {
        return string.concat(
            '"sup":[', _u(s.aWad), ",", _u(s.xLo), ",", _u(s.xHi), ",", _u(s.yHi), "],"
        );
    }

    function _supportFor(uint256 a, uint256 up, uint256 dn) internal view returns (bool ok, Sup memory s, bytes memory ret) {
        (ok, ret) = ALM.staticcall(abi.encodeWithSelector(SEL_SUPPORT_FOR, Cfg(uint128(a), uint128(up), uint128(dn))));
        if (ok) s = abi.decode(ret, (Sup));
    }

    function _reservesAt(Sup memory s, uint256 anchor, uint256 kappa, uint256 x)
        internal
        view
        returns (bool ok, uint256 st, uint256 vol, bytes memory ret)
    {
        (ok, ret) = ALM.staticcall(abi.encodeWithSelector(SEL_RESERVES_AT, s, uint160(anchor), kappa, x));
        if (ok) (st, vol) = abi.decode(ret, (uint256, uint256));
    }

    function _swap(Sup memory s, uint256 anchor, uint256 kappa, uint256 x, bool stableIn, uint256 amt)
        internal
        view
        returns (bool ok, uint256 out, uint256 xAfter, uint256 unspent, bytes memory ret)
    {
        (ok, ret) = ALM.staticcall(abi.encodeWithSelector(SEL_SWAP_X96, s, uint160(anchor), kappa, x, stableIn, amt));
        if (ok) (out, xAfter, unspent) = abi.decode(ret, (uint256, uint256, uint256));
    }

    function _aList() internal pure returns (uint256[6] memory) {
        return [uint256(5e17 + 1), 0.51e18 + 1, 2e18, 34e18, 1000e18, 0.51e18];
    }

    function test_almCurveGrid() public {
        vm.createSelectFork("https://mainnet.base.org", BLOCK);
        if (vm.exists(OUT)) vm.removeFile(OUT);
        this.supportCells();
        _yAndPrice();
        _normalizedSwaps();
        _tokenSwaps();
        vm.writeLine(OUT, "]");
    }

    // ------------------------------------------------------------------ supportFor
    function supportCells() external {
        uint256[6] memory as_ = _aList();
        uint256[2][9] memory spans = [
            [uint256(6e18), 6e18],
            [uint256(4e18), 6e18],
            [uint256(1.0001e18), 1.0001e18],
            [uint256(1.5e18), 100e18],
            [uint256(100e18), 1.5e18],
            [uint256(2e18), 2e18],
            [uint256(1e18), 6e18],
            [uint256(6e18), 1e18],
            [uint256(1e30), 1e30]
        ];
        for (uint256 i; i < 8; ++i) {
            uint256 a = i < 6 ? as_[i] : (i == 6 ? 5e17 : 1000e18 + 1);
            for (uint256 j; j < spans.length; ++j) {
                (bool ok, Sup memory s, bytes memory ret) = _supportFor(a, spans[j][0], spans[j][1]);
                _row(string.concat('{"k":"sup",', _kv("a", a), _kv("up", spans[j][0]), _kv("dn", spans[j][1]), _supJson(s), _res(ok, ret), "}"));
            }
        }
    }

    // ------------------------------------------------------------------ yAtX and priceAtX
    function _xProbe(uint256 a) internal pure returns (uint256[] memory xs) {
        xs = new uint256[](44);
        uint256[24] memory fixed_ = [
            uint256(MIN_X - 1),
            MIN_X,
            MIN_X + 1,
            2e15,
            1e16,
            24096289941794299,
            24096289941794300,
            1e17,
            3e17,
            HALF - 1,
            HALF,
            HALF + 1,
            532533306204662064,
            7e17,
            1e18 - 1,
            1e18,
            1e18 + 1,
            1099912607172170593,
            1.5e18,
            1.998e18,
            MAX_X - 1,
            MAX_X,
            MAX_X + 1,
            0
        ];
        for (uint256 i; i < 24; ++i) xs[i] = fixed_[i];
        for (uint256 i = 24; i < 44; ++i) {
            xs[i] = MIN_X + uint256(keccak256(abi.encode("m1x", a, i))) % (MAX_X - MIN_X + 1);
        }
    }

    function _yAndPrice() internal {
        uint256[6] memory as_ = _aList();
        uint256 slot0 = uint256(vm.load(HOOK, bytes32(uint256(0))));
        uint256 slot15 = uint256(vm.load(HOOK, bytes32(uint256(15))));
        uint256 slot17 = uint256(vm.load(HOOK, bytes32(uint256(17))));
        for (uint256 i; i < 8; ++i) {
            uint256 a = i < 6 ? as_[i] : (i == 6 ? 5e17 : 1000e18 + 1);
            this.yCell(a, slot0);
        }
        vm.store(HOOK, bytes32(uint256(0)), bytes32(slot0));
        vm.store(HOOK, bytes32(uint256(15)), bytes32(slot15));
        vm.store(HOOK, bytes32(uint256(17)), bytes32(slot17));
    }

    function yCell(uint256 a, uint256 slot0) external {
        {
            uint256[] memory xs = _xProbe(a);
            for (uint256 j; j < xs.length; ++j) {
                uint256 x = xs[j];
                // yAtX, exact: full-domain support with zero stable offset at the identity scale
                if (x >= MIN_X && x <= MAX_X) {
                    (bool ok, uint256 y, uint256 v, bytes memory ret) = _reservesAt(Sup(a, MIN_X, MAX_X, 0), Q96, WAD, x);
                    _row(string.concat('{"k":"y",', _kv("a", a), _kv("x", x), _kv("y", y), _kv("held", v), _res(ok, ret), "}"));
                }
                // priceAtX, exact: the live hook's spot with reservationPriceWad = WAD^2
                vm.store(HOOK, bytes32(uint256(0)), bytes32((slot0 >> 128) << 128 | a));
                vm.store(HOOK, bytes32(uint256(15)), bytes32(WAD * WAD));
                vm.store(HOOK, bytes32(uint256(17)), bytes32(x));
                (bool ok2, bytes memory ret2) =
                    HOOK.staticcall(abi.encodeWithSelector(SEL_SPOT, uint256(0), uint256(0), uint256(0), uint256(0), uint256(0), uint256(0), uint256(0), uint256(0), uint256(0)));
                uint256 p = ok2 ? abi.decode(ret2, (uint256)) : 0;
                _row(string.concat('{"k":"p",', _kv("a", a), _kv("x", x), _kv("p", p), _res(ok2, ret2), "}"));
            }
        }
    }

    // ------------------------------------------------------------------ normalized swapExactIn (identity scale)
    function _swapSupports() internal pure returns (uint256[3][10] memory c) {
        c = [
            [uint256(34e18), 6e18, 6e18],
            [uint256(34e18), 4e18, 6e18],
            [uint256(34e18), 1.0001e18, 1.0001e18],
            [uint256(5e17 + 1), 6e18, 6e18],
            [uint256(5e17 + 1), 1.0001e18, 1.0001e18],
            [uint256(0.51e18 + 1), 2e18, 2e18],
            [uint256(2e18), 4e18, 6e18],
            [uint256(2e18), 100e18, 1.5e18],
            [uint256(1000e18), 6e18, 6e18],
            [uint256(1000e18), 1.5e18, 100e18]
        ];
    }

    function _normalizedSwaps() internal {
        uint256[3][10] memory cs = _swapSupports();
        for (uint256 i; i < cs.length; ++i) {
            (bool okS,,) = _supportFor(cs[i][0], cs[i][1], cs[i][2]);
            if (!okS) continue;
            for (uint256 j; j < 12; ++j) {
                this.normCell(i, j);
            }
        }
    }

    function normCell(uint256 i, uint256 j) external {
        uint256[3][10] memory cs = _swapSupports();
        (, Sup memory s,) = _supportFor(cs[i][0], cs[i][1], cs[i][2]);
        uint256[12] memory xs = [
            s.xLo,
            s.xLo + 1,
            s.xLo + (s.xHi - s.xLo) / 7,
            HALF,
            (s.xLo + s.xHi) / 2,
            s.xHi - (s.xHi - s.xLo) / 5,
            s.xHi - 1,
            s.xHi,
            s.xLo - 1,
            s.xHi + 1,
            MIN_X - 1,
            MAX_X + 1
        ];
        uint256 x = xs[j];
        // yMax = y(xLo) via the identity-scale reservesAt trick
        (, uint256 yMax,,) = _reservesAt(Sup(s.aWad, MIN_X, MAX_X, 0), Q96, WAD, s.xLo);
        uint256 yx;
        if (x >= MIN_X && x <= MAX_X) (, yx,,) = _reservesAt(Sup(s.aWad, MIN_X, MAX_X, 0), Q96, WAD, x);
        _normAmounts(s, x, true, yMax > yx ? yMax - yx : 0);
        _normAmounts(s, x, false, s.xHi > x ? s.xHi - x : 0);
    }

    function _normAmounts(Sup memory s, uint256 x, bool stableIn, uint256 reach) internal {
        uint256[16] memory amts = [
            uint256(0),
            1,
            2,
            1e3,
            1e9,
            1e12,
            1e15,
            1e16,
            1e17,
            reach > 0 ? reach - 1 : 3,
            reach,
            reach + 1,
            reach / 2 + 17,
            5e17,
            2e18,
            1e30
        ];
        for (uint256 k; k < amts.length; ++k) {
            _tokenRow(s, Q96, WAD, x, stableIn, amts[k]);
        }
    }

    // ------------------------------------------------------------------ token-space swapExactInX96 and reservesAt
    function _tokenSwaps() internal {
        uint256[3][3] memory cs = [[uint256(34e18), 6e18, 6e18], [uint256(2e18), 4e18, 6e18], [uint256(1000e18), 1.5e18, 100e18]];
        uint256 liveKappa = uint256(vm.load(HOOK, bytes32(uint256(16))));
        uint256[4] memory anchors = [LIVE_ANCHOR, Q96, uint256(1) << 60, uint256(1) << 150];
        uint256[5] memory kappas = [uint256(1e30), liveKappa, WAD, 1, uint256(1) << 200];
        for (uint256 i; i < cs.length; ++i) {
            (bool okS, Sup memory s,) = _supportFor(cs[i][0], cs[i][1], cs[i][2]);
            if (!okS) continue;
            uint256[7] memory xs = [s.xLo, HALF, 532533306204662064, s.xHi - 1, s.xHi, s.xLo - 1, s.xHi + 1];
            for (uint256 ai; ai < anchors.length; ++ai) {
                for (uint256 ki; ki < kappas.length; ++ki) {
                    this.tokenCell(s, anchors[ai], kappas[ki], xs);
                }
            }
            // retracted book and zero anchor reverts
            _tokenRow(s, LIVE_ANCHOR, 0, HALF, true, 1e18);
            _tokenRow(s, LIVE_ANCHOR, 0, HALF, false, 1e5);
            _tokenRow(s, 0, 1e30, HALF, true, 1e18);
            (bool ok0, uint256 st0, uint256 v0, bytes memory r0) = _reservesAt(s, 0, 1e30, HALF);
            _resRow(s, 0, 1e30, HALF, ok0, st0, v0, r0);
        }
    }

    function tokenCell(Sup memory s, uint256 anchor, uint256 kappa, uint256[7] memory xs) external {
        (bool okA, uint256 sMax,,) = _reservesAt(s, anchor, kappa, s.xLo);
        (bool okB,, uint256 vMax,) = _reservesAt(s, anchor, kappa, s.xHi);
        if (!okA) sMax = 1e18;
        if (!okB) vMax = 1e8;
        for (uint256 j; j < xs.length; ++j) {
            _tokenRes(s, anchor, kappa, xs[j]);
            if (j >= 5) continue; // out-of-band coordinates only for the book read
            _tokenAmounts(s, anchor, kappa, xs[j], true, sMax);
            _tokenAmounts(s, anchor, kappa, xs[j], false, vMax);
        }
    }

    function _tokenRes(Sup memory s, uint256 anchor, uint256 kappa, uint256 x) internal {
        (bool ok, uint256 st, uint256 v, bytes memory ret) = _reservesAt(s, anchor, kappa, x);
        _resRow(s, anchor, kappa, x, ok, st, v, ret);
    }

    function _tokenAmounts(Sup memory s, uint256 anchor, uint256 kappa, uint256 x, bool stableIn, uint256 m) internal {
        uint256[12] memory amts =
            [uint256(0), 1, m / 1e12, m / 1e6, m / 1000, m / 10, m / 2, m > 0 ? m - 1 : 5, m, m + 1, m * 2 + 3, uint256(1) << 255];
        for (uint256 k; k < amts.length; ++k) {
            _tokenRow(s, anchor, kappa, x, stableIn, amts[k]);
        }
    }

    function _tokenRow(Sup memory s, uint256 anchor, uint256 kappa, uint256 x, bool stableIn, uint256 amt) internal {
        (bool ok, uint256 out, uint256 xa, uint256 un, bytes memory ret) = _swap(s, anchor, kappa, x, stableIn, amt);
        _row(
            string.concat(
                '{"k":"swap",',
                _supJson(s),
                _kv("anchor", anchor),
                _kv("kappa", kappa),
                _kv("x", x),
                '"stableIn":',
                stableIn ? "true," : "false,",
                _kv("amt", amt),
                _kv("out", out),
                _kv("xAfter", xa),
                _kv("unspent", un),
                _res(ok, ret),
                "}"
            )
        );
    }

    function _resRow(Sup memory s, uint256 anchor, uint256 kappa, uint256 x, bool ok, uint256 st, uint256 v, bytes memory ret)
        internal
    {
        _row(
            string.concat(
                '{"k":"res",', _supJson(s), _kv("anchor", anchor), _kv("kappa", kappa), _kv("x", x), _kv("stable", st), _kv("volatile", v), _res(ok, ret), "}"
            )
        );
    }
}
