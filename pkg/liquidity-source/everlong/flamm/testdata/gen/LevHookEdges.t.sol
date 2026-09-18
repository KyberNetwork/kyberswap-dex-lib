// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import {Math} from "@openzeppelin/contracts/utils/math/Math.sol";
import {LevEdgeRows} from "./LevEdgeRows.sol";

struct LevHookEdgesPoolCtx {
    uint256 physicalPoolAsset;
    uint256 postedPoolAsset;
    uint256 liquidLoanAsset;
    uint256 suppliedLoanAsset;
    uint256 debtLoanAsset;
    uint256 shareSupply;
    uint256 priceWad;
    uint48 priceTs;
    uint8 loanCount;
}

struct LevHookEdgesLeverCtx {
    LevHookEdgesPoolCtx pool;
    bool up;
    uint256 spreadPpm;
    uint256 amountIn;
    uint256 maxOut;
}

interface ILevEdgesCurve {
    function anchorAndBase(uint256, uint256, uint256, uint256) external pure returns (uint256, uint256);
}

/// @notice EverlongLeverageHook._assertAnchorAndBand transcribed line for line (it is internal), with
///         anchorAndBase delegated to the DEPLOYED library by a plain call.
contract LevHookEdgesBandCopy {
    address constant LIB = 0xc002d0731e6A2E6e80bE754779BCef6B01aFF0BB;
    uint256 internal constant WAD = 1e18;
    uint256 internal constant RATIO = 444_444_444_444_444_444;
    uint256 internal constant PRICE_BAND_NUM = 2;

    error FillValueDrop();
    error FillPriceBand();

    function assertAnchorAndBand(uint256 xAnchorBefore, uint256 cvAfter, uint256 dAfter) external view {
        (uint256 xAnchorAfter, uint256 baseXAfter) = ILevEdgesCurve(LIB).anchorAndBase(cvAfter, dAfter, WAD, RATIO);
        if (xAnchorAfter < xAnchorBefore || baseXAfter == 0) revert FillValueDrop();
        uint256 internalValueAfter = Math.mulDiv(baseXAfter, WAD, cvAfter);
        if (internalValueAfter > WAD * PRICE_BAND_NUM || internalValueAfter < WAD / PRICE_BAND_NUM) revert FillPriceBand();
    }
}

/// @notice Leverage hook edge fixture: the DEPLOYED EverlongLeverageHook on a Base fork, with the swap
///         hook's bookFor(ctx) and reservationPriceWad() mocked to arbitrary books so frame()/previewLever() run the
///         shipped bytecode (and the deployed CollRebalancerMath it links) over edge and random frames.
///         Row inputs: [kappa, rs, is_, rv, iv, x, reservationPriceWad, physical, posted, liquid, supplied, debt,
///         shareSupply, priceWad, priceTs, loanCount, up, spreadPpm, amountIn, maxOut].
contract LevHookEdgesTest is LevEdgeRows {
    address constant LEV = 0xE0A98d8e60035832B8BaD7f7af7B9B0b3A7308F3;
    address constant SWAP_HOOK = 0x65CBD227cBC61248ae77a5fC813A29C54C092134;
    address constant POOL = 0xc0fdCB1799cCc2CEBaA1fe247157b0dF33D57572;
    uint256 constant BLOCK = 51318000;
    uint256 constant WAD = 1e18;
    uint256 constant PPM = 1e6;
    uint256 constant MAXU = type(uint256).max;
    uint256 constant RES = 768_302_232_848_967_000_000_000_000_000_000;
    uint256 constant FEED = 768_302_232_848_967;

    bytes4 constant BOOK_FOR = bytes4(keccak256("bookFor((uint256,uint256,uint256,uint256,uint256,uint256,uint256,uint48,uint8))"));
    bytes4 constant RES_SEL = bytes4(keccak256("reservationPriceWad()"));
    bytes4 constant FRAME = bytes4(keccak256("frame((uint256,uint256,uint256,uint256,uint256,uint256,uint256,uint48,uint8))"));
    bytes4 constant PREVIEW = bytes4(
        keccak256("previewLever(((uint256,uint256,uint256,uint256,uint256,uint256,uint256,uint48,uint8),bool,uint256,uint256,uint256))")
    );
    bytes4 constant EXECUTE = bytes4(
        keccak256("executeLever(((uint256,uint256,uint256,uint256,uint256,uint256,uint256,uint48,uint8),bool,uint256,uint256,uint256))")
    );

    LevHookEdgesBandCopy band;
    uint256 fills;
    uint256 realRes;

    struct Book {
        uint256 kappa;
        uint256 rs;
        uint256 is_;
        uint256 rv;
        uint256 iv;
        uint256 x;
    }

    function test_levHookEdges() public {
        vm.createSelectFork("https://mainnet.base.org", BLOCK);
        band = new LevHookEdgesBandCopy();
        seed = 0x6d3372316c6576;
        (, bytes memory rr) = SWAP_HOOK.staticcall(abi.encodeWithSelector(RES_SEL));
        realRes = abi.decode(rr, (uint256));
        _begin("test/kyber/fixtures/lev_hook_edges.json", string.concat('"block":', vm.toString(BLOCK)));
        _mockSanity();
        _sectionEdges();
        _sectionRealistic();
        _sectionFeedMismatch();
        _sectionTiny();
        _sectionRandom(2000);
        _sectionBand();
        _sectionTieFills();
        _end();
        emit log_named_uint("rows", rowCount);
        emit log_named_uint("fills", fills);
    }

    // ------------------------------------------------------------------ plumbing
    function _pc(uint256 liquid, uint256 supplied, uint256 debt, uint256 priceWad)
        internal
        returns (LevHookEdgesPoolCtx memory p)
    {
        p.physicalPoolAsset = _rand() % 1e12;
        p.postedPoolAsset = _rand() % 1e12;
        p.liquidLoanAsset = liquid;
        p.suppliedLoanAsset = supplied;
        p.debtLoanAsset = debt;
        p.shareSupply = _rand() % 1e24;
        p.priceWad = priceWad;
        p.priceTs = uint48(_rand());
        p.loanCount = uint8(_rand() % 3);
    }

    function _ins(Book memory b, uint256 res, LevHookEdgesLeverCtx memory c) internal pure returns (uint256[] memory r) {
        r = new uint256[](20);
        (r[0], r[1], r[2], r[3], r[4], r[5], r[6]) = (b.kappa, b.rs, b.is_, b.rv, b.iv, b.x, res);
        LevHookEdgesPoolCtx memory p = c.pool;
        (r[7], r[8], r[9], r[10], r[11], r[12], r[13]) = (
            p.physicalPoolAsset, p.postedPoolAsset, p.liquidLoanAsset, p.suppliedLoanAsset, p.debtLoanAsset,
            p.shareSupply, p.priceWad
        );
        (r[14], r[15], r[16], r[17], r[18], r[19]) =
            (p.priceTs, p.loanCount, c.up ? 1 : 0, c.spreadPpm, c.amountIn, c.maxOut);
    }

    /// @dev res == type(uint256).max leaves reservationPriceWad() unmocked (the live storage value is recorded).
    function _run(Book memory b, uint256 res, LevHookEdgesLeverCtx memory c) internal {
        vm.clearMockedCalls();
        vm.mockCall(SWAP_HOOK, abi.encodeWithSelector(BOOK_FOR), abi.encode(b));
        if (res == MAXU) res = realRes;
        else vm.mockCall(SWAP_HOOK, abi.encodeWithSelector(RES_SEL), abi.encode(res));
        uint256[] memory ins = _ins(b, res, c);
        _call("hf", LEV, abi.encodeWithSelector(FRAME, c.pool), ins);
        (bool ok, bytes memory ret) = _call("hq", LEV, abi.encodeWithSelector(PREVIEW, c), ins);
        if (ok) ++fills;
        if (_rand() % 16 == 0) {
            vm.prank(POOL);
            (bool eok, bytes memory eret) = LEV.call(abi.encodeWithSelector(EXECUTE, c));
            require(eok == ok && keccak256(eret) == keccak256(ret), "executeLever != previewLever");
        }
    }

    function _lc(LevHookEdgesPoolCtx memory p, bool up, uint256 spread, uint256 amt)
        internal
        returns (LevHookEdgesLeverCtx memory c)
    {
        c.pool = p;
        c.up = up;
        c.spreadPpm = spread;
        c.amountIn = amt;
        c.maxOut = _rand() % 1e30;
    }

    function _mockSanity() internal {
        Book memory b = Book(7, 3e20, 2e20, 6e7, 4e7, 9);
        vm.mockCall(SWAP_HOOK, abi.encodeWithSelector(BOOK_FOR), abi.encode(b));
        vm.mockCall(SWAP_HOOK, abi.encodeWithSelector(RES_SEL), abi.encode(RES));
        (bool ok, bytes memory ret) = LEV.staticcall(abi.encodeWithSelector(FRAME, _pc(0, 0, 5e20, FEED)));
        require(ok, "frame reverted");
        (uint256 cv, int256 d, uint256 v, uint256 s,) = abi.decode(ret, (uint256, int256, uint256, uint256, uint256));
        require(v == 1e8 && s == 5e20 && d == 1e21 && cv == 5e20 + Math.mulDiv(1e8, RES, WAD), "mock not applied");
        vm.clearMockedCalls();
    }

    function _one(Book memory b, uint256 res, uint256 liquid, uint256 supplied, uint256 debt, uint256 price, bool up, uint256 spread, uint256 amt)
        internal
    {
        _run(b, res, _lc(_pc(liquid, supplied, debt, price), up, spread, amt));
    }

    function _both(Book memory b, uint256 res, uint256 liquid, uint256 supplied, uint256 debt, uint256 price, uint256 spread, uint256 amt)
        internal
    {
        _one(b, res, liquid, supplied, debt, price, true, spread, amt);
        _one(b, res, liquid, supplied, debt, price, false, spread, amt);
    }

    // ------------------------------------------------------------------ sections
    function _sectionEdges() internal {
        uint256 h = 2 ** 255;
        Book memory n = Book(0, 3e20, 2e20, 6e7, 4e7, 0); // v = 1e8 sats, s = 5e20
        // frame: checked adds / mulDiv / signed arithmetic
        _both(Book(0, 1, 1, MAXU, 1, 0), RES, 0, 0, 1e21, FEED, 17_500, 1); // rv + iv
        _both(Book(0, MAXU, 1, 1, 1, 0), RES, 0, 0, 1e21, FEED, 17_500, 1); // rs + is_
        _both(Book(0, 0, 0, 2 ** 200, 0, 0), 2 ** 100, 0, 0, 1e21, FEED, 17_500, 1); // v * res / WAD overflow
        _both(Book(0, MAXU - 5, 0, 1e8, 0, 0), RES, 0, 0, 1e21, FEED, 17_500, 1); // s + vValue
        _both(n, RES, MAXU, 1, 1e21, FEED, 17_500, 1); // supplied + liquid
        _both(Book(0, h - 1, 0, 1e8, 0, 0), 0, 0, 0, 1, FEED, 17_500, 1); // int256(s) + debt overflow
        _both(Book(0, h, 0, 1e8, 0, 0), 0, 0, 0, h, FEED, 17_500, 1); // negative + negative overflow
        _both(Book(0, 0, 0, 1e8, 0, 0), RES, 0, h, 0, FEED, 17_500, 1); // 0 - int256.min overflow
        _both(Book(0, 0, 0, 1e8, 0, 0), RES, 1, h, 0, FEED, 17_500, 1); // 0 - (int256.min + 1)
        _both(Book(0, h - 1, 0, 1e8, 0, 0), 0, 0, h, 0, FEED, 17_500, 1); // max - min overflow
        _both(Book(0, h, 0, 1e8, 0, 0), 0, 0, MAXU - 1, h - 1, 1, 17_500, 1); // D = 1 on a negative s
        _both(Book(0, h + 5, 0, 1e8, 0, 0), 1e12, 0, 0, h - 1, 1, 17_500, 1e8); // D = 4, cv past MAX_INPUT
        // FrameUnquotable
        _both(n, RES, 0, 5e20, 0, FEED, 17_500, 1); // D = 0
        _both(n, RES, 1, 5e20, 0, FEED, 17_500, 1); // D = -1
        _both(n, RES, 0, 0, 1, FEED, 17_500, 1); // D = s + 1 > 0
        _both(Book(0, 5e20, 0, 0, 0, 0), RES, 0, 0, 1e21, FEED, 17_500, 1); // v = 0
        _both(Book(0, 0, 0, 1e8, 0, 0), 0, 0, 0, 1e21, FEED, 17_500, 1); // cv = 0 (res = 0)
        _both(Book(0, 0, 0, 1, 0, 0), WAD - 1, 0, 0, 1e21, FEED, 17_500, 1); // cv = floor((WAD-1)/WAD) = 0
        // gavAtFeed
        _both(n, RES, 0, 0, 1e21, MAXU / 1e8 + 1, 17_500, 1e6); // v * priceWad overflow
        _both(n, RES, 0, 0, 1e21, MAXU / 1e8, 17_500, 1e6); // v * priceWad fits, s + it overflows
        _both(n, RES, 0, 0, 1e21, (MAXU - 5e20) / 1e8, 17_500, 1e6); // exactly fits
        // up: mulDiv reverts, spread arithmetic
        _one(n, RES, 0, 0, 1e21, FEED, true, 17_500, MAXU); // dCv = cv*in/v overflow
        _one(n, RES, 0, 0, 1e21, FEED, true, 0, 0); // zero amount
        _one(n, RES, 0, 0, 1e21, FEED, true, PPM, 1e6); // spread == PPM
        _one(n, RES, 0, 0, 1e21, FEED, true, PPM + 1, 1e6); // PPM - spread underflow
        _one(n, RES, 0, 0, 1e21, FEED, true, MAXU, 1e6);
        _one(n, RES, 0, 0, 1e21, FEED, true, PPM - 1, 1e6);
        _one(
            Book(0, 0x8cb1e29c658cda1495e60af593bd04cf0fd630f1f29d0da9953f48f1a09f76b5, 0, h + 1, 0, 0),
            0,
            0,
            0,
            h - 1,
            0,
            true,
            17_500,
            0xe8e6b3cc5085e6fdf874a399368d7093977c7ee9b99774d133eaa01a7985f9d6
        ); // dS = mulDiv(..., Up) += 1 overflow
        _one(Book(0, 1, 0, 3, 0, 0), WAD, 0, 0, 2 ** 254, 1, true, 17_500, MAXU / 2); // dS rounding, cv tiny
        // down: thresholds on in18 vs D
        uint256 d0 = 1e21 - 5e20 + 5e20; // D = s + debt = 1e21 for debt = 5e20
        _one(n, RES, 0, 0, 5e20, FEED, false, 17_500, d0);
        _one(n, RES, 0, 0, 5e20, FEED, false, 17_500, d0 - 1);
        _one(n, RES, 0, 0, 5e20, FEED, false, 17_500, d0 + 1);
        _one(n, RES, 0, 0, 5e20, FEED, false, 17_500, 0);
        _one(n, RES, 0, 0, 5e20, FEED, false, PPM, 1e18);
        _one(n, RES, 0, 0, 5e20, FEED, false, PPM + 1, 1e18);
        // crAfter / gavAfter overflow with a huge feed
        _both(n, RES, 0, 0, 1e21, 1e62, 17_500, 1e6);
        _one(n, RES, 0, 0, 5e20, 1e62, false, 17_500, d0 - 1);
        _one(n, RES, 0, 0, 5e20, 1e56, false, 17_500, d0 - 1e3);
        _one(n, RES, 0, 0, 5e20, 1e68, true, 17_500, 1e7);
        // volOut == 0 / in18 <= dSBurn
        _both(Book(0, 5e37, 0, 1, 0, 0), RES, 0, 0, 1e37, FEED, 17_500, 1e30);
        _both(Book(0, 9e20, 0, 1e3, 0, 0), RES, 0, 0, 1e20, FEED, 13_001, 1e18);
    }

    function _bookAt(uint256 vSats, uint256 sFrac, uint256 res) internal pure returns (Book memory b, uint256 cv) {
        uint256 vValue = Math.mulDiv(vSats, res, WAD);
        uint256 s = sFrac == 0 ? 0 : vValue * sFrac / (PPM - sFrac);
        uint256 rv = vSats * 6 / 10;
        uint256 rs = s * 3 / 10;
        b = Book(1, rs, s - rs, rv, vSats - rv, 2);
        cv = s + vValue;
    }

    function _legs(uint256 s, uint256 d) internal pure returns (uint256 liquid, uint256 supplied, uint256 debt) {
        // D = s + debt - (supplied + liquid)
        if (d >= s) return (7, s / 10, d - s + s / 10 + 7);
        return (s - d - (s - d) / 2 + 5, (s - d) / 2, 5);
    }

    function _sectionRealistic() internal {
        uint256[3] memory scales = [uint256(234_423), 1e8, 1e11];
        uint256[4] memory sFracs = [uint256(0), 300_000, 500_000, 900_000];
        uint256[11] memory crs =
            [uint256(1_200_000), 1_500_000, 1_549_000, 1_560_000, 1_800_000, 1_990_000, 2_000_000, 2_010_000, 2_200_000, 2_500_000, 4_000_000];
        uint256[5] memory spreads = [uint256(0), 5_000, 13_000, 17_500, 100_000];
        for (uint256 a; a < scales.length; ++a) {
            for (uint256 b; b < sFracs.length; ++b) {
                for (uint256 c; c < crs.length; ++c) {
                    for (uint256 s; s < spreads.length; ++s) {
                        _realisticState(scales[a], sFracs[b], crs[c], spreads[s], (a + b + c + s) % 7 == 0 ? MAXU : RES, FEED);
                    }
                }
            }
        }
    }

    function _realisticState(uint256 vSats, uint256 sFrac, uint256 cr, uint256 spread, uint256 res, uint256 feed) internal {
        uint256 resUsed = res == MAXU ? realRes : res;
        (Book memory bk, uint256 cv) = _bookAt(vSats, sFrac, resUsed);
        uint256 d = cv * PPM / cr;
        uint256 s = bk.rs + bk.is_;
        (uint256 liquid, uint256 supplied, uint256 debt) = _legs(s, d);
        uint256[6] memory ups = [uint256(1), vSats / 1000 + 1, vSats / 100, vSats / 10, vSats / 2, vSats];
        for (uint256 k; k < ups.length; ++k) {
            _one(bk, res, liquid, supplied, debt, feed, true, spread, ups[k]);
        }
        uint256[7] memory downs = [uint256(1), d / 1000 + 1, d / 100, d / 10, d / 2, d - 1, d];
        for (uint256 k; k < downs.length; ++k) {
            _one(bk, res, liquid, supplied, debt, feed, false, spread, downs[k]);
        }
    }

    function _sectionFeedMismatch() internal {
        uint256[3] memory scales = [uint256(234_423), 1e8, 1e11];
        uint256[11] memory crs =
            [uint256(1_200_000), 1_500_000, 1_549_000, 1_560_000, 1_800_000, 1_990_000, 2_000_000, 2_010_000, 2_200_000, 2_500_000, 4_000_000];
        uint256[3] memory feeds = [FEED / 2, FEED * 2, 1e40];
        for (uint256 a; a < scales.length; ++a) {
            for (uint256 c; c < crs.length; ++c) {
                for (uint256 f; f < feeds.length; ++f) {
                    _realisticState(scales[a], 500_000, crs[c], 17_500, RES, feeds[f]);
                }
            }
        }
    }

    /// @dev Dust frames at res = WAD (cv = s + v): rounding corners of every hook floor/ceil.
    function _sectionTiny() internal {
        for (uint256 v = 1; v <= 24; ++v) {
            for (uint256 sm; sm < 2; ++sm) {
                uint256 s = sm == 0 ? 0 : v / 2 + 1;
                uint256 cv = s + v;
                for (uint256 d = 1; d <= cv + 1; ++d) {
                    Book memory bk = Book(0, s, 0, v, 0, 0);
                    (uint256 liquid, uint256 supplied, uint256 debt) = d >= s ? (uint256(0), uint256(0), d - s) : (s - d, uint256(0), uint256(0));
                    uint256 spread = (v + d) % 3 == 0 ? 0 : 13_001;
                    _one(bk, WAD, liquid, supplied, debt, 1, true, spread, 1);
                    _one(bk, WAD, liquid, supplied, debt, 1, true, spread, v);
                    _one(bk, WAD, liquid, supplied, debt, 1, false, spread, 1);
                    if (d > 1) _one(bk, WAD, liquid, supplied, debt, 1, false, spread, d - 1);
                    if (d > 2) _one(bk, WAD, liquid, supplied, debt, 1, false, spread, d / 2);
                }
            }
        }
    }

    function _mag() internal returns (uint256) {
        uint256 e = _rand() % 40;
        return _rand() % (10 ** e) + 1;
    }

    function _sectionRandom(uint256 n) internal {
        for (uint256 i; i < n; ++i) {
            uint256 mode = _rand() % 4;
            uint256 vSats = mode == 3 ? _rand() : _mag() % 1e30 + 1;
            uint256 res = mode == 0 ? RES : mode == 1 ? WAD * (_rand() % 1e6 + 1) : _mag();
            uint256 sFrac = _rand() % PPM;
            uint256 vValue;
            {
                (bool ok, uint256 hi) = _mulFits(vSats, res);
                vValue = ok ? Math.mulDiv(vSats, res, WAD) : hi;
            }
            uint256 s = vValue < 2 ** 200 ? vValue * sFrac / PPM : _rand() % 2 ** 200;
            uint256 cv = s + (vValue < 2 ** 250 ? vValue : 0);
            uint256 crMode = _rand() % 3;
            uint256 d = crMode == 0
                ? (cv > 2 ** 200 ? cv / 2 : cv * PPM / (1_000_000 + _rand() % 4_000_000))
                : crMode == 1 ? cv * PPM / (1_540_000 + _rand() % 20_000) + (cv > 2 ** 200 ? 0 : 1) : _mag();
            if (d == 0) d = 1;
            Book memory bk = Book(_rand(), s / 3, s - s / 3, vSats / 2, vSats - vSats / 2, _rand());
            (uint256 liquid, uint256 supplied, uint256 debt) = _legs(s, d);
            if (_rand() % 10 == 0) {
                liquid = _rand() % 2 == 0 ? 0 : _rand();
                debt = _rand();
                supplied = _rand() >> 1;
            }
            uint256 sp = _rand() % 8 == 0 ? (_rand() % 3 == 0 ? PPM + _rand() % 5 : 13_000 + _rand() % 3) : _rand() % 200_000;
            uint256 feed = _rand() % 5 == 0 ? _mag() : (res / WAD == 0 ? 1 : res / WAD);
            bool up = _rand() % 2 == 0;
            uint256 amt = up ? (_rand() % 4 == 0 ? _mag() : _rand() % (vSats + 1)) : (_rand() % 4 == 0 ? _mag() : _rand() % (d + 2));
            _one(bk, res, liquid, supplied, debt, feed, up, sp, amt);
        }
    }

    function _mulFits(uint256 a, uint256 b) internal pure returns (bool, uint256) {
        unchecked {
            if (a == 0) return (true, 0);
            uint256 c = a * b;
            if (c / a != b) return (false, 2 ** 251);
            return (true, c);
        }
    }

    /// @dev _assertAnchorAndBand transcription over thresholds of the anchor and the band.
    function _sectionBand() internal {
        for (uint256 i; i < 400; ++i) {
            uint256 cv = i < 200 ? (i % 40) + 1 : _mag();
            uint256 d = i < 200 ? (i / 40) * cv / 4 + i % 3 : cv * (_rand() % 1_200_000) / PPM + _rand() % 3;
            (uint256 x,) = ILevEdgesCurve(0xc002d0731e6A2E6e80bE754779BCef6B01aFF0BB).anchorAndBase(cv, d, WAD, 444_444_444_444_444_444);
            uint256[5] memory pres = [uint256(0), x == 0 ? 0 : x - 1, x, x + 1, 2 ** 255];
            for (uint256 k; k < pres.length; ++k) {
                _call("band", address(band), abi.encodeCall(band.assertAnchorAndBand, (pres[k], cv, d)), _a(pres[k], cv, d));
            }
        }
    }
    /// @dev Down fills where the burnt virtual leg equals in18 exactly (volOut > 0): the `in18 <= dSBurn` tie. The
    ///      frames were located off-chain; outputs are the deployed hook's. [v, s, D, in18, spread] at res = WAD.
    function _sectionTieFills() internal {
        uint256[5][12] memory t = [
            [uint256(86), 14136, 12610, 610, 11591],
            [uint256(49), 13024, 6818, 336, 8261],
            [uint256(30), 12343, 11679, 6002, 6028],
            [uint256(236), 28122, 14747, 624, 3496],
            [uint256(3), 33197, 20350, 15743, 15462],
            [uint256(47), 34384, 18157, 3188, 3416],
            [uint256(192), 29299, 15509, 535, 12982],
            [uint256(162), 30392, 27215, 677, 15660],
            [uint256(14), 18424, 17111, 1731, 7201],
            [uint256(126), 22269, 20660, 3636, 4743],
            [uint256(31), 10406, 9889, 880, 667],
            [uint256(135), 32855, 31127, 2422, 891]
        ];
        for (uint256 i; i < t.length; ++i) {
            (uint256 v, uint256 s, uint256 d) = (t[i][0], t[i][1], t[i][2]);
            Book memory bk = Book(0, s, 0, v, 0, 0);
            (uint256 liquid, uint256 debt) = d >= s ? (uint256(0), d - s) : (s - d, uint256(0));
            for (uint256 k = t[i][3] - 1; k <= t[i][3] + 1; ++k) {
                _one(bk, WAD, liquid, 0, debt, 1, false, t[i][4], k);
            }
        }
    }
}
