// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import "./RouterSettlementEdgesBase.sol";
import {IERC20} from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import {IMMRouter} from "src/interfaces/core/mm/IMMRouter.sol";

/// @dev Edge generator for morpho.go / irm.go / account.go against the DEPLOYED Morpho Blue,
///      AdaptiveCurveIrm and MorphoBlueAccount on a Base fork. Two created USDC/cbBTC markets registered on the
///      live account through the live Router (a 77% market on the AdaptiveCurveIrm and a 91.5% market with no rate
///      model), states written into storage: realistic-magnitude seeded grids plus the 1-wei neighbours of the
///      IRM's zero-speed point, wExp's clip thresholds, the account's grace, the rate-quote fail-closed and
///      saturation edges, Blue's health / liquidity / zero-floor edges, the repay ExactDelta and a zero-answer oracle.
contract MorphoMarketEdges is RouterSettlementEdgesBase {
    address internal constant USER = address(0xB1BE52);
    uint256 internal constant ORACLE0 = 788432322552395 * 1e24;
    uint256 internal constant ADJ = uint256(50e18) / 365 days;

    struct St {
        uint128 tsa;
        uint128 tss;
        uint128 tba;
        uint128 tbs;
        uint64 elapsed;
        uint128 fee;
        uint256 sup;
        uint128 bor;
        uint128 coll;
        int256 rat;
        uint256 oracleP;
        uint8 oracleMode;
        bool irmDead;
        bool noIrm;
    }

    R2Oracle[3] internal oracles;
    R2Params[3] internal params;
    bytes32[3] internal ids;
    uint256 internal T0;
    uint256 internal baseSnap;

    function _setup() internal {
        vm.createSelectFork(RPC, BLOCK);
        T0 = block.timestamp + 7 days;
        for (uint256 i = 1; i < 3; ++i) {
            oracles[i] = new R2Oracle();
            oracles[i].set(ORACLE0, 0);
        }
        params[1] = R2Params(USDC, CBBTC, address(oracles[1]), IRM, 0.77e18);
        params[2] = R2Params(USDC, CBBTC, address(oracles[2]), address(0), 0.915e18);
        for (uint256 i = 1; i < 3; ++i) {
            ids[i] = keccak256(abi.encode(params[i]));
            R2Morpho(MORPHO).createMarket(params[i]);
        }
        vm.startPrank(POOL);
        IMMRouter(ROUTER).addVenue(0, 0, abi.encode(params[1]), true, true, 788432322552395);
        IMMRouter(ROUTER).addVenue(0, 0, abi.encode(params[2]), true, true, 788432322552395);
        vm.stopPrank();
        deal(USDC, MORPHO, 1e30);
        deal(CBBTC, MORPHO, 1e30);
        deal(USDC, USER, 1e30);
        deal(CBBTC, USER, 1e30);
        vm.startPrank(USER);
        IERC20(USDC).approve(MORPHO, type(uint256).max);
        IERC20(CBBTC).approve(MORPHO, type(uint256).max);
        vm.stopPrank();
        deal(CBBTC, ACCOUNT, 1e30);
        baseSnap = vm.snapshotState();
    }

    function _apply(St memory s, address who) internal returns (uint256 m) {
        vm.revertToState(baseSnap);
        vm.clearMockedCalls();
        m = s.noIrm ? 2 : 1;
        vm.warp(T0);
        writeMarket(ids[m], s.tsa, s.tss, s.tba, s.tbs, uint128(T0 - s.elapsed), s.fee);
        writePosition(ids[m], who, s.sup, s.bor, s.coll);
        if (!s.noIrm) writeRat(ids[m], s.rat);
        oracles[m].set(s.oracleP, s.oracleMode);
        if (s.irmDead) {
            vm.mockCallRevert(IRM, abi.encodeWithSelector(R2Irm.borrowRateView.selector, params[m]), "irm down");
            vm.mockCallRevert(IRM, abi.encodeWithSelector(R2Irm.borrowRate.selector, params[m]), "irm down");
        }
    }

    function _stJson(St memory s) internal pure returns (string memory o) {
        o = "{";
        o = add(o, "tsa", q(s.tsa));
        o = add(o, "tss", q(s.tss));
        o = add(o, "tba", q(s.tba));
        o = add(o, "tbs", q(s.tbs));
        o = add(o, "elapsed", q(s.elapsed));
        o = add(o, "fee", q(s.fee));
        o = add(o, "sup", q(s.sup));
        o = add(o, "bor", q(s.bor));
        o = add(o, "coll", q(s.coll));
        o = add(o, "rat", qi(s.rat));
        o = add(o, "oracleP", q(s.oracleP));
        o = add(o, "oracleMode", q(s.oracleMode));
        o = add(o, "irmDead", qb(s.irmDead));
        o = end(add(o, "noIrm", qb(s.noIrm)));
    }

    /// @dev op: 0 accrue, 1 supply, 2 withdraw, 3 borrow, 4 repay, 5 supplyCollateral, 6 withdrawCollateral (Blue,
    ///      onBehalf USER); 10 tryPosition, 11 debtOf, 12 suppliedOf, 13 supplySharesToAssets(a), 14 freeLiquidity,
    ///      15 borrowRateAfter(a, b), 16 oraclePrice (account views); 20 supplyCollateral(a), 21 withdrawCollateral(a),
    ///      22 borrow(a), 23 repay(a), 24 supply(a), 25 withdraw(a, b) (account, pranked Router); 30 IRM
    ///      borrowRateView over the stored market, 31 borrowRate pranked as Blue.
    function runRow(string memory tag, St memory s, uint8 op, uint256 a, uint256 b) external {
        bool acct = op >= 10 && op < 30;
        address who = acct ? ACCOUNT : USER;
        uint256 m = _apply(s, who);
        R2Params memory p = params[m];
        bytes32 id = ids[m];
        bool ok;
        bytes memory ret;
        if (op < 10) {
            vm.prank(USER);
            if (op == 0) (ok, ret) = MORPHO.call(abi.encodeCall(R2Morpho.accrueInterest, (p)));
            else if (op == 1) (ok, ret) = MORPHO.call(abi.encodeCall(R2Morpho.supply, (p, a, b, USER, "")));
            else if (op == 2) (ok, ret) = MORPHO.call(abi.encodeCall(R2Morpho.withdraw, (p, a, b, USER, USER)));
            else if (op == 3) (ok, ret) = MORPHO.call(abi.encodeCall(R2Morpho.borrow, (p, a, b, USER, USER)));
            else if (op == 4) (ok, ret) = MORPHO.call(abi.encodeCall(R2Morpho.repay, (p, a, b, USER, "")));
            else if (op == 5) (ok, ret) = MORPHO.call(abi.encodeCall(R2Morpho.supplyCollateral, (p, a, USER, "")));
            else if (op == 6) (ok, ret) = MORPHO.call(abi.encodeCall(R2Morpho.withdrawCollateral, (p, a, USER, USER)));
        } else if (op < 20) {
            if (op == 10) (ok, ret) = ACCOUNT.staticcall(abi.encodeCall(R2Account.tryPosition, (id)));
            else if (op == 11) (ok, ret) = ACCOUNT.staticcall(abi.encodeCall(R2Account.debtOf, (id)));
            else if (op == 12) (ok, ret) = ACCOUNT.staticcall(abi.encodeCall(R2Account.suppliedOf, (id)));
            else if (op == 13) (ok, ret) = ACCOUNT.staticcall(abi.encodeCall(R2Account.supplySharesToAssets, (id, a)));
            else if (op == 14) (ok, ret) = ACCOUNT.staticcall(abi.encodeCall(R2Account.freeLiquidity, (id)));
            else if (op == 15) (ok, ret) = ACCOUNT.staticcall(abi.encodeCall(R2Account.borrowRateAfter, (id, a, b)));
            else if (op == 16) (ok, ret) = ACCOUNT.staticcall(abi.encodeCall(R2Account.oraclePrice, (id)));
        } else if (op < 30) {
            if (op == 23 || op == 24) deal(USDC, ACCOUNT, a);
            vm.prank(ROUTER);
            if (op == 20) (ok, ret) = ACCOUNT.call(abi.encodeCall(R2Account.supplyCollateral, (id, a)));
            else if (op == 21) (ok, ret) = ACCOUNT.call(abi.encodeCall(R2Account.withdrawCollateral, (id, a, POOL)));
            else if (op == 22) (ok, ret) = ACCOUNT.call(abi.encodeCall(R2Account.borrow, (id, a, POOL)));
            else if (op == 23) (ok, ret) = ACCOUNT.call(abi.encodeCall(R2Account.repay, (id, a)));
            else if (op == 24) (ok, ret) = ACCOUNT.call(abi.encodeCall(R2Account.supply, (id, a)));
            else if (op == 25) (ok, ret) = ACCOUNT.call(abi.encodeCall(R2Account.withdraw, (id, a, b, POOL)));
        } else {
            (uint128 x0, uint128 x1, uint128 x2, uint128 x3, uint128 x4, uint128 x5) = R2Morpho(MORPHO).market(id);
            R2MarketState memory ms = R2MarketState(x0, x1, x2, x3, x4, x5);
            if (op == 30) {
                (ok, ret) = IRM.staticcall(abi.encodeCall(R2Irm.borrowRateView, (p, ms)));
            } else {
                vm.prank(MORPHO);
                (ok, ret) = IRM.call(abi.encodeCall(R2Irm.borrowRate, (p, ms)));
            }
        }
        string memory o = "{";
        o = add(o, "tag", qs(tag));
        o = add(o, "now", q(block.timestamp));
        o = add(o, "lltv", q(p.lltv));
        o = add(o, "st", _stJson(s));
        o = add(o, "op", q(op));
        o = add(o, "a", q(a));
        o = add(o, "b", q(b));
        o = add(o, "ok", qb(ok));
        o = add(o, "ret", qh(ret));
        o = add(o, "postMarket", marketJson(id));
        o = add(o, "postPosition", positionJson(id, who));
        o = end(add(o, "postRat", qi(s.noIrm ? int256(0) : R2Irm(IRM).rateAtTarget(id))));
        row(o);
    }

    // ------------------------------------------------------------------ states
    function _st(uint128 tsa, uint128 tba, uint64 elapsed, uint128 fee, int256 rat) internal pure returns (St memory s) {
        s.tsa = tsa;
        s.tss = uint128(uint256(tsa) * 1e6 + 7777);
        s.tba = tba;
        s.tbs = uint128(uint256(tba) * 1e6 + 333);
        s.elapsed = elapsed;
        s.fee = fee;
        s.rat = rat;
        s.oracleP = ORACLE0;
    }

    function _all(string memory tag, St memory s, uint256 a, uint256 b) internal {
        uint8[23] memory ops = [0, 1, 2, 3, 4, 5, 6, 10, 11, 12, 13, 14, 15, 16, 20, 21, 22, 23, 24, 25, 30, 31, 2];
        for (uint256 i; i < 22; ++i) {
            uint8 op = ops[i];
            // Blue takes exactly one of assets / shares; keep b for the two-argument calls only
            uint256 bb = (op == 2 || op == 4 || op == 15 || op == 25) ? b : 0;
            this.runRow(tag, s, op, a, bb);
        }
    }

    function test_blue() public {
        open("mm_market_edges.json");
        _setup();
        St memory s;

        // A. an oracle answering zero, one reverting, one live, on an indebted position (health reads)
        for (uint8 mode; mode < 3; ++mode) {
            for (uint256 ni; ni < 2; ++ni) {
                s = _st(5e12, 3e12, 600, 0.05e18, INIT_RAT * 2);
                s.noIrm = ni == 1;
                s.bor = 2e15;
                s.coll = 1e7;
                s.sup = 1e17;
                s.oracleMode = mode;
                string memory tag = string.concat("oracle-mode-", vm.toString(mode));
                this.runRow(tag, s, 3, 1, 0);
                this.runRow(tag, s, 6, 1, 0);
                this.runRow(tag, s, 22, 1, 0);
                this.runRow(tag, s, 21, 1, 0);
                this.runRow(tag, s, 16, 0, 0);
                s.bor = 0;
                this.runRow(string.concat(tag, "-nodebt"), s, 6, 1, 0);
                this.runRow(string.concat(tag, "-nodebt"), s, 3, 1, 0);
            }
        }

        // B. IRM: the zero-speed point and its 1-wei neighbours, the wExp clip thresholds, the rate bounds
        int256[7] memory rats = [int256(0), MIN_RAT, MIN_RAT + 1, INIT_RAT, MAX_RAT - 1, MAX_RAT, MAX_RAT / 3 + 7];
        uint64[6] memory els = [uint64(0), 1, 2, 3599, 86400, 30 days];
        for (uint256 r; r < rats.length; ++r) {
            for (uint256 e; e < els.length; ++e) {
                for (uint256 d; d < 3; ++d) {
                    s = _st(1e18, uint128(0.9e18 + d - 1), els[e], 0, rats[r]);
                    this.runRow("irm-target", s, 30, 0, 0);
                    this.runRow("irm-target", s, 31, 0, 0);
                }
            }
        }
        // wExp lower clip: utilisation 0 (err = -WAD, speed = -ADJ); upper clip: utilisation 1 (speed = +ADJ)
        uint256 lnWei = 41446531673892822312;
        uint256 upper = 93859467695000404319;
        uint256[2] memory thr = [lnWei / ADJ, upper / ADJ];
        for (uint256 k; k < 2; ++k) {
            for (uint256 d; d < 5; ++d) {
                uint64 el = uint64(thr[k] + d - 2);
                for (uint256 r = 1; r < 6; r += 2) {
                    s = _st(1e18, k == 0 ? 0 : 1e18, el, 0, rats[r]);
                    this.runRow("irm-clip", s, 30, 0, 0);
                    this.runRow("irm-clip", s, 31, 0, 0);
                    this.runRow("irm-clip", s, 0, 0, 0);
                }
            }
        }
        // odd negative / positive linear adaptation (the halving truncates toward zero), utilisation above 1
        for (uint256 k; k < 12; ++k) {
            s = _st(uint128(1e18 + k * 7919), uint128((k % 2 == 0 ? 0.3e18 : 0.97e18) + k * 104729), uint64(1 + k * 3), 0, INIT_RAT + int256(k));
            this.runRow("irm-odd", s, 30, 0, 0);
            this.runRow("irm-odd", s, 31, 0, 0);
            s.tba = uint128(2e18 + k);
            this.runRow("irm-util-over-1", s, 30, 0, 0);
            this.runRow("irm-util-over-1", s, 15, 0, 0);
            s.tsa = 0;
            this.runRow("irm-tsa-0", s, 30, 0, 0);
        }

        // C. account grace: 3599 / 3600 / 3601 elapsed with the IRM down, with and without a fee; tba 0; stamped now
        for (uint256 e; e < 3; ++e) {
            for (uint256 f; f < 2; ++f) {
                s = _st(5e12, 4e12, uint64(3599 + e), f == 0 ? 0 : 0.25e18, INIT_RAT * 5);
                s.irmDead = true;
                s.sup = 3e17;
                s.bor = 1e18;
                s.coll = 5e8;
                string memory tag = string.concat("grace-", vm.toString(3599 + e));
                for (uint8 op = 10; op <= 15; ++op) this.runRow(tag, s, op, op == 13 ? 12345678901 : 1e6, 0);
                this.runRow(tag, s, 23, 5e6, 0);
                this.runRow(tag, s, 25, 5e6, 0);
                this.runRow(tag, s, 25, 0, 0);
                this.runRow(tag, s, 22, 5e6, 0);
                s.irmDead = false;
                for (uint8 op = 10; op <= 13; ++op) this.runRow(string.concat(tag, "-live"), s, op, 777, 0);
            }
        }
        s = _st(5e12, 0, 5000, 0.1e18, INIT_RAT);
        s.sup = 1e18;
        s.bor = 12345;
        s.irmDead = true;
        _all("tba-zero-irm-down", s, 1e6, 3);
        s = _st(5e12, 1e12, 0, 0.1e18, INIT_RAT);
        s.sup = 1e18;
        s.bor = 1e17;
        s.coll = 1e8;
        s.irmDead = true;
        _all("stamped-now-irm-down", s, 1e6, 3);

        // D. rate quotes: supply delta fail-closed at tsa, borrow saturation at uint128, the checked add
        s = _st(8e11, 6e11, 1200, 0.02e18, INIT_RAT * 4);
        uint256 max128 = type(uint128).max;
        uint256[6] memory dss = [uint256(0), 1, 8e11 - 1, 8e11, 8e11 + 1, type(uint256).max];
        uint256[7] memory dbs = [uint256(0), 1, max128 - 6e11 - 1, max128 - 6e11, max128 - 6e11 + 1, type(uint256).max - 6e11, type(uint256).max - 6e11 + 1];
        for (uint256 i; i < dss.length; ++i) {
            for (uint256 j; j < dbs.length; ++j) {
                this.runRow("rate-after", s, 15, dbs[j], dss[i]);
                s.noIrm = true;
                this.runRow("rate-after-noirm", s, 15, dbs[j], dss[i]);
                s.noIrm = false;
                s.irmDead = true;
                this.runRow("rate-after-dead", s, 15, dbs[j], dss[i]);
                s.irmDead = false;
            }
        }
        s.tsa = 0;
        s.tss = 0;
        s.tba = 0;
        s.tbs = 0;
        this.runRow("rate-after-empty", s, 15, 0, 0);
        this.runRow("rate-after-empty", s, 15, 1, 0);
        this.runRow("rate-after-empty", s, 15, 0, 1);

        // E. account repay around the accrued debt, and a share price that burns zero shares
        s = _st(5e12, 3e12, 2400, 0.1e18, INIT_RAT * 3);
        s.bor = 1e15;
        s.coll = 1e8;
        s.sup = 5e16;
        (bool ok0, bytes memory dret) = _debtAt(s);
        uint256 owed = ok0 ? abi.decode(dret, (uint256)) : 1e9;
        uint256[7] memory pays = [uint256(0), 1, owed - 1, owed, owed + 1, owed * 2, owed / 2];
        for (uint256 i; i < pays.length; ++i) {
            this.runRow("repay-owed", s, 23, pays[i], 0);
            s.noIrm = true;
            this.runRow("repay-owed-noirm", s, 23, pays[i], 0);
            s.noIrm = false;
        }
        s = _st(5e12, 1e12, 100, 0, INIT_RAT);
        s.tbs = 1e11;
        s.bor = 1e10;
        s.coll = 1e8;
        for (uint256 i; i < 12; ++i) this.runRow("repay-zero-shares", s, 23, 1 + i * 3, 0);
        this.runRow("repay-zero-shares", s, 4, 7, 0);
        this.runRow("repay-zero-shares", s, 4, 0, 1);

        // F. Blue edges: zero-floor repay by shares, liquidity and health 1-wei boundaries, uint128 casts
        s = _st(5e12, 3, 0, 0, INIT_RAT);
        s.tbs = 5e6;
        s.bor = 5e6;
        s.coll = 1e8;
        this.runRow("zero-floor", s, 4, 0, 5e6);
        this.runRow("zero-floor", s, 4, 0, 4e6);
        this.runRow("zero-floor", s, 23, 1e6, 0);
        for (uint256 d; d < 3; ++d) {
            s = _st(5e12, 4e12, 0, 0, INIT_RAT);
            s.sup = 1e18 * 1e6;
            this.runRow("withdraw-liquidity", s, 2, 1e12 + d - 1, 0);
            this.runRow("borrow-liquidity", s, 3, 1e12 + d - 1, 0);
            this.runRow("account-withdraw-liquidity", s, 25, 1e12 + d - 1, 0);
        }
        // health: collateral c at price ORACLE0 carries floor(floor(c*P/1e36)*lltv/WAD) borrowed assets
        uint256 c = 1e8;
        uint256 maxB = ((c * ORACLE0) / 1e36) * 0.77e18 / 1e18;
        for (uint256 d; d < 3; ++d) {
            s = _st(5e12, 1e12, 0, 0, INIT_RAT);
            s.coll = uint128(c);
            s.tbs = uint128(1e12 * 1e6);
            s.tba = uint128(1e12 - 1);
            s.bor = uint128((maxB - 2) * 1e6);
            this.runRow("health-borrow", s, 3, 1 + d, 0);
            this.runRow("health-borrow-acct", s, 22, 1 + d, 0);
        }
        s = _st(5e12, 1e12, 0, 0, INIT_RAT);
        s.coll = uint128(c);
        s.tbs = uint128(1e12 * 1e6);
        s.tba = uint128(1e12 - 1);
        s.bor = uint128(maxB * 1e6);
        for (uint256 d; d < 4; ++d) this.runRow("health-withdrawCollateral", s, 6, 1 + d, 0);
        for (uint256 d; d < 4; ++d) this.runRow("health-withdrawCollateral-acct", s, 21, 1 + d, 0);
        s = _st(5e12, 1e12, 0, 0, INIT_RAT);
        s.coll = uint128(max128 - 5);
        this.runRow("u128-collateral", s, 5, 5, 0);
        this.runRow("u128-collateral", s, 5, 6, 0);
        this.runRow("u128-collateral", s, 5, max128 + 1, 0);
        this.runRow("u128-collateral", s, 20, 6, 0);
        s = _st(uint128(max128 - 10), 1e12, 0, 0, INIT_RAT);
        s.tss = uint128(max128 - 10);
        this.runRow("u128-supply", s, 1, 10, 0);
        this.runRow("u128-supply", s, 1, 11, 0);
        this.runRow("u128-supply", s, 1, max128 + 1, 0);
        this.runRow("u128-supply", s, 24, 11, 0);

        // G. seeded grid: realistic magnitudes, every op
        for (uint256 seed; seed < 420; ++seed) {
            St memory g = _randomSt(seed);
            uint8[21] memory ops = [0, 1, 2, 3, 4, 5, 6, 10, 11, 12, 13, 14, 15, 20, 21, 22, 23, 24, 25, 30, 31];
            uint8 op = ops[rnd(seed, 90) % 21];
            uint256 a = rlog(seed, 91, 0, rnd(seed, 92) % 5 == 0 ? 1e18 : 1e13);
            uint256 b = (op == 2 || op == 4) ? (rnd(seed, 93) % 2 == 0 ? 0 : rlog(seed, 94, 1, 1e19)) : ((op == 15 || op == 25) ? rlog(seed, 95, 0, 1e13) : 0);
            if ((op == 2 || op == 4) && b != 0) a = 0;
            this.runRow(string.concat("grid-", vm.toString(seed)), g, op, a, b);
        }
        close();
    }

    function _debtAt(St memory s) internal returns (bool ok, bytes memory ret) {
        uint256 snap = vm.snapshotState();
        uint256 m = _apply(s, ACCOUNT);
        (ok, ret) = ACCOUNT.staticcall(abi.encodeCall(R2Account.debtOf, (ids[m])));
        vm.revertToState(snap);
    }

    function _randomSt(uint256 seed) internal pure returns (St memory s) {
        uint128 tsa = uint128(rlog(seed, 1, 1e6, 1e15));
        uint256 u = rnd(seed, 2) % 10 < 7 ? 800 + rnd(seed, 3) % 200 : rnd(seed, 4) % 1001;
        uint128 tba = uint128(uint256(tsa) * u / 1000);
        uint64 el = uint64(rnd(seed, 5) % 4 == 0 ? rnd(seed, 6) % 400 days : rnd(seed, 7) % 7200);
        s = _st(tsa, tba, el, rnd(seed, 8) % 3 == 0 ? uint128(rnd(seed, 9) % 0.25e18) : 0, MIN_RAT + int256(rnd(seed, 10) % uint256(MAX_RAT - MIN_RAT)));
        if (rnd(seed, 11) % 9 == 0) s.rat = 0;
        s.tss = uint128(uint256(tsa) * (1e6 + rnd(seed, 12) % 30000) / 1e6 + rnd(seed, 13) % 1e6);
        s.tbs = uint128(uint256(tba) * (1e6 - rnd(seed, 14) % 20000) / 1e6 + rnd(seed, 15) % 1e6);
        s.sup = rnd(seed, 16) % 2 == 0 ? s.tss * (rnd(seed, 17) % 1000) / 1000 : 0;
        s.bor = uint128(uint256(s.tbs) * (rnd(seed, 18) % 1000) / 1000);
        s.coll = uint128(rlog(seed, 19, 0, 1e10));
        s.oracleP = ORACLE0 * (500 + rnd(seed, 20) % 1000) / 1000;
        s.oracleMode = rnd(seed, 21) % 12 == 0 ? uint8(1 + rnd(seed, 22) % 2) : 0;
        s.irmDead = rnd(seed, 23) % 8 == 0;
        s.noIrm = rnd(seed, 24) % 6 == 0;
    }
}
