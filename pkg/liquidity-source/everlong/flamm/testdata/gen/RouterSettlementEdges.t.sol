// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import "./RouterSettlementEdgesBase.sol";
import {Math} from "@openzeppelin/contracts/utils/math/Math.sol";
import {IERC20} from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import {IMMRouter} from "src/interfaces/core/mm/IMMRouter.sol";
import {IPriceFeed} from "src/interfaces/core/IPriceFeed.sol";
import {FLAMMStore} from "src/core/flamm/FLAMMStore.sol";

/// @dev Stands in for the pool's code at the pool address, keeping the pool's storage: every settlement leg and gate
///      composite is DELEGATECALLed straight into the libraries the deployed implementation links (FLAMMSwapLib
///      0x89aA…D405, FLAMMGateLib 0x50417…844e) with the ERC-7201 slot as the storage pointer, so arbitrary
///      (used, net) reach the shipped settlement code without the preview bounding them. The buy mirrors
///      FLAMMSwapLib.execute's branch: anchor, settleBuy, assertEntryGate.
contract R2PoolHarness {
    bytes32 internal constant LOC = 0x5b7e76949cacd5346234367c3806fe494a22f183af782d834d5fc4ee5b0f4500;
    address internal constant SWAPLIB = 0x89aA5f76765D16c460A5B4a7AC6a385e35d2D405;
    address internal constant GATELIB = 0x50417cB978f856b3885AbfA97308e0377FC2844e;
    address internal constant ROUTER_ = 0x19A9b39E6710AAD109C829294b0841F0851c6bB4;

    function _dc(address lib, bytes memory data) internal returns (bytes memory ret) {
        bool ok;
        (ok, ret) = lib.delegatecall(data);
        if (!ok) {
            assembly {
                revert(add(ret, 32), mload(ret))
            }
        }
    }

    function sell(uint8 idx, uint256 priceWad, uint256 used, uint256 net) external {
        _dc(SWAPLIB, abi.encodeWithSelector(bytes4(0x694544f2), uint256(LOC), idx, priceWad, used, net, msg.sender));
    }

    function buy(uint8 idx, uint256 used, uint256 net) external {
        bytes memory a = _dc(GATELIB, abi.encodeWithSelector(bytes4(0x4e8ca163), uint256(LOC)));
        (int256[] memory u0, uint256 g0, bool q0) = abi.decode(a, (int256[], uint256, bool));
        _dc(SWAPLIB, abi.encodeWithSelector(bytes4(0xf2c9fa79), uint256(LOC), idx, used, net, msg.sender));
        _dc(GATELIB, abi.encodeWithSelector(bytes4(0xd5e81999), uint256(LOC), u0, g0, q0));
    }

    function release() external {
        _dc(SWAPLIB, abi.encodeWithSelector(bytes4(0x4be4b53a), uint256(LOC)));
    }

    function take(uint8 idx, uint256 used) external {
        _dc(SWAPLIB, abi.encodeWithSelector(bytes4(0xa145f2b1), uint256(LOC), idx, used));
    }

    function pay(uint8 idx, uint256 priceWad, uint256 net) external {
        _dc(SWAPLIB, abi.encodeWithSelector(bytes4(0x31451a3a), uint256(LOC), idx, priceWad, net, msg.sender));
    }

    function gate(bytes4 sel) external returns (bytes memory) {
        return _dc(GATELIB, abi.encodeWithSelector(sel, uint256(LOC)));
    }

    function entryGate(int256[] calldata u0, uint256 g0, bool q0) external {
        _dc(GATELIB, abi.encodeWithSelector(bytes4(0xd5e81999), uint256(LOC), u0, g0, q0));
    }

    function exitGate(int256[] calldata u0, uint256 g0) external {
        _dc(GATELIB, abi.encodeWithSelector(bytes4(0xc25dc550), uint256(LOC), u0, g0));
    }

    function setBook(uint256 physical, uint256 features, uint64 ltv) external {
        FLAMMStore.S storage $ = FLAMMStore.s();
        $.physicalPoolAsset = physical;
        $.features = features;
        $.ltvWad = ltv;
    }

    function setLoan(uint256 i, uint256 liquid, uint256 reserve) external {
        FLAMMStore.LoanCfg storage c = FLAMMStore.s().loans[i];
        c.liquid = liquid;
        c.reserveTarget = reserve;
    }

    function pushLoan(address token, uint8 dec) external {
        FLAMMStore.LoanCfg storage c = FLAMMStore.s().loans.push();
        c.token = token;
        c.decimals = dec;
        c.scale = 10 ** (18 - dec);
    }

    function approveRouter(address token) external {
        IERC20(token).approve(ROUTER_, type(uint256).max);
    }

    function st()
        external
        view
        returns (uint256 physical, uint256 features, uint64 ltv, uint64 phi, uint64 eps, uint256[] memory liquid, uint256[] memory reserve, uint256[] memory scale)
    {
        FLAMMStore.S storage $ = FLAMMStore.s();
        physical = $.physicalPoolAsset;
        features = $.features;
        ltv = $.ltvWad;
        phi = $.phiWad;
        eps = $.roomEpsilonWad;
        uint256 n = $.loans.length;
        liquid = new uint256[](n);
        reserve = new uint256[](n);
        scale = new uint256[](n);
        for (uint256 i; i < n; ++i) {
            liquid[i] = $.loans[i].liquid;
            reserve[i] = $.loans[i].reserveTarget;
            scale[i] = $.loans[i].scale;
        }
    }
}

interface R2Harness {
    function sell(uint8, uint256, uint256, uint256) external;
    function buy(uint8, uint256, uint256) external;
    function release() external;
    function take(uint8, uint256) external;
    function pay(uint8, uint256, uint256) external;
    function gate(bytes4) external returns (bytes memory);
    function entryGate(int256[] calldata, uint256, bool) external;
    function exitGate(int256[] calldata, uint256) external;
    function setBook(uint256, uint256, uint64) external;
    function setLoan(uint256, uint256, uint256) external;
    function pushLoan(address, uint8) external;
    function approveRouter(address) external;
    function st() external view returns (uint256, uint256, uint64, uint64, uint64, uint256[] memory, uint256[] memory, uint256[] memory);
}

/// @dev Edge generator for the settlement legs and Router composites (router.go / gate.go
///      settlement half) against the DEPLOYED FLAMMSwapLib, FLAMMGateLib, MMRouter, MorphoBlueAccount, Morpho Blue and
///      AdaptiveCurveIrm on a Base fork. The live pool record is extended in place to three USDC venues (the live
///      86% market, a created 77% market with a fee, a created 91.5% market with no rate model) and, in the two-asset
///      run, a fourth venue on an 18-decimal loan asset. Scenario states are written into storage; each op runs on a
///      snapshot and records its result and full post-state.
contract RouterSettlementEdges is RouterSettlementEdgesBase {
    address internal constant USER = address(0xBEEF52);
    uint256 internal constant P0 = 788432322552395;
    uint256 internal constant P1 = 262810774184131;
    uint256 internal constant ORACLE0 = 788432322552395 * 1e24;
    bytes4 internal constant SEL_ANCHOR = 0x4e8ca163;
    bytes4 internal constant SEL_ASSERT = 0x09d152d0;
    bytes4 internal constant SEL_TOTAL = 0x788abe73;

    struct V {
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
        uint256 mColl;
        uint256 mShares;
        uint128 debtCap;
        uint128 supplyCap;
        uint64 maxRate;
        bool be;
        bool se;
    }

    struct Sc {
        string tag;
        V[4] v;
        uint64 pin;
        uint64 band;
        uint8 maxDrawn;
        bool paused;
        uint128[2] lDebtCap;
        uint128[2] lSupCap;
        bool[2] lBorrow;
        uint16[] bo;
        uint16[] so;
        uint16[] wo;
        uint16[] ro;
        uint256 physical;
        uint256[2] liquid;
        uint256[2] reserve;
        uint256 features;
        uint64 ltv;
        uint256[2] feedP;
        bool[2] feedOk;
        uint256[2] usd;
        bool[2] usdOk;
    }

    R2Token internal loan1;
    R2Oracle[4] internal oracles;
    R2Params[4] internal params;
    bytes32[4] internal ids;
    address[4] internal accts;
    uint256 internal nV;
    uint256 internal nL;
    uint256 internal T0;
    uint256 internal venuesBase;
    uint256 internal baseSnap;
    Sc internal cur;
    V internal live;
    uint256 internal livePhysical;
    uint256 internal liveLiquid;
    uint256 internal liveReserve;
    uint64 internal liveLtv;
    uint256 internal scIndex;

    // ------------------------------------------------------------------ setup
    function _setup(bool twoLoans) internal {
        vm.createSelectFork(RPC, BLOCK);
        T0 = block.timestamp + 4000;
        nV = twoLoans ? 4 : 3;
        nL = twoLoans ? 2 : 1;
        venuesBase = uint256(keccak256(abi.encode(uint256(keccak256(abi.encode(POOL, uint256(1)))) + 3)));
        loan1 = new R2Token(18);
        for (uint256 i = 1; i < 4; ++i) oracles[i] = new R2Oracle();
        oracles[1].set(ORACLE0, 0);
        oracles[2].set(ORACLE0, 0);
        oracles[3].set(P1 * 1e36, 0);
        params[0] = R2Params(USDC, CBBTC, LIVE_ORACLE, IRM, 0.86e18);
        params[1] = R2Params(USDC, CBBTC, address(oracles[1]), IRM, 0.77e18);
        params[2] = R2Params(USDC, CBBTC, address(oracles[2]), address(0), 0.915e18);
        params[3] = R2Params(address(loan1), CBBTC, address(oracles[3]), IRM, 0.86e18);
        for (uint256 i; i < 4; ++i) ids[i] = keccak256(abi.encode(params[i]));
        require(ids[0] == LIVE_ID, "live id");
        R2Morpho(MORPHO).createMarket(params[1]);
        R2Morpho(MORPHO).createMarket(params[2]);
        if (twoLoans) R2Morpho(MORPHO).createMarket(params[3]);
        vm.startPrank(POOL);
        IMMRouter(ROUTER).addVenue(0, 0, abi.encode(params[1]), true, true, P0);
        IMMRouter(ROUTER).addVenue(0, 0, abi.encode(params[2]), true, true, P0);
        if (twoLoans) {
            IMMRouter(ROUTER).addLoanAsset(address(loan1));
            IMMRouter(ROUTER).addVenue(0, 1, abi.encode(params[3]), true, true, P1);
        }
        vm.stopPrank();
        for (uint256 i; i < nV; ++i) accts[i] = IMMRouter(ROUTER).venue(POOL, uint16(i)).account;
        // live figures before the pool's code is replaced
        (live.tsa, live.tss, live.tba, live.tbs,,) = R2Morpho(MORPHO).market(LIVE_ID);
        (live.sup, live.bor, live.coll) = R2Morpho(MORPHO).position(LIVE_ID, ACCOUNT);
        live.rat = R2Irm(IRM).rateAtTarget(LIVE_ID);
        live.mColl = uint256(vm.load(ROUTER, bytes32(venuesBase + 4)));
        live.mShares = uint256(vm.load(ROUTER, bytes32(venuesBase + 5)));
        vm.etch(POOL, address(new R2PoolHarness()).code);
        if (twoLoans) R2Harness(POOL).pushLoan(address(loan1), 18);
        uint256[] memory lq;
        uint256[] memory rs;
        (livePhysical,, liveLtv,,, lq, rs,) = R2Harness(POOL).st();
        liveLiquid = lq[0];
        liveReserve = rs[0];
        deal(USDC, POOL, 1e30);
        deal(CBBTC, POOL, 1e30);
        loan1.mint(POOL, 1e40);
        deal(USDC, MORPHO, 1e30);
        deal(CBBTC, MORPHO, 1e30);
        loan1.mint(MORPHO, 1e40);
        deal(USDC, USER, 1e30);
        deal(CBBTC, USER, 1e30);
        loan1.mint(USER, 1e40);
        vm.startPrank(USER);
        IERC20(USDC).approve(POOL, type(uint256).max);
        IERC20(CBBTC).approve(POOL, type(uint256).max);
        loan1.approve(POOL, type(uint256).max);
        vm.stopPrank();
        baseSnap = vm.snapshotState();
    }

    // ------------------------------------------------------------------ scenario application
    function _apply(Sc memory s) internal {
        vm.revertToState(baseSnap);
        vm.clearMockedCalls();
        vm.startPrank(POOL);
        for (uint256 i; i < nV; ++i) {
            IMMRouter(ROUTER).setVenueFlags(uint16(i), s.v[i].be, s.v[i].se);
            IMMRouter(ROUTER).setVenueCaps(uint16(i), s.v[i].debtCap, s.v[i].supplyCap, s.v[i].maxRate);
        }
        for (uint256 k; k < nL; ++k) IMMRouter(ROUTER).setLoanCaps(uint8(k), s.lDebtCap[k], s.lSupCap[k], s.lBorrow[k]);
        if (nL == 2) IMMRouter(ROUTER).setMaxDrawnAssets(s.maxDrawn);
        if (s.bo.length != 0) IMMRouter(ROUTER).setPriorities(s.bo, s.so, s.wo, s.ro);
        vm.stopPrank();
        for (uint256 i; i < nV; ++i) {
            V memory v = s.v[i];
            writeMarket(ids[i], v.tsa, v.tss, v.tba, v.tbs, uint128(T0 - v.elapsed), v.fee);
            writePosition(ids[i], accts[i], v.sup, v.bor, v.coll);
            if (params[i].irm != address(0)) writeRat(ids[i], v.rat);
            if (i == 0) {
                if (v.oracleMode == 1) vm.mockCallRevert(LIVE_ORACLE, abi.encodeWithSignature("price()"), "oracle down");
                else vm.mockCall(LIVE_ORACLE, abi.encodeWithSignature("price()"), abi.encode(v.oracleMode == 2 ? 0 : v.oracleP));
            } else {
                oracles[i].set(v.oracleP, v.oracleMode);
            }
            if (v.irmDead) {
                vm.mockCallRevert(IRM, abi.encodeWithSelector(R2Irm.borrowRateView.selector, params[i]), "irm down");
                vm.mockCallRevert(IRM, abi.encodeWithSelector(R2Irm.borrowRate.selector, params[i]), "irm down");
            }
            vm.store(ROUTER, bytes32(venuesBase + 6 * i + 4), bytes32(v.mColl));
            vm.store(ROUTER, bytes32(venuesBase + 6 * i + 5), bytes32(v.mShares));
        }
        uint256 p0 = uint256(keccak256(abi.encode(POOL, uint256(1))));
        uint256 w0 = uint256(vm.load(ROUTER, bytes32(p0)));
        w0 = (w0 & ~(uint256(type(uint64).max) << 160)) | (uint256(s.pin) << 160);
        vm.store(ROUTER, bytes32(p0), bytes32(w0));
        uint256 w1 = uint256(vm.load(ROUTER, bytes32(p0 + 1)));
        w1 = (w1 & ~(uint256(type(uint64).max) << 64)) | (uint256(s.band) << 64);
        vm.store(ROUTER, bytes32(p0 + 1), bytes32(w1));
        vm.store(ROUTER, bytes32(0), bytes32(uint256(s.paused ? 1 : 0)));
        R2Harness(POOL).setBook(s.physical, s.features, s.ltv);
        for (uint256 k; k < nL; ++k) R2Harness(POOL).setLoan(k, s.liquid[k], s.reserve[k]);
        address[2] memory toks = [USDC, address(loan1)];
        for (uint256 k; k < nL; ++k) {
            vm.mockCall(FEED, abi.encodeCall(IPriceFeed.peekCross, (CBBTC, toks[k])), abi.encode(s.feedOk[k], s.feedP[k], uint48(T0)));
            vm.mockCall(FEED, abi.encodeCall(IPriceFeed.peekUsd, (toks[k])), abi.encode(s.usdOk[k], s.usd[k], uint48(T0)));
        }
        vm.warp(T0);
        (uint64 pin_,, uint64 band_) = IMMRouter(ROUTER).pin(POOL);
        require(pin_ == s.pin && band_ == s.band && IMMRouter(ROUTER).globalPaused() == s.paused, "router record");
        cur = s;
    }

    // ------------------------------------------------------------------ dumps
    function _venueJson(uint256 i) internal view returns (string memory o) {
        IMMRouter.VenueView memory vv = IMMRouter(ROUTER).venue(POOL, uint16(i));
        (bool ook, uint256 op) = R2Account(accts[i]).oraclePrice(ids[i]);
        o = "{";
        o = add(o, "loanIndex", q(vv.loanIndex));
        o = add(o, "lltv", q(vv.lltvWad));
        o = add(o, "borrowEnabled", qb(vv.borrowEnabled));
        o = add(o, "supplyEnabled", qb(vv.supplyEnabled));
        o = add(o, "retired", qb(vv.retired));
        o = add(o, "debtCap", q(vv.debtCap));
        o = add(o, "supplyCap", q(vv.supplyCap));
        o = add(o, "maxRate", q(vv.maxBorrowRateWad));
        o = add(o, "managedColl", q(uint256(vm.load(ROUTER, bytes32(venuesBase + 6 * i + 4)))));
        o = add(o, "managedShares", q(uint256(vm.load(ROUTER, bytes32(venuesBase + 6 * i + 5)))));
        o = add(o, "hasIrm", qb(params[i].irm != address(0)));
        o = add(o, "irmDead", qb(cur.v[i].irmDead));
        o = add(o, "rat", qi(params[i].irm != address(0) ? R2Irm(IRM).rateAtTarget(ids[i]) : int256(0)));
        o = add(o, "oracleOk", qb(ook));
        o = add(o, "oraclePrice", q(op));
        o = add(o, "oracleZero", qb(cur.v[i].oracleMode == 2));
        o = add(o, "market", marketJson(ids[i]));
        o = end(add(o, "position", positionJson(ids[i], accts[i])));
    }

    function _routerJson() internal view returns (string memory o) {
        (uint64 pin_, uint64 gap_, uint64 band_) = IMMRouter(ROUTER).pin(POOL);
        (uint16[] memory bo, uint16[] memory so, uint16[] memory wo, uint16[] memory ro) = IMMRouter(ROUTER).priorities(POOL);
        o = "{";
        o = add(o, "paused", qb(IMMRouter(ROUTER).globalPaused()));
        o = add(o, "pin", q(pin_));
        o = add(o, "gap", q(gap_));
        o = add(o, "band", q(band_));
        o = add(o, "maxDrawn", q(IMMRouter(ROUTER).maxDrawnAssets(POOL)));
        o = add(o, "bo", ords(bo));
        o = add(o, "so", ords(so));
        o = add(o, "wo", ords(wo));
        o = add(o, "ro", ords(ro));
        string memory ls = "[";
        for (uint256 k; k < nL; ++k) {
            IMMRouter.LoanView memory l = IMMRouter(ROUTER).loan(POOL, uint8(k));
            string memory x = "{";
            x = add(x, "decimals", q(l.decimals));
            x = add(x, "scale", q(l.loanScale));
            x = add(x, "debtCap", q(l.debtCap));
            x = add(x, "supplyCap", q(l.supplyCap));
            x = add(x, "borrowEnabled", qb(l.borrowEnabled));
            x = end(add(x, "retired", qb(l.retired)));
            ls = string.concat(ls, k == 0 ? "" : ",", x);
        }
        o = add(o, "loans", string.concat(ls, "]"));
        string memory vs = "[";
        for (uint256 i; i < nV; ++i) vs = string.concat(vs, i == 0 ? "" : ",", _venueJson(i));
        o = end(add(o, "venues", string.concat(vs, "]")));
    }

    function _poolJson() internal view returns (string memory o) {
        (uint256 physical, uint256 features, uint64 ltv, uint64 phi, uint64 eps, uint256[] memory lq, uint256[] memory rs, uint256[] memory sc) =
            R2Harness(POOL).st();
        uint256[] memory pw = new uint256[](nL);
        uint256[] memory cw = new uint256[](nL);
        for (uint256 k; k < nL; ++k) {
            pw[k] = cur.feedOk[k] ? cur.feedP[k] : 0;
            cw[k] = k == 0 ? 1e18 : ((cur.usdOk[0] && cur.usdOk[k] && cur.usd[0] != 0) ? Math.mulDiv(cur.usd[k], 1e18, cur.usd[0]) : 0);
        }
        o = "{";
        o = add(o, "physical", q(physical));
        o = add(o, "features", q(features));
        o = add(o, "ltv", q(ltv));
        o = add(o, "phi", q(phi));
        o = add(o, "eps", q(eps));
        o = add(o, "liquid", arr(lq));
        o = add(o, "reserve", arr(rs));
        o = add(o, "scale", arr(sc));
        o = add(o, "priceWad", arr(pw));
        o = end(add(o, "crossWad", arr(cw)));
    }

    function _stateJson() internal view returns (string memory) {
        return string.concat('{"now":', q(block.timestamp), ',"pool":', _poolJson(), ',"router":', _routerJson(), "}");
    }

    function _prices() internal view returns (uint256[] memory pw) {
        pw = new uint256[](nL);
        for (uint256 k; k < nL; ++k) pw[k] = cur.feedOk[k] ? cur.feedP[k] : 0;
    }

    // ------------------------------------------------------------------ ops
    function _opRow(string memory op, string memory args, bool ok, bytes memory ret, bool mutates) internal {
        string memory o = "{";
        o = add(o, "s", vm.toString(scIndex));
        o = add(o, "op", qs(op));
        o = add(o, "args", args);
        o = add(o, "ok", qb(ok));
        o = add(o, "ret", qh(ret));
        if (mutates && ok) o = add(o, "post", _stateJson());
        row(end(o));
    }

    function _a(uint256[] memory xs) internal pure returns (string memory) {
        return arr(xs);
    }

    function _u(uint256 a) internal pure returns (uint256[] memory x) {
        x = new uint256[](1);
        x[0] = a;
    }

    function _u(uint256 a, uint256 b) internal pure returns (uint256[] memory x) {
        x = new uint256[](2);
        (x[0], x[1]) = (a, b);
    }

    function _u(uint256 a, uint256 b, uint256 c) internal pure returns (uint256[] memory x) {
        x = new uint256[](3);
        (x[0], x[1], x[2]) = (a, b, c);
    }

    function _u(uint256 a, uint256 b, uint256 c, uint256 d) internal pure returns (uint256[] memory x) {
        x = new uint256[](4);
        (x[0], x[1], x[2], x[3]) = (a, b, c, d);
    }

    function opSell(uint8 idx, uint256 price, uint256 used, uint256 net) public {
        uint256 snap = vm.snapshotState();
        vm.prank(USER);
        (bool ok, bytes memory ret) = POOL.call(abi.encodeCall(R2Harness.sell, (idx, price, used, net)));
        _opRow("sell", _a(_u(idx, price, used, net)), ok, ret, true);
        vm.revertToState(snap);
    }

    function opBuy(uint8 idx, uint256 used, uint256 net) public {
        uint256 snap = vm.snapshotState();
        vm.prank(USER);
        (bool ok, bytes memory ret) = POOL.call(abi.encodeCall(R2Harness.buy, (idx, used, net)));
        _opRow("buy", _a(_u(idx, used, net)), ok, ret, true);
        vm.revertToState(snap);
    }

    function opRelease() public {
        uint256 snap = vm.snapshotState();
        (bool ok, bytes memory ret) = POOL.call(abi.encodeCall(R2Harness.release, ()));
        _opRow("release", "[]", ok, ret, true);
        vm.revertToState(snap);
    }

    function opTake(uint8 idx, uint256 used) public {
        uint256 snap = vm.snapshotState();
        vm.prank(USER);
        (bool ok, bytes memory ret) = POOL.call(abi.encodeCall(R2Harness.take, (idx, used)));
        _opRow("take", _a(_u(idx, used)), ok, ret, true);
        vm.revertToState(snap);
    }

    function opPay(uint8 idx, uint256 price, uint256 net) public {
        uint256 snap = vm.snapshotState();
        vm.prank(USER);
        (bool ok, bytes memory ret) = POOL.call(abi.encodeCall(R2Harness.pay, (idx, price, net)));
        _opRow("pay", _a(_u(idx, price, net)), ok, ret, true);
        vm.revertToState(snap);
    }

    function opRouter(string memory name, string memory args, bytes memory data) public {
        uint256 snap = vm.snapshotState();
        R2Harness(POOL).approveRouter(USDC);
        R2Harness(POOL).approveRouter(CBBTC);
        R2Harness(POOL).approveRouter(address(loan1));
        vm.prank(POOL);
        (bool ok, bytes memory ret) = ROUTER.call(data);
        _opRow(name, args, ok, ret, true);
        vm.revertToState(snap);
    }

    function _collBor(uint256 i) internal view returns (uint256 coll, uint128 bor) {
        (, bor, coll) = R2Morpho(MORPHO).position(ids[i], accts[i]);
    }

    /// @dev MMRouter.repay then a proportional withdrawCollateral in the SAME transaction (the snapshot is live).
    function opRepayWithdraw(uint16 id, uint256 repayAssets, uint256 collAssets) public {
        uint256 snap = vm.snapshotState();
        R2Harness(POOL).approveRouter(USDC);
        R2Harness(POOL).approveRouter(address(loan1));
        vm.prank(POOL);
        (bool ok, bytes memory ret) = ROUTER.call(abi.encodeCall(IMMRouter.repay, (id, repayAssets)));
        if (ok) {
            vm.prank(POOL);
            (ok, ret) = ROUTER.call(abi.encodeCall(IMMRouter.withdrawCollateral, (id, collAssets, 0, true)));
        }
        _opRow("repayWithdraw", _a(_u(id, repayAssets, collAssets)), ok, ret, true);
        vm.revertToState(snap);
    }

    function opView(string memory name, string memory args, address target, bytes memory data) public {
        (bool ok, bytes memory ret) = target.staticcall(data);
        _opRow(name, args, ok, ret, false);
    }

    function _fc(uint8 idx, uint256 coll, uint256 price) internal view returns (uint256 fc) {
        (bool ok, bytes memory ret) =
            ROUTER.staticcall(abi.encodeCall(IMMRouter.fundingCeiling, (POOL, idx, coll, price)));
        if (ok) fc = abi.decode(ret, (uint256));
    }

    function _recl() internal view returns (uint256 r) {
        (bool ok, bytes memory ret) = ROUTER.staticcall(abi.encodeCall(IMMRouter.reclaimable, (POOL, _prices())));
        if (ok) r = abi.decode(ret, (uint256));
    }

    function _debt(uint8 idx) internal view returns (uint256 d) {
        (bool ok, bytes memory ret) = ROUTER.staticcall(abi.encodeCall(IMMRouter.position, (POOL, idx)));
        if (ok) (,, d) = abi.decode(ret, (uint256, uint256, uint256));
    }

    /// @dev Every op family over one scenario state, amounts placed on the thresholds the state implies.
    function runScenario(Sc memory s, uint256 ordinal) external {
        _apply(s);
        scIndex = ordinal;
        string memory o = "{";
        o = add(o, "scenario", qs(s.tag));
        o = add(o, "s", vm.toString(ordinal));
        o = end(add(o, "state", _stateJson()));
        row(o);
        uint256[] memory prices = _prices();
        // views
        opView("positions", "[]", ROUTER, abi.encodeCall(IMMRouter.positions, (POOL)));
        opView("drawn", "[]", ROUTER, abi.encodeCall(IMMRouter.drawnAssets, (POOL)));
        opView("minLltv", "[]", ROUTER, abi.encodeCall(IMMRouter.minLltv, (POOL)));
        opView("reclaimable", "[]", ROUTER, abi.encodeCall(IMMRouter.reclaimable, (POOL, prices)));
        for (uint256 k; k < nL; ++k) {
            opView("quarantine", _a(_u(k)), ROUTER, abi.encodeCall(IMMRouter.quarantine, (POOL, uint8(k))));
            opView("fundingCeiling", _a(_u(k, s.physical, prices[k])), ROUTER, abi.encodeCall(IMMRouter.fundingCeiling, (POOL, uint8(k), s.physical, prices[k])));
            opView("fundingCeiling", _a(_u(k, s.physical + 1e8, prices[k])), ROUTER, abi.encodeCall(IMMRouter.fundingCeiling, (POOL, uint8(k), s.physical + 1e8, prices[k])));
        }
        opView("totalAssets", "[]", POOL, abi.encodeCall(R2Harness.gate, (SEL_TOTAL)));
        opView("assertGate", "[]", POOL, abi.encodeCall(R2Harness.gate, (SEL_ASSERT)));
        (bool aok, bytes memory aret) = POOL.staticcall(abi.encodeCall(R2Harness.gate, (SEL_ANCHOR)));
        _opRow("anchor", "[]", aok, aret, false);
        if (aok) {
            (int256[] memory u0, uint256 g0, bool q0) = abi.decode(abi.decode(aret, (bytes)), (int256[], uint256, bool));
            string memory ea = string.concat('{"u0":', arri(u0), ',"g0":', q(g0), ',"q0":', qb(q0), "}");
            opView("entryGate", ea, POOL, abi.encodeCall(R2Harness.entryGate, (u0, g0, q0)));
            opView("exitGate", ea, POOL, abi.encodeCall(R2Harness.exitGate, (u0, g0)));
            // synthetic anchors: zero anchor with zero gross, a unit anchor, and the anchor one wei lower
            int256[3][] memory syn = new int256[3][](u0.length);
            for (uint256 k; k < u0.length; ++k) syn[k] = [int256(0), int256(1), u0[k] - 1];
            uint256[3] memory gs = [uint256(0), g0, g0];
            for (uint256 m; m < 3; ++m) {
                int256[] memory ux = new int256[](u0.length);
                for (uint256 k; k < u0.length; ++k) ux[k] = syn[k][m];
                for (uint256 qq; qq < 2; ++qq) {
                    string memory sa = string.concat('{"u0":', arri(ux), ',"g0":', q(gs[m]), ',"q0":', qb(qq == 1), "}");
                    opView("entryGate", sa, POOL, abi.encodeCall(R2Harness.entryGate, (ux, gs[m], qq == 1)));
                }
                string memory sb = string.concat('{"u0":', arri(ux), ',"g0":', q(gs[m]), ',"q0":false}');
                opView("exitGate", sb, POOL, abi.encodeCall(R2Harness.exitGate, (ux, gs[m])));
            }
        }
        uint256 recl = _recl();
        for (uint256 k; k < nL; ++k) {
            uint8 idx = uint8(k);
            uint256 L = s.liquid[k];
            uint256 debt = _debt(idx);
            uint256[3] memory useds = [uint256(1), 5000, 400000];
            for (uint256 j; j < 3; ++j) {
                uint256 fc = _fc(idx, s.physical + useds[j], prices[k]);
                uint256[7] memory nets = [uint256(1), L, L + 1, fc == 0 ? L + 2 : L + fc - 1, L + fc, L + fc + 1, L + fc / 2 + 1];
                for (uint256 m; m < 7; ++m) {
                    if (m > 0 && nets[m] == nets[m - 1]) continue;
                    opSell(idx, prices[k], useds[j], nets[m]);
                }
            }
            uint256 fc0 = _fc(idx, s.physical + 5000, prices[k]);
            opSell(idx, prices[k] * 97 / 100, 5000, L + fc0 / 2 + 1);
            opSell(idx, prices[k] * 1015 / 1000, 5000, L + fc0 / 2 + 1);
            opSell(idx, 0, 5000, L + 1);
            uint256[4] memory buyUsed = [uint256(1), debt == 0 ? 7 : debt, debt + 1, 3e11];
            for (uint256 j; j < 4; ++j) {
                uint256[6] memory nets = [uint256(1), s.physical, s.physical + 1, s.physical + (recl == 0 ? 2 : recl - 1), s.physical + recl, s.physical + recl + 1];
                for (uint256 m; m < 6; ++m) {
                    if (m > 0 && nets[m] == nets[m - 1]) continue;
                    opBuy(idx, buyUsed[j], nets[m]);
                }
            }
            opTake(idx, 0);
            opTake(idx, 1);
            opTake(idx, debt);
            opTake(idx, debt + 12345);
            opPay(idx, prices[k], L);
            opPay(idx, prices[k], L + 1);
            opPay(idx, prices[k], L + fc0);
            opPay(idx, prices[k], L + fc0 + 1);
            // Router entries: fund / cascades / reclaim at the thresholds, plus their revert-order edges
            opRouter("fund", _a(_u(k, fc0, s.physical + 5000, prices[k])), abi.encodeCall(IMMRouter.fund, (idx, fc0, POOL, s.physical + 5000, prices[k])));
            opRouter("fund", _a(_u(k, fc0 + 1, s.physical + 5000, prices[k])), abi.encodeCall(IMMRouter.fund, (idx, fc0 + 1, POOL, s.physical + 5000, prices[k])));
            opRouter("repayCascade", _a(_u(k, debt)), abi.encodeCall(IMMRouter.repayCascade, (idx, debt)));
            opRouter("repayCascade", _a(_u(k, debt / 3 + 1)), abi.encodeCall(IMMRouter.repayCascade, (idx, debt / 3 + 1)));
            opRouter("supplyCascade", _a(_u(k, 25e9)), abi.encodeCall(IMMRouter.supplyCascade, (idx, 25e9)));
        }
        opRelease();
        opRouter("reclaim", _a(_u(recl)), abi.encodeCall(IMMRouter.reclaim, (recl, prices)));
        opRouter("reclaim", _a(_u(recl + 1)), abi.encodeCall(IMMRouter.reclaim, (recl + 1, prices)));
        opRouter("reclaimBestEffort", _a(_u(recl + 1)), abi.encodeCall(IMMRouter.reclaimBestEffort, (recl + 1, prices)));
        for (uint256 i; i < nV; ++i) {
            uint256 pr = prices[params[i].loanToken == USDC ? 0 : 1];
            opRouter("borrow", _a(_u(i, type(uint256).max, pr)), abi.encodeCall(IMMRouter.borrow, (uint16(i), type(uint256).max, POOL, pr)));
            opRouter("borrow", _a(_u(i, 1e6, pr)), abi.encodeCall(IMMRouter.borrow, (uint16(i), 1e6, POOL, pr)));
            opRouter("withdrawSupplied", _a(_u(i, 0)), abi.encodeCall(IMMRouter.withdrawSupplied, (uint16(i), 0, POOL)));
            opRouter("withdrawSupplied", _a(_u(i, type(uint256).max)), abi.encodeCall(IMMRouter.withdrawSupplied, (uint16(i), type(uint256).max, POOL)));
            (uint256 coll_, uint128 bor_) = _collBor(i);
            opRouter("postCollateral", _a(_u(i, 12345)), abi.encodeCall(IMMRouter.postCollateral, (uint16(i), 12345)));
            opRouter("withdrawCollateral", _a(_u(i, coll_ / 7 + 1, pr, 0)), abi.encodeCall(IMMRouter.withdrawCollateral, (uint16(i), coll_ / 7 + 1, pr, false)));
            opRouter("withdrawCollateral", _a(_u(i, coll_ / 7 + 1, pr, 1)), abi.encodeCall(IMMRouter.withdrawCollateral, (uint16(i), coll_ / 7 + 1, pr, true)));
            opRouter("supply", _a(_u(i, 3e9)), abi.encodeCall(IMMRouter.supply, (uint16(i), 3e9)));
            opRepayWithdraw(uint16(i), bor_ == 0 ? 1 : 3e9, coll_ / 3 + 1);
            opRepayWithdraw(uint16(i), type(uint256).max, coll_);
        }
    }

    // ------------------------------------------------------------------ scenarios
    function _liveV() internal view returns (V memory v) {
        v = live;
        v.elapsed = 600;
        v.oracleP = ORACLE0;
        v.be = true;
        v.se = true;
    }

    function _mk(uint128 tsa, uint128 tba, uint64 elapsed, uint128 fee, int256 rat) internal pure returns (V memory v) {
        v.tsa = tsa;
        v.tss = tsa * 1e6 + 12345;
        v.tba = tba;
        v.tbs = tba * 1e6 + 999;
        v.elapsed = elapsed;
        v.fee = fee;
        v.rat = rat;
        v.oracleP = ORACLE0;
        v.be = true;
        v.se = true;
    }

    function _base() internal view returns (Sc memory s) {
        s.tag = "base";
        s.v[0] = _liveV();
        s.v[1] = _mk(5e12, 4e12, 900, 0.1e18, INIT_RAT * 3);
        s.v[2] = _mk(1e12, 5e11, 300, 0, 0);
        if (nV == 4) {
            s.v[3] = _mk(2e24, 1.5e24, 1200, 0.05e18, INIT_RAT);
            s.v[3].oracleP = P1 * 1e36;
        }
        s.pin = 0.55e18;
        s.band = 0.02e18;
        s.maxDrawn = 1;
        s.lBorrow = [true, true];
        s.physical = livePhysical;
        s.liquid = [liveLiquid, uint256(0)];
        s.reserve = [liveReserve, uint256(0)];
        s.features = 63;
        s.ltv = liveLtv;
        s.feedP = [P0, P1];
        s.feedOk = [true, true];
        s.usd = [uint256(1e18), 3000e18];
        s.usdOk = [true, true];
    }

    function _pos(V memory v, uint256 supAssets, uint256 debtAssets, uint128 coll) internal pure returns (V memory) {
        v.sup = supAssets * v.tss / (uint256(v.tsa) + 1);
        v.bor = uint128(debtAssets * (uint256(v.tbs) + 1e6) / (uint256(v.tba) + 1));
        v.coll = coll;
        v.mColl = coll;
        v.mShares = v.sup;
        return v;
    }

    /// @dev A fresh memory struct each call (scenarios must not alias one another).
    function _indebted() internal view returns (Sc memory s) {
        s = _base();
        s.tag = "indebted-multi";
        s.v[1] = _pos(s.v[1], 20_000e6, 30_000e6, 1e8);
        s.v[2] = _pos(s.v[2], 0, 1_000e6, 5e6);
        s.physical = 3e7;
    }

    function _scenarios() internal view returns (Sc[] memory out) {
        out = new Sc[](22);
        Sc memory s;
        out[0] = _base();

        s = _base();
        s.tag = "supplied";
        s.v[1] = _pos(s.v[1], 100_000e6, 0, 0);
        s.v[2] = _pos(s.v[2], 5_000e6, 0, 0);
        s.liquid[0] = 0;
        out[1] = s;

        out[2] = _indebted();

        s = _indebted();
        s.tag = "managed-low";
        s.v[1].mColl = s.v[1].coll / 2;
        s.v[1].mShares = s.v[1].sup / 3;
        s.v[2].mColl = s.v[2].coll + 1;
        out[3] = s;

        s = _indebted();
        s.tag = "rate-caps";
        s.v[0].maxRate = 1;
        s.v[1].maxRate = uint64(uint256(INIT_RAT) * 3 * 4);
        s.v[2].maxRate = 1;
        out[4] = s;

        s = _indebted();
        s.tag = "quarantine-v1";
        s.v[1].irmDead = true;
        s.v[1].elapsed = 7200;
        out[5] = s;

        s = _indebted();
        s.tag = "grace-v1";
        s.v[1].irmDead = true;
        s.v[1].elapsed = 1800;
        out[6] = s;

        s = _indebted();
        s.tag = "oracles";
        s.v[2].oracleMode = 1;
        s.v[1].oracleMode = 2;
        out[7] = s;

        s = _indebted();
        s.tag = "caps-tight";
        s.v[1].debtCap = 30_000e6 + 5e6;
        s.v[1].supplyCap = 20_000e6 + 3e6;
        s.v[2].debtCap = 900e6;
        s.lDebtCap[0] = uint128(31_020e6);
        s.lSupCap[0] = uint128(20_010e6);
        out[8] = s;

        s = _indebted();
        s.tag = "priorities";
        s.bo = new uint16[](nV);
        s.so = new uint16[](nV);
        s.wo = new uint16[](nV);
        s.ro = new uint16[](nV);
        for (uint256 i; i < nV; ++i) {
            s.bo[i] = uint16(nV - 1 - i);
            s.so[i] = uint16((i + 1) % nV);
            s.wo[i] = uint16((i + 2) % nV);
            s.ro[i] = uint16(nV - 1 - i);
        }
        out[9] = s;

        s = _indebted();
        s.tag = "paused";
        s.paused = true;
        out[10] = s;

        s = _indebted();
        s.tag = "excess-collateral";
        s.v[0].coll = 5e6;
        s.v[0].mColl = 5e6;
        s.v[1].coll = 4e8;
        s.v[1].mColl = 4e8;
        s.v[2].coll = 3e7;
        s.v[2].mColl = 3e7;
        s.physical = 2000;
        out[11] = s;

        s = _indebted();
        s.tag = "over-pin";
        s.ltv = 0.05e18;
        out[12] = s;

        s = _indebted();
        s.tag = "feed-unchecked";
        s.feedOk[0] = false;
        out[13] = s;

        for (uint256 k; k < 3; ++k) {
            s = _indebted();
            s.tag = string.concat("grace-edge-", vm.toString(3599 + k));
            s.v[1].irmDead = true;
            s.v[1].elapsed = uint64(3599 + k);
            s.v[1].sup = 0;
            s.v[1].mShares = 0;
            out[14 + k] = s;
        }

        // exposure exactly on the bound: a round cross, one exact-share debt, gross = U / ltv, physical -1 / 0 / +1
        for (uint256 k; k < 3; ++k) {
            s = _base();
            s.tag = string.concat("gate-equal-", vm.toString(k));
            s.feedP[0] = 1e15;
            s.v[0].bor = 0;
            s.v[0].coll = 0;
            s.v[0].mColl = 0;
            s.v[0].oracleP = 1e39;
            s.v[1].oracleP = 1e39;
            s.v[2].oracleP = 1e39;
            s.v[2].tba = 1e12;
            s.v[2].tbs = 1e18;
            s.v[2].tsa = 2e12;
            s.v[2].tss = 2e18;
            s.v[2].bor = uint128(5e11 * 1e6);
            s.v[2].coll = 9e8;
            s.v[2].mColl = 9e8;
            s.liquid[0] = 0;
            s.ltv = 0.5e18;
            s.physical = 1e8 + k - 1;
            out[17 + k] = s;
        }

        s = _indebted();
        s.tag = "grace-rate-cap";
        s.v[1].irmDead = true;
        s.v[1].elapsed = 1800;
        s.v[1].maxRate = type(uint64).max;
        s.v[0].maxRate = type(uint64).max;
        out[20] = s;

        s = _indebted();
        s.tag = "quarantine-rate-cap-supplied";
        s.v[1].irmDead = true;
        s.v[1].elapsed = 7200;
        s.v[1].maxRate = type(uint64).max;
        s.v[2] = _pos(s.v[2], 40_000e6, 1_000e6, 5e6);
        s.v[2].maxRate = 1;
        s.liquid[0] = 7e6;
        out[21] = s;
    }

    function _random(uint256 seed) internal view returns (Sc memory s) {
        s = _base();
        s.tag = string.concat("random-", vm.toString(seed));
        for (uint256 i; i < nV; ++i) {
            uint256 b = seed * 100 + i * 10;
            uint128 tsa = uint128(rlog(b, 1, 1e9, i == 3 ? 1e25 : 5e13));
            uint128 tba = uint128(uint256(tsa) * (rnd(b, 2) % 1001) / 1000);
            V memory v = _mk(tsa, tba, uint64(rnd(b, 3) % 3 == 0 ? rnd(b, 50) % 90000 : rnd(b, 51) % 5000), rnd(b, 4) % 3 == 0 ? uint128(rnd(b, 5) % 0.25e18) : 0, i == 2 ? int256(0) : MIN_RAT + int256(rnd(b, 6) % uint256(MAX_RAT - MIN_RAT)));
            if (i == 3) v.oracleP = P1 * 1e36;
            v.oracleP = v.oracleP * (9900 + rnd(b, 7) % 250) / 10000;
            v.oracleMode = rnd(b, 8) % 11 == 0 ? uint8(1 + rnd(b, 9) % 2) : 0;
            v.irmDead = i != 2 && rnd(b, 10) % 5 == 0;
            v = _pos(v, rnd(b, 11) % 2 == 0 ? uint256(tsa) * (rnd(b, 12) % 300) / 1000 : 0, uint256(tba) * (rnd(b, 13) % 500) / 1000, uint128(rlog(b, 14, 0, 2e9)));
            if (rnd(b, 15) % 4 == 0) v.mColl = v.coll * (rnd(b, 16) % 100) / 100;
            if (rnd(b, 17) % 4 == 0) v.mShares = v.sup * (rnd(b, 18) % 100) / 100;
            if (rnd(b, 19) % 3 == 0) v.debtCap = uint128(rlog(b, 20, 1, 1e13));
            if (rnd(b, 21) % 3 == 0) v.supplyCap = uint128(rlog(b, 22, 1, 1e13));
            if (rnd(b, 23) % 3 == 0) v.maxRate = uint64(rlog(b, 24, 1, 3e11));
            v.be = rnd(b, 25) % 6 != 0;
            v.se = rnd(b, 26) % 6 != 0;
            s.v[i] = v;
        }
        s.physical = rlog(seed, 30, 0, 1e9);
        s.liquid[0] = rlog(seed, 31, 0, 1e11);
        s.liquid[1] = rlog(seed, 32, 0, 1e21);
        s.reserve[0] = rlog(seed, 33, 0, 1e11);
        s.reserve[1] = rlog(seed, 34, 0, 1e21);
        s.features = rnd(seed, 35) % 2 == 0 ? 63 : 55;
        s.ltv = uint64(0.2e18 + rnd(seed, 36) % 0.35e18);
        s.maxDrawn = uint8(1 + rnd(seed, 37) % 2);
        if (rnd(seed, 38) % 3 == 0) s.lDebtCap[0] = uint128(rlog(seed, 39, 1, 1e12));
        if (rnd(seed, 40) % 3 == 0) s.lSupCap[0] = uint128(rlog(seed, 41, 1, 1e12));
        s.lBorrow[0] = rnd(seed, 42) % 8 != 0;
        s.lBorrow[1] = rnd(seed, 43) % 8 != 0;
        s.feedOk[1] = rnd(seed, 44) % 9 != 0;
        s.usdOk[1] = rnd(seed, 45) % 9 != 0;
        s.band = rnd(seed, 46) % 4 == 0 ? 0 : 0.02e18;
    }

    /// @dev A scenario row followed by one op, for the threshold sweeps below.
    function runOne(Sc memory s, uint256 ordinal, uint8 kind, uint256 a0, uint256 a1, uint256 a2) external {
        _apply(s);
        scIndex = ordinal;
        string memory o = "{";
        o = add(o, "scenario", qs(s.tag));
        o = add(o, "s", vm.toString(ordinal));
        o = end(add(o, "state", _stateJson()));
        row(o);
        if (kind == 0) opRelease();
        else if (kind == 1) opSell(0, P0, a0, a1);
        else if (kind == 2) opRouter("fund", _a(_u(0, a0, a1, a2)), abi.encodeCall(IMMRouter.fund, (0, a0, POOL, a1, a2)));
        else if (kind == 3) opBuy(0, a0, a1);
        else if (kind == 4) opPay(0, P0, a0);
    }

    /// @dev requiredPostedAll of the current book at the scenario's crosses, from the deployed `priced`.
    function _need(Sc memory s) internal returns (uint256 need, uint256 posted) {
        _apply(s);
        (,,, posted) = IMMRouter(ROUTER).positions(POOL);
        uint256[] memory pw = _prices();
        for (uint256 k; k < nL; ++k) {
            (,, uint256 d) = IMMRouter(ROUTER).position(POOL, uint8(k));
            if (d == 0) continue;
            uint256 sc = k == 0 ? 1e12 : 1;
            need += Math.mulDiv(d * sc, 1e18, uint256(s.ltv) * pw[k], Math.Rounding.Up);
        }
    }

    /// @dev Release hysteresis on a three-venue book: tune one venue's debt until need % 9 == 0, then sweep the
    ///      posted collateral across posted == 10 * excess, where floor(excess * WAD / posted) == 0.1e18 exactly.
    function _sweepRelease(uint256 base) internal returns (uint256 ord) {
        ord = base;
        Sc memory s = _indebted();
        s.tag = "release-sweep";
        s.physical = 1000;
        s.v[1].coll = 6e7;
        s.v[1].mColl = 6e7;
        uint256 need;
        uint256 posted;
        for (uint256 t; t < 40; ++t) {
            (need, posted) = _need(s);

            if (need % 9 == 0) break;
            s.v[2].bor += 4e8;
        }
        require(need % 9 == 0, "sweep need");
        uint256 other = posted - s.v[0].coll;
        uint256 target = need * 10 / 9;
        for (uint256 d; d < 24; ++d) {
            uint256 p = target + d - 12;
            if (p <= other) continue;
            Sc memory x = s;
            x.v[0].coll = uint128(p - other);
            x.v[0].mColl = p - other;
            this.runOne(x, ord++, 0, 0, 0, 0);
            s = x;
        }
    }

    /// @dev The plan prices a same-venue borrow at the pre-withdrawal (unaccrued) state, execution after the
    ///      withdrawal's accrual: a rate ceiling set to exactly the planned rate (and one wei below / above).
    function _rateExec(uint256 base) internal returns (uint256 ord) {
        ord = base;
        Sc memory s = _indebted();
        s.tag = "rate-exec";
        s.v[0].be = false;
        s.v[2].be = false;
        s.v[1] = _mk(5e12, 4.75e12, 86400, 0.1e18, INIT_RAT * 3);
        s.v[1] = _pos(s.v[1], 20_000e6, 30_000e6, 1e8);
        s.liquid[0] = 0;
        _apply(s);
        (, uint256 sup,) = IMMRouter(ROUTER).position(POOL, 0);
        uint256 cash = R2Account(ACCOUNT).freeLiquidity(ids[1]);
        uint256 take = sup < cash ? sup : cash;
        uint256 slice = 5_000e6;
        (bool ok, uint256 planRate) = R2Account(ACCOUNT).borrowRateAfter(ids[1], slice, take);
        require(ok && take != 0, "rate-exec setup");
        uint256 amount = take + slice;
        for (uint256 d; d < 3; ++d) {
            s.v[1].maxRate = uint64(planRate + d - 1);
            this.runOne(s, ord++, 2, amount, s.physical, P0);
            this.runOne(s, ord++, 1, 5000, amount, 0);
            this.runOne(s, ord++, 4, amount, 0, 0);
            this.runOne(s, ord++, 2, take, s.physical, P0);
        }
    }

    function _run(bool twoLoans, string memory file, uint256 nRandom) internal {
        open(file);
        _setup(twoLoans);
        Sc[] memory sc = _scenarios();
        for (uint256 i; i < sc.length; ++i) this.runScenario(sc[i], i);
        for (uint256 i; i < nRandom; ++i) this.runScenario(_random(i + (twoLoans ? 1000 : 0)), sc.length + i);
        if (!twoLoans) {
            uint256 ord = _sweepRelease(sc.length + nRandom);
            _rateExec(ord);
        }
        close();
    }

    function test_settleOneLoan() public virtual {
        _run(false, "router_settlement_edges_one_loan.json", 10);
    }

    function test_settleTwoLoans() public virtual {
        _run(true, "router_settlement_edges_two_loans.json", 8);
    }
}
