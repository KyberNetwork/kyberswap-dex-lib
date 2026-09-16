// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import {LevRecorder, LevRecLeverFill} from "test/kyber/LevRecorder.sol";

interface ILevForkPool {
    function core() external view returns (address);
    function setLevPaused(bool p) external;
    function leverUp(uint256 poolAssetIn, uint256 minLoanOut, address to, uint256 deadline)
        external
        returns (uint256 amountInUsed, uint256 loanOut);
    function leverDown(uint256 loanIn, uint256 minPoolOut, address to, uint256 deadline)
        external
        returns (uint256 amountInUsed, uint256 poolOut);
}

interface ILevForkCore {
    function owner() external view returns (address);
    function keeper() external view returns (address);
}

interface ILevForkSpreadHook {
    function setSpread(uint24 newSpread) external;
    function setMaxSpreadAge(uint32 maxSpreadAge_) external;
    function maxSpreadAge() external view returns (uint32);
}

interface ILevForkErc20 {
    function approve(address spender, uint256 amount) external returns (bool);
}

/// @notice Leverage-hook fixtures against the DEPLOYED Base c104 stack at a pinned block.
///         (a) lev_hook_fork_fixture.json: the curator unpauses the venue and the keeper posts a live spread, then
///             the recorder captures, per log-spaced amount and direction, the LeverContext core builds, the swap
///             hook's bookFor and reservation price, the leverage hook's frame and previewLever on that context,
///             and pool.previewLever -- with revert data wherever a call reverts. Scenarios vary the posted spread,
///             let it go stale, and replay a small executed lever-up / lever-down sequence.
///         (b) lev_curve_fork_fixture.json: the deployed CollRebalancerMath library's frozenParams() and a
///             leverageQuote / deleverageQuote / anchorAndBase / isStateSafe grid called on its shipped bytecode.
contract LevForkFixtureTest is LevRecorder {
    uint256 internal constant FORK_BLOCK = 51_317_000;

    address internal constant POOL = 0xc0fdCB1799cCc2CEBaA1fe247157b0dF33D57572;
    address internal constant SWAP_HOOK = 0x65CBD227cBC61248ae77a5fC813A29C54C092134;
    address internal constant LEV_HOOK = 0xE0A98d8e60035832B8BaD7f7af7B9B0b3A7308F3;
    address internal constant SPREAD_HOOK = 0x04988aF54ec88D2de77b191025EAef2fe488f93b;
    address internal constant MATH = 0xc002d0731e6A2E6e80bE754779BCef6B01aFF0BB;
    address internal constant CBBTC = 0xcbB7C0000aB88B473b1f5aFd9ef808440eed33Bf;
    address internal constant USDC = 0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913;

    uint256 internal constant RATIO = 444_444_444_444_444_444;

    function _grid(uint256 scenario) internal {
        uint256 sats = 1;
        for (uint256 i; i < 17; ++i) {
            _levRecord(scenario, POOL, LEV_HOOK, SWAP_HOOK, true, sats, 0);
            sats = i % 2 == 0 ? sats * 3 : (sats * 10) / 3;
        }
        uint256 usd = 1;
        for (uint256 i; i < 21; ++i) {
            _levRecord(scenario, POOL, LEV_HOOK, SWAP_HOOK, false, usd, 0);
            usd = i % 2 == 0 ? usd * 3 : (usd * 10) / 3;
        }
        _levRecord(scenario, POOL, LEV_HOOK, SWAP_HOOK, false, 1, 1);
        _levRecord(scenario, POOL, LEV_HOOK, SWAP_HOOK, false, 1, 777_777_777_777);
        _levRecord(scenario, POOL, LEV_HOOK, SWAP_HOOK, false, 1, 3_141_592_653_589_793_238);
    }

    function test_levForkFixture() public {
        vm.createSelectFork("https://mainnet.base.org", FORK_BLOCK);
        _levInitFields();
        address core = ILevForkPool(POOL).core();
        address curator = ILevForkCore(core).owner();
        address keeper = ILevForkCore(core).keeper();

        // Scenario 0: paused as deployed.
        _levRecord(0, POOL, LEV_HOOK, SWAP_HOOK, true, 1000, 0);
        _levRecord(0, POOL, LEV_HOOK, SWAP_HOOK, false, 1000, 0);

        vm.prank(curator);
        ILevForkPool(POOL).setLevPaused(false);

        // Scenarios 1-3: live posted spreads.
        uint24[3] memory spreads = [uint24(17_500), 5_000, 100_000];
        for (uint256 k; k < 3; ++k) {
            vm.prank(keeper);
            ILevForkSpreadHook(SPREAD_HOOK).setSpread(spreads[k]);
            _grid(1 + k);
        }

        // Scenario 4: the post goes stale (lever-ups fail closed, lever-downs degrade). The staleness window is
        // shortened to one second so the price feed stays fresh.
        uint256 snap = vm.snapshotState();
        vm.prank(curator);
        ILevForkSpreadHook(SPREAD_HOOK).setMaxSpreadAge(1);
        vm.warp(block.timestamp + 2);
        _grid(4);
        _levRevertKeepingRows(snap);

        // Scenario 5: an executed sequence at a fresh 17,500 ppm post. Pre-step rows, then the step.
        vm.prank(keeper);
        ILevForkSpreadHook(SPREAD_HOOK).setSpread(17_500);
        address trader = address(0x7A7A7A);
        deal(CBBTC, trader, 1e8);
        deal(USDC, trader, 1_000_000e6);
        vm.startPrank(trader);
        ILevForkErc20(CBBTC).approve(POOL, type(uint256).max);
        ILevForkErc20(USDC).approve(POOL, type(uint256).max);
        vm.stopPrank();
        uint256[4] memory stepAmt = [uint256(10_000), 3_000_000, 5_000, 1_000_000];
        for (uint256 k; k < 4; ++k) {
            bool up = k % 2 == 0;
            _levRecord(5, POOL, LEV_HOOK, SWAP_HOOK, up, stepAmt[k], 0);
            vm.prank(trader);
            if (up) {
                try ILevForkPool(POOL).leverUp(stepAmt[k], 0, trader, block.timestamp) {} catch {}
            } else {
                try ILevForkPool(POOL).leverDown(stepAmt[k], 0, trader, block.timestamp) {} catch {}
            }
        }
        _levRecord(5, POOL, LEV_HOOK, SWAP_HOOK, true, 1000, 0);
        _levRecord(5, POOL, LEV_HOOK, SWAP_HOOK, false, 1000, 0);

        // Scenario 6: synthetic hook-only contexts on the deployed hook at the post-sequence state.
        _levSynthetic(6, POOL, LEV_HOOK, SWAP_HOOK);

        string[] memory scenarios = new string[](7);
        scenarios[0] = "paused: as deployed";
        scenarios[1] = "spread17500: unpaused, keeper posts 17500 ppm";
        scenarios[2] = "spread5000: keeper posts 5000 ppm";
        scenarios[3] = "spread100000: keeper posts 100000 ppm";
        scenarios[4] = "stale: 100000 ppm post aged past a 1 s maxSpreadAge";
        scenarios[5] = "sequence: fresh 17500 post; pre-step rows for leverUp 10000 sats, leverDown 3 USDC, leverUp 5000 sats, leverDown 1 USDC, then two probes";
        scenarios[6] = "synthetic: hook-only CR ladder and edge contexts on the deployed hook after the sequence";
        _levWrite("lev_fork", "test/kyber/lev_hook_fork_fixture.json", scenarios, FORK_BLOCK);
    }

    // ------------------------------------------------------------------ deployed CollRebalancerMath grid

    uint256[] internal curveRows;

    function _call(bytes memory data) internal view returns (bytes memory ret) {
        bool ok;
        (ok, ret) = MATH.staticcall(data);
        require(ok, "math call reverted");
    }

    function test_levCurveForkFixture() public {
        vm.createSelectFork("https://mainnet.base.org", FORK_BLOCK);
        bytes memory params = _call(abi.encodeWithSignature("frozenParams()"));
        uint256[] memory p = new uint256[](params.length / 32);
        for (uint256 i; i < p.length; ++i) {
            uint256 w;
            assembly {
                w := mload(add(params, add(32, mul(i, 32))))
            }
            p[i] = w;
        }

        uint256[4] memory collaterals = [uint256(1_000), 234_423_000_000_000_000_000, 7e24, 3e37];
        uint256[3] memory prices = [uint256(1e18), 768_302_232_848_967_000, 3_000_000_000_000_000_001];
        // cv/debt in bp; 0 = no debt
        uint256[15] memory crs = [uint256(0), 10_200, 13_000, 15_000, 15_500, 15_600, 17_000, 19_000, 19_900, 20_000, 20_100, 22_000, 25_000, 30_000, 50_000];
        for (uint256 a; a < collaterals.length; ++a) {
            for (uint256 b; b < prices.length; ++b) {
                for (uint256 c; c < crs.length; ++c) {
                    _curveState(collaterals[a], prices[b], crs[c]);
                }
            }
        }
        // A deterministic pseudo-random sweep: magnitudes from dust to past MAX_INPUT, arbitrary prices, CR from
        // below the mark through the recovery band to far above target, arbitrary spreads and fill sizes.
        for (uint256 i; i < 2_500; ++i) {
            _curveRandom(i);
        }
        string memory obj = "lev_curve_fork";
        vm.serializeUint(obj, "block", FORK_BLOCK);
        vm.serializeAddress(obj, "math", MATH);
        vm.serializeUint(obj, "frozenParams", p);
        vm.serializeString(obj, "fields", _curveFields());
        vm.serializeUint(obj, "stride", 9);
        string memory out = vm.serializeUint(obj, "v", curveRows);
        vm.writeJson(out, "test/kyber/lev_curve_fork_fixture.json");
    }

    function _curveState(uint256 coll, uint256 price, uint256 crBp) internal {
        uint256 cv = coll * price / 1e18;
        uint256 debt = crBp == 0 ? 0 : cv * 10_000 / crBp;
        _curveAnchor(coll, debt, price);
        uint256[4] memory spreads = [uint256(0), 2_500, 13_000, 17_500];
        uint256[5] memory fracs = [uint256(1), 100_000, 10_000_000, 100_000_000, 1_000_000_000]; // of 1e9
        for (uint256 s; s < spreads.length; ++s) {
            for (uint256 f; f < fracs.length; ++f) {
                uint256 inL = coll * fracs[f] / 1e9;
                uint256 inD = debt * fracs[f] / 1e9;
                _curveQuotePair(coll, debt, price, spreads[s], inL == 0 ? 1 : inL, inD == 0 ? 1 : inD);
            }
        }
    }

    function _curveAnchor(uint256 coll, uint256 debt, uint256 price) internal {
        (uint256 xa, uint256 bx) = abi.decode(
            _call(abi.encodeWithSignature("anchorAndBase(uint256,uint256,uint256,uint256)", coll, debt, price, RATIO)),
            (uint256, uint256)
        );
        bool safe = abi.decode(
            _call(abi.encodeWithSignature("isStateSafe(uint256,uint256,uint256,uint256,uint256)", coll, debt, price, xa, RATIO)),
            (bool)
        );
        _curveRow(2, coll, debt, price, 0, 0, xa, bx, safe ? 1 : 0);
    }

    function _curveQuotePair(uint256 coll, uint256 debt, uint256 price, uint256 spread, uint256 inL, uint256 inD)
        internal
    {
        (uint256 o1, uint256 n1, uint256 d1) = abi.decode(
            _call(abi.encodeWithSignature("leverageQuote(uint256,uint256,uint256,uint256,uint256,uint256)", coll, debt, price, RATIO, spread, inL)),
            (uint256, uint256, uint256)
        );
        _curveRow(0, coll, debt, price, spread, inL, o1, n1, d1);
        (uint256 o2, uint256 n2, uint256 d2) = abi.decode(
            _call(abi.encodeWithSignature("deleverageQuote(uint256,uint256,uint256,uint256,uint256,uint256)", coll, debt, price, RATIO, spread, inD)),
            (uint256, uint256, uint256)
        );
        _curveRow(1, coll, debt, price, spread, inD, o2, n2, d2);
    }

    function _rnd(uint256 i, uint256 k) internal pure returns (uint256) {
        return uint256(keccak256(abi.encode("lev-curve", i, k)));
    }

    function _curveRandom(uint256 i) internal {
        uint256 coll = (_rnd(i, 0) % 1_000_000 + 1) * 10 ** (_rnd(i, 1) % 36);
        uint256 price = (_rnd(i, 2) % 1_000_000 + 1) * 10 ** (_rnd(i, 3) % 31);
        if (coll > type(uint256).max / price) return;
        uint256 debt = _rnd(i, 4) % 16 == 0 ? 0 : (coll * price / 1e18) * 10_000 / (_rnd(i, 5) % 40_000 + 9_000);
        _curveAnchor(coll, debt, price);
        uint256 frac = _rnd(i, 7) % 1_000_000_000 + 1;
        _curveQuotePair(
            coll,
            debt,
            price,
            _rnd(i, 6) % 8 == 0 ? 13_000 : _rnd(i, 6) % 30_001,
            _rnd(i, 8) % 8 == 0 ? _rnd(i, 9) % 10 + 1 : coll * frac / 1e9,
            _rnd(i, 8) % 8 == 1 ? _rnd(i, 9) % 10 + 1 : debt * frac / 1e9
        );
    }

    function _curveFields() internal pure returns (string[] memory f) {
        f = new string[](9);
        f[0] = "kind (0 leverageQuote, 1 deleverageQuote, 2 anchorAndBase+isStateSafe)";
        f[1] = "collateral";
        f[2] = "debt";
        f[3] = "priceWad";
        f[4] = "spreadPpm";
        f[5] = "amountIn";
        f[6] = "out | xAnchor";
        f[7] = "newCollateral | baseX";
        f[8] = "newDebt | isStateSafe(xAnchor)";
    }

    function _curveRow(uint256 k, uint256 c, uint256 d, uint256 p, uint256 s, uint256 i, uint256 o, uint256 n, uint256 nd)
        internal
    {
        curveRows.push(k);
        curveRows.push(c);
        curveRows.push(d);
        curveRows.push(p);
        curveRows.push(s);
        curveRows.push(i);
        curveRows.push(o);
        curveRows.push(n);
        curveRows.push(nd);
    }
}
