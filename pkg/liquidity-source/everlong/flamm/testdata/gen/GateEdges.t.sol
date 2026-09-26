// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import "./FinancingEdgesBase.sol";
import {FLAMMGateLib} from "src/core/flamm/FLAMMGateLib.sol";
import {FLAMMStore} from "src/core/flamm/FLAMMStore.sol";
import {IMMRouter} from "src/interfaces/core/mm/IMMRouter.sol";
import {IPriceFeed} from "src/interfaces/core/IPriceFeed.sol";
import {PoolContext} from "src/interfaces/core/flamm/IFLAMMHooks.sol";

contract VGateRouter {
    uint256[] public sup;
    uint256[] public debt;
    uint256 public posted;
    bool[] public qAny;
    uint256[] public qDebt;
    uint256[] public qColl;

    function set(uint256[] memory s, uint256[] memory d, uint256 p, bool[] memory a, uint256[] memory qd, uint256[] memory qc)
        external
    {
        sup = s;
        debt = d;
        posted = p;
        qAny = a;
        qDebt = qd;
        qColl = qc;
    }

    function positions(address) external view returns (uint256[] memory, uint256[] memory, uint256[] memory, uint256) {
        return (new uint256[](sup.length), sup, debt, posted);
    }

    function quarantine(address, uint8 idx) external view returns (bool, uint256, uint256) {
        return (qAny[idx], qDebt[idx], qColl[idx]);
    }
}

contract VGateFeed {
    mapping(address => uint256) public cross;
    mapping(address => bool) public crossOk;
    mapping(address => uint256) public usd;
    mapping(address => bool) public usdOk;

    function set(address t, bool cok, uint256 c, bool uok, uint256 u) external {
        cross[t] = c;
        crossOk[t] = cok;
        usd[t] = u;
        usdOk[t] = uok;
    }

    function peekCross(address, address t) external view returns (bool, uint256, uint48) {
        return (crossOk[t], cross[t], 1);
    }

    function peekUsd(address t) external view returns (bool, uint256, uint48) {
        return (usdOk[t], usd[t], 1);
    }
}

/// @dev FLAMMGateLib (80abd43) behind external entry points: the pure law on arbitrary books and the storage
///      composites over a FLAMMStore.S at the ERC-7201 slot with a mock router / feed.
contract VGateHarness {
    function netL18(FLAMMGateLib.Leg memory L) external pure returns (int256) {
        return FLAMMGateLib.netL18(L);
    }

    function netPW(FLAMMGateLib.Leg memory L) external pure returns (int256) {
        return FLAMMGateLib.netPW(L);
    }

    function exposurePW(FLAMMGateLib.Book memory b) external pure returns (uint256) {
        return FLAMMGateLib.exposurePW(b);
    }

    function gross(FLAMMGateLib.Book memory b) external pure returns (uint256) {
        return FLAMMGateLib.gross(b);
    }

    function boundPW(uint256 g, uint256 l) external pure returns (uint256) {
        return FLAMMGateLib.boundPW(g, l);
    }

    function boundOf(uint256 g, uint256 p, uint256 l) external pure returns (uint256) {
        return FLAMMGateLib.boundOf(g, p, l);
    }

    function lift(uint256 h, uint256 l, uint256 f) external pure returns (uint256) {
        return FLAMMGateLib.lift(h, l, f);
    }

    function roomWad(int256 u, uint256 g, uint256 p, uint256 l, uint256 f) external pure returns (uint256) {
        return FLAMMGateLib.roomWad(u, g, p, l, f);
    }

    function headOf(FLAMMGateLib.Book memory b, uint256 idx, uint256 u, uint256 l) external pure returns (uint256) {
        return FLAMMGateLib.headOf(b, idx, u, l);
    }

    function structuralDistWad(uint256 l, uint256 ll) external pure returns (uint256) {
        return FLAMMGateLib.structuralDistWad(l, ll);
    }

    function requiredPosted(uint256 d, uint256 s, uint256 l, uint256 p) external pure returns (uint256) {
        return FLAMMGateLib.requiredPosted(d, s, l, p);
    }

    function requiredPostedAll(FLAMMGateLib.Book memory b, uint256 l) external pure returns (uint256) {
        return FLAMMGateLib.requiredPostedAll(b, l);
    }

    function navAt(FLAMMGateLib.Book memory b) external pure returns (uint256) {
        return FLAMMGateLib.navAt(b);
    }

    function context(FLAMMGateLib.Book memory b, uint256 p, uint48 ts, uint256 supply)
        external
        pure
        returns (PoolContext memory)
    {
        return FLAMMGateLib.context(b, p, ts, supply);
    }

    function roomNative(FLAMMGateLib.Book memory b, uint256 idx, uint256 u) external view returns (uint256) {
        return FLAMMGateLib.roomNative(FLAMMStore.s(), b, idx, u);
    }

    function setStore(
        address router,
        address feed,
        address[] memory tokens,
        uint256[] memory scales,
        uint256[] memory liquids,
        uint256 physical,
        uint64 ltv,
        uint64 phi,
        uint64 eps
    ) external {
        FLAMMStore.S storage $ = FLAMMStore.s();
        $.poolAsset = address(0xB7C);
        $.router = IMMRouter(router);
        $.priceFeed = IPriceFeed(feed);
        delete $.loans;
        for (uint256 i; i < tokens.length; ++i) {
            FLAMMStore.LoanCfg storage c = $.loans.push();
            c.token = tokens[i];
            c.scale = scales[i];
            c.liquid = liquids[i];
        }
        $.physicalPoolAsset = physical;
        $.ltvWad = ltv;
        $.phiWad = phi;
        $.roomEpsilonWad = eps;
    }

    function assertGate() external view {
        FLAMMGateLib.assertGate(FLAMMStore.s());
    }

    function anchor() external view returns (int256[] memory, uint256, bool) {
        return FLAMMGateLib.anchor(FLAMMStore.s());
    }

    function assertEntryGate(int256[] memory u0, uint256 g0, bool q) external view {
        FLAMMGateLib.assertEntryGate(FLAMMStore.s(), u0, g0, q);
    }

    function assertExitNotWorsened(int256[] memory u0, uint256 g0) external view {
        FLAMMGateLib.assertExitNotWorsened(FLAMMStore.s(), u0, g0);
    }

    function totalAssets() external view returns (uint256) {
        return FLAMMGateLib.totalAssets(FLAMMStore.s());
    }
}

contract GateEdges is FinancingEdgesBase {
    struct G {
        uint256 n;
        uint256[4] scale;
        uint256[4] liquid;
        uint256[4] supplied;
        uint256[4] debt;
        uint256[4] price;
        uint256[4] cross;
        bool[4] okCross;
        uint256[4] usd;
        bool[4] okUsd;
        uint256 physical;
        uint256 posted;
        uint64 ltv;
        uint64 phi;
        uint64 eps;
        bool[4] qAny;
        uint256[4] qDebt;
        uint256[4] qColl;
        int256[4] u0;
        uint256 gross0;
        bool quarantined;
        int256 uArg;
        uint256 headArg;
        uint256 ltvArg;
        uint256 phiArg;
        uint256 lltvArg;
        uint256 routerLegs;
        string tag;
    }

    VGateHarness internal h;
    VGateRouter internal router;
    VGateFeed internal feed;
    address[4] internal tokens = [address(0x1001), address(0x1002), address(0x1003), address(0x1004)];

    function _res(bool ok, bytes memory ret) internal pure returns (string memory) {
        return string.concat("{", kv("ok", qb(ok)), ",", kv("ret", qh(ret)), "}");
    }

    function _book(G memory g) internal pure returns (FLAMMGateLib.Book memory b) {
        b.physical = g.physical;
        b.posted = g.posted;
        b.legs = new FLAMMGateLib.Leg[](g.n);
        for (uint256 i; i < g.n; ++i) {
            b.legs[i] = FLAMMGateLib.Leg(g.liquid[i], g.supplied[i], g.debt[i], g.scale[i], g.price[i], g.cross[i]);
        }
    }

    function _arr(uint256[4] memory a, uint256 n) internal pure returns (uint256[] memory o) {
        o = new uint256[](n);
        for (uint256 i; i < n; ++i) o[i] = a[i];
    }

    function _in(G memory g) internal pure returns (string memory o) {
        o = "{";
        o = add(o, "tag", string.concat('"', g.tag, '"'));
        o = add(o, "n", q(g.n));
        o = add(o, "scale", arr(_arr(g.scale, g.n)));
        o = add(o, "liquid", arr(_arr(g.liquid, g.n)));
        o = add(o, "supplied", arr(_arr(g.supplied, g.n)));
        o = add(o, "debt", arr(_arr(g.debt, g.n)));
        o = add(o, "price", arr(_arr(g.price, g.n)));
        o = add(o, "cross", arr(_arr(g.cross, g.n)));
        o = add(o, "usd", arr(_arr(g.usd, g.n)));
        string memory oc = "[";
        string memory ou = "[";
        string memory qa = "[";
        string memory u0 = "[";
        for (uint256 i; i < g.n; ++i) {
            oc = string.concat(oc, i == 0 ? "" : ",", qb(g.okCross[i]));
            ou = string.concat(ou, i == 0 ? "" : ",", qb(g.okUsd[i]));
            qa = string.concat(qa, i == 0 ? "" : ",", qb(g.qAny[i]));
            u0 = string.concat(u0, i == 0 ? "" : ",", qi(g.u0[i]));
        }
        o = add(o, "okCross", string.concat(oc, "]"));
        o = add(o, "okUsd", string.concat(ou, "]"));
        o = add(o, "qAny", string.concat(qa, "]"));
        o = add(o, "qDebt", arr(_arr(g.qDebt, g.n)));
        o = add(o, "qColl", arr(_arr(g.qColl, g.n)));
        o = add(o, "u0", string.concat(u0, "]"));
        o = add(o, "physical", q(g.physical));
        o = add(o, "posted", q(g.posted));
        o = add(o, "ltv", q(g.ltv));
        o = add(o, "phi", q(g.phi));
        o = add(o, "eps", q(g.eps));
        o = add(o, "gross0", q(g.gross0));
        o = add(o, "quarantined", qb(g.quarantined));
        o = add(o, "uArg", qi(g.uArg));
        o = add(o, "headArg", q(g.headArg));
        o = add(o, "ltvArg", q(g.ltvArg));
        o = add(o, "phiArg", q(g.phiArg));
        o = add(o, "lltvArg", q(g.lltvArg));
        o = end(add(o, "routerLegs", q(g.routerLegs)));
    }

    function _c(bytes memory data) internal view returns (string memory) {
        (bool ok, bytes memory ret) = address(h).staticcall(data);
        return _res(ok, ret);
    }

    function runCase(G memory g) external {
        // the store first: roomNative reads its dials
        {
            uint256[] memory sup = _arr(g.supplied, g.routerLegs);
            uint256[] memory debt = _arr(g.debt, g.routerLegs);
            bool[] memory qa = new bool[](g.n);
            for (uint256 i; i < g.n; ++i) qa[i] = g.qAny[i];
            router.set(sup, debt, g.posted, qa, _arr(g.qDebt, g.n), _arr(g.qColl, g.n));
            address[] memory toks = new address[](g.n);
            for (uint256 i; i < g.n; ++i) {
                toks[i] = tokens[i];
                feed.set(tokens[i], g.okCross[i], g.price[i], g.okUsd[i], g.usd[i]);
            }
            h.setStore(address(router), address(feed), toks, _arr(g.scale, g.n), _arr(g.liquid, g.n), g.physical, g.ltv, g.phi, g.eps);
        }
        FLAMMGateLib.Book memory b = _book(g);
        string memory o = "{";
        o = add(o, "in", _in(g));
        string memory legs = "[";
        for (uint256 i; i < g.n; ++i) {
            string memory x = "{";
            x = add(x, "netL18", _c(abi.encodeCall(VGateHarness.netL18, (b.legs[i]))));
            x = add(x, "netPW", _c(abi.encodeCall(VGateHarness.netPW, (b.legs[i]))));
            x = add(x, "requiredPosted", _c(abi.encodeCall(VGateHarness.requiredPosted, (g.debt[i], g.scale[i], g.ltvArg, g.price[i]))));
            x = add(x, "headOf", _c(abi.encodeCall(VGateHarness.headOf, (b, i, g.headArg, g.ltvArg))));
            x = end(add(x, "roomNative", _c(abi.encodeCall(VGateHarness.roomNative, (b, i, g.headArg)))));
            legs = string.concat(legs, i == 0 ? "" : ",", x);
        }
        o = add(o, "legs", string.concat(legs, "]"));
        o = add(o, "exposurePW", _c(abi.encodeCall(VGateHarness.exposurePW, (b))));
        o = add(o, "gross", _c(abi.encodeCall(VGateHarness.gross, (b))));
        o = add(o, "boundPW", _c(abi.encodeCall(VGateHarness.boundPW, (g.physical, g.ltvArg))));
        o = add(o, "boundOf", _c(abi.encodeCall(VGateHarness.boundOf, (g.physical, g.price[0], g.ltvArg))));
        o = add(o, "lift", _c(abi.encodeCall(VGateHarness.lift, (g.headArg, g.ltvArg, g.phiArg))));
        o = add(o, "roomWad", _c(abi.encodeCall(VGateHarness.roomWad, (g.uArg, g.physical, g.price[0], g.ltvArg, g.phiArg))));
        o = add(o, "structuralDistWad", _c(abi.encodeCall(VGateHarness.structuralDistWad, (g.ltvArg, g.lltvArg))));
        o = add(o, "requiredPostedAll", _c(abi.encodeCall(VGateHarness.requiredPostedAll, (b, g.ltvArg))));
        o = add(o, "navAt", _c(abi.encodeCall(VGateHarness.navAt, (b))));
        o = add(o, "context", _c(abi.encodeCall(VGateHarness.context, (b, g.price[0], uint48(g.headArg % (1 << 48)), g.gross0))));
        int256[] memory u0 = new int256[](g.n);
        for (uint256 i; i < g.n; ++i) u0[i] = g.u0[i];
        o = add(o, "assertGate", _c(abi.encodeCall(VGateHarness.assertGate, ())));
        o = add(o, "anchor", _c(abi.encodeCall(VGateHarness.anchor, ())));
        o = add(o, "entry", _c(abi.encodeCall(VGateHarness.assertEntryGate, (u0, g.gross0, g.quarantined))));
        o = add(o, "exit", _c(abi.encodeCall(VGateHarness.assertExitNotWorsened, (u0, g.gross0))));
        o = add(o, "totalAssets", _c(abi.encodeCall(VGateHarness.totalAssets, ())));
        row(end(o));
    }

    function _base(string memory tag, uint256 n) internal pure returns (G memory g) {
        g.tag = tag;
        g.n = n;
        uint256[4] memory scales = [uint256(1e12), 1, 1e10, 1e12];
        for (uint256 i; i < 4; ++i) {
            g.scale[i] = scales[i];
            g.price[i] = i == 0 ? 788432322552395 : i == 1 ? 262810774184131 : 12345678901234 * (i + 1);
            g.okCross[i] = true;
            g.okUsd[i] = true;
            g.usd[i] = [uint256(1e18), 3000e18, 1.001e18, 0.9998e18][i];
            g.cross[i] = i == 0 ? 1e18 : g.usd[i];
        }
        g.debt[0] = 11_301_759;
        g.liquid[0] = 0;
        g.physical = 219_423;
        g.posted = 15_000;
        g.ltv = 0.5e18;
        g.phi = 0.9e18;
        g.eps = 0.001e18;
        g.ltvArg = 0.5e18;
        g.phiArg = 0.9e18;
        g.lltvArg = 0.86e18;
        g.headArg = 1e20;
        g.uArg = 5e20;
        g.gross0 = 234_423;
        g.u0[0] = 11_301_759e12;
        g.routerLegs = n;
    }

    function test_gate() public {
        h = new VGateHarness();
        router = new VGateRouter();
        feed = new VGateFeed();
        open("gate_edges.json");
        G memory g;
        uint256 max = type(uint256).max;
        for (uint256 k; k < 60; ++k) {
            g = _base(string.concat("edge-", vm.toString(k)), 1 + k % 3);
            if (k == 0) {}
            if (k == 1) { g.debt[0] = max / 1e12 + 1; }
            if (k == 2) { g.debt[0] = max / 1e12; g.price[0] = 1; }
            if (k == 3) { g.supplied[0] = max; g.liquid[0] = 1; }
            if (k == 4) { g.supplied[0] = max / 1e12; g.liquid[0] = 1; }
            if (k == 5) { g.debt[0] = uint256(type(int256).max) / 1e12 + 1; g.price[0] = 1e30; } // int256 wrap of debt*scale
            if (k == 6) { g.price[0] = 0; }
            if (k == 7) { g.price[0] = 0; g.debt[0] = 0; }
            if (k == 8) { g.price[0] = 0; g.debt[0] = 5; g.liquid[0] = 5; }
            if (k == 9) { g.debt[0] = 1; g.price[0] = 1e30 + 1; } // netPW ceil of a tiny positive
            if (k == 10) { g.debt[0] = 0; g.liquid[0] = 1; g.price[0] = 1e31; } // floor of a tiny surplus to -0
            if (k == 11) { g.debt[0] = 788432322552395; g.price[0] = 788432322552395; g.scale[0] = 1; } // exact ceil tie
            if (k == 12) { g.ltvArg = 1e18; g.phiArg = 1e18; } // lift denominator zero
            if (k == 13) { g.ltvArg = 1e18; g.phiArg = 1e18 + 1; } // lift underflow
            if (k == 14) { g.ltvArg = 1e18 - 1; g.phiArg = 1e18; g.headArg = max; } // lift quotient overflow
            if (k == 15) { g.ltvArg = max; g.phiArg = 1; }
            if (k == 16) { g.physical = max; g.posted = 1; }
            if (k == 17) { g.physical = max / 788432322552395 + 1; }
            if (k == 18) { g.uArg = type(int256).min + 1; g.physical = max / 788432322552395; g.ltvArg = 2e18; }
            if (k == 19) { g.uArg = -1; g.physical = 0; }
            if (k == 20) { g.headArg = max; g.ltvArg = 1; }
            if (k == 21) { g.lltvArg = 0; }
            if (k == 22) { g.lltvArg = g.ltvArg; }
            if (k == 23) { g.lltvArg = 3; g.ltvArg = 1; }
            if (k == 24) { g.ltvArg = 0; }
            if (k == 25) { g.ltvArg = max; g.price[0] = 2; }
            if (k == 26) { g.debt[1] = 1e30; g.price[1] = 0; g.n = 2; g.routerLegs = 2; }
            if (k == 27) { g.cross[1] = 0; g.n = 2; g.routerLegs = 2; g.liquid[1] = 1; }
            if (k == 28) { g.cross[1] = 0; g.n = 2; g.routerLegs = 2; }
            if (k == 29) { g.liquid[0] = max / 1e12 + 1; }
            if (k == 30) { g.cross[0] = max; g.debt[0] = 2; }
            if (k == 31) { g.eps = 1e18 - 1; }
            if (k == 32) { g.scale[0] = 0; }
            if (k == 33) { g.routerLegs = g.n - 1; }
            if (k == 34) { g.qAny[0] = true; g.qDebt[0] = 11_866_847; g.qColl[0] = 15_000; g.quarantined = true; g.u0[0] = 0; }
            if (k == 35) { g.qAny[0] = true; g.qDebt[0] = max; g.quarantined = true; }
            if (k == 36) { g.gross0 = 0; g.physical = 1; g.posted = 0; }
            if (k == 37) { g.u0[0] = -5; g.physical = 1; g.posted = 0; }
            if (k == 38) { g.u0[0] = 1; g.physical = 1; g.posted = 0; g.gross0 = 1; }
            if (k == 39) { g.u0[0] = 11_301_759e12; g.debt[0] = 11_301_759; g.physical = 1000; g.posted = 0; g.gross0 = 1000; }
            if (k == 40) { g.u0[0] = type(int256).max; g.gross0 = max; g.physical = 1; g.posted = 0; }
            if (k == 41) { g.u0[0] = 1e30; g.gross0 = 1e40; g.physical = 1; g.posted = 0; }
            if (k == 42) { g.physical = 0; g.posted = 0; g.gross0 = 5; g.u0[0] = 5; }
            if (k == 43) { g.debt[0] = 0; g.supplied[0] = 1e12; g.liquid[0] = 7; }
            if (k == 44) { g.okUsd[0] = false; g.n = 2; g.routerLegs = 2; g.debt[1] = 1e18; }
            if (k == 45) { g.usd[0] = 0; g.n = 2; g.routerLegs = 2; g.debt[1] = 1e18; }
            if (k == 46) { g.okCross[0] = false; }
            if (k == 47) { g.okCross[1] = false; g.n = 2; g.routerLegs = 2; }
            if (k == 48) { g.okCross[1] = false; g.n = 2; g.routerLegs = 2; g.supplied[1] = 1; }
            if (k == 49) { g.eps = 0; g.phi = 0; g.ltv = 0; }
            if (k == 50) { g.qAny[1] = true; g.qDebt[1] = 1e18; g.qColl[1] = 1000; g.quarantined = true; g.n = 2; g.routerLegs = 2; g.debt[1] = 2e18; g.u0[1] = 1e17; }
            if (k == 51) { g.qAny[0] = true; g.qDebt[0] = 1; g.qColl[0] = max; g.quarantined = true; }
            if (k == 52) { g.uArg = int256(uint256(1e40)); g.physical = 1e10; g.ltvArg = 0.9e18; }
            if (k == 53) { g.headArg = 0; g.uArg = 0; }
            if (k == 54) { g.debt[0] = 1e6; g.liquid[0] = 1e6; }
            if (k == 55) { g.physical = 28_668_861; g.posted = 0; }
            if (k == 56) { g.physical = 28_668_860; g.posted = 0; g.gross0 = 28_668_860; g.u0[0] = int256(uint256(11_301_759e12) * 1e18 / 788432322552395); }
            if (k == 57) { g.headArg = max; g.ltv = 0.5e18; g.phi = 0.9e18; }
            if (k == 58) { g.supplied[1] = max / 2; g.liquid[1] = max / 2 + 2; g.n = 2; g.routerLegs = 2; }
            if (k == 59) { g.debt[2] = max / 1e10; g.n = 3; g.routerLegs = 3; g.price[2] = max; }
            this.runCase(g);
        }
        // the 1e-9 monotone slack of _requireNotWorsened, u0 swept across the exact boundary
        for (uint256 k; k < 4; ++k) {
            for (uint256 d; d < 7; ++d) {
                g = _base(string.concat("slack-", vm.toString(k), "-", vm.toString(d)), 1);
                g.ltv = 1;
                g.debt[0] = [uint256(11_301_759), 999_999_937, 12_345_678_901, 1][k];
                g.physical = [uint256(1000), 219_423, 7, 3][k];
                g.posted = [uint256(0), 15_000, 0, 1][k];
                g.gross0 = [uint256(1000), 234_424, 9, 4][k];
                g.qAny[0] = k == 1;
                g.qDebt[0] = k == 1 ? 1_000 : 0;
                g.qColl[0] = k == 1 ? 77 : 0;
                g.quarantined = k == 1 && d % 2 == 0;
                uint256 u1 = g.debt[0] * g.scale[0] - (k == 1 && !g.quarantined ? 0 : 0);
                uint256 g1 = g.physical + g.posted + (g.quarantined ? g.qColl[0] : 0);
                uint256 g0 = g.gross0 + (g.quarantined ? g.qColl[0] : 0);
                uint256 b0 = u1 * g0 * 1e18 / (g1 * (1e18 + 1e9));
                int256 u0 = int256(b0) + int256(d) - 3;
                // the quarantined entry form adds frozen debt back before comparing; subtract it here so the sweep
                // still straddles the boundary
                if (g.quarantined) u0 -= int256(g.qDebt[0] * g.scale[0]);
                g.u0[0] = u0;
                this.runCase(g);
                if (k == 1) {
                    // the exit form subtracts frozen debt from u1
                    g.tag = string.concat(g.tag, "-exit");
                    uint256 u1x = u1 - g.qDebt[0] * g.scale[0];
                    uint256 bx = u1x * g.gross0 * 1e18 / ((g.physical + g.posted) * (1e18 + 1e9));
                    g.u0[0] = int256(bx) + int256(d) - 3;
                    this.runCase(g);
                }
            }
        }
        for (uint256 seed; seed < 420; ++seed) this.runCase(_random(seed));
        close();
    }

    function _random(uint256 seed) internal pure returns (G memory g) {
        g = _base(string.concat("random-", vm.toString(seed)), 1 + rnd(seed, 1) % 4);
        g.routerLegs = g.n;
        for (uint256 i; i < g.n; ++i) {
            g.scale[i] = [uint256(1), 1e12, 1e10, 1e2][rnd(seed, 10 + i) % 4];
            uint256 m = rnd(seed, 20 + i) % 5;
            g.debt[i] = m == 0 ? 0 : m == 4 ? rbits(seed, 30 + i, 256) : rbits(seed, 30 + i, 100);
            g.liquid[i] = rnd(seed, 40 + i) % 3 == 0 ? 0 : rbits(seed, 41 + i, 100);
            g.supplied[i] = rnd(seed, 50 + i) % 3 == 0 ? 0 : rbits(seed, 51 + i, 100);
            g.price[i] = rnd(seed, 60 + i) % 9 == 0 ? 0 : rbits(seed, 61 + i, 64) + 1;
            g.cross[i] = rnd(seed, 70 + i) % 9 == 0 ? 0 : rbits(seed, 71 + i, 64);
            g.okCross[i] = rnd(seed, 80 + i) % 10 != 0;
            g.okUsd[i] = rnd(seed, 81 + i) % 10 != 0;
            g.usd[i] = rnd(seed, 82 + i) % 10 == 0 ? 0 : rbits(seed, 83 + i, 70);
            g.qAny[i] = rnd(seed, 90 + i) % 5 == 0;
            g.qDebt[i] = g.qAny[i] ? rbits(seed, 91 + i, 100) : 0;
            g.qColl[i] = g.qAny[i] ? rbits(seed, 92 + i, 60) : 0;
            uint256 ua = rbits(seed, 93 + i, rnd(seed, 94 + i) % 2 == 0 ? 130 : 254);
            g.u0[i] = rnd(seed, 95 + i) % 2 == 0 ? int256(ua) : -int256(ua);
        }
        g.physical = rbits(seed, 2, rnd(seed, 3) % 5 == 0 ? 256 : 70);
        g.posted = rbits(seed, 4, 60);
        g.ltv = uint64(rnd(seed, 5) % 1e18);
        g.phi = uint64(rnd(seed, 6) % 1e18);
        g.eps = uint64(rnd(seed, 7) % 0.01e18);
        g.ltvArg = rnd(seed, 8) % 7 == 0 ? rbits(seed, 9, 256) : rnd(seed, 9) % 1.2e18;
        g.phiArg = rnd(seed, 11) % 7 == 0 ? rbits(seed, 12, 70) : rnd(seed, 12) % 1.1e18;
        g.lltvArg = rnd(seed, 13) % 1e18;
        g.headArg = rbits(seed, 14, rnd(seed, 15) % 4 == 0 ? 256 : 120);
        uint256 ua2 = rbits(seed, 16, 200);
        g.uArg = rnd(seed, 17) % 2 == 0 ? int256(ua2) : -int256(ua2);
        g.gross0 = rnd(seed, 18) % 6 == 0 ? 0 : rbits(seed, 19, 80);
        g.quarantined = rnd(seed, 21) % 4 == 0;
        if (rnd(seed, 22) % 25 == 0) g.routerLegs = g.n - 1;
    }
}
