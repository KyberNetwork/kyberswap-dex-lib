// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import {LevEdgeRows} from "./LevEdgeRows.sol";
import {LevCurveEdgesHarness} from "./LevCurveEdgesHarness.sol";

/// @notice Leverage curve edge fixture. Public entrypoints (ab/lq/dq/ss/fp) run against the DEPLOYED
///         CollRebalancerMath on a Base fork; every public row is re-run on the internal-visibility copy and must
///         agree, which licenses the copy's internal-function rows (mv/sa/hla/ric/cvr/dcap/rs/rdy/dsp/pr/rdl/
///         psa/paa/abe/phi/dn/lerp/b3/b4/m512/pgt).
contract LevCurveEdgesTest is LevEdgeRows {
    address constant LIB = 0xc002d0731e6A2E6e80bE754779BCef6B01aFF0BB;
    uint256 constant BLOCK = 51318000;
    uint256 constant WAD = 1e18;
    uint256 constant PPM = 1e6;
    uint256 constant MAXI = 1e38;
    uint256 constant RATIO = 444_444_444_444_444_444;
    uint256 constant H_ZERO = 562_500_000_000_000_000;
    uint256 constant H_JOIN = 1_010_000_000_000_000_000;
    uint256 constant H_WALL = 1_882_448_291_726_770_582;
    uint256 constant D_JOIN = 509_975_124_224_178_054;
    uint256 constant D_WALL = 1_214_482_768_855_981_020;
    uint256 constant MAXU = type(uint256).max;

    LevCurveEdgesHarness h;
    uint256 publicRows;

    // ------------------------------------------------------------------ call wrappers
    function _pub(string memory op, bytes memory libData, bytes memory harnessData, uint256[] memory ins)
        internal
        returns (bytes memory ret)
    {
        bool ok;
        (ok, ret) = _call(op, LIB, libData, ins);
        (bool hok, bytes memory hret) = address(h).staticcall(harnessData);
        require(ok && hok, "public entrypoint reverted");
        require(keccak256(ret) == keccak256(hret), "internal copy diverges from deployed library");
        ++publicRows;
    }

    function _ab(uint256 coll, uint256 debt, uint256 price, uint256 r) internal returns (uint256 x, uint256 b) {
        bytes memory ret = _pub(
            "ab",
            abi.encodeWithSignature("anchorAndBase(uint256,uint256,uint256,uint256)", coll, debt, price, r),
            abi.encodeCall(h.ab, ([coll, debt, price, r])),
            _a(coll, debt, price, r)
        );
        (x, b) = abi.decode(ret, (uint256, uint256));
    }

    function _lq(uint256 coll, uint256 debt, uint256 price, uint256 spread, uint256 amt) internal {
        _pub(
            "lq",
            abi.encodeWithSignature(
                "leverageQuote(uint256,uint256,uint256,uint256,uint256,uint256)", coll, debt, price, RATIO, spread, amt
            ),
            abi.encodeCall(h.lq, ([coll, debt, price, RATIO, spread, amt])),
            _a(coll, debt, price, RATIO, spread, amt)
        );
    }

    function _dq(uint256 coll, uint256 debt, uint256 price, uint256 spread, uint256 amt) internal {
        _dqr(coll, debt, price, RATIO, spread, amt);
    }

    function _dqr(uint256 coll, uint256 debt, uint256 price, uint256 r, uint256 spread, uint256 amt) internal {
        _pub(
            "dq",
            abi.encodeWithSignature(
                "deleverageQuote(uint256,uint256,uint256,uint256,uint256,uint256)", coll, debt, price, r, spread, amt
            ),
            abi.encodeCall(h.dq, ([coll, debt, price, r, spread, amt])),
            _a(coll, debt, price, r, spread, amt)
        );
    }

    function _ss(uint256 coll, uint256 debt, uint256 price, uint256 req, uint256 r) internal {
        _pub(
            "ss",
            abi.encodeWithSignature("isStateSafe(uint256,uint256,uint256,uint256,uint256)", coll, debt, price, req, r),
            abi.encodeCall(h.ss, ([coll, debt, price, req, r])),
            _a(coll, debt, price, req, r)
        );
    }

    function _int(string memory op, bytes memory data, uint256[] memory ins) internal returns (bool, bytes memory) {
        return _call(op, address(h), data, ins);
    }

    function _rs(uint256 cv, uint256 debt) internal returns (bool ok, uint256 y, uint256 wallCv, uint256 stw) {
        (bool cok, bytes memory ret) = _int("rs", abi.encodeCall(h.rs, (cv, debt, RATIO)), _a(cv, debt, RATIO));
        if (!cok) return (false, 0, 0, 0);
        uint256[7] memory w = abi.decode(ret, (uint256[7]));
        return (w[0] == 1, w[3], w[4], w[6]);
    }

    function _sub(uint256 a, uint256 b) internal pure returns (uint256) {
        return a > b ? a - b : 0;
    }

    // ------------------------------------------------------------------ generator
    function test_levCurveEdges() public {
        vm.createSelectFork("https://mainnet.base.org", BLOCK);
        h = new LevCurveEdgesHarness();
        seed = 0x6d33723163757276;
        _begin("test/kyber/fixtures/lev_curve_edges.json", string.concat('"block":', vm.toString(BLOCK)));

        // frozenParams() on the deployed library: 20 words.
        _call("fp", LIB, abi.encodeWithSignature("frozenParams()"), new uint256[](0));

        _sectionWadStates();
        _sectionPrices();
        _sectionMarkedEdges();
        _sectionRandom(900);
        _sectionPureMath();
        _sectionDust();
        _sectionTies();
        _end();
        emit log_named_uint("rows", rowCount);
        emit log_named_uint("public rows (deployed == copy)", publicRows);
    }

    function _cvList() internal pure returns (uint256[] memory l) {
        l = new uint256[](20);
        l[0] = 0;
        l[1] = 1;
        l[2] = 2;
        l[3] = 3;
        l[4] = 10;
        l[5] = 66;
        l[6] = 67;
        l[7] = 101;
        l[8] = 151;
        l[9] = 999_999;
        l[10] = 1e18;
        l[11] = 123_456_789_012_345_678_901;
        l[12] = 7_683_022_328_489_670_000_000;
        l[13] = 2.2e36;
        l[14] = 3e36;
        l[15] = 1e37 + 7;
        l[16] = MAXI - 1;
        l[17] = MAXI;
        l[18] = MAXI + 1;
        l[19] = 5_555_555_555_555_555_555_555_555;
    }

    function _debtList(uint256 cv) internal pure returns (uint256[] memory l) {
        uint256 dH = cv * D_JOIN / H_JOIN;
        uint256 dW = cv * D_WALL / H_WALL;
        l = new uint256[](18);
        l[0] = 0;
        l[1] = 1;
        l[2] = _sub(dH, 1);
        l[3] = dH;
        l[4] = dH + 1;
        l[5] = _sub(dW, 1);
        l[6] = dW;
        l[7] = dW + 1;
        l[8] = dW + 2;
        l[9] = cv / 2;
        l[10] = cv / 2 + 1;
        l[11] = cv * 2 / 3 + 1;
        l[12] = _sub(cv, 1);
        l[13] = cv;
        l[14] = cv * 58 / 100;
        l[15] = cv * 45 / 100;
        l[16] = cv * 80 / 100;
        l[17] = MAXI + 1;
    }

    /// @dev price = WAD (the hook's frame): full adversarial surface per state.
    function _sectionWadStates() internal {
        uint256[] memory cvs = _cvList();
        for (uint256 i; i < cvs.length; ++i) {
            uint256[] memory ds = _debtList(cvs[i]);
            for (uint256 j; j < ds.length; ++j) {
                _fullState(cvs[i], ds[j]);
            }
        }
    }

    function _fullState(uint256 cv, uint256 debt) internal {
        (uint256 x,) = _ab(cv, debt, WAD, RATIO);
        _ss(cv, debt, WAD, 0, RATIO);
        _ss(cv, debt, WAD, x, RATIO);
        _ss(cv, debt, WAD, x + 1, RATIO);
        _int("sa", abi.encodeCall(h.sa, (cv, debt, RATIO)), _a(cv, debt, RATIO));
        _int("abe", abi.encodeCall(h.abe, (cv, debt, RATIO)), _a(cv, debt, RATIO));
        _int("dsp", abi.encodeCall(h.dsp, (cv, debt, 13_000)), _a(cv, debt, 13_000));
        _int("dsp", abi.encodeCall(h.dsp, (cv, debt, 13_001)), _a(cv, debt, 13_001));
        (bool rok, uint256 y, uint256 wallCv, uint256 stw) = _rs(cv, debt);
        if (cv <= MAXI && 3 * debt <= 2 * cv) {
            uint256 a = 0;
            (bool ok, bytes memory ret) = _int("hla", abi.encodeCall(h.hla, (cv, debt)), _a(cv, debt));
            if (ok) a = abi.decode(ret, (uint256));
            for (uint256 k; k < 4; ++k) {
                uint256 cand = a + k;
                if (cand < 2) continue;
                cand -= 2;
                _int("ric", abi.encodeCall(h.ric, (cv, debt, cand)), _a(cv, debt, cand));
            }
        }
        if (rok) {
            uint256[7] memory ys = [uint256(0), 1, y - 1, y, y + 1, WAD - 2, WAD - 1];
            for (uint256 k; k < ys.length; ++k) {
                _int("rdy", abi.encodeCall(h.rdy, (wallCv, ys[k])), _a(wallCv, ys[k]));
            }
        }
        if (cv > MAXI || debt > MAXI) {
            _lq(cv, debt, WAD, 13_001, 1);
            _dq(cv, debt, WAD, 13_001, debt == 0 ? 1 : debt);
            return;
        }
        _lqSet(cv, debt, WAD, cv);
        _dqSet(cv, debt, WAD, x, stw);
        if (x > 0) {
            _cvrSet(x, debt);
            _dcapSet(x);
            _prSet(x, cv, debt, WAD);
            _postSet(x, cv, debt, WAD);
        }
        if (rok) {
            uint256[4] memory ins = [uint256(1), stw / 2, _sub(stw, 1), stw];
            for (uint256 k; k < ins.length; ++k) {
                if (ins[k] == 0) continue;
                _int(
                    "rdl",
                    abi.encodeCall(h.rdl, ([cv, debt, cv, WAD, RATIO, uint256(13_001), ins[k]])),
                    _a7([cv, debt, cv, WAD, RATIO, uint256(13_001), ins[k]])
                );
            }
        }
    }

    function _lqSet(uint256 coll, uint256 debt, uint256 price, uint256 cv) internal {
        uint256[7] memory ins =
            [uint256(1), coll / 1000 + 1, coll / 7 + 1, coll, MAXI, MAXI + 1, MAXU - coll + (coll == 0 ? 0 : 1)];
        uint256[2] memory spreads = [uint256(0), 13_001];
        for (uint256 s; s < spreads.length; ++s) {
            for (uint256 k; k < ins.length; ++k) {
                _lq(coll, debt, price, spreads[s], ins[k]);
            }
        }
        _lq(coll, debt, price, 999_999, coll / 3 + 1);
        _lq(coll, debt, price, 1_000_000, coll / 3 + 1);
        cv;
    }

    function _dqSet(uint256 coll, uint256 debt, uint256 price, uint256 x, uint256 stw) internal {
        uint256 t3 = x / 3;
        uint256[13] memory ins = [
            uint256(1),
            2,
            debt / 3,
            _sub(debt, t3 + 1),
            _sub(debt, t3),
            _sub(debt, _sub(t3, 1)),
            _sub(debt, (x + 2) / 3),
            _sub(debt, x / 6),
            _sub(debt, 1),
            debt,
            debt + 1,
            _sub(stw, 1),
            stw + 1
        ];
        uint256[4] memory spreads = [uint256(0), 13_000, 13_001, 999_999];
        for (uint256 s; s < spreads.length; ++s) {
            for (uint256 k; k < ins.length; ++k) {
                _dq(coll, debt, price, spreads[s], ins[k]);
            }
        }
        if (stw > 0) _dq(coll, debt, price, 13_001, stw);
        _dq(coll, debt, price, 1_000_000, debt / 2 + 1);
        _dqr(coll, debt, price, RATIO + 1, 13_001, debt / 2 + 1);
    }

    function _cvrSet(uint256 x, uint256 debt) internal {
        uint256 dj = 2 * x * D_JOIN / (3 * WAD);
        uint256 dw = 2 * x * D_WALL / (3 * WAD);
        uint256[10] memory ds = [uint256(0), 1, _sub(dj, 1), dj, dj + 1, dw, dw + 1, x / 3, (x + 2) / 3, debt];
        for (uint256 k; k < ds.length; ++k) {
            // In-domain only: callers keep anchor + debt <= 8cv/3 <= 2.67e38.
            if (x + ds[k] > 266_666_666_666_666_666_666_666_666_666_666_666_667) continue;
            _int("cvr", abi.encodeCall(h.cvr, (x, ds[k])), _a(x, ds[k]));
        }
    }

    function _dcapSet(uint256 x) internal {
        uint256[7] memory hs = [H_ZERO - 1, H_ZERO, H_JOIN, H_JOIN + 1, H_WALL, H_WALL + 1, (H_JOIN + H_WALL) / 2];
        for (uint256 k; k < hs.length; ++k) {
            // least cv with floor(3cv*WAD/(2x)) >= h, and its predecessor
            uint256 c0 = (hs[k] * 2 * x + 3 * WAD - 1) / (3 * WAD);
            if (c0 > MAXI) continue;
            _int("dcap", abi.encodeCall(h.dcap, (x, c0)), _a(x, c0));
            if (c0 > 0) _int("dcap", abi.encodeCall(h.dcap, (x, c0 - 1)), _a(x, c0 - 1));
        }
    }

    function _prSet(uint256 x, uint256 coll, uint256 debt, uint256 price) internal {
        uint256 t3 = x / 3;
        uint256[5] memory ins = [uint256(1), debt / 3, _sub(debt, t3), _sub(debt, t3 + 1), _sub(debt, x / 6)];
        for (uint256 k; k < ins.length; ++k) {
            if (ins[k] == 0 || ins[k] > debt) continue;
            uint256 nd = debt - ins[k];
            (bool ok, bytes memory ret) = address(h).staticcall(abi.encodeCall(h.cvr, (x, nd)));
            if (!ok) continue;
            (bool feasible, uint256 cvReq) = abi.decode(ret, (bool, uint256));
            if (!feasible) continue;
            uint256 collReq = cvReq * WAD / price + (mulmod(cvReq, WAD, price) > 0 ? 1 : 0);
            if (collReq >= coll) continue;
            uint256 og = coll - collReq;
            uint256[3] memory spreads = [uint256(13_001), 250_000, 999_999];
            for (uint256 s; s < spreads.length; ++s) {
                uint256[7] memory a = [x, coll, debt, nd, price, og, spreads[s]];
                _int("pr", abi.encodeCall(h.pr, (a)), _a7(a));
            }
        }
    }

    function _postSet(uint256 x, uint256 coll, uint256 debt, uint256 price) internal {
        uint256[3] memory pres = [x - 1, x, x + 1];
        for (uint256 k; k < pres.length; ++k) {
            for (uint256 s; s < 2; ++s) {
                uint256[6] memory a = [pres[k], coll, debt, price, RATIO, s];
                _int("psa", abi.encodeCall(h.psa, (a)), _a(a[0], a[1], a[2], a[3], a[4], a[5]));
                _int("paa", abi.encodeCall(h.paa, (a)), _a(a[0], a[1], a[2], a[3], a[4], a[5]));
            }
        }
    }

    /// @dev Non-WAD reservation prices, collateral floored and ceiled onto the target marked value.
    function _sectionPrices() internal {
        uint256[10] memory prices = [
            uint256(1),
            3,
            1e6,
            768_302_232_848_967,
            WAD - 1,
            WAD + 1,
            768_302_232_848_967_000_000_000_000_000_000,
            1e36,
            2 ** 200,
            MAXU
        ];
        uint256[4] memory cvs = [uint256(101), 1e18, 1e30 + 3, MAXI];
        for (uint256 p; p < prices.length; ++p) {
            for (uint256 c; c < cvs.length; ++c) {
                uint256 collF = _mulDivFloor(cvs[c], WAD, prices[p]);
                for (uint256 up; up < 2; ++up) {
                    uint256 coll = collF + up;
                    _priceState(coll, cvs[c], prices[p]);
                }
            }
        }
    }

    function _priceState(uint256 coll, uint256 cv, uint256 price) internal {
        uint256 dH = cv * D_JOIN / H_JOIN;
        uint256 dW = cv * D_WALL / H_WALL;
        uint256[6] memory ds = [uint256(0), dH, dH + 1, dW + 1, cv / 2, cv * 9 / 10];
        for (uint256 j; j < ds.length; ++j) {
            uint256 debt = ds[j];
            (uint256 x,) = _ab(coll, debt, price, RATIO);
            _ss(coll, debt, price, x, RATIO);
            _int("mv", abi.encodeCall(h.mv, (coll, price)), _a(coll, price));
            _lq(coll, debt, price, 13_001, 1);
            _lq(coll, debt, price, 13_001, coll / 3 + 1);
            _lq(coll, debt, price, 13_001, coll);
            (,,, uint256 stw) = _rs(cv, debt);
            uint256[4] memory ins = [uint256(1), _sub(debt, x / 3), debt, stw];
            for (uint256 k; k < ins.length; ++k) {
                _dq(coll, debt, price, 0, ins[k]);
                _dq(coll, debt, price, 13_001, ins[k]);
            }
            if (x > 0 && coll <= type(uint128).max) {
                _prSet(x, coll, debt, price);
            }
        }
    }

    function _sectionMarkedEdges() internal {
        uint256[2][12] memory e = [
            [uint256(2 ** 255), 2 * WAD - 2],
            [uint256(2 ** 255), 2 * WAD],
            [MAXU, 1],
            [MAXU, WAD],
            [MAXI, WAD],
            [MAXI + 1, WAD],
            [MAXI * WAD / (WAD + 1), WAD + 1],
            [MAXI * WAD / (WAD + 1) + 1, WAD + 1],
            [uint256(0), WAD],
            [uint256(1), 0],
            [WAD - 1, 1],
            [WAD, 1]
        ];
        for (uint256 i; i < e.length; ++i) {
            _int("mv", abi.encodeCall(h.mv, (e[i][0], e[i][1])), _a(e[i][0], e[i][1]));
            _ab(e[i][0], 1, e[i][1], RATIO);
            _ab(e[i][0], 0, e[i][1], RATIO);
            _ss(e[i][0], 0, e[i][1], 0, RATIO);
            _lq(e[i][0], 1, e[i][1], 0, 1);
            _dq(e[i][0], 1, e[i][1], 0, 1);
        }
    }

    function _mulDivFloor(uint256 a, uint256 b, uint256 d) internal pure returns (uint256) {
        // a*b fits for every call site (a <= 1e38, b = WAD)
        return a * b / d;
    }

    function _randState() internal returns (uint256 cv, uint256 debt) {
        uint256 r = _rand();
        uint256 e = r % 39;
        cv = (_rand() % (10 ** e)) + 1;
        uint256 mode = _rand() % 6;
        uint256 j = _rand() % 7;
        if (mode == 0) debt = cv * (_rand() % 1_100_001) / 1_000_000;
        else if (mode == 1) debt = _sub(cv * D_JOIN / H_JOIN + j, 3);
        else if (mode == 2) debt = _sub(cv * D_WALL / H_WALL + j, 3);
        else if (mode == 3) debt = _sub(cv / 2 + j, 3);
        else if (mode == 4) debt = cv * (500_000 + (_rand() % 150_000)) / 1_000_000;
        else debt = _rand() % (cv + 2);
    }

    function _sectionRandom(uint256 n) internal {
        for (uint256 i; i < n; ++i) {
            (uint256 cv, uint256 debt) = _randState();
            uint256 pm = _rand() % 4;
            uint256 price = pm == 0 ? WAD : pm == 1 ? 768_302_232_848_967 : pm == 2 ? (_rand() % 1e36) + 1 : WAD + 1;
            uint256 coll = _mulDivFloor(cv, WAD, price) + (_rand() % 2);
            (uint256 x,) = _ab(coll, debt, price, RATIO);
            _ss(coll, debt, price, x + (_rand() % 2), RATIO);
            uint256 sp = _rand() % 5 == 0 ? 13_000 + (_rand() % 3) : _rand() % PPM;
            _lq(coll, debt, price, sp, (_rand() % (coll + 1)) + 1);
            _dq(coll, debt, price, sp, debt == 0 ? 1 : (_rand() % debt) + 1);
            _dq(coll, debt, price, _rand() % PPM, debt == 0 ? 1 : _sub(debt, x / 3) + (_rand() % 3));
            if (price == WAD) {
                _int("sa", abi.encodeCall(h.sa, (cv, debt, RATIO)), _a(cv, debt, RATIO));
                _rs(cv, debt);
            }
            if (x > 0 && x <= 2.6e38 && coll <= type(uint128).max) {
                uint256 d2 = _rand() % (x + 1);
                if (x + d2 <= 2.66e38) _int("cvr", abi.encodeCall(h.cvr, (x, d2)), _a(x, d2));
                uint256 c2 = (_rand() % (MAXI)) + 1;
                if (i % 2 == 0) c2 = x * (500_000 + (_rand() % 1_000_000)) / 1_000_000;
                if (c2 <= MAXI) _int("dcap", abi.encodeCall(h.dcap, (x, c2)), _a(x, c2));
                if (i % 3 == 0) _prSet(x, coll, debt, price);
            }
        }
    }

    function _sectionPureMath() internal {
        uint256[12] memory hs = [
            H_ZERO,
            H_ZERO + 1,
            H_JOIN - 1,
            H_JOIN,
            H_JOIN + 1,
            H_WALL - 1,
            H_WALL,
            H_WALL + 1,
            (H_JOIN + H_WALL) / 2,
            H_JOIN + uint256(872_448_291_726_770_582) / 3,
            1_500_000_000_000_000_000,
            2 * WAD
        ];
        for (uint256 i; i < hs.length; ++i) {
            _int("phi", abi.encodeCall(h.phi, (hs[i])), _a(hs[i]));
            _int("dn", abi.encodeCall(h.dn, (hs[i])), _a(hs[i]));
        }
        for (uint256 i; i < 200; ++i) {
            uint256 hh = H_ZERO + (_rand() % (H_WALL - H_ZERO + 1));
            _int("phi", abi.encodeCall(h.phi, (hh)), _a(hh));
            _int("dn", abi.encodeCall(h.dn, (hh)), _a(hh));
            uint256 xx = _rand() % (WAD + 1);
            _int("b3", abi.encodeCall(h.b3, (xx)), _a(xx));
            _int("b4", abi.encodeCall(h.b4, (xx)), _a(xx));
            uint256 la = _rand() % (2 * WAD);
            uint256 lb = i % 5 == 0 ? la : _rand() % (2 * WAD);
            _int("lerp", abi.encodeCall(h.lerp, (la, lb, xx)), _a(la, lb, xx));
            uint256 ma = _rand() >> (_rand() % 256);
            uint256 mb = _rand() >> (_rand() % 256);
            _int("m512", abi.encodeCall(h.m512, (ma, mb)), _a(ma, mb));
            uint256 mc = i % 4 == 0 ? mb : _rand() >> (_rand() % 256);
            uint256 md = i % 4 == 0 ? ma : _rand() >> (_rand() % 256);
            _int("pgt", abi.encodeCall(h.pgt, (ma, mb, mc, md)), _a(ma, mb, mc, md));
        }
        uint256[6] memory xs = [uint256(0), 1, WAD / 2, WAD - 1, WAD, 333_333_333_333_333_333];
        uint256[6] memory ends = [uint256(0), 1, 995_037_190_209_989_135, 645_161_290_322_580_645, WAD, 807_506_474_953_861_782];
        for (uint256 i; i < xs.length; ++i) {
            _int("b3", abi.encodeCall(h.b3, (xs[i])), _a(xs[i]));
            _int("b4", abi.encodeCall(h.b4, (xs[i])), _a(xs[i]));
            for (uint256 a; a < ends.length; ++a) {
                for (uint256 b; b < ends.length; ++b) {
                    _int("lerp", abi.encodeCall(h.lerp, (ends[a], ends[b], xs[i])), _a(ends[a], ends[b], xs[i]));
                }
            }
        }
        uint256[2][8] memory mm = [
            [MAXU, MAXU],
            [MAXU, 1],
            [MAXU, 2],
            [uint256(2 ** 128), 2 ** 128],
            [uint256(2 ** 128 - 1), 2 ** 128 + 1],
            [uint256(0), MAXU],
            [uint256(2 ** 255), 2],
            [MAXU - 1, MAXU - 1]
        ];
        for (uint256 i; i < mm.length; ++i) {
            _int("m512", abi.encodeCall(h.m512, (mm[i][0], mm[i][1])), _a(mm[i][0], mm[i][1]));
            _int("pgt", abi.encodeCall(h.pgt, (mm[i][0], mm[i][1], mm[i][1], mm[i][0])), _a(mm[i][0], mm[i][1], mm[i][1], mm[i][0]));
            _int("pgt", abi.encodeCall(h.pgt, (mm[i][0], mm[i][1], 1, 1)), _a(mm[i][0], mm[i][1], 1, 1));
        }
    }

    /// @dev Dust region exhaustively: every (cv, debt) with cv <= 160 at price WAD through the deployed library,
    ///      and the pro-rata dust guard around anchor 101.
    function _sectionDust() internal {
        for (uint256 cv = 1; cv <= 160; cv += (cv < 40 ? 1 : 3)) {
            for (uint256 debt; debt <= cv; ++debt) {
                (uint256 x,) = _ab(cv, debt, WAD, RATIO);
                if (debt > 0) {
                    _dq(cv, debt, WAD, 13_001, debt == 1 ? 1 : _sub(debt, x / 3) == 0 ? 1 : _sub(debt, x / 3));
                    _dq(cv, debt, WAD, 999_999, 1);
                }
                if (cv % 3 == 0) _lq(cv, debt, WAD, 13_001, cv);
            }
        }
        for (uint256 x = 95; x <= 106; ++x) {
            uint256[7] memory a = [x, uint256(80), 40, 20, WAD, 30, 13_001];
            _int("pr", abi.encodeCall(h.pr, (a)), _a7(a));
            a[3] = (x - 1) / 3; // 3*newDebt just below anchor
            _int("pr", abi.encodeCall(h.pr, (a)), _a7(a));
            a[3] = x / 3 + (x % 3 == 0 ? 0 : 1); // 3*newDebt >= anchor
            _int("pr", abi.encodeCall(h.pr, (a)), _a7(a));
            a[2] = (x - 1) / 3; // 3*debt < anchor
            _int("pr", abi.encodeCall(h.pr, (a)), _a7(a));
        }
    }
    /// @dev Exact product ties of every Mul512 domain decision (the reals-equal case each `>` must reject), and
    ///      recovery states whose partial retirement needs the y+1 upper bracket of _recoveryDeleverage's bisection.
    ///      The bracket cases were located off-chain; every output below is the deployed library's.
    function _sectionTies() internal {
        // debt*H_JOIN == cv*D_JOIN and debt*H_WALL == cv*D_WALL (both reduce by gcd 2)
        uint256[4] memory ks = [uint256(1), 3, 1_000_000_007, 197_000_000_000_000_000_000];
        for (uint256 i; i < ks.length; ++i) {
            _fullState(505_000_000_000_000_000 * ks[i], 254_987_562_112_089_027 * ks[i]);
            _fullState(941_224_145_863_385_291 * ks[i], 607_241_384_427_990_510 * ks[i]);
            // 3*debt*WAD == 2*anchor*D_JOIN and == 2*anchor*D_WALL
            uint256 a1 = 250_000_000_000_000_000 * ks[i];
            uint256 d1 = 84_995_854_037_363_009 * ks[i];
            uint256 a2 = 25_000_000_000_000_000 * ks[i];
            uint256 d2 = 20_241_379_480_933_017 * ks[i];
            for (uint256 j; j < 3; ++j) {
                _int("cvr", abi.encodeCall(h.cvr, (a1, d1 + j - 1)), _a(a1, d1 + j - 1));
                _int("cvr", abi.encodeCall(h.cvr, (a2, d2 + j - 1)), _a(a2, d2 + j - 1));
            }
        }
        // 2*debt == cv on odd/even marks, and the pre-state tie 3*debt == anchor
        for (uint256 cv = 1e18; cv < 1e18 + 4; ++cv) {
            _fullState(cv, cv / 2);
        }
        // cvRequired(anchor, ceil(anchor/3)) == cv exactly: collAtTarget ties collateral inside the crossing split
        uint256[2][12] memory bt = [
            [uint256(684), 342],
            [uint256(806), 403],
            [uint256(302), 151],
            [uint256(280), 140],
            [uint256(1620), 810],
            [uint256(278), 139],
            [uint256(730), 365],
            [uint256(184), 92],
            [uint256(722), 361],
            [uint256(828), 414],
            [uint256(392), 196],
            [uint256(258), 129]
        ];
        for (uint256 i; i < bt.length; ++i) {
            (uint256 cv, uint256 debt) = (bt[i][0], bt[i][1]);
            (uint256 x,) = _ab(cv, debt, WAD, RATIO);
            uint256[4] memory ins = [debt - x / 6, debt - (x + 2) / 3 + 1, debt / 2, debt - x / 3];
            for (uint256 k; k < ins.length; ++k) {
                _dq(cv, debt, WAD, 13_001, ins[k]);
                _dq(cv, debt, WAD, 500_000, ins[k]);
                uint256 nd = debt - ins[k];
                (, bytes memory ret) = address(h).staticcall(abi.encodeCall(h.cvr, (x, nd)));
                (, uint256 cvReq) = abi.decode(ret, (bool, uint256));
                if (cvReq < cv) {
                    uint256[7] memory a = [x, cv, debt, nd, WAD, cv - cvReq, uint256(500_000)];
                    _int("pr", abi.encodeCall(h.pr, (a)), _a7(a));
                }
            }
        }
        // collateralIn that leaves the marked value unchanged (newCv == cv) on states a non-strict gate would fill
        uint256[3][12] memory ct = [
            [uint256(3867187128965454365), 756116906189934747, 376869752367556455],
            [uint256(2581594438417041633), 400549598801933203, 283907799313621597],
            [uint256(1884613367032912779), 292144566013750709, 299205448386316191],
            [uint256(3578510612404199222), 1453351956783375409, 738599916261293348],
            [uint256(2601284299320062622), 287031139543461608, 208441567785110768],
            [uint256(1650076850273810143), 441742102279424780, 453612783689966557],
            [uint256(3371193163207077120), 1266261969034863557, 715994990857988178],
            [uint256(3471368011986547380), 1607716759754207125, 825783173900406418],
            [uint256(3789011453796380164), 1261541534432918348, 619864885141248210],
            [uint256(2302455814512053789), 677638326243959766, 537364130136481213],
            [uint256(3750912412334117602), 1647602066850171258, 785510736625826802],
            [uint256(3608067945712115894), 886896005985542126, 479404919900228967]
        ];
        for (uint256 i; i < ct.length; ++i) {
            for (uint256 k = 1; k <= 3; ++k) {
                _lq(ct[i][0], ct[i][1], ct[i][2], 0, k);
                _lq(ct[i][0], ct[i][1], ct[i][2], 1, k);
            }
        }
        uint256[3][40] memory rc = [
            [uint256(376226315931390630), 300454712129124488, 3],
            [uint256(567557514173029290), 485494940756265157, 3],
            [uint256(891703611769165116), 812737906725334929, 3],
            [uint256(283567842895664391), 228824503688970589, 3],
            [uint256(228608478158514915), 200826147024866915, 3],
            [uint256(290227850139552560), 265253453407193924, 3],
            [uint256(623301453044333905), 556821989966984385, 3],
            [uint256(60977155694940543), 50301153321558964, 2],
            [uint256(70912057825208276), 56964436083625887, 2],
            [uint256(254214121505153952), 236231268764000867, 3],
            [uint256(79377419791043915), 74921568331073664, 3],
            [uint256(519309210687105329), 480962899260758784, 3],
            [uint256(92752961187274069), 82646134771502750, 3],
            [uint256(579893350421101869), 491784934651470074, 3],
            [uint256(649182407779508631), 538395534797488806, 3],
            [uint256(88217995530076252), 69977248812317916, 2],
            [uint256(451954182941706625), 348497350924520563, 2],
            [uint256(256342411335842577), 198847884582149117, 2],
            [uint256(3757504118075512), 3062985844411026, 3],
            [uint256(550689538531142374), 455589310053583805, 3],
            [uint256(487470094262886523), 461749709679293841, 3],
            [uint256(8324312684201590), 6652524319208016, 3],
            [uint256(90645118183355508), 80144878338113789, 3],
            [uint256(991261940169897), 788195984154453, 2],
            [uint256(706102438172141433), 567952083938885617, 2],
            [uint256(92301756768657799), 84389004066151072, 2],
            [uint256(120571964968124247), 91946854193076903, 2],
            [uint256(405194043211987597), 321811187029608731, 2],
            [uint256(358823315271497396), 306801828670066246, 2],
            [uint256(73132131111662339), 57630678940578833, 2],
            [uint256(899468764478895985), 831473423228113845, 3],
            [uint256(182986827567220260), 150367412698260011, 3],
            [uint256(465153845617243227), 375145181028769812, 2],
            [uint256(8809212024464728), 7166602304322913, 2],
            [uint256(841782188190016058), 731428752229245903, 3],
            [uint256(523718450870712865), 433033982510194578, 2],
            [uint256(204292677556082434), 171612386512791040, 2],
            [uint256(527358331067439705), 450480033564428344, 2],
            [uint256(244115299882355676), 224361001585275690, 3],
            [uint256(83001069175494312), 65708543422402651, 2]
        ];
        for (uint256 i; i < rc.length; ++i) {
            (uint256 cv, uint256 debt, uint256 sIn) = (rc[i][0], rc[i][1], rc[i][2]);
            _rs(cv, debt);
            for (uint256 k = sIn - 1; k <= sIn + 1; ++k) {
                if (k == 0) continue;
                _dq(cv, debt, WAD, 13_001, k);
                _dq(cv, debt, WAD, 0, k);
                _int(
                    "rdl",
                    abi.encodeCall(h.rdl, ([cv, debt, cv, WAD, RATIO, uint256(13_001), k])),
                    _a7([cv, debt, cv, WAD, RATIO, uint256(13_001), k])
                );
            }
        }
    }
}
