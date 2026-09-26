// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import "./FinancingSequenceBase.sol";

interface R3Pool {
    function swap(address, address, uint256, uint256, address, uint256) external returns (uint256, uint256);
    function previewSwap(bool, uint256) external view returns (uint256, uint256, uint256);
    function poolAssetPosition() external view returns (uint256, uint256);
    function loanPosition() external view returns (uint256, uint256, uint256);
    function totalAssets() external view returns (uint256);
    function loanConfig(uint8) external view returns (address, uint8, uint64, uint64, uint256, uint256, uint256);
    function dials() external view returns (uint64, uint64, uint64, uint64, uint64, uint64, uint64, uint32, uint64);
    function limits() external view returns (uint256, uint64, uint32, uint64, uint64, uint16, uint96, uint16);
    function switches() external view returns (uint256, uint32, bool);
}

interface R3Feed {
    function peekCross(address, address) external view returns (bool, uint256, uint48);
    function config(address) external view returns (address, uint32, uint64, uint64, uint64);
}

interface R3Agg {
    function latestRoundData() external view returns (uint80, int256, uint256, uint256, uint80);
}

/// @dev Stateful sequence generator for the swap settlement: one long pseudo-random sequence of REAL pool.swap
///      transactions through the deployed c104 pool (FLAMMSwapLib settleSell / the buy branch with anchor, takeLoan,
///      strict reclaim, releaseExcess and the entry gate) on a Base fork, each its own transaction (--isolate).
///      Between swaps: time warps (the Chainlink answers re-stamped so the feed stays fresh), cbBTC price moves,
///      third-party Morpho activity on the live market, IRM outages, Morpho-oracle band moves, lowered or inflated
///      tracked physical balances, reserve-target and lending-feature changes, and venue / loan caps, rate ceilings
///      and the global pause. Each step records the pool ledger and the Router record before and after, the
///      preview, the swap result and the pool / Router views.
contract SwapSettlementSequences is FinancingSequenceBase {
    uint256 internal constant STEPS = 600;
    bytes32 internal constant LOC = 0x5b7e76949cacd5346234367c3806fe494a22f183af782d834d5fc4ee5b0f4500;
    address internal constant SAFE = 0xeb765F3184f705F5292679f42beEDBbe5D272c49;
    address internal constant TAKER = address(0x7A4E7A4E);

    R3Params internal live;
    uint256 internal nV = 1;
    bytes32[3] internal vids;
    R3Params[3] internal vps;
    R3SetOracle[3] internal vor;
    bool[3] internal vIrmDown;
    uint256[3] internal vOffBps; // created oracles sit at cross * 1e24 * (10000 + off) / 10000
    address internal aggBtc;
    address internal aggUsd;
    int256 internal btcAnswer;
    int256 internal usdAnswer;
    uint80 internal btcRound;
    uint80 internal usdRound;
    uint8 internal oracleMode; // 0 real, 1 revert, 2 zero, 3 price
    uint256 internal oraclePx;
    uint256 internal stepSalt;
    uint256 internal loanBase;
    bool internal lastSell;

    function _applyMocks() internal {
        vm.clearMockedCalls();
        vm.mockCall(aggBtc, abi.encodeWithSelector(R3Agg.latestRoundData.selector), abi.encode(btcRound, btcAnswer, block.timestamp - 5, block.timestamp - 5, btcRound));
        vm.mockCall(aggUsd, abi.encodeWithSelector(R3Agg.latestRoundData.selector), abi.encode(usdRound, usdAnswer, block.timestamp - 5, block.timestamp - 5, usdRound));
        for (uint256 v; v < nV; ++v) {
            if (!vIrmDown[v] || vps[v].irm == address(0)) continue;
            vm.mockCallRevert(IRM, abi.encodeWithSelector(R3Irm.borrowRateView.selector, vps[v]), "irm down");
            vm.mockCallRevert(IRM, abi.encodeWithSelector(R3Irm.borrowRate.selector, vps[v]), "irm down");
        }
        if (oracleMode == 1) vm.mockCallRevert(LIVE_ORACLE, abi.encodeWithSelector(R3Oracle.price.selector), "oracle down");
        if (oracleMode == 2) vm.mockCall(LIVE_ORACLE, abi.encodeWithSelector(R3Oracle.price.selector), abi.encode(uint256(0)));
        if (oracleMode == 3) vm.mockCall(LIVE_ORACLE, abi.encodeWithSelector(R3Oracle.price.selector), abi.encode(oraclePx));
    }

    function setUpSeq(uint256 seed_) internal {
        seed = seed_;
        live = params(LIVE_ID);
        vids[0] = LIVE_ID;
        vps[0] = live;
        (aggBtc,,,,) = R3Feed(FEED).config(CBBTC);
        (aggUsd,,,,) = R3Feed(FEED).config(USDC);
        (btcRound, btcAnswer,,,) = R3Agg(aggBtc).latestRoundData();
        (usdRound, usdAnswer,,,) = R3Agg(aggUsd).latestRoundData();
        _applyMocks();
        (, uint256 feat,) = _switches();
        require(uint256(vm.load(POOL, bytes32(uint256(LOC) + 23))) == feat, "features slot");
        loanBase = uint256(keccak256(abi.encode(uint256(LOC) + 11)));
        (,,,,, uint256 reserve, uint256 liquid) = R3Pool(POOL).loanConfig(0);
        require(uint256(vm.load(POOL, bytes32(loanBase + 4))) == reserve && uint256(vm.load(POOL, bytes32(loanBase + 5))) == liquid, "loan slots");
        (uint256 phys,) = R3Pool(POOL).poolAssetPosition();
        require(uint256(vm.load(POOL, bytes32(uint256(LOC) + 12))) == phys, "physical slot");

        deal(USDC, WHALE, 2_000_000_000e6);
        deal(CBBTC, WHALE, 100_000e8);
        vm.startPrank(WHALE);
        R3ERC20(USDC).approve(MORPHO, type(uint256).max);
        R3ERC20(CBBTC).approve(MORPHO, type(uint256).max);
        R3Morpho(MORPHO).supplyCollateral(live, 20_000e8, WHALE, "");
        R3Morpho(MORPHO).supply(live, 20_000_000e6, 0, WHALE, "");
        vm.stopPrank();
        vm.startPrank(TAKER);
        R3ERC20(USDC).approve(POOL, type(uint256).max);
        R3ERC20(CBBTC).approve(POOL, type(uint256).max);
        vm.stopPrank();
    }

    function _switches() internal view returns (uint256 a, uint256 feat, bool c) {
        (feat,, c) = R3Pool(POOL).switches();
        a = 0;
    }

    function test_swapSeqA() public {
        _run(0xA11CE, "swap_settlement_sequence_a.jsonl", false, false);
    }

    function test_swapSeqB() public {
        _run(0xB0B, "swap_settlement_sequence_b.jsonl", true, false);
    }

    function test_swapSeqC() public {
        _run(0xC0C0, "swap_settlement_sequence_c.jsonl", true, true);
    }

    /// @dev Scripted: a debt-free venue holding posted collateral while its IRM is down. Inside the grace a buy's
    ///      releaseExcess reaches it and reclaimBestEffort reverts inside the try (swallowed, nothing released);
    ///      past the grace the venue is quarantined (entry gate's quarantined frame, no release from it); back up,
    ///      the release goes through. Pool-side Router calls and market touches between swaps are recorded as
    ///      `poolcall` / `tp` events.
    function test_swapScenarioRelease() public {
        vm.createSelectFork(RPC, BLOCK);
        setUpSeq(0xE1E);
        _addVenues();
        deal(CBBTC, POOL, R3ERC20(CBBTC).balanceOf(POOL) + 50e8);
        (uint256 phys,) = R3Pool(POOL).poolAssetPosition();
        vm.store(POOL, bytes32(uint256(LOC) + 12), bytes32(phys + 1e8));
        vm.store(POOL, bytes32(loanBase + 4), bytes32(uint256(1e15)));
        open("swap_settlement_sequence_e.jsonl");
        uint256 i;
        uint256[4] memory amounts = [uint256(2e7), 5e6, 3e7, 1e6];
        for (uint256 round; round < 4; ++round) {
            this.stepCustom(i++, 1, amounts[round], false); // post on venue 1 then a buy: released at once
            vm.warp(block.timestamp + 5);
            _applyMocks();
            this.stepCustom(i++, 4, 0, true); // touch venue 1's market (accrues), then a buy
            vIrmDown[1] = true;
            vm.warp(block.timestamp + 600);
            _applyMocks();
            this.stepCustom(i++, 1, amounts[round], false); // post inside the grace, then a buy: the release reverts in the try
            this.stepCustom(i++, 0, 6e6, false);
            this.stepCustom(i++, 2, 40_000, false); // a sell inside the grace
            this.stepCustom(i++, 0, 5e6, false); // a buy repaying venue 0 (writes its transient repay snapshot)
            this.stepCustom(i++, 5, 1, false); // next transaction: proportional withdraw finds no snapshot
            vm.warp(block.timestamp + 3700);
            _applyMocks();
            this.stepCustom(i++, 0, 9e6, false); // quarantined
            this.stepCustom(i++, 2, 30_000, false);
            vIrmDown[1] = false;
            vm.warp(block.timestamp + 30);
            _applyMocks();
            this.stepCustom(i++, 0, 8e6, false); // back up: the release goes through
            this.stepCustom(i++, 3, 400e6 + round * 300e6, false); // borrow on venue 1 as the pool, then a buy
            vm.warp(block.timestamp + 5);
            _applyMocks();
            this.stepCustom(i++, 4, 0, true);
            vIrmDown[1] = true;
            vm.warp(block.timestamp + 300);
            _applyMocks();
            this.stepCustom(i++, 0, 2e6, false); // indebted venue inside the grace
            this.stepCustom(i++, 0, 900e6, false); // a repay reaching venue 1 inside the grace
            vIrmDown[1] = false;
            vm.warp(block.timestamp + 10);
            _applyMocks();
            this.stepCustom(i++, 0, 1_500e6, false); // repays through the cascade
        }
    }

    /// @dev mode 0: a buy of `x` USDC; 1: postCollateral(1, x) as the pool, then a small buy; 2: a sell of `x`
    ///      sats; 3: borrow(1, x) as the pool, then a small buy; 4: a small buy; 5: withdrawCollateral(0, x, 0,
    ///      proportional) as the pool, then a small buy. `touch` first supplies one unit to
    ///      venue 1's market from the third party (an accrual).
    function stepCustom(uint256 i, uint256 mode, uint256 x, bool touch) external {
        stepSalt = (i + 7) * 1000;
        string memory env = "[";
        if (touch) {
            string memory wpre = positionJson(vids[1], WHALE);
            string memory mpre = marketJson(vids[1]);
            string memory rpre = q(uint256(R3Irm(IRM).rateAtTarget(vids[1])));
            vm.prank(WHALE);
            (bool okt, bytes memory rett) = MORPHO.call(abi.encodeCall(R3Morpho.supply, (vps[1], 1, 0, WHALE, "")));
            env = string.concat(env, '{"kind":"tp","venue":1,"op":0,"a":"1","s":"0","who":"whale","wpre":', wpre, ',"mpre":', mpre);
            env = string.concat(env, ',"rpre":', rpre, ',"res":', res(okt, rett), ',"wpost":', positionJson(vids[1], WHALE));
            env = string.concat(env, ',"mpost":', marketJson(vids[1]), ',"rpost":', q(uint256(R3Irm(IRM).rateAtTarget(vids[1]))), "}");
        }
        if (mode == 1 || mode == 3 || mode == 5) {
            (, uint256 p,) = R3Feed(FEED).peekCross(CBBTC, USDC);
            bytes memory data = mode == 1
                ? abi.encodeCall(IMMRouter.postCollateral, (uint16(1), x))
                : mode == 3
                    ? abi.encodeCall(IMMRouter.borrow, (uint16(1), x, POOL, p))
                    : abi.encodeCall(IMMRouter.withdrawCollateral, (uint16(0), x, 0, true));
            vm.prank(POOL);
            R3ERC20(CBBTC).approve(ROUTER, type(uint256).max);
            vm.prank(POOL);
            (bool ok, bytes memory ret) = ROUTER.call(data);
            vm.prank(POOL);
            R3ERC20(CBBTC).approve(ROUTER, 0);
            env = string.concat(env, touch ? "," : "", '{"kind":"poolcall","data":', qh(data), ',"res":', res(ok, ret), "}");
            x = 3e6;
        }
        if (mode == 4) x = 2e6;
        _swapStep(i, string.concat(env, "]"), mode == 2, x);
    }

    function test_swapSeqD() public {
        _run(0xD00D, "swap_settlement_sequence_d.jsonl", false, true);
    }

    /// @dev Two more USDC venues behind the live one: a created 77% market with a 10% fee and a created 91.5% market
    ///      with no rate model, both on settable oracles that track the feed's cross; borrow order 1,0,2.
    function _addVenues() internal {
        (, uint256 p,) = R3Feed(FEED).peekCross(CBBTC, USDC);
        address[3] memory irms = [IRM, IRM, address(0)];
        uint256[3] memory lltvs = [uint256(0), 0.77e18, 0.915e18];
        vm.startPrank(WHALE);
        for (uint256 v = 1; v < 3; ++v) {
            vor[v] = new R3SetOracle();
            vor[v].set(p * 1e24, 0);
            vps[v] = R3Params(USDC, CBBTC, address(vor[v]), irms[v], lltvs[v]);
            vids[v] = keccak256(abi.encode(vps[v]));
            R3Morpho(MORPHO).createMarket(vps[v]);
            R3Morpho(MORPHO).supply(vps[v], 5_000_000e6, 0, WHALE, "");
            R3Morpho(MORPHO).supplyCollateral(vps[v], 1_000e8, WHALE, "");
            R3Morpho(MORPHO).borrow(vps[v], 3_000_000e6, 0, WHALE, WHALE);
        }
        vm.stopPrank();
        vm.prank(R3Morpho(MORPHO).owner());
        R3Morpho(MORPHO).setFee(vps[1], 0.1e18);
        for (uint256 v = 1; v < 3; ++v) {
            vm.prank(POOL);
            IMMRouter(ROUTER).addVenue(0, 0, abi.encode(vps[v]), true, true, p);
            vm.prank(POOL);
            IMMRouter(ROUTER).setVenueCaps(uint16(v), 0, 0, v == 1 ? 7386586395 : 0);
        }
        uint16[] memory bo = new uint16[](3);
        (bo[0], bo[1], bo[2]) = (1, 0, 2);
        uint16[] memory so = new uint16[](3);
        (so[0], so[1], so[2]) = (2, 1, 0);
        uint16[] memory ro = new uint16[](3);
        (ro[0], ro[1], ro[2]) = (0, 2, 1);
        uint16[] memory wo = new uint16[](3);
        (wo[0], wo[1], wo[2]) = (1, 2, 0);
        vm.prank(POOL);
        IMMRouter(ROUTER).setPriorities(bo, so, wo, ro);
        nV = 3;
    }

    function _syncOracles() internal {
        (bool ok, uint256 p,) = R3Feed(FEED).peekCross(CBBTC, USDC);
        for (uint256 v = 1; v < nV; ++v) {
            uint8 mode = vor[v].mode();
            vor[v].set((ok ? p : 788432322552395) * 1e24 / 10000 * (10000 + vOffBps[v]), mode);
        }
    }

    function _run(uint256 seed_, string memory name, bool inflate, bool multi) internal {
        vm.createSelectFork(RPC, BLOCK);
        setUpSeq(seed_);
        if (multi) _addVenues();
        if (inflate) {
            deal(CBBTC, POOL, R3ERC20(CBBTC).balanceOf(POOL) + 3e8);
            (uint256 phys,) = R3Pool(POOL).poolAssetPosition();
            vm.store(POOL, bytes32(uint256(LOC) + 12), bytes32(phys + 3e8));
        }
        open(name);
        for (uint256 i; i < STEPS; ++i) {
            uint256 r = rnd(2_000_000 + i) % 100;
            uint256 dt = r < 40 ? 0 : r < 55 ? 1 : r < 70 ? 12 : r < 85 ? 120 : r < 92 ? 1800 : r < 96 ? 3601 : 7200;
            vm.warp(block.timestamp + dt);
            _applyMocks();
            this.step(i);
        }
    }

    // ------------------------------------------------------------------ state dumps
    function poolJson() internal view returns (string memory s) {
        (uint256 phys,) = R3Pool(POOL).poolAssetPosition();
        (,,,,, uint256 reserve, uint256 liquid) = R3Pool(POOL).loanConfig(0);
        (uint64 phi, uint64 ltv,,,,,,,) = R3Pool(POOL).dials();
        (, uint64 eps,,,,,,) = R3Pool(POOL).limits();
        (, uint256 feat,) = _switches();
        (bool ok, uint256 p,) = R3Feed(FEED).peekCross(CBBTC, USDC);
        s = string.concat('{"physical":', q(phys), ',"loans":[{"scale":"1000000000000","liquid":', q(liquid), ',"reserveTarget":', q(reserve), "}]");
        s = string.concat(s, ',"ltvWad":', q(ltv), ',"phiWad":', q(phi), ',"roomEpsilonWad":', q(eps), ',"features":', q(feat));
        s = string.concat(s, ',"priceWad":[', q(ok ? p : 0), '],"crossWad":["1000000000000000000"]}');
    }

    function stateJson() internal view returns (string memory) {
        return string.concat('{"pool":', poolJson(), ',"router":', routerJson(), "}");
    }

    // ------------------------------------------------------------------ one transaction
    function step(uint256 i) external {
        stepSalt = (i + 1) * 1000;
        string memory env = _env();
        bool sell = rnd(stepSalt + 50) % 10 < 7 ? !lastSell : lastSell;
        _swapStep(i, env, sell, _size(sell));
    }

    function _swapStep(uint256 i, string memory env, bool sell, uint256 amountIn) internal {
        string memory pre = stateJson();
        (bool pok, bytes memory pret) = POOL.staticcall(abi.encodeCall(R3Pool.previewSwap, (sell, amountIn)));
        (address tin, address tout) = sell ? (CBBTC, USDC) : (USDC, CBBTC);
        deal(tin, TAKER, amountIn);
        vm.prank(TAKER);
        (bool ok, bytes memory ret) = POOL.call(abi.encodeCall(R3Pool.swap, (tin, tout, amountIn, 0, TAKER, block.timestamp)));
        if (ok) lastSell = sell;
        string memory s = string.concat('{"i":', vm.toString(i), ',"t":', q(block.timestamp), ',"env":', env, ',"pre":', pre);
        s = string.concat(s, ',"op":{"sell":', qb(sell), ',"amountIn":', q(amountIn), '},"preview":', res(pok, pret), ',"res":', res(ok, ret));
        s = string.concat(s, ',"post":', stateJson(), ',"views":', _views(), "}");
        line(s);
    }

    function _previewOk(bool sell, uint256 x) internal view returns (bool ok) {
        (ok,) = POOL.staticcall(abi.encodeCall(R3Pool.previewSwap, (sell, x)));
    }

    /// @dev Preview-guided size: the largest previewable amount on a doubling-then-bisection search from a small
    ///      seed, then that edge, a uniform draw below it, or an unguided log draw.
    function _size(bool sell) internal view returns (uint256) {
        uint256 r = rnd(stepSalt + 51) % 10;
        uint256 blind = sell ? rlog(stepSalt + 52, 1, 3e8) : rlog(stepSalt + 52, 1, 300_000e6);
        if (r < 2) return blind;
        uint256 lo = sell ? 50 : 50_000;
        uint256 n;
        while (!_previewOk(sell, lo) && n < 8) {
            lo *= 4;
            ++n;
        }
        if (!_previewOk(sell, lo)) return blind;
        uint256 hi = lo * 2;
        n = 0;
        while (_previewOk(sell, hi) && n < 40) {
            lo = hi;
            hi *= 2;
            ++n;
        }
        for (uint256 k; k < 40 && hi > lo + 1; ++k) {
            uint256 mid = (lo + hi) / 2;
            if (_previewOk(sell, mid)) lo = mid;
            else hi = mid;
        }
        uint256 e = rnd(stepSalt + 54) % 20;
        if (e == 0) return lo;
        if (e < 4) return lo / 2 + rnd(stepSalt + 53) % (lo / 2 + 1);
        return lo / 50 + rnd(stepSalt + 53) % (lo / 6 + 1);
    }

    function _views() internal view returns (string memory s) {
        (bool ok, uint256 p,) = R3Feed(FEED).peekCross(CBBTC, USDC);
        uint256[] memory prices = new uint256[](1);
        prices[0] = ok ? p : 0;
        (uint256 phys,) = R3Pool(POOL).poolAssetPosition();
        uint256[] memory cins = new uint256[](2);
        cins[0] = phys;
        cins[1] = phys + rlog(stepSalt + 90, 1, 3e8);
        s = string.concat('{"prices":', arr(prices), ',"collIns":', arr(cins), ',"router":', routerViewsJson(prices, cins));
        s = string.concat(s, ',"totalAssets":', sv(POOL, abi.encodeCall(R3Pool.totalAssets, ())));
        s = string.concat(s, ',"loanPosition":', sv(POOL, abi.encodeCall(R3Pool.loanPosition, ())));
        s = string.concat(s, ',"poolAssetPosition":', sv(POOL, abi.encodeCall(R3Pool.poolAssetPosition, ())), "}");
    }

    // ------------------------------------------------------------------ environment
    function _env() internal returns (string memory s) {
        s = "[";
        uint256 r = rnd(stepSalt + 1) % 100;
        if (r < 14) s = string.concat(s, _envThirdParty());
        else if (r < 19) s = string.concat(s, _envIrm());
        else if (r < 28) s = string.concat(s, _envPrice());
        else if (r < 33) s = string.concat(s, _envOracle());
        else if (r < 39) s = string.concat(s, _envPhysical());
        else if (r < 43) s = string.concat(s, _envReserve());
        else if (r < 46) s = string.concat(s, _envFeature());
        else if (r < 53) s = string.concat(s, _envConfig());
        s = string.concat(s, "]");
    }

    function _envIrm() internal returns (string memory) {
        uint256 v = rnd(stepSalt + 60) % (nV == 1 ? 1 : 2);
        vIrmDown[v] = !vIrmDown[v];
        _applyMocks();
        return string.concat('{"kind":"irm","venue":', vm.toString(v), ',"down":', qb(vIrmDown[v]), "}");
    }

    function _envPrice() internal returns (string memory) {
        int256[8] memory bps = [int256(-300), -200, -50, -10, 10, 50, 200, 300];
        int256 b = bps[rnd(stepSalt + 2) % 8];
        btcAnswer = btcAnswer * (10000 + b) / 10000;
        btcRound += 1;
        bool follow = rnd(stepSalt + 3) % 2 == 0;
        if (follow && oracleMode == 3) oraclePx = uint256(int256(oraclePx) * (10000 + b) / 10000);
        _applyMocks();
        if (follow) _syncOracles();
        return string.concat('{"kind":"price","bps":', vm.toString(b), "}");
    }

    function _envOracle() internal returns (string memory) {
        if (nV > 1 && rnd(stepSalt + 61) % 2 == 0) {
            uint256 v = 1 + rnd(stepSalt + 62) % 2;
            uint256 mm = rnd(stepSalt + 63) % 6;
            vOffBps[v] = mm == 0 ? 0 : mm == 1 ? 200 : mm == 2 ? 300 : 0;
            (bool okp, uint256 px,) = R3Feed(FEED).peekCross(CBBTC, USDC);
            uint256 e0 = (okp ? px : 788432322552395) * 1e24;
            uint256 o = mm == 1 ? e0 / 100 * 102 : mm == 2 ? e0 / 100 * 103 : mm == 3 ? e0 / 100 * 98 : e0;
            vor[v].set(o, mm == 4 ? 1 : mm == 5 ? 2 : 0);
            return string.concat('{"kind":"oracle","venue":', vm.toString(v), ',"mode":', vm.toString(mm), ',"price":', q(o), "}");
        }
        uint256 m = rnd(stepSalt + 4) % 7;
        (bool ok, uint256 p,) = R3Feed(FEED).peekCross(CBBTC, USDC);
        uint256 e = ok ? p * 1e24 : 7.88e38;
        oracleMode = m == 0 ? 0 : m == 1 ? 1 : m == 2 ? 2 : 3;
        oraclePx = m == 3 ? e / 100 * 98 : m == 4 ? e / 100 * 102 : m == 5 ? e / 100 * 97 : e;
        _applyMocks();
        return string.concat('{"kind":"oracle","mode":', vm.toString(m), ',"price":', q(oraclePx), "}");
    }

    function _envPhysical() internal returns (string memory) {
        (uint256 phys,) = R3Pool(POOL).poolAssetPosition();
        uint256 np;
        if (rnd(stepSalt + 5) % 2 == 0) {
            np = phys / (2 + rnd(stepSalt + 6) % 20);
        } else {
            uint256 add = rlog(stepSalt + 6, 1, 1e8);
            deal(CBBTC, POOL, R3ERC20(CBBTC).balanceOf(POOL) + add);
            np = phys + add;
        }
        vm.store(POOL, bytes32(uint256(LOC) + 12), bytes32(np));
        return string.concat('{"kind":"phys","physical":', q(np), "}");
    }

    function _envReserve() internal returns (string memory) {
        uint256 x = rnd(stepSalt + 7) % 3 == 0 ? 0 : rlog(stepSalt + 8, 1, 100e6);
        vm.store(POOL, bytes32(loanBase + 4), bytes32(x));
        return string.concat('{"kind":"reserve","v":', q(x), "}");
    }

    function _envFeature() internal returns (string memory) {
        uint256 f = uint256(vm.load(POOL, bytes32(uint256(LOC) + 23))) ^ (1 << 3);
        vm.store(POOL, bytes32(uint256(LOC) + 23), bytes32(f));
        return string.concat('{"kind":"features","v":', q(f), "}");
    }

    function _envThirdParty() internal returns (string memory s) {
        uint256 kind = rnd(stepSalt + 10) % 8;
        uint256 v = rnd(stepSalt + 64) % nV;
        R3Params memory live = vps[v];
        bytes32 LIVE_ID = vids[v];
        R3Market memory m = mkt(LIVE_ID);
        (uint256 wSup, uint128 wBor,) = R3Morpho(MORPHO).position(LIVE_ID, WHALE);
        uint256 cash = m.totalSupplyAssets > m.totalBorrowAssets ? m.totalSupplyAssets - m.totalBorrowAssets : 0;
        uint256 a;
        uint256 sh;
        address who = WHALE;
        bytes memory data;
        if (kind == 0) {
            a = rlog(stepSalt + 11, 1, 60_000_000e6);
            data = abi.encodeCall(R3Morpho.supply, (live, a, 0, WHALE, ""));
        } else if (kind == 1) {
            if (rnd(stepSalt + 12) % 2 == 0) {
                sh = rlog(stepSalt + 13, 1, wSup);
                data = abi.encodeCall(R3Morpho.withdraw, (live, 0, sh, WHALE, WHALE));
            } else {
                a = rlog(stepSalt + 13, 1, cash);
                data = abi.encodeCall(R3Morpho.withdraw, (live, a, 0, WHALE, WHALE));
            }
        } else if (kind == 2) {
            uint256 r = rnd(stepSalt + 14) % 4;
            a = r == 0 ? cash : r == 1 ? (cash > 5e6 ? cash - rnd(stepSalt + 15) % 5e6 : cash) : rlog(stepSalt + 15, 1, cash);
            data = abi.encodeCall(R3Morpho.borrow, (live, a, 0, WHALE, WHALE));
        } else if (kind == 3) {
            if (rnd(stepSalt + 16) % 3 == 0) {
                sh = rlog(stepSalt + 17, 1, wBor);
                data = abi.encodeCall(R3Morpho.repay, (live, 0, sh, WHALE, ""));
            } else {
                a = rlog(stepSalt + 17, 1, wBor == 0 ? 1 : m.totalBorrowAssets / 20);
                data = abi.encodeCall(R3Morpho.repay, (live, a, 0, WHALE, ""));
            }
        } else if (kind >= 5) {
            who = IMMRouter(ROUTER).venue(POOL, uint16(v)).account;
            (, uint128 aBor,) = R3Morpho(MORPHO).position(LIVE_ID, who);
            if (kind == 5) {
                a = rlog(stepSalt + 19, 1, 50_000e6 * 1);
                data = abi.encodeCall(R3Morpho.supply, (live, a, 0, who, ""));
            } else if (kind == 6) {
                a = rlog(stepSalt + 19, 1, 1e7);
                data = abi.encodeCall(R3Morpho.supplyCollateral, (live, a, who, ""));
            } else if (rnd(stepSalt + 18) % 2 == 0) {
                sh = rlog(stepSalt + 19, 1, aBor);
                data = abi.encodeCall(R3Morpho.repay, (live, 0, sh, who, ""));
            } else {
                a = rlog(stepSalt + 19, 1, 20e6 * 1);
                data = abi.encodeCall(R3Morpho.repay, (live, a, 0, who, ""));
            }
        } else {
            a = rlog(stepSalt + 18, 1, 1_000e8);
            data = abi.encodeCall(R3Morpho.supplyCollateral, (live, a, WHALE, ""));
        }
        string memory wpre = positionJson(LIVE_ID, who);
        string memory mpre = marketJson(LIVE_ID);
        string memory rpre = q(live.irm == address(0) ? 0 : uint256(R3Irm(IRM).rateAtTarget(LIVE_ID)));
        vm.prank(WHALE);
        (bool ok, bytes memory ret) = MORPHO.call(data);
        s = string.concat('{"kind":"tp","venue":', vm.toString(v), ',"op":', vm.toString(kind), ',"a":', q(a), ',"s":', q(sh), ',"who":', qs(who == WHALE ? "whale" : "account"));
        s = string.concat(s, ',"wpre":', wpre, ',"mpre":', mpre, ',"rpre":', rpre, ',"res":', res(ok, ret));
        s = string.concat(s, ',"wpost":', positionJson(LIVE_ID, who), ',"mpost":', marketJson(LIVE_ID));
        s = string.concat(s, ',"rpost":', q(live.irm == address(0) ? 0 : uint256(R3Irm(IRM).rateAtTarget(LIVE_ID))), "}");
    }

    function _envConfig() internal returns (string memory s) {
        uint256 k = rnd(stepSalt + 20) % 4;
        bytes memory data;
        if (k == 0) {
            uint16 cid = uint16(rnd(stepSalt + 65) % nV);
            IMMRouter.VenueView memory vv = IMMRouter(ROUTER).venue(POOL, cid);
            (bool okd, bytes memory rd) = vv.account.staticcall(abi.encodeCall(R3Account.debtOf, (vv.id)));
            uint256 debt = okd ? abi.decode(rd, (uint256)) : 0;
            uint256 c = rnd(stepSalt + 21) % 3;
            uint128 debtCap = c == 0 ? 1.5e12 : uint128(debt + rlog(stepSalt + 22, 0, 200e6));
            uint128 supplyCap = rnd(stepSalt + 23) % 3 == 0 ? 2e12 : uint128(rlog(stepSalt + 24, 1, 500e6));
            uint64 maxRate = 7386586395;
            uint256 cr = rnd(stepSalt + 25) % 4;
            if (cr >= 1) {
                uint256 dB = rlog(stepSalt + 26, 0, 1e11);
                (bool okr, bytes memory rr) = vv.account.staticcall(abi.encodeCall(R3Account.borrowRateAfter, (vv.id, dB, 0)));
                (, uint256 rate) = okr ? abi.decode(rr, (bool, uint256)) : (false, uint256(0));
                maxRate = uint64(rate + (cr == 1 ? 0 : cr == 2 ? 1 : 1e6));
            }
            data = abi.encodeWithSignature("setVenueCaps(uint16,uint128,uint128,uint64)", cid, debtCap, supplyCap, maxRate);
        } else if (k == 1) {
            (bool okp, bytes memory rp) = ROUTER.staticcall(abi.encodeCall(IMMRouter.position, (POOL, uint8(0))));
            (, uint256 sup, uint256 debt) = okp ? abi.decode(rp, (uint256, uint256, uint256)) : (0, 0, 0);
            uint128 debtCap = rnd(stepSalt + 27) % 2 == 0 ? 0 : uint128(debt + rlog(stepSalt + 28, 0, 100e6));
            uint128 supplyCap = rnd(stepSalt + 29) % 2 == 0 ? 0 : uint128(sup + rlog(stepSalt + 30, 0, 100e6));
            data = abi.encodeWithSignature("setLoanCaps(uint8,uint128,uint128,bool)", uint8(0), debtCap, supplyCap, rnd(stepSalt + 31) % 6 != 0);
        } else if (k == 2) {
            bool p = rnd(stepSalt + 32) % 3 == 0;
            vm.prank(SAFE);
            IMMRouter(ROUTER).setGlobalPaused(p);
            return string.concat('{"kind":"cfg","what":"pause","v":', qb(p), "}");
        } else {
            data = abi.encodeWithSignature("setVenueFlags(uint16,bool,bool)", uint16(rnd(stepSalt + 66) % nV), rnd(stepSalt + 33) % 5 != 0, rnd(stepSalt + 34) % 5 != 0);
        }
        vm.prank(POOL);
        (bool ok, bytes memory ret) = ROUTER.call(data);
        s = string.concat('{"kind":"cfg","data":', qh(data), ',"res":', res(ok, ret), "}");
    }
}
