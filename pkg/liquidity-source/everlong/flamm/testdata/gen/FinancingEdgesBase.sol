// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import {Test} from "forge-std/Test.sol";

/// @dev Financing edge generator helpers: JSON-array writer one row per line, Base constants,
///      Morpho / IRM storage pokes and a mintable token + settable oracle.
struct VMParams {
    address loanToken;
    address collateralToken;
    address oracle;
    address irm;
    uint256 lltv;
}

interface VMorpho {
    function market(bytes32) external view returns (uint128, uint128, uint128, uint128, uint128, uint128);
    function position(bytes32, address) external view returns (uint256, uint128, uint128);
    function createMarket(VMParams memory) external;
    function supply(VMParams memory, uint256, uint256, address, bytes memory) external returns (uint256, uint256);
    function withdraw(VMParams memory, uint256, uint256, address, address) external returns (uint256, uint256);
    function borrow(VMParams memory, uint256, uint256, address, address) external returns (uint256, uint256);
    function repay(VMParams memory, uint256, uint256, address, bytes memory) external returns (uint256, uint256);
    function supplyCollateral(VMParams memory, uint256, address, bytes memory) external;
    function withdrawCollateral(VMParams memory, uint256, address, address) external;
    function accrueInterest(VMParams memory) external;
}

struct VMarketState {
    uint128 totalSupplyAssets;
    uint128 totalSupplyShares;
    uint128 totalBorrowAssets;
    uint128 totalBorrowShares;
    uint128 lastUpdate;
    uint128 fee;
}

interface VIrmFull {
    function borrowRateView(VMParams memory, VMarketState memory) external view returns (uint256);
    function borrowRate(VMParams memory, VMarketState memory) external returns (uint256);
}

interface VIrm {
    function rateAtTarget(bytes32) external view returns (int256);
}

contract VToken {
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

contract VOracle {
    uint256 public p;
    bool public dead;

    function set(uint256 p_, bool dead_) external {
        p = p_;
        dead = dead_;
    }

    function price() external view returns (uint256) {
        require(!dead, "dead");
        return p;
    }
}

abstract contract FinancingEdgesBase is Test {
    string internal constant RPC = "https://mainnet.base.org";
    uint256 internal constant BLOCK = 51317000;
    address internal constant MORPHO = 0xBBBBBbbBBb9cC5e90e3b3Af64bdAF62C37EEFFCb;
    address internal constant IRM = 0x46415998764C29aB2a25CbeA6254146D50D22687;
    address internal constant USDC = 0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913;
    address internal constant CBBTC = 0xcbB7C0000aB88B473b1f5aFd9ef808440eed33Bf;
    address internal constant WETH = 0x4200000000000000000000000000000000000006;
    address internal constant POOL = 0xc0fdCB1799cCc2CEBaA1fe247157b0dF33D57572;
    address internal constant ROUTER = 0x19A9b39E6710AAD109C829294b0841F0851c6bB4;
    address internal constant ACCOUNT = 0x6760E3b032eE2d670Cb684d9076b8f48cb066c48;
    address internal constant LIVE_ORACLE = 0x663BECd10daE6C4A3Dcd89F1d76c1174199639B9;
    bytes32 internal constant LIVE_ID = 0x9103c3b4e834476c9a62ea009ba2c884ee42e94e6e314a26f04d312434191836;

    int256 internal constant MIN_RAT = int256(uint256(0.001e18) / 365 days);
    int256 internal constant MAX_RAT = int256(uint256(2e18) / 365 days);
    int256 internal constant INIT_RAT = int256(uint256(0.04e18) / 365 days);

    string internal outPath;

    // ------------------------------------------------------------------ JSON
    function open(string memory name) internal {
        outPath = string.concat("kyber-out/edges/", name);
        vm.createDir("kyber-out/edges", true);
        // storage snapshots revert contract state, so the array opens with a header element and every row is
        // written with a leading comma
        vm.writeFile(outPath, '[{"header":true}\n');
    }

    function row(string memory r) internal {
        vm.writeLine(outPath, string.concat(",", r));
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

    function kv(string memory k, string memory v) internal pure returns (string memory) {
        return string.concat('"', k, '":', v);
    }

    /// @dev Appends `"k":v` to an object under construction that was opened with "{".
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

    // ------------------------------------------------------------------ randomness
    function rnd(uint256 seed, uint256 salt) internal pure returns (uint256) {
        return uint256(keccak256(abi.encode(seed, salt)));
    }

    /// @dev A value of random bit-length in [0, maxBits].
    function rbits(uint256 seed, uint256 salt, uint256 maxBits) internal pure returns (uint256) {
        uint256 r = rnd(seed, salt);
        uint256 bits = r % (maxBits + 1);
        if (bits == 0) return 0;
        uint256 v = rnd(seed, salt + 7777);
        return bits == 256 ? v : v % (uint256(1) << bits);
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

    function marketJson(bytes32 id) internal view returns (string memory) {
        (uint128 a, uint128 b, uint128 c, uint128 d, uint128 e, uint128 f) = VMorpho(MORPHO).market(id);
        return string.concat(
            "{", kv("tsa", q(a)), ",", kv("tss", q(b)), ",", kv("tba", q(c)), ",", kv("tbs", q(d)), ",",
            kv("lastUpdate", q(e)), ",", kv("fee", q(f)), "}"
        );
    }

    function positionJson(bytes32 id, address user) internal view returns (string memory) {
        (uint256 a, uint128 b, uint128 c) = VMorpho(MORPHO).position(id, user);
        return string.concat("{", kv("supplyShares", q(a)), ",", kv("borrowShares", q(b)), ",", kv("collateral", q(c)), "}");
    }
}
