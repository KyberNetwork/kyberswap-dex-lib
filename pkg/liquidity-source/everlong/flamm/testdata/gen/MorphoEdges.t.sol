// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import "./FinancingEdgesBase.sol";


/// @dev Edge generator for morpho.go / irm.go against the DEPLOYED Morpho Blue singleton and
///      AdaptiveCurveIrm on a Base fork: arbitrary market / position / rateAtTarget written straight into storage,
///      then one Blue transition (or one IRM read) per row, with the revert payload or the post-state recorded.
contract MorphoEdges is FinancingEdgesBase {
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
        bool dead;
        bool noIrm;
        uint8 op; // 0 accrue 1 supply 2 withdraw 3 borrow 4 repay 5 supplyCollateral 6 withdrawCollateral
        uint256 a;
        uint256 s;
        string tag;
    }

    VToken internal loanTok;
    VToken internal collTok;
    VOracle internal oracle;
    VMParams internal pIrm;
    VMParams internal pNoIrm;
    bytes32 internal idIrm;
    bytes32 internal idNoIrm;
    address internal actor = address(0xA11CE);
    uint256 internal T0;

    function _setup() internal {
        vm.createSelectFork(RPC, BLOCK);
        T0 = block.timestamp + 1000;
        loanTok = new VToken(6);
        collTok = new VToken(8);
        oracle = new VOracle();
        pIrm = VMParams(address(loanTok), address(collTok), address(oracle), IRM, 0.86e18);
        pNoIrm = VMParams(address(loanTok), address(collTok), address(oracle), address(0), 0.86e18);
        VMorpho(MORPHO).createMarket(pIrm);
        VMorpho(MORPHO).createMarket(pNoIrm);
        idIrm = keccak256(abi.encode(pIrm));
        idNoIrm = keccak256(abi.encode(pNoIrm));
        loanTok.mint(actor, 1 << 255);
        collTok.mint(actor, 1 << 255);
        loanTok.mint(MORPHO, 1 << 255);
        collTok.mint(MORPHO, 1 << 255);
        vm.startPrank(actor);
        loanTok.approve(MORPHO, type(uint256).max);
        collTok.approve(MORPHO, type(uint256).max);
        vm.stopPrank();
    }

    function _run(C memory c) internal {
        uint256 snap = vm.snapshotState();
        VMParams memory p = c.noIrm ? pNoIrm : pIrm;
        bytes32 id = c.noIrm ? idNoIrm : idIrm;
        uint256 lastUpdate = c.future ? T0 + 1 : T0 - c.elapsed;
        writeMarket(id, c.tsa, c.tss, c.tba, c.tbs, uint128(lastUpdate), c.fee);
        writePosition(id, actor, c.supplyShares, c.borrowShares, c.collateral);
        writeRat(id, c.rat);
        oracle.set(c.price, c.dead);
        vm.warp(T0);
        string memory pre = string.concat(
            kv("tag", string.concat('"', c.tag, '"')), ",", kv("now", q(T0)), ",", kv("noIrm", qb(c.noIrm)), ",",
            kv("lltv", q(0.86e18)), ",", kv("rat", qi(c.rat)), ",", kv("price", q(c.price)), ",", kv("dead", qb(c.dead)),
            ",", kv("op", q(c.op)), ",", kv("a", q(c.a)), ",", kv("s", q(c.s)), ",", kv("market", marketJson(id)), ",",
            kv("position", positionJson(id, actor))
        );
        bool ok;
        bytes memory ret;
        vm.prank(actor);
        if (c.op == 0) {
            (ok, ret) = MORPHO.call(abi.encodeCall(VMorpho.accrueInterest, (p)));
        } else if (c.op == 1) {
            (ok, ret) = MORPHO.call(abi.encodeCall(VMorpho.supply, (p, c.a, 0, actor, "")));
        } else if (c.op == 2) {
            (ok, ret) = MORPHO.call(abi.encodeCall(VMorpho.withdraw, (p, c.a, c.s, actor, actor)));
        } else if (c.op == 3) {
            (ok, ret) = MORPHO.call(abi.encodeCall(VMorpho.borrow, (p, c.a, 0, actor, actor)));
        } else if (c.op == 4) {
            (ok, ret) = MORPHO.call(abi.encodeCall(VMorpho.repay, (p, c.a, c.s, actor, "")));
        } else if (c.op == 5) {
            (ok, ret) = MORPHO.call(abi.encodeCall(VMorpho.supplyCollateral, (p, c.a, actor, "")));
        } else {
            (ok, ret) = MORPHO.call(abi.encodeCall(VMorpho.withdrawCollateral, (p, c.a, actor, actor)));
        }
        uint256 r0;
        uint256 r1;
        if (ok && ret.length >= 64) (r0, r1) = abi.decode(ret, (uint256, uint256));
        row(string.concat(
            "{", pre, ",", kv("ok", qb(ok)), ",", kv("revert", qh(ok ? bytes("") : ret)), ",", kv("r0", q(r0)), ",",
            kv("r1", q(r1)), ",", kv("postMarket", marketJson(id)), ",", kv("postPosition", positionJson(id, actor)), ",",
            kv("postRat", qi(VIrm(IRM).rateAtTarget(id))), "}"
        ));
        vm.revertToState(snap);
    }

    function _base(string memory tag) internal pure returns (C memory c) {
        c.tsa = 1e15;
        c.tss = 1e21;
        c.tba = 9e14;
        c.tbs = 8e20;
        c.elapsed = 3600;
        c.rat = INIT_RAT;
        c.price = 1e39;
        c.collateral = 1e8;
        c.borrowShares = 1e17;
        c.supplyShares = 1e20;
        c.tag = tag;
    }

    function test_morphoTransitions() public {
        _setup();
        open("mm_morpho_edges.json");
        C memory c;

        // ---------------- accrue
        uint64[7] memory els = [uint64(0), 1, 59, 3600, 86400, 365 days, 20 * 365 days];
        int256[6] memory rats = [int256(0), MIN_RAT, MIN_RAT + 1, INIT_RAT, MAX_RAT - 1, MAX_RAT];
        for (uint256 i; i < els.length; ++i) {
            for (uint256 j; j < rats.length; ++j) {
                c = _base("accrue-grid");
                c.elapsed = els[i];
                c.rat = rats[j];
                c.fee = uint128(j % 2 == 0 ? 0 : 0.25e18);
                _run(c);
            }
        }
        // zero borrow (IRM still adapts), zero supply, full utilisation, 1-wei totals
        c = _base("accrue-tba0");
        c.tba = 0;
        c.tbs = 0;
        _run(c);
        c = _base("accrue-tsa0");
        c.tsa = 0;
        c.tss = 0;
        c.tba = 0;
        c.tbs = 0;
        c.rat = MAX_RAT;
        _run(c);
        c = _base("accrue-full-util");
        c.tba = c.tsa;
        c.fee = 1;
        _run(c);
        c = _base("accrue-1wei");
        c.tsa = 1;
        c.tss = 1;
        c.tba = 1;
        c.tbs = 1;
        c.fee = 0.1e18;
        c.elapsed = 365 days;
        c.rat = MAX_RAT;
        _run(c);
        // uint128 overflow on the interest cast and on the checked add
        c = _base("accrue-interest-cast");
        c.tsa = type(uint128).max;
        c.tba = type(uint128).max;
        c.elapsed = 20 * 365 days;
        c.rat = MAX_RAT;
        _run(c);
        c = _base("accrue-add-overflow");
        c.tsa = type(uint128).max - 1000;
        c.tba = type(uint128).max / 2;
        c.elapsed = 3600;
        c.rat = MAX_RAT;
        _run(c);
        c = _base("accrue-tba-add-overflow-only");
        c.tsa = type(uint128).max;
        c.tba = type(uint128).max - 5;
        c.tbs = 1;
        c.elapsed = 60;
        _run(c);
        c = _base("accrue-fee-shares-overflow");
        c.tss = type(uint128).max - 1;
        c.fee = 0.25e18;
        c.elapsed = 365 days;
        c.rat = MAX_RAT;
        _run(c);
        c = _base("accrue-future-lastUpdate");
        c.future = true;
        _run(c);
        c = _base("accrue-future-lastUpdate-rat0");
        c.future = true;
        c.rat = 0;
        _run(c);
        c = _base("accrue-noirm");
        c.noIrm = true;
        _run(c);
        c = _base("accrue-noirm-future");
        c.noIrm = true;
        c.future = true;
        _run(c);
        // wExp thresholds: utilisation 0 (err = -1) and 1 (err = +1) around ln(1e-18) and the upper clip
        for (uint256 k; k < 8; ++k) {
            c = _base("accrue-wexp-low");
            c.tba = 0;
            c.tbs = 0;
            c.tsa = 1e18;
            c.elapsed = uint64(26140 + k); // -1585489599188 * e crosses LN_WEI near 26141
            c.rat = MAX_RAT;
            _run(c);
            c = _base("accrue-wexp-high");
            c.tba = 1e18;
            c.tsa = 1e18;
            c.tbs = 1e24;
            c.elapsed = uint64(59196 + k); // +1585489599188 * e crosses WEXP_UPPER_BOUND near 59199
            c.rat = MIN_RAT;
            _run(c);
        }

        // ---------------- supply
        uint256[12] memory sa = [
            uint256(0), 1, 999, 1e6, 7e14, uint256(type(uint128).max) - 1e15, uint256(type(uint128).max) - 1e15 + 1,
            type(uint128).max, uint256(type(uint128).max) + 1, 1 << 200, type(uint256).max / 1e21, type(uint256).max
        ];
        for (uint256 i; i < sa.length; ++i) {
            c = _base("supply-edge");
            c.elapsed = 0;
            c.a = sa[i];
            c.op = 1;
            _run(c);
        }
        c = _base("supply-position-overflow");
        c.elapsed = 0;
        c.supplyShares = type(uint256).max - 10;
        c.a = 1e6;
        c.op = 1;
        _run(c);
        c = _base("supply-shares-cast");
        c.elapsed = 0;
        c.tsa = 0;
        c.tss = type(uint128).max - 1e6;
        c.a = 1;
        c.op = 1;
        _run(c);
        c = _base("supply-shares-cast-2");
        c.elapsed = 0;
        c.tsa = 0;
        c.tss = type(uint128).max;
        c.a = 2;
        c.op = 1;
        _run(c);
        c = _base("supply-empty");
        c.tsa = 0;
        c.tss = 0;
        c.tba = 0;
        c.tbs = 0;
        c.a = 12345;
        c.op = 1;
        _run(c);
        c = _base("supply-accrue");
        c.a = 5e14;
        c.fee = 0.2e18;
        c.op = 1;
        _run(c);
        c = _base("supply-dead-oracle");
        c.dead = true;
        c.a = 5e14;
        c.op = 1;
        _run(c);

        // ---------------- withdraw
        for (uint256 i; i < 16; ++i) {
            c = _base("withdraw-edge");
            c.elapsed = 0;
            c.op = 2;
            c.tsa = 1000;
            c.tss = 1000e6;
            c.tba = 900;
            c.tbs = 900e6;
            c.supplyShares = 1000e6;
            if (i == 0) { c.a = 0; c.s = 0; }
            if (i == 1) { c.a = 1; c.s = 1; }
            if (i == 2) { c.a = 100; }
            if (i == 3) { c.a = 101; }
            if (i == 4) { c.s = 100e6 + 999; }
            if (i == 5) { c.s = 101e6; }
            if (i == 6) { c.s = 1; }
            if (i == 7) { c.a = 1000; }
            if (i == 8) { c.a = 1001; }
            if (i == 9) { c.s = 1000e6 + 1; }
            if (i == 10) { c.a = uint256(type(uint128).max) + 1; }
            if (i == 11) { c.s = uint256(type(uint128).max) + 1; c.supplyShares = type(uint256).max; }
            if (i == 12) { c.a = 50; c.elapsed = 86400; c.rat = MAX_RAT; c.fee = 0.25e18; }
            if (i == 13) { c.s = 1000e6; c.tba = 0; c.tbs = 0; }
            if (i == 14) { c.a = 7; c.tss = 1003e6 + 7; }
            if (i == 15) { c.s = 999; c.tss = 1003e6 + 7; }
            _run(c);
        }

        // ---------------- borrow (health / liquidity boundaries at price 1:1, lltv 0.86)
        for (uint256 i; i < 16; ++i) {
            c = _base("borrow-edge");
            c.elapsed = 0;
            c.op = 3;
            c.tsa = 86;
            c.tss = 86e6;
            c.tba = 0;
            c.tbs = 0;
            c.borrowShares = 0;
            c.supplyShares = 0;
            c.collateral = 100;
            c.price = 1e36;
            if (i == 0) c.a = 0;
            if (i == 1) c.a = 86;
            if (i == 2) c.a = 87;
            if (i == 3) { c.a = 86; c.tsa = 85; }
            if (i == 4) { c.a = 1; c.dead = true; }
            if (i == 5) { c.a = 1; c.price = 0; }
            if (i == 6) { c.a = uint256(type(uint128).max) + 1; }
            if (i == 7) { c.a = 1; c.tba = 1; c.tbs = type(uint128).max - 1e6; c.borrowShares = 1; }
            if (i == 8) { c.a = 3; c.tba = 2; c.tbs = 4e6; c.borrowShares = 4e6; c.tsa = 1000; c.collateral = 10; }
            if (i == 9) { c.a = 2; c.tba = 2; c.tbs = 4e6; c.borrowShares = 4e6; c.tsa = 1000; c.collateral = 10; }
            if (i == 10) { c.a = 1; c.collateral = 0; }
            if (i == 11) { c.a = 1; c.borrowShares = type(uint128).max; c.tbs = 5; c.tba = 5; c.tsa = 1e30; c.collateral = type(uint128).max; }
            if (i == 12) { c.a = 40; c.elapsed = 3600; c.tba = 40; c.tbs = 40e6; c.borrowShares = 40e6; c.tsa = 200; c.collateral = 200; c.rat = MAX_RAT; }
            if (i == 13) { c.a = 1; c.price = type(uint256).max; c.collateral = 2; }
            if (i == 14) { c.a = 1; c.collateral = type(uint128).max; c.price = type(uint256).max / type(uint128).max; }
            if (i == 15) { c.a = 1; c.collateral = type(uint128).max; c.price = type(uint256).max / type(uint128).max + 1; }
            _run(c);
        }

        // ---------------- repay
        for (uint256 i; i < 14; ++i) {
            c = _base("repay-edge");
            c.elapsed = 0;
            c.op = 4;
            c.tba = 1000;
            c.tbs = 1000e6;
            c.borrowShares = 500e6;
            if (i == 0) { c.a = 0; c.s = 0; }
            if (i == 1) { c.a = 1; c.s = 1; }
            if (i == 2) c.a = 500;
            if (i == 3) c.s = 500e6;
            if (i == 4) c.s = 500e6 + 1;
            if (i == 5) c.a = 501;
            if (i == 6) { c.tba = 1; c.tbs = 2e6; c.borrowShares = 2e6; c.s = 2e6; }
            if (i == 7) { c.a = 1; c.tba = 3; c.tbs = 5e6; c.borrowShares = 5e6; }
            if (i == 8) c.s = 1;
            if (i == 9) c.a = uint256(type(uint128).max) + 1;
            if (i == 10) c.s = uint256(type(uint128).max) + 1;
            if (i == 11) { c.s = 500e6; c.elapsed = 7 days; c.rat = MAX_RAT; c.fee = 0.05e18; }
            if (i == 12) { c.a = 999; c.borrowShares = 1000e6; c.dead = true; }
            if (i == 13) { c.a = type(uint256).max; }
            _run(c);
        }

        // ---------------- collateral
        for (uint256 i; i < 12; ++i) {
            c = _base("collateral-edge");
            c.elapsed = 0;
            c.op = i < 5 ? 5 : 6;
            c.price = 1e36;
            c.tba = 86;
            c.tbs = 86e6;
            c.borrowShares = 86e6;
            c.collateral = 100;
            if (i == 0) c.a = 0;
            if (i == 1) c.a = 1;
            if (i == 2) { c.collateral = type(uint128).max - 1; c.a = 1; }
            if (i == 3) { c.collateral = type(uint128).max; c.a = 1; }
            if (i == 4) c.a = uint256(type(uint128).max) + 1;
            if (i == 5) c.a = 0;
            if (i == 6) c.a = 1;
            if (i == 7) { c.collateral = 101; c.a = 1; }
            if (i == 8) { c.a = 101; }
            if (i == 9) { c.a = 50; c.borrowShares = 0; c.dead = true; }
            if (i == 10) { c.collateral = 101; c.a = 1; c.dead = true; }
            if (i == 11) { c.collateral = 101; c.a = 1; c.elapsed = 1; c.rat = MAX_RAT; }
            _run(c);
        }

        // ---------------- pseudo-random grid
        for (uint256 seed; seed < 420; ++seed) {
            _run(_random(seed));
        }
        close();
    }

    function _random(uint256 seed) internal view returns (C memory c) {
        c.tag = "random";
        uint256 mode = rnd(seed, 1) % 4; // 0 realistic, 1 small, 2 wide, 3 near-cap
        uint256 tsa;
        if (mode == 0) tsa = 1e9 + rnd(seed, 2) % 1e18;
        else if (mode == 1) tsa = rnd(seed, 2) % 5000;
        else if (mode == 2) tsa = rbits(seed, 2, 128);
        else tsa = type(uint128).max - rnd(seed, 2) % 1e20;
        c.tsa = uint128(tsa);
        uint256 util = rnd(seed, 3) % 1_000_001; // ppm
        uint256 tba = rnd(seed, 30) % 7 == 0 ? tsa : tsa * util / 1_000_000;
        c.tba = uint128(tba);
        uint256 shareMul = [uint256(1e6), 1e6 + 1, 999_999, 1e3, 1e9, 1][rnd(seed, 4) % 6];
        c.tss = uint128(_cap(tsa * shareMul + rnd(seed, 5) % 1000));
        c.tbs = uint128(_cap(tba * [uint256(1e6), 1e6 - 3, 1e6 + 17, 1e2, 1e8][rnd(seed, 6) % 5] + rnd(seed, 7) % 100));
        c.elapsed = uint64([uint256(0), 1, 12, 3600, 86400, 30 days, 365 days, rnd(seed, 8) % 1e7][rnd(seed, 9) % 8]);
        c.fee = uint128([uint256(0), 0, 1, 0.1e18, 0.25e18][rnd(seed, 10) % 5]);
        c.rat = rnd(seed, 11) % 5 == 0 ? int256(0) : MIN_RAT + int256(rnd(seed, 12) % uint256(MAX_RAT - MIN_RAT + 1));
        c.supplyShares = rnd(seed, 13) % 3 == 0 ? uint256(c.tss) : uint256(c.tss) * (rnd(seed, 14) % 1001) / 1000;
        c.borrowShares = uint128(rnd(seed, 15) % 3 == 0 ? uint256(c.tbs) : uint256(c.tbs) * (rnd(seed, 16) % 1001) / 1000);
        c.collateral = uint128(rbits(seed, 17, 90));
        c.price = [uint256(1e36), 1e39, 1e39 + rnd(seed, 18) % 1e39, rbits(seed, 18, 160), 0][rnd(seed, 19) % 5];
        c.dead = rnd(seed, 20) % 11 == 0;
        c.noIrm = rnd(seed, 21) % 13 == 0;
        c.op = uint8(rnd(seed, 22) % 7);
        uint256 kind = rnd(seed, 23) % 6;
        if (c.op == 1) {
            c.a = kind == 0 ? 0 : kind == 1 ? rbits(seed, 24, 130) : rnd(seed, 24) % (tsa / 3 + 2);
        } else if (c.op == 2 || c.op == 4) {
            uint256 posShares = c.op == 2 ? c.supplyShares : c.borrowShares;
            uint256 ta = c.op == 2 ? tsa : tba;
            if (kind == 0) c.s = posShares;
            else if (kind == 1) c.s = rnd(seed, 24) % (posShares + 2);
            else if (kind == 2) c.a = rnd(seed, 24) % (ta + 2);
            else if (kind == 3) c.a = rbits(seed, 24, 132);
            else if (kind == 4) { c.a = rnd(seed, 24) % 3; c.s = rnd(seed, 25) % 3; }
            else c.a = ta - ta / 3;
        } else if (c.op == 3) {
            uint256 room = tsa - tba;
            c.a = kind == 0 ? room : kind == 1 ? room + 1 : kind == 2 ? rbits(seed, 24, 130) : rnd(seed, 24) % (room + 2);
            // make the health side plausible
            if (rnd(seed, 26) % 2 == 0 && c.price != 0) {
                uint256 need = (tba + c.a) * 2 * 1e36 / c.price + 1;
                if (need < type(uint128).max) c.collateral = uint128(need * (50 + rnd(seed, 27) % 100) / 100);
            }
        } else if (c.op == 5) {
            c.a = kind == 0 ? 0 : rbits(seed, 24, 130);
        } else if (c.op == 6) {
            c.a = kind == 0 ? c.collateral : kind == 1 ? uint256(c.collateral) + 1 : rnd(seed, 24) % (uint256(c.collateral) + 2);
        }
    }

    function _cap(uint256 x) internal pure returns (uint256) {
        return x > type(uint128).max ? type(uint128).max : x;
    }

    // ------------------------------------------------------------------ IRM reads
    function test_irm() public {
        _setup();
        open("mm_irm_edges.json");
        // utilisation around the zero-speed band, both sides of target, beyond 1, tiny / empty supply
        uint256[] memory utils = new uint256[](0);
        uint256 n;
        uint256[64] memory u;
        uint256[16] memory d = [uint256(0), 1, 63071, 63072, 63073, 567647, 567648, 567649, 630719, 630720, 630721, 1e6, 1e12, 5e16, 1e17, 9e17];
        for (uint256 i; i < d.length; ++i) {
            u[n++] = 0.9e18 + d[i];
            if (d[i] <= 0.9e18) u[n++] = 0.9e18 - d[i];
        }
        u[n++] = 1e18 + 1;
        u[n++] = 2e18;
        u[n++] = 1;
        utils = new uint256[](n);
        for (uint256 i; i < n; ++i) utils[i] = u[i];
        int256[7] memory rats = [int256(0), MIN_RAT, MIN_RAT + 1, INIT_RAT, 1_000_000_000, MAX_RAT - 1, MAX_RAT];
        uint256[9] memory els = [uint256(0), 1, 2, 3, 3599, 86400, 26141, 59199, 400 days];
        uint256 rows;
        for (uint256 i; i < utils.length; ++i) {
            for (uint256 j; j < rats.length; ++j) {
                uint256 e = els[(i * 7 + j * 3) % els.length];
                _irmRow(1e18, uint128(utils[i]), rats[j], T0 - e, "grid");
                ++rows;
            }
        }
        for (uint256 j; j < rats.length; ++j) {
            _irmRow(0, 0, rats[j], T0 - 3600, "tsa0");
            _irmRow(0, 5, rats[j], T0 - 3600, "tsa0-tba5");
            _irmRow(1, type(uint128).max, rats[j], T0 - 3600, "tsa1-tbamax");
            _irmRow(type(uint128).max, type(uint128).max, rats[j], T0 - 86400, "max-full");
            _irmRow(1e18, 5e17, rats[j], T0 + 1, "future");
            _irmRow(1e18, 5e17, rats[j], 0, "lastUpdate0");
            _irmRow(1e18, 1e18, rats[j], 1, "lastUpdate1-full");
            _irmRow(1e18, 0, rats[j], 1, "lastUpdate1-empty");
        }
        for (uint256 seed; seed < 360; ++seed) {
            uint256 tsa = rnd(seed, 100) % 3 == 0 ? rbits(seed, 101, 128) : 1e6 + rnd(seed, 101) % 1e20;
            uint256 tba = rnd(seed, 102) % 9 == 0 ? rbits(seed, 103, 128) : tsa * (rnd(seed, 103) % 1_000_001) / 1_000_000;
            int256 rat = rnd(seed, 104) % 8 == 0 ? int256(0) : MIN_RAT + int256(rnd(seed, 105) % uint256(MAX_RAT - MIN_RAT + 1));
            uint256 e = [uint256(0), 1, 13, 3600, 86400, 7 days, 180 days, rnd(seed, 106) % 1e8][rnd(seed, 107) % 8];
            _irmRow(uint128(tsa), uint128(tba), rat, T0 - e, "random");
        }
        close();
    }

    function _irmRow(uint128 tsa, uint128 tba, int256 rat, uint256 lastUpdate, string memory tag) internal {
        uint256 snap = vm.snapshotState();
        writeRat(idIrm, rat);
        vm.warp(T0);
        VMarketState memory m = VMarketState(tsa, tsa, tba, tba, uint128(lastUpdate), 0);
        (bool ok1, bytes memory r1) = IRM.staticcall(abi.encodeCall(VIrmFull.borrowRateView, (pIrm, m)));
        vm.prank(MORPHO);
        (bool ok2, bytes memory r2) = IRM.call(abi.encodeCall(VIrmFull.borrowRate, (pIrm, m)));
        int256 end = VIrm(IRM).rateAtTarget(idIrm);
        row(string.concat(
            "{", kv("tag", string.concat('"', tag, '"')), ",", kv("now", q(T0)), ",", kv("tsa", q(tsa)), ",",
            kv("tba", q(tba)), ",", kv("lastUpdate", q(lastUpdate)), ",", kv("rat", qi(rat)), ",", kv("viewOk", qb(ok1)),
            ",", kv("view", ok1 ? q(abi.decode(r1, (uint256))) : qh(r1)), ",", kv("mutOk", qb(ok2)), ",",
            kv("mut", ok2 ? q(abi.decode(r2, (uint256))) : qh(r2)), ",", kv("end", qi(end)), "}"
        ));
        vm.revertToState(snap);
    }
}
