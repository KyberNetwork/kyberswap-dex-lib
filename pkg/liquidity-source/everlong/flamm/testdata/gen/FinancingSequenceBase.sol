// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import {Test} from "forge-std/Test.sol";
import {IMMRouter} from "src/interfaces/core/mm/IMMRouter.sol";

/// @dev Financing sequence generator helpers: Base mainnet addresses, minimal interfaces, a mintable
///      token, a settable oracle, a call-bundling proxy etched over the pool address, and one-line JSON dumps of the
///      Router record (every venue with its Morpho market, the account's position, the IRM's rateAtTarget, oracle and
///      IRM readability and the two managed fields read from Router storage).
struct R3Params {
    address loanToken;
    address collateralToken;
    address oracle;
    address irm;
    uint256 lltv;
}

struct R3Market {
    uint128 totalSupplyAssets;
    uint128 totalSupplyShares;
    uint128 totalBorrowAssets;
    uint128 totalBorrowShares;
    uint128 lastUpdate;
    uint128 fee;
}

interface R3Morpho {
    function market(bytes32) external view returns (uint128, uint128, uint128, uint128, uint128, uint128);
    function position(bytes32, address) external view returns (uint256, uint128, uint128);
    function idToMarketParams(bytes32) external view returns (address, address, address, address, uint256);
    function createMarket(R3Params memory) external;
    function setFee(R3Params memory, uint256) external;
    function owner() external view returns (address);
    function supply(R3Params memory, uint256, uint256, address, bytes memory) external returns (uint256, uint256);
    function withdraw(R3Params memory, uint256, uint256, address, address) external returns (uint256, uint256);
    function borrow(R3Params memory, uint256, uint256, address, address) external returns (uint256, uint256);
    function repay(R3Params memory, uint256, uint256, address, bytes memory) external returns (uint256, uint256);
    function supplyCollateral(R3Params memory, uint256, address, bytes memory) external;
    function withdrawCollateral(R3Params memory, uint256, address, address) external;
    function liquidate(R3Params memory, address, uint256, uint256, bytes memory) external returns (uint256, uint256);
}

interface R3Irm {
    function borrowRateView(R3Params memory, R3Market memory) external view returns (uint256);
    function borrowRate(R3Params memory, R3Market memory) external returns (uint256);
    function rateAtTarget(bytes32) external view returns (int256);
}

interface R3Oracle {
    function price() external view returns (uint256);
}

interface R3Account {
    function oraclePrice(bytes32) external view returns (bool, uint256);
    function tryPosition(bytes32) external view returns (bool, uint256, uint256, uint256, uint256);
    function debtOf(bytes32) external view returns (uint256);
    function suppliedOf(bytes32) external view returns (uint256);
    function freeLiquidity(bytes32) external view returns (uint256);
    function borrowRateAfter(bytes32, uint256, uint256) external view returns (bool, uint256);
    function supplySharesToAssets(bytes32, uint256) external view returns (uint256);
}

interface R3ERC20 {
    function balanceOf(address) external view returns (uint256);
    function approve(address, uint256) external returns (bool);
    function transfer(address, uint256) external returns (bool);
}

contract R3Token {
    mapping(address => uint256) public balanceOf;
    mapping(address => mapping(address => uint256)) public allowance;
    uint8 public constant decimals = 18;

    function mint(address to, uint256 a) external {
        balanceOf[to] += a;
    }

    function approve(address s, uint256 a) external returns (bool) {
        allowance[msg.sender][s] = a;
        return true;
    }

    function transfer(address to, uint256 a) external returns (bool) {
        balanceOf[msg.sender] -= a;
        balanceOf[to] += a;
        return true;
    }

    function transferFrom(address f, address to, uint256 a) external returns (bool) {
        if (allowance[f][msg.sender] != type(uint256).max) allowance[f][msg.sender] -= a;
        balanceOf[f] -= a;
        balanceOf[to] += a;
        return true;
    }
}

/// @dev mode 0 answers p, 1 reverts, 2 answers zero.
contract R3SetOracle {
    uint256 public p;
    uint8 public mode;

    function set(uint256 p_, uint8 mode_) external {
        p = p_;
        mode = mode_;
    }

    function price() external view returns (uint256) {
        require(mode != 1, "oracle down");
        return mode == 2 ? 0 : p;
    }
}

/// @dev Etched over the pool address: runs a bundle of calls as the pool inside one transaction; any failing call
///      reverts the whole bundle with its revert data.
contract R3PoolProxy {
    function exec(address[] calldata t, bytes[] calldata d) external returns (bytes[] memory r) {
        r = new bytes[](t.length);
        for (uint256 i; i < t.length; ++i) {
            (bool ok, bytes memory ret) = t[i].call(d[i]);
            if (!ok) {
                assembly {
                    revert(add(ret, 32), mload(ret))
                }
            }
            r[i] = ret;
        }
    }
}

abstract contract FinancingSequenceBase is Test {
    string internal constant RPC = "https://mainnet.base.org";
    uint256 internal constant BLOCK = 51317000;
    address internal constant MORPHO = 0xBBBBBbbBBb9cC5e90e3b3Af64bdAF62C37EEFFCb;
    address internal constant IRM = 0x46415998764C29aB2a25CbeA6254146D50D22687;
    address internal constant USDC = 0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913;
    address internal constant CBBTC = 0xcbB7C0000aB88B473b1f5aFd9ef808440eed33Bf;
    address internal constant POOL = 0xc0fdCB1799cCc2CEBaA1fe247157b0dF33D57572;
    address internal constant ROUTER = 0x19A9b39E6710AAD109C829294b0841F0851c6bB4;
    address internal constant FEED = 0xbED275459578C87a63F2f50A0b077C720e838816;
    address internal constant LIVE_ORACLE = 0x663BECd10daE6C4A3Dcd89F1d76c1174199639B9;
    bytes32 internal constant LIVE_ID = 0x9103c3b4e834476c9a62ea009ba2c884ee42e94e6e314a26f04d312434191836;
    address internal constant WHALE = address(0x3A1E3A1E);

    string internal outPath;
    uint256 internal nrows;

    function open(string memory name) internal {
        vm.createDir("kyber-out/edges", true);
        outPath = string.concat("kyber-out/edges/", name);
        vm.writeFile(outPath, "");
        nrows = 0;
    }

    function line(string memory r) internal {
        vm.writeLine(outPath, r);
        ++nrows;
    }

    // ------------------------------------------------------------------ JSON
    function q(uint256 x) internal pure returns (string memory) {
        return string.concat('"', vm.toString(x), '"');
    }

    function qi(int256 x) internal pure returns (string memory) {
        return string.concat('"', vm.toString(x), '"');
    }

    function qb(bool x) internal pure returns (string memory) {
        return x ? "true" : "false";
    }

    function qh(bytes memory x) internal pure returns (string memory) {
        return string.concat('"', vm.toString(x), '"');
    }

    function qs(string memory x) internal pure returns (string memory) {
        return string.concat('"', x, '"');
    }

    function kv(string memory k, string memory v) internal pure returns (string memory) {
        return string.concat('"', k, '":', v);
    }

    function obj2(string memory a, string memory b) internal pure returns (string memory) {
        return string.concat("{", a, ",", b, "}");
    }

    function res(bool ok, bytes memory ret) internal pure returns (string memory) {
        return string.concat('{"ok":', qb(ok), ',"ret":', qh(ret), "}");
    }

    function arr(uint256[] memory a) internal pure returns (string memory s) {
        s = "[";
        for (uint256 i; i < a.length; ++i) s = string.concat(s, i == 0 ? "" : ",", q(a[i]));
        s = string.concat(s, "]");
    }

    function ords(uint16[] memory o) internal pure returns (string memory s) {
        s = "[";
        for (uint256 i; i < o.length; ++i) s = string.concat(s, i == 0 ? "" : ",", vm.toString(o[i]));
        s = string.concat(s, "]");
    }

    // ------------------------------------------------------------------ randomness
    uint256 internal seed;

    function rnd(uint256 salt) internal view returns (uint256) {
        return uint256(keccak256(abi.encode("m2r3", seed, salt)));
    }

    /// @dev Log-uniform-ish in [lo, hi]: a random bit length, then a uniform value below it.
    function rlog(uint256 salt, uint256 lo, uint256 hi) internal view returns (uint256) {
        if (hi <= lo) return lo;
        uint256 span = hi - lo;
        uint256 bits;
        for (uint256 x = span; x > 0; x >>= 1) ++bits;
        uint256 b = 1 + rnd(salt) % bits;
        uint256 cap = b >= 256 ? type(uint256).max : (uint256(1) << b);
        uint256 v = rnd(salt + 7) % cap;
        return v > span ? lo + span : lo + v;
    }

    // ------------------------------------------------------------------ Router record dump
    function venueBase(uint16 id) internal pure returns (uint256) {
        uint256 arrSlot = uint256(keccak256(abi.encode(POOL, uint256(1)))) + 3;
        return uint256(keccak256(abi.encode(arrSlot))) + 6 * uint256(id);
    }

    function params(bytes32 id) internal view returns (R3Params memory p) {
        (p.loanToken, p.collateralToken, p.oracle, p.irm, p.lltv) = R3Morpho(MORPHO).idToMarketParams(id);
    }

    function mkt(bytes32 id) internal view returns (R3Market memory m) {
        (m.totalSupplyAssets, m.totalSupplyShares, m.totalBorrowAssets, m.totalBorrowShares, m.lastUpdate, m.fee) =
            R3Morpho(MORPHO).market(id);
    }

    function marketJson(bytes32 id) internal view returns (string memory) {
        R3Market memory m = mkt(id);
        return string.concat(
            '{"tsa":', q(m.totalSupplyAssets), ',"tss":', q(m.totalSupplyShares), ',"tba":', q(m.totalBorrowAssets),
            ',"tbs":', q(m.totalBorrowShares), ',"lastUpdate":', q(m.lastUpdate), ',"fee":', q(m.fee), "}"
        );
    }

    function positionJson(bytes32 id, address user) internal view returns (string memory) {
        (uint256 a, uint128 b, uint128 c) = R3Morpho(MORPHO).position(id, user);
        return string.concat('{"supplyShares":', q(a), ',"borrowShares":', q(b), ',"collateral":', q(c), "}");
    }

    function morphoJson(bytes32 id, address account) internal view returns (string memory s) {
        R3Params memory p = params(id);
        bool irmOk = true;
        uint256 rat;
        if (p.irm != address(0)) {
            try R3Irm(p.irm).borrowRateView(p, mkt(id)) returns (uint256) {} catch { irmOk = false; }
            rat = uint256(R3Irm(p.irm).rateAtTarget(id));
        }
        (bool ook, uint256 op) = R3Account(account).oraclePrice(id);
        bool ozero;
        try R3Oracle(p.oracle).price() returns (uint256 x) { ozero = x == 0; } catch {}
        s = string.concat('{"market":', marketJson(id), ',"position":', positionJson(id, account));
        s = string.concat(s, ',"lltvWad":', q(p.lltv), ',"hasIrm":', qb(p.irm != address(0)), ',"irmReadable":', qb(irmOk));
        s = string.concat(s, ',"rateAtTarget":', q(rat), ',"oracleOk":', qb(ook), ',"oraclePrice":', q(op), ',"oracleZero":', qb(ozero), "}");
    }

    function venueJson(uint16 id) internal view returns (string memory s) {
        IMMRouter.VenueView memory v = IMMRouter(ROUTER).venue(POOL, id);
        uint256 b = venueBase(id);
        uint256 mColl = uint256(vm.load(ROUTER, bytes32(b + 4)));
        uint256 mShares = uint256(vm.load(ROUTER, bytes32(b + 5)));
        s = string.concat('{"morpho":', morphoJson(v.id, v.account), ',"kind":', vm.toString(v.kind), ',"loanIndex":', vm.toString(v.loanIndex));
        s = string.concat(s, ',"lltvWad":', q(v.lltvWad), ',"borrowEnabled":', qb(v.borrowEnabled), ',"supplyEnabled":', qb(v.supplyEnabled));
        s = string.concat(s, ',"retired":', qb(v.retired), ',"debtCap":', q(v.debtCap), ',"supplyCap":', q(v.supplyCap));
        s = string.concat(s, ',"maxBorrowRateWad":', q(v.maxBorrowRateWad), ',"managedCollateral":', q(mColl), ',"managedSupplyShares":', q(mShares), "}");
    }

    function loanJson(uint8 idx) internal view returns (string memory) {
        IMMRouter.LoanView memory l = IMMRouter(ROUTER).loan(POOL, idx);
        return string.concat(
            '{"decimals":', vm.toString(l.decimals), ',"loanScale":', q(l.loanScale), ',"debtCap":', q(l.debtCap),
            ',"supplyCap":', q(l.supplyCap), ',"borrowEnabled":', qb(l.borrowEnabled), ',"retired":', qb(l.retired), "}"
        );
    }

    function routerJson() internal view returns (string memory s) {
        (uint64 pin, uint64 gap, uint64 band) = IMMRouter(ROUTER).pin(POOL);
        s = string.concat('{"globalPaused":', qb(IMMRouter(ROUTER).globalPaused()), ',"pinLtvWad":', q(pin), ',"safetyGapWad":', q(gap));
        s = string.concat(s, ',"oracleBandWad":', q(band), ',"maxDrawnAssets":', vm.toString(IMMRouter(ROUTER).maxDrawnAssets(POOL)));
        uint256 nl = IMMRouter(ROUTER).loanCount(POOL);
        s = string.concat(s, ',"loans":[');
        for (uint256 i; i < nl; ++i) s = string.concat(s, i == 0 ? "" : ",", loanJson(uint8(i)));
        uint256 nv = IMMRouter(ROUTER).venueCount(POOL);
        s = string.concat(s, '],"venues":[');
        for (uint256 i; i < nv; ++i) s = string.concat(s, i == 0 ? "" : ",", venueJson(uint16(i)));
        (uint16[] memory bo, uint16[] memory so, uint16[] memory wo, uint16[] memory ro) = IMMRouter(ROUTER).priorities(POOL);
        s = string.concat(s, '],"borrowOrder":', ords(bo), ',"supplyOrder":', ords(so), ',"withdrawOrder":', ords(wo));
        s = string.concat(s, ',"repayOrder":', ords(ro), "}");
    }

    // ------------------------------------------------------------------ Router / account views on the current state
    function sv(address target, bytes memory data) internal view returns (string memory) {
        (bool ok, bytes memory ret) = target.staticcall(data);
        return res(ok, ret);
    }

    function routerViewsJson(uint256[] memory prices, uint256[] memory collIns) internal view returns (string memory s) {
        uint256 nl = IMMRouter(ROUTER).loanCount(POOL);
        uint256 nv = IMMRouter(ROUTER).venueCount(POOL);
        s = string.concat('{"positions":', sv(ROUTER, abi.encodeCall(IMMRouter.positions, (POOL))));
        s = string.concat(s, ',"drawn":', sv(ROUTER, abi.encodeCall(IMMRouter.drawnAssets, (POOL))));
        s = string.concat(s, ',"minLltv":', sv(ROUTER, abi.encodeCall(IMMRouter.minLltv, (POOL))));
        s = string.concat(s, ',"reclaimable":', sv(ROUTER, abi.encodeCall(IMMRouter.reclaimable, (POOL, prices))));
        s = string.concat(s, ',"quarantine":[');
        for (uint256 i; i < nl; ++i) {
            s = string.concat(s, i == 0 ? "" : ",", sv(ROUTER, abi.encodeCall(IMMRouter.quarantine, (POOL, uint8(i)))));
        }
        s = string.concat(s, '],"ceiling":[');
        for (uint256 i; i < nl; ++i) {
            for (uint256 j; j < collIns.length; ++j) {
                s = string.concat(s, (i == 0 && j == 0) ? "" : ",", sv(ROUTER, abi.encodeCall(IMMRouter.fundingCeiling, (POOL, uint8(i), collIns[j], prices[i]))));
            }
        }
        s = string.concat(s, '],"venuePosition":[');
        for (uint256 i; i < nv; ++i) {
            s = string.concat(s, i == 0 ? "" : ",", sv(ROUTER, abi.encodeCall(IMMRouter.venuePosition, (POOL, uint16(i)))));
        }
        s = string.concat(s, '],"health":[');
        for (uint256 i; i < nv; ++i) {
            uint8 li = IMMRouter(ROUTER).venue(POOL, uint16(i)).loanIndex;
            s = string.concat(s, i == 0 ? "" : ",", sv(ROUTER, abi.encodeCall(IMMRouter.venueHealth, (POOL, uint16(i), prices[li]))));
        }
        s = string.concat(s, "]}");
    }

    /// @dev Account views of venue `id` with rate quotes at the given (deltaBorrow, deltaSupplyDown) pairs.
    function accountViewsJson(uint16 id, uint256[] memory dB, uint256[] memory dS) internal view returns (string memory s) {
        IMMRouter.VenueView memory v = IMMRouter(ROUTER).venue(POOL, id);
        address a = v.account;
        s = string.concat('{"tryPosition":', sv(a, abi.encodeCall(R3Account.tryPosition, (v.id))));
        s = string.concat(s, ',"debtOf":', sv(a, abi.encodeCall(R3Account.debtOf, (v.id))));
        s = string.concat(s, ',"suppliedOf":', sv(a, abi.encodeCall(R3Account.suppliedOf, (v.id))));
        s = string.concat(s, ',"freeLiquidity":', sv(a, abi.encodeCall(R3Account.freeLiquidity, (v.id))));
        s = string.concat(s, ',"dB":', arr(dB), ',"dS":', arr(dS), ',"rates":[');
        for (uint256 i; i < dB.length; ++i) {
            s = string.concat(s, i == 0 ? "" : ",", sv(a, abi.encodeCall(R3Account.borrowRateAfter, (v.id, dB[i], dS[i]))));
        }
        s = string.concat(s, "]}");
    }
}
