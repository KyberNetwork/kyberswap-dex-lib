// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import {MMFixtureBase, KRouter, KMorpho, KIrm, KAccount, KVenueView, KLoanView, MParams, MMarket} from "test/kyber/MMFixtureBase.sol";

struct EHookSet {
    address invariantHook;
    address feeHook;
    address recenterHook;
    address controllerHook;
    address leverageHook;
    address spreadHook;
    address loanSwapHook;
}

struct EDials {
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

struct ELimits {
    uint256 depositCapPoolAsset;
    uint64 roomEpsilonWad;
    uint32 maxPriceAgeSec;
    uint64 feeFloorWad;
    uint64 feeCapWad;
    uint16 performanceFeeBp;
    uint96 peakSharePriceWad;
    uint16 maxPerformanceFeeBp;
}

struct EFeeParams {
    uint64 midFeeWad;
    uint64 outFeeWad;
    uint64 gammaWad;
    uint64 sigmaRefWad;
    uint64 volBetaWad;
    uint64 volMinWad;
    uint64 volMaxWad;
    uint64 dirSkewWad;
}

struct ETuning {
    EFeeParams fee;
    uint64 invSkewKappaWad;
    uint64 invSkewBandWad;
    uint64 emaHalfLife;
    uint64 rvHalfLife;
    uint128 stepDivisorWad;
    uint64 inertiaWad;
    uint64 inertiaMaxWad;
}

struct EBounds {
    uint64 minFeeWad;
    uint64 maxFeeWad;
    uint64 minEmaHalfLife;
    uint64 maxEmaHalfLife;
    uint64 maxRvHalfLife;
    uint64 maxStepWad;
    uint128 minStepDivisorWad;
}

struct EParams {
    uint128 aWad;
    uint128 spanUpWad;
    uint128 spanDnWad;
    uint256 anchorPriceWad;
    uint8 loanDecimals;
    ETuning tuning;
    EBounds bounds;
}

struct ESupport {
    uint256 aWad;
    uint256 xLo;
    uint256 xHi;
    uint256 yHi;
}

struct EFeedToken {
    address aggregator;
    uint32 heartbeat;
    uint64 scale;
    uint64 unit;
    uint64 pegBandWad;
}

interface EPool {
    function asset() external view returns (address);
    function router() external view returns (address);
    function priceFeed() external view returns (address);
    function core() external view returns (address);
    function loanCount() external view returns (uint256);
    function loanConfig(uint8) external view returns (address, uint8, uint64, uint64, uint256, uint256, uint256);
    function paused() external view returns (bool);
    function switches() external view returns (uint256, uint32, bool);
    function hooks() external view returns (EHookSet memory);
    function poolAssetPosition() external view returns (uint256, uint256);
    function loanPosition() external view returns (uint256, uint256, uint256);
    function dials() external view returns (EDials memory);
    function limits() external view returns (ELimits memory);
    function totalSupply() external view returns (uint256);
    function totalAssets() external view returns (uint256);
    function previewSwap(bool, uint256) external view returns (uint256, uint256, uint256);
    function previewLever(bool, uint256) external view returns (uint256, uint256, uint256, uint256);
    function swap(address, address, uint256, uint256, address, uint256) external returns (uint256, uint256);
    function leverUp(uint256, uint256, address, uint256) external returns (uint256, uint256);
    function leverDown(uint256, uint256, address, uint256) external returns (uint256, uint256);
    function setLevPaused(bool) external;
    function setLoanConfig(uint8, uint64, uint64, uint256, uint256) external;
}

interface EPoolAdmin {
    function setPaused(bool) external;
    function setFeatures(uint256) external;
    function setFeeBounds(uint64, uint64) external;
}

interface EHook {
    function params() external view returns (EParams memory);
    function support() external view returns (ESupport memory);
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
    function POOL() external view returns (address);
    function genesisStrategyHash() external view returns (bytes32);
}

interface ELevHook {
    function HOOK() external view returns (address);
    function LOAN_SCALE() external view returns (uint256);
    function POOL() external view returns (address);
}

interface ESpreadHook {
    function spread() external view returns (uint24);
    function maxSpreadAge() external view returns (uint32);
    function lastSetTs() external view returns (uint48);
    function POOL() external view returns (address);
    function setSpread(uint24) external;
    function setMaxSpreadAge(uint32) external;
}

interface EFeed {
    function SEQUENCER_FEED() external view returns (address);
    function SEQUENCER_GRACE() external view returns (uint256);
    function config(address) external view returns (EFeedToken memory);
    function peekCross(address, address) external view returns (bool, uint256, uint48);
    function pegOk(address) external view returns (bool);
}

interface EAggregator {
    function latestRoundData() external view returns (uint80, int256, uint256, uint256, uint80);
}

interface EOracle {
    function price() external view returns (uint256);
}

interface ECore {
    function owner() external view returns (address);
    function keeper() external view returns (address);
}

/// @notice Shared by the core end-to-end generators: the COMPLETE state the Go port evaluates, read the way the
///         tracker reads it -- view getters only, plus three raw storage words that no view exposes:
///           - FLAMMStore.lastLeverSpreadPpm: ERC-7201 base + 22, bits 160..191 (checked against executed levers);
///           - MMRouter venue managedCollateral / managedSupplyShares: keccak(keccak(pool . 1) + 3) + 6i + 4 / + 5.
///         Each dump also carries a few deployed views the port must reproduce from that state (attestation).
abstract contract CoreE2EBase is MMFixtureBase {
    address constant HOOK = 0x65CBD227cBC61248ae77a5fC813A29C54C092134;
    address constant LEV_HOOK = 0xE0A98d8e60035832B8BaD7f7af7B9B0b3A7308F3;
    address constant SPREAD_HOOK = 0x04988aF54ec88D2de77b191025EAef2fe488f93b;
    uint256 constant FLAMM_STORE = 0x5b7e76949cacd5346234367c3806fe494a22f183af782d834d5fc4ee5b0f4500;

    bytes4 constant SEL_ROUND = EAggregator.latestRoundData.selector;

    string internal _out;

    function _line(string memory s) internal {
        vm.writeLine(_out, s);
    }

    function _lastLeverSpreadPpm() internal view returns (uint256) {
        return (uint256(vm.load(POOL, bytes32(FLAMM_STORE + 22))) >> 160) & type(uint32).max;
    }

    // ------------------------------------------------------------------ pool
    function _poolLoans() internal view returns (string memory s) {
        EPool p = EPool(POOL);
        uint256 n = p.loanCount();
        s = "[";
        for (uint256 i; i < n; ++i) {
            (address token, uint8 dec, uint64 band, uint64 floorWad, uint256 maxN, uint256 reserve, uint256 liquid) =
                p.loanConfig(uint8(i));
            s = string.concat(
                s, i == 0 ? "" : ",", '{"token":"', vm.toString(token), '","decimals":', vm.toString(uint256(dec)),
                ',"swapPriceBandWad":', _u(band), ',"feeFloorWad":', _u(floorWad), ',"maxSwapNotional":', _u(maxN)
            );
            s = string.concat(s, ',"reserveTarget":', _u(reserve), ',"liquid":', _u(liquid), "}");
        }
        s = string.concat(s, "]");
    }

    function _hooksJson(EHookSet memory h) internal pure returns (string memory) {
        return string.concat(
            '["', vm.toString(h.invariantHook), '","', vm.toString(h.feeHook), '","', vm.toString(h.recenterHook), '","',
            vm.toString(h.controllerHook), '","', vm.toString(h.leverageHook), '","', vm.toString(h.spreadHook), '","',
            string.concat(vm.toString(h.loanSwapHook), '"]')
        );
    }

    function _poolDump() internal view returns (string memory s) {
        EPool p = EPool(POOL);
        (uint256 physical, uint256 gross) = p.poolAssetPosition();
        (uint256 features,, bool levPaused) = p.switches();
        EDials memory d = p.dials();
        ELimits memory lim = p.limits();
        s = string.concat(
            '{"asset":"', vm.toString(p.asset()), '","router":"', vm.toString(p.router()), '","priceFeed":"',
            vm.toString(p.priceFeed()), '","paused":', _b(p.paused()), ',"features":', _u(features), ',"levPaused":',
            _b(levPaused)
        );
        s = string.concat(
            s, ',"hooks":', _hooksJson(p.hooks()), ',"physical":', _u(physical), ',"gross":', _u(gross), ',"totalSupply":',
            _u(p.totalSupply()), ',"phiWad":', _u(d.phiWad), ',"ltvWad":', _u(d.ltvWad)
        );
        s = string.concat(
            s, ',"roomEpsilonWad":', _u(lim.roomEpsilonWad), ',"feeFloorWad":', _u(lim.feeFloorWad), ',"feeCapWad":',
            _u(lim.feeCapWad), ',"loans":', _poolLoans(), ',"lastLeverSpreadPpm":', _u(_lastLeverSpreadPpm()), "}"
        );
    }

    // ------------------------------------------------------------------ swap hook, leverage hook, spread hook
    function _feeRow(EFeeParams memory f) internal pure returns (string memory) {
        return string.concat(
            "[", _u(f.midFeeWad), ",", _u(f.outFeeWad), ",", _u(f.gammaWad), ",", _u(f.sigmaRefWad), ",",
            string.concat(_u(f.volBetaWad), ",", _u(f.volMinWad), ",", _u(f.volMaxWad), ",", _u(f.dirSkewWad), "]")
        );
    }

    function _hookDump() internal view returns (string memory s) {
        EHook h = EHook(HOOK);
        EParams memory p = h.params();
        ESupport memory sup = h.support();
        s = string.concat(
            '{"aWad":', _u(p.aWad), ',"fee":', _feeRow(p.tuning.fee), ',"invSkewKappaWad":', _u(p.tuning.invSkewKappaWad),
            ',"invSkewBandWad":', _u(p.tuning.invSkewBandWad), ',"support":[', _u(sup.aWad), ",", _u(sup.xLo), ","
        );
        s = string.concat(
            s, _u(sup.xHi), ",", _u(sup.yHi), '],"anchorSqrtX96":', _u(h.anchorSqrtX96()), ',"reservationPriceWad":',
            _u(h.reservationPriceWad()), ',"kappa":', _u(h.kappa()), ',"xWad":', _u(h.xWad())
        );
        s = string.concat(
            s, ',"reserveStable":', _u(h.reserveStable()), ',"idleStable":', _u(h.idleStable()), ',"reserveVolatile":',
            _u(h.reserveVolatile()), ',"idleVolatile":', _u(h.idleVolatile()), ',"rvWad":', _u(h.rvWad())
        );
        s = string.concat(
            s, ',"loanScale":', _u(h.LOAN_SCALE()), ',"pool":"', vm.toString(h.POOL()), '","genesisStrategyHash":"',
            vm.toString(h.genesisStrategyHash()), '"}'
        );
    }

    function _levDump() internal view returns (string memory) {
        return string.concat(
            '{"hook":"', vm.toString(ELevHook(LEV_HOOK).HOOK()), '","loanScale":', _u(ELevHook(LEV_HOOK).LOAN_SCALE()),
            ',"pool":"', vm.toString(ELevHook(LEV_HOOK).POOL()), '"}'
        );
    }

    function _spreadDump() internal view returns (string memory) {
        ESpreadHook h = ESpreadHook(SPREAD_HOOK);
        return string.concat(
            '{"spread":', _u(h.spread()), ',"maxSpreadAge":', _u(h.maxSpreadAge()), ',"lastSetTs":', _u(h.lastSetTs()),
            ',"pool":"', vm.toString(h.POOL()), '"}'
        );
    }

    // ------------------------------------------------------------------ price feed
    function _round(address agg) internal view returns (string memory) {
        if (agg == address(0)) return '{"ok":false}';
        try EAggregator(agg).latestRoundData() returns (uint80 id, int256 answer, uint256 startedAt, uint256 updatedAt, uint80) {
            return string.concat(
                '{"ok":true,"roundId":', _u(id), ',"answer":', _i(answer), ',"startedAt":', _u(startedAt), ',"updatedAt":',
                _u(updatedAt), "}"
            );
        } catch {
            return '{"ok":false}';
        }
    }

    function _feedToken(address token) internal view returns (string memory) {
        try EFeed(FEED).config(token) returns (EFeedToken memory c) {
            return string.concat(
                '{"token":"', vm.toString(token), '","known":true,"aggregator":"', vm.toString(c.aggregator), '","heartbeat":',
                _u(c.heartbeat), ',"scale":', _u(c.scale), ',"unit":', _u(c.unit), ',"pegBandWad":', _u(c.pegBandWad),
                string.concat(',"round":', _round(c.aggregator), "}")
            );
        } catch {
            return string.concat('{"token":"', vm.toString(token), '","known":false}');
        }
    }

    function _feedDump() internal view returns (string memory s) {
        EFeed f = EFeed(FEED);
        address seq = f.SEQUENCER_FEED();
        s = string.concat(
            '{"sequencerFeed":"', vm.toString(seq), '","sequencerGrace":', _u(f.SEQUENCER_GRACE()), ',"sequencer":',
            _round(seq), ',"tokens":[', _feedToken(EPool(POOL).asset())
        );
        uint256 n = EPool(POOL).loanCount();
        for (uint256 i; i < n; ++i) {
            (address token,,,,,,) = EPool(POOL).loanConfig(uint8(i));
            s = string.concat(s, ",", _feedToken(token));
        }
        s = string.concat(s, "]}");
    }

    // ------------------------------------------------------------------ router, venues, Morpho, IRM, oracle
    function _oracleZero(address oracle) internal view returns (bool) {
        try EOracle(oracle).price() returns (uint256 v) {
            return v == 0;
        } catch {
            return false;
        }
    }

    function _venueMorpho(KVenueView memory v) internal view returns (string memory) {
        MParams memory p = KMorpho(MORPHO).idToMarketParams(v.id);
        MMarket memory m = _market(v.id);
        (uint256 ss, uint128 bs, uint128 col) = KMorpho(MORPHO).position(v.id, v.account);
        int256 rat = p.irm == address(0) ? int256(0) : KIrm(p.irm).rateAtTarget(v.id);
        bool irmOk = true;
        if (p.irm != address(0)) {
            try KIrm(p.irm).borrowRateView(p, m) returns (uint256) {} catch { irmOk = false; }
        }
        (bool ook, uint256 op) = KAccount(v.account).oraclePrice(v.id);
        return string.concat(
            ',"market":', _marketJson(m), ',"position":{"supplyShares":', _u(ss), ',"borrowShares":', _u(bs),
            ',"collateral":', _u(col), '},"irm":"', vm.toString(p.irm), '","marketLltv":', _u(p.lltv),
            string.concat(
                ',"rateAtTarget":', _i(rat), ',"hasIrm":', _b(p.irm != address(0)), ',"irmReadable":', _b(irmOk),
                ',"oracleOk":', _b(ook), ',"oraclePrice":', _u(op), ',"oracleZero":', _b(!ook && _oracleZero(p.oracle)), "}"
            )
        );
    }

    function _venueDump(uint256 i) internal view returns (string memory) {
        KVenueView memory v = KRouter(ROUTER).venue(POOL, uint16(i));
        (uint256 mc, uint256 ms) = _managed(ROUTER, POOL, i, v);
        string memory cfg = string.concat(
            '{"account":"', vm.toString(v.account), '","id":"', vm.toString(v.id), '","kind":', vm.toString(uint256(v.kind)),
            ',"loanIndex":', vm.toString(uint256(v.loanIndex)), ',"lltvWad":', _u(v.lltvWad), ',"borrowEnabled":',
            _b(v.borrowEnabled), ',"supplyEnabled":', _b(v.supplyEnabled), ',"retired":', _b(v.retired)
        );
        cfg = string.concat(
            cfg, ',"debtCap":', _u(v.debtCap), ',"supplyCap":', _u(v.supplyCap), ',"maxBorrowRateWad":',
            _u(v.maxBorrowRateWad), ',"managedCollateral":', _u(mc), ',"managedSupplyShares":', _u(ms)
        );
        return string.concat(cfg, _venueMorpho(v));
    }

    function _routerDump() internal view returns (string memory s) {
        (uint64 pinLtv, uint64 gap, uint64 band) = KRouter(ROUTER).pin(POOL);
        s = string.concat(
            '{"globalPaused":', _b(KRouter(ROUTER).globalPaused()), ',"pinLtvWad":', _u(pinLtv), ',"safetyGapWad":',
            _u(gap), ',"oracleBandWad":', _u(band), ',"maxDrawnAssets":',
            vm.toString(uint256(KRouter(ROUTER).maxDrawnAssets(POOL))), ',"loans":['
        );
        uint256 nl = KRouter(ROUTER).loanCount(POOL);
        for (uint256 i; i < nl; ++i) {
            KLoanView memory l = KRouter(ROUTER).loan(POOL, uint8(i));
            s = string.concat(
                s, i == 0 ? "" : ",", '{"token":"', vm.toString(l.token), '","decimals":', vm.toString(uint256(l.decimals)),
                ',"loanScale":', _u(l.loanScale), ',"debtCap":', _u(l.debtCap), ',"supplyCap":', _u(l.supplyCap),
                string.concat(',"borrowEnabled":', _b(l.borrowEnabled), ',"retired":', _b(l.retired), "}")
            );
        }
        (uint16[] memory bo, uint16[] memory so, uint16[] memory wo, uint16[] memory ro) = KRouter(ROUTER).priorities(POOL);
        s = string.concat(
            s, '],"borrowOrder":', _u16a(bo), ',"supplyOrder":', _u16a(so), ',"withdrawOrder":', _u16a(wo),
            ',"repayOrder":', _u16a(ro), ',"venues":['
        );
        uint256 nv = KRouter(ROUTER).venueCount(POOL);
        for (uint256 i; i < nv; ++i) {
            s = string.concat(s, i == 0 ? "" : ",", _venueDump(i));
        }
        s = string.concat(s, "]}");
    }

    // ------------------------------------------------------------------ attestation views
    function _viewsDump() internal view returns (string memory s) {
        (address loan0,,,,,,) = EPool(POOL).loanConfig(0);
        (bool ok, uint256 px, uint48 ts) = EFeed(FEED).peekCross(EPool(POOL).asset(), loan0);
        (uint256 liq, uint256 sup, uint256 debt) = EPool(POOL).loanPosition();
        (uint256[] memory c, uint256[] memory sp, uint256[] memory d, uint256 tc) = KRouter(ROUTER).positions(POOL);
        s = string.concat(
            '{"peekCross":{"ok":', _b(ok), ',"priceWad":', _u(px), ',"ts":', _u(ts), '},"pegOk":', _b(EFeed(FEED).pegOk(loan0)),
            ',"loanPosition":[', _u(liq), ",", _u(sup), ",", _u(debt), ']'
        );
        s = string.concat(
            s, ',"positions":{"coll":', _ua(c), ',"sup":', _ua(sp), ',"debt":', _ua(d), ',"totalColl":', _u(tc), "}"
        );
        try EPool(POOL).totalAssets() returns (uint256 ta) {
            s = string.concat(s, ',"totalAssets":{"v":', _u(ta), "}}");
        } catch (bytes memory e) {
            s = string.concat(s, ',"totalAssets":{"err":', _h(e), "}}");
        }
    }

    /// @dev One complete state at the current block and timestamp.
    function _dump() internal view returns (string memory) {
        return string.concat(
            '{"block":', vm.toString(block.number), ',"timestamp":', vm.toString(block.timestamp), ',"pool":', _poolDump(),
            ',"hook":', _hookDump(), ',"lev":', _levDump(), ',"spread":', _spreadDump(),
            string.concat(',"feed":', _feedDump(), ',"router":', _routerDump(), ',"views":', _viewsDump(), "}")
        );
    }

    // ------------------------------------------------------------------ preview rows
    function _words(bytes memory ret, uint256 n) internal pure returns (string memory s) {
        s = "[";
        for (uint256 i; i < n; ++i) {
            uint256 w;
            assembly ("memory-safe") {
                w := mload(add(ret, add(32, mul(32, i))))
            }
            s = string.concat(s, i == 0 ? "" : ",", _u(w));
        }
        s = string.concat(s, "]");
    }

    /// @dev previewSwap: returns whether it answered.
    function _pSwap(string memory tag, bool sell, uint256 amount) internal returns (bool ok) {
        bytes memory ret;
        (ok, ret) = POOL.staticcall(abi.encodeCall(EPool.previewSwap, (sell, amount)));
        _line(
            string.concat(
                '{"k":"sw","tag":"', tag, '","d":', sell ? "1" : "0", ',"a":', _u(amount),
                ok ? string.concat(',"r":', _words(ret, 3), "}") : string.concat(',"e":', _h(ret), "}")
            )
        );
    }

    function _pLever(string memory tag, bool up, uint256 amount) internal returns (bool ok) {
        bytes memory ret;
        (ok, ret) = POOL.staticcall(abi.encodeCall(EPool.previewLever, (up, amount)));
        _line(
            string.concat(
                '{"k":"lv","tag":"', tag, '","d":', up ? "1" : "0", ',"a":', _u(amount),
                ok ? string.concat(',"r":', _words(ret, 4), "}") : string.concat(',"e":', _h(ret), "}")
            )
        );
    }

    function _okSwap(bool sell, uint256 amount) internal view returns (bool ok) {
        (ok,) = POOL.staticcall(abi.encodeCall(EPool.previewSwap, (sell, amount)));
    }

    function _okLever(bool up, uint256 amount) internal view returns (bool ok) {
        (ok,) = POOL.staticcall(abi.encodeCall(EPool.previewLever, (up, amount)));
    }

    function _state(string memory tag) internal {
        _line(string.concat('{"k":"state","tag":"', tag, '","s":', _dump(), "}"));
    }

    // ------------------------------------------------------------------ roles
    function _curator() internal view returns (address) {
        return ECore(EPool(POOL).core()).owner();
    }

    function _keeper() internal view returns (address) {
        return ECore(EPool(POOL).core()).keeper();
    }

    // ------------------------------------------------------------------ scenario moves
    function _arm(uint24 spread) internal {
        vm.prank(_curator());
        EPool(POOL).setLevPaused(false);
        if (spread != 0) {
            vm.prank(_keeper());
            ESpreadHook(SPREAD_HOOK).setSpread(spread);
        }
    }

    function _loanCfg(uint64 band, uint64 floorWad, uint256 maxN, uint256 reserve) internal {
        vm.prank(_curator());
        EPool(POOL).setLoanConfig(0, band, floorWad, maxN, reserve);
    }

    function _liveLoanCfg() internal view returns (uint64 band, uint64 floorWad, uint256 maxN, uint256 reserve) {
        (, , band, floorWad, maxN, reserve,) = EPool(POOL).loanConfig(0);
    }

    function _agg(address token) internal view returns (address) {
        return EFeed(FEED).config(token).aggregator;
    }

    /// @dev Re-answer an aggregator at `answer`, observed `age` seconds ago (a mocked latestRoundData).
    function _mockRound(address agg, int256 answer, uint256 age) internal {
        (uint80 id,,,,) = EAggregator(agg).latestRoundData();
        uint256 ts = block.timestamp - age;
        vm.mockCall(agg, abi.encodeWithSelector(SEL_ROUND), abi.encode(id, answer, ts, ts, id));
    }

    function _answer(address agg) internal view returns (int256 a) {
        (, a,,,) = EAggregator(agg).latestRoundData();
    }

    /// @dev Both USD aggregators re-answer their current answers as observed `age` seconds ago.
    function _freshFeeds(uint256 age) internal {
        address a = _agg(CBBTC);
        address b = _agg(USDC);
        int256 x = _answer(a);
        int256 y = _answer(b);
        _mockRound(a, x, age);
        _mockRound(b, y, age);
    }

    /// @dev The venue market's AdaptiveCurveIrm reverts both its view and its mutating rate.
    function _irmDown() internal {
        vm.mockCallRevert(IRM, abi.encodeWithSignature("borrowRateView((address,address,address,address,uint256),(uint128,uint128,uint128,uint128,uint128,uint128))"), "irm down");
        vm.mockCallRevert(IRM, abi.encodeWithSignature("borrowRate((address,address,address,address,uint256),(uint128,uint128,uint128,uint128,uint128,uint128))"), "irm down");
    }

    /// @dev MMRouterLib.PoolRecord slot 0 of this pool: poolAsset (bits 0..159), pinLtvWad (160..223), maxDrawnAssets.
    function _storePin(uint64 pinWad) internal {
        bytes32 slot = bytes32(_poolBase(POOL));
        uint256 w = uint256(vm.load(ROUTER, slot));
        (uint64 pinBefore,,) = KRouter(ROUTER).pin(POOL);
        require(uint64(w >> 160) == pinBefore, "slot:pin");
        w = (w & ~(uint256(type(uint64).max) << 160)) | (uint256(pinWad) << 160);
        vm.store(ROUTER, slot, bytes32(w));
        (uint64 pinAfter,,) = KRouter(ROUTER).pin(POOL);
        require(pinAfter == pinWad, "slot:pin write");
    }

    /// @dev Back to a snapshot: snapshots do not undo mocked calls, so those are cleared too.
    function _reset(uint256 snap) internal {
        vm.revertToState(snap);
        vm.clearMockedCalls();
    }
}
