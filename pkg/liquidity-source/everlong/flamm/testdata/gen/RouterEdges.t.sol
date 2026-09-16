// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import "./FinancingEdgesBase.sol";
import {IMMRouter} from "src/interfaces/core/mm/IMMRouter.sol";

interface VAcct {
    function oraclePrice(bytes32) external view returns (bool, uint256);
    function borrowRateAfter(bytes32, uint256, uint256) external view returns (bool, uint256);
    function lltv(bytes32) external view returns (uint64);
}

interface VERC20 {
    function approve(address, uint256) external returns (bool);
}

/// @dev Edge generator for router.go against the DEPLOYED MMRouter (0x19A9…6bB4, linking the
///      deployed MMRouterLib) on the live c104 pool record, extended in place by the pool itself to four venues over
///      two loan assets: the live 86% USDC market, a created 77% USDC market with a fee, a created 91.5% USDC market
///      with NO rate model, and a created 86% market of a second 18-decimal loan asset (whose account the Factory's
///      deployer creates). Each scenario writes Blue state, rateAtTarget, oracles, IRM outages, caps, flags, pin,
///      band, priorities, pause and the managed figures, dumps the full state and every view, then replays each
///      Router entry from that state with its result and the full post-state.
contract RouterEdges is FinancingEdgesBase {
    uint256 internal constant P0 = 788432322552395; // cbBTC/USDC cross at the fork block
    uint256 internal constant P1 = 262810774184131; // cbBTC per 18-dp loan asset 1 (a 3000 USD unit), L18 per sat

    struct VS {
        uint128 tsa;
        uint128 tss;
        uint128 tba;
        uint128 tbs;
        uint64 elapsed;
        uint128 fee;
        uint256 supplyShares;
        uint128 borrowShares;
        uint128 collateral;
        int256 rat;
        uint256 oraclePrice;
        bool oracleDead;
        bool irmDead;
        uint256 managedColl;
        uint256 managedShares;
        uint128 debtCap;
        uint128 supplyCap;
        uint64 maxRate;
        bool borrowEnabled;
        bool supplyEnabled;
    }

    struct S {
        VS[4] v;
        uint128[2] loanDebtCap;
        uint128[2] loanSupplyCap;
        bool[2] loanBorrow;
        uint64 pin;
        uint64 band;
        uint8 maxDrawn;
        bool paused;
        uint16[] bo;
        uint16[] so;
        uint16[] wo;
        uint16[] ro;
        bool retire2;
        string tag;
    }

    struct Op {
        uint8 kind;
        uint8 idx;
        uint16 id;
        uint256 a;
        uint256 collIn;
        uint256 price;
        uint256 price1;
        bool prop;
        uint256 pre;
    }

    VToken internal loan1;
    VOracle[4] internal oracles; // [0] unused (live oracle is mocked)
    VMParams[4] internal params;
    bytes32[4] internal ids;
    address[4] internal accts;
    uint256 internal T0;
    uint256 internal venuesBase;

    function _setup() internal {
        vm.createSelectFork(RPC, BLOCK);
        T0 = block.timestamp + 3000;
        loan1 = new VToken(18);
        for (uint256 i = 1; i < 4; ++i) oracles[i] = new VOracle();
        oracles[1].set(787686868792500000000000000000000000000, false);
        oracles[2].set(787686868792500000000000000000000000000, false);
        oracles[3].set(P1 * 1e36, false);
        params[0] = VMParams(USDC, CBBTC, LIVE_ORACLE, IRM, 0.86e18);
        params[1] = VMParams(USDC, CBBTC, address(oracles[1]), IRM, 0.77e18);
        params[2] = VMParams(USDC, CBBTC, address(oracles[2]), address(0), 0.915e18);
        params[3] = VMParams(address(loan1), CBBTC, address(oracles[3]), IRM, 0.86e18);
        for (uint256 i = 1; i < 4; ++i) VMorpho(MORPHO).createMarket(params[i]);
        for (uint256 i; i < 4; ++i) ids[i] = keccak256(abi.encode(params[i]));
        require(ids[0] == LIVE_ID, "live id");
        vm.startPrank(POOL);
        IMMRouter(ROUTER).addVenue(0, 0, abi.encode(params[1]), true, true, P0);
        IMMRouter(ROUTER).addVenue(0, 0, abi.encode(params[2]), true, true, P0);
        IMMRouter(ROUTER).addLoanAsset(address(loan1));
        IMMRouter(ROUTER).addVenue(0, 1, abi.encode(params[3]), true, true, P1);
        IMMRouter(ROUTER).setMaxDrawnAssets(1);
        VERC20(USDC).approve(ROUTER, type(uint256).max);
        VERC20(CBBTC).approve(ROUTER, type(uint256).max);
        loan1.approve(ROUTER, type(uint256).max);
        vm.stopPrank();
        for (uint256 i; i < 4; ++i) accts[i] = IMMRouter(ROUTER).venue(POOL, uint16(i)).account;
        require(accts[0] == ACCOUNT && accts[1] == ACCOUNT && accts[3] != ACCOUNT, "accounts");
        deal(USDC, POOL, 1 << 200);
        deal(CBBTC, POOL, 1 << 200);
        loan1.mint(POOL, 1 << 200);
        deal(USDC, MORPHO, 1 << 200);
        deal(CBBTC, MORPHO, 1 << 200);
        loan1.mint(MORPHO, 1 << 200);
        venuesBase = uint256(keccak256(abi.encode(uint256(keccak256(abi.encode(POOL, uint256(1)))) + 3)));
    }

    // ------------------------------------------------------------------ scenario application
    function _apply(S memory s) internal {
        vm.clearMockedCalls();
        vm.startPrank(POOL);
        for (uint256 i; i < 4; ++i) {
            VS memory v = s.v[i];
            if (s.retire2 && i == 2) continue;
            IMMRouter(ROUTER).setVenueFlags(uint16(i), v.borrowEnabled, v.supplyEnabled);
            IMMRouter(ROUTER).setVenueCaps(uint16(i), v.debtCap, v.supplyCap, v.maxRate);
        }
        IMMRouter(ROUTER).setLoanCaps(0, s.loanDebtCap[0], s.loanSupplyCap[0], s.loanBorrow[0]);
        IMMRouter(ROUTER).setLoanCaps(1, s.loanDebtCap[1], s.loanSupplyCap[1], s.loanBorrow[1]);
        IMMRouter(ROUTER).setMaxDrawnAssets(s.maxDrawn);
        vm.stopPrank();
        for (uint256 i; i < 4; ++i) {
            VS memory v = s.v[i];
            if (s.retire2 && i == 2) {
                writeMarket(ids[i], v.tsa, v.tss, v.tba, v.tbs, uint128(T0 - v.elapsed), v.fee);
                writePosition(ids[i], accts[i], 0, 0, 0);
                vm.prank(POOL);
                IMMRouter(ROUTER).retireVenue(2);
                continue;
            }
            writeMarket(ids[i], v.tsa, v.tss, v.tba, v.tbs, uint128(T0 - v.elapsed), v.fee);
            writePosition(ids[i], accts[i], v.supplyShares, v.borrowShares, v.collateral);
            if (params[i].irm != address(0)) writeRat(ids[i], v.rat);
            if (i == 0) {
                if (v.oracleDead) vm.mockCallRevert(LIVE_ORACLE, abi.encodeWithSignature("price()"), "dead");
                else vm.mockCall(LIVE_ORACLE, abi.encodeWithSignature("price()"), abi.encode(v.oraclePrice));
            } else {
                oracles[i].set(v.oraclePrice, v.oracleDead);
            }
            if (v.irmDead) {
                vm.mockCallRevert(IRM, abi.encodeWithSelector(VIrmFull.borrowRateView.selector, params[i]), abi.encodeWithSignature("Error(string)", "irm dead"));
                vm.mockCallRevert(IRM, abi.encodeWithSelector(VIrmFull.borrowRate.selector, params[i]), abi.encodeWithSignature("Error(string)", "irm dead"));
            }
            vm.store(ROUTER, bytes32(venuesBase + 6 * i + 4), bytes32(v.managedColl));
            vm.store(ROUTER, bytes32(venuesBase + 6 * i + 5), bytes32(v.managedShares));
        }
        if (s.bo.length != 0) {
            vm.prank(POOL);
            IMMRouter(ROUTER).setPriorities(s.bo, s.so, s.wo, s.ro);
        }
        // pin and band straight into PoolRecord slots 0 / 1 (setPin would gate on minLltv)
        uint256 p0 = uint256(keccak256(abi.encode(POOL, uint256(1))));
        uint256 w0 = uint256(vm.load(ROUTER, bytes32(p0)));
        w0 = (w0 & ~(uint256(type(uint64).max) << 160)) | (uint256(s.pin) << 160);
        vm.store(ROUTER, bytes32(p0), bytes32(w0));
        uint256 w1 = uint256(vm.load(ROUTER, bytes32(p0 + 1)));
        w1 = (w1 & ~(uint256(type(uint64).max) << 64)) | (uint256(s.band) << 64);
        vm.store(ROUTER, bytes32(p0 + 1), bytes32(w1));
        vm.store(ROUTER, bytes32(0), bytes32(uint256(s.paused ? 1 : 0)));
        vm.warp(T0);
        (uint64 pin_,, uint64 band_) = IMMRouter(ROUTER).pin(POOL);
        require(pin_ == s.pin && band_ == s.band && IMMRouter(ROUTER).globalPaused() == s.paused, "pin/band/paused");
    }

    // ------------------------------------------------------------------ dumps
    function _ords(uint16[] memory o) internal pure returns (string memory s) {
        s = "[";
        for (uint256 i; i < o.length; ++i) s = string.concat(s, i == 0 ? "" : ",", vm.toString(o[i]));
        s = string.concat(s, "]");
    }

    function _venueJson(uint256 i, S memory s) internal view returns (string memory o) {
        IMMRouter.VenueView memory vv = IMMRouter(ROUTER).venue(POOL, uint16(i));
        (bool ook, uint256 op) = VAcct(accts[i]).oraclePrice(ids[i]);
        o = "{";
        o = add(o, "loanIndex", q(vv.loanIndex));
        o = add(o, "lltv", q(vv.lltvWad));
        o = add(o, "borrowEnabled", qb(vv.borrowEnabled));
        o = add(o, "supplyEnabled", qb(vv.supplyEnabled));
        o = add(o, "retired", qb(vv.retired));
        o = add(o, "debtCap", q(vv.debtCap));
        o = add(o, "supplyCap", q(vv.supplyCap));
        o = add(o, "maxRate", q(vv.maxBorrowRateWad));
        o = add(o, "managedColl", q(uint256(vm.load(ROUTER, bytes32(venuesBase + 6 * i + 4)))));
        o = add(o, "managedShares", q(uint256(vm.load(ROUTER, bytes32(venuesBase + 6 * i + 5)))));
        o = add(o, "hasIrm", qb(params[i].irm != address(0)));
        o = add(o, "irmDead", qb(s.v[i].irmDead));
        o = add(o, "rat", qi(params[i].irm == address(0) ? int256(0) : VIrm(IRM).rateAtTarget(ids[i])));
        o = add(o, "oracleOk", qb(ook));
        o = add(o, "oraclePrice", q(op));
        o = add(o, "acctLltv", q(VAcct(accts[i]).lltv(ids[i])));
        o = add(o, "market", marketJson(ids[i]));
        o = end(add(o, "position", positionJson(ids[i], accts[i])));
    }

    function _stateJson(S memory s) internal view returns (string memory o) {
        (uint64 pin_, uint64 gap_, uint64 band_) = IMMRouter(ROUTER).pin(POOL);
        (uint16[] memory bo, uint16[] memory so, uint16[] memory wo, uint16[] memory ro) = IMMRouter(ROUTER).priorities(POOL);
        o = "{";
        o = add(o, "paused", qb(IMMRouter(ROUTER).globalPaused()));
        o = add(o, "pin", q(pin_));
        o = add(o, "gap", q(gap_));
        o = add(o, "band", q(band_));
        o = add(o, "maxDrawn", q(IMMRouter(ROUTER).maxDrawnAssets(POOL)));
        o = add(o, "borrowOrder", _ords(bo));
        o = add(o, "supplyOrder", _ords(so));
        o = add(o, "withdrawOrder", _ords(wo));
        o = add(o, "repayOrder", _ords(ro));
        string memory ls = "[";
        for (uint256 i; i < 2; ++i) {
            IMMRouter.LoanView memory l = IMMRouter(ROUTER).loan(POOL, uint8(i));
            string memory x = "{";
            x = add(x, "decimals", q(l.decimals));
            x = add(x, "scale", q(l.loanScale));
            x = add(x, "debtCap", q(l.debtCap));
            x = add(x, "supplyCap", q(l.supplyCap));
            x = add(x, "borrowEnabled", qb(l.borrowEnabled));
            x = end(add(x, "retired", qb(l.retired)));
            ls = string.concat(ls, i == 0 ? "" : ",", x);
        }
        o = add(o, "loans", string.concat(ls, "]"));
        string memory vs = "[";
        for (uint256 i; i < 4; ++i) vs = string.concat(vs, i == 0 ? "" : ",", _venueJson(i, s));
        o = end(add(o, "venues", string.concat(vs, "]")));
    }

    function _call(bytes memory data) internal view returns (string memory) {
        (bool ok, bytes memory ret) = ROUTER.staticcall(data);
        return string.concat("{", kv("ok", qb(ok)), ",", kv("ret", qh(ret)), "}");
    }

    function _views() internal view returns (string memory o) {
        uint256[] memory pw = new uint256[](2);
        pw[0] = P0;
        pw[1] = P1;
        o = "{";
        o = add(o, "positions", _call(abi.encodeCall(IMMRouter.positions, (POOL))));
        o = add(o, "position0", _call(abi.encodeCall(IMMRouter.position, (POOL, 0))));
        o = add(o, "position1", _call(abi.encodeCall(IMMRouter.position, (POOL, 1))));
        o = add(o, "position2", _call(abi.encodeCall(IMMRouter.position, (POOL, 2))));
        o = add(o, "quarantine0", _call(abi.encodeCall(IMMRouter.quarantine, (POOL, 0))));
        o = add(o, "quarantine1", _call(abi.encodeCall(IMMRouter.quarantine, (POOL, 1))));
        o = add(o, "drawn", _call(abi.encodeCall(IMMRouter.drawnAssets, (POOL))));
        o = add(o, "minLltv", _call(abi.encodeCall(IMMRouter.minLltv, (POOL))));
        o = add(o, "reclaimable", _call(abi.encodeCall(IMMRouter.reclaimable, (POOL, pw))));
        pw[1] = 0;
        o = add(o, "reclaimableP1zero", _call(abi.encodeCall(IMMRouter.reclaimable, (POOL, pw))));
        pw = new uint256[](1);
        o = add(o, "reclaimableBadLen", _call(abi.encodeCall(IMMRouter.reclaimable, (POOL, pw))));
        string memory vs = "[";
        for (uint256 i; i < 4; ++i) {
            uint256 pr = i == 3 ? P1 : P0;
            string memory x = "{";
            x = add(x, "venuePosition", _call(abi.encodeCall(IMMRouter.venuePosition, (POOL, uint16(i)))));
            x = add(x, "health", _call(abi.encodeCall(IMMRouter.venueHealth, (POOL, uint16(i), pr))));
            x = add(x, "healthLow", _call(abi.encodeCall(IMMRouter.venueHealth, (POOL, uint16(i), pr / 2))));
            x = add(x, "readable", _call(abi.encodeCall(IMMRouter.venueReadable, (POOL, uint16(i)))));
            x = end(add(x, "freeLiquidity", _call(abi.encodeCall(IMMRouter.venueFreeLiquidity, (POOL, uint16(i))))));
            vs = string.concat(vs, i == 0 ? "" : ",", x);
        }
        o = end(add(o, "venueViews", string.concat(vs, "]")));
    }

    function _ceilings() internal view returns (string memory out) {
        uint256[9] memory colls = [uint256(0), 1, 1e4, 1e6, 3e7, 1e8, 1e10, uint256(1) << 128, type(uint256).max];
        out = "[";
        for (uint256 idx; idx < 3; ++idx) {
            uint256 base = idx == 1 ? P1 : P0;
            uint256[6] memory prices = [base, base * 1005 / 1000, base * 103 / 100, base / 2, 0, base * 99 / 100];
            for (uint256 c; c < colls.length; ++c) {
                for (uint256 k; k < prices.length; ++k) {
                    if (idx == 2 && (c > 1 || k > 0)) continue;
                    string memory x = "{";
                    x = add(x, "idx", q(idx));
                    x = add(x, "coll", q(colls[c]));
                    x = add(x, "price", q(prices[k]));
                    x = end(add(x, "res", _call(abi.encodeCall(IMMRouter.fundingCeiling, (POOL, uint8(idx), colls[c], prices[k])))));
                    out = string.concat(out, (idx == 0 && c == 0 && k == 0) ? "" : ",", x);
                }
            }
        }
        out = string.concat(out, "]");
    }

    // ------------------------------------------------------------------ ops
    function runOp(S memory s, Op memory o) external {
        uint256 snap = vm.snapshotState();
        bool ok;
        bytes memory ret;
        bool preOk;
        bytes memory preRet;
        uint256[] memory pw = new uint256[](2);
        pw[0] = o.price;
        pw[1] = o.price1;
        vm.startPrank(POOL);
        if (o.kind == 1) (ok, ret) = ROUTER.call(abi.encodeCall(IMMRouter.fund, (o.idx, o.a, POOL, o.collIn, o.price)));
        else if (o.kind == 2) (ok, ret) = ROUTER.call(abi.encodeCall(IMMRouter.repayCascade, (o.idx, o.a)));
        else if (o.kind == 3) (ok, ret) = ROUTER.call(abi.encodeCall(IMMRouter.supplyCascade, (o.idx, o.a)));
        else if (o.kind == 4) (ok, ret) = ROUTER.call(abi.encodeCall(IMMRouter.reclaim, (o.a, pw)));
        else if (o.kind == 5) (ok, ret) = ROUTER.call(abi.encodeCall(IMMRouter.reclaimBestEffort, (o.a, pw)));
        else if (o.kind == 6) (ok, ret) = ROUTER.call(abi.encodeCall(IMMRouter.borrow, (o.id, o.a, POOL, o.price)));
        else if (o.kind == 7) (ok, ret) = ROUTER.call(abi.encodeCall(IMMRouter.repay, (o.id, o.a)));
        else if (o.kind == 8) (ok, ret) = ROUTER.call(abi.encodeCall(IMMRouter.supply, (o.id, o.a)));
        else if (o.kind == 9) (ok, ret) = ROUTER.call(abi.encodeCall(IMMRouter.withdrawSupplied, (o.id, o.a, POOL)));
        else if (o.kind == 10) {
            if (o.prop) (preOk, preRet) = ROUTER.call(abi.encodeCall(IMMRouter.repay, (o.id, o.pre)));
            (ok, ret) = ROUTER.call(abi.encodeCall(IMMRouter.withdrawCollateral, (o.id, o.a, o.price, o.prop)));
        } else if (o.kind == 11) (ok, ret) = ROUTER.call(abi.encodeCall(IMMRouter.postCollateral, (o.id, o.a)));
        vm.stopPrank();
        string memory x = "{";
        x = add(x, "scenario", string.concat('"', s.tag, '"'));
        x = add(x, "kind", q(o.kind));
        x = add(x, "idx", q(o.idx));
        x = add(x, "id", q(o.id));
        x = add(x, "a", q(o.a));
        x = add(x, "collIn", q(o.collIn));
        x = add(x, "price", q(o.price));
        x = add(x, "price1", q(o.price1));
        x = add(x, "prop", qb(o.prop));
        x = add(x, "pre", q(o.pre));
        x = add(x, "preOk", qb(preOk));
        x = add(x, "preRet", qh(preRet));
        x = add(x, "ok", qb(ok));
        x = add(x, "ret", qh(ret));
        row(end(add(x, "post", _stateJson(s))));
        vm.revertToState(snap);
    }

    function runScenario(S memory s) external {
        uint256 snap = vm.snapshotState();
        _apply(s);
        string memory x = "{";
        x = add(x, "scenario", string.concat('"', s.tag, '"'));
        x = add(x, "kind", q(0));
        x = add(x, "now", q(T0));
        x = add(x, "state", _stateJson(s));
        x = add(x, "views", _views());
        row(end(add(x, "ceilings", _ceilings())));
        Op[] memory ops = _ops();
        for (uint256 i; i < ops.length; ++i) this.runOp(s, ops[i]);
        vm.revertToState(snap);
    }

    function _op(uint8 kind, uint8 idx, uint16 id, uint256 a, uint256 collIn, uint256 price) internal pure returns (Op memory o) {
        o.kind = kind;
        o.idx = idx;
        o.id = id;
        o.a = a;
        o.collIn = collIn;
        o.price = price;
        o.price1 = P1;
    }

    function _ops() internal view returns (Op[] memory ops) {
        ops = new Op[](72);
        uint256 n;
        uint256 c0 = _ceil(0, 1e8, P0);
        uint256 c0z = _ceil(0, 0, P0);
        uint256 c1 = _ceil(1, 1e8, P1);
        uint256[5] memory sizes = [uint256(1), c0z, c0z + 1, c0 == 0 ? 7 : c0, c0 + 1];
        for (uint256 i; i < sizes.length; ++i) ops[n++] = _op(1, 0, 0, sizes[i], i < 3 ? 0 : 1e8, P0);
        ops[n++] = _op(1, 0, 0, c0 / 2 + 1, 1e8, P0);
        ops[n++] = _op(1, 0, 0, 5e9, type(uint256).max / 2, P0);
        ops[n++] = _op(1, 0, 0, 1000e6, 1e8, P0 * 103 / 100);
        ops[n++] = _op(1, 1, 0, c1 == 0 ? 1e18 : c1, 1e8, P1);
        ops[n++] = _op(1, 1, 0, c1 + 1, 1e8, P1);
        ops[n++] = _op(1, 1, 0, 1, 0, P1);
        ops[n++] = _op(1, 2, 0, 1, 0, P1);
        uint256[5] memory rep = [uint256(1), 1000e6, 12e6, 50_000e6, type(uint256).max];
        for (uint256 i; i < rep.length; ++i) ops[n++] = _op(2, 0, 0, rep[i], 0, 0);
        ops[n++] = _op(2, 1, 0, 1e30, 0, 0);
        ops[n++] = _op(2, 3, 0, 1, 0, 0);
        uint256[4] memory sup = [uint256(1), 100e6, 3e12, type(uint128).max];
        for (uint256 i; i < sup.length; ++i) ops[n++] = _op(3, 0, 0, sup[i], 0, 0);
        ops[n++] = _op(3, 1, 0, 1e21, 0, 0);
        uint256 recl = _reclaimable(P0, P1);
        uint256[4] memory rc = [uint256(1), recl, recl + 1, 1e20];
        for (uint256 i; i < rc.length; ++i) {
            ops[n++] = _op(4, 0, 0, rc[i], 0, P0);
            ops[n++] = _op(5, 0, 0, rc[i], 0, P0);
        }
        Op memory o = _op(5, 0, 0, 1e9, 0, 0);
        o.price1 = 0;
        ops[n++] = o;
        ops[n++] = _op(6, 0, 0, 1e6, 0, P0);
        ops[n++] = _op(6, 0, 1, 50e6, 0, P0);
        ops[n++] = _op(6, 0, 2, 1, 0, P0);
        ops[n++] = _op(6, 0, 3, 1e15, 0, P1);
        ops[n++] = _op(6, 0, 0, 0, 0, P0);
        ops[n++] = _op(7, 0, 0, 3e6, 0, 0);
        ops[n++] = _op(7, 0, 1, type(uint256).max, 0, 0);
        ops[n++] = _op(7, 0, 3, 5, 0, 0);
        ops[n++] = _op(8, 0, 1, 10e6, 0, 0);
        ops[n++] = _op(8, 0, 2, 1, 0, 0);
        ops[n++] = _op(8, 0, 0, 0, 0, 0);
        ops[n++] = _op(9, 0, 1, 1, 0, 0);
        ops[n++] = _op(9, 0, 1, type(uint256).max, 0, 0);
        ops[n++] = _op(9, 0, 2, 1e9, 0, 0);
        ops[n++] = _op(9, 0, 0, 0, 0, 0);
        ops[n++] = _op(10, 0, 0, 1, 0, P0);
        ops[n++] = _op(10, 0, 1, 1e6, 0, P0);
        ops[n++] = _op(10, 0, 1, 5e6, 0, P0 * 97 / 100);
        ops[n++] = _op(10, 0, 0, 100, 0, P0 / 2);
        o = _op(10, 0, 1, 1e6, 0, 0);
        o.prop = true;
        o.pre = 400e6;
        ops[n++] = o;
        o = _op(10, 0, 0, 5000, 0, 0);
        o.prop = true;
        o.pre = 6e6;
        ops[n++] = o;
        o = _op(10, 0, 1, 1e6, 0, 0);
        o.prop = true;
        o.pre = 0;
        ops[n++] = o;
        // short-circuit order: `debtCap != 0 && debt + assets > cap` and `assets == 0 || assets > recognizedSupplied`
        ops[n++] = _op(6, 0, 1, type(uint256).max, 0, P0);
        ops[n++] = _op(6, 0, 0, type(uint256).max, 0, P0);
        ops[n++] = _op(6, 0, 3, type(uint256).max - 1, 0, P1);
        ops[n++] = _op(9, 0, 1, 0, 0, 0);
        ops[n++] = _op(9, 0, 1, type(uint256).max - 1, 0, 0);
        ops[n++] = _op(11, 0, 1, 12345, 0, 0);
        ops[n++] = _op(11, 0, 2, 0, 0, 0);
        assembly {
            mstore(ops, n)
        }
    }

    function _ceil(uint8 idx, uint256 coll, uint256 price) internal view returns (uint256) {
        (bool ok, bytes memory ret) = ROUTER.staticcall(abi.encodeCall(IMMRouter.fundingCeiling, (POOL, idx, coll, price)));
        return ok ? abi.decode(ret, (uint256)) : 0;
    }

    function _reclaimable(uint256 p0, uint256 p1) internal view returns (uint256) {
        uint256[] memory pw = new uint256[](2);
        pw[0] = p0;
        pw[1] = p1;
        (bool ok, bytes memory ret) = ROUTER.staticcall(abi.encodeCall(IMMRouter.reclaimable, (POOL, pw)));
        return ok ? abi.decode(ret, (uint256)) : 0;
    }

    // ------------------------------------------------------------------ scenarios
    function _base(string memory tag) internal pure returns (S memory s) {
        s.tag = tag;
        s.pin = 0.55e18;
        s.band = 0.02e18;
        s.maxDrawn = 1;
        s.loanBorrow = [true, true];
        // venue 0: the live market and position
        s.v[0] = VS(1567884338546841, 1418373654349324291165, 1410885862270200, 1260337866908047722881, 600, 0, 0,
            10096228333889, 26221, 1476127382, 787686868792500000000000000000000000000, false, false, 26221, 0,
            1.5e12, 2e12, 7386586395, true, true);
        s.v[1] = VS(4e12, 3.9e18, 3e12, 2.9e18, 1200, 0.05e18, 5e17, 1e15, 5e6, 3 * INIT_RAT,
            787686868792500000000000000000000000000, false, false, 5e6, 5e17, 0, 0, 0, true, true);
        s.v[2] = VS(2e11, 2e17, 5e10, 5e16, 50, 0, 1e17, 0, 0, 0, 787686868792500000000000000000000000000, false, false,
            0, 1e17, 0, 0, 0, true, true);
        s.v[3] = VS(5e21, 5e27, 2e21, 2e27, 300, 0, 0, 0, 0, INIT_RAT, P1 * 1e36, false, false, 0, 0, 0, 0, 0, true, true);
    }

    function test_router() public {
        _setup();
        open("mm_router_edges.json");
        S memory s = _base("base");
        this.runScenario(s);

        s = _base("rate-ceiling");
        s.v[0].supplyShares = 1e20;
        s.v[0].managedShares = 1e20;
        (, uint256 r0) = VAcct(ACCOUNT).borrowRateAfter(LIVE_ID, 0, 0);
        s.v[0].maxRate = uint64(r0 + 3);
        s.v[1].maxRate = uint64(uint256(INIT_RAT * 4));
        s.v[2].maxRate = 1;
        this.runScenario(s);

        s = _base("caps");
        s.v[0].debtCap = 11_400_000;
        s.v[1].debtCap = 3_000e6;
        s.v[1].supplyCap = 600_000e6;
        s.v[2].supplyCap = 1;
        s.loanDebtCap[0] = 2_500e6;
        s.loanSupplyCap[0] = 700_000e6;
        this.runScenario(s);

        s = _base("irm-outage");
        s.v[0].irmDead = true;
        s.v[0].elapsed = 7200;
        s.v[1].irmDead = true;
        s.v[1].elapsed = 1800;
        this.runScenario(s);

        s = _base("irm-outage-edge");
        s.v[0].irmDead = true;
        s.v[0].elapsed = 3601;
        s.v[1].irmDead = true;
        s.v[1].elapsed = 3600;
        s.v[3].irmDead = true;
        s.v[3].borrowShares = 1e24;
        s.v[3].collateral = 1e7;
        s.v[3].managedColl = 1e7;
        s.maxDrawn = 2;
        this.runScenario(s);

        s = _base("irm-outage-v1-past-grace");
        s.v[1].irmDead = true;
        s.v[1].elapsed = 7200;
        this.runScenario(s);

        s = _base("oracles");
        s.v[1].oraclePrice = 787686868792500000000000000000000000000 * 105 / 100;
        s.v[0].oracleDead = true;
        s.v[2].oraclePrice = 0;
        this.runScenario(s);

        s = _base("managed-low");
        s.v[0].managedColl = 20000;
        s.v[1].managedColl = 1e6;
        s.v[1].managedShares = 1e17;
        s.v[2].managedShares = 5e16;
        this.runScenario(s);

        s = _base("paused");
        s.paused = true;
        this.runScenario(s);

        s = _base("drawn-set");
        s.v[3].borrowShares = 1e24;
        s.v[3].collateral = 2e7;
        s.v[3].managedColl = 2e7;
        this.runScenario(s);

        s = _base("drawn-set-2");
        s.v[3].borrowShares = 1e24;
        s.v[3].collateral = 2e7;
        s.v[3].managedColl = 2e7;
        s.v[0].borrowShares = 0;
        s.v[1].borrowShares = 0;
        s.maxDrawn = 2;
        this.runScenario(s);

        s = _base("drawn-set-blocked");
        s.v[3].borrowShares = 1e24;
        s.v[3].collateral = 2e7;
        s.v[3].managedColl = 2e7;
        s.v[0].borrowShares = 0;
        s.v[1].borrowShares = 0;
        this.runScenario(s);

        s = _base("priorities");
        s.bo = new uint16[](4);
        s.so = new uint16[](4);
        s.wo = new uint16[](4);
        s.ro = new uint16[](4);
        s.bo[0] = 2; s.bo[1] = 1; s.bo[2] = 3; s.bo[3] = 0;
        s.so[0] = 1; s.so[1] = 3; s.so[2] = 0; s.so[3] = 2;
        s.wo[0] = 2; s.wo[1] = 0; s.wo[2] = 1; s.wo[3] = 3;
        s.ro[0] = 1; s.ro[1] = 0; s.ro[2] = 3; s.ro[3] = 2;
        s.v[0].supplyShares = 1e18;
        s.v[0].managedShares = 1e18;
        this.runScenario(s);

        s = _base("tiny-cash");
        s.v[0].tba = s.v[0].tsa - 3;
        s.v[1].tba = s.v[1].tsa - 2_000e6;
        s.v[1].supplyShares = 3.8e18;
        s.v[1].managedShares = 3.8e18;
        s.v[2].tba = s.v[2].tsa - 1;
        this.runScenario(s);

        s = _base("retired-disabled");
        s.retire2 = true;
        s.v[2].supplyShares = 0;
        s.v[2].managedShares = 0;
        s.v[1].borrowEnabled = false;
        s.loanBorrow[1] = false;
        this.runScenario(s);

        s = _base("pin-high-band0");
        s.pin = 0.75e18;
        s.band = 0;
        this.runScenario(s);

        s = _base("pin-low-underwater");
        s.pin = 0.3e18;
        s.v[0].collateral = 5000;
        s.v[0].managedColl = 5000;
        this.runScenario(s);

        for (uint256 seed; seed < 10; ++seed) this.runScenario(_random(seed));
        close();
    }

    function _random(uint256 seed) internal pure returns (S memory s) {
        s = _base(string.concat("random-", vm.toString(seed)));
        for (uint256 i; i < 4; ++i) {
            VS memory v = s.v[i];
            uint256 k = rnd(seed, 100 + i);
            v.elapsed = uint64([uint256(0), 1, 600, 3599, 3601, 86400][k % 6]);
            v.tba = uint128(uint256(v.tsa) * (rnd(seed, 200 + i) % 1001) / 1000);
            v.borrowShares = uint128(uint256(v.borrowShares) * (rnd(seed, 300 + i) % 3));
            v.collateral = uint128(uint256(v.collateral) * (rnd(seed, 400 + i) % 4) + rnd(seed, 410 + i) % 1000);
            v.managedColl = rnd(seed, 500 + i) % 3 == 0 ? v.collateral / 2 : v.collateral;
            v.supplyShares = rnd(seed, 600 + i) % 2 == 0 ? 0 : uint256(v.tss) * (rnd(seed, 610 + i) % 100) / 100;
            v.managedShares = rnd(seed, 620 + i) % 3 == 0 ? v.supplyShares / 3 : v.supplyShares;
            v.irmDead = rnd(seed, 700 + i) % 5 == 0;
            v.oracleDead = rnd(seed, 710 + i) % 9 == 0;
            v.oraclePrice = v.oraclePrice * (970 + rnd(seed, 720 + i) % 61) / 1000;
            v.debtCap = rnd(seed, 800 + i) % 3 == 0 ? uint128(rnd(seed, 810 + i) % 1e13) : 0;
            v.supplyCap = rnd(seed, 820 + i) % 3 == 0 ? uint128(rnd(seed, 830 + i) % 1e13) : 0;
            v.maxRate = rnd(seed, 840 + i) % 3 == 0 ? uint64(uint256(MIN_RAT) + rnd(seed, 850 + i) % uint256(4 * MAX_RAT)) : 0;
            v.borrowEnabled = rnd(seed, 860 + i) % 4 != 0;
            v.supplyEnabled = rnd(seed, 870 + i) % 4 != 0;
            v.fee = uint128([uint256(0), 0.1e18, 0.25e18][rnd(seed, 880 + i) % 3]);
        }
        s.loanDebtCap[0] = rnd(seed, 900) % 3 == 0 ? uint128(rnd(seed, 901) % 1e10) : 0;
        s.loanSupplyCap[0] = rnd(seed, 902) % 3 == 0 ? uint128(rnd(seed, 903) % 1e12) : 0;
        s.loanDebtCap[1] = rnd(seed, 904) % 3 == 0 ? uint128(rnd(seed, 905) % 1e22) : 0;
        s.loanBorrow = [rnd(seed, 906) % 5 != 0, rnd(seed, 907) % 5 != 0];
        s.pin = uint64(0.4e18 + rnd(seed, 908) % 0.3e18);
        s.band = uint64([uint256(0), 0.01e18, 0.02e18, 0.05e18][rnd(seed, 909) % 4]);
        s.maxDrawn = uint8(1 + rnd(seed, 910) % 2);
        s.paused = rnd(seed, 911) % 10 == 0;
    }
}
