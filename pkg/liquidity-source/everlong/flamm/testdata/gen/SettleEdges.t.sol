// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import "./FinancingEdgesBase.sol";
import {IMMRouter} from "src/interfaces/core/mm/IMMRouter.sol";
import {IFLAMM} from "src/interfaces/core/flamm/IFLAMM.sol";

interface VSAcct {
    function oraclePrice(bytes32) external view returns (bool, uint256);
    function lltv(bytes32) external view returns (uint64);
}

interface VSFeed {
    function peekCross(address, address) external view returns (bool, uint256, uint48);
}

interface VSErc20 {
    function approve(address, uint256) external returns (bool);
    function balanceOf(address) external view returns (uint256);
}

/// @dev Edge generator for the FLAMMSwapLib settlement legs in router.go (mmSettleSell /
///      mmSettleBuy: payLoan, takeLoan, strict reclaim, releaseExcess and the gates) through REAL swaps on the
///      deployed c104 pool at block 51317000, from pool / Router / Blue states written into storage: tracked liquid,
///      reserve target, the lending feature bit, physical below the payout, venue supply to withdraw first, excess
///      posted collateral swept across the release hysteresis, an IRM outage and a warp. previewSwap gives the
///      (used, net) the settlement runs on; a swap that previews but reverts is a settlement revert.
contract SettleEdges is FinancingEdgesBase {
    bytes32 internal constant L = 0x5b7e76949cacd5346234367c3806fe494a22f183af782d834d5fc4ee5b0f4500;
    address internal constant FEED = 0xbED275459578C87a63F2f50A0b077C720e838816;
    address internal constant USER = address(0xBEEF5);

    struct SC {
        string tag;
        bool buy;
        uint256 amountIn;
        bool setPhysical;
        uint256 physical;
        bool setLiquid;
        uint256 liquid;
        bool setReserve;
        uint256 reserve;
        bool setFeatures;
        uint256 features;
        bool setSupply;
        uint256 supplyShares;
        bool setColl;
        uint256 collateral;
        uint64 warp;
        bool irmDead;
    }

    uint256 internal venuesBase;
    VMParams internal liveParams;

    function _setup() internal {
        vm.createSelectFork(RPC, BLOCK);
        venuesBase = uint256(keccak256(abi.encode(uint256(keccak256(abi.encode(POOL, uint256(1)))) + 3)));
        liveParams = VMParams(USDC, CBBTC, LIVE_ORACLE, IRM, 0.86e18);
        deal(CBBTC, USER, 1e12);
        deal(USDC, USER, 1e15);
        vm.startPrank(USER);
        VSErc20(CBBTC).approve(POOL, type(uint256).max);
        VSErc20(USDC).approve(POOL, type(uint256).max);
        vm.stopPrank();
    }

    function _loanSlot() internal pure returns (uint256) {
        return uint256(keccak256(abi.encode(uint256(L) + 11)));
    }

    function _apply(SC memory c) internal {
        if (c.warp != 0) vm.warp(block.timestamp + c.warp);
        if (c.setPhysical) vm.store(POOL, bytes32(uint256(L) + 12), bytes32(c.physical));
        if (c.setLiquid) {
            vm.store(POOL, bytes32(_loanSlot() + 5), bytes32(c.liquid));
            deal(USDC, POOL, VSErc20(USDC).balanceOf(POOL) + c.liquid);
        }
        if (c.setReserve) vm.store(POOL, bytes32(_loanSlot() + 4), bytes32(c.reserve));
        if (c.setFeatures) vm.store(POOL, bytes32(uint256(L) + 23), bytes32(c.features));
        if (c.setSupply) {
            (uint256 s0, uint128 b0, uint128 k0) = VMorpho(MORPHO).position(LIVE_ID, ACCOUNT);
            s0;
            writePosition(LIVE_ID, ACCOUNT, c.supplyShares, b0, k0);
            vm.store(ROUTER, bytes32(venuesBase + 5), bytes32(c.supplyShares));
        }
        if (c.setColl) {
            (uint256 s1, uint128 b1,) = VMorpho(MORPHO).position(LIVE_ID, ACCOUNT);
            writePosition(LIVE_ID, ACCOUNT, s1, b1, uint128(c.collateral));
            vm.store(ROUTER, bytes32(venuesBase + 4), bytes32(c.collateral));
        }
        if (c.irmDead) {
            vm.mockCallRevert(IRM, abi.encodeWithSelector(VIrmFull.borrowRateView.selector, liveParams), abi.encodeWithSignature("Error(string)", "irm dead"));
            vm.mockCallRevert(IRM, abi.encodeWithSelector(VIrmFull.borrowRate.selector, liveParams), abi.encodeWithSignature("Error(string)", "irm dead"));
        }
        (,,,,, uint256 rt, uint256 lq) = IFLAMM(POOL).loanConfig(0);
        (uint256 ph,) = IFLAMM(POOL).poolAssetPosition();
        (uint256 fb,,) = IFLAMM(POOL).switches();
        if (c.setLiquid) require(lq == c.liquid, "liquid slot");
        if (c.setReserve) require(rt == c.reserve, "reserve slot");
        if (c.setPhysical) require(ph == c.physical, "physical slot");
        if (c.setFeatures) require(fb == c.features, "features slot");
    }

    function _ords(uint16[] memory o) internal pure returns (string memory s) {
        s = "[";
        for (uint256 i; i < o.length; ++i) s = string.concat(s, i == 0 ? "" : ",", vm.toString(o[i]));
        s = string.concat(s, "]");
    }

    function _routerJson(bool irmDead) internal view returns (string memory o) {
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
        IMMRouter.LoanView memory l = IMMRouter(ROUTER).loan(POOL, 0);
        string memory x = "{";
        x = add(x, "decimals", q(l.decimals));
        x = add(x, "scale", q(l.loanScale));
        x = add(x, "debtCap", q(l.debtCap));
        x = add(x, "supplyCap", q(l.supplyCap));
        x = add(x, "borrowEnabled", qb(l.borrowEnabled));
        x = end(add(x, "retired", qb(l.retired)));
        o = add(o, "loans", string.concat("[", x, "]"));
        IMMRouter.VenueView memory vv = IMMRouter(ROUTER).venue(POOL, 0);
        (bool ook, uint256 op) = VSAcct(ACCOUNT).oraclePrice(LIVE_ID);
        string memory v = "{";
        v = add(v, "loanIndex", q(vv.loanIndex));
        v = add(v, "lltv", q(vv.lltvWad));
        v = add(v, "borrowEnabled", qb(vv.borrowEnabled));
        v = add(v, "supplyEnabled", qb(vv.supplyEnabled));
        v = add(v, "retired", qb(vv.retired));
        v = add(v, "debtCap", q(vv.debtCap));
        v = add(v, "supplyCap", q(vv.supplyCap));
        v = add(v, "maxRate", q(vv.maxBorrowRateWad));
        v = add(v, "managedColl", q(uint256(vm.load(ROUTER, bytes32(venuesBase + 4)))));
        v = add(v, "managedShares", q(uint256(vm.load(ROUTER, bytes32(venuesBase + 5)))));
        v = add(v, "hasIrm", "true");
        v = add(v, "irmDead", qb(irmDead));
        v = add(v, "rat", qi(VIrm(IRM).rateAtTarget(LIVE_ID)));
        v = add(v, "oracleOk", qb(ook));
        v = add(v, "oraclePrice", q(op));
        v = add(v, "acctLltv", q(VSAcct(ACCOUNT).lltv(LIVE_ID)));
        v = add(v, "market", marketJson(LIVE_ID));
        v = end(add(v, "position", positionJson(LIVE_ID, ACCOUNT)));
        o = end(add(o, "venues", string.concat("[", v, "]")));
    }

    function _poolJson() internal view returns (string memory o) {
        (,,,,, uint256 rt, uint256 lq) = IFLAMM(POOL).loanConfig(0);
        (uint256 ph,) = IFLAMM(POOL).poolAssetPosition();
        (uint256 fb,,) = IFLAMM(POOL).switches();
        IFLAMM.Dials memory d = IFLAMM(POOL).dials();
        IFLAMM.Limits memory lim = IFLAMM(POOL).limits();
        (bool ok, uint256 p,) = VSFeed(FEED).peekCross(CBBTC, USDC);
        o = "{";
        o = add(o, "physical", q(ph));
        o = add(o, "liquid", q(lq));
        o = add(o, "reserveTarget", q(rt));
        o = add(o, "features", q(fb));
        o = add(o, "ltv", q(d.ltvWad));
        o = add(o, "phi", q(d.phiWad));
        o = add(o, "eps", q(lim.roomEpsilonWad));
        o = end(add(o, "priceWad", q(ok ? p : 0)));
    }

    function runCase(SC memory c) external {
        uint256 snap = vm.snapshotState();
        vm.clearMockedCalls(); // mocks are cheatcode state and survive revertToState
        _apply(c);
        string memory o = "{";
        o = add(o, "tag", string.concat('"', c.tag, '"'));
        o = add(o, "buy", qb(c.buy));
        o = add(o, "amountIn", q(c.amountIn));
        o = add(o, "now", q(block.timestamp));
        o = add(o, "prePool", _poolJson());
        o = add(o, "preRouter", _routerJson(c.irmDead));
        (bool pok, bytes memory pret) = POOL.staticcall(abi.encodeCall(IFLAMM.previewSwap, (!c.buy, c.amountIn)));
        uint256 pu;
        uint256 pn;
        if (pok) (pu, pn,) = abi.decode(pret, (uint256, uint256, uint256));
        o = add(o, "previewOk", qb(pok));
        o = add(o, "previewRet", qh(pok ? bytes("") : pret));
        o = add(o, "used", q(pu));
        o = add(o, "net", q(pn));
        vm.prank(USER);
        (bool ok, bytes memory ret) = POOL.call(
            abi.encodeCall(IFLAMM.swap, (c.buy ? USDC : CBBTC, c.buy ? CBBTC : USDC, c.amountIn, 0, USER, block.timestamp))
        );
        uint256 su;
        uint256 sn;
        if (ok) (su, sn) = abi.decode(ret, (uint256, uint256));
        o = add(o, "ok", qb(ok));
        o = add(o, "ret", qh(ok ? bytes("") : ret));
        o = add(o, "swapUsed", q(su));
        o = add(o, "swapNet", q(sn));
        o = add(o, "postPool", _poolJson());
        row(end(add(o, "postRouter", _routerJson(c.irmDead))));
        vm.revertToState(snap);
    }

    function _c(string memory tag, bool buy, uint256 amt) internal pure returns (SC memory c) {
        c.tag = tag;
        c.buy = buy;
        c.amountIn = amt;
    }

    function test_settle() public {
        _setup();
        open("mm_settle_edges.json");
        uint256[7] memory sells = [uint256(1), 1000, 5000, 15000, 20000, 60000, 139000];
        uint256[8] memory buys = [uint256(1), 1e6, 5e6, 11_301_760, 11_400_000, 30e6, 150e6, 189e6];
        SC memory c;
        for (uint256 i; i < sells.length; ++i) this.runCase(_c("live-sell", false, sells[i]));
        for (uint256 i; i < buys.length; ++i) this.runCase(_c("live-buy", true, buys[i]));
        // tracked liquid covering part / all of the payout
        uint256[3] memory liq = [uint256(1), 3_000_000, 50e6];
        for (uint256 k; k < liq.length; ++k) {
            for (uint256 i = 1; i < 6; ++i) {
                c = _c("liquid-sell", false, sells[i]);
                c.setLiquid = true;
                c.liquid = liq[k];
                this.runCase(c);
            }
            c = _c("liquid-buy", true, 30e6);
            c.setLiquid = true;
            c.liquid = liq[k];
            this.runCase(c);
        }
        // venue supply the funding plan withdraws before it borrows
        uint256[4] memory sup = [uint256(1e9), 2e12, 4.5e12, 1e14];
        for (uint256 k; k < sup.length; ++k) {
            for (uint256 i = 2; i < 6; ++i) {
                c = _c("supply-sell", false, sells[i]);
                c.setSupply = true;
                c.supplyShares = sup[k];
                this.runCase(c);
            }
            c = _c("supply-buy", true, 5e6);
            c.setSupply = true;
            c.supplyShares = sup[k];
            this.runCase(c);
        }
        // reserve target and the lending bit on a buy whose input beats the debt
        uint256[4] memory rts = [uint256(0), 1, 10e6, 1e12];
        for (uint256 k; k < rts.length; ++k) {
            for (uint256 f; f < 2; ++f) {
                c = _c("reserve-buy", true, k % 2 == 0 ? 30e6 : 150e6);
                c.setReserve = true;
                c.reserve = rts[k];
                c.setFeatures = true;
                c.features = f == 0 ? 63 : 63 & ~uint256(1 << 3);
                this.runCase(c);
            }
        }
        // physical below the payout: strict reclaim
        uint256[5] memory phys = [uint256(0), 1, 1000, 5000, 200000];
        for (uint256 k; k < phys.length; ++k) {
            for (uint256 i = 1; i < 8; ++i) {
                c = _c("reclaim-buy", true, buys[i]);
                c.setPhysical = true;
                c.physical = phys[k];
                this.runCase(c);
            }
            for (uint256 i = 1; i < sells.length; ++i) {
                c = _c("reclaim-sell", false, sells[i]);
                c.setPhysical = true;
                c.physical = phys[k];
                this.runCase(c);
            }
        }
        // excess posted collateral: sweep across the 10% release hysteresis after a 1 USDC buy
        for (uint256 p = 26_380; p < 26_420; ++p) {
            c = _c("release-buy", true, 1e6);
            c.setColl = true;
            c.collateral = p;
            this.runCase(c);
        }
        uint256[6] memory colls = [uint256(15000), 26221, 40000, 100000, 1e6, 1e8];
        for (uint256 k; k < colls.length; ++k) {
            c = _c("release-buy-wide", true, 5e6);
            c.setColl = true;
            c.collateral = colls[k];
            this.runCase(c);
            c = _c("release-sell-wide", false, 20000);
            c.setColl = true;
            c.collateral = colls[k];
            this.runCase(c);
        }
        // IRM outage (inside the grace) and a warp
        c = _c("irm-dead-buy", true, 5e6);
        c.irmDead = true;
        this.runCase(c);
        c = _c("irm-dead-sell", false, 5000);
        c.irmDead = true;
        this.runCase(c);
        uint64[3] memory warps = [uint64(1), 120, 1800];
        for (uint256 k; k < warps.length; ++k) {
            c = _c("warp-sell", false, 5000);
            c.warp = warps[k];
            this.runCase(c);
            c = _c("warp-buy", true, 11_301_760);
            c.warp = warps[k];
            this.runCase(c);
        }
        close();
    }
}
