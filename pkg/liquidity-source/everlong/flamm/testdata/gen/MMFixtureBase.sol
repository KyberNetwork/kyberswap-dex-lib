// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import {Test} from "forge-std/Test.sol";

struct MParams {
    address loanToken;
    address collateralToken;
    address oracle;
    address irm;
    uint256 lltv;
}

struct MMarket {
    uint128 totalSupplyAssets;
    uint128 totalSupplyShares;
    uint128 totalBorrowAssets;
    uint128 totalBorrowShares;
    uint128 lastUpdate;
    uint128 fee;
}

struct KVenueView {
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

struct KLoanView {
    address token;
    uint8 decimals;
    uint256 loanScale;
    uint128 debtCap;
    uint128 supplyCap;
    bool borrowEnabled;
    bool retired;
    address[4] accounts;
}

struct KDials {
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

struct KLimits {
    uint256 depositCapPoolAsset;
    uint64 roomEpsilonWad;
    uint32 maxPriceAgeSec;
    uint64 feeFloorWad;
    uint64 feeCapWad;
    uint16 performanceFeeBp;
    uint96 peakSharePriceWad;
    uint16 maxPerformanceFeeBp;
}

interface KMorpho {
    function market(bytes32) external view returns (uint128, uint128, uint128, uint128, uint128, uint128);
    function position(bytes32, address) external view returns (uint256, uint128, uint128);
    function idToMarketParams(bytes32) external view returns (MParams memory);
    function createMarket(MParams memory) external;
    function supply(MParams memory, uint256, uint256, address, bytes memory) external returns (uint256, uint256);
    function supplyCollateral(MParams memory, uint256, address, bytes memory) external;
    function borrow(MParams memory, uint256, uint256, address, address) external returns (uint256, uint256);
    function accrueInterest(MParams memory) external;
    function owner() external view returns (address);
    function setFee(MParams memory, uint256) external;
}

interface KIrm {
    function borrowRateView(MParams memory, MMarket memory) external view returns (uint256);
    function borrowRate(MParams memory, MMarket memory) external returns (uint256);
    function rateAtTarget(bytes32) external view returns (int256);
}

interface KRouter {
    function globalPaused() external view returns (bool);
    function pin(address) external view returns (uint64, uint64, uint64);
    function maxDrawnAssets(address) external view returns (uint8);
    function loanCount(address) external view returns (uint256);
    function venueCount(address) external view returns (uint256);
    function loan(address, uint8) external view returns (KLoanView memory);
    function venue(address, uint16) external view returns (KVenueView memory);
    function priorities(address) external view returns (uint16[] memory, uint16[] memory, uint16[] memory, uint16[] memory);
    function positions(address) external view returns (uint256[] memory, uint256[] memory, uint256[] memory, uint256);
    function position(address, uint8) external view returns (uint256, uint256, uint256);
    function venuePosition(address, uint16) external view returns (uint256, uint256, uint256, uint256, uint256);
    function venueReadable(address, uint16) external view returns (bool);
    function quarantine(address, uint8) external view returns (bool, uint256, uint256);
    function drawnAssets(address) external view returns (uint8, uint8);
    function minLltv(address) external view returns (uint256);
    function fundingCeiling(address, uint8, uint256, uint256) external view returns (uint256);
    function reclaimable(address, uint256[] calldata) external view returns (uint256);
    function venueHealth(address, uint16, uint256) external view returns (uint256);
    function venueFreeLiquidity(address, uint16) external view returns (uint256);
    function registerPool(address, address, uint64, uint64, uint64) external;
    function addLoanAsset(address) external returns (uint8);
    function setLoanCaps(uint8, uint128, uint128, bool) external;
    function setMaxDrawnAssets(uint8) external;
    function addVenue(uint8, uint8, bytes calldata, bool, bool, uint256) external returns (uint16);
    function setVenueCaps(uint16, uint128, uint128, uint64) external;
    function setPriorities(uint16[] calldata, uint16[] calldata, uint16[] calldata, uint16[] calldata) external;
    function fund(uint8, uint256, address, uint256, uint256) external returns (uint256, uint256, uint256);
    function repayCascade(uint8, uint256) external returns (uint256);
    function supplyCascade(uint8, uint256) external returns (uint256);
    function reclaim(uint256, uint256[] calldata) external returns (uint256);
    function reclaimBestEffort(uint256, uint256[] calldata) external returns (uint256);
    function postCollateral(uint16, uint256) external;
    function withdrawCollateral(uint16, uint256, uint256, bool) external;
    function borrow(uint16, uint256, address, uint256) external;
    function repay(uint16, uint256) external returns (uint256);
    function supply(uint16, uint256) external returns (uint256);
    function withdrawSupplied(uint16, uint256, address) external returns (uint256);
}

interface KAccount {
    function tryPosition(bytes32) external view returns (bool, uint256, uint256, uint256, uint256);
    function debtOf(bytes32) external view returns (uint256);
    function collateralOf(bytes32) external view returns (uint256);
    function suppliedOf(bytes32) external view returns (uint256);
    function supplySharesToAssets(bytes32, uint256) external view returns (uint256);
    function freeLiquidity(bytes32) external view returns (uint256);
    function borrowRateAfter(bytes32, uint256, uint256) external view returns (bool, uint256);
    function oraclePrice(bytes32) external view returns (bool, uint256);
}

interface KPool {
    function poolAssetPosition() external view returns (uint256, uint256);
    function loanPosition() external view returns (uint256, uint256, uint256);
    function loanCount() external view returns (uint256);
    function loanConfig(uint8)
        external
        view
        returns (address, uint8, uint64, uint64, uint256, uint256, uint256);
    function dials() external view returns (KDials memory);
    function limits() external view returns (KLimits memory);
    function switches() external view returns (uint256, uint32, bool);
    function totalSupply() external view returns (uint256);
    function totalAssets() external view returns (uint256);
    function paused() external view returns (bool);
    function asset() external view returns (address);
    function previewSwap(bool, uint256) external view returns (uint256, uint256, uint256);
    function swap(address, address, uint256, uint256, address, uint256) external returns (uint256, uint256);
}

struct KLensFacts {
    uint256 physical;
    uint256 posted;
    uint256 gross;
    uint256 liquid;
    uint256 supplied;
    uint256 debt;
    uint256 totalSupply;
    bool paused;
    uint48 lastObservationTs;
}

struct KLensVenue {
    uint16 id;
    bytes32 marketId;
    address account;
    uint8 kind;
    uint8 loanIndex;
    uint64 lltvWad;
    bool borrowEnabled;
    bool supplyEnabled;
    bool retired;
    uint128 debtCap;
    uint128 supplyCap;
    uint64 maxBorrowRateWad;
    uint256 collateral;
    uint256 recognizedCollateral;
    uint256 supplied;
    uint256 recognizedSupplied;
    uint256 debt;
    uint256 freeLiquidity;
    bool readable;
    bool healthOk;
    uint256 healthWad;
}

interface KLens {
    function facts(address) external view returns (KLensFacts memory);
    function venues(address) external view returns (KLensVenue[] memory);
}

interface KFeed {
    function peekCross(address, address) external view returns (bool, uint256, uint48);
    function peekUsd(address) external view returns (bool, uint256, uint48);
}

interface KERC20 {
    function balanceOf(address) external view returns (uint256);
    function approve(address, uint256) external returns (bool);
    function transfer(address, uint256) external returns (bool);
    function decimals() external view returns (uint8);
}

/// @dev A pool stand-in for the synthetic multi-venue router: answers the registration bindings and forwards calls.
contract KMockPool {
    address public factory;
    address public router;
    address public asset;

    constructor(address f, address r, address a) {
        factory = f;
        router = r;
        asset = a;
    }

    function exec(address target, bytes calldata data) external returns (bytes memory out) {
        bool ok;
        (ok, out) = target.call(data);
        if (!ok) {
            assembly ("memory-safe") {
                revert(add(out, 32), mload(out))
            }
        }
    }

    function approve(address token, address spender, uint256 amount) external {
        KERC20(token).approve(spender, amount);
    }
}

/// @dev A constant-price Morpho oracle.
contract KConstOracle {
    uint256 public price;

    constructor(uint256 p) {
        price = p;
    }
}

/// @notice JSON writers shared by the financing fixture generators: the router / account / Morpho / IRM state a Go
///         port tracks, and the deployed views it must reproduce.
abstract contract MMFixtureBase is Test {
    address constant CBBTC = 0xcbB7C0000aB88B473b1f5aFd9ef808440eed33Bf;
    address constant USDC = 0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913;
    address constant WETH = 0x4200000000000000000000000000000000000006;
    address constant MORPHO = 0xBBBBBbbBBb9cC5e90e3b3Af64bdAF62C37EEFFCb;
    address constant IRM = 0x46415998764C29aB2a25CbeA6254146D50D22687;
    address constant ROUTER = 0x19A9b39E6710AAD109C829294b0841F0851c6bB4;
    address constant POOL = 0xc0fdCB1799cCc2CEBaA1fe247157b0dF33D57572;
    address constant ACCOUNT = 0x6760E3b032eE2d670Cb684d9076b8f48cb066c48;
    address constant FACTORY = 0x1BfcE014774D0DD7e04bC595D46Fa09F7dCCF45f;
    address constant FEED = 0xbED275459578C87a63F2f50A0b077C720e838816;
    address constant LENS = 0x4E09952F42de3617198fcde3D673AFdb90B49800;
    bytes32 constant MID = 0x9103c3b4e834476c9a62ea009ba2c884ee42e94e6e314a26f04d312434191836;
    string constant RPC = "https://mainnet.base.org";
    uint256 constant WAD = 1e18;

    // ------------------------------------------------------------------ json primitives
    function _u(uint256 v) internal pure returns (string memory) {
        return string.concat('"', vm.toString(v), '"');
    }

    function _i(int256 v) internal pure returns (string memory) {
        return string.concat('"', vm.toString(v), '"');
    }

    function _b(bool v) internal pure returns (string memory) {
        return v ? "true" : "false";
    }

    function _h(bytes memory v) internal pure returns (string memory) {
        return string.concat('"', vm.toString(v), '"');
    }

    function _kv(string memory k, string memory v) internal pure returns (string memory) {
        return string.concat('"', k, '":', v);
    }

    function _ua(uint256[] memory a) internal pure returns (string memory s) {
        s = "[";
        for (uint256 i; i < a.length; ++i) s = string.concat(s, i == 0 ? "" : ",", _u(a[i]));
        s = string.concat(s, "]");
    }

    function _u16a(uint16[] memory a) internal pure returns (string memory s) {
        s = "[";
        for (uint256 i; i < a.length; ++i) s = string.concat(s, i == 0 ? "" : ",", vm.toString(uint256(a[i])));
        s = string.concat(s, "]");
    }

    function _vOrErr(bool ok, uint256 v, bytes memory e) internal pure returns (string memory) {
        return ok ? string.concat('{"v":', _u(v), "}") : string.concat('{"err":', _h(e), "}");
    }

    // ------------------------------------------------------------------ router storage (verified against views)
    function _poolBase(address pool) internal pure returns (uint256) {
        return uint256(keccak256(abi.encode(pool, uint256(1))));
    }

    function _venueSlot(address pool, uint256 i) internal pure returns (uint256) {
        return uint256(keccak256(abi.encode(_poolBase(pool) + 3))) + i * 6;
    }

    function _managed(address router, address pool, uint256 i, KVenueView memory v)
        internal
        view
        returns (uint256 coll, uint256 shares)
    {
        uint256 s = _venueSlot(pool, i);
        require(address(uint160(uint256(vm.load(router, bytes32(s))))) == v.account, "slot:account");
        require(vm.load(router, bytes32(s + 1)) == v.id, "slot:id");
        require(uint64(uint256(vm.load(router, bytes32(s + 3))) >> 128) == v.maxBorrowRateWad, "slot:rate");
        coll = uint256(vm.load(router, bytes32(s + 4)));
        shares = uint256(vm.load(router, bytes32(s + 5)));
    }

    function _setMaxRate(address router, address pool, uint256 i, uint64 rate) internal {
        uint256 s = _venueSlot(pool, i) + 3;
        uint256 w = uint256(vm.load(router, bytes32(s)));
        w = (w & ~(uint256(type(uint64).max) << 128)) | (uint256(rate) << 128);
        vm.store(router, bytes32(s), bytes32(w));
    }

    function _setManaged(address router, address pool, uint256 i, uint256 coll, uint256 shares) internal {
        uint256 s = _venueSlot(pool, i);
        vm.store(router, bytes32(s + 4), bytes32(coll));
        vm.store(router, bytes32(s + 5), bytes32(shares));
    }

    function _market(bytes32 id) internal view returns (MMarket memory m) {
        (m.totalSupplyAssets, m.totalSupplyShares, m.totalBorrowAssets, m.totalBorrowShares, m.lastUpdate, m.fee) =
            KMorpho(MORPHO).market(id);
    }

    function _marketJson(MMarket memory m) internal pure returns (string memory) {
        return string.concat(
            '{"tsa":', _u(m.totalSupplyAssets), ',"tss":', _u(m.totalSupplyShares), ',"tba":', _u(m.totalBorrowAssets),
            ',"tbs":', _u(m.totalBorrowShares), ',"lastUpdate":', _u(m.lastUpdate), ',"fee":', _u(m.fee), "}"
        );
    }

    // ------------------------------------------------------------------ state
    function _venueState(address router, address pool, uint256 i) internal view returns (string memory) {
        KVenueView memory v = KRouter(router).venue(pool, uint16(i));
        (uint256 mc, uint256 ms) = _managed(router, pool, i, v);
        MParams memory p = KMorpho(MORPHO).idToMarketParams(v.id);
        MMarket memory m = _market(v.id);
        (uint256 ss, uint128 bs, uint128 col) = KMorpho(MORPHO).position(v.id, v.account);
        int256 rat = p.irm == address(0) ? int256(0) : KIrm(p.irm).rateAtTarget(v.id);
        bool irmOk = true;
        if (p.irm != address(0)) {
            try KIrm(p.irm).borrowRateView(p, m) returns (uint256) {} catch { irmOk = false; }
        }
        (bool ook, uint256 op) = KAccount(v.account).oraclePrice(v.id);
        string memory cfg = string.concat(
            '{"account":"', vm.toString(v.account), '","id":"', vm.toString(v.id), '","kind":', vm.toString(uint256(v.kind)),
            ',"loanIndex":', vm.toString(uint256(v.loanIndex)), ',"lltvWad":', _u(v.lltvWad), ',"borrowEnabled":',
            _b(v.borrowEnabled), ',"supplyEnabled":', _b(v.supplyEnabled), ',"retired":', _b(v.retired)
        );
        cfg = string.concat(
            cfg, ',"debtCap":', _u(v.debtCap), ',"supplyCap":', _u(v.supplyCap), ',"maxBorrowRateWad":',
            _u(v.maxBorrowRateWad), ',"managedCollateral":', _u(mc), ',"managedSupplyShares":', _u(ms)
        );
        return string.concat(
            cfg, ',"market":', _marketJson(m), ',"position":{"supplyShares":', _u(ss), ',"borrowShares":', _u(bs),
            ',"collateral":', _u(col), '},"rateAtTarget":', _i(rat), ',"hasIrm":', _b(p.irm != address(0)),
            ',"irmReadable":', _b(irmOk), ',"oracleOk":', _b(ook), ',"oraclePrice":', _u(op), "}"
        );
    }

    function _routerState(address router, address pool) internal view returns (string memory s) {
        (uint64 pinLtv, uint64 gap, uint64 band) = KRouter(router).pin(pool);
        s = string.concat(
            '{"globalPaused":', _b(KRouter(router).globalPaused()), ',"pinLtvWad":', _u(pinLtv), ',"safetyGapWad":',
            _u(gap), ',"oracleBandWad":', _u(band), ',"maxDrawnAssets":',
            vm.toString(uint256(KRouter(router).maxDrawnAssets(pool))), ',"loans":['
        );
        uint256 nl = KRouter(router).loanCount(pool);
        for (uint256 i; i < nl; ++i) {
            KLoanView memory l = KRouter(router).loan(pool, uint8(i));
            s = string.concat(
                s, i == 0 ? "" : ",", '{"token":"', vm.toString(l.token), '","decimals":', vm.toString(uint256(l.decimals)),
                ',"loanScale":', _u(l.loanScale), ',"debtCap":', _u(l.debtCap), ',"supplyCap":', _u(l.supplyCap),
                ',"borrowEnabled":', _b(l.borrowEnabled), ',"retired":', _b(l.retired), "}"
            );
        }
        (uint16[] memory bo, uint16[] memory so, uint16[] memory wo, uint16[] memory ro) = KRouter(router).priorities(pool);
        s = string.concat(
            s, '],"borrowOrder":', _u16a(bo), ',"supplyOrder":', _u16a(so), ',"withdrawOrder":', _u16a(wo),
            ',"repayOrder":', _u16a(ro), ',"venues":['
        );
        uint256 nv = KRouter(router).venueCount(pool);
        for (uint256 i; i < nv; ++i) {
            s = string.concat(s, i == 0 ? "" : ",", _venueState(router, pool, i));
        }
        s = string.concat(s, "]}");
    }

    function _poolState() internal view returns (string memory s) {
        KPool p = KPool(POOL);
        (uint256 physical, uint256 gross) = p.poolAssetPosition();
        KDials memory d = p.dials();
        KLimits memory lim = p.limits();
        (uint256 features,,) = p.switches();
        s = string.concat(
            '{"physical":', _u(physical), ',"gross":', _u(gross), ',"phiWad":', _u(d.phiWad), ',"ltvWad":', _u(d.ltvWad),
            ',"roomEpsilonWad":', _u(lim.roomEpsilonWad), ',"features":', _u(features), ',"totalSupply":',
            _u(p.totalSupply()), ',"loans":['
        );
        uint256 n = p.loanCount();
        (bool okU0, uint256 usd0,) = KFeed(FEED).peekUsd(USDC);
        for (uint256 i; i < n; ++i) {
            (address token, uint8 dec, uint64 band,, uint256 maxN, uint256 reserve, uint256 liquid) = p.loanConfig(uint8(i));
            (bool ok, uint256 px,) = KFeed(FEED).peekCross(CBBTC, token);
            (bool okU, uint256 usdI,) = KFeed(FEED).peekUsd(token);
            uint256 crossW = i == 0 ? WAD : ((okU0 && okU && usd0 != 0) ? usdI * WAD / usd0 : 0);
            s = string.concat(
                s, i == 0 ? "" : ",", '{"token":"', vm.toString(token), '","scale":', _u(10 ** (18 - dec)), ',"liquid":',
                _u(liquid), ',"reserveTarget":', _u(reserve), ',"maxSwapNotional":', _u(maxN), ',"swapPriceBandWad":',
                _u(band), ',"priceWad":', _u(ok ? px : 0), ',"crossWad":', _u(crossW), "}"
            );
        }
        (uint256 liq, uint256 sup, uint256 debt) = p.loanPosition();
        s = string.concat(s, '],"loanPosition":[', _u(liq), ",", _u(sup), ",", _u(debt), "]");
        try p.totalAssets() returns (uint256 ta) {
            s = string.concat(s, ',"totalAssets":{"v":', _u(ta), "}}");
        } catch (bytes memory e) {
            s = string.concat(s, ',"totalAssets":{"err":', _h(e), "}}");
        }
    }

    // ------------------------------------------------------------------ views
    function _accountViews(bytes32 id, address account) internal view returns (string memory s) {
        KAccount a = KAccount(account);
        (bool r, uint256 c, uint256 sh, uint256 sup, uint256 debt) = a.tryPosition(id);
        s = string.concat(
            '{"tryPosition":{"readable":', _b(r), ',"collateral":', _u(c), ',"supplyShares":', _u(sh), ',"supplied":',
            _u(sup), ',"debt":', _u(debt), "}"
        );
        try a.debtOf(id) returns (uint256 v) {
            s = string.concat(s, ',"debtOf":', _vOrErr(true, v, ""));
        } catch (bytes memory e) {
            s = string.concat(s, ',"debtOf":', _vOrErr(false, 0, e));
        }
        try a.suppliedOf(id) returns (uint256 v) {
            s = string.concat(s, ',"suppliedOf":', _vOrErr(true, v, ""));
        } catch (bytes memory e) {
            s = string.concat(s, ',"suppliedOf":', _vOrErr(false, 0, e));
        }
        s = string.concat(s, ',"collateralOf":', _u(a.collateralOf(id)), ',"freeLiquidity":', _u(a.freeLiquidity(id)));
        uint256[5] memory shares = [uint256(1), 1e6, 1e12 + 7, 1e18 + 3, 1e24 + 11];
        s = string.concat(s, ',"supplySharesToAssets":[');
        for (uint256 i; i < shares.length; ++i) {
            try a.supplySharesToAssets(id, shares[i]) returns (uint256 v) {
                s = string.concat(s, i == 0 ? "" : ",", '{"shares":', _u(shares[i]), ',"v":', _u(v), "}");
            } catch (bytes memory e) {
                s = string.concat(s, i == 0 ? "" : ",", '{"shares":', _u(shares[i]), ',"err":', _h(e), "}");
            }
        }
        s = string.concat(s, '],"borrowRateAfter":', _rateGrid(id, a), "}");
    }

    function _rateGrid(bytes32 id, KAccount a) internal view returns (string memory s) {
        MMarket memory m = _market(id);
        uint256 tsa = m.totalSupplyAssets;
        uint256 tba = m.totalBorrowAssets;
        uint256 free = tsa > tba ? tsa - tba : 0;
        uint256[9] memory db = [uint256(0), 1, 1e6, 1e9, 1e12, free / 2, free, tsa, uint256(1) << 130];
        uint256[6] memory ds = [uint256(0), 1, free / 3, free, tsa == 0 ? 0 : tsa - 1, tsa];
        s = "[";
        bool first = true;
        for (uint256 i; i < db.length; ++i) {
            for (uint256 j; j < ds.length; ++j) {
                (bool ok, uint256 rate) = a.borrowRateAfter(id, db[i], ds[j]);
                s = string.concat(
                    s, first ? "" : ",", '{"dBorrow":', _u(db[i]), ',"dSupplyDown":', _u(ds[j]), ',"ok":', _b(ok),
                    ',"rate":', _u(rate), "}"
                );
                first = false;
            }
        }
        s = string.concat(s, "]");
    }

    function _ceilingGrid(address router, address pool, uint8 idx, uint256 px) internal view returns (string memory s) {
        uint256[12] memory coll = [uint256(0), 1, 1000, 15000, 26221, 100000, 234423, 1e6, 1e8, 1e10, 1e12, 1e14];
        uint256[7] memory prices = [uint256(0), 1, px * 97 / 100, px * 99 / 100, px, px * 101 / 100, px * 103 / 100];
        s = "[";
        bool first = true;
        for (uint256 i; i < coll.length; ++i) {
            for (uint256 j; j < prices.length; ++j) {
                string memory row;
                try KRouter(router).fundingCeiling(pool, idx, coll[i], prices[j]) returns (uint256 v) {
                    row = string.concat('"v":', _u(v));
                } catch (bytes memory e) {
                    row = string.concat('"err":', _h(e));
                }
                s = string.concat(
                    s, first ? "" : ",", '{"collIn":', _u(coll[i]), ',"priceWad":', _u(prices[j]), ",", row, "}"
                );
                first = false;
            }
        }
        s = string.concat(s, "]");
    }

    /// @dev `prices[i]` is loan asset i's poolAsset cross (L18 per poolAsset base unit).
    function _routerViews(address router, address pool, uint256[] memory prices) internal view returns (string memory s) {
        KRouter R = KRouter(router);
        (uint256[] memory c, uint256[] memory sp, uint256[] memory d, uint256 tc) = R.positions(pool);
        (uint8 cnt, uint8 mask) = R.drawnAssets(pool);
        s = string.concat(
            '{"positions":{"coll":', _ua(c), ',"sup":', _ua(sp), ',"debt":', _ua(d), ',"totalColl":', _u(tc),
            '},"drawn":[', vm.toString(uint256(cnt)), ",", vm.toString(uint256(mask)), '],"minLltv":', _u(R.minLltv(pool))
        );
        try R.reclaimable(pool, prices) returns (uint256 v) {
            s = string.concat(s, ',"reclaimable":', _vOrErr(true, v, ""));
        } catch (bytes memory e) {
            s = string.concat(s, ',"reclaimable":', _vOrErr(false, 0, e));
        }
        try R.reclaimable(pool, new uint256[](prices.length)) returns (uint256 v) {
            s = string.concat(s, ',"reclaimableUnpriced":', _vOrErr(true, v, ""));
        } catch (bytes memory e) {
            s = string.concat(s, ',"reclaimableUnpriced":', _vOrErr(false, 0, e));
        }
        s = string.concat(s, ',"loans":[');
        for (uint256 i; i < prices.length; ++i) {
            (uint256 rc, uint256 rs, uint256 dd) = R.position(pool, uint8(i));
            (bool any, uint256 fd, uint256 fc) = R.quarantine(pool, uint8(i));
            s = string.concat(
                s, i == 0 ? "" : ",", '{"position":[', _u(rc), ",", _u(rs), ",", _u(dd), '],"quarantine":{"any":', _b(any),
                ',"frozenDebt":', _u(fd), ',"frozenColl":', _u(fc), '},"fundingCeiling":',
                _ceilingGrid(router, pool, uint8(i), prices[i]), "}"
            );
        }
        s = string.concat(s, '],"venues":[');
        uint256 nv = R.venueCount(pool);
        for (uint256 i; i < nv; ++i) {
            s = string.concat(s, i == 0 ? "" : ",", _venueViews(router, pool, i, prices));
        }
        s = string.concat(s, "]}");
    }

    function _venueViews(address router, address pool, uint256 i, uint256[] memory prices)
        internal
        view
        returns (string memory s)
    {
        KRouter R = KRouter(router);
        KVenueView memory v = R.venue(pool, uint16(i));
        (uint256 a, uint256 b, uint256 c, uint256 d, uint256 e) = R.venuePosition(pool, uint16(i));
        s = string.concat(
            '{"venuePosition":[', _u(a), ",", _u(b), ",", _u(c), ",", _u(d), ",", _u(e), '],"readable":',
            _b(R.venueReadable(pool, uint16(i)))
        );
        try R.venueHealth(pool, uint16(i), prices[v.loanIndex]) returns (uint256 h) {
            s = string.concat(s, ',"health":', _vOrErr(true, h, ""));
        } catch (bytes memory err) {
            s = string.concat(s, ',"health":', _vOrErr(false, 0, err));
        }
        s = string.concat(s, ',"account":', _accountViews(v.id, v.account), "}");
    }

    function _pricesLive() internal view returns (uint256[] memory px) {
        uint256 n = KPool(POOL).loanCount();
        px = new uint256[](n);
        for (uint256 i; i < n; ++i) {
            (address token,,,,,,) = KPool(POOL).loanConfig(uint8(i));
            (bool ok, uint256 p,) = KFeed(FEED).peekCross(CBBTC, token);
            if (ok) px[i] = p;
        }
    }

    function _lensJson() internal view returns (string memory s) {
        KLensFacts memory f = KLens(LENS).facts(POOL);
        s = string.concat(
            '{"facts":[', _u(f.physical), ",", _u(f.posted), ",", _u(f.gross), ",", _u(f.liquid), ",", _u(f.supplied), ",",
            _u(f.debt), ",", _u(f.totalSupply), '],"venues":['
        );
        KLensVenue[] memory vs = KLens(LENS).venues(POOL);
        for (uint256 i; i < vs.length; ++i) {
            KLensVenue memory v = vs[i];
            s = string.concat(
                s, i == 0 ? "" : ",", "[", _u(v.collateral), ",", _u(v.recognizedCollateral), ",", _u(v.supplied), ",",
                _u(v.recognizedSupplied), ",", _u(v.debt), ",", _u(v.freeLiquidity), ",", _u(v.healthWad), "]"
            );
            s = string.concat(s, "");
            require(v.healthOk || v.healthWad == 0, "lens health");
        }
        s = string.concat(s, "]}");
    }

    /// @dev The live pool: block env, router/venue/Morpho/IRM state, pool ledger and the deployed views.
    function _liveSnap(uint256[] memory prices) internal view returns (string memory) {
        return string.concat(
            '{"block":', vm.toString(block.number), ',"timestamp":', vm.toString(block.timestamp), ',"router":',
            _routerState(ROUTER, POOL), ',"pool":', _poolState(), ',"prices":', _ua(prices), ',"views":',
            _routerViews(ROUTER, POOL, prices), ',"lens":', _lensJson(), "}"
        );
    }

    function _liveStateOnly() internal view returns (string memory) {
        return string.concat(
            '{"block":', vm.toString(block.number), ',"timestamp":', vm.toString(block.timestamp), ',"router":',
            _routerState(ROUTER, POOL), ',"pool":', _poolState(), "}"
        );
    }
}
