// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import {Test, Vm} from "forge-std/Test.sol";

// Edge generators for the Go pool core, independent of the CoreE2E generators: own interfaces
// (transcribed from c104 @ 80abd43 src/interfaces), own dump, own scenario logic. Views only, plus the three raw
// words the Go reads contract names (FLAMMStore base+22 bits 160..191; MMRouter venue slots +4/+5), whose layout
// this file re-derives from the struct declarations (see _lastLever / _managedWords).

interface VPool {
    struct HookSet {
        address invariantHook;
        address feeHook;
        address recenterHook;
        address controllerHook;
        address leverageHook;
        address spreadHook;
        address loanSwapHook;
    }

    struct Dials {
        uint64 phiWad;
        uint64 ltvWad;
        uint64 phiMinWad;
        uint64 phiMaxWad;
        uint64 ltvMinWad;
        uint64 ltvMaxWad;
        uint64 ltvMaxStepWad;
        uint32 ltvCooldownSec;
        uint64 minStructDistWad;
    }

    struct Limits {
        uint256 depositCapPoolAsset;
        uint64 roomEpsilonWad;
        uint32 maxPriceAgeSec;
        uint64 feeFloorWad;
        uint64 feeCapWad;
        uint16 performanceFeeBp;
        uint96 peakSharePriceWad;
        uint16 maxPerformanceFeeBp;
    }

    function asset() external view returns (address);
    function router() external view returns (address);
    function priceFeed() external view returns (address);
    function core() external view returns (address);
    function loanCount() external view returns (uint256);
    function loanConfig(uint8) external view returns (address, uint8, uint64, uint64, uint256, uint256, uint256);
    function hooks() external view returns (HookSet memory);
    function paused() external view returns (bool);
    function poolAssetPosition() external view returns (uint256, uint256);
    function totalSupply() external view returns (uint256);
    function dials() external view returns (Dials memory);
    function limits() external view returns (Limits memory);
    function switches() external view returns (uint256, uint32, bool);
    function previewSwap(bool, uint256) external view returns (uint256, uint256, uint256);
    function previewLever(bool, uint256) external view returns (uint256, uint256, uint256, uint256);
    function swap(address, address, uint256, uint256, address, uint256) external returns (uint256, uint256);
    function leverUp(uint256, uint256, address, uint256) external returns (uint256, uint256);
    function leverDown(uint256, uint256, address, uint256) external returns (uint256, uint256);
    function setLevPaused(bool) external;
    function setPaused(bool) external;
    function setDials(uint64, uint64) external;
    function setFeeBounds(uint64, uint64) external;
    function setFeatures(uint256) external;
    function setLoanConfig(uint8, uint64, uint64, uint256, uint256) external;
    function setVenueCaps(uint16, uint128, uint128, uint64) external;
    function setVenueFlags(uint16, bool, bool) external;
    function setRiskLimits(uint256, uint256, uint64, uint256, uint8) external;
}

interface VCore {
    function owner() external view returns (address);
    function keeper() external view returns (address);
    function guardian() external view returns (address);
}

interface VHook {
    struct FeeParams {
        uint64 midFeeWad;
        uint64 outFeeWad;
        uint64 gammaWad;
        uint64 sigmaRefWad;
        uint64 volBetaWad;
        uint64 volMinWad;
        uint64 volMaxWad;
        uint64 dirSkewWad;
    }

    struct Tuning {
        FeeParams fee;
        uint64 invSkewKappaWad;
        uint64 invSkewBandWad;
        uint64 emaHalfLife;
        uint64 rvHalfLife;
        uint128 stepDivisorWad;
        uint64 inertiaWad;
        uint64 inertiaMaxWad;
    }

    struct Bounds {
        uint64 minFeeWad;
        uint64 maxFeeWad;
        uint64 minEmaHalfLife;
        uint64 maxEmaHalfLife;
        uint64 maxRvHalfLife;
        uint64 maxStepWad;
        uint128 minStepDivisorWad;
    }

    struct Params {
        uint128 aWad;
        uint128 spanUpWad;
        uint128 spanDnWad;
        uint256 anchorPriceWad;
        uint8 loanDecimals;
        Tuning tuning;
        Bounds bounds;
    }

    struct Support {
        uint256 aWad;
        uint256 xLo;
        uint256 xHi;
        uint256 yHi;
    }

    function params() external view returns (Params memory);
    function support() external view returns (Support memory);
    function anchorSqrtX96() external view returns (uint160);
    function reservationPriceWad() external view returns (uint256);
    function kappa() external view returns (uint256);
    function xWad() external view returns (uint256);
    function reserveStable() external view returns (uint256);
    function idleStable() external view returns (uint256);
    function reserveVolatile() external view returns (uint256);
    function idleVolatile() external view returns (uint256);
    function rvWad() external view returns (uint256);
    function LOAN_SCALE() external view returns (uint256);
}

interface VSpread {
    function spread() external view returns (uint24);
    function minSpread() external view returns (uint24);
    function maxSpread() external view returns (uint24);
    function maxSpreadAge() external view returns (uint32);
    function lastSetTs() external view returns (uint48);
    function setSpread(uint24) external;
    function setMaxSpreadAge(uint32) external;
    function setBounds(uint24, uint24) external;
}

interface VFeed {
    struct Token {
        address aggregator;
        uint32 heartbeat;
        uint64 scale;
        uint64 unit;
        uint64 pegBandWad;
    }

    function SEQUENCER_FEED() external view returns (address);
    function SEQUENCER_GRACE() external view returns (uint256);
    function config(address) external view returns (Token memory);
}

interface VAgg {
    function latestRoundData() external view returns (uint80, int256, uint256, uint256, uint80);
}

interface VRouter {
    struct VenueView {
        address account;
        bytes32 id;
        uint8 kind;
        uint8 loanIndex;
        uint64 lltvWad;
        bool borrowEnabled;
        bool supplyEnabled;
        bool retired;
        uint128 debtCap;
        uint128 supplyCap;
        uint64 maxBorrowRateWad;
    }

    struct LoanView {
        address token;
        uint8 decimals;
        uint256 loanScale;
        uint128 debtCap;
        uint128 supplyCap;
        bool borrowEnabled;
        bool retired;
        address[4] accounts;
    }

    function globalPaused() external view returns (bool);
    function loanCount(address) external view returns (uint256);
    function loan(address, uint8) external view returns (LoanView memory);
    function venueCount(address) external view returns (uint256);
    function venue(address, uint16) external view returns (VenueView memory);
    function priorities(address) external view returns (uint16[] memory, uint16[] memory, uint16[] memory, uint16[] memory);
    function pin(address) external view returns (uint64, uint64, uint64);
    function maxDrawnAssets(address) external view returns (uint8);
    function fundingCeiling(address, uint8, uint256, uint256) external view returns (uint256);
    function protocolSafe() external view returns (address);
    function setGlobalPaused(bool) external;
}

struct VMarketParams {
    address loanToken;
    address collateralToken;
    address oracle;
    address irm;
    uint256 lltv;
}

struct VMarket {
    uint128 totalSupplyAssets;
    uint128 totalSupplyShares;
    uint128 totalBorrowAssets;
    uint128 totalBorrowShares;
    uint128 lastUpdate;
    uint128 fee;
}

interface VMorpho {
    function market(bytes32) external view returns (uint128, uint128, uint128, uint128, uint128, uint128);
    function position(bytes32, address) external view returns (uint256, uint128, uint128);
    function idToMarketParams(bytes32) external view returns (address, address, address, address, uint256);
    function owner() external view returns (address);
    function setFee(VMarketParams memory, uint256) external;
    function supply(VMarketParams memory, uint256, uint256, address, bytes memory) external returns (uint256, uint256);
    function borrow(VMarketParams memory, uint256, uint256, address, address) external returns (uint256, uint256);
    function supplyCollateral(VMarketParams memory, uint256, address, bytes memory) external;
    function withdraw(VMarketParams memory, uint256, uint256, address, address) external returns (uint256, uint256);
    function accrueInterest(VMarketParams memory) external;
}

interface VIrm {
    function rateAtTarget(bytes32) external view returns (int256);
    function borrowRateView(VMarketParams memory, VMarket memory) external view returns (uint256);
}

interface VAccount {
    function oraclePrice(bytes32) external view returns (bool, uint256);
}

interface VOracle {
    function price() external view returns (uint256);
}

interface VERC20 {
    function approve(address, uint256) external returns (bool);
    function balanceOf(address) external view returns (uint256);
}

abstract contract CoreEdgesBase is Test {
    string constant RPC = "https://mainnet.base.org";
    address constant POOL = 0xc0fdCB1799cCc2CEBaA1fe247157b0dF33D57572;
    address constant HOOK = 0x65CBD227cBC61248ae77a5fC813A29C54C092134;
    address constant SPREAD = 0x04988aF54ec88D2de77b191025EAef2fe488f93b;
    address constant ROUTER = 0x19A9b39E6710AAD109C829294b0841F0851c6bB4;
    address constant FEED = 0xbED275459578C87a63F2f50A0b077C720e838816;
    address constant MORPHO = 0xBBBBBbbBBb9cC5e90e3b3Af64bdAF62C37EEFFCb;
    address constant CBBTC = 0xcbB7C0000aB88B473b1f5aFd9ef808440eed33Bf;
    address constant USDC = 0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913;
    uint256 constant STORE = 0x5b7e76949cacd5346234367c3806fe494a22f183af782d834d5fc4ee5b0f4500;

    string internal out;
    uint256 internal rng;

    // ------------------------------------------------------------------ json
    function _u(uint256 v) internal pure returns (string memory) {
        return vm.toString(v);
    }

    function _i(int256 v) internal pure returns (string memory) {
        return vm.toString(v);
    }

    function _b(bool v) internal pure returns (string memory) {
        return v ? "true" : "false";
    }

    function _a(address v) internal pure returns (string memory) {
        return string.concat('"', vm.toString(v), '"');
    }

    function _hex(bytes memory v) internal pure returns (string memory) {
        return string.concat('"', vm.toString(v), '"');
    }

    function _emit(string memory s) internal {
        vm.writeLine(out, s);
    }

    function _next() internal returns (uint256) {
        rng = uint256(keccak256(abi.encode(rng)));
        return rng;
    }

    /// @dev log-uniform-ish integer in [lo, hi].
    function _logRand(uint256 lo, uint256 hi) internal returns (uint256) {
        uint256 r = _next();
        uint256 lz = 0;
        for (uint256 t = lo; t > 1; t >>= 1) lz++;
        uint256 hz = 0;
        for (uint256 t = hi; t > 1; t >>= 1) hz++;
        uint256 e = lz + (r % (hz - lz + 1));
        uint256 base = uint256(1) << e;
        uint256 v = base + ((r >> 16) % base);
        if (v < lo) v = lo;
        if (v > hi) v = hi;
        return v;
    }

    // ------------------------------------------------------------------ raw words (layouts re-derived)
    /// @dev FLAMMStore.S (FLAMMStore.sol:246): slots 0..6 bindings, 7 poolDecimals+invariantHook, 8 feeHook,
    ///      9 recenterHook, 10 controllerHook+4 bools, 11 loans, 12 physical, 13 phi/phiMin/phiMax/ltv, 14 ltvMin/
    ///      ltvMax/ltvMaxStep/cooldown, 15 lastDialMoveTs/minStruct/roomEps/feeFloor, 16 feeCap/maxPriceAge,
    ///      17 depositCap, 18 peak/perf/lastObs/maxPerf/govDelay, 19..21 pending hooks, 22 pendingController(160) +
    ///      lastLeverSpreadPpm(32) + hookSetExecutableAt(48).
    function _lastLever() internal view returns (uint256) {
        uint256 w = uint256(vm.load(POOL, bytes32(STORE + 22)));
        // cross-check the layout on neighbours: slot 12 is the physical balance.
        (uint256 phys,) = VPool(POOL).poolAssetPosition();
        require(uint256(vm.load(POOL, bytes32(STORE + 12))) == phys, "layout:physical");
        return (w >> 160) & 0xffffffff;
    }

    /// @dev MMRouter: slot 0 globalPaused, slot 1 `_pools`; PoolRecord slots 0 asset/pin/maxDrawn, 1 gap/band,
    ///      2 loans, 3 venues; Venue = 6 slots (account; id; kind..debtCap; supplyCap+maxRate; managedColl; managedShares).
    function _managedWords(uint256 i, VRouter.VenueView memory v) internal view returns (uint256 c, uint256 s) {
        uint256 rec = uint256(keccak256(abi.encode(POOL, uint256(1))));
        uint256 base = uint256(keccak256(abi.encode(rec + 3))) + 6 * i;
        require(address(uint160(uint256(vm.load(ROUTER, bytes32(base))))) == v.account, "layout:account");
        require(vm.load(ROUTER, bytes32(base + 1)) == v.id, "layout:id");
        require(uint128(uint256(vm.load(ROUTER, bytes32(base + 3)))) == v.supplyCap, "layout:supplyCap");
        c = uint256(vm.load(ROUTER, bytes32(base + 4)));
        s = uint256(vm.load(ROUTER, bytes32(base + 5)));
    }

    // ------------------------------------------------------------------ dump (flammReads JSON)
    function _loansJson() internal view returns (string memory s) {
        uint256 n = VPool(POOL).loanCount();
        s = "[";
        for (uint256 i; i < n; ++i) {
            (address t, uint8 d, uint64 band, uint64 fl, uint256 mx, uint256 rt, uint256 liq) = VPool(POOL).loanConfig(uint8(i));
            s = string.concat(s, i == 0 ? "" : ",", '{"token":', _a(t), ',"decimals":', _u(d), ',"swapPriceBandWad":', _u(band));
            s = string.concat(s, ',"feeFloorWad":', _u(fl), ',"maxSwapNotional":', _u(mx), ',"reserveTarget":', _u(rt), ',"liquid":', _u(liq), "}");
        }
        s = string.concat(s, "]");
    }

    function _hooksArr() internal view returns (string memory) {
        VPool.HookSet memory h = VPool(POOL).hooks();
        return string.concat(
            "[", _a(h.invariantHook), ",", _a(h.feeHook), ",", _a(h.recenterHook), ",", _a(h.controllerHook), ",",
            string.concat(_a(h.leverageHook), ",", _a(h.spreadHook), ",", _a(h.loanSwapHook), "]")
        );
    }

    function _poolJson() internal view returns (string memory s) {
        VPool p = VPool(POOL);
        (uint256 phys, uint256 gross) = p.poolAssetPosition();
        (uint256 feats,, bool lp) = p.switches();
        VPool.Dials memory d = p.dials();
        VPool.Limits memory l = p.limits();
        s = string.concat('{"asset":', _a(p.asset()), ',"router":', _a(p.router()), ',"priceFeed":', _a(p.priceFeed()));
        s = string.concat(s, ',"paused":', _b(p.paused()), ',"features":', _u(feats), ',"levPaused":', _b(lp), ',"hooks":', _hooksArr());
        s = string.concat(s, ',"physical":', _u(phys), ',"gross":', _u(gross), ',"totalSupply":', _u(p.totalSupply()));
        s = string.concat(s, ',"phiWad":', _u(d.phiWad), ',"ltvWad":', _u(d.ltvWad), ',"roomEpsilonWad":', _u(l.roomEpsilonWad));
        s = string.concat(s, ',"feeFloorWad":', _u(l.feeFloorWad), ',"feeCapWad":', _u(l.feeCapWad), ',"loans":', _loansJson());
        s = string.concat(s, ',"lastLeverSpreadPpm":', _u(_lastLever()), "}");
    }

    function _hookJson() internal view returns (string memory s) {
        VHook h = VHook(HOOK);
        VHook.Params memory p = h.params();
        VHook.FeeParams memory f = p.tuning.fee;
        VHook.Support memory su = h.support();
        s = string.concat('{"aWad":', _u(p.aWad), ',"fee":[', _u(f.midFeeWad), ",", _u(f.outFeeWad), ",", _u(f.gammaWad), ",");
        s = string.concat(s, _u(f.sigmaRefWad), ",", _u(f.volBetaWad), ",", _u(f.volMinWad), ",", _u(f.volMaxWad), ",", _u(f.dirSkewWad), "]");
        s = string.concat(s, ',"invSkewKappaWad":', _u(p.tuning.invSkewKappaWad), ',"invSkewBandWad":', _u(p.tuning.invSkewBandWad));
        s = string.concat(s, ',"support":[', _u(su.aWad), ",", _u(su.xLo), ",", _u(su.xHi), ",", _u(su.yHi), "]");
        s = string.concat(s, ',"anchorSqrtX96":', _u(h.anchorSqrtX96()), ',"reservationPriceWad":', _u(h.reservationPriceWad()));
        s = string.concat(s, ',"kappa":', _u(h.kappa()), ',"xWad":', _u(h.xWad()), ',"reserveStable":', _u(h.reserveStable()));
        s = string.concat(s, ',"idleStable":', _u(h.idleStable()), ',"reserveVolatile":', _u(h.reserveVolatile()));
        s = string.concat(s, ',"idleVolatile":', _u(h.idleVolatile()), ',"rvWad":', _u(h.rvWad()), ',"loanScale":', _u(h.LOAN_SCALE()), "}");
    }

    function _spreadJson() internal view returns (string memory) {
        VSpread h = VSpread(SPREAD);
        return string.concat('{"spread":', _u(h.spread()), ',"maxSpreadAge":', _u(h.maxSpreadAge()), ',"lastSetTs":', _u(h.lastSetTs()), "}");
    }

    function _roundJson(address agg) internal view returns (string memory) {
        if (agg == address(0)) return '{"ok":false}';
        try VAgg(agg).latestRoundData() returns (uint80 id, int256 ans, uint256 st, uint256 up, uint80) {
            return string.concat('{"ok":true,"roundId":', _u(id), ',"answer":', _i(ans), ',"startedAt":', _u(st), ',"updatedAt":', _u(up), "}");
        } catch {
            return '{"ok":false}';
        }
    }

    function _tokenJson(address t) internal view returns (string memory) {
        try VFeed(FEED).config(t) returns (VFeed.Token memory c) {
            return string.concat(
                '{"token":', _a(t), ',"known":true,"heartbeat":', _u(c.heartbeat), ',"scale":', _u(c.scale), ',"unit":', _u(c.unit),
                string.concat(',"pegBandWad":', _u(c.pegBandWad), ',"round":', _roundJson(c.aggregator), "}")
            );
        } catch {
            return string.concat('{"token":', _a(t), ',"known":false}');
        }
    }

    function _feedJson() internal view returns (string memory s) {
        address sq = VFeed(FEED).SEQUENCER_FEED();
        s = string.concat('{"sequencerFeed":', _a(sq), ',"sequencerGrace":', _u(VFeed(FEED).SEQUENCER_GRACE()), ',"sequencer":', _roundJson(sq));
        s = string.concat(s, ',"tokens":[', _tokenJson(VPool(POOL).asset()));
        uint256 n = VPool(POOL).loanCount();
        for (uint256 i; i < n; ++i) {
            (address t,,,,,,) = VPool(POOL).loanConfig(uint8(i));
            s = string.concat(s, ",", _tokenJson(t));
        }
        s = string.concat(s, "]}");
    }

    function _mkt(bytes32 id) internal view returns (VMarket memory m) {
        (m.totalSupplyAssets, m.totalSupplyShares, m.totalBorrowAssets, m.totalBorrowShares, m.lastUpdate, m.fee) = VMorpho(MORPHO).market(id);
    }

    function _params(bytes32 id) internal view returns (VMarketParams memory p) {
        (p.loanToken, p.collateralToken, p.oracle, p.irm, p.lltv) = VMorpho(MORPHO).idToMarketParams(id);
    }

    function _marketJson(bytes32 id) internal view returns (string memory) {
        VMarket memory m = _mkt(id);
        return string.concat(
            '{"tsa":', _u(m.totalSupplyAssets), ',"tss":', _u(m.totalSupplyShares), ',"tba":', _u(m.totalBorrowAssets),
            string.concat(',"tbs":', _u(m.totalBorrowShares), ',"lastUpdate":', _u(m.lastUpdate), ',"fee":', _u(m.fee), "}")
        );
    }

    function _venueTail(VRouter.VenueView memory v) internal view returns (string memory s) {
        VMarketParams memory p = _params(v.id);
        (uint256 ss, uint128 bs, uint128 col) = VMorpho(MORPHO).position(v.id, v.account);
        int256 rat;
        bool readable = true;
        if (p.irm != address(0)) {
            rat = VIrm(p.irm).rateAtTarget(v.id);
            try VIrm(p.irm).borrowRateView(p, _mkt(v.id)) returns (uint256) {} catch { readable = false; }
        }
        (bool ook, uint256 op) = VAccount(v.account).oraclePrice(v.id);
        bool zero;
        if (!ook) {
            try VOracle(p.oracle).price() returns (uint256 x) { zero = x == 0; } catch {}
        }
        s = string.concat(',"market":', _marketJson(v.id), ',"position":{"supplyShares":', _u(ss), ',"borrowShares":', _u(bs));
        s = string.concat(s, ',"collateral":', _u(col), '},"irm":', _a(p.irm), ',"marketLltv":', _u(p.lltv), ',"rateAtTarget":', _i(rat));
        s = string.concat(s, ',"hasIrm":', _b(p.irm != address(0)), ',"irmReadable":', _b(readable), ',"oracleOk":', _b(ook));
        s = string.concat(s, ',"oraclePrice":', _u(op), ',"oracleZero":', _b(zero), "}");
    }

    function _venueJson(uint256 i) internal view returns (string memory s) {
        VRouter.VenueView memory v = VRouter(ROUTER).venue(POOL, uint16(i));
        (uint256 mc, uint256 ms) = _managedWords(i, v);
        s = string.concat('{"kind":', _u(v.kind), ',"loanIndex":', _u(v.loanIndex), ',"lltvWad":', _u(v.lltvWad));
        s = string.concat(s, ',"borrowEnabled":', _b(v.borrowEnabled), ',"supplyEnabled":', _b(v.supplyEnabled), ',"retired":', _b(v.retired));
        s = string.concat(s, ',"debtCap":', _u(v.debtCap), ',"supplyCap":', _u(v.supplyCap), ',"maxBorrowRateWad":', _u(v.maxBorrowRateWad));
        s = string.concat(s, ',"managedCollateral":', _u(mc), ',"managedSupplyShares":', _u(ms), _venueTail(v));
    }

    function _u16(uint16[] memory a) internal pure returns (string memory s) {
        s = "[";
        for (uint256 i; i < a.length; ++i) s = string.concat(s, i == 0 ? "" : ",", _u(a[i]));
        s = string.concat(s, "]");
    }

    function _routerJson() internal view returns (string memory s) {
        VRouter r = VRouter(ROUTER);
        (uint64 pinW, uint64 gap, uint64 band) = r.pin(POOL);
        s = string.concat('{"globalPaused":', _b(r.globalPaused()), ',"pinLtvWad":', _u(pinW), ',"safetyGapWad":', _u(gap));
        s = string.concat(s, ',"oracleBandWad":', _u(band), ',"maxDrawnAssets":', _u(r.maxDrawnAssets(POOL)), ',"loans":[');
        uint256 n = r.loanCount(POOL);
        for (uint256 i; i < n; ++i) {
            VRouter.LoanView memory l = r.loan(POOL, uint8(i));
            s = string.concat(s, i == 0 ? "" : ",", '{"decimals":', _u(l.decimals), ',"loanScale":', _u(l.loanScale), ',"debtCap":', _u(l.debtCap));
            s = string.concat(s, ',"supplyCap":', _u(l.supplyCap), ',"borrowEnabled":', _b(l.borrowEnabled), ',"retired":', _b(l.retired), "}");
        }
        (uint16[] memory bo, uint16[] memory so, uint16[] memory wo, uint16[] memory ro) = r.priorities(POOL);
        s = string.concat(s, '],"borrowOrder":', _u16(bo), ',"supplyOrder":', _u16(so), ',"withdrawOrder":', _u16(wo), ',"repayOrder":', _u16(ro));
        s = string.concat(s, ',"venues":[');
        uint256 nv = r.venueCount(POOL);
        for (uint256 i; i < nv; ++i) s = string.concat(s, i == 0 ? "" : ",", _venueJson(i));
        s = string.concat(s, "]}");
    }

    function _dump() internal view returns (string memory) {
        return string.concat(
            '{"block":', _u(block.number), ',"timestamp":', _u(block.timestamp), ',"pool":', _poolJson(), ',"hook":', _hookJson(),
            string.concat(',"spread":', _spreadJson(), ',"feed":', _feedJson(), ',"router":', _routerJson(), "}")
        );
    }

    // ------------------------------------------------------------------ probes
    function _classOf(bool ok, bytes memory ret) internal pure returns (bytes32) {
        if (ok) return bytes32("ok");
        return keccak256(ret);
    }

    function _pvSwap(bool sell, uint256 a) internal view returns (bool ok, bytes memory ret) {
        (ok, ret) = POOL.staticcall(abi.encodeCall(VPool.previewSwap, (sell, a)));
    }

    function _pvLever(bool up, uint256 a) internal view returns (bool ok, bytes memory ret) {
        (ok, ret) = POOL.staticcall(abi.encodeCall(VPool.previewLever, (up, a)));
    }

    function _probe(bool lever, bool dir, uint256 a) internal view returns (bool ok, bytes memory ret) {
        return lever ? _pvLever(dir, a) : _pvSwap(dir, a);
    }

    function _row(string memory tag, bool lever, bool dir, uint256 a) internal returns (bool ok) {
        bytes memory ret;
        (ok, ret) = _probe(lever, dir, a);
        _emit(
            string.concat(
                '{"k":"', lever ? "lv" : "sw", '","tag":"', tag, '","d":', dir ? "1" : "0", ',"a":', _u(a),
                ok ? string.concat(',"r":', _hex(ret), "}") : string.concat(',"e":', _hex(ret), "}")
            )
        );
    }

    function _fcRow(string memory tag, uint256 coll, uint256 priceWad) internal {
        (bool ok, bytes memory ret) = ROUTER.staticcall(abi.encodeCall(VRouter.fundingCeiling, (POOL, 0, coll, priceWad)));
        _emit(string.concat('{"k":"fc","tag":"', tag, '","c":', _u(coll), ',"p":', _u(priceWad), ok ? ',"r":' : ',"e":', _hex(ret), "}"));
    }

    function _state(string memory tag) internal {
        _emit(string.concat('{"k":"state","tag":"', tag, '","s":', _dump(), "}"));
    }

    function _cls(bool lever, bool dir, uint256 a) internal view returns (bytes32) {
        (bool ok, bytes memory ret) = _probe(lever, dir, a);
        return _classOf(ok, ret);
    }

    /// @dev Bisects [lo, hi] (classes differ at the ends) to adjacent units and records a +-w unit window.
    function _edgeAt(string memory tag, bool lever, bool dir, uint256 lo, uint256 hi, uint256 w) internal {
        bytes32 cL = _cls(lever, dir, lo);
        while (hi - lo > 1) {
            uint256 mid = lo + (hi - lo) / 2;
            if (_cls(lever, dir, mid) == cL) lo = mid;
            else hi = mid;
        }
        uint256 from = lo > w ? lo - w : 1;
        for (uint256 a = from; a <= hi + w; ++a) _row(tag, lever, dir, a);
    }

    /// @dev Every class transition between consecutive probe points, bisected to adjacent units, with a +-w window.
    function _edges(string memory tag, bool lever, bool dir, uint256[] memory pts, uint256 w, uint256 maxEdges) internal {
        uint256 found;
        for (uint256 k = 1; k < pts.length && found < maxEdges; ++k) {
            if (pts[k] <= pts[k - 1]) continue;
            if (_cls(lever, dir, pts[k - 1]) == _cls(lever, dir, pts[k])) continue;
            _edgeAt(tag, lever, dir, pts[k - 1], pts[k], w);
            found++;
        }
    }

    function _geo(uint256 lo, uint256 hi, uint256 n) internal pure returns (uint256[] memory pts) {
        pts = new uint256[](n);
        // geometric-ish: lo * (hi/lo)^(k/(n-1)) via repeated multiply in 1e6 fixed point
        uint256 ratio = 1e6;
        {
            // find r s.t. r^(n-1) ~= hi/lo, by bisection on 1e6..1e7
            uint256 a = 1e6;
            uint256 b = 5e6;
            for (uint256 it; it < 40; ++it) {
                uint256 m = (a + b) / 2;
                uint256 v = lo * 1e6;
                for (uint256 k = 1; k < n && v <= hi * 1e6; ++k) v = v * m / 1e6;
                if (v > hi * 1e6) b = m;
                else a = m;
            }
            ratio = a;
        }
        uint256 x = lo * 1e6;
        for (uint256 k; k < n; ++k) {
            pts[k] = x / 1e6 == 0 ? 1 : x / 1e6;
            x = x * ratio / 1e6;
            if (k > 0 && pts[k] <= pts[k - 1]) pts[k] = pts[k - 1] + 1;
        }
    }

    function _randGrid(string memory tag, bool lever, bool dir, uint256 lo, uint256 hi, uint256 n) internal {
        for (uint256 k; k < n; ++k) _row(tag, lever, dir, _logRand(lo, hi));
    }

    // ------------------------------------------------------------------ roles and moves
    function _curator() internal view returns (address) {
        return VCore(VPool(POOL).core()).owner();
    }

    function _keeper() internal view returns (address) {
        return VCore(VPool(POOL).core()).keeper();
    }

    function _asCurator(bytes memory data) internal returns (bool ok, bytes memory ret) {
        vm.prank(_curator());
        (ok, ret) = POOL.call(data);
    }

    function _aggOf(address token) internal view returns (address) {
        return VFeed(FEED).config(token).aggregator;
    }

    function _mockRound(address agg, uint80 id, int256 ans, uint256 startedAt, uint256 updatedAt) internal {
        vm.mockCall(agg, abi.encodeWithSelector(VAgg.latestRoundData.selector), abi.encode(id, ans, startedAt, updatedAt, id));
    }

    function _answerOf(address agg) internal view returns (uint80 id, int256 ans, uint256 up) {
        (id, ans,, up,) = VAgg(agg).latestRoundData();
    }

    /// @dev A pro-rata deposit of `amount` poolAsset by `who` (allowlisted by the core owner first), taking the
    ///      proportional loan legs. Returns whether it landed.
    function _depositBig(address who, uint256 amount) internal returns (bool ok) {
        address al = abi.decode(_scall(POOL, abi.encodeWithSignature("depositAllowlist()")), (address));
        address owner = _curator();
        if (al != address(0)) {
            vm.prank(owner);
            (bool a,) = al.call(abi.encodeWithSignature("setDepositorAllowed(address,bool)", who, true));
            a;
        }
        uint256 cap = VPool(POOL).limits().depositCapPoolAsset;
        (, uint256 gross) = VPool(POOL).poolAssetPosition();
        if (gross + amount > cap) amount = cap - gross - 1000;
        deal(CBBTC, who, VERC20(CBBTC).balanceOf(who) + amount);
        deal(USDC, who, VERC20(USDC).balanceOf(who) + 10_000_000e6);
        vm.startPrank(who);
        VERC20(CBBTC).approve(POOL, type(uint256).max);
        VERC20(USDC).approve(POOL, type(uint256).max);
        uint256[] memory mx = new uint256[](1);
        mx[0] = type(uint256).max;
        uint256[] memory mn = new uint256[](1);
        bytes memory ret;
        (ok, ret) = POOL.call(
            abi.encodeWithSignature(
                "depositProRata(uint256,address,uint256[],uint256[],uint256,uint256)", amount, who, mx, mn, 0, block.timestamp
            )
        );
        vm.stopPrank();
        if (!ok) {
            vm.writeLine(out, string.concat('{"k":"note","tag":"deposit","msg":"', vm.toString(ret), '"}'));
        }
    }

    function _scall(address to, bytes memory data) internal view returns (bytes memory ret) {
        bool ok;
        (ok, ret) = to.staticcall(data);
        require(ok, "scall");
    }

    function _venue0() internal view returns (VRouter.VenueView memory) {
        return VRouter(ROUTER).venue(POOL, 0);
    }
}
