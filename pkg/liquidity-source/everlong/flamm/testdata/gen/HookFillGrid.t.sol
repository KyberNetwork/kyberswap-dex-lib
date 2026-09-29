// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import {Test} from "forge-std/Test.sol";
import {FillResult, PoolContext, SwapContext} from "src/interfaces/core/flamm/IFLAMMHooks.sol";

struct KyberHookBook {
    uint256 kappa;
    uint256 rs;
    uint256 is_;
    uint256 rv;
    uint256 iv;
    uint256 x;
}

interface IKyberHook {
    function spot(PoolContext calldata ctx) external view returns (uint256);
    function bookFor(PoolContext calldata ctx) external view returns (KyberHookBook memory);
    function previewFeeWad(SwapContext calldata ctx, uint256 spotBeforeWad) external view returns (uint256);
    function previewExactIn(SwapContext calldata ctx, uint256 feeWad) external view returns (FillResult memory);
    function executeExactIn(SwapContext calldata ctx, uint256 feeWad) external returns (FillResult memory);
}

/// @notice Kyber fixture generator: the LIVE EverlongHook swap path on a Base fork. The hook's storage is
///         overwritten per state (curve row, anchor, book, variance, fee row) and `bookFor`, `previewFeeWad`,
///         `previewExactIn` and a pool-pranked `executeExactIn` (post-state read back from storage, then the
///         book slots restored) are recorded per context. A second test records the real 15000-sat sell
///         (tx 0x46c3cd72..., block 51302916) against the state at block 51302915.
contract KyberHookFillGrid is Test {
    address constant HOOK = 0x65CBD227cBC61248ae77a5fC813A29C54C092134;
    address constant POOL = 0xc0fdCB1799cCc2CEBaA1fe247157b0dF33D57572;
    address constant ALM = 0xf82DdF0A8a50bc2C3F163997766bA1839E527A17;
    address constant USDC = 0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913;
    uint256 constant BLOCK = 51310000;
    uint256 constant WAD = 1e18;
    uint256 constant HALF = 5e17;
    uint256 constant Q192 = 1 << 192;
    bytes4 constant SEL_SUPPORT_FOR = 0x573f89ed;
    bytes4 constant SEL_RESERVES_AT = 0xfa07504e;
    string constant OUT = "test/kyber/fixtures/hook_fill_grid.json";
    string constant OUT_LIVE = "test/kyber/fixtures/hook_live_swap.json";

    struct Sup {
        uint256 aWad;
        uint256 xLo;
        uint256 xHi;
        uint256 yHi;
    }

    struct Cfg {
        uint128 a;
        uint128 up;
        uint128 dn;
    }

    /// @dev The storage the swap path reads, slot for slot.
    struct St {
        uint256 slot0; // spanUpWad << 128 | aWad
        uint256 slot1; // spanDnWad
        uint256 slot4; // fee row word 0: sigmaRef | gamma | out | mid
        uint256 slot5; // fee row word 1: dirSkew | volMax | volMin | volBeta
        uint256 slot6; // rvHalfLife | emaHalfLife | invSkewBand | invSkewKappa
        Sup sup; // slots 10..13
        uint256 anchorSqrtX96; // 14
        uint256 rp; // 15
        uint256 kappa; // 16
        uint256 x; // 17
        uint256 rs; // 18
        uint256 is_; // 19
        uint256 rv; // 20
        uint256 iv; // 21
        uint256 rvWad; // 23
    }

    bool internal _first = true;
    uint256 internal _states;

    function _u(uint256 v) internal pure returns (string memory) {
        return string.concat('"', vm.toString(v), '"');
    }

    function _kv(string memory k, uint256 v) internal pure returns (string memory) {
        return string.concat('"', k, '":', _u(v), ",");
    }

    function _res(bool ok, bytes memory ret) internal pure returns (string memory) {
        return string.concat('"ok":', ok ? "true" : "false", ',"err":"', ok ? "0x" : vm.toString(ret), '"');
    }

    function _write(string memory path, bool firstRow, string memory r) internal {
        vm.writeLine(path, string.concat(firstRow ? "[" : ",", r));
    }

    function _row(string memory r) internal {
        _write(OUT, _first, r);
        _first = false;
    }

    function _load(uint256 slot) internal view returns (uint256) {
        return uint256(vm.load(HOOK, bytes32(slot)));
    }

    function _put(uint256 slot, uint256 v) internal {
        vm.store(HOOK, bytes32(slot), bytes32(v));
    }

    function _readLive() internal view returns (St memory s) {
        s.slot0 = _load(0);
        s.slot1 = _load(1);
        s.slot4 = _load(4);
        s.slot5 = _load(5);
        s.slot6 = _load(6);
        s.sup = Sup(_load(10), _load(11), _load(12), _load(13));
        s.anchorSqrtX96 = _load(14);
        s.rp = _load(15);
        s.kappa = _load(16);
        s.x = _load(17);
        s.rs = _load(18);
        s.is_ = _load(19);
        s.rv = _load(20);
        s.iv = _load(21);
        s.rvWad = _load(23);
    }

    function _apply(St memory s) internal {
        _put(0, s.slot0);
        _put(1, s.slot1);
        _put(4, s.slot4);
        _put(5, s.slot5);
        _put(6, s.slot6);
        _put(10, s.sup.aWad);
        _put(11, s.sup.xLo);
        _put(12, s.sup.xHi);
        _put(13, s.sup.yHi);
        _put(14, s.anchorSqrtX96);
        _put(15, s.rp);
        _put(16, s.kappa);
        _put(17, s.x);
        _put(18, s.rs);
        _put(19, s.is_);
        _put(20, s.rv);
        _put(21, s.iv);
        _put(23, s.rvWad);
    }

    function _w64(uint256 word, uint256 i) internal pure returns (uint256) {
        return (word >> (64 * i)) & type(uint64).max;
    }

    function _pack(uint256 a, uint256 b, uint256 c, uint256 d) internal pure returns (uint256) {
        return a | (b << 64) | (c << 128) | (d << 192);
    }

    function _stateJson(uint256 id, string memory name, St memory s) internal pure returns (string memory) {
        string memory feeRow = string.concat(
            string.concat(_u(_w64(s.slot4, 0)), ",", _u(_w64(s.slot4, 1)), ",", _u(_w64(s.slot4, 2)), ",", _u(_w64(s.slot4, 3)), ","),
            string.concat(_u(_w64(s.slot5, 0)), ",", _u(_w64(s.slot5, 1)), ",", _u(_w64(s.slot5, 2)), ",", _u(_w64(s.slot5, 3)))
        );
        return string.concat(
            string.concat('{"k":"state","id":"', vm.toString(id), '","name":"', name, '",'),
            string.concat(_kv("aWad", s.slot0 & type(uint128).max), '"sup":[', _u(s.sup.aWad), ",", _u(s.sup.xLo), ",", _u(s.sup.xHi), ",", _u(s.sup.yHi), "],"),
            string.concat(_kv("anchorSqrtX96", s.anchorSqrtX96), _kv("rp", s.rp), _kv("kappa", s.kappa), _kv("x", s.x)),
            string.concat(_kv("rs", s.rs), _kv("is", s.is_), _kv("rv", s.rv), _kv("iv", s.iv), _kv("rvWad", s.rvWad)),
            string.concat('"fee":[', feeRow, "],", _kv("invKappa", _w64(s.slot6, 0)), _kv("invBand", _w64(s.slot6, 1))),
            '"loanScale":"1000000000000"}'
        );
    }

    function _supportFor(uint256 a, uint256 up, uint256 dn) internal view returns (Sup memory s) {
        (bool ok, bytes memory ret) = ALM.staticcall(abi.encodeWithSelector(SEL_SUPPORT_FOR, Cfg(uint128(a), uint128(up), uint128(dn))));
        require(ok, "supportFor");
        s = abi.decode(ret, (Sup));
    }

    function _reservesAt(Sup memory s, uint256 anchor, uint256 kappa, uint256 x) internal view returns (uint256 st, uint256 vol) {
        (bool ok, bytes memory ret) = ALM.staticcall(abi.encodeWithSelector(SEL_RESERVES_AT, s, uint160(anchor), kappa, x));
        require(ok, "reservesAt");
        (st, vol) = abi.decode(ret, (uint256, uint256));
    }

    /// @dev A book on the curve at `x` for scale `kappa`.
    function _onCurve(St memory base, uint256 x, uint256 kappa) internal view returns (St memory s) {
        s = _copy(base);
        s.x = x;
        s.kappa = kappa;
        (s.rs, s.rv) = _reservesAt(s.sup, s.anchorSqrtX96, kappa, x);
        s.is_ = 0;
        s.iv = 0;
    }

    function _copy(St memory b) internal pure returns (St memory s) {
        s = St(b.slot0, b.slot1, b.slot4, b.slot5, b.slot6, Sup(b.sup.aWad, b.sup.xLo, b.sup.xHi, b.sup.yHi), b.anchorSqrtX96, b.rp, b.kappa, b.x, b.rs, b.is_, b.rv, b.iv, b.rvWad);
    }

    function _sqrt(uint256 a) internal pure returns (uint256 r) {
        if (a == 0) return 0;
        r = a;
        uint256 y = (a + 1) / 2;
        while (y < r) {
            r = y;
            y = (a / y + y) / 2;
        }
    }

    // ------------------------------------------------------------------ states
    function _buildStates() internal view returns (St[] memory ss, string[] memory names) {
        St memory L = _readLive();
        ss = new St[](28);
        names = new string[](28);
        uint256 n;
        ss[n] = L;
        names[n++] = "live";
        // the constructor seed, exactly the pre-swap book
        ss[n] = _onCurve(L, HALF, 1e30);
        names[n++] = "seed";
        uint256 k = L.kappa;
        uint256[9] memory xs = [L.sup.xLo, L.sup.xLo + 1e12, 3e17, HALF - 1e9, HALF + 3e15, 7e17, L.sup.xHi - 1e12, L.sup.xHi, 1e18];
        for (uint256 i; i < xs.length; ++i) {
            ss[n] = _onCurve(L, xs[i], k);
            if (i % 2 == 1) {
                ss[n].is_ = 3e12 + i;
                ss[n].iv = 17 * i;
            }
            ss[n].rvWad = i % 3 == 0 ? 0 : (i % 3 == 1 ? 1.6e11 : 5e13);
            names[n++] = string.concat("curve-x-", vm.toString(i));
        }
        // acc == 0: seeded at x, and at xLo where the seed holds no volatile (reset to HALF)
        ss[n] = _copy(L);
        ss[n].rv = 0;
        ss[n].iv = 0;
        names[n++] = "acc0-x";
        ss[n] = _copy(L);
        ss[n].rv = 0;
        ss[n].iv = 0;
        ss[n].x = L.sup.xLo;
        names[n++] = "acc0-xLo";
        ss[n] = _copy(L);
        ss[n].kappa = 0;
        names[n++] = "kappa0";
        // fee rows
        St memory t = _onCurve(L, 7e17, k);
        t.rvWad = 1.6e11;
        ss[n] = _copy(t);
        ss[n].slot4 = _pack(0.030e18, 0.005e18, 0, 0);
        ss[n].slot5 = _pack(4e18, 0, 0, 0);
        ss[n].slot6 = _pack(0, 0, 4500, 1800);
        names[n++] = "row-bare";
        ss[n] = _copy(t);
        ss[n].slot4 = _pack(0.030e18, 0.005e18, 0.05e18, 0);
        ss[n].slot5 = _pack(4e18, 0, 0, 0.15e18);
        names[n++] = "row-novol";
        ss[n] = _copy(t);
        ss[n].slot6 = _pack(16e18, 0.49e18 - 1, 4500, 1800);
        names[n++] = "row-kappa16";
        ss[n] = _copy(t);
        ss[n].slot6 = _pack(6e18, 0.02e18, 4500, 1800);
        ss[n].x = 3e17;
        (ss[n].rs, ss[n].rv) = _reservesAt(t.sup, t.anchorSqrtX96, k, 3e17);
        names[n++] = "row-kappa6";
        ss[n] = _copy(t);
        ss[n].slot4 = _pack(0.9e18, 0.5e18, 1e18, 0.0004e18);
        ss[n].slot5 = _pack(1e18, 0.5e18, 8e18, 0.5e18);
        ss[n].slot6 = _pack(16e18, 0, 4500, 1800);
        names[n++] = "row-clamp";
        // anchor moved +20%, book re-read on the new anchor
        {
            St memory m = _copy(L);
            m.rp = L.rp * 12 / 10;
            m.anchorSqrtX96 = _sqrt(m.rp * (Q192 / WAD) + (m.rp * (Q192 % WAD)) / WAD);
            ss[n] = _onCurve(m, 3e17, k * 3);
            ss[n].rvWad = 2e12;
            names[n++] = "anchor-up";
        }
        // a second curve row: A = 2, spans 4/6
        {
            St memory c = _copy(L);
            c.sup = _supportFor(2e18, 4e18, 6e18);
            c.slot0 = (uint256(4e18) << 128) | 2e18;
            ss[n] = _onCurve(c, 6e17, k);
            names[n++] = "curve-a2";
        }
        // an off-curve book (the hook never checks the legs against the coordinate)
        ss[n] = _copy(L);
        ss[n].rs = L.rs * 3;
        ss[n].rv = L.rv / 2;
        ss[n].iv = 999;
        names[n++] = "offcurve";
        // a large book on the live row
        ss[n] = _onCurve(L, 532533306204662064, 1e30);
        ss[n].is_ = 5e20;
        names[n++] = "large";
        // near-empty book
        ss[n] = _onCurve(L, 4e17, 1e9);
        names[n++] = "dust";
        // a coordinate outside the curve domain: spot and the solve revert
        ss[n] = _copy(L);
        ss[n].x = 0;
        names[n++] = "x0";
        // zero curvature on an exactly value-balanced book: the reduction divides by zero
        ss[n] = _copy(t);
        ss[n].slot4 = _pack(0.030e18, 0.005e18, 0, 0.0004e18);
        ss[n].rv = 1e6;
        ss[n].iv = 0;
        ss[n].rs = 1e6 * L.rp / WAD;
        ss[n].is_ = 0;
        names[n++] = "gamma0-balanced";
        assembly {
            mstore(ss, n)
            mstore(names, n)
        }
    }

    // ------------------------------------------------------------------ grid
    function test_hookGrid() public {
        vm.createSelectFork("https://mainnet.base.org", BLOCK);
        if (vm.exists(OUT)) vm.removeFile(OUT);
        (St[] memory ss, string[] memory names) = _buildStates();
        for (uint256 i; i < ss.length; ++i) {
            _row(_stateJson(i, names[i], ss[i]));
        }
        for (uint256 i; i < ss.length; ++i) {
            _apply(ss[i]);
            (bool okS, bytes memory retS) = HOOK.staticcall(abi.encodeCall(IKyberHook.spot, (_ctx(1, true, 0, 0).pool)));
            uint256 sp = okS ? abi.decode(retS, (uint256)) : 0;
            _row(string.concat('{"k":"spot","s":"', vm.toString(i), '",', _kv("v", sp), _res(okS, retS), "}"));
            uint256 acc = ss[i].rv + ss[i].iv;
            if (acc == 0) {
                uint256 x = ss[i].x;
                (, acc) = _reservesAt(ss[i].sup, ss[i].anchorSqrtX96, 1e30, x);
                if (acc == 0) (, acc) = _reservesAt(ss[i].sup, ss[i].anchorSqrtX96, 1e30, HALF);
            }
            uint256[6] memory grosses = [acc, acc * 2, acc / 3, acc + 1, 234423, 1];
            uint256 ng = (i <= 1 || i == 22) ? 6 : 3;
            for (uint256 g; g < ng; ++g) {
                this.fillCell(i, grosses[g], true);
                this.fillCell(i, grosses[g], false);
            }
        }
        // gross overflows in the book read
        _apply(ss[0]);
        SwapContext memory c = _ctx(0, true, 15000, type(uint256).max);
        c.pool.physicalPoolAsset = type(uint256).max;
        c.pool.postedPoolAsset = 1;
        _fillRow(0, c, 17499999999999999);
        c.poolAssetIn = false;
        c.amountIn = 1e18;
        _fillRow(0, c, 17499999999999999);
        vm.writeLine(OUT, "]");
    }

    function _ctx(uint256 gross, bool poolAssetIn, uint256 amt, uint256 cap) internal pure returns (SwapContext memory c) {
        c.pool.physicalPoolAsset = gross - gross / 4;
        c.pool.postedPoolAsset = gross / 4;
        c.pool.priceWad = 768302232848967;
        c.pool.priceTs = 1789389719;
        c.pool.loanCount = 1;
        c.poolAssetIn = poolAssetIn;
        c.amountIn = amt;
        c.maxAmountOut = cap;
        c.loanAsset = USDC;
        c.crossWad = WAD;
    }

    function fillCell(uint256 sid, uint256 gross, bool poolAssetIn) external {
        SwapContext memory c0 = _ctx(gross, poolAssetIn, 0, type(uint256).max);
        (bool okB, bytes memory retB) = HOOK.staticcall(abi.encodeCall(IKyberHook.bookFor, (c0.pool)));
        KyberHookBook memory b;
        if (okB) b = abi.decode(retB, (KyberHookBook));
        uint256[10] memory amts = poolAssetIn
            ? [uint256(0), 1, 100, 15000, b.rv / 10, b.rv / 2, b.rv, b.rv * 10 + 7, 1e8, uint256(1) << 200]
            : [uint256(0), 1e12 - 1, 1e12, 1e18, 11301759e12, b.rs / 10 / 1e12 * 1e12, b.rs / 2 + 12345, b.rs, b.rs * 10, uint256(1) << 255];
        for (uint256 j; j < amts.length; ++j) {
            _amountCell(sid, gross, poolAssetIn, amts[j]);
        }
    }

    function _amountCell(uint256 sid, uint256 gross, bool poolAssetIn, uint256 amt) internal {
        SwapContext memory c = _ctx(gross, poolAssetIn, amt, type(uint256).max);
        (bool okF, bytes memory retF) = HOOK.staticcall(abi.encodeCall(IKyberHook.previewFeeWad, (c, 768302232848967)));
        uint256 hf = okF ? abi.decode(retF, (uint256)) : 17499999999999999;
        (bool ok, bytes memory ret) = HOOK.staticcall(abi.encodeCall(IKyberHook.previewExactIn, (c, hf)));
        uint256 net;
        if (ok) {
            FillResult memory fr = abi.decode(ret, (FillResult));
            net = fr.grossOut - fr.feeOut;
        }
        _fillRow(sid, c, hf);
        if (net != 0) {
            c.maxAmountOut = net / 2;
            _fillRow(sid, c, hf);
            c.maxAmountOut = net;
            _fillRow(sid, c, hf);
            c.maxAmountOut = 0;
            _fillRow(sid, c, hf);
            c.maxAmountOut = type(uint256).max;
        }
        _fillRow(sid, c, 0);
        _fillRow(sid, c, WAD - 1);
        _fillRow(sid, c, WAD);
    }

    function _fillRow(uint256 sid, SwapContext memory c, uint256 feeWad) internal {
        string memory head = string.concat(
            string.concat('{"k":"fill","s":"', vm.toString(sid), '",', _kv("phys", c.pool.physicalPoolAsset), _kv("posted", c.pool.postedPoolAsset)),
            string.concat('"in":', c.poolAssetIn ? "true," : "false,", _kv("amt", c.amountIn), _kv("cap", c.maxAmountOut), _kv("fee", feeWad))
        );
        (bool okB, bytes memory retB) = HOOK.staticcall(abi.encodeCall(IKyberHook.bookFor, (c.pool)));
        (bool okF, bytes memory retF) = HOOK.staticcall(abi.encodeCall(IKyberHook.previewFeeWad, (c, 768302232848967)));
        (bool okP, bytes memory retP) = HOOK.staticcall(abi.encodeCall(IKyberHook.previewExactIn, (c, feeWad)));
        uint256[6] memory pre = [_load(16), _load(17), _load(18), _load(19), _load(20), _load(21)];
        vm.prank(POOL);
        (bool okE, bytes memory retE) = HOOK.call(abi.encodeCall(IKyberHook.executeExactIn, (c, feeWad)));
        uint256[6] memory post = [_load(16), _load(18), _load(19), _load(20), _load(21), _load(17)];
        for (uint256 i; i < 6; ++i) {
            _put(16 + i, pre[i]);
        }
        _row(string.concat(head, _bookJson(okB, retB), _feeJson(okF, retF), _frJson("px", okP, retP), _frJson("ex", okE, retE), _postJson(post), "}"));
    }

    function _bookJson(bool ok, bytes memory ret) internal pure returns (string memory) {
        KyberHookBook memory b;
        if (ok) b = abi.decode(ret, (KyberHookBook));
        return string.concat(
            '"book":{',
            string.concat(_kv("kappa", b.kappa), _kv("rs", b.rs), _kv("is", b.is_), _kv("rv", b.rv), _kv("iv", b.iv), _kv("x", b.x)),
            _res(ok, ret),
            "},"
        );
    }

    function _feeJson(bool ok, bytes memory ret) internal pure returns (string memory) {
        uint256 f = ok ? abi.decode(ret, (uint256)) : 0;
        return string.concat('"pf":{', _kv("v", f), _res(ok, ret), "},");
    }

    function _frJson(string memory key, bool ok, bytes memory ret) internal pure returns (string memory) {
        FillResult memory fr;
        if (ok) fr = abi.decode(ret, (FillResult));
        return string.concat(
            '"', key, '":{',
            string.concat(_kv("used", fr.amountInUsed), _kv("gross", fr.grossOut), _kv("feeOut", fr.feeOut), _kv("spot", fr.spotAfterWad)),
            _res(ok, ret),
            "},"
        );
    }

    /// @dev The committed book after execute (the pool-pranked call writes only these six slots).
    function _postJson(uint256[6] memory post) internal pure returns (string memory) {
        return string.concat(
            '"post":{',
            string.concat(_kv("kappa", post[0]), _kv("rs", post[1]), _kv("is", post[2]), _kv("rv", post[3]), _kv("iv", post[4])),
            '"x":',
            _u(post[5]),
            "}"
        );
    }

    // ------------------------------------------------------------------ the one real swap
    function test_liveSwap() public {
        vm.createSelectFork("https://mainnet.base.org", 51302915);
        if (vm.exists(OUT_LIVE)) vm.removeFile(OUT_LIVE);
        St memory L = _readLive();
        _write(OUT_LIVE, true, _stateJson(0, "block-51302915", L));
        SwapContext memory c;
        c.pool.physicalPoolAsset = 219423;
        c.pool.shareSupply = 2194230000000000;
        c.pool.priceWad = 783691038987516;
        c.pool.priceTs = 1789389719;
        c.pool.loanCount = 1;
        c.poolAssetIn = true;
        c.amountIn = 15000;
        c.maxAmountOut = 101043362000000000000;
        c.loanAsset = USDC;
        c.crossWad = WAD;
        uint256 feeWad = 17499999999999999;
        string memory head = string.concat(
            '{"k":"fill","s":"0",', _kv("phys", 219423), _kv("posted", 0), '"in":true,', _kv("amt", 15000), _kv("cap", c.maxAmountOut), _kv("fee", feeWad)
        );
        (bool okB, bytes memory retB) = HOOK.staticcall(abi.encodeCall(IKyberHook.bookFor, (c.pool)));
        (bool okF, bytes memory retF) = HOOK.staticcall(abi.encodeCall(IKyberHook.previewFeeWad, (c, 768302232848967)));
        (bool okP, bytes memory retP) = HOOK.staticcall(abi.encodeCall(IKyberHook.previewExactIn, (c, feeWad)));
        vm.prank(POOL);
        (bool okE, bytes memory retE) = HOOK.call(abi.encodeCall(IKyberHook.executeExactIn, (c, feeWad)));
        uint256[6] memory post = [_load(16), _load(18), _load(19), _load(20), _load(21), _load(17)];
        _write(OUT_LIVE, false, string.concat(head, _bookJson(okB, retB), _feeJson(okF, retF), _frJson("px", okP, retP), _frJson("ex", okE, retE), _postJson(post), "}"));
        vm.writeLine(OUT_LIVE, "]");
        FillResult memory fr = abi.decode(retP, (FillResult));
        assertEq(abi.decode(retF, (uint256)), feeWad, "hook fee is the settled fee");
        assertEq(fr.grossOut - fr.feeOut, 11301759e12, "net settles 11301759 USDC");
        assertEq(post[5], 532533306204662064, "xAfter as emitted");
    }
}
