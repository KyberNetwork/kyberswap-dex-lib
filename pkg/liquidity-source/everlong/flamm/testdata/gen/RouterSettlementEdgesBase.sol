// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import {Test} from "forge-std/Test.sol";

/// @dev Settlement edge generator helpers: a one-row-per-line JSON array writer, Base mainnet addresses, Morpho Blue /
///      AdaptiveCurveIrm storage writers and a mintable token plus a settable oracle.
struct R2Params {
    address loanToken;
    address collateralToken;
    address oracle;
    address irm;
    uint256 lltv;
}

struct R2MarketState {
    uint128 totalSupplyAssets;
    uint128 totalSupplyShares;
    uint128 totalBorrowAssets;
    uint128 totalBorrowShares;
    uint128 lastUpdate;
    uint128 fee;
}

interface R2Morpho {
    function market(bytes32) external view returns (uint128, uint128, uint128, uint128, uint128, uint128);
    function position(bytes32, address) external view returns (uint256, uint128, uint128);
    function createMarket(R2Params memory) external;
    function supply(R2Params memory, uint256, uint256, address, bytes memory) external returns (uint256, uint256);
    function withdraw(R2Params memory, uint256, uint256, address, address) external returns (uint256, uint256);
    function borrow(R2Params memory, uint256, uint256, address, address) external returns (uint256, uint256);
    function repay(R2Params memory, uint256, uint256, address, bytes memory) external returns (uint256, uint256);
    function supplyCollateral(R2Params memory, uint256, address, bytes memory) external;
    function withdrawCollateral(R2Params memory, uint256, address, address) external;
    function accrueInterest(R2Params memory) external;
}

interface R2Irm {
    function borrowRateView(R2Params memory, R2MarketState memory) external view returns (uint256);
    function borrowRate(R2Params memory, R2MarketState memory) external returns (uint256);
    function rateAtTarget(bytes32) external view returns (int256);
}

interface R2Account {
    function oraclePrice(bytes32) external view returns (bool, uint256);
    function borrowRateAfter(bytes32, uint256, uint256) external view returns (bool, uint256);
    function tryPosition(bytes32) external view returns (bool, uint256, uint256, uint256, uint256);
    function debtOf(bytes32) external view returns (uint256);
    function suppliedOf(bytes32) external view returns (uint256);
    function supplySharesToAssets(bytes32, uint256) external view returns (uint256);
    function freeLiquidity(bytes32) external view returns (uint256);
    function supplyCollateral(bytes32, uint256) external;
    function withdrawCollateral(bytes32, uint256, address) external;
    function borrow(bytes32, uint256, address) external;
    function repay(bytes32, uint256) external returns (uint256);
    function supply(bytes32, uint256) external returns (uint256);
    function withdraw(bytes32, uint256, uint256, address) external returns (uint256, uint256);
}

contract R2Token {
    mapping(address => uint256) public balanceOf;
    mapping(address => mapping(address => uint256)) public allowance;
    uint8 public decimals;

    constructor(uint8 d) {
        decimals = d;
    }

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

/// @dev `mode` 0 answers `p`, 1 reverts, 2 answers zero.
contract R2Oracle {
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

abstract contract RouterSettlementEdgesBase is Test {
    string internal constant RPC = "https://mainnet.base.org";
    uint256 internal constant BLOCK = 51317000;
    address internal constant MORPHO = 0xBBBBBbbBBb9cC5e90e3b3Af64bdAF62C37EEFFCb;
    address internal constant IRM = 0x46415998764C29aB2a25CbeA6254146D50D22687;
    address internal constant USDC = 0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913;
    address internal constant CBBTC = 0xcbB7C0000aB88B473b1f5aFd9ef808440eed33Bf;
    address internal constant POOL = 0xc0fdCB1799cCc2CEBaA1fe247157b0dF33D57572;
    address internal constant ROUTER = 0x19A9b39E6710AAD109C829294b0841F0851c6bB4;
    address internal constant ACCOUNT = 0x6760E3b032eE2d670Cb684d9076b8f48cb066c48;
    address internal constant FEED = 0xbED275459578C87a63F2f50A0b077C720e838816;
    address internal constant LIVE_ORACLE = 0x663BECd10daE6C4A3Dcd89F1d76c1174199639B9;
    bytes32 internal constant LIVE_ID = 0x9103c3b4e834476c9a62ea009ba2c884ee42e94e6e314a26f04d312434191836;

    int256 internal constant MIN_RAT = int256(uint256(0.001e18) / 365 days);
    int256 internal constant MAX_RAT = int256(uint256(2e18) / 365 days);
    int256 internal constant INIT_RAT = int256(uint256(0.04e18) / 365 days);

    string internal outPath;
    uint256 internal rows;

    function open(string memory name) internal {
        outPath = string.concat("kyber-out/edges/", name);
        vm.createDir("kyber-out/edges", true);
        vm.writeFile(outPath, '[{"header":true}\n');
        rows = 0;
    }

    function row(string memory r) internal {
        vm.writeLine(outPath, string.concat(",", r));
        ++rows;
    }

    function close() internal {
        vm.writeLine(outPath, "]");
    }

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

    function add(string memory o, string memory k, string memory v) internal pure returns (string memory) {
        return string.concat(o, bytes(o).length == 1 ? "" : ",", '"', k, '":', v);
    }

    function end(string memory o) internal pure returns (string memory) {
        return string.concat(o, "}");
    }

    function arr(uint256[] memory a) internal pure returns (string memory s) {
        s = "[";
        for (uint256 i; i < a.length; ++i) s = string.concat(s, i == 0 ? "" : ",", q(a[i]));
        s = string.concat(s, "]");
    }

    function arri(int256[] memory a) internal pure returns (string memory s) {
        s = "[";
        for (uint256 i; i < a.length; ++i) s = string.concat(s, i == 0 ? "" : ",", qi(a[i]));
        s = string.concat(s, "]");
    }

    function ords(uint16[] memory o) internal pure returns (string memory s) {
        s = "[";
        for (uint256 i; i < o.length; ++i) s = string.concat(s, i == 0 ? "" : ",", vm.toString(o[i]));
        s = string.concat(s, "]");
    }

    /// @dev `{"ok":bool,"ret":hex}` of a low-level call: returndata on success, revert data otherwise.
    function res(bool ok, bytes memory ret) internal pure returns (string memory) {
        return string.concat('{"ok":', qb(ok), ',"ret":', qh(ret), "}");
    }

    function rnd(uint256 seed, uint256 salt) internal pure returns (uint256) {
        return uint256(keccak256(abi.encode("r2", seed, salt)));
    }

    /// @dev Log-uniform-ish magnitude in [lo, hi): a random decade then a random mantissa.
    function rlog(uint256 seed, uint256 salt, uint256 lo, uint256 hi) internal pure returns (uint256) {
        if (hi <= lo + 1) return lo;
        uint256 r = rnd(seed, salt);
        uint256 span = hi - lo;
        uint256 bits;
        for (uint256 x = span; x > 1; x >>= 1) ++bits;
        uint256 b = (r % (bits + 1));
        uint256 cap = b >= 256 ? span : (uint256(1) << b);
        if (cap > span) cap = span;
        return lo + (rnd(seed, salt + 991) % cap);
    }

    // ------------------------------------------------------------------ Morpho / IRM storage
    function marketSlot(bytes32 id) internal pure returns (uint256) {
        return uint256(keccak256(abi.encode(id, uint256(3))));
    }

    function positionSlot(bytes32 id, address user) internal pure returns (uint256) {
        return uint256(keccak256(abi.encode(user, keccak256(abi.encode(id, uint256(2))))));
    }

    function writeMarket(bytes32 id, uint128 tsa, uint128 tss, uint128 tba, uint128 tbs, uint128 lastUpdate, uint128 fee)
        internal
    {
        uint256 s = marketSlot(id);
        vm.store(MORPHO, bytes32(s), bytes32((uint256(tss) << 128) | tsa));
        vm.store(MORPHO, bytes32(s + 1), bytes32((uint256(tbs) << 128) | tba));
        vm.store(MORPHO, bytes32(s + 2), bytes32((uint256(fee) << 128) | lastUpdate));
    }

    function writePosition(bytes32 id, address user, uint256 supplyShares, uint128 borrowShares, uint128 collateral)
        internal
    {
        uint256 s = positionSlot(id, user);
        vm.store(MORPHO, bytes32(s), bytes32(supplyShares));
        vm.store(MORPHO, bytes32(s + 1), bytes32((uint256(collateral) << 128) | borrowShares));
    }

    function writeRat(bytes32 id, int256 rat) internal {
        vm.store(IRM, keccak256(abi.encode(id, uint256(0))), bytes32(uint256(rat)));
    }

    function marketJson(bytes32 id) internal view returns (string memory o) {
        (uint128 a, uint128 b, uint128 c, uint128 d, uint128 e, uint128 f) = R2Morpho(MORPHO).market(id);
        o = "{";
        o = add(o, "tsa", q(a));
        o = add(o, "tss", q(b));
        o = add(o, "tba", q(c));
        o = add(o, "tbs", q(d));
        o = add(o, "lastUpdate", q(e));
        o = end(add(o, "fee", q(f)));
    }

    function positionJson(bytes32 id, address user) internal view returns (string memory o) {
        (uint256 a, uint128 b, uint128 c) = R2Morpho(MORPHO).position(id, user);
        o = "{";
        o = add(o, "supplyShares", q(a));
        o = add(o, "borrowShares", q(b));
        o = end(add(o, "collateral", q(c)));
    }
}
