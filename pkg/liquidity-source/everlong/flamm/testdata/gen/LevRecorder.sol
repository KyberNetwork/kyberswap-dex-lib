// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import {Test} from "forge-std/Test.sol";

/// @dev Local mirrors of the c104 structs (IFLAMMHooks.PoolContext, IFLAMMLeverage.LeverContext/LeverFill,
///      EverlongHook.Book, EverlongLeverageHook.Frame) so the recorder compiles without the via_ir src graph.
struct LevRecPoolContext {
    uint256 physicalPoolAsset;
    uint256 postedPoolAsset;
    uint256 liquidLoanAsset;
    uint256 suppliedLoanAsset;
    uint256 debtLoanAsset;
    uint256 shareSupply;
    uint256 priceWad;
    uint48 priceTs;
    uint8 loanCount;
}

struct LevRecLeverContext {
    LevRecPoolContext pool;
    bool up;
    uint256 spreadPpm;
    uint256 amountIn;
    uint256 maxOut;
}

struct LevRecLeverFill {
    uint256 amountInUsed;
    uint256 grossOut;
    uint256 virtualLegL18;
    uint256 crAfterWad;
}

struct LevRecBook {
    uint256 kappa;
    uint256 rs;
    uint256 is_;
    uint256 rv;
    uint256 iv;
    uint256 x;
}

struct LevRecFrame {
    uint256 cv;
    int256 d;
    uint256 v;
    uint256 s;
    uint256 xAnchor;
}

interface ILevRecPool {
    function previewLever(bool up, uint256 amountIn)
        external
        view
        returns (uint256 amountInUsed, uint256 amountOut, uint256 spreadPpm, uint256 crAfterWad);
}

interface ILevRecLeverageHook {
    function previewLever(LevRecLeverContext calldata ctx) external view returns (LevRecLeverFill memory);
    function frame(LevRecPoolContext calldata ctx) external view returns (LevRecFrame memory);
}

interface ILevRecSwapHook {
    function bookFor(LevRecPoolContext calldata ctx) external view returns (LevRecBook memory);
    function reservationPriceWad() external view returns (uint256);
}

/// @dev Reverts with its own calldata: etched over the leverage hook, it hands back the exact LeverContext
///      core built for a preview.
contract LevCtxProbe {
    fallback() external {
        assembly {
            let n := calldatasize()
            calldatacopy(0, 0, n)
            revert(0, n)
        }
    }
}

/// @notice Row recorder for the leverage-hook fixtures. One row per (state, direction, amount): the LeverContext
///         core builds (captured by etching LevCtxProbe over the hook for one preview), the swap hook's book and
///         reservation price for that context, the leverage hook's frame, the hook's own previewLever on the
///         captured context and the pool's previewLever, each with its revert data when it reverts.
abstract contract LevRecorder is Test {
    string[] internal levFields;
    uint256[] internal levRows;
    uint256 internal levStride;

    bytes4 internal constant PREVIEW_LEVER_SEL = ILevRecLeverageHook.previewLever.selector;

    function _levInitFields() internal {
        string[47] memory f = [
            "scenario", "up", "amount", "captured",
            "ctx.physicalPoolAsset", "ctx.postedPoolAsset", "ctx.liquidLoanAsset", "ctx.suppliedLoanAsset",
            "ctx.debtLoanAsset", "ctx.shareSupply", "ctx.priceWad", "ctx.priceTs", "ctx.loanCount",
            "ctx.up", "ctx.spreadPpm", "ctx.amountIn", "ctx.maxOut",
            "book.kappa", "book.rs", "book.is", "book.rv", "book.iv", "book.x", "reservationPriceWad",
            "frameOk", "frame.cv", "frame.d", "frame.v", "frame.s", "frame.xAnchor",
            "hookOk", "hook.amountInUsed", "hook.grossOut", "hook.virtualLegL18", "hook.crAfterWad",
            "hook.revertLen", "hook.revertSelector", "hook.revertArg",
            "poolOk", "pool.amountInUsed", "pool.amountOut", "pool.spreadPpm", "pool.crAfterWad",
            "pool.revertLen", "pool.revertSelector", "pool.revertArg", "hookAmountOverride"
        ];
        for (uint256 i; i < f.length; ++i) {
            levFields.push(f[i]);
        }
        levStride = f.length;
    }

    function _levRevertWords(bytes memory err) internal pure returns (uint256 len, uint256 sel, uint256 arg) {
        len = err.length;
        if (len >= 4) sel = uint256(uint32(bytes4(err)));
        if (len >= 36) {
            assembly {
                arg := mload(add(err, 36))
            }
        }
    }

    /// @dev Capture the LeverContext core builds for `pool.previewLever(up, amount)`.
    function _levCapture(address pool, address levHook, bool up, uint256 amount)
        internal
        returns (bool ok, LevRecLeverContext memory c)
    {
        bytes memory code = levHook.code;
        vm.etch(levHook, address(new LevCtxProbe()).code);
        bytes memory err;
        try ILevRecPool(pool).previewLever(up, amount) {}
        catch (bytes memory e) {
            err = e;
        }
        vm.etch(levHook, code);
        if (err.length != 4 + 13 * 32 || bytes4(err) != PREVIEW_LEVER_SEL) return (false, c);
        uint256[13] memory w;
        for (uint256 i; i < 13; ++i) {
            uint256 word;
            assembly {
                word := mload(add(err, add(36, mul(i, 32))))
            }
            w[i] = word;
        }
        c.pool = LevRecPoolContext(w[0], w[1], w[2], w[3], w[4], w[5], w[6], uint48(w[7]), uint8(w[8]));
        c.up = w[9] != 0;
        c.spreadPpm = w[10];
        c.amountIn = w[11];
        c.maxOut = w[12];
        ok = true;
    }

    /// @dev Record one row. `hookAmountOverride` (nonzero) replaces the captured amountIn for the hook-level call
    ///      only, for amounts core cannot express (a lever-down L18 amount off the native grid).
    function _levRecord(
        uint256 scenario,
        address pool,
        address levHook,
        address swapHook,
        bool up,
        uint256 amount,
        uint256 hookAmountOverride
    ) internal returns (bool captured, LevRecLeverFill memory hookFill, bool hookOk) {
        uint256[] memory r = new uint256[](levStride);
        r[0] = scenario;
        r[1] = up ? 1 : 0;
        r[2] = amount;
        r[46] = hookAmountOverride;
        LevRecLeverContext memory c;
        (captured, c) = _levCapture(pool, levHook, up, amount);
        if (captured) {
            if (hookAmountOverride != 0) c.amountIn = hookAmountOverride;
            (hookFill, hookOk) = _levHookColumns(r, levHook, swapHook, c);
        }
        try ILevRecPool(pool).previewLever(up, amount) returns (uint256 a, uint256 o, uint256 s, uint256 cr) {
            r[38] = 1;
            r[39] = a;
            r[40] = o;
            r[41] = s;
            r[42] = cr;
        } catch (bytes memory e) {
            (r[43], r[44], r[45]) = _levRevertWords(e);
        }
        for (uint256 i; i < r.length; ++i) {
            levRows.push(r[i]);
        }
    }

    function _levHookColumns(uint256[] memory r, address levHook, address swapHook, LevRecLeverContext memory c)
        internal
        view
        returns (LevRecLeverFill memory hookFill, bool hookOk)
    {
        r[3] = 1;
        r[4] = c.pool.physicalPoolAsset;
        r[5] = c.pool.postedPoolAsset;
        r[6] = c.pool.liquidLoanAsset;
        r[7] = c.pool.suppliedLoanAsset;
        r[8] = c.pool.debtLoanAsset;
        r[9] = c.pool.shareSupply;
        r[10] = c.pool.priceWad;
        r[11] = c.pool.priceTs;
        r[12] = c.pool.loanCount;
        r[13] = c.up ? 1 : 0;
        r[14] = c.spreadPpm;
        r[15] = c.amountIn;
        r[16] = c.maxOut;
        LevRecBook memory b = ILevRecSwapHook(swapHook).bookFor(c.pool);
        r[17] = b.kappa;
        r[18] = b.rs;
        r[19] = b.is_;
        r[20] = b.rv;
        r[21] = b.iv;
        r[22] = b.x;
        r[23] = ILevRecSwapHook(swapHook).reservationPriceWad();
        try ILevRecLeverageHook(levHook).frame(c.pool) returns (LevRecFrame memory f) {
            r[24] = 1;
            r[25] = f.cv;
            r[26] = uint256(f.d);
            r[27] = f.v;
            r[28] = f.s;
            r[29] = f.xAnchor;
        } catch {}
        try ILevRecLeverageHook(levHook).previewLever(c) returns (LevRecLeverFill memory lf) {
            hookOk = true;
            hookFill = lf;
            r[30] = 1;
            r[31] = lf.amountInUsed;
            r[32] = lf.grossOut;
            r[33] = lf.virtualLegL18;
            r[34] = lf.crAfterWad;
        } catch (bytes memory e) {
            (r[35], r[36], r[37]) = _levRevertWords(e);
        }
    }

    /// @dev Record a hook-level row for a synthetic context (no pool call; the pool columns stay zero).
    function _levRecordCtx(uint256 scenario, address levHook, address swapHook, LevRecLeverContext memory c) internal {
        uint256[] memory r = new uint256[](levStride);
        r[0] = scenario;
        r[1] = c.up ? 1 : 0;
        r[2] = c.amountIn;
        _levHookColumns(r, levHook, swapHook, c);
        for (uint256 i; i < r.length; ++i) {
            levRows.push(r[i]);
        }
    }

    /// @dev Hook-only rows on synthetic contexts derived from the context core builds now: a CR ladder set through the
    ///      loan legs (the book and mark stay the chain's) and the unquotable / overflow / spread edges.
    function _levSynthetic(uint256 scenario, address pool, address lhook, address ehook) internal {
        (bool ok, LevRecLeverContext memory base) = _levCapture(pool, lhook, true, 1e6);
        require(ok, "capture");
        LevRecFrame memory f = ILevRecLeverageHook(lhook).frame(base.pool);
        uint256[12] memory crs = [uint256(10_500), 14_000, 15_400, 15_600, 18_000, 19_950, 20_000, 20_050, 21_000, 22_500, 30_000, 60_000];
        for (uint256 i; i < crs.length; ++i) {
            _levLadderStep(scenario, lhook, ehook, base, f, f.cv * 10_000 / crs[i]);
        }
        _levEdges(scenario, lhook, ehook, base, f);
    }

    function _levCopy(LevRecLeverContext memory x) internal pure returns (LevRecLeverContext memory) {
        return abi.decode(abi.encode(x), (LevRecLeverContext));
    }

    function _levLadderStep(
        uint256 scenario,
        address lhook,
        address ehook,
        LevRecLeverContext memory base,
        LevRecFrame memory f,
        uint256 targetD
    ) internal {
        LevRecLeverContext memory c = _levCopy(base);
        c.pool.liquidLoanAsset = 0;
        if (targetD > f.s) {
            c.pool.debtLoanAsset = targetD - f.s;
            c.pool.suppliedLoanAsset = 0;
        } else {
            c.pool.debtLoanAsset = 0;
            c.pool.suppliedLoanAsset = f.s - targetD;
        }
        for (uint256 k; k < 2; ++k) {
            c.spreadPpm = k == 0 ? 5_000 : 17_500;
            c.up = true;
            uint256[4] memory ups = [f.v / 1e6 + 1, f.v / 1e3, f.v / 30, f.v / 3];
            for (uint256 j; j < 4; ++j) {
                c.amountIn = ups[j];
                _levRecordCtx(scenario, lhook, ehook, c);
            }
            c.up = false;
            uint256[5] memory downs = [targetD / 1e6 + 1, targetD / 1e3, targetD / 30, targetD / 3, targetD * 9 / 10];
            for (uint256 j; j < 5; ++j) {
                c.amountIn = downs[j];
                _levRecordCtx(scenario, lhook, ehook, c);
            }
        }
    }

    function _levEdges(uint256 scenario, address lhook, address ehook, LevRecLeverContext memory base, LevRecFrame memory f)
        internal
    {
        LevRecLeverContext memory c = _levCopy(base);
        // D = 0 and D < 0: unquotable
        c.pool.liquidLoanAsset = 0;
        c.pool.debtLoanAsset = 0;
        c.pool.suppliedLoanAsset = f.s;
        _levRecordCtx(scenario, lhook, ehook, c);
        c.pool.suppliedLoanAsset = f.s + 1;
        _levRecordCtx(scenario, lhook, ehook, c);
        // int256 reinterpretation of a huge debt leg: D wraps negative
        c.pool.suppliedLoanAsset = 0;
        c.pool.debtLoanAsset = 1 << 255;
        _levRecordCtx(scenario, lhook, ehook, c);
        // checked add of the credit legs
        c.pool.debtLoanAsset = base.pool.debtLoanAsset;
        c.pool.suppliedLoanAsset = type(uint256).max;
        c.pool.liquidLoanAsset = 1;
        _levRecordCtx(scenario, lhook, ehook, c);
        // gavAtFeed overflow
        c = _levCopy(base);
        c.pool.priceWad = 1 << 250;
        _levRecordCtx(scenario, lhook, ehook, c);
        c.up = false;
        c.amountIn = 1e18;
        _levRecordCtx(scenario, lhook, ehook, c);
        c.pool.priceWad = 0;
        _levRecordCtx(scenario, lhook, ehook, c);
        // mulDiv overflow of the collateral delta
        c = _levCopy(base);
        c.amountIn = 1 << 250;
        _levRecordCtx(scenario, lhook, ehook, c);
        // the spread at and past PPM
        c = _levCopy(base);
        c.spreadPpm = 1e6;
        _levRecordCtx(scenario, lhook, ehook, c);
        c.spreadPpm = 1e6 + 1;
        _levRecordCtx(scenario, lhook, ehook, c);
        c.up = false;
        c.amountIn = 1e18;
        _levRecordCtx(scenario, lhook, ehook, c);
        c.spreadPpm = 0;
        _levRecordCtx(scenario, lhook, ehook, c);
        // lever-down at and past D
        c.spreadPpm = base.spreadPpm;
        c.amountIn = uint256(f.d);
        _levRecordCtx(scenario, lhook, ehook, c);
        c.amountIn = uint256(f.d) - 1;
        _levRecordCtx(scenario, lhook, ehook, c);
    }

    /// @dev Revert the chain to `snap` while keeping the rows recorded since (a state revert also rewinds this
    ///      contract's storage).
    function _levRevertKeepingRows(uint256 snap) internal {
        uint256[] memory keep = levRows;
        vm.revertToState(snap);
        levRows = keep;
    }

    function _levWrite(string memory obj, string memory path, string[] memory scenarios, uint256 blockNumber) internal {
        vm.serializeUint(obj, "stride", levStride);
        vm.serializeUint(obj, "rows", levRows.length / levStride);
        vm.serializeUint(obj, "block", blockNumber);
        vm.serializeString(obj, "scenarios", scenarios);
        vm.serializeString(obj, "fields", levFields);
        string memory out = vm.serializeUint(obj, "v", levRows);
        vm.writeJson(out, path);
    }
}
