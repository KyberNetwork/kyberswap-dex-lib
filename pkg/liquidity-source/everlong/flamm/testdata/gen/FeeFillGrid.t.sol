// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import {Test} from "forge-std/Test.sol";
import {EverlongStrategy as S} from "src/hooks/everlong/EverlongStrategy.sol";

/// @notice Thin external face over the internal EverlongStrategy fee law, compiled from the c104 source.
contract KyberFeeHarness {
    function fillFee(S.FeeParams memory p, S.FeeState memory s, bool loanAssetIn, uint256 k, uint256 b)
        external
        pure
        returns (uint256)
    {
        return S.fillFee(p, s, loanAssetIn, k, b);
    }

    function reductionG(S.FeeState memory s, uint256 gammaWad) external pure returns (uint256, uint256) {
        return S.reductionG(s, gammaWad);
    }

    function volMultiplier(S.FeeParams memory p, uint256 rvWad, uint256 dislocWad) external pure returns (uint256) {
        return S.volMultiplier(p, rvWad, dislocWad);
    }

    function logRatioAbsWad(uint256 a, uint256 b) external pure returns (uint256) {
        return S.logRatioAbsWad(a, b);
    }
}

/// @notice Kyber fixture generator: EverlongStrategy.fillFee (and its helpers) over weights at and around
///         the directional tie, both directions, curvature and volatility on/off, surcharge rows, one-sided
///         books and dislocated spots. Reverts are recorded as raw revert data.
contract KyberFeeFillGrid is Test {
    uint256 constant WAD = 1e18;
    uint256 constant HALF = 5e17;
    uint256 constant LIVE_RP = 768302232848967000000000000000000;
    string constant OUT = "test/kyber/fixtures/fee_fill_grid.json";

    KyberFeeHarness internal h;
    bool internal _first = true;
    S.FeeParams[] internal _ps;

    function _row(string memory r) internal {
        vm.writeLine(OUT, string.concat(_first ? "[" : ",", r));
        _first = false;
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

    function _params() internal {
        // 0: the sealed c104 row
        _ps.push(S.FeeParams(0.030e18, 0.005e18, 0.05e18, 0.0004e18, 4e18, 0.5e18, 2e18, 0.15e18));
        // 1: zero curvature (every other term live, so the short-circuit is what is measured)
        _ps.push(S.FeeParams(0.030e18, 0.005e18, 0, 0.0004e18, 4e18, 0.5e18, 2e18, 0.15e18));
        // 2: volatility disabled
        _ps.push(S.FeeParams(0.030e18, 0.005e18, 0.05e18, 0, 4e18, 0, 0, 0.15e18));
        // 3: full curvature, wide vol envelope, extreme skew
        _ps.push(S.FeeParams(0.05e18, 0.001e18, 1e18, 0.0001e18, 0, 0, 8e18, 0.99e18));
        // 4: fees near 100% so the final clamp binds
        _ps.push(S.FeeParams(0.9e18, 0.5e18, 1e18, 0.0004e18, 1e18, 0.5e18, 8e18, 0.5e18));
        // 5: malformed rows the law does not validate (out > mid; skew > WAD)
        _ps.push(S.FeeParams(0.005e18, 0.030e18, 0.05e18, 0.0004e18, 4e18, 0.5e18, 2e18, 0.15e18));
        _ps.push(S.FeeParams(0.030e18, 0.005e18, 0.05e18, 0.0004e18, 4e18, 0.5e18, 2e18, 1.5e18));
        for (uint256 i; i < _ps.length; ++i) {
            S.FeeParams memory p = _ps[i];
            _row(
                string.concat(
                    '{"k":"params","i":"',
                    vm.toString(i),
                    '","v":[',
                    string.concat(_u(p.midFeeWad), ",", _u(p.outFeeWad), ",", _u(p.gammaWad), ",", _u(p.sigmaRefWad), ","),
                    string.concat(_u(p.volBetaWad), ",", _u(p.volMinWad), ",", _u(p.volMaxWad), ",", _u(p.dirSkewWad)),
                    "]}"
                )
            );
        }
    }

    function _fee(uint256 pi, S.FeeState memory s, bool loanIn, uint256 k, uint256 b) internal {
        (bool ok, bytes memory ret) = address(h).staticcall(abi.encodeCall(KyberFeeHarness.fillFee, (_ps[pi], s, loanIn, k, b)));
        (bool okG, bytes memory retG) = address(h).staticcall(abi.encodeCall(KyberFeeHarness.reductionG, (s, _ps[pi].gammaWad)));
        uint256 f = ok ? abi.decode(ret, (uint256)) : 0;
        (uint256 g, uint256 w) = okG ? abi.decode(retG, (uint256, uint256)) : (0, 0);
        _row(
            string.concat(
                string.concat('{"k":"fee","p":"', vm.toString(pi), '",', _kv("kap", k), _kv("band", b)),
                string.concat(_kv("rs", s.reserveStable), _kv("rv", s.reserveVolatile), _kv("an", s.anchorWad)),
                string.concat(_kv("sp", s.spotWad), _kv("rvw", s.rvWad), '"loanIn":', loanIn ? "true," : "false,"),
                string.concat(_kv("f", f), _kv("g", g), _kv("w", w), '"gOk":', okG ? "true," : "false,", _res(ok, ret), "}")
            )
        );
    }

    function _book(uint256 w, uint256 bookKind) internal pure returns (uint256 rs, uint256 rv, uint256 an) {
        if (bookKind == 0) {
            // anchor WAD and total WAD: the weight is exact
            return (WAD - w, w, WAD);
        }
        // live-frame book: 219423 sats at the live reservation price, stable leg sized to the target weight
        rv = 219423;
        an = LIVE_RP;
        uint256 vv = rv * an / WAD;
        rs = w == 0 ? 1e20 : vv * (WAD - w) / w;
    }

    function test_feeGrid() public {
        h = new KyberFeeHarness();
        if (vm.exists(OUT)) vm.removeFile(OUT);
        _params();
        uint256[2][6] memory kbs = [
            [uint256(0), 0],
            [uint256(4e18), 0.06e18],
            [uint256(6e18), 0.02e18],
            [uint256(16e18), 0.49e18],
            [uint256(4e18), 0.03e18],
            [uint256(1e18), 0]
        ];
        for (uint256 pi; pi < 5; ++pi) {
            for (uint256 kb; kb < kbs.length; ++kb) {
                _kbCell(pi, kbs[kb][0], kbs[kb][1]);
            }
        }
        _specials();
        this.lnCells();
        this.volCells();
        vm.writeLine(OUT, "]");
    }

    function _kbCell(uint256 pi, uint256 k, uint256 b) internal {
        uint256 ramp = b > 3e16 ? b : 3e16;
        uint256[10] memory ds = [uint256(0), 1, ramp / 2, ramp - 1, ramp, ramp + 1, 2 * ramp, 0.49e18, b, b + 1];
        for (uint256 i; i < ds.length; ++i) {
            if (ds[i] >= HALF) continue;
            for (uint256 sign; sign < 2; ++sign) {
                if (ds[i] == 0 && sign == 1) continue;
                uint256 w = sign == 0 ? HALF + ds[i] : HALF - ds[i];
                for (uint256 bk; bk < 2; ++bk) {
                    _wCell(pi, k, b, w, bk);
                }
            }
        }
    }

    function _wCell(uint256 pi, uint256 k, uint256 b, uint256 w, uint256 bk) internal {
        (uint256 rs, uint256 rv, uint256 an) = _book(w, bk);
        uint256 n = pi == 0 ? 6 : 2;
        for (uint256 c; c < n; ++c) {
            (uint256 sp, uint256 rvw) = _spotRv(an, c);
            S.FeeState memory s = S.FeeState(rs, rv, an, sp, rvw);
            _fee(pi, s, true, k, b);
            _fee(pi, s, false, k, b);
        }
    }

    function _spotRv(uint256 an, uint256 c) internal pure returns (uint256 sp, uint256 rvw) {
        if (c == 0) return (an, 0);
        if (c == 1) return (an * 103 / 100, 1.6e11);
        if (c == 2) return (an / 2, 1e14);
        if (c == 3) return (an * 100, 4e8);
        if (c == 4) return (1, 1e18);
        return (0, 1.6e11);
    }

    function _specials() internal {
        // one-sided and floored-away books
        _fee(0, S.FeeState(0, 1e18, WAD, WAD, 0), true, 4e18, 0.06e18);
        _fee(0, S.FeeState(0, 1e18, WAD, WAD, 0), false, 4e18, 0.06e18);
        _fee(0, S.FeeState(1e18, 0, WAD, WAD, 0), true, 4e18, 0.06e18);
        _fee(0, S.FeeState(1e18, 1, 1, WAD, 0), false, 4e18, 0.06e18);
        _fee(0, S.FeeState(1, 1, WAD, WAD, 0), true, 4e18, 0.06e18);
        _fee(0, S.FeeState(1, 2, WAD, WAD, 0), false, 4e18, 0.06e18);
        // zero curvature on an exactly balanced book: gK + WAD - K == 0
        _fee(1, S.FeeState(5e17, 5e17, WAD, WAD, 0), true, 4e18, 0.06e18);
        _fee(1, S.FeeState(7, 7, WAD, WAD, 0), false, 0, 0);
        _fee(1, S.FeeState(5e17, 5e17 + 1, WAD, WAD, 0), false, 0, 0);
        // malformed rows
        _fee(5, S.FeeState(4e17, 6e17, WAD, WAD, 0), true, 4e18, 0.06e18);
        _fee(6, S.FeeState(4e17, 6e17, WAD, WAD, 0), true, 4e18, 0.06e18);
        _fee(6, S.FeeState(4e17, 6e17, WAD, WAD, 0), false, 4e18, 0.06e18);
        // logRatio beyond int256 (lnWad undefined) and a mulDiv overflow in the ratio
        _fee(0, S.FeeState(4e17, 6e17, WAD, (1 << 255) + 12345, 0), true, 4e18, 0.06e18);
        _fee(0, S.FeeState(4e17, 6e17, 1, type(uint256).max, 0), true, 4e18, 0.06e18);
        _fee(0, S.FeeState(4e17, 6e17, WAD, type(uint256).max / WAD, 0), true, 4e18, 0.06e18);
        // huge variance saturates the vol clamp; tiny ratio floors to zero
        _fee(0, S.FeeState(4e17, 6e17, WAD, WAD, type(uint256).max), false, 4e18, 0.06e18);
        _fee(0, S.FeeState(4e17, 6e17, 1e30, 1, 1e12), false, 4e18, 0.06e18);
        // large books: 4*vs overflow and wide totals
        _fee(0, S.FeeState(type(uint256).max / 3, 1e18, WAD, WAD, 1e12), true, 4e18, 0.06e18);
        _fee(0, S.FeeState(type(uint256).max / 8, type(uint256).max / 8, WAD, WAD, 1e12), true, 4e18, 0.06e18);
        _fee(0, S.FeeState(1e40, 3e40, WAD, WAD, 1e12), true, 4e18, 0.06e18);
        _fee(0, S.FeeState(13191225306449892018436295903974154245, 17169317935645028434960, LIVE_RP, LIVE_RP, 0), false, 4e18, 0.06e18);
        // pseudo-random sweep over the live row
        for (uint256 i; i < 400; ++i) {
            uint256 r = uint256(keccak256(abi.encode("kyberfee", i)));
            uint256 rs = (r % 1e24) + (i % 3 == 0 ? 0 : 1);
            uint256 rv = (r >> 80) % 1e24;
            uint256 an = i % 2 == 0 ? WAD : LIVE_RP / 1e12;
            uint256 sp = an * (50 + ((r >> 160) % 100)) / 100;
            uint256 rvw = (r >> 200) % 1e15;
            uint256 kb = (r >> 240) % 4;
            uint256 k = kb == 0 ? 0 : kb == 1 ? 4e18 : kb == 2 ? 6e18 : 16e18;
            uint256 b = kb == 0 ? 0 : kb == 1 ? 0.06e18 : kb == 2 ? 0.02e18 : 0.3e18;
            _fee(i % 5, S.FeeState(rs, rv, an, sp, rvw), (r >> 250) % 2 == 0, k, b);
        }
    }
    /// @dev |ln(a/b)| directly: b = WAD makes the ratio exactly `a`, so every bit length of lnWad's input is
    ///      exercised, plus random pairs and the degenerate inputs.
    function lnCells() external {
        for (uint256 i; i < 600; ++i) {
            uint256 r = uint256(keccak256(abi.encode("kyberln", i)));
            uint256 a;
            uint256 b = WAD;
            if (i < 256) {
                a = (r >> (255 - i)) | (uint256(1) << i); // bit length i + 1
            } else if (i < 400) {
                a = WAD + (r % 1e16) - 5e15;
            } else if (i < 550) {
                a = (r >> 128) % 1e40 + 1;
                b = (r % 1e40) + 1;
            } else {
                a = i % 5 == 0 ? 0 : (r >> (i % 256));
                b = i % 7 == 0 ? 0 : (r % 1e30);
            }
            (bool ok, bytes memory ret) = address(h).staticcall(abi.encodeCall(KyberFeeHarness.logRatioAbsWad, (a, b)));
            uint256 v = ok ? abi.decode(ret, (uint256)) : 0;
            _row(string.concat('{"k":"ln",', _kv("a", a), _kv("b", b), _kv("v", v), _res(ok, ret), "}"));
        }
    }

    function volCells() external {
        for (uint256 i; i < 300; ++i) {
            uint256 r = uint256(keccak256(abi.encode("kybervol", i)));
            uint256 pi = i % 5;
            uint256 rvw = i % 11 == 0 ? type(uint256).max >> (i % 64) : (r % 1e16);
            uint256 disloc = (r >> 64) % 5e18;
            (bool ok, bytes memory ret) = address(h).staticcall(abi.encodeCall(KyberFeeHarness.volMultiplier, (_ps[pi], rvw, disloc)));
            uint256 v = ok ? abi.decode(ret, (uint256)) : 0;
            _row(string.concat('{"k":"vol","p":"', vm.toString(pi), '",', _kv("rvw", rvw), _kv("d", disloc), _kv("v", v), _res(ok, ret), "}"));
        }
    }
}
