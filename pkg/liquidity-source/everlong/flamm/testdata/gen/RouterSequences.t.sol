// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import "./FinancingSequenceBase.sol";

/// @dev Stateful sequence generator for the financing stack: one long pseudo-random sequence of Router
///      transactions against the DEPLOYED MMRouter, MorphoBlueAccount, Morpho Blue and AdaptiveCurveIrm on a Base
///      fork, with the live pool record extended to four venues over two loan assets (the live 86% USDC market, a
///      created 77% USDC market with a 10% fee, a created 91.5% USDC market with no rate model, a created 86% market
///      on an 18-decimal loan asset). The pool address carries a call-bundling proxy so a transaction can hold
///      several Router calls (repay then proportional withdraw). Run with --isolate: every `step` is one
///      transaction, so the Router's transient repay snapshot ends with it. Between transactions: time warps,
///      third-party Morpho activity (utilisation up to 100%), IRM outages (grace and quarantine), oracle moves onto
///      the exact band edges, zero and reverting oracles, and Router config changes (caps, rate ceilings on the
///      live rate +-1, flags, pin, pause, drawn set, priorities). Each step records the pre-state, the op, its
///      result, the post-state and the Router / account views.
contract RouterSequences is FinancingSequenceBase {
    uint256 internal constant STEPS = 520;
    uint256 internal constant P0 = 788432322552395;
    uint256 internal constant P1 = 262810774184131;
    address internal constant SAFE = 0xeb765F3184f705F5292679f42beEDBbe5D272c49;

    R3Token internal token;
    R3SetOracle[4] internal oracles;
    bytes32[4] internal ids;
    R3Params[4] internal ps;
    bool[4] internal irmDown;
    uint8 internal liveMode; // 0 real, 1 revert, 2 zero, 3 price
    uint256 internal livePrice;
    uint256 internal stepSalt;

    // ------------------------------------------------------------------ setup
    function _pool(address target, bytes memory data) internal returns (bool ok, bytes memory ret) {
        vm.prank(POOL);
        (ok, ret) = target.call(data);
    }

    function _mustPool(address target, bytes memory data) internal returns (bytes memory ret) {
        bool ok;
        (ok, ret) = _pool(target, data);
        require(ok, string(ret));
    }

    function _create(uint256 v, address loanToken, address irm, uint256 lltv, uint256 oraclePrice) internal {
        oracles[v] = new R3SetOracle();
        oracles[v].set(oraclePrice, 0);
        ps[v] = R3Params(loanToken, CBBTC, address(oracles[v]), irm, lltv);
        ids[v] = keccak256(abi.encode(ps[v]));
        R3Morpho(MORPHO).createMarket(ps[v]);
    }

    function setUpSeq() internal {
        seed = 0x5eed3;
        token = new R3Token();
        vm.etch(POOL, address(new R3PoolProxy()).code);
        ids[0] = LIVE_ID;
        ps[0] = params(LIVE_ID);
        _create(1, USDC, IRM, 0.77e18, P0 * 0.98e24);
        _create(2, USDC, address(0), 0.915e18, P0 * 1e24);
        _create(3, address(token), IRM, 0.86e18, P1 * 1.02e36);
        vm.prank(R3Morpho(MORPHO).owner());
        R3Morpho(MORPHO).setFee(ps[1], 0.1e18);

        deal(USDC, WHALE, 2_000_000_000e6);
        deal(CBBTC, WHALE, 100_000e8);
        token.mint(WHALE, 1e40);
        deal(USDC, POOL, 500_000_000e6);
        deal(CBBTC, POOL, 50_000e8);
        token.mint(POOL, 1e40);
        vm.startPrank(WHALE);
        R3ERC20(USDC).approve(MORPHO, type(uint256).max);
        R3ERC20(CBBTC).approve(MORPHO, type(uint256).max);
        token.approve(MORPHO, type(uint256).max);
        R3Morpho(MORPHO).supply(ps[1], 40_000_000e6, 0, WHALE, "");
        R3Morpho(MORPHO).supply(ps[2], 30_000_000e6, 0, WHALE, "");
        R3Morpho(MORPHO).supply(ps[3], 1e27, 0, WHALE, "");
        for (uint256 v; v < 4; ++v) R3Morpho(MORPHO).supplyCollateral(ps[v], 20_000e8, WHALE, "");
        R3Morpho(MORPHO).borrow(ps[1], 30_000_000e6, 0, WHALE, WHALE);
        R3Morpho(MORPHO).borrow(ps[3], 3e26, 0, WHALE, WHALE);
        R3Morpho(MORPHO).borrow(ps[2], 10_000_000e6, 0, WHALE, WHALE);
        vm.stopPrank();

        _mustPool(ROUTER, abi.encodeWithSignature("addLoanAsset(address)", address(token)));
        _mustPool(ROUTER, abi.encodeWithSignature("addVenue(uint8,uint8,bytes,bool,bool,uint256)", uint8(0), uint8(0), abi.encode(ps[1]), true, true, P0));
        _mustPool(ROUTER, abi.encodeWithSignature("addVenue(uint8,uint8,bytes,bool,bool,uint256)", uint8(0), uint8(0), abi.encode(ps[2]), true, true, P0));
        _mustPool(ROUTER, abi.encodeWithSignature("addVenue(uint8,uint8,bytes,bool,bool,uint256)", uint8(0), uint8(1), abi.encode(ps[3]), true, true, P1));
        _mustPool(USDC, abi.encodeWithSignature("approve(address,uint256)", ROUTER, type(uint256).max));
        _mustPool(CBBTC, abi.encodeWithSignature("approve(address,uint256)", ROUTER, type(uint256).max));
        _mustPool(address(token), abi.encodeWithSignature("approve(address,uint256)", ROUTER, type(uint256).max));
    }

    function test_routerSeqA() public {
        _run(0x5eed3, "router_sequence_a.jsonl");
    }

    function test_routerSeqB() public {
        _run(0xBEEF5, "router_sequence_b.jsonl");
    }

    function _run(uint256 seed_, string memory name) internal {
        vm.createSelectFork(RPC, BLOCK);
        setUpSeq();
        seed = seed_;
        open(name);
        for (uint256 i; i < STEPS; ++i) {
            uint256 r = rnd(1_000_000 + i) % 100;
            uint256 dt = r < 35 ? 0 : r < 45 ? 1 : r < 55 ? 13 : r < 72 ? 600 : r < 77 ? 3599 : r < 82 ? 3600 : r < 87 ? 3601 : r < 96 ? 86400 : 864000;
            vm.warp(block.timestamp + dt);
            this.step(i);
        }
    }

    // ------------------------------------------------------------------ one transaction
    function step(uint256 i) external {
        stepSalt = i * 1000;
        string memory env = _env(i);
        string memory pre = routerJson();
        (string memory opj, string memory resj) = _op(i);
        string memory s = string.concat('{"i":', vm.toString(i), ',"t":', q(block.timestamp), ',"env":', env);
        s = string.concat(s, ',"pre":', pre, ',"op":', opj, ',"res":', resj, ',"post":', routerJson());
        s = string.concat(s, ',"views":', _views(i), "}");
        line(s);
    }

    function _prices(uint256 salt) internal view returns (uint256[] memory p) {
        p = new uint256[](2);
        p[0] = _price(0, salt);
        p[1] = _price(1, salt + 1);
    }

    function _price(uint8 idx, uint256 salt) internal view returns (uint256) {
        uint256 base = idx == 0 ? P0 : P1;
        uint256 r = rnd(salt) % 20;
        if (r < 11) return base;
        if (r < 14) return base + 1;
        if (r < 17) return base - 1;
        if (r == 17) return 0;
        if (r == 18) return base * 105 / 100;
        return base * 97 / 100;
    }

    function _loanToken(uint8 idx) internal view returns (address) {
        return idx == 0 ? USDC : address(token);
    }

    function _unit(uint8 idx) internal pure returns (uint256) {
        return idx == 0 ? 1 : 1e12;
    }

    // ------------------------------------------------------------------ environment between transactions
    function _applyMocks() internal {
        vm.clearMockedCalls();
        for (uint256 v; v < 4; ++v) {
            if (!irmDown[v] || ps[v].irm == address(0)) continue;
            vm.mockCallRevert(IRM, abi.encodeWithSelector(R3Irm.borrowRateView.selector, ps[v]), "irm down");
            vm.mockCallRevert(IRM, abi.encodeWithSelector(R3Irm.borrowRate.selector, ps[v]), "irm down");
        }
        if (liveMode == 1) vm.mockCallRevert(LIVE_ORACLE, abi.encodeWithSelector(R3Oracle.price.selector), "oracle down");
        if (liveMode == 2) vm.mockCall(LIVE_ORACLE, abi.encodeWithSelector(R3Oracle.price.selector), abi.encode(uint256(0)));
        if (liveMode == 3) vm.mockCall(LIVE_ORACLE, abi.encodeWithSelector(R3Oracle.price.selector), abi.encode(livePrice));
    }

    function _env(uint256 i) internal returns (string memory s) {
        s = "[";
        uint256 r = rnd(stepSalt + 1) % 100;
        if (r < 3) s = string.concat(s, _envLiquidate());
        else if (r < 16) s = string.concat(s, _envThirdParty(i));
        else if (r < 22) s = string.concat(s, _envIrm());
        else if (r < 29) s = string.concat(s, _envOracle());
        else if (r < 40) s = string.concat(s, _envConfig());
        s = string.concat(s, "]");
    }

    function _envIrm() internal returns (string memory) {
        uint256[3] memory withIrm = [uint256(0), 1, 3];
        uint256 v = withIrm[rnd(stepSalt + 2) % 3];
        irmDown[v] = !irmDown[v];
        _applyMocks();
        return string.concat('{"kind":"irm","venue":', vm.toString(v), ',"down":', qb(irmDown[v]), "}");
    }

    function _envOracle() internal returns (string memory) {
        uint256 v = rnd(stepSalt + 3) % 4;
        uint256 m = rnd(stepSalt + 4) % 8;
        uint256 unit = v == 3 ? 1e36 : 1e24;
        uint256 base = v == 3 ? P1 : P0;
        uint256 p = m < 2 ? base * unit : m == 2 ? base * (unit / 100 * 98) : m == 3 ? base * (unit / 100 * 102) : m == 4 ? base * (unit / 100 * 110) : base * unit + rnd(stepSalt + 5) % (unit / 50);
        uint8 mode = m == 5 ? 1 : m == 6 ? 2 : 0;
        if (v == 0) {
            liveMode = m == 7 ? 0 : mode == 1 ? 1 : mode == 2 ? 2 : 3;
            livePrice = p;
            _applyMocks();
        } else {
            oracles[v].set(p, mode);
        }
        return string.concat('{"kind":"oracle","venue":', vm.toString(v), ',"mode":', vm.toString(m), ',"price":', q(p), "}");
    }

    function _envThirdParty(uint256) internal returns (string memory s) {
        uint256 v = rnd(stepSalt + 6) % 4;
        uint256 kind = rnd(stepSalt + 7) % 8;
        uint256 unit = v == 3 ? 1e12 : 1;
        R3Market memory m = mkt(ids[v]);
        (uint256 wSup, uint128 wBor,) = R3Morpho(MORPHO).position(ids[v], WHALE);
        uint256 cash = m.totalSupplyAssets > m.totalBorrowAssets ? m.totalSupplyAssets - m.totalBorrowAssets : 0;
        uint256 a;
        uint256 sh;
        address who = WHALE;
        bytes memory data;
        if (kind == 0) {
            a = rlog(stepSalt + 8, 1, 60_000_000e6 * unit);
            data = abi.encodeCall(R3Morpho.supply, (ps[v], a, 0, WHALE, ""));
        } else if (kind == 1) {
            uint256 r = rnd(stepSalt + 9) % 3;
            if (r == 0) {
                sh = rlog(stepSalt + 10, 1, wSup);
                data = abi.encodeCall(R3Morpho.withdraw, (ps[v], 0, sh, WHALE, WHALE));
            } else {
                a = r == 1 ? cash : rlog(stepSalt + 10, 1, cash);
                data = abi.encodeCall(R3Morpho.withdraw, (ps[v], a, 0, WHALE, WHALE));
            }
        } else if (kind == 2) {
            uint256 r = rnd(stepSalt + 11) % 4;
            a = r == 0 ? cash : r == 1 ? (cash > 1000 ? cash - rnd(stepSalt + 12) % 1000 : cash) : rlog(stepSalt + 12, 1, cash);
            data = abi.encodeCall(R3Morpho.borrow, (ps[v], a, 0, WHALE, WHALE));
        } else if (kind == 3) {
            if (rnd(stepSalt + 13) % 3 == 0) {
                sh = rlog(stepSalt + 14, 1, wBor);
                data = abi.encodeCall(R3Morpho.repay, (ps[v], 0, sh, WHALE, ""));
            } else {
                a = rlog(stepSalt + 14, 1, m.totalBorrowAssets / 20);
                data = abi.encodeCall(R3Morpho.repay, (ps[v], a, 0, WHALE, ""));
            }
        } else if (kind >= 5) {
            who = IMMRouter(ROUTER).venue(POOL, uint16(v)).account;
            (, uint128 aBor,) = R3Morpho(MORPHO).position(ids[v], who);
            if (kind == 5) {
                a = rlog(stepSalt + 19, 1, 50_000e6 * unit);
                data = abi.encodeCall(R3Morpho.supply, (ps[v], a, 0, who, ""));
            } else if (kind == 6) {
                a = rlog(stepSalt + 19, 1, 1e7);
                data = abi.encodeCall(R3Morpho.supplyCollateral, (ps[v], a, who, ""));
            } else if (rnd(stepSalt + 18) % 2 == 0) {
                sh = rlog(stepSalt + 19, 1, aBor);
                data = abi.encodeCall(R3Morpho.repay, (ps[v], 0, sh, who, ""));
            } else {
                a = rlog(stepSalt + 19, 1, 20e6 * unit);
                data = abi.encodeCall(R3Morpho.repay, (ps[v], a, 0, who, ""));
            }
        } else {
            a = rlog(stepSalt + 15, 1, 1_000e8);
            data = abi.encodeCall(R3Morpho.supplyCollateral, (ps[v], a, WHALE, ""));
        }
        string memory wpre = positionJson(ids[v], who);
        string memory mpre = marketJson(ids[v]);
        string memory rpre = q(ps[v].irm == address(0) ? 0 : uint256(R3Irm(IRM).rateAtTarget(ids[v])));
        vm.prank(WHALE);
        (bool ok, bytes memory ret) = MORPHO.call(data);
        s = string.concat('{"kind":"tp","venue":', vm.toString(v), ',"op":', vm.toString(kind), ',"a":', q(a), ',"s":', q(sh), ',"who":', qs(who == WHALE ? "whale" : "account"));
        s = string.concat(s, ',"wpre":', wpre, ',"mpre":', mpre, ',"rpre":', rpre, ',"res":', res(ok, ret));
        s = string.concat(s, ',"wpost":', positionJson(ids[v], who), ',"mpost":', marketJson(ids[v]));
        s = string.concat(s, ',"rpost":', q(ps[v].irm == address(0) ? 0 : uint256(R3Irm(IRM).rateAtTarget(ids[v]))), "}");
    }

    /// @dev A third party liquidates the financing account on a settable-oracle venue: the oracle is dropped far
    ///      enough to break Blue's health, part (or all, realising bad debt) of the position is liquidated, and the
    ///      oracle is restored. The port does not model liquidation; the replay takes the chain's post figures.
    function _envLiquidate() internal returns (string memory s) {
        uint256 v = rnd(stepSalt + 90) % 2 == 0 ? 1 : 3;
        address acct = IMMRouter(ROUTER).venue(POOL, uint16(v)).account;
        (, uint128 bor,) = R3Morpho(MORPHO).position(ids[v], acct);
        if (bor == 0) {
            uint256 u = v == 3 ? 1e12 : 1;
            bytes memory d1 = abi.encodeCall(IMMRouter.postCollateral, (uint16(v), 1e7));
            bytes memory d2 = abi.encodeCall(IMMRouter.borrow, (uint16(v), rlog(stepSalt + 92, 1e6, 30_000e6) * u, POOL, v == 3 ? P1 : P0));
            (bool ok1, bytes memory r1) = _pool(ROUTER, d1);
            (bool ok2, bytes memory r2) = _pool(ROUTER, d2);
            s = string.concat('{"kind":"poolcall","data":', qh(d1), ',"res":', res(ok1, r1), '},{"kind":"poolcall","data":', qh(d2), ',"res":', res(ok2, r2), "},");
        }
        s = string.concat(s, _liq(v, rnd(stepSalt + 91) % 3));
    }

    /// @dev mode 0 seizes all the collateral (bad debt realised), 1 a third of it, 2 repays half the borrow shares.
    function _liq(uint256 v, uint256 mode) internal returns (string memory s) {
        address acct = IMMRouter(ROUTER).venue(POOL, uint16(v)).account;
        (, uint128 bor, uint128 col) = R3Morpho(MORPHO).position(ids[v], acct);
        uint256 oldP = oracles[v].p();
        uint8 oldMode = oracles[v].mode();
        oracles[v].set(oldP / 4, 0);
        uint256 seize = mode == 0 ? col : mode == 1 ? col / 3 : 0;
        uint256 shares = mode == 2 ? bor / 2 : 0;
        string memory wpre = positionJson(ids[v], acct);
        string memory mpre = marketJson(ids[v]);
        string memory rpre = q(uint256(R3Irm(IRM).rateAtTarget(ids[v])));
        vm.prank(WHALE);
        (bool ok, bytes memory ret) = MORPHO.call(abi.encodeCall(R3Morpho.liquidate, (ps[v], acct, seize, shares, "")));
        oracles[v].set(oldP, oldMode);
        s = string.concat('{"kind":"liq","venue":', vm.toString(v), ',"wpre":', wpre, ',"mpre":', mpre, ',"rpre":', rpre);
        s = string.concat(s, ',"res":', res(ok, ret), ',"wpost":', positionJson(ids[v], acct), ',"mpost":', marketJson(ids[v]));
        s = string.concat(s, ',"rpost":', q(uint256(R3Irm(IRM).rateAtTarget(ids[v]))), "}");
    }

    /// @dev Scripted: the account indebted on the fee market and on the 18-decimal market, then partly and fully
    ///      liquidated by a third party (managed collateral above the actual, bad debt realised), with Router
    ///      transactions between: reclaim, proportional and pinned withdrawals, fund, cascades.
    function test_routerScenarioLiq() public {
        vm.createSelectFork(RPC, BLOCK);
        setUpSeq();
        seed = 0x1100;
        open("router_sequence_liquidation.jsonl");
        uint256[] memory pr = new uint256[](2);
        (pr[0], pr[1]) = (P0, P1);
        uint256 i;
        this.stepScripted(i++, 0, 0, abi.encodeCall(IMMRouter.postCollateral, (uint16(1), 1e8)));
        this.stepScripted(i++, 0, 0, abi.encodeCall(IMMRouter.borrow, (uint16(1), 30_000e6, POOL, P0)));
        this.stepScripted(i++, 0, 0, abi.encodeCall(IMMRouter.postCollateral, (uint16(3), 1e8)));
        this.stepScripted(i++, 0, 0, abi.encodeCall(IMMRouter.borrow, (uint16(3), 1e22, POOL, P1)));
        vm.warp(block.timestamp + 3000);
        this.stepScripted(i++, 1, 2, abi.encodeCall(IMMRouter.reclaimBestEffort, (5e7, pr)));
        this.stepScripted(i++, 0, 0, abi.encodeCall(IMMRouter.withdrawCollateral, (uint16(1), 1e6, P0, false)));
        this.stepScripted(i++, 0, 0, abi.encodeCall(IMMRouter.repayCascade, (uint8(0), 5_000e6)));
        this.stepScripted(i++, 0, 0, abi.encodeCall(IMMRouter.withdrawCollateral, (uint16(1), 1e6, 0, true)));
        this.stepScripted(i++, 1, 1, abi.encodeCall(IMMRouter.fund, (uint8(0), 20_000e6, POOL, 5e7, P0)));
        vm.warp(block.timestamp + 600);
        this.stepScripted(i++, 3, 2, abi.encodeCall(IMMRouter.reclaim, (1e6, pr)));
        this.stepScripted(i++, 0, 0, abi.encodeCall(IMMRouter.supplyCascade, (uint8(1), 1e21)));
        this.stepScripted(i++, 3, 0, abi.encodeCall(IMMRouter.repayCascade, (uint8(1), 2e21)));
        this.stepScripted(i++, 0, 0, abi.encodeCall(IMMRouter.withdrawCollateral, (uint16(3), 1, P1, false)));
        this.stepScripted(i++, 0, 0, abi.encodeCall(IMMRouter.postCollateral, (uint16(3), 5e7)));
        this.stepScripted(i++, 0, 0, abi.encodeCall(IMMRouter.fund, (uint8(1), 1e21, POOL, 0, P1)));
        this.stepScripted(i++, 1, 0, abi.encodeCall(IMMRouter.reclaimBestEffort, (type(uint256).max / 2, pr)));
        vm.warp(block.timestamp + 86400);
        this.stepScripted(i++, 0, 0, abi.encodeCall(IMMRouter.repay, (uint16(1), type(uint256).max)));
        this.stepScripted(i++, 0, 0, abi.encodeCall(IMMRouter.withdrawCollateral, (uint16(1), 1, 0, false)));
        this.stepScripted(i++, 0, 0, abi.encodeCall(IMMRouter.repayCascade, (uint8(1), 1e30)));
        this.stepScripted(i++, 0, 0, abi.encodeCall(IMMRouter.reclaim, (1, pr)));
    }

    function stepScripted(uint256 i, uint256 liqVenue, uint256 liqMode, bytes calldata data) external {
        stepSalt = (i + 5000) * 1000;
        string memory env = liqVenue == 0 ? "[]" : string.concat("[", _liq(liqVenue, liqMode), "]");
        string memory pre = routerJson();
        string memory resj = _call(data);
        string memory s = string.concat('{"i":', vm.toString(i), ',"t":', q(block.timestamp), ',"env":', env);
        s = string.concat(s, ',"pre":', pre, ',"op":{"kind":"call","data":', qh(data), '},"res":', resj, ',"post":', routerJson());
        s = string.concat(s, ',"views":', _views(i), "}");
        line(s);
    }

    function _envConfig() internal returns (string memory s) {
        uint256 k = rnd(stepSalt + 20) % 9;
        bytes memory data;
        address target = ROUTER;
        uint16 id = uint16(rnd(stepSalt + 21) % 4);
        if (k <= 2) {
            IMMRouter.VenueView memory vv = IMMRouter(ROUTER).venue(POOL, id);
            (bool okd, bytes memory rd) = vv.account.staticcall(abi.encodeCall(R3Account.debtOf, (vv.id)));
            uint256 debt = okd ? abi.decode(rd, (uint256)) : 0;
            uint256 c = rnd(stepSalt + 22) % 4;
            uint128 debtCap = c == 0 ? 0 : c == 1 ? uint128(debt + rlog(stepSalt + 23, 0, 1e7 * _unit(vv.loanIndex))) : uint128(rlog(stepSalt + 23, 1, 1e13 * _unit(vv.loanIndex)));
            uint128 supplyCap = rnd(stepSalt + 24) % 3 == 0 ? 0 : uint128(rlog(stepSalt + 25, 1, 1e13 * _unit(vv.loanIndex)));
            uint64 maxRate;
            uint256 cr = rnd(stepSalt + 26) % 4;
            if (cr == 1) {
                maxRate = uint64(rlog(stepSalt + 27, 1e8, 1e11));
            } else if (cr >= 2) {
                uint256 dB = rlog(stepSalt + 28, 0, 1e11 * _unit(vv.loanIndex));
                (bool okr, bytes memory rr) = vv.account.staticcall(abi.encodeCall(R3Account.borrowRateAfter, (vv.id, dB, 0)));
                (, uint256 rate) = okr ? abi.decode(rr, (bool, uint256)) : (false, uint256(0));
                maxRate = uint64(rate + (cr == 2 ? 0 : 1));
            }
            data = abi.encodeWithSignature("setVenueCaps(uint16,uint128,uint128,uint64)", id, debtCap, supplyCap, maxRate);
        } else if (k == 3) {
            uint8 idx = uint8(rnd(stepSalt + 30) % 2);
            (bool okp, bytes memory rp) = ROUTER.staticcall(abi.encodeCall(IMMRouter.position, (POOL, idx)));
            (, uint256 sup, uint256 debt) = okp ? abi.decode(rp, (uint256, uint256, uint256)) : (0, 0, 0);
            uint256 c = rnd(stepSalt + 31) % 3;
            uint128 debtCap = c == 0 ? 0 : uint128(debt + rlog(stepSalt + 32, 0, 1e8 * _unit(idx)));
            uint128 supplyCap = rnd(stepSalt + 33) % 2 == 0 ? 0 : uint128(sup + rlog(stepSalt + 34, 0, 1e8 * _unit(idx)));
            data = abi.encodeWithSignature("setLoanCaps(uint8,uint128,uint128,bool)", idx, debtCap, supplyCap, rnd(stepSalt + 35) % 5 != 0);
        } else if (k == 4) {
            data = abi.encodeWithSignature("setVenueFlags(uint16,bool,bool)", id, rnd(stepSalt + 36) % 5 != 0, rnd(stepSalt + 37) % 5 != 0);
        } else if (k == 5) {
            bool p = rnd(stepSalt + 38) % 3 == 0;
            vm.prank(SAFE);
            IMMRouter(ROUTER).setGlobalPaused(p);
            return string.concat('{"kind":"cfg","what":"pause","v":', qb(p), "}");
        } else if (k == 6) {
            data = abi.encodeWithSignature("setMaxDrawnAssets(uint8)", uint8(1 + rnd(stepSalt + 39) % 2));
        } else if (k == 7) {
            data = abi.encodeWithSignature("setPriorities(uint16[],uint16[],uint16[],uint16[])", _perm(40), _perm(41), _perm(42), _perm(43));
        } else {
            uint64[4] memory pins = [uint64(0.5e18), 0.55e18, 0.6e18, 0.7e18];
            data = abi.encodeWithSignature("setPin(uint64)", pins[rnd(stepSalt + 44) % 4]);
        }
        (bool ok, bytes memory ret) = _pool(target, data);
        s = string.concat('{"kind":"cfg","data":', qh(data), ',"res":', res(ok, ret), "}");
    }

    function _perm(uint256 salt) internal view returns (uint16[] memory o) {
        o = new uint16[](4);
        for (uint256 i; i < 4; ++i) o[i] = uint16(i);
        for (uint256 i = 3; i > 0; --i) {
            uint256 j = rnd(stepSalt + salt * 7 + i) % (i + 1);
            (o[i], o[j]) = (o[j], o[i]);
        }
    }

    // ------------------------------------------------------------------ the op
    function _near(uint256 x, uint256 salt, uint256 hi) internal view returns (uint256) {
        uint256 r = rnd(salt) % 8;
        if (r == 0) return x;
        if (r == 1) return x + 1;
        if (r == 2) return x > 0 ? x - 1 : 0;
        if (r == 3) return x / 2;
        if (r == 4) return 0;
        if (r == 5) return 1;
        return rlog(salt + 1, 1, hi);
    }

    function _call(bytes memory data) internal returns (string memory) {
        (bool ok, bytes memory ret) = _pool(ROUTER, data);
        return res(ok, ret);
    }

    function _op(uint256) internal returns (string memory opj, string memory resj) {
        uint256 r = rnd(stepSalt + 50) % 100;
        uint16 id = uint16(rnd(stepSalt + 51) % 4);
        IMMRouter.VenueView memory vv = IMMRouter(ROUTER).venue(POOL, id);
        uint8 idx = uint8(rnd(stepSalt + 52) % 5 == 0 ? 1 : 0);
        uint256 u = _unit(idx);
        bytes memory data;
        if (r < 22) {
            uint256[5] memory cins = [uint256(0), 1e5, 1e7, 1e9, 1e11];
            uint256 cin = cins[rnd(stepSalt + 53) % 5];
            uint256 p = _price(idx, stepSalt + 54);
            (bool okc, bytes memory rc) = ROUTER.staticcall(abi.encodeCall(IMMRouter.fundingCeiling, (POOL, idx, cin, p)));
            uint256 ceil = okc ? abi.decode(rc, (uint256)) : 0;
            uint256 a = _near(ceil, stepSalt + 55, 1e13 * u);
            data = abi.encodeCall(IMMRouter.fund, (idx, a, POOL, cin, p));
        } else if (r < 34) {
            (bool okp, bytes memory rp) = ROUTER.staticcall(abi.encodeCall(IMMRouter.position, (POOL, idx)));
            (,, uint256 debt) = okp ? abi.decode(rp, (uint256, uint256, uint256)) : (0, 0, 0);
            data = abi.encodeCall(IMMRouter.repayCascade, (idx, _near(debt, stepSalt + 56, 1e12 * u)));
        } else if (r < 43) {
            data = abi.encodeCall(IMMRouter.supplyCascade, (idx, rlog(stepSalt + 57, 1, 1e13 * u)));
        } else if (r < 54) {
            uint256[] memory prices = _prices(stepSalt + 58);
            (bool okr, bytes memory rr) = ROUTER.staticcall(abi.encodeCall(IMMRouter.reclaimable, (POOL, prices)));
            uint256 a = _near(okr ? abi.decode(rr, (uint256)) : 0, stepSalt + 60, 1e9);
            data = rnd(stepSalt + 61) % 2 == 0 ? abi.encodeCall(IMMRouter.reclaim, (a, prices)) : abi.encodeCall(IMMRouter.reclaimBestEffort, (a, prices));
        } else if (r < 62) {
            uint256 p = _price(vv.loanIndex, stepSalt + 62);
            uint256 room = _borrowRoom(vv, p);
            uint256 a = rnd(stepSalt + 63) % 25 == 0 ? type(uint256).max : _near(room, stepSalt + 64, 1e12 * _unit(vv.loanIndex));
            data = abi.encodeCall(IMMRouter.borrow, (id, a, POOL, p));
        } else if (r < 68) {
            (bool okd, bytes memory rd) = vv.account.staticcall(abi.encodeCall(R3Account.debtOf, (vv.id)));
            data = abi.encodeCall(IMMRouter.repay, (id, _near(okd ? abi.decode(rd, (uint256)) : 0, stepSalt + 65, 1e12 * _unit(vv.loanIndex))));
        } else if (r < 74) {
            data = abi.encodeCall(IMMRouter.supply, (id, rnd(stepSalt + 74) % 12 == 0 ? 0 : rlog(stepSalt + 66, 0, 1e13 * _unit(vv.loanIndex))));
        } else if (r < 81) {
            (bool okv, bytes memory rv) = ROUTER.staticcall(abi.encodeCall(IMMRouter.venuePosition, (POOL, id)));
            (,,, uint256 rs,) = okv ? abi.decode(rv, (uint256, uint256, uint256, uint256, uint256)) : (0, 0, 0, 0, 0);
            uint256 a = rnd(stepSalt + 67) % 6 == 0 ? type(uint256).max : _near(rs, stepSalt + 68, 1e13 * _unit(vv.loanIndex));
            data = abi.encodeCall(IMMRouter.withdrawSupplied, (id, a, POOL));
        } else if (r < 86) {
            data = abi.encodeCall(IMMRouter.postCollateral, (id, rnd(stepSalt + 69) % 10 == 0 ? 0 : rlog(stepSalt + 70, 1, 1e9)));
        } else if (r < 93) {
            uint256 p = _price(vv.loanIndex, stepSalt + 71);
            uint256 free = _freeColl(vv, p);
            bool prop = rnd(stepSalt + 72) % 4 == 0;
            data = abi.encodeCall(IMMRouter.withdrawCollateral, (id, _near(free, stepSalt + 73, 1e9), p, prop));
        } else {
            return _bundle(id, vv);
        }
        opj = string.concat('{"kind":"call","data":', qh(data), "}");
        resj = _call(data);
    }

    /// @dev repay(id, x) then withdrawCollateral(id, y, 0, proportional) in ONE transaction.
    function _bundle(uint16 id, IMMRouter.VenueView memory vv) internal returns (string memory opj, string memory resj) {
        (bool okd, bytes memory rd) = vv.account.staticcall(abi.encodeCall(R3Account.debtOf, (vv.id)));
        uint256 d = okd ? abi.decode(rd, (uint256)) : 0;
        (,, uint128 c) = R3Morpho(MORPHO).position(vv.id, vv.account);
        uint256 x = rnd(stepSalt + 80) % 5 == 0 ? d : rlog(stepSalt + 81, 1, d);
        uint256 edge = d == 0 ? c : c - Math_mulDivUp(d > x ? d - x : 0, c, d);
        uint256 y = _near(edge, stepSalt + 82, c);
        address[] memory t = new address[](2);
        bytes[] memory ds = new bytes[](2);
        t[0] = ROUTER;
        t[1] = ROUTER;
        ds[0] = abi.encodeCall(IMMRouter.repay, (id, x));
        ds[1] = abi.encodeCall(IMMRouter.withdrawCollateral, (id, y, 0, rnd(stepSalt + 83) % 6 != 0));
        opj = string.concat('{"kind":"bundle","data":[', qh(ds[0]), ",", qh(ds[1]), "]}");
        (bool ok, bytes memory ret) = POOL.call(abi.encodeCall(R3PoolProxy.exec, (t, ds)));
        if (!ok) return (opj, string.concat('{"ok":false,"ret":', qh(ret), "}"));
        bytes[] memory rets = abi.decode(ret, (bytes[]));
        resj = string.concat('{"ok":true,"rets":[', qh(rets[0]), ",", qh(rets[1]), "]}");
    }

    function Math_mulDivUp(uint256 a, uint256 b, uint256 d) internal pure returns (uint256) {
        if (d == 0) return 0;
        return (a * b + d - 1) / d;
    }

    function _borrowRoom(IMMRouter.VenueView memory vv, uint256 p) internal view returns (uint256) {
        (bool okd, bytes memory rd) = vv.account.staticcall(abi.encodeCall(R3Account.debtOf, (vv.id)));
        uint256 debt = okd ? abi.decode(rd, (uint256)) : 0;
        (,, uint128 c) = R3Morpho(MORPHO).position(vv.id, vv.account);
        (uint64 pin,,) = IMMRouter(ROUTER).pin(POOL);
        uint256 scale = vv.loanIndex == 0 ? 1e12 : 1;
        uint256 maxDebt = uint256(c) * p / scale * pin / 1e18;
        uint256 room = maxDebt > debt ? maxDebt - debt : 0;
        uint256 cash = R3Account(vv.account).freeLiquidity(vv.id);
        if (cash < room && rnd(stepSalt + 90) % 2 == 0) room = cash;
        if (vv.debtCap != 0 && rnd(stepSalt + 91) % 3 == 0) room = vv.debtCap > debt ? vv.debtCap - debt : 0;
        return room;
    }

    function _freeColl(IMMRouter.VenueView memory vv, uint256 p) internal view returns (uint256) {
        (bool okd, bytes memory rd) = vv.account.staticcall(abi.encodeCall(R3Account.debtOf, (vv.id)));
        uint256 debt = okd ? abi.decode(rd, (uint256)) : 0;
        (,, uint128 c) = R3Morpho(MORPHO).position(vv.id, vv.account);
        if (debt == 0 || p == 0) return c;
        (uint64 pin,,) = IMMRouter(ROUTER).pin(POOL);
        uint256 scale = vv.loanIndex == 0 ? 1e12 : 1;
        uint256 value = Math_mulDivUp(debt, 1e18, pin);
        uint256 req = Math_mulDivUp(value, scale, p);
        return c > req ? c - req : 0;
    }

    // ------------------------------------------------------------------ views
    function _views(uint256) internal view returns (string memory s) {
        uint256[] memory prices = _prices(stepSalt + 100);
        uint256[] memory cins = new uint256[](3);
        cins[0] = 0;
        cins[1] = rlog(stepSalt + 101, 1, 1e8);
        cins[2] = 1e11;
        s = string.concat('{"prices":', arr(prices), ',"collIns":', arr(cins), ',"router":', routerViewsJson(prices, cins));
        uint16 id = uint16(rnd(stepSalt + 102) % 4);
        IMMRouter.VenueView memory vv = IMMRouter(ROUTER).venue(POOL, id);
        R3Market memory m = mkt(vv.id);
        uint256[] memory dB = new uint256[](5);
        uint256[] memory dS = new uint256[](5);
        uint256 u = _unit(vv.loanIndex);
        dB[1] = rlog(stepSalt + 103, 1, 1e12 * u);
        dS[2] = m.totalSupplyAssets;
        dB[2] = 1;
        dS[3] = m.totalSupplyAssets > 0 ? m.totalSupplyAssets - 1 : 0;
        dB[3] = rlog(stepSalt + 104, 0, 1e9 * u);
        dS[4] = rlog(stepSalt + 105, 1, m.totalSupplyAssets);
        dB[4] = rnd(stepSalt + 106) % 4 == 0 ? type(uint256).max - m.totalBorrowAssets + 1 : rlog(stepSalt + 107, 1, type(uint128).max);
        s = string.concat(s, ',"acct":{"venue":', vm.toString(id), ',"v":', accountViewsJson(id, dB, dS), "}}");
    }
}
