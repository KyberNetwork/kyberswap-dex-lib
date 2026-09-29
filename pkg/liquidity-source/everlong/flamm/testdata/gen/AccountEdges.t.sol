// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import "./FinancingEdgesBase.sol";

interface VAccount {
    function tryPosition(bytes32) external view returns (bool, uint256, uint256, uint256, uint256);
    function debtOf(bytes32) external view returns (uint256);
    function suppliedOf(bytes32) external view returns (uint256);
    function supplySharesToAssets(bytes32, uint256) external view returns (uint256);
    function freeLiquidity(bytes32) external view returns (uint256);
    function borrowRateAfter(bytes32, uint256, uint256) external view returns (bool, uint256);
    function oraclePrice(bytes32) external view returns (bool, uint256);
    function lltv(bytes32) external view returns (uint64);
    function supplyCollateral(bytes32, uint256) external;
    function withdrawCollateral(bytes32, uint256, address) external;
    function borrow(bytes32, uint256, address) external;
    function repay(bytes32, uint256) external returns (uint256);
    function supply(bytes32, uint256) external returns (uint256);
    function withdraw(bytes32, uint256, uint256, address) external returns (uint256, uint256);
}


/// @dev Edge generator for account.go against the DEPLOYED MorphoBlueAccount (0x6760…6c48) on its
///      live Morpho market: Blue's market / position and the IRM's rateAtTarget written into storage, the market
///      oracle and the IRM mocked alive / reverting, then every view and one Router-pranked mutator per row.
contract AccountEdges is FinancingEdgesBase {
    struct C {
        uint128 tsa;
        uint128 tss;
        uint128 tba;
        uint128 tbs;
        uint64 elapsed;
        bool future;
        uint128 fee;
        uint256 supplyShares;
        uint128 borrowShares;
        uint128 collateral;
        int256 rat;
        uint256 price;
        bool oracleDead;
        bool irmDead;
        uint8 op; // 0 none 1 repay 2 withdraw 3 borrow 4 supply 5 supplyCollateral 6 withdrawCollateral
        uint256 a;
        uint256 s;
        string tag;
    }

    uint256 internal T0;

    function _setup() internal {
        vm.createSelectFork(RPC, BLOCK);
        T0 = block.timestamp + 5000;
    }

    function _deltas(C memory c) internal pure returns (uint256[2][] memory d) {
        d = new uint256[2][](14);
        uint256 max128 = type(uint128).max;
        d[0] = [uint256(0), 0];
        d[1] = [uint256(1), 0];
        d[2] = [uint256(0), 1];
        d[3] = [uint256(0), c.tsa == 0 ? 0 : uint256(c.tsa) - 1];
        d[4] = [uint256(0), c.tsa];
        d[5] = [uint256(0), uint256(c.tsa) + 1];
        d[6] = [max128 - c.tba, 0];
        d[7] = [max128 - c.tba + 1, 0];
        d[8] = [type(uint256).max - c.tba, 0];
        d[9] = [c.tba == 0 ? 0 : type(uint256).max - c.tba + 1, 0];
        d[10] = [uint256(c.tsa) - (c.tba < c.tsa ? c.tba : c.tsa), c.tsa / 2];
        d[11] = [uint256(c.tsa) / 3, uint256(c.tsa) / 3];
        d[12] = [uint256(12345), c.tsa == 0 ? 0 : uint256(c.tsa) - 1];
        d[13] = [type(uint256).max, uint256(1)];
    }

    function runRow(C memory c) external {
        _run(c);
    }

    function _run(C memory c) internal {
        uint256 snap = vm.snapshotState();
        uint256 lastUpdate = c.future ? T0 + 7 : T0 - c.elapsed;
        writeMarket(LIVE_ID, c.tsa, c.tss, c.tba, c.tbs, uint128(lastUpdate), c.fee);
        writePosition(LIVE_ID, ACCOUNT, c.supplyShares, c.borrowShares, c.collateral);
        writeRat(LIVE_ID, c.rat);
        vm.clearMockedCalls();
        if (c.oracleDead) vm.mockCallRevert(LIVE_ORACLE, abi.encodeWithSignature("price()"), abi.encodeWithSignature("Error(string)", "dead"));
        else vm.mockCall(LIVE_ORACLE, abi.encodeWithSignature("price()"), abi.encode(c.price));
        if (c.irmDead) {
            vm.mockCallRevert(IRM, abi.encodePacked(VIrmFull.borrowRateView.selector), abi.encodeWithSignature("Error(string)", "irm dead"));
            vm.mockCallRevert(IRM, abi.encodePacked(VIrmFull.borrowRate.selector), abi.encodeWithSignature("Error(string)", "irm dead"));
        }
        vm.warp(T0);
        string memory head = string.concat(
            "{", kv("tag", string.concat('"', c.tag, '"')), ",", kv("now", q(T0)), ",", kv("lltv", q(VAccount(ACCOUNT).lltv(LIVE_ID))),
            ",", kv("rat", qi(c.rat)), ",", kv("price", q(c.price)), ",", kv("oracleDead", qb(c.oracleDead)), ",",
            kv("irmDead", qb(c.irmDead)), ",", kv("market", marketJson(LIVE_ID)), ",", kv("position", positionJson(LIVE_ID, ACCOUNT))
        );
        string memory views = _views(c);
        string memory mut = _mutate(c);
        row(string.concat(head, ",", views, ",", mut, "}"));
        vm.revertToState(snap);
    }

    function _res(bool ok, bytes memory ret) internal pure returns (string memory) {
        if (!ok) return string.concat("{", kv("ok", "false"), ",", kv("revert", qh(ret)), "}");
        uint256 v = ret.length >= 32 ? abi.decode(ret, (uint256)) : 0;
        return string.concat("{", kv("ok", "true"), ",", kv("v", q(v)), "}");
    }

    function _views(C memory c) internal view returns (string memory s) {
        (bool ok, bytes memory ret) = ACCOUNT.staticcall(abi.encodeCall(VAccount.tryPosition, (LIVE_ID)));
        if (ok) {
            (bool r, uint256 a, uint256 b, uint256 d0, uint256 e) = abi.decode(ret, (bool, uint256, uint256, uint256, uint256));
            s = string.concat(kv("tryPosition", string.concat("{", kv("ok", "true"), ",", kv("readable", qb(r)), ",",
                kv("collateral", q(a)), ",", kv("supplyShares", q(b)), ",", kv("supplied", q(d0)), ",", kv("debt", q(e)), "}")));
        } else {
            s = kv("tryPosition", _res(ok, ret));
        }
        (ok, ret) = ACCOUNT.staticcall(abi.encodeCall(VAccount.debtOf, (LIVE_ID)));
        s = string.concat(s, ",", kv("debtOf", _res(ok, ret)));
        (ok, ret) = ACCOUNT.staticcall(abi.encodeCall(VAccount.suppliedOf, (LIVE_ID)));
        s = string.concat(s, ",", kv("suppliedOf", _res(ok, ret)));
        (ok, ret) = ACCOUNT.staticcall(abi.encodeCall(VAccount.freeLiquidity, (LIVE_ID)));
        s = string.concat(s, ",", kv("freeLiquidity", _res(ok, ret)));
        uint256[6] memory ks = [uint256(0), 1, c.supplyShares, c.supplyShares / 2 + 1, type(uint256).max, uint256(1) << 200];
        string memory sh = "[";
        for (uint256 i; i < ks.length; ++i) {
            (ok, ret) = ACCOUNT.staticcall(abi.encodeCall(VAccount.supplySharesToAssets, (LIVE_ID, ks[i])));
            sh = string.concat(sh, i == 0 ? "" : ",", "{", kv("shares", q(ks[i])), ",", kv("res", _res(ok, ret)), "}");
        }
        s = string.concat(s, ",", kv("sharesToAssets", string.concat(sh, "]")));
        uint256[2][] memory d = _deltas(c);
        string memory rr = "[";
        for (uint256 i; i < d.length; ++i) {
            (ok, ret) = ACCOUNT.staticcall(abi.encodeCall(VAccount.borrowRateAfter, (LIVE_ID, d[i][0], d[i][1])));
            string memory res;
            if (ok) {
                (bool rok, uint256 rate) = abi.decode(ret, (bool, uint256));
                res = string.concat("{", kv("ok", "true"), ",", kv("rok", qb(rok)), ",", kv("rate", q(rate)), "}");
            } else {
                res = _res(ok, ret);
            }
            rr = string.concat(rr, i == 0 ? "" : ",", "{", kv("dB", q(d[i][0])), ",", kv("dS", q(d[i][1])), ",", kv("res", res), "}");
        }
        s = string.concat(s, ",", kv("rateAfter", string.concat(rr, "]")));
        (ok, ret) = ACCOUNT.staticcall(abi.encodeCall(VAccount.oraclePrice, (LIVE_ID)));
        (bool pok, uint256 p) = abi.decode(ret, (bool, uint256));
        s = string.concat(s, ",", kv("oraclePrice", string.concat("{", kv("ok", qb(pok)), ",", kv("v", q(p)), "}")));
    }

    function _mutate(C memory c) internal returns (string memory) {
        bool ok;
        bytes memory ret;
        uint256 r0;
        uint256 r1;
        if (c.op == 1) {
            deal(USDC, ACCOUNT, c.a > 1 << 200 ? 1 << 200 : c.a);
            vm.prank(ROUTER);
            (ok, ret) = ACCOUNT.call(abi.encodeCall(VAccount.repay, (LIVE_ID, c.a)));
        } else if (c.op == 2) {
            deal(USDC, MORPHO, 1 << 200);
            vm.prank(ROUTER);
            (ok, ret) = ACCOUNT.call(abi.encodeCall(VAccount.withdraw, (LIVE_ID, c.a, c.s, POOL)));
            if (ok) (r0, r1) = abi.decode(ret, (uint256, uint256));
        } else if (c.op == 3) {
            deal(USDC, MORPHO, 1 << 200);
            vm.prank(ROUTER);
            (ok, ret) = ACCOUNT.call(abi.encodeCall(VAccount.borrow, (LIVE_ID, c.a, POOL)));
        } else if (c.op == 4) {
            deal(USDC, ACCOUNT, c.a > 1 << 200 ? 1 << 200 : c.a);
            vm.prank(ROUTER);
            (ok, ret) = ACCOUNT.call(abi.encodeCall(VAccount.supply, (LIVE_ID, c.a)));
        } else if (c.op == 5) {
            deal(CBBTC, ACCOUNT, c.a > 1 << 200 ? 1 << 200 : c.a);
            vm.prank(ROUTER);
            (ok, ret) = ACCOUNT.call(abi.encodeCall(VAccount.supplyCollateral, (LIVE_ID, c.a)));
        } else if (c.op == 6) {
            deal(CBBTC, MORPHO, 1 << 200);
            vm.prank(ROUTER);
            (ok, ret) = ACCOUNT.call(abi.encodeCall(VAccount.withdrawCollateral, (LIVE_ID, c.a, POOL)));
        } else {
            return string.concat(kv("op", q(0)));
        }
        if (ok && c.op != 2 && ret.length >= 32) r0 = abi.decode(ret, (uint256));
        return string.concat(
            kv("op", q(c.op)), ",", kv("a", q(c.a)), ",", kv("s", q(c.s)), ",", kv("ok", qb(ok)), ",",
            kv("revert", qh(ok ? bytes("") : ret)), ",", kv("r0", q(r0)), ",", kv("r1", q(r1)), ",",
            kv("postMarket", marketJson(LIVE_ID)), ",", kv("postPosition", positionJson(LIVE_ID, ACCOUNT)), ",",
            kv("postRat", qi(VIrm(IRM).rateAtTarget(LIVE_ID)))
        );
    }

    function _base(string memory tag) internal pure returns (C memory c) {
        c.tsa = 1567884338546841;
        c.tss = 1418373654349324291165;
        c.tba = 1410885862270200;
        c.tbs = 1260337866908047722881;
        c.elapsed = 600;
        c.rat = 1476127382;
        c.price = 1e39;
        c.collateral = 26221;
        c.borrowShares = 10096228333889;
        c.supplyShares = 0;
        c.tag = tag;
    }

    function test_account() public {
        _setup();
        open("mm_account_edges.json");
        C memory c;
        // grace boundary with and without debt, both IRM states
        uint64[9] memory els = [uint64(0), 1, 1800, 3599, 3600, 3601, 7200, 30 days, 365 days];
        for (uint256 i; i < els.length; ++i) {
            for (uint256 k; k < 4; ++k) {
                c = _base("grace");
                c.elapsed = els[i];
                c.irmDead = k % 2 == 1;
                c.fee = k >= 2 ? 0.1e18 : 0;
                c.supplyShares = k >= 2 ? 5e17 : 0;
                this.runRow(c);
            }
            c = _base("grace-nodebt");
            c.elapsed = els[i];
            c.irmDead = true;
            c.tba = 0;
            c.tbs = 0;
            c.borrowShares = 0;
            this.runRow(c);
        }
        c = _base("future");
        c.future = true;
        this.runRow(c);
        c = _base("future-repay");
        c.future = true;
        c.op = 1;
        c.a = 1000;
        this.runRow(c);
        c = _base("rat0");
        c.rat = 0;
        this.runRow(c);
        c = _base("tsa0");
        c.tsa = 0;
        c.tss = 0;
        c.tba = 0;
        c.tbs = 0;
        c.borrowShares = 0;
        this.runRow(c);
        c = _base("tsa1-tbamax");
        c.tsa = 1;
        c.tba = type(uint128).max;
        c.tbs = type(uint128).max;
        c.borrowShares = type(uint128).max;
        c.supplyShares = type(uint256).max;
        c.tss = type(uint128).max;
        this.runRow(c);
        c = _base("max-shares");
        c.tsa = type(uint128).max;
        c.tss = 1;
        c.supplyShares = type(uint256).max;
        c.tba = type(uint128).max - 1;
        c.tbs = 1;
        c.borrowShares = type(uint128).max;
        c.elapsed = 365 days;
        c.fee = 0.25e18;
        this.runRow(c);

        // ---------------- repay
        for (uint256 i; i < 14; ++i) {
            c = _base("repay");
            c.op = 1;
            c.elapsed = i < 8 ? 0 : 3600;
            if (i == 0) c.a = 0;
            if (i == 1) c.a = 1;
            if (i == 2) c.a = 11295839999999; // below the accrued debt
            if (i == 3) { c.tba = 1000; c.tbs = 1000e6; c.borrowShares = 400e6; c.a = 399; }
            if (i == 4) { c.tba = 1000; c.tbs = 1000e6; c.borrowShares = 400e6; c.a = 400; }
            if (i == 5) { c.tba = 1000; c.tbs = 1000e6; c.borrowShares = 400e6; c.a = 401; }
            if (i == 6) { c.tba = 1e18; c.tbs = 1e6; c.borrowShares = 1e6; c.a = 1; }
            if (i == 7) { c.tba = 1; c.tbs = 2e6; c.borrowShares = 2e6; c.a = 5; }
            if (i == 8) { c.borrowShares = 0; c.a = 100; }
            if (i == 9) { c.irmDead = true; c.a = 100; c.elapsed = 10; }
            if (i == 10) { c.a = type(uint256).max; }
            if (i == 11) { c.a = 1e30; c.fee = 0.25e18; c.elapsed = 30 days; }
            if (i == 12) { c.oracleDead = true; c.a = 1e30; }
            if (i == 13) { c.tba = 999; c.tbs = 1000e6; c.borrowShares = 1000e6; c.a = 999; c.elapsed = 0; }
            this.runRow(c);
        }
        // ---------------- withdraw
        for (uint256 i; i < 12; ++i) {
            c = _base("withdraw");
            c.op = 2;
            c.elapsed = i < 6 ? 0 : 1200;
            c.supplyShares = 1e20;
            if (i == 0) { c.a = 0; c.s = 0; }
            if (i == 1) { c.a = 1; c.s = 1; }
            if (i == 2) { c.supplyShares = 0; c.a = 5; }
            if (i == 3) { c.a = 1; }
            if (i == 4) { c.s = 1e20; }
            if (i == 5) { c.s = 1e20 + 1; }
            if (i == 6) { c.a = 110536; }
            if (i == 7) { c.irmDead = true; c.a = 5; }
            if (i == 8) { c.supplyShares = 1e24; c.s = 1e24; c.tsa = c.tba + 1; }
            if (i == 9) { c.supplyShares = 1e24; c.a = uint256(c.tsa) - c.tba; }
            if (i == 10) { c.supplyShares = 1e24; c.a = uint256(c.tsa) - c.tba + 1000; }
            if (i == 11) { c.a = uint256(type(uint128).max) + 1; }
            this.runRow(c);
        }
        // ---------------- borrow (price 1:1 at the account's 0.86 lltv)
        for (uint256 i; i < 10; ++i) {
            c = _base("borrow");
            c.op = 3;
            c.elapsed = 0;
            c.price = 1e36;
            c.tba = 0;
            c.tbs = 0;
            c.borrowShares = 0;
            c.collateral = 1000;
            if (i == 0) c.a = 0;
            if (i == 1) c.a = 860;
            if (i == 2) c.a = 861;
            if (i == 3) { c.a = 860; c.tsa = 859; }
            if (i == 4) { c.a = 10; c.oracleDead = true; }
            if (i == 5) { c.a = 10; c.price = 0; }
            if (i == 6) { c.a = 10; c.irmDead = true; c.elapsed = 5; }
            if (i == 7) { c.a = 10; c.irmDead = true; c.elapsed = 0; }
            if (i == 8) { c.a = 400; c.tba = 400; c.tbs = 400e6; c.borrowShares = 400e6; c.elapsed = 86400; }
            if (i == 9) { c.a = uint256(type(uint128).max) + 1; }
            this.runRow(c);
        }
        // ---------------- supply / collateral
        for (uint256 i; i < 12; ++i) {
            c = _base("supply-coll");
            c.elapsed = i % 3 == 0 ? 0 : 900;
            c.op = i < 4 ? 4 : i < 8 ? 5 : 6;
            c.price = 1e39;
            if (i == 0) c.a = 0;
            if (i == 1) c.a = 1;
            if (i == 2) c.a = 1e12;
            if (i == 3) { c.a = 1e12; c.irmDead = true; }
            if (i == 4) c.a = 0;
            if (i == 5) c.a = 1;
            if (i == 6) { c.a = 1; c.collateral = type(uint128).max; }
            if (i == 7) { c.a = 1e8; c.oracleDead = true; }
            if (i == 8) c.a = 0;
            if (i == 9) c.a = 1000;
            if (i == 10) { c.a = 26221; }
            if (i == 11) { c.a = 10; c.borrowShares = 0; c.oracleDead = true; c.irmDead = true; }
            this.runRow(c);
        }
        for (uint256 seed; seed < 360; ++seed) {
            this.runRow(_random(seed));
        }
        close();
    }

    function _random(uint256 seed) internal view returns (C memory c) {
        c.tag = "random";
        uint256 mode = rnd(seed, 1) % 4;
        uint256 tsa;
        if (mode == 0) tsa = 1e9 + rnd(seed, 2) % 1e18;
        else if (mode == 1) tsa = rnd(seed, 2) % 5000;
        else if (mode == 2) tsa = rbits(seed, 2, 128);
        else tsa = type(uint128).max - rnd(seed, 2) % 1e20;
        c.tsa = uint128(tsa);
        uint256 tba = rnd(seed, 30) % 7 == 0 ? tsa : tsa * (rnd(seed, 3) % 1_000_001) / 1_000_000;
        c.tba = uint128(tba);
        c.tss = uint128(_cap(tsa * [uint256(1e6), 1e6 + 1, 999_999, 1e3, 1e9, 1][rnd(seed, 4) % 6] + rnd(seed, 5) % 1000));
        c.tbs = uint128(_cap(tba * [uint256(1e6), 1e6 - 3, 1e6 + 17, 1e2, 1e8][rnd(seed, 6) % 5] + rnd(seed, 7) % 100));
        c.elapsed = uint64([uint256(0), 1, 12, 3599, 3600, 3601, 86400, 365 days, rnd(seed, 8) % 1e7][rnd(seed, 9) % 9]);
        c.future = rnd(seed, 31) % 25 == 0;
        c.fee = uint128([uint256(0), 0, 1, 0.1e18, 0.25e18][rnd(seed, 10) % 5]);
        c.rat = rnd(seed, 11) % 5 == 0 ? int256(0) : MIN_RAT + int256(rnd(seed, 12) % uint256(MAX_RAT - MIN_RAT + 1));
        c.supplyShares = rnd(seed, 13) % 3 == 0 ? uint256(c.tss) : uint256(c.tss) * (rnd(seed, 14) % 1001) / 1000;
        c.borrowShares = uint128(rnd(seed, 15) % 3 == 0 ? uint256(c.tbs) : uint256(c.tbs) * (rnd(seed, 16) % 1001) / 1000);
        c.collateral = uint128(rbits(seed, 17, 90));
        c.price = [uint256(1e36), 1e39, 1e39 + rnd(seed, 18) % 1e39, rbits(seed, 18, 160), 0][rnd(seed, 19) % 5];
        c.oracleDead = rnd(seed, 20) % 11 == 0;
        c.irmDead = rnd(seed, 21) % 6 == 0;
        c.op = uint8(rnd(seed, 22) % 7);
        uint256 kind = rnd(seed, 23) % 6;
        if (c.op == 1) {
            c.a = kind == 0 ? 0 : kind == 1 ? rbits(seed, 24, 130) : rnd(seed, 24) % (tba + 2);
        } else if (c.op == 2) {
            if (kind == 0) c.s = c.supplyShares;
            else if (kind == 1) c.s = rnd(seed, 24) % (c.supplyShares + 2);
            else if (kind == 2) c.a = rnd(seed, 24) % (tsa + 2);
            else if (kind == 3) c.a = rbits(seed, 24, 132);
            else if (kind == 4) { c.a = rnd(seed, 24) % 3; c.s = rnd(seed, 25) % 3; }
            else c.a = tsa - tsa / 3;
        } else if (c.op == 3) {
            uint256 room = tsa - tba;
            c.a = kind == 0 ? room : kind == 1 ? room + 1 : kind == 2 ? rbits(seed, 24, 130) : rnd(seed, 24) % (room + 2);
            if (rnd(seed, 26) % 2 == 0 && c.price != 0) {
                uint256 need = (tba + c.a) * 2 * 1e36 / c.price + 1;
                if (need < type(uint128).max) c.collateral = uint128(need * (50 + rnd(seed, 27) % 100) / 100);
            }
        } else if (c.op == 4 || c.op == 5) {
            c.a = kind == 0 ? 0 : kind < 3 ? rbits(seed, 24, 130) : rnd(seed, 24) % 1e18;
        } else if (c.op == 6) {
            c.a = kind == 0 ? c.collateral : kind == 1 ? uint256(c.collateral) + 1 : rnd(seed, 24) % (uint256(c.collateral) + 2);
        }
    }

    function _cap(uint256 x) internal pure returns (uint256) {
        return x > type(uint128).max ? type(uint128).max : x;
    }
}
