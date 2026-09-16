// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import "./MMFixtureBase.sol";

/// @notice Financing-side fixture generator on a Base fork: the deployed MMRouter / MorphoBlueAccount / Morpho Blue /
///         AdaptiveCurveIrm views at pinned blocks and warped timestamps, real settlement sequences through the live
///         FLAMM pool, the 0x46c3cd72 sell replayed in its own block, a synthetic multi-venue router running the
///         deployed router bytecode, and an IRM grid over rateAtTarget / utilisation / elapsed.
contract MMFinancingFixture is MMFixtureBase {
    uint256 constant BLOCK = 51317000;
    uint256 constant PRE_SELL_BLOCK = 51302915;
    bytes32 constant REAL_SELL_TX = 0x46c3cd72a5860b2fe546e5a2130e066314e3777027151661e1e4f19a935901fa;
    string constant OUT = "kyber-out/";
    address constant TAKER = address(0x7A4E5);

    function _open(string memory name) internal returns (string memory path) {
        path = string.concat(OUT, name);
        vm.writeFile(path, "");
    }

    function _line(string memory path, bool first, string memory s) internal {
        vm.writeLine(path, string.concat(first ? "" : ",", s));
    }

    // ------------------------------------------------------------------ live views across time
    function test_liveViews() public {
        vm.pauseGasMetering();
        vm.createSelectFork(RPC, BLOCK);
        uint256[] memory px = _pricesLive();
        string memory path = _open("mm_live_views.json");
        vm.writeLine(path, '{"snaps":[');
        _line(path, true, _named("now", _liveSnap(px)));
        uint256 t0 = block.timestamp;
        uint256[5] memory dts = [uint256(1), 3600, 86400, 30 days, 365 days];
        for (uint256 i; i < dts.length; ++i) {
            vm.warp(t0 + dts[i]);
            _line(path, false, _named(string.concat("warp+", vm.toString(dts[i])), _liveSnap(px)));
        }
        vm.warp(t0);
        // recognized < actual: managed figures below the Morpho position
        uint256 snap = vm.snapshotState();
        (uint256 ss,, uint128 col) = KMorpho(MORPHO).position(MID, ACCOUNT);
        _setManaged(ROUTER, POOL, 0, col / 3, ss / 2);
        _line(path, false, _named("managedLow", _liveSnap(px)));
        vm.revertToState(snap);
        // a binding rate ceiling just above the live rate: large slices fail the ceiling, small ones pass
        (, uint256 r0) = KAccount(ACCOUNT).borrowRateAfter(MID, 0, 0);
        _setMaxRate(ROUTER, POOL, 0, uint64(r0 + 3));
        _line(path, false, _named("rateCapTight", _liveSnap(px)));
        (, uint256 rBig) = KAccount(ACCOUNT).borrowRateAfter(MID, 5e13, 0);
        _setMaxRate(ROUTER, POOL, 0, uint64((r0 + rBig) / 2));
        _line(path, false, _named("rateCapMid", _liveSnap(px)));
        vm.revertToState(snap);
        // unreadable IRM: inside the grace (debt marked up at GRACE_RATE) and beyond it (quarantine haircut)
        MParams memory p = KMorpho(MORPHO).idToMarketParams(MID);
        vm.mockCallRevert(IRM, abi.encodeWithSelector(KIrm.borrowRateView.selector, p), "");
        vm.mockCallRevert(IRM, abi.encodeWithSelector(KIrm.borrowRate.selector, p), "");
        vm.warp(t0 + 1800);
        _line(path, false, _named("irmDownGrace", _liveSnap(px)));
        vm.warp(t0 + 7200);
        _line(path, false, _named("irmDownQuarantine", _liveSnap(px)));
        vm.clearMockedCalls();
        vm.revertToState(snap);
        vm.writeLine(path, "]}");
    }

    function _named(string memory name, string memory body) internal pure returns (string memory) {
        return string.concat('{"name":"', name, '","snap":', body, "}");
    }

    // ------------------------------------------------------------------ live settlement sequence
    function _swap(bool sell, uint256 amountIn) internal returns (bool ok, uint256 used, uint256 out, bytes memory err) {
        address tin = sell ? CBBTC : USDC;
        address tout = sell ? USDC : CBBTC;
        vm.prank(MORPHO);
        KERC20(tin).transfer(TAKER, amountIn);
        vm.prank(TAKER);
        KERC20(tin).approve(POOL, amountIn);
        vm.prank(TAKER);
        try KPool(POOL).swap(tin, tout, amountIn, 0, TAKER, block.timestamp) returns (uint256 u, uint256 o) {
            return (true, u, o, "");
        } catch (bytes memory e) {
            return (false, 0, 0, e);
        }
    }

    function _step(string memory path, bool first, string memory name, bool sell, uint256 amountIn, uint256[] memory px)
        internal
        returns (bool ok)
    {
        string memory pre = _liveStateOnly();
        (bool preOk, uint256 pUsed, uint256 pOut) = _preview(sell, amountIn);
        uint256 used;
        uint256 out;
        bytes memory err;
        (ok, used, out, err) = _swap(sell, amountIn);
        string memory head = string.concat(
            '{"name":"', name, '","sell":', _b(sell), ',"amountIn":', _u(amountIn), ',"ok":', _b(ok), ',"used":',
            _u(used), ',"out":', _u(out), ',"err":', _h(err)
        );
        head = string.concat(head, ',"preview":{"ok":', _b(preOk), ',"used":', _u(pUsed), ',"out":', _u(pOut), "}");
        _line(path, first, string.concat(head, ',"pre":', pre, ',"post":', _liveSnap(px), "}"));
    }

    function _preview(bool sell, uint256 amountIn) internal view returns (bool, uint256, uint256) {
        try KPool(POOL).previewSwap(sell, amountIn) returns (uint256 u, uint256 o, uint256) {
            return (true, u, o);
        } catch {
            return (false, 0, 0);
        }
    }

    /// @dev Smallest loan-asset input (on a coarse ladder up to `cap`) whose previewed poolAsset out exceeds `minOut`.
    function _searchBuy(uint256 minOut, uint256 lo, uint256 cap) internal view returns (uint256) {
        for (uint256 x = lo; x <= cap; x += 1e6) {
            (bool ok,, uint256 o) = _preview(false, x);
            if (ok && o > minOut) return x;
        }
        return 0;
    }

    /// @dev FLAMMStore.S.physicalPoolAsset (ERC-7201 base + 12): lowering it models unaccounted custody, so a buy's
    ///      payout exceeds the tracked physical balance and settlement must reclaim posted collateral.
    function _setPhysical(uint256 physical) internal {
        bytes32 slot = bytes32(uint256(0x5b7e76949cacd5346234367c3806fe494a22f183af782d834d5fc4ee5b0f4500) + 12);
        (uint256 cur,) = KPool(POOL).poolAssetPosition();
        require(uint256(vm.load(POOL, slot)) == cur, "physical slot");
        vm.store(POOL, slot, bytes32(physical));
    }

    function test_liveSettle() public {
        vm.pauseGasMetering();
        vm.createSelectFork(RPC, BLOCK);
        uint256[] memory px = _pricesLive();
        string memory path = _open("mm_live_settle.json");
        vm.writeLine(path, string.concat('{"prices":', _ua(px), ',"steps":['));
        _step(path, true, "sellBorrow", true, 100_000, px);
        vm.warp(block.timestamp + 30);
        if (!_step(path, false, "sellBorrow2", true, 100_000, px)) {
            _step(path, false, "sellBorrow2b", true, 30_000, px);
        }
        vm.warp(block.timestamp + 30);
        (uint256 physical,) = KPool(POOL).poolAssetPosition();
        (,, uint256 debt) = KPool(POOL).loanPosition();
        _setPhysical(physical / 5);
        uint256 x = _searchBuy(physical / 5, 1e6, debt * 95 / 100);
        _step(path, false, "buyReclaimIndebted", false, x == 0 ? debt / 2 : x, px);
        vm.warp(block.timestamp + 30);
        (,, debt) = KPool(POOL).loanPosition();
        if (!_step(path, false, "buyRepaySupply", false, debt + 20e6, px)) {
            _step(path, false, "buyRepaySupplyB", false, debt + 5e6, px);
        }
        vm.warp(block.timestamp + 30);
        (uint256 liq, uint256 sup,) = KPool(POOL).loanPosition();
        uint256 want = liq + sup + 8e6;
        uint256 sats = want * 1e12 * 104 / (px[0] * 100);
        _step(path, false, "sellWithdrawBorrow", true, sats, px);
        vm.warp(block.timestamp + 30);
        (physical,) = KPool(POOL).poolAssetPosition();
        (,, debt) = KPool(POOL).loanPosition();
        _setPhysical(physical / 20);
        x = _searchBuy(physical / 20, debt + 1e6, debt + 400e6);
        _step(path, false, "buyReclaimFree", false, x == 0 ? debt + 1e6 : x, px);
        vm.warp(block.timestamp + 30);
        _step(path, false, "sellSmall", true, 3_000, px);
        vm.writeLine(path, "]}");
    }

    // ------------------------------------------------------------------ the real sell
    function test_realSell() public {
        vm.pauseGasMetering();
        string memory path = _open("mm_real_sell.json");
        vm.createSelectFork(RPC, PRE_SELL_BLOCK);
        uint256[] memory px = _pricesLive();
        vm.writeLine(path, string.concat('{"preBlock":', _liveSnap(px)));
        vm.createSelectFork(RPC, REAL_SELL_TX);
        px = _pricesLive();
        vm.writeLine(path, string.concat(',"pre":', _liveSnap(px)));
        vm.transact(REAL_SELL_TX);
        vm.writeLine(path, string.concat(',"post":', _liveSnap(px), "}"));
    }

    // ------------------------------------------------------------------ IRM grid
    function test_irmGrid() public {
        vm.pauseGasMetering();
        vm.createSelectFork(RPC, BLOCK);
        MParams memory p = KMorpho(MORPHO).idToMarketParams(MID);
        bytes32 slot = keccak256(abi.encode(MID, uint256(0)));
        int256 live = KIrm(IRM).rateAtTarget(MID);
        require(uint256(vm.load(IRM, slot)) == uint256(live), "irm slot");
        int256[7] memory rats = [int256(0), 31709791, live, 1268391679, 63419583967, 5e9, 31709792];
        uint256[3] memory tsas = [uint256(0), 1e6, 1_567_730_220_003_881];
        uint256[8] memory utils = [uint256(0), 3000, 9000, 9500, 9999, 10000, 12000, 40000];
        uint256[8] memory els = [uint256(0), 1, 12, 3600, 86400, 30 days, 365 days, 5 * 365 days];
        string memory path = _open("mm_irm_grid.json");
        vm.writeLine(path, string.concat('{"timestamp":', vm.toString(block.timestamp), ',"rows":['));
        bool first = true;
        for (uint256 a; a < rats.length; ++a) {
            for (uint256 b; b < tsas.length; ++b) {
                for (uint256 c; c < utils.length; ++c) {
                    uint256 tba = tsas[b] == 0 ? (utils[c] == 0 ? 0 : 5 * c) : tsas[b] * utils[c] / 10000;
                    for (uint256 d; d < els.length; ++d) {
                        _irmRow(path, first, p, slot, rats[a], tsas[b], tba, block.timestamp - els[d]);
                        first = false;
                    }
                }
            }
        }
        _irmRow(path, false, p, slot, live, tsas[2], tsas[2] / 2, block.timestamp + 1);
        vm.writeLine(path, "]}");
    }

    function _irmRow(
        string memory path,
        bool first,
        MParams memory p,
        bytes32 slot,
        int256 rat,
        uint256 tsa,
        uint256 tba,
        uint256 lastUpdate
    ) internal {
        vm.store(IRM, slot, bytes32(uint256(rat)));
        MMarket memory m = MMarket(uint128(tsa), uint128(tsa * 1e6), uint128(tba), uint128(tba * 1e6), uint128(lastUpdate), 0);
        string memory row = string.concat(
            '{"rateAtTarget":', _i(rat), ',"tsa":', _u(tsa), ',"tba":', _u(tba), ',"lastUpdate":', _u(lastUpdate)
        );
        try KIrm(IRM).borrowRateView(p, m) returns (uint256 v) {
            vm.prank(MORPHO);
            uint256 w = KIrm(IRM).borrowRate(p, m);
            require(w == v, "view != mutating");
            row = string.concat(row, ',"rate":', _u(v), ',"endRateAtTarget":', _i(KIrm(IRM).rateAtTarget(MID)), "}");
        } catch (bytes memory e) {
            row = string.concat(row, ',"err":', _h(e), "}");
        }
        _line(path, first, row);
    }

    // ------------------------------------------------------------------ synthetic multi-venue router
    struct Multi {
        address r2;
        KMockPool mp;
        MParams pA;
        MParams pB;
        MParams pC;
        uint256 pxA;
        uint256 pxC;
    }

    function _seedMarket(MParams memory p, uint256 supplyAmt, uint256 collAmt, uint256 borrowAmt) internal {
        bytes32 id = keccak256(abi.encode(p));
        (,,,, uint128 lu,) = KMorpho(MORPHO).market(id);
        if (lu == 0) KMorpho(MORPHO).createMarket(p);
        address lender = address(0x1E4D);
        address borrower = address(0xB0B);
        vm.prank(MORPHO);
        KERC20(p.loanToken).transfer(lender, supplyAmt);
        vm.startPrank(lender);
        KERC20(p.loanToken).approve(MORPHO, supplyAmt);
        KMorpho(MORPHO).supply(p, supplyAmt, 0, lender, "");
        vm.stopPrank();
        vm.prank(MORPHO);
        KERC20(CBBTC).transfer(borrower, collAmt);
        vm.startPrank(borrower);
        KERC20(CBBTC).approve(MORPHO, collAmt);
        KMorpho(MORPHO).supplyCollateral(p, collAmt, borrower, "");
        KMorpho(MORPHO).borrow(p, borrowAmt, 0, borrower, borrower);
        vm.stopPrank();
    }

    function _setupMulti() internal returns (Multi memory m) {
        m.r2 = address(0x2222000000000000000000000000000000002222);
        vm.etch(m.r2, ROUTER.code);
        m.mp = new KMockPool(FACTORY, m.r2, CBBTC);
        vm.prank(FACTORY);
        KRouter(m.r2).registerPool(address(m.mp), CBBTC, 0.55e18, 0.02e18, 0.02e18);
        m.mp.exec(m.r2, abi.encodeCall(KRouter.addLoanAsset, (USDC)));
        m.mp.exec(m.r2, abi.encodeCall(KRouter.addLoanAsset, (WETH)));
        m.pA = KMorpho(MORPHO).idToMarketParams(MID);
        m.pB = MParams(USDC, CBBTC, m.pA.oracle, IRM, 0.77e18);
        _seedMarket(m.pB, 5000e6, 1e7, 3500e6);
        // a 10% market fee on the small market: accrual mints fee shares
        vm.prank(KMorpho(MORPHO).owner());
        KMorpho(MORPHO).setFee(m.pB, 0.1e18);
        KConstOracle oc = new KConstOracle(3e47); // 30 WETH per cbBTC at 1e36 * 10^(18-8)
        m.pC = MParams(WETH, CBBTC, address(oc), IRM, 0.77e18);
        _seedMarket(m.pC, 3e18, 2e7, 15e17);
        m.pxA = KConstOracle(m.pA.oracle).price() / 1e24;
        m.pxC = 3e11;
        m.mp.exec(m.r2, abi.encodeCall(KRouter.addVenue, (0, 0, abi.encode(m.pA), true, true, m.pxA)));
        m.mp.exec(m.r2, abi.encodeCall(KRouter.addVenue, (0, 0, abi.encode(m.pB), true, true, m.pxA)));
        m.mp.exec(m.r2, abi.encodeCall(KRouter.addVenue, (0, 1, abi.encode(m.pC), true, true, m.pxC)));
        m.mp.exec(m.r2, abi.encodeCall(KRouter.setVenueCaps, (0, 50e6, 10e6, 0)));
        m.mp.exec(m.r2, abi.encodeCall(KRouter.setVenueCaps, (1, 0, 3000e6, 0)));
        uint16[] memory bo = new uint16[](3);
        (bo[0], bo[1], bo[2]) = (1, 0, 2);
        uint16[] memory so = new uint16[](3);
        (so[0], so[1], so[2]) = (0, 1, 2);
        m.mp.exec(m.r2, abi.encodeCall(KRouter.setPriorities, (bo, so, bo, so)));
    }

    function _mprices(Multi memory m) internal pure returns (uint256[] memory px) {
        px = new uint256[](2);
        (px[0], px[1]) = (m.pxA, m.pxC);
    }

    function _mSnap(Multi memory m, bool views) internal view returns (string memory) {
        return string.concat(
            '{"block":', vm.toString(block.number), ',"timestamp":', vm.toString(block.timestamp), ',"router":',
            _routerState(m.r2, address(m.mp)), views ? string.concat(',"views":', _routerViews(m.r2, address(m.mp), _mprices(m))) : "",
            "}"
        );
    }

    function _give(Multi memory m, address token, uint256 amount) internal {
        vm.prank(MORPHO);
        KERC20(token).transfer(address(m.mp), amount);
        m.mp.approve(token, m.r2, amount);
    }

    function _op(Multi memory m, string memory path, string memory name, string memory args, bytes memory data)
        internal
    {
        string memory pre = _mSnap(m, false);
        string memory res;
        try m.mp.exec(m.r2, data) returns (bytes memory ret) {
            res = string.concat('"ok":true,"ret":', _h(ret));
        } catch (bytes memory e) {
            res = string.concat('"ok":false,"err":', _h(e));
        }
        // clear any unspent grant so the next op starts from a clean allowance
        m.mp.approve(USDC, m.r2, 0);
        m.mp.approve(WETH, m.r2, 0);
        m.mp.approve(CBBTC, m.r2, 0);
        _line(
            path, false,
            string.concat('{"name":"', name, '","args":', args, ",", res, ',"pre":', pre, ',"post":', _mSnap(m, true), "}")
        );
        vm.warp(block.timestamp + 13);
    }

    function _pa(uint256 a, uint256 b) internal pure returns (uint256[] memory x) {
        x = new uint256[](2);
        (x[0], x[1]) = (a, b);
    }

    function _fundArgs(uint8 idx, uint256 assets, uint256 coll, uint256 px) internal pure returns (string memory) {
        return string.concat(
            '{"op":"fund","idx":', vm.toString(uint256(idx)), ',"assets":', _u(assets), ',"collateralIn":', _u(coll),
            ',"priceWad":', _u(px), "}"
        );
    }

    function _fund(Multi memory m, string memory path, string memory name, uint8 idx, uint256 assets, uint256 coll, uint256 px)
        internal
    {
        _give(m, CBBTC, coll);
        _op(
            m, path, name, _fundArgs(idx, assets, coll, px),
            abi.encodeCall(KRouter.fund, (idx, assets, address(m.mp), coll, px))
        );
    }

    function _cascade(Multi memory m, string memory path, string memory name, bool repay, uint8 idx, uint256 assets)
        internal
    {
        _give(m, idx == 0 ? USDC : WETH, assets);
        _op(
            m, path, name,
            string.concat('{"op":"', repay ? "repayCascade" : "supplyCascade", '","idx":', vm.toString(uint256(idx)), ',"assets":', _u(assets), "}"),
            repay ? abi.encodeCall(KRouter.repayCascade, (idx, assets)) : abi.encodeCall(KRouter.supplyCascade, (idx, assets))
        );
    }

    function _reclaim(Multi memory m, string memory path, string memory name, uint256 assets, bool best) internal {
        uint256[] memory px = _mprices(m);
        _op(
            m, path, name,
            string.concat('{"op":"', best ? "reclaimBestEffort" : "reclaim", '","assets":', _u(assets), ',"priceWads":', _ua(px), "}"),
            best ? abi.encodeCall(KRouter.reclaimBestEffort, (assets, px)) : abi.encodeCall(KRouter.reclaim, (assets, px))
        );
    }

    /// @dev A ceiling on venue 1 priced between its rate after the plan's own withdrawal (quote side) and without
    ///      it, so only the post-withdrawal pricing refuses the slice; then a fund that needs that slice.
    function _rateCapWithdraw(Multi memory m, string memory path) internal {
        bytes32 idB = keccak256(abi.encode(m.pB));
        KVenueView memory vb = KRouter(m.r2).venue(address(m.mp), 1);
        (,, uint256 supB,,) = KRouter(m.r2).venuePosition(address(m.mp), 1);
        (,, uint256 supA,,) = KRouter(m.r2).venuePosition(address(m.mp), 0);
        (, uint256 rWith) = KAccount(vb.account).borrowRateAfter(idB, 5e6, supB);
        (, uint256 rWithout) = KAccount(vb.account).borrowRateAfter(idB, 5e6, 0);
        uint64 cap = uint64((rWith + rWithout) / 2);
        _op(
            m, path, "rateCapWithdrawSet",
            string.concat('{"op":"setVenueCaps","id":1,"debtCap":"0","supplyCap":"3000000000","maxBorrowRateWad":', _u(cap), "}"),
            abi.encodeCall(KRouter.setVenueCaps, (1, 0, 3000e6, cap))
        );
        _fund(m, path, "fundRateCapWithdraw", 0, supB + supA + 5e6, 20000, m.pxA);
        _op(
            m, path, "rateCapWithdrawClear",
            '{"op":"setVenueCaps","id":1,"debtCap":"0","supplyCap":"3000000000","maxBorrowRateWad":"0"}',
            abi.encodeCall(KRouter.setVenueCaps, (1, 0, 3000e6, 0))
        );
        _cascade(m, path, "supplyRefill", false, 0, 25e6);
    }

    function _idArgs(string memory op, uint16 id, uint256 assets) internal pure returns (string memory) {
        return string.concat('{"op":"', op, '","id":', vm.toString(uint256(id)), ',"assets":', _u(assets));
    }

    /// @dev The single-venue MMRouter entries the pool's pro-rata flows use, including the proportional collateral
    ///      withdrawal against the repay snapshot left in transient storage earlier in the same transaction.
    function _primitives(Multi memory m, string memory path) internal {
        _give(m, USDC, 3e6);
        _op(m, path, "primSupplyA", string.concat(_idArgs("supply", 0, 3e6), "}"), abi.encodeCall(KRouter.supply, (0, 3e6)));
        _give(m, USDC, 4e6);
        _op(m, path, "primSupplyB", string.concat(_idArgs("supply", 1, 4e6), "}"), abi.encodeCall(KRouter.supply, (1, 4e6)));
        _op(
            m, path, "primWithdrawMaxB", string.concat(_idArgs("withdrawSupplied", 1, type(uint256).max), "}"),
            abi.encodeCall(KRouter.withdrawSupplied, (1, type(uint256).max, address(m.mp)))
        );
        _op(
            m, path, "primWithdrawAmtA", string.concat(_idArgs("withdrawSupplied", 0, 1e6), "}"),
            abi.encodeCall(KRouter.withdrawSupplied, (0, 1e6, address(m.mp)))
        );
        _op(
            m, path, "primWithdrawTooMuchA", string.concat(_idArgs("withdrawSupplied", 0, 1e12), "}"),
            abi.encodeCall(KRouter.withdrawSupplied, (0, 1e12, address(m.mp)))
        );
        _give(m, CBBTC, 5000);
        _op(m, path, "primPostA", string.concat(_idArgs("postCollateral", 0, 5000), "}"), abi.encodeCall(KRouter.postCollateral, (0, 5000)));
        _op(
            m, path, "primBorrowA", string.concat(_idArgs("borrow", 0, 1e6), ',"priceWad":', _u(m.pxA), "}"),
            abi.encodeCall(KRouter.borrow, (0, 1e6, address(m.mp), m.pxA))
        );
        _op(
            m, path, "primBorrowTooMuchB", string.concat(_idArgs("borrow", 1, 1e12), ',"priceWad":', _u(m.pxA), "}"),
            abi.encodeCall(KRouter.borrow, (1, 1e12, address(m.mp), m.pxA))
        );
        _give(m, USDC, 4e5);
        _op(m, path, "primRepayA", string.concat(_idArgs("repay", 0, 4e5), "}"), abi.encodeCall(KRouter.repay, (0, 4e5)));
        _op(
            m, path, "primWithdrawPropA", string.concat(_idArgs("withdrawCollateral", 0, 1000), ',"priceWad":"0","proportional":true}'),
            abi.encodeCall(KRouter.withdrawCollateral, (0, 1000, 0, true))
        );
        _op(
            m, path, "primWithdrawPropTooMuchA", string.concat(_idArgs("withdrawCollateral", 0, 20000), ',"priceWad":"0","proportional":true}'),
            abi.encodeCall(KRouter.withdrawCollateral, (0, 20000, 0, true))
        );
        _op(
            m, path, "primWithdrawPricedA",
            string.concat(_idArgs("withdrawCollateral", 0, 2000), ',"priceWad":', _u(m.pxA), ',"proportional":false}'),
            abi.encodeCall(KRouter.withdrawCollateral, (0, 2000, m.pxA, false))
        );
        _op(
            m, path, "primWithdrawPropNoSnapC", string.concat(_idArgs("withdrawCollateral", 2, 100), ',"priceWad":"0","proportional":true}'),
            abi.encodeCall(KRouter.withdrawCollateral, (2, 100, 0, true))
        );
    }

    function test_multiVenue() public {
        vm.pauseGasMetering();
        vm.createSelectFork(RPC, BLOCK);
        Multi memory m = _setupMulti();
        string memory path = _open("mm_multi_venue.json");
        vm.writeLine(path, string.concat('{"setup":', _mSnap(m, true), ',"steps":[{"name":"start"}'));
        _fund(m, path, "fundSplit", 0, 60e6, 200000, m.pxA);
        _cascade(m, path, "supplySplit", false, 0, 30e6);
        _rateCapWithdraw(m, path);
        _fund(m, path, "fundWithdrawThenBorrow", 0, 40e6, 50000, m.pxA);
        _cascade(m, path, "repayOrder", true, 0, 15e6);
        _reclaim(m, path, "reclaimOrder", 5000, true);
        _reclaim(m, path, "reclaimBest", 40000, true);
        _reclaim(m, path, "reclaimStrictShort", 1e9, false);
        _fund(m, path, "fundOutsideDrawnSet", 1, 1e17, 100000, m.pxC);
        _op(m, path, "setMaxDrawn2", '{"op":"setMaxDrawnAssets","n":2}', abi.encodeCall(KRouter.setMaxDrawnAssets, (2)));
        _fund(m, path, "fundWeth", 1, 1e17, 1e6, m.pxC);
        _cascade(m, path, "supplyWeth", false, 1, 5e16);
        _primitives(m, path);
        vm.warp(block.timestamp + 1 days);
        _fund(m, path, "fundAfterDay", 0, 5e6, 0, m.pxA);
        uint256 ceil = KRouter(m.r2).fundingCeiling(address(m.mp), 0, 30000, m.pxA);
        _fund(m, path, "fundAtCeiling", 0, ceil, 30000, m.pxA);
        _fund(m, path, "fundBandOut", 0, 5e6, 1000, m.pxA * 90 / 100);
        // a ceiling that bites in the small market: priced between a small and a large slice
        bytes32 idB = keccak256(abi.encode(m.pB));
        (, uint256 rs) = KAccount(KRouter(m.r2).venue(address(m.mp), 1).account).borrowRateAfter(idB, 1e6, 0);
        (, uint256 rl) = KAccount(KRouter(m.r2).venue(address(m.mp), 1).account).borrowRateAfter(idB, 800e6, 0);
        uint64 cap = uint64((rs + rl) / 2);
        _op(
            m, path, "rateCap",
            string.concat('{"op":"setVenueCaps","id":1,"debtCap":"0","supplyCap":"3000000000","maxBorrowRateWad":', _u(cap), "}"),
            abi.encodeCall(KRouter.setVenueCaps, (1, 0, 3000e6, cap))
        );
        _fund(m, path, "fundUnderCap", 0, 200e6, 3e6, m.pxA);
        _fund(m, path, "fundUnderCapSmall", 0, 2e6, 10000, m.pxA);
        // venue 1's IRM unreadable: inside the grace, then quarantined
        vm.mockCallRevert(IRM, abi.encodeWithSelector(KIrm.borrowRateView.selector, m.pB), "");
        vm.mockCallRevert(IRM, abi.encodeWithSelector(KIrm.borrowRate.selector, m.pB), "");
        vm.warp(block.timestamp + 1800);
        _cascade(m, path, "repayGrace", true, 0, 5e6);
        _fund(m, path, "fundGrace", 0, 10e6, 20000, m.pxA);
        vm.warp(block.timestamp + 7200);
        _cascade(m, path, "repayQuarantine", true, 0, 5e6);
        _fund(m, path, "fundQuarantine", 0, 10e6, 20000, m.pxA);
        _reclaim(m, path, "reclaimQuarantine", 5000, true);
        _cascade(m, path, "supplyQuarantine", false, 0, 3e6);
        vm.clearMockedCalls();
        _cascade(m, path, "repayAll", true, 0, 1000e6);
        _reclaim(m, path, "reclaimAll", 1e9, true);
        vm.writeLine(path, "]}");
    }
}
